package contracts

import "time"

type FeatureVector struct {
	Symbol              string             `json:"symbol"`
	EventTime           time.Time          `json:"event_time"`
	FeatureSetVersion   string             `json:"feature_set_version"`
	AllowedLatenessMS   int                `json:"allowed_lateness_ms"`
	WatermarkPolicyID   string             `json:"watermark_policy_id"`
	FeatureSnapshotHash string             `json:"feature_snapshot_hash"`
	Payload             map[string]float64 `json:"payload"`
}
