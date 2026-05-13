package monitor

import "testing"

func TestCalcTickSize(t *testing.T) {
	cases := []struct {
		price float64
		want  float64
	}{
		{500, 1},
		{999, 1},
		{1000, 5},
		{4999, 5},
		{5000, 10},
		{9999, 10},
		{10000, 50},
		{49999, 50},
		{50000, 100},
		{99999, 100},
		{100000, 500},
		{499999, 500},
		{500000, 1000},
		{1000000, 1000},
		{0, 0},
		{-500, 0},
	}
	for _, tc := range cases {
		got := CalcTickSize(tc.price)
		if got != tc.want {
			t.Errorf("CalcTickSize(%.0f) = %.0f, want %.0f", tc.price, got, tc.want)
		}
	}
}
