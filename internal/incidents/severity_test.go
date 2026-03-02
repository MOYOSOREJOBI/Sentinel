package incidents

import "testing"

func TestSeverityDQOverride(t *testing.T) {
	if SeverityFromPriority(99, 0.9, 0.61) != "Data Unreliable" {
		t.Fatal("dq override")
	}
}
