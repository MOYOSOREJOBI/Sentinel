package logic

import "testing"

func TestSeverity(t *testing.T) {
	if Severity(0.9) != "critical" || Severity(0.5) != "medium" {
		t.Fatal("bad severity")
	}
}
func TestState(t *testing.T) {
	if Transition(Draft, "approve") != Approved {
		t.Fatal("bad transition")
	}
}
func TestFeature(t *testing.T) {
	if FeatureLogReturn(100) != 0 {
		t.Fatal("expected 0")
	}
}
