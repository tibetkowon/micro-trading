// Package scorer implements the rule-based hard filter and composite scoring system (v2).
// Candidates are first filtered by hard rules; those that pass are scored 0-100 per indicator
// and combined with configurable weights into a final score.
package scorer

import (
	"fmt"
	"math"

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
	Strength float64
	RSI      float64
	MACD     float64
	BidAsk   float64
	VWAP     float64
	Volume   float64
	Total    float64 // weighted sum normalised to 0-100
}

// String returns a compact log-friendly representation.
func (d ScoreDetail) String() string {
	return fmt.Sprintf("total=%.1f str=%.1f rsi=%.1f macd=%.1f bid=%.1f vwap=%.1f vol=%.1f",
		d.Total, d.Strength, d.RSI, d.MACD, d.BidAsk, d.VWAP, d.Volume)
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
// Each indicator is scored 0-100, then blended by configured weights.
// Indicators with no data return 50 (neutral) so missing data doesn't unfairly penalise.
func CalcScore(c CandidateInfo, s database.TradingSettings) ScoreDetail {
	info := c.Info
	d := ScoreDetail{
		Strength: scoreStrength(c.Strength),
		RSI:      scoreRSI(info.RSI14),
		MACD:     scoreMACD(info.MACDLine, info.MACDSignal),
		BidAsk:   scoreBidAsk(info.BidAskRatio),
		VWAP:     scoreVWAP(info.VWAPDiff, info.VWAP),
		Volume:   scoreVolume(info.VolVs3AvgRatio),
	}

	totalWeight := float64(s.ScoreWeightStrength + s.ScoreWeightRSI + s.ScoreWeightMACD +
		s.ScoreWeightBidAsk + s.ScoreWeightVWAP + s.ScoreWeightVolume)
	if totalWeight == 0 {
		return d // all weights zero → total stays 0
	}

	d.Total = math.Round((d.Strength*float64(s.ScoreWeightStrength)+
		d.RSI*float64(s.ScoreWeightRSI)+
		d.MACD*float64(s.ScoreWeightMACD)+
		d.BidAsk*float64(s.ScoreWeightBidAsk)+
		d.VWAP*float64(s.ScoreWeightVWAP)+
		d.Volume*float64(s.ScoreWeightVolume))/totalWeight*10) / 10

	return d
}

// scoreStrength: >= 130 → 100, 100-130 → linear, < 100 → 0. 0 = no data → 50.
func scoreStrength(v float64) float64 {
	if v <= 0 {
		return 50
	}
	if v >= 130 {
		return 100
	}
	if v < 100 {
		return 0
	}
	return clamp((v-100)/30*100, 0, 100)
}

// scoreRSI: 40-60 → 100 (optimal), 30-40 and 60-70 → linear ramps, outside → 0. 0 = no data → 50.
func scoreRSI(v float64) float64 {
	if v <= 0 {
		return 50
	}
	switch {
	case v >= 70 || v <= 30:
		return 0
	case v >= 40 && v <= 60:
		return 100
	case v > 60:
		return clamp((70-v)/10*100, 0, 100)
	default: // 30 < v < 40
		return clamp((v-30)/10*100, 0, 100)
	}
}

// scoreMACD: golden cross (line > signal) → 100, dead cross → 0. both 0 = no data → 50.
func scoreMACD(line, signal float64) float64 {
	if line == 0 && signal == 0 {
		return 50
	}
	if line > signal {
		return 100
	}
	return 0
}

// scoreBidAsk: >= 2.0 → 100, 1.0-2.0 → linear, < 1.0 → 0. 0 = no data → 50.
func scoreBidAsk(v float64) float64 {
	if v <= 0 {
		return 50
	}
	if v >= 2.0 {
		return 100
	}
	if v < 1.0 {
		return 0
	}
	return clamp((v-1.0)*100, 0, 100)
}

// scoreVWAP: -1% to +3% → 100 (trading near/above VWAP), outside → linear decay. 0 = no data → 50.
func scoreVWAP(diff, vwap float64) float64 {
	if vwap <= 0 {
		return 50
	}
	switch {
	case diff >= -1 && diff <= 3:
		return 100
	case diff > 3:
		return clamp((6-diff)/3*100, 0, 100)
	default: // diff < -1: -1 to -4 → linear 100→0
		return clamp((diff+4)/3*100, 0, 100)
	}
}

// scoreVolume: >= 2.0 → 100, 1.0-2.0 → linear, < 1.0 → 0. 0 = no data → 50.
func scoreVolume(v float64) float64 {
	if v <= 0 {
		return 50
	}
	if v >= 2.0 {
		return 100
	}
	if v < 1.0 {
		return 0
	}
	return clamp((v-1.0)*100, 0, 100)
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}
