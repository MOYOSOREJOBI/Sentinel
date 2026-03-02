package scoring

import "testing"

func TestScoreAndSeverity(t *testing.T) {
	s, sev := ScoreAndSeverity(100, 101)
	if s <= 0 || sev == "" {
		t.Fatalf("unexpected score=%v sev=%q", s, sev)
	}
	if s, sev := ScoreAndSeverity(0, 100); s != 0 || sev != "stable" {
		t.Fatalf("expected stable zero, got %v %q", s, sev)
	}
}
