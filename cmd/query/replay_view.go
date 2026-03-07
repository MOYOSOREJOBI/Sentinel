package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func coerceReplaySummary(raw any) map[string]any {
	switch v := raw.(type) {
	case map[string]any:
		return v
	case []byte:
		out := map[string]any{}
		if json.Unmarshal(v, &out) == nil {
			return out
		}
	case string:
		out := map[string]any{}
		if json.Unmarshal([]byte(v), &out) == nil {
			return out
		}
	}
	return map[string]any{}
}

func replaySummaryValue(summary map[string]any, key string, fallback any) any {
	if value, ok := summary[key]; ok {
		return value
	}
	return fallback
}

func replayBriefPayload(job map[string]any, summary map[string]any) map[string]any {
	id := fmt.Sprintf("%v", job["id"])
	return map[string]any{
		"replayId":             id,
		"window":               map[string]any{"start": job["timeWindowStart"], "end": job["timeWindowEnd"]},
		"modelVersion":         job["modelVersion"],
		"featureSetVersion":    job["featureSetVersion"],
		"watermarkPolicy":      job["watermarkPolicy"],
		"allowedLatenessMs":    job["allowedLatenessMs"],
		"parityStatus":         replaySummaryValue(summary, "parity_status", "UNKNOWN"),
		"parityExplanation":    replaySummaryValue(summary, "parity_explanation", "No replay diff summary recorded yet."),
		"determinismStatus":    replaySummaryValue(summary, "determinism_status", "UNKNOWN"),
		"determinismExplain":   replaySummaryValue(summary, "determinism_explanation", "No prior replay fingerprint comparison recorded."),
		"matchedCount":         replaySummaryValue(summary, "matched_count", 0),
		"mismatchedCount":      replaySummaryValue(summary, "mismatched_count", 0),
		"maxScoreDelta":        replaySummaryValue(summary, "max_score_delta", 0),
		"commonMismatchCauses": replaySummaryValue(summary, "common_mismatch_causes", []any{}),
		"pinnedEvidence": []map[string]any{
			{"type": "replay_detail", "path": fmt.Sprintf("/replay/%s", id)},
			{"type": "replay_summary", "path": fmt.Sprintf("/replay/%s/summary", id)},
		},
		"timestamps": map[string]any{
			"requestedAt": job["requestedAt"],
			"startedAt":   job["startedAt"],
			"completedAt": job["completedAt"],
			"exportedAt":  time.Now().UTC().Format(time.RFC3339),
		},
	}
}
