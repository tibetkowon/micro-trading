package simulation

import "fmt"

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

	return scenarios
}

func roundPct(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}
