package marketmath

import "testing"

func TestDQPenaltyBounded(t *testing.T) {
	v := DQPenalty(1, 1, 1, 1, 1)
	if v != 1 {
		t.Fatalf("expected 1 got %v", v)
	}
}
