package replay

import (
	"testing"

	pipelinescoring "sentinel/internal/pipeline/scoring"
)

func TestReplayScoreAndSeverityBounds(t *testing.T) {
	s, sev := pipelinescoring.ScoreAndSeverity(100, 100)
	if s != 0 || sev != "stable" {
		t.Fatalf("expected stable zero, got %f %s", s, sev)
	}

	s, sev = pipelinescoring.ScoreAndSeverity(100, 120)
	if s <= 0 || s > 1 {
		t.Fatalf("score out of bounds: %f", s)
	}
	if sev != "critical" {
		t.Fatalf("expected critical, got %s", sev)
	}
}

func TestReplayScoreAndSeverityMissingBaseline(t *testing.T) {
	s, sev := pipelinescoring.ScoreAndSeverity(0, 101)
	if s != 0 || sev != "stable" {
		t.Fatalf("expected stable for missing baseline, got %f %s", s, sev)
	}
}
