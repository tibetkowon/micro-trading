package agent

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/micro-trading-for-agent/backend/internal/kis"
)

// StockInfo holds key stock data for the AI agent's decision-making,
// including current price, moving averages, trading value, RSI, and MACD.
type StockInfo struct {
	StockCode     string  `json:"stock_code"`
	CurrentPrice  string  `json:"current_price"`
	ChangeRate    string  `json:"change_rate"`
	Volume        string  `json:"volume"`
	TradingValue  float64 `json:"trading_value"`   // 거래대금 (volume × price, KRW); 0 = unavailable
	DayOpen       string  `json:"day_open"`        // 당일 시가
	DayHigh       string  `json:"day_high"`        // 당일 고가
	DayLow        string  `json:"day_low"`         // 당일 저가
	HighPriceDiff float64 `json:"high_price_diff"` // (현재가-고가)/고가×100 (음수=눌림 정도)
	OpenPriceDiff float64 `json:"open_price_diff"` // (현재가-시가)/시가×100 (오늘 상승률)
	DisparityM5   float64 `json:"disparity_m5"`    // (현재가-5분봉MA5)/5분봉MA5×100
	MA5           float64 `json:"ma5"`
	MA20          float64 `json:"ma20"`
	RSI14         float64 `json:"rsi14"`          // RSI(14) from 5-minute closes; 0 = insufficient data
	MACDLine      float64 `json:"macd_line"`      // MACD line (EMA12 − EMA26) from 5m candles
	MACDSignal    float64 `json:"macd_signal"`    // Signal line (EMA9 of MACD line) from 5m candles
	MACDHisto     float64 `json:"macd_histogram"` // Histogram (MACD line − Signal line)
	VWAP            float64 `json:"vwap"`              // 당일 VWAP (거래량가중평균가); 0=데이터부족
	VWAPDiff        float64 `json:"vwap_diff"`         // (현재가-VWAP)/VWAP×100 (%)
	M5MA10          float64 `json:"m5_ma10"`           // 5분봉 MA10; 0=데이터부족
	PrevVolumeRatio float64 `json:"prev_volume_ratio"` // 직전봉 대비 현재봉 거래량 비율; 0=데이터부족
	BidAskRatio     float64 `json:"bid_ask_ratio"`     // 총 매수잔량 / 총 매도잔량; 0=API 실패 또는 데이터 없음
}

// GetStockInfo fetches the latest price and computes all technical indicators:
//   - MA5 / MA20: from 5-minute candle closes (up to 200 1-min bars → ~40 5-min bars)
//   - TradingValue: current_price × today's volume (KRW)
//   - RSI(14), MACD(12,26,9): from 5-minute candles
//
// Indicator fields are 0 when there is insufficient market data (pre-open, thin session, etc.).
func GetStockInfo(ctx context.Context, client *kis.Client, stockCode string) (*StockInfo, error) {
	if stockCode == "" {
		return nil, fmt.Errorf("stock_code is required")
	}

	resp, err := client.GetStockPrice(ctx, stockCode)
	if err != nil {
		return nil, fmt.Errorf("GetStockInfo [%s]: %w", stockCode, err)
	}

	info := &StockInfo{
		StockCode:    resp.StockCode,
		CurrentPrice: resp.CurrentPrice,
		ChangeRate:   resp.ChangeRate,
		Volume:       resp.Volume,
		DayOpen:      resp.DayOpen,
		DayHigh:      resp.DayHigh,
		DayLow:       resp.DayLow,
	}

	// --- TradingValue: current_price × today's volume ---
	price, _ := strconv.ParseFloat(resp.CurrentPrice, 64)
	vol, _ := strconv.ParseFloat(resp.Volume, 64)
	if price > 0 && vol > 0 {
		info.TradingValue = math.Round(price * vol)
	}

	// --- HighPriceDiff / OpenPriceDiff ---
	if high, err := strconv.ParseFloat(resp.DayHigh, 64); err == nil && high > 0 {
		info.HighPriceDiff = math.Round((price-high)/high*10000) / 100
	}
	if open, err := strconv.ParseFloat(resp.DayOpen, 64); err == nil && open > 0 {
		info.OpenPriceDiff = math.Round((price-open)/open*10000) / 100
	}

	// --- MA5 / MA20 / RSI(14) / MACD(12,26,9) / DisparityM5 from 5-minute candles ---
	// Fetch 200 1-minute bars → aggregate to ~40 5-minute bars.
	// 40 bars is sufficient for MACD(12,26,9) which needs 26+9-1 = 34 periods minimum.
	// MA5/MA20 are also computed from these 5m closes for intraday consistency.
	bars, chartErr := fetchMinuteBars(ctx, client, stockCode, 200)
	if chartErr == nil && len(bars) > 0 {
		// VWAP from 1-min bars (oldest-first after fetchMinuteBars reversal)
		var sumPV, sumVol float64
		for _, b := range bars {
			p, _ := strconv.ParseFloat(b.Close, 64)
			v, _ := strconv.ParseFloat(b.Volume, 64)
			if p > 0 && v > 0 {
				sumPV += p * v
				sumVol += v
			}
		}
		if sumVol > 0 && len(bars) >= 5 { // 5분 이상 데이터여야 신뢰성 있음
			info.VWAP = math.Round(sumPV/sumVol*100) / 100
			if info.VWAP > 0 && price > 0 {
				info.VWAPDiff = math.Round((price-info.VWAP)/info.VWAP*10000) / 100
			}
		}

		candles5m := aggregateMinuteBars(bars, 5)
		if len(candles5m) >= 2 {
			closes5m := make([]float64, len(candles5m))
			for i, c := range candles5m {
				closes5m[i] = c.Close
			}
			info.MA5 = calcMA(closes5m, 5)
			info.MA20 = calcMA(closes5m, 20)
			info.RSI14 = calcRSI(closes5m, 14)
			info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes5m, 12, 26, 9)
			// DisparityM5: 현재가와 5분봉 MA5의 이격도
			if ma5m := calcMA(closes5m, 5); ma5m > 0 && price > 0 {
				info.DisparityM5 = math.Round((price-ma5m)/ma5m*10000) / 100
			}
			info.M5MA10 = calcMA(closes5m, 10)

			// PrevVolumeRatio: 직전 5분봉 대비 최근 5분봉 거래량 비율
			if len(candles5m) >= 2 {
				curVol := float64(candles5m[len(candles5m)-1].Volume)
				prevVol := float64(candles5m[len(candles5m)-2].Volume)
				if prevVol > 0 {
					info.PrevVolumeRatio = math.Round(curVol/prevVol*100) / 100
				}
			}
		}
	}

	// BidAskRatio: 호가 잔량 비율 (매수잔량/매도잔량) — 별도 KIS API 호출
	// 실패해도 info.BidAskRatio=0으로 처리하고 에러는 무시 (non-critical)
	if ratio, err := client.GetBidAskRatio(ctx, stockCode); err == nil {
		info.BidAskRatio = math.Round(ratio*100) / 100
	}

	return info, nil
}

// --- Moving Average ---

// calcMA returns the simple moving average of the last `period` values in closes.
// Returns 0 if there are fewer than `period` values.
func calcMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return 0
	}
	sum := 0.0
	for _, v := range closes[len(closes)-period:] {
		sum += v
	}
	return math.Round(sum/float64(period)*100) / 100
}

// --- RSI ---

// calcRSI computes Wilder's RSI for the given period.
// Returns 0 when len(closes) < period+1 (insufficient data).
func calcRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	// Seed: initial average gain / average loss over first `period` changes.
	var avgGain, avgLoss float64
	for i := 1; i <= period; i++ {
		delta := closes[i] - closes[i-1]
		if delta > 0 {
			avgGain += delta
		} else {
			avgLoss += -delta
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Wilder smoothing for subsequent bars.
	for i := period + 1; i < len(closes); i++ {
		delta := closes[i] - closes[i-1]
		gain, loss := 0.0, 0.0
		if delta > 0 {
			gain = delta
		} else {
			loss = -delta
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return math.Round((100-100/(1+rs))*100) / 100
}

// --- EMA / MACD ---

// calcEMA returns the full EMA series for the given period.
// The first period-1 entries are 0 (seeding phase); valid values start at index period-1.
// Returns nil when len(closes) < period.
func calcEMA(closes []float64, period int) []float64 {
	if len(closes) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	ema := make([]float64, len(closes))

	// Seed: SMA of first `period` closes.
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema[period-1] = sum / float64(period)

	for i := period; i < len(closes); i++ {
		ema[i] = closes[i]*k + ema[i-1]*(1-k)
	}
	return ema
}

// calcMACD returns the last MACD line, signal, and histogram values.
// Standard parameters: fastPeriod=12, slowPeriod=26, signalPeriod=9.
// Returns (0, 0, 0) when data is insufficient.
func calcMACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (macdLine, signal, histogram float64) {
	if len(closes) < slowPeriod {
		return 0, 0, 0
	}

	fastEMA := calcEMA(closes, fastPeriod)
	slowEMA := calcEMA(closes, slowPeriod)
	if fastEMA == nil || slowEMA == nil {
		return 0, 0, 0
	}

	// MACD series: valid from index slowPeriod-1 onward.
	validLen := len(closes) - slowPeriod + 1
	macdSeries := make([]float64, validLen)
	for i := 0; i < validLen; i++ {
		idx := i + slowPeriod - 1
		macdSeries[i] = fastEMA[idx] - slowEMA[idx]
	}

	lastMACD := macdSeries[len(macdSeries)-1]

	if len(macdSeries) < signalPeriod {
		// Not enough MACD values for the signal EMA yet.
		return math.Round(lastMACD*100) / 100, 0, 0
	}

	signalEMA := calcEMA(macdSeries, signalPeriod)
	if signalEMA == nil {
		return math.Round(lastMACD*100) / 100, 0, 0
	}

	lastSignal := signalEMA[len(signalEMA)-1]
	lastHisto := lastMACD - lastSignal

	return math.Round(lastMACD*100) / 100,
		math.Round(lastSignal*100) / 100,
		math.Round(lastHisto*100) / 100
}

// GetOverseasStockInfo fetches current price and computes all technical indicators
// for an overseas (US) stock using 5-minute bars (HHDFS76950200).
//   - MA5 / MA20: from 5-minute closes (up to 120 bars)
//   - RSI(14), MACD(12,26,9): from 5-minute closes
//   - DisparityM5: (currentPrice - MA5) / MA5 × 100
//
// Indicator fields are 0 when there is insufficient data.
func GetOverseasStockInfo(ctx context.Context, client *kis.Client, excd, symb string) (*StockInfo, error) {
	priceResp, err := client.GetOverseasPrice(ctx, excd, symb)
	if err != nil {
		return nil, fmt.Errorf("GetOverseasStockInfo [%s]: %w", symb, err)
	}

	currentPrice, _ := strconv.ParseFloat(priceResp.Last, 64)
	vol, _ := strconv.ParseFloat(priceResp.TVol, 64)

	info := &StockInfo{
		StockCode:    symb,
		CurrentPrice: priceResp.Last,
		ChangeRate:   priceResp.Rate,
		Volume:       priceResp.TVol,
		DayOpen:      priceResp.Open,
		DayHigh:      priceResp.High,
		DayLow:       priceResp.Low,
	}

	if currentPrice > 0 && vol > 0 {
		info.TradingValue = currentPrice * vol
	}

	high, _ := strconv.ParseFloat(priceResp.High, 64)
	open, _ := strconv.ParseFloat(priceResp.Open, 64)
	if currentPrice > 0 && high > 0 {
		info.HighPriceDiff = math.Round((currentPrice-high)/high*10000) / 100
	}
	if currentPrice > 0 && open > 0 {
		info.OpenPriceDiff = math.Round((currentPrice-open)/open*10000) / 100
	}

	// 5분봉 데이터 조회 — newest-first → reverse to oldest-first for indicators.
	bars, chartErr := client.GetOverseasMinuteChart(ctx, excd, symb)
	if chartErr == nil && len(bars) > 0 {
		closes5m := make([]float64, 0, len(bars))
		for i := len(bars) - 1; i >= 0; i-- {
			v, parseErr := strconv.ParseFloat(bars[i].Last, 64)
			if parseErr == nil && v > 0 {
				closes5m = append(closes5m, v)
			}
		}
		if len(closes5m) >= 2 {
			info.MA5 = calcMA(closes5m, 5)
			info.MA20 = calcMA(closes5m, 20)
			info.RSI14 = calcRSI(closes5m, 14)
			info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes5m, 12, 26, 9)
			if info.MA5 > 0 && currentPrice > 0 {
				info.DisparityM5 = math.Round((currentPrice-info.MA5)/info.MA5*10000) / 100
			}
		}
	}

	return info, nil
}
