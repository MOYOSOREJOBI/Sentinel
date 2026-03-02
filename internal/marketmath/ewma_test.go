package marketmath

import "testing"

func TestEWMAUpdate(t *testing.T) {
	e := NewEWMA(0.94)
	e.Update(0.1)
	if !e.Init || e.Var <= 0 {
		t.Fatalf("init")
	}
}
