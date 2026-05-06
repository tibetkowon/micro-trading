package scorer

import (
	"testing"

	"github.com/micro-trading-for-agent/backend/internal/database"
	"github.com/micro-trading-for-agent/backend/internal/ops"
)

func makeSettings(ma60enabled, ma120enabled bool) database.TradingSettings {
	return database.TradingSettings{
		HardMA60SupportEnabled:  ma60enabled,
		HardMA120SupportEnabled: ma120enabled,
	}
}

func TestApplyHardFilter_MA60Support_Reject(t *testing.T) {
	// 현재가(9500)가 MA60(10000) 미만 → 탈락
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "9500",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if result.Passed {
		t.Error("expected rejection when price < MA60 with HardMA60SupportEnabled=true")
	}
}

func TestApplyHardFilter_MA60Support_Pass(t *testing.T) {
	// 현재가(10500)가 MA60(10000) 이상 → 통과
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "10500",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if !result.Passed {
		t.Errorf("expected pass when price >= MA60, got rejection: %s", result.Reason)
	}
}

func TestApplyHardFilter_MA60Disabled_Skips(t *testing.T) {
	// MA60 조건 비활성화 → 현재가 < MA60 이어도 이 조건으로는 탈락하지 않음
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "5000",
			MA60:         10000,
		},
	}
	result := ApplyHardFilter(c, makeSettings(false, false))
	if !result.Passed {
		t.Errorf("expected pass when HardMA60SupportEnabled=false, got: %s", result.Reason)
	}
}

func TestApplyHardFilter_MA120Support_Reject(t *testing.T) {
	// 현재가(9000)가 MA120(9500) 미만 → 탈락
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "9000",
			MA120:        9500,
		},
	}
	result := ApplyHardFilter(c, makeSettings(false, true))
	if result.Passed {
		t.Error("expected rejection when price < MA120 with HardMA120SupportEnabled=true")
	}
}

func TestApplyHardFilter_MA60Zero_Skips(t *testing.T) {
	// MA60=0 (데이터 부족) → 조건 무시
	c := CandidateInfo{
		StockCode: "000001",
		StockName: "테스트",
		Info: &ops.StockInfo{
			CurrentPrice: "5000",
			MA60:         0,
		},
	}
	result := ApplyHardFilter(c, makeSettings(true, false))
	if !result.Passed {
		t.Errorf("expected pass when MA60=0 (no data), got: %s", result.Reason)
	}
}
