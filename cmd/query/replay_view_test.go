package main

import (
	"encoding/json"
	"testing"
)

func TestCoerceReplaySummarySupportsJSONBytes(t *testing.T) {
	raw := []byte(`{"parity_status":"MATCH","matched_count":9}`)
	summary := coerceReplaySummary(raw)
	if summary["parity_status"] != "MATCH" {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestReplayBriefPayloadIncludesProvenance(t *testing.T) {
	job := map[string]any{
		"id":                "job-1",
		"timeWindowStart":   "2026-03-04T10:00:00Z",
		"timeWindowEnd":     "2026-03-04T10:05:00Z",
		"modelVersion":      "baseline-v2",
		"featureSetVersion": "v3",
		"watermarkPolicy":   "wm_v1",
		"allowedLatenessMs": 5000,
		"requestedAt":       "2026-03-04T10:06:00Z",
		"startedAt":         "2026-03-04T10:06:01Z",
		"completedAt":       "2026-03-04T10:06:02Z",
	}
	summary := map[string]any{
		"parity_status":          "MATCH",
		"determinism_status":     "MATCH",
		"matched_count":          11,
		"mismatched_count":       0,
		"max_score_delta":        0.0,
		"common_mismatch_causes": []any{},
	}

	payload := replayBriefPayload(job, summary)
	if payload["replayId"] != "job-1" {
		t.Fatalf("missing replay id: %+v", payload)
	}
	if payload["parityStatus"] != "MATCH" {
		t.Fatalf("expected parity status: %+v", payload)
	}
	if payload["determinismStatus"] != "MATCH" {
		t.Fatalf("expected determinism status: %+v", payload)
	}
	pinned, ok := payload["pinnedEvidence"].([]map[string]any)
	if !ok || len(pinned) != 2 {
		t.Fatalf("expected pinned evidence links: %+v", payload["pinnedEvidence"])
	}
	if _, err := json.Marshal(payload); err != nil {
		t.Fatalf("payload should be serializable: %v", err)
	}
}
