package monitor

import "testing"

func makeTrailPos(filledPrice float64, ts TickTrailState) *MonitoredEntry {
	return &MonitoredEntry{
		FilledPrice:  filledPrice,
		TrailingMode: "tick",
		TickTrail:    ts,
	}
}

func TestEvaluateTickTrail_Tier0_StopLoss(t *testing.T) {
	// 진입가 10000원, 틱사이즈=50원, 3틱 손절 → 9850 이하에서 청산
	pos := makeTrailPos(10000, TickTrailState{Tier0StopLossTicks: 3})
	// bid1=9851 → 유지
	sell, _ := evaluateTickTrail(pos, 10000, 9851)
	if sell {
		t.Error("9851 should not trigger stop (stop=9850)")
	}
	// bid1=9850 → 청산
	sell, reason := evaluateTickTrail(pos, 10000, 9850)
	if !sell {
		t.Error("9850 should trigger Tier0 stop")
	}
	if reason != "틱트레일-Tier0-진입손절" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_Tier1_Activation(t *testing.T) {
	// 진입가 10000원, Tier1 발동 +2%, 트레일 3틱(50원=150원)
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
	})
	// 체결가 10200 (=+2%) → Tier1 활성화, peak=10200
	sell, _ := evaluateTickTrail(pos, 10200, 10200)
	if sell {
		t.Error("should not sell on activation tick")
	}
	if pos.TickTrail.CurrentTier != 1 {
		t.Errorf("expected Tier1, got %d", pos.TickTrail.CurrentTier)
	}
	if pos.TickTrail.PeakBid1Price != 10200 {
		t.Errorf("expected peak=10200, got %f", pos.TickTrail.PeakBid1Price)
	}
}

func TestEvaluateTickTrail_Tier1_Trail(t *testing.T) {
	// Tier1 이미 활성, peak=10400, 3틱=150원 → 10250 이하에서 청산
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
		CurrentTier:     1,
		PeakBid1Price:   10400,
	})
	// bid1=10251 → 유지
	sell, _ := evaluateTickTrail(pos, 10300, 10251)
	if sell {
		t.Error("10251 should not trigger trail (stop=10250)")
	}
	// bid1=10250 → 청산
	sell, reason := evaluateTickTrail(pos, 10200, 10250)
	if !sell {
		t.Error("10250 should trigger Tier1 trail")
	}
	if reason != "틱트레일-Tier1-브레이크이븐" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_Tier2_SkipsTier1(t *testing.T) {
	// A=2%, B=4%, 동시에 +5% 도달 시 Tier2 직접 진입
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 5,
		Tier2TriggerPct: 4.0,
		Tier2TrailTicks: 2,
	})
	evaluateTickTrail(pos, 10500, 10500) // +5%
	if pos.TickTrail.CurrentTier != 2 {
		t.Errorf("expected Tier2, got %d", pos.TickTrail.CurrentTier)
	}
}

func TestEvaluateTickTrail_Tier2_Trail(t *testing.T) {
	// Tier2, peak=50000(틱사이즈=100원), 2틱=200원 → 49800 이하 청산
	pos := makeTrailPos(48000, TickTrailState{
		Tier2TriggerPct: 3.0,
		Tier2TrailTicks: 2,
		CurrentTier:     2,
		PeakBid1Price:   50000,
	})
	sell, reason := evaluateTickTrail(pos, 49900, 49800)
	if !sell {
		t.Error("49800 should trigger Tier2 trail")
	}
	if reason != "틱트레일-Tier2-급등익절" {
		t.Errorf("wrong reason: %s", reason)
	}
}

func TestEvaluateTickTrail_PeakUpdate(t *testing.T) {
	// Tier1 활성, peak 갱신 확인
	pos := makeTrailPos(10000, TickTrailState{
		Tier1TriggerPct: 2.0,
		Tier1TrailTicks: 3,
		CurrentTier:     1,
		PeakBid1Price:   10200,
	})
	evaluateTickTrail(pos, 10400, 10400)
	if pos.TickTrail.PeakBid1Price != 10400 {
		t.Errorf("peak should update to 10400, got %f", pos.TickTrail.PeakBid1Price)
	}
}
