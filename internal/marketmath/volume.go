package marketmath

func VolumeRatio(vol, mean float64) float64 {
	if mean < 1e-9 {
		return 0
	}
	return vol / mean
}

func VolumeSurprise(vol, mean, std float64) float64 {
	return ZScore(vol, mean, std)
}
