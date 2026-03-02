package marketmath

import "testing"

func TestEWMAVolHasNumericalFloor(t *testing.T) {
	e := NewEWMA(0.94)
	if v := e.Vol(); v <= 0 {
		t.Fatalf("expected positive floor vol, got %f", v)
	}
}
