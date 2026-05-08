package ops

import (
	"context"
	"fmt"
	"math"
	"strconv"

	"github.com/micro-trading-for-agent/backend/internal/kis"
	"github.com/micro-trading-for-agent/backend/internal/logger"
)

// CandleSnap is a compact snapshot of a single 5-minute candle.
type CandleSnap struct {
	Close  float64 `json:"c"`           // 종가
	Volume int64   `json:"v"`           // 거래량
	Dir    string  `json:"d,omitempty"` // "U"=상승봉, "D"=하락봉, "="=보합
}

// StockInfo holds key stock data for the AI agent's decision-making,
// including current price, moving averages, trading value, RSI, and MACD.
type StockInfo struct {
	StockCode       string  `json:"stock_code"`
	CurrentPrice    string  `json:"current_price"`
	ChangeRate      string  `json:"change_rate"`
	Volume          string  `json:"volume"`
	TradingValue    float64 `json:"trading_value"`   // 거래대금 (volume × price, KRW); 0 = unavailable
	Strength        float64 `json:"strength"`        // 체결강도 (당일); 0 = unavailable
	DayOpen         string  `json:"day_open"`        // 당일 시가
	DayHigh         string  `json:"day_high"`        // 당일 고가
	DayLow          string  `json:"day_low"`         // 당일 저가
	HighPriceDiff   float64 `json:"high_price_diff"` // (현재가-고가)/고가×100 (음수=눌림 정도)
	OpenPriceDiff   float64 `json:"open_price_diff"` // (현재가-시가)/시가×100 (오늘 상승률)
	DisparityM5     float64 `json:"disparity_m5"`    // (현재가-5분봉MA5)/5분봉MA5×100
	MA5             float64 `json:"ma5"`
	MA20            float64 `json:"ma20"`
	MA60            float64 `json:"ma60"`
	MA120           float64 `json:"ma120"`
	RSI14           float64 `json:"rsi14"`             // RSI(14) from 5-minute closes; 0 = insufficient data
	MACDLine        float64 `json:"macd_line"`         // MACD line (EMA12 − EMA26) from 5m candles
	MACDSignal      float64 `json:"macd_signal"`       // Signal line (EMA9 of MACD line) from 5m candles
	MACDHisto       float64 `json:"macd_histogram"`    // Histogram (MACD line − Signal line)
	VWAP            float64 `json:"vwap"`              // 당일 VWAP (거래량가중평균가); 0=데이터부족
	VWAPDiff        float64 `json:"vwap_diff"`         // (현재가-VWAP)/VWAP×100 (%)
	M5MA10          float64 `json:"m5_ma10"`           // 5분봉 MA10; 0=데이터부족
	PrevVolumeRatio float64 `json:"prev_volume_ratio"` // 직전봉 대비 현재봉 거래량 비율; 0=데이터부족
	BidAskRatio     float64 `json:"bid_ask_ratio"`     // 총 매수잔량 / 총 매도잔량; 0=API 실패 또는 데이터 없음
	BidAskSpread    float64 `json:"bid_ask_spread"`    // (매도1호가-매수1호가)/매도1호가×100 (%); 0=API 실패
	// 자산 타입 (MST 기반 — engine이 태깅)
	AssetType string `json:"asset_type"` // "STOCK" | "ETF" | "ETF_DOMESTIC"

	// 데이터 품질 개선 필드 (1~3순위)
	RecentCandles     []CandleSnap `json:"recent_candles,omitempty"` // 최근 5개 5분봉 (구→신 순서)
	HighFormedMinsAgo int          `json:"high_formed_mins_ago"`     // 당일 고점 형성 후 경과 시간(분); 0=데이터부족
	VolTrend3         float64      `json:"vol_trend_3"`              // 최근 3봉 거래량 기울기 (-1=감소, 0=보합, 1=증가)
	VolAtHigh         int64        `json:"vol_at_high"`              // 고점 형성 봉의 거래량; 0=데이터부족
	VolVs3AvgRatio    float64      `json:"vol_vs_3avg_ratio"`        // 현재봉 거래량 / 직전 3봉 평균 거래량 (거래량 회복 비율); 0=데이터부족
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

	// --- Strength: 체결강도 from inquire-price response ---
	if s, err := strconv.ParseFloat(resp.Strength, 64); err == nil && s > 0 {
		info.Strength = math.Round(s*10) / 10
	}

	// --- HighPriceDiff / OpenPriceDiff ---
	if high, err := strconv.ParseFloat(resp.DayHigh, 64); err == nil && high > 0 {
		info.HighPriceDiff = math.Round((price-high)/high*10000) / 100
	}
	if open, err := strconv.ParseFloat(resp.DayOpen, 64); err == nil && open > 0 {
		info.OpenPriceDiff = math.Round((price-open)/open*10000) / 100
	}

	// Fetch current-day 1-minute bars using the single-call API (reduces TPS load).
	bars, chartErr := fetchCurrentDayBars(ctx, client, stockCode)
	if chartErr != nil {
		logger.Warn("GetStockInfo: minute chart fetch failed",
			map[string]any{"code": stockCode, "error": chartErr.Error()})
	} else if len(bars) == 0 {
		logger.Warn("GetStockInfo: minute chart returned no bars",
			map[string]any{"code": stockCode})
	}
	if chartErr == nil && len(bars) > 0 {
		fillChartIndicators(info, bars, price)
	}

	return info, nil
}

// fillChartIndicators populates chart-based indicators on info from 1-minute bars.
// price must be pre-parsed from info.CurrentPrice.
func fillChartIndicators(info *StockInfo, bars []kis.ChartBar, price float64) {
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

	// 1분봉 캔들 변환 — RSI/MACD/MA/VWAP 계산 기준
	candles1m := minuteBarsToCandles(bars)
	if len(candles1m) >= 2 {
		closes1m := make([]float64, len(candles1m))
		for i, c := range candles1m {
			closes1m[i] = c.Close
		}
		info.MA5 = calcMA(closes1m, 5)
		info.MA20 = calcMA(closes1m, 20)
		info.MA60 = calcMA(closes1m, 60)
		info.MA120 = calcMA(closes1m, 120)
		info.RSI14 = calcRSI(closes1m, 14)
		info.MACDLine, info.MACDSignal, info.MACDHisto = calcMACD(closes1m, 12, 26, 9)

		// DisparityM5: 현재가와 1분봉 MA5의 이격도
		if info.MA5 > 0 && price > 0 {
			info.DisparityM5 = math.Round((price-info.MA5)/info.MA5*10000) / 100
		}
		info.M5MA10 = calcMA(closes1m, 10)

		// PrevVolumeRatio: 직전 1분봉 대비 현재 1분봉 거래량 비율
		if len(candles1m) >= 2 {
			curVol := float64(candles1m[len(candles1m)-1].Volume)
			prevVol := float64(candles1m[len(candles1m)-2].Volume)
			if prevVol > 0 {
				info.PrevVolumeRatio = math.Round(curVol/prevVol*100) / 100
			}
		}

		// ── 최근 5개 1분봉 캔들 시퀀스 ──────────────────────────────
		{
			n := len(candles1m)
			start := n - 5
			if start < 0 {
				start = 0
			}
			recent := candles1m[start:]
			snaps := make([]CandleSnap, len(recent))
			for i, c := range recent {
				dir := "="
				if c.Close > c.Open {
					dir = "U"
				} else if c.Close < c.Open {
					dir = "D"
				}
				snaps[i] = CandleSnap{
					Close:  math.Round(c.Close*100) / 100,
					Volume: c.Volume,
					Dir:    dir,
				}
			}
			info.RecentCandles = snaps
		}

		// ── 고점 형성 경과 시간 (1분봉 기준, 1봉 = 1분) ──────────
		{
			dayHigh, _ := strconv.ParseFloat(info.DayHigh, 64)
			if dayHigh > 0 && len(candles1m) >= 1 {
				highIdx := -1
				for i := len(candles1m) - 1; i >= 0; i-- {
					if candles1m[i].High >= dayHigh*0.9999 {
						highIdx = i
						break
					}
				}
				if highIdx >= 0 {
					info.HighFormedMinsAgo = len(candles1m) - 1 - highIdx // 1봉 = 1분
					info.VolAtHigh = candles1m[highIdx].Volume
				}
			}

			// VolTrend3: 최근 3개 1분봉 거래량 기울기
			if len(candles1m) >= 3 {
				v1 := float64(candles1m[len(candles1m)-3].Volume)
				v3 := float64(candles1m[len(candles1m)-1].Volume)
				maxV := math.Max(v1, math.Max(float64(candles1m[len(candles1m)-2].Volume), v3))
				if maxV > 0 {
					slope := (v3 - v1) / maxV
					info.VolTrend3 = math.Round(slope*100) / 100
				}
			}

			// VolVs3AvgRatio: 현재 1분봉 거래량 / 직전 3봉 평균
			if len(candles1m) >= 4 {
				cur := float64(candles1m[len(candles1m)-1].Volume)
				avg3 := (float64(candles1m[len(candles1m)-2].Volume) +
					float64(candles1m[len(candles1m)-3].Volume) +
					float64(candles1m[len(candles1m)-4].Volume)) / 3
				if avg3 > 0 {
					info.VolVs3AvgRatio = math.Round(cur/avg3*100) / 100
				}
			}
		}
	}
}

// GetStockInfoWithPrice fetches only the chart data, reusing an already-fetched
// StockPriceResponse. Use this in a two-pass filter where GetStockPrice was
// already called for pre-filtering.
func GetStockInfoWithPrice(ctx context.Context, client *kis.Client, stockCode string, resp *kis.StockPriceResponse) (*StockInfo, error) {
	if stockCode == "" {
		return nil, fmt.Errorf("stock_code is required")
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

	price, _ := strconv.ParseFloat(resp.CurrentPrice, 64)
	vol, _ := strconv.ParseFloat(resp.Volume, 64)
	if price > 0 && vol > 0 {
		info.TradingValue = math.Round(price * vol)
	}
	if s, err := strconv.ParseFloat(resp.Strength, 64); err == nil && s > 0 {
		info.Strength = math.Round(s*10) / 10
	}
	if high, err := strconv.ParseFloat(resp.DayHigh, 64); err == nil && high > 0 {
		info.HighPriceDiff = math.Round((price-high)/high*10000) / 100
	}
	if open, err := strconv.ParseFloat(resp.DayOpen, 64); err == nil && open > 0 {
		info.OpenPriceDiff = math.Round((price-open)/open*10000) / 100
	}

	bars, chartErr := fetchCurrentDayBars(ctx, client, stockCode)
	if chartErr != nil {
		logger.Warn("GetStockInfoWithPrice: chart fetch failed",
			map[string]any{"code": stockCode, "error": chartErr.Error()})
	}
	if chartErr == nil && len(bars) > 0 {
		fillChartIndicators(info, bars, price)
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
