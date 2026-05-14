package simulation

import (
	"fmt"
	"math"
)

// Scenario is a labeled parameter set for 1D sensitivity analysis.
type Scenario struct {
	Label  string
	Params SimParams
}

// GenerateScenarios varies one parameter at a time while keeping the rest at base values.
func GenerateScenarios(base SimParams) []Scenario {
	scenarios := []Scenario{
		{Label: "현재 설정", Params: base},
	}

	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5, 2.0} {
		p := base
		p.TakeProfitPct = roundPct(base.TakeProfitPct * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("목표가 %.1f%%", p.TakeProfitPct),
			Params: p,
		})
	}

	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5, 2.0} {
		p := base
		p.StopLossPct = roundPct(base.StopLossPct * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("손절 %.1f%%", p.StopLossPct),
			Params: p,
		})
	}

	triggerBase := base.TrailingTriggerPct
	if triggerBase == 0 {
		triggerBase = 1.0
	}
	for _, val := range []float64{0, triggerBase * 0.5, triggerBase * 0.75, triggerBase * 1.25, triggerBase * 1.5} {
		if val == base.TrailingTriggerPct {
			continue
		}
		p := base
		p.TrailingTriggerPct = roundPct(val)
		label := fmt.Sprintf("트레일링 트리거 %.1f%%", p.TrailingTriggerPct)
		if val == 0 {
			label = "트레일링 비활성"
		}
		scenarios = append(scenarios, Scenario{Label: label, Params: p})
	}

	stopBase := base.TrailingStopPct
	if stopBase == 0 {
		stopBase = 0.5
	}
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5} {
		p := base
		p.TrailingStopPct = roundPct(stopBase * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("트레일링 스탑 %.1f%%", p.TrailingStopPct),
			Params: p,
		})
	}

	scoreBase := base.MinScoreThreshold
	if scoreBase == 0 {
		scoreBase = 50.0
	}
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5} {
		p := base
		p.MinScoreThreshold = roundPct(scoreBase * mult)
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("최소점수 %.1f", p.MinScoreThreshold),
			Params: p,
		})
	}

	coolBase := float64(base.UniversalCooldownMin)
	if coolBase == 0 {
		coolBase = 20.0
	}
	for _, mult := range []float64{0.5, 0.75, 1.25, 1.5} {
		p := base
		p.UniversalCooldownMin = int(math.Round(coolBase * mult))
		if p.UniversalCooldownMin < 1 {
			p.UniversalCooldownMin = 1
		}
		scenarios = append(scenarios, Scenario{
			Label:  fmt.Sprintf("쿨타임 %d분", p.UniversalCooldownMin),
			Params: p,
		})
	}

	type wspec struct {
		name string
		get  func(SimParams) int
		set  func(*SimParams, int)
	}
	wspecs := []wspec{
		{"강도", func(p SimParams) int { return p.WeightStrength }, func(p *SimParams, v int) { p.WeightStrength = v }},
		{"RSI", func(p SimParams) int { return p.WeightRSI }, func(p *SimParams, v int) { p.WeightRSI = v }},
		{"MACD", func(p SimParams) int { return p.WeightMACD }, func(p *SimParams, v int) { p.WeightMACD = v }},
		{"호가", func(p SimParams) int { return p.WeightBidAsk }, func(p *SimParams, v int) { p.WeightBidAsk = v }},
		{"VWAP", func(p SimParams) int { return p.WeightVWAP }, func(p *SimParams, v int) { p.WeightVWAP = v }},
		{"거래량", func(p SimParams) int { return p.WeightVolume }, func(p *SimParams, v int) { p.WeightVolume = v }},
		{"프로그램매수", func(p SimParams) int { return p.WeightProgramBuy }, func(p *SimParams, v int) { p.WeightProgramBuy = v }},
		{"미시호가", func(p SimParams) int { return p.WeightMicroBidAsk }, func(p *SimParams, v int) { p.WeightMicroBidAsk = v }},
		{"VI이격", func(p SimParams) int { return p.WeightVIDisparity }, func(p *SimParams, v int) { p.WeightVIDisparity = v }},
	}
	for _, ws := range wspecs {
		baseVal := ws.get(base)
		if baseVal == 0 {
			baseVal = 10
		}
		for _, mult := range []float64{0.5, 0.75, 1.25, 1.5} {
			newVal := int(math.Round(float64(baseVal) * mult))
			if newVal < 1 {
				newVal = 1
			}
			p := base
			ws.set(&p, newVal)
			scenarios = append(scenarios, Scenario{
				Label:  fmt.Sprintf("가중치-%s %d", ws.name, newVal),
				Params: p,
			})
		}
	}

	return scenarios
}

func roundPct(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
