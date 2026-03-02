package main

import "testing"

func TestNormalizeReplayMode(t *testing.T) {
	if got := normalizeReplayMode("as_scored"); got != "as_scored" {
		t.Fatalf("expected as_scored, got %s", got)
	}
	if got := normalizeReplayMode("recompute"); got != "recompute" {
		t.Fatalf("expected recompute, got %s", got)
	}
	if got := normalizeReplayMode("unknown"); got != "recompute" {
		t.Fatalf("expected default recompute, got %s", got)
	}
}
