package replay

import (
	"testing"
	"time"
)

func TestSummarizeReplayDiffParityMatchForIdenticalRows(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	asScored := []scorePoint{
		{Key: "AAA|1", Symbol: "AAA", Timestamp: now, Score: 0.25, Severity: "stable", Explanation: "stored"},
		{Key: "BBB|1", Symbol: "BBB", Timestamp: now.Add(time.Second), Score: 0.72, Severity: "high", Explanation: "stored"},
	}
	recomputed := []scorePoint{
		{Key: "AAA|1", Symbol: "AAA", Timestamp: now, Score: 0.25, Severity: "stable", Explanation: "stored"},
		{Key: "BBB|1", Symbol: "BBB", Timestamp: now.Add(time.Second), Score: 0.72, Severity: "high", Explanation: "stored"},
	}

	summary := summarizeReplayDiff(Result{TickCount: 2, ScoreCount: 2}, asScored, recomputed, "", "")
	if summary.ParityStatus != "MATCH" {
		t.Fatalf("expected MATCH, got %s", summary.ParityStatus)
	}
	if summary.MatchedCount != 2 || summary.MismatchedCount != 0 {
		t.Fatalf("unexpected parity counts: %+v", summary)
	}
	if summary.MaxScoreDelta != 0 {
		t.Fatalf("expected zero delta, got %f", summary.MaxScoreDelta)
	}
}

func TestSummarizeReplayDiffDeterminismMatchesPreviousFingerprint(t *testing.T) {
	now := time.Date(2026, 3, 4, 10, 0, 0, 0, time.UTC)
	rows := []scorePoint{
		{Key: "AAA|1", Symbol: "AAA", Timestamp: now, Score: 0.33, Severity: "elevated", Explanation: "stored"},
		{Key: "BBB|1", Symbol: "BBB", Timestamp: now.Add(time.Second), Score: 0.91, Severity: "critical", Explanation: "stored"},
	}
	previousFingerprint := fingerprintScoreRows(rows)

	summary := summarizeReplayDiff(Result{TickCount: 2, ScoreCount: 2}, rows, rows, "prior-job", previousFingerprint)
	if summary.DeterminismStatus != "MATCH" {
		t.Fatalf("expected determinism MATCH, got %s", summary.DeterminismStatus)
	}
	if summary.PreviousReplayID != "prior-job" {
		t.Fatalf("expected prior replay id, got %s", summary.PreviousReplayID)
	}
	if summary.RecomputedFingerprint != previousFingerprint {
		t.Fatalf("expected fingerprint reuse")
	}
}
