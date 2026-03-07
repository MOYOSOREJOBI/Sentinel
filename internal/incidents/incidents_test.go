package incidents

import "testing"

func TestSeverityBandForScore(t *testing.T) {
	cases := []struct {
		score float64
		want  string
	}{
		{score: 0.2, want: "stable"},
		{score: 0.45, want: "elevated"},
		{score: 0.7, want: "high"},
		{score: 0.85, want: "critical"},
	}
	for _, tc := range cases {
		if got := SeverityBandForScore(tc.score); got != tc.want {
			t.Fatalf("SeverityBandForScore(%v) = %q, want %q", tc.score, got, tc.want)
		}
	}
}
