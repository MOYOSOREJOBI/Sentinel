package scoring

import "math"

func ScoreAndSeverity(prevPrice, price float64) (float64, string) {
	if prevPrice <= 0 || price <= 0 {
		return 0, "stable"
	}
	ret := math.Abs((price / prevPrice) - 1)
	score := math.Max(0, math.Min(1, ret*25.0))
	switch {
	case score >= 0.85:
		return score, "critical"
	case score >= 0.65:
		return score, "high"
	case score >= 0.40:
		return score, "elevated"
	default:
		return score, "stable"
	}
}
