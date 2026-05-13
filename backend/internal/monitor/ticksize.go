package monitor

// CalcTickSize returns the KRX 호가 단위 (tick size) for a given price.
func CalcTickSize(price float64) float64 {
	switch {
	case price < 1_000:
		return 1
	case price < 5_000:
		return 5
	case price < 10_000:
		return 10
	case price < 50_000:
		return 50
	case price < 100_000:
		return 100
	case price < 500_000:
		return 500
	default:
		return 1_000
	}
}
