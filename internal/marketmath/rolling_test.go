package marketmath

import "testing"

func TestRingStats(t *testing.T) {
	r := NewRing(3)
	r.Push(1)
	r.Push(2)
	r.Push(3)
	if r.Mean() != 2 {
		t.Fatalf("mean")
	}
	r.Push(4)
	if r.Mean() != 3 {
		t.Fatalf("rolling mean")
	}
}
