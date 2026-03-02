package marketmath

func Clip(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func DQPenalty(stalenessScore, lateRatio, outOfOrderRatio, duplicateRatio, missingnessRatio float64) float64 {
	return Clip(
		0.35*stalenessScore+
			0.25*lateRatio+
			0.20*outOfOrderRatio+
			0.10*duplicateRatio+
			0.10*missingnessRatio,
		0,
		1,
	)
}
