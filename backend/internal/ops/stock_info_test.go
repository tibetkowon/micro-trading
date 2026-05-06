package ops

import (
	"math"
	"testing"
)

func TestCalcMA(t *testing.T) {
	closes := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}

	// MA5: 평균(6,7,8,9,10) = 8.0
	got := calcMA(closes, 5)
	if math.Abs(got-8.0) > 0.01 {
		t.Errorf("MA5 = %.2f, want 8.0", got)
	}

	// 데이터 부족: MA20 = 0
	got = calcMA(closes, 20)
	if got != 0 {
		t.Errorf("MA20 (insufficient) = %.2f, want 0", got)
	}
}

func TestCalcRSI_Insufficient(t *testing.T) {
	// 15개 미만이면 0 반환 (RSI14 needs period+1=15)
	closes := make([]float64, 14)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	got := calcRSI(closes, 14)
	if got != 0 {
		t.Errorf("RSI with 14 bars = %.2f, want 0", got)
	}
}

func TestCalcRSI_AllGain(t *testing.T) {
	// 단조 증가 수열 → RSI = 100
	closes := make([]float64, 20)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	got := calcRSI(closes, 14)
	if got != 100 {
		t.Errorf("RSI (all gain) = %.2f, want 100", got)
	}
}

func TestCalcMACD_Insufficient(t *testing.T) {
	// slowPeriod(26)개 미만이면 (0,0,0) 반환
	closes := make([]float64, 25)
	for i := range closes {
		closes[i] = float64(i + 1)
	}
	line, signal, histo := calcMACD(closes, 12, 26, 9)
	if line != 0 || signal != 0 || histo != 0 {
		t.Errorf("MACD (insufficient) = (%.4f,%.4f,%.4f), want (0,0,0)", line, signal, histo)
	}
}

func TestCalcMA60And120(t *testing.T) {
	// MA60, MA120 — 충분한 데이터로 0이 아닌 값 반환
	closes := make([]float64, 150)
	for i := range closes {
		closes[i] = float64(i + 100)
	}
	ma60 := calcMA(closes, 60)
	ma120 := calcMA(closes, 120)
	if ma60 == 0 {
		t.Error("MA60 should not be 0 with 150 bars")
	}
	if ma120 == 0 {
		t.Error("MA120 should not be 0 with 150 bars")
	}
	// MA120 < MA60 (단조증가 수열에서 더 오래된 평균이 더 낮음)
	if ma120 >= ma60 {
		t.Errorf("MA120 (%.2f) should be less than MA60 (%.2f) for ascending series", ma120, ma60)
	}
}
