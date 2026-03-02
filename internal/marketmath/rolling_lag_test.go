package marketmath

import "testing"

func TestRingAtLag(t *testing.T) {
	r := NewRing(3)
	r.Push(10)
	r.Push(20)
	r.Push(30)
	if v, ok := r.AtLag(0); !ok || v != 30 {
		t.Fatalf("expected latest 30, got %v ok=%v", v, ok)
	}
	if v, ok := r.AtLag(2); !ok || v != 10 {
		t.Fatalf("expected lag2 10, got %v ok=%v", v, ok)
	}
	r.Push(40) // drops 10
	if v, ok := r.AtLag(2); !ok || v != 20 {
		t.Fatalf("expected lag2 20 after roll, got %v ok=%v", v, ok)
	}
}
