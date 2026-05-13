package simulation_test

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/simulation"
)

func TestGenerateScenarios_CountAndBaseline(t *testing.T) {
	base := simulation.SimParams{
		TakeProfitPct:      3.0,
		StopLossPct:        2.0,
		TrailingTriggerPct: 1.5,
		TrailingStopPct:    0.5,
	}
	scenarios := simulation.GenerateScenarios(base)

	if len(scenarios) < 20 {
		t.Errorf("want at least 20 scenarios, got %d", len(scenarios))
	}
	if scenarios[0].Label != "현재 설정" {
		t.Errorf("first scenario should be baseline, got %q", scenarios[0].Label)
	}
}

func TestGenerateScenarios_LabelNotEmpty(t *testing.T) {
	base := simulation.SimParams{TakeProfitPct: 2.0, StopLossPct: 1.5}
	scenarios := simulation.GenerateScenarios(base)
	for i, s := range scenarios {
		if s.Label == "" {
			t.Errorf("scenario[%d] has empty label", i)
		}
	}
}
