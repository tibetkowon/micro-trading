// Package scorer implements the rule-based hard filter and composite scoring system (v2).
// Candidates are first filtered by hard rules; those that pass are scored 0-100 per indicator
// and combined with configurable weights into a final score.
package scorer

import (
	"fmt"
	"math"
	"strconv"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/ops"
)

// CandidateInfo combines StockInfo with ranking-derived data not available from the price API.
type CandidateInfo struct {
	StockCode string
	StockName string
	Info      *ops.StockInfo
	Strength  float64 // 체결강도 from ranking API; 0 = not available
}

// FilterResult reports whether a candidate passed all hard filters.
type FilterResult struct {
	Passed bool
	Reason string // rejection reason when !Passed; empty when Passed
}

// ScoreDetail holds per-indicator raw scores (0-100) and the weighted composite total.
type ScoreDetail struct {
	Strength    float64
	RSI         float64
	MACD        float64
	BidAsk      float64
	VWAP        float64
	Volume      float64
	ProgramBuy  float64
	MicroBidAsk float64
	VIDisparity float64
	Total       float64 // weighted sum normalised to 0-100
}

// String returns a compact log-friendly representation.
func (d ScoreDetail) String() string {
	return fmt.Sprintf("total=%.1f str=%.1f rsi=%.1f macd=%.1f bid=%.1f vwap=%.1f vol=%.1f pgb=%.1f mbid=%.1f vi=%.1f",
		d.Total, d.Strength, d.RSI, d.MACD, d.BidAsk, d.VWAP, d.Volume, d.ProgramBuy, d.MicroBidAsk, d.VIDisparity)
}

// ApplyHardFilter returns whether the candidate passes all configured hard filters.
// Each failing rule returns immediately with the reason for logging.
func ApplyHardFilter(c CandidateInfo, s database.TradingSettings) FilterResult {
	info := c.Info

	// RSI 상한 (초과 시 과매수로 제외)
	if s.HardRSIMax > 0 && info.RSI14 > 0 && info.RSI14 > s.HardRSIMax {
		return FilterResult{Reason: fmt.Sprintf("RSI %.1f > max %.1f", info.RSI14, s.HardRSIMax)}
	}

	// 체결강도 하한
	if s.HardStrengthMin > 0 && c.Strength > 0 && c.Strength < s.HardStrengthMin {
		return FilterResult{Reason: fmt.Sprintf("strength %.1f < min %.1f", c.Strength, s.HardStrengthMin)}
	}

	// 프로그램 순매수 하한
	if info.ProgramNetBuy < s.HardProgramBuyMin {
		return FilterResult{Reason: fmt.Sprintf("program net buy %.0f < min %.0f", info.ProgramNetBuy, s.HardProgramBuyMin)}
	}

	// 5분봉 이격도 범위
	if info.DisparityM5 != 0 {
		if s.HardDisparityM5Min != 0 && info.DisparityM5 < s.HardDisparityM5Min {
			return FilterResult{Reason: fmt.Sprintf("disparityM5 %.2f < min %.2f", info.DisparityM5, s.HardDisparityM5Min)}
		}
		if s.HardDisparityM5Max != 0 && info.DisparityM5 > s.HardDisparityM5Max {
			return FilterResult{Reason: fmt.Sprintf("disparityM5 %.2f > max %.2f", info.DisparityM5, s.HardDisparityM5Max)}
		}
	}

	// 고가 대비 현재가 이격 범위
	if s.HardHighPriceDiffMin != 0 && info.HighPriceDiff < s.HardHighPriceDiffMin {
		return FilterResult{Reason: fmt.Sprintf("highPriceDiff %.2f < min %.2f", info.HighPriceDiff, s.HardHighPriceDiffMin)}
	}
	if s.HardHighPriceDiffMax != 0 && info.HighPriceDiff > s.HardHighPriceDiffMax {
		return FilterResult{Reason: fmt.Sprintf("highPriceDiff %.2f > max %.2f", info.HighPriceDiff, s.HardHighPriceDiffMax)}
	}

	// 시가 대비 상승률 상한 (너무 많이 오른 종목 제외)
	if s.HardOpenPriceDiffMax != 0 && info.OpenPriceDiff > s.HardOpenPriceDiffMax {
		return FilterResult{Reason: fmt.Sprintf("openPriceDiff %.2f > max %.2f", info.OpenPriceDiff, s.HardOpenPriceDiffMax)}
	}

	// MACD 데드크로스 제외
	if s.HardMACDBearishEnabled && info.MACDLine != 0 && info.MACDLine < info.MACDSignal {
		return FilterResult{Reason: "MACD bearish (deadcross)"}
	}

	// MA60 지지선 (1분봉 기준 1시간 추세선) — 현재가 < MA60이면 탈락
	if s.HardMA60SupportEnabled && info.MA60 > 0 {
		price, _ := strconv.ParseFloat(info.CurrentPrice, 64)
		if price > 0 && price < info.MA60 {
			return FilterResult{Reason: fmt.Sprintf("price %.0f below MA60 %.0f", price, info.MA60)}
		}
	}

	// MA120 지지선 (1분봉 기준 2시간 추세선)
	if s.HardMA120SupportEnabled && info.MA120 > 0 {
		price, _ := strconv.ParseFloat(info.CurrentPrice, 64)
		if price > 0 && price < info.MA120 {
			return FilterResult{Reason: fmt.Sprintf("price %.0f below MA120 %.0f", price, info.MA120)}
		}
	}

	// 고점 형성 후 경과 시간 상한 (너무 오래된 고점 제외)
	if s.HardHighFormedMinsMax > 0 && info.HighFormedMinsAgo > 0 &&
		float64(info.HighFormedMinsAgo) > s.HardHighFormedMinsMax {
		return FilterResult{Reason: fmt.Sprintf("highFormedMinsAgo %d > max %.0f",
			info.HighFormedMinsAgo, s.HardHighFormedMinsMax)}
	}

	// 거래량 회복 비율 하한 (최근 거래량이 직전 3봉 평균 대비 너무 낮으면 제외)
	if s.HardVolVs3AvgRatioMin > 0 && info.VolVs3AvgRatio > 0 &&
		info.VolVs3AvgRatio < s.HardVolVs3AvgRatioMin {
		return FilterResult{Reason: fmt.Sprintf("volVs3AvgRatio %.2f < min %.2f",
			info.VolVs3AvgRatio, s.HardVolVs3AvgRatioMin)}
	}

	// 직전봉 대비 현재봉 거래량 비율 상한 (거래량 급감 방지)
	if s.HardPrevVolRatioMax > 0 && info.PrevVolumeRatio > 0 &&
		info.PrevVolumeRatio > s.HardPrevVolRatioMax {
		return FilterResult{Reason: fmt.Sprintf("prevVolRatio %.2f > max %.2f",
			info.PrevVolumeRatio, s.HardPrevVolRatioMax)}
	}

	// 최소 거래대금
	if s.MinTradingValue > 0 && info.TradingValue > 0 && info.TradingValue < s.MinTradingValue {
		return FilterResult{Reason: fmt.Sprintf("tradingValue %.0f < min %.0f",
			info.TradingValue, s.MinTradingValue)}
	}

	return FilterResult{Passed: true}
}

// CalcScore computes the weighted composite score for a candidate (0-100 scale).
// Each indicator is scored 0-100 using scalping-optimised thresholds, then blended by configured weights.
// Indicators with missing data return 0 so incomplete candidates are eliminated from the scan cycle.
func CalcScore(c CandidateInfo, s database.TradingSettings) ScoreDetail {
	info := c.Info
	d := ScoreDetail{
		Strength:    scoreStrength(c.Strength),
		RSI:         scoreRSI(info.RSI14),
		MACD:        scoreMACD(info.MACDLine, info.MACDSignal),
		BidAsk:      scoreBidAsk(info.BidAskRatio),
		VWAP:        scoreVWAP(info.VWAPDiff, info.VWAP),
		Volume:      scoreVolume(info.VolVs3AvgRatio),
		ProgramBuy:  scoreProgramBuy(info.ProgramNetBuy, info.Volume),
		MicroBidAsk: scoreMicroBidAsk(info.MicroBidAskRatio),
		VIDisparity: scoreVIDisparity(info.VIDisparity),
	}

	totalWeight := float64(s.ScoreWeightStrength + s.ScoreWeightRSI + s.ScoreWeightMACD +
		s.ScoreWeightBidAsk + s.ScoreWeightVWAP + s.ScoreWeightVolume +
		s.ScoreWeightProgramBuy + s.ScoreWeightMicroBidAsk + s.ScoreWeightVIDisparity)
	if totalWeight == 0 {
		return d // all weights zero → total stays 0
	}

	d.Total = math.Round((d.Strength*float64(s.ScoreWeightStrength)+
		d.RSI*float64(s.ScoreWeightRSI)+
		d.MACD*float64(s.ScoreWeightMACD)+
		d.BidAsk*float64(s.ScoreWeightBidAsk)+
		d.VWAP*float64(s.ScoreWeightVWAP)+
		d.Volume*float64(s.ScoreWeightVolume)+
		d.ProgramBuy*float64(s.ScoreWeightProgramBuy)+
		d.MicroBidAsk*float64(s.ScoreWeightMicroBidAsk)+
		d.VIDisparity*float64(s.ScoreWeightVIDisparity))/totalWeight*10) / 10

	return d
}

// scoreStrength: >= 250 → 100 (대장주급 수급), 100-250 → linear, < 100 → 0. missing → 0.
func scoreStrength(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 250 {
		return 100
	}
	if v < 100 {
		return 0
	}
	return clamp((v-100)/150*100, 0, 100)
}

// scoreRSI (1분봉): 60-75 → 100 (본격 슈팅 구간), 50-60 → linear, 75-85 → linear, outside → 0. missing → 0.
func scoreRSI(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v < 50 || v >= 85 {
		return 0
	}
	if v >= 60 && v <= 75 {
		return 100
	}
	if v < 60 { // 50 ≤ v < 60: 진입 초입
		return clamp((v-50)/10*100, 0, 100)
	}
	// 75 < v < 85: 윗꼬리 리스크 구간
	return clamp((85-v)/10*100, 0, 100)
}

// scoreMACD: golden cross (line > signal) → 100, dead cross → 0. both 0 = no data → 0.
func scoreMACD(line, signal float64) float64 {
	if line == 0 && signal == 0 {
		return 0
	}
	if line > signal {
		return 100
	}
	return 0
}

// scoreBidAsk: >= 3.0 → 100 (매도 물량 충분), 1.0-3.0 → linear, < 1.0 → 0. missing → 0.
func scoreBidAsk(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 3.0 {
		return 100
	}
	if v < 1.0 {
		return 0
	}
	return clamp((v-1.0)/2.0*100, 0, 100)
}

// scoreVWAP (이격률 %): +0.5~+2.0% → 100 (돌파 지지 구간), 0~0.5% → linear, 2~5% → linear, outside → 0. missing → 0.
func scoreVWAP(diff, vwap float64) float64 {
	if vwap <= 0 {
		return 0
	}
	if diff < 0 || diff >= 5.0 {
		return 0
	}
	if diff >= 0.5 && diff <= 2.0 {
		return 100
	}
	if diff < 0.5 { // 0 ≤ diff < 0.5: VWAP 돌파 직전
		return clamp(diff/0.5*100, 0, 100)
	}
	// 2.0 < diff < 5.0: 급등 리스크 구간
	return clamp((5.0-diff)/3.0*100, 0, 100)
}

// scoreVolume: >= 3.0 → 100 (폭발적 거래량), 1.0-3.0 → linear, < 1.0 → 0. missing → 0.
func scoreVolume(v float64) float64 {
	if v <= 0 {
		return 0
	}
	if v >= 3.0 {
		return 100
	}
	if v < 1.0 {
		return 0
	}
	return clamp((v-1.0)/2.0*100, 0, 100)
}

// scoreProgramBuy: (순매수 / 총거래량 * 100)
// >= 10.0% -> 100, <= 0.0% -> 0. 0 = no data -> 50.
func scoreProgramBuy(netBuy float64, volStr string) float64 {
	vol, _ := strconv.ParseFloat(volStr, 64)
	if vol <= 0 {
		return 50
	}
	ratio := netBuy / vol * 100
	if ratio >= 10.0 {
		return 100
	}
	if ratio <= 0 {
		return 0
	}
	return clamp(ratio*10, 0, 100)
}

// scoreMicroBidAsk: >= 3.0 -> 100, < 1.0 -> 0. 0 = no data -> 50.
func scoreMicroBidAsk(v float64) float64 {
	if v <= 0 {
		return 50
	}
	if v >= 3.0 {
		return 100
	}
	if v < 1.0 {
		return 0
	}
	return clamp((v-1.0)/2.0*100, 0, 100)
}

// scoreVIDisparity: 1.0%~2.0% -> 100. <0.5% -> 0. >=5.0% -> 0. 0 = no data -> 50.
func scoreVIDisparity(v float64) float64 {
	if v <= 0 {
		return 50 // Could be no data, or below 0 (downward VI? We only care about upward).
	}
	switch {
	case v < 0.5:
		return 0
	case v >= 0.5 && v < 1.0:
		return clamp((v-0.5)*200, 0, 100)
	case v >= 1.0 && v <= 2.0:
		return 100
	case v > 2.0 && v < 5.0:
		return clamp((5.0-v)/3.0*100, 0, 100)
	default:
		return 0
	}
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
