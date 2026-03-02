package marketmath

import "math"

func RealizedVariance(r *Ring) float64 {
	if r == nil {
		return 0
	}
	return r.sumSq
}

func RealizedVol(r *Ring) float64 {
	return math.Sqrt(math.Max(RealizedVariance(r), 0))
}
