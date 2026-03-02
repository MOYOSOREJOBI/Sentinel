package marketmath

func ZScore(x, mean, std float64) float64 {
	if std < 1e-9 {
		return 0
	}
	z := (x - mean) / std
	if z > 8 {
		return 8
	}
	if z < -8 {
		return -8
	}
	return z
}
