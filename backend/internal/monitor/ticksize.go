package monitor

// CalcTickSize returns the KRX 호가 단위 (tick size) for a given price.
// Returns 0 if price <= 0 (invalid input; caller should skip tick trail evaluation).
func CalcTickSize(price float64) float64 {
	if price <= 0 {
		return 0
	}
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
