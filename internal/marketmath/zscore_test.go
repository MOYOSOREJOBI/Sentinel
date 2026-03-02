package marketmath

import "testing"

func TestZScoreClip(t *testing.T) {
	if ZScore(100, 0, 1) != 8 {
		t.Fatalf("clip hi")
	}
	if ZScore(-100, 0, 1) != -8 {
		t.Fatalf("clip lo")
	}
}
