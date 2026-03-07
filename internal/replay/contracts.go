package replay

import "time"

type Job struct {
	ID                string
	Status            string
	RequestedBy       string
	TimeWindowStart   time.Time
	TimeWindowEnd     time.Time
	WatermarkPolicyID string
	AllowedLatenessMS int
	ModelVersion      string
	FeatureSetVersion string
	ReplayMode        string
}

type Tick struct {
	EventID    string
	Symbol     string
	Price      float64
	Volume     float64
	EventTime  time.Time
	SequenceID int64
}

type Result struct {
	TickCount    int `json:"tick_count"`
	CandleCount  int `json:"candle_count"`
	FeatureCount int `json:"feature_count"`
	ScoreCount   int `json:"score_count"`
}

type ScoreStats struct {
	Count        int     `json:"count"`
	AvgScore     float64 `json:"avg_score"`
	HighOrHigher int     `json:"high_or_higher"`
}

type MismatchSample struct {
	Key                string  `json:"key"`
	Symbol             string  `json:"symbol"`
	Timestamp          string  `json:"timestamp"`
	AsScoredScore      float64 `json:"as_scored_score"`
	RecomputedScore    float64 `json:"recomputed_score"`
	ScoreDelta         float64 `json:"score_delta"`
	AsScoredSeverity   string  `json:"as_scored_severity"`
	RecomputedSeverity string  `json:"recomputed_severity"`
	Cause              string  `json:"cause"`
}

type DiffSummary struct {
	Result                 Result           `json:"result"`
	AsScored               ScoreStats       `json:"as_scored"`
	Recomputed             ScoreStats       `json:"recomputed"`
	AvgScoreDelta          float64          `json:"avg_score_delta"`
	HighCountDelta         int              `json:"high_count_delta"`
	MatchedCount           int              `json:"matched_count"`
	MismatchedCount        int              `json:"mismatched_count"`
	MissingAsScoredCount   int              `json:"missing_as_scored_count"`
	MissingRecomputedCount int              `json:"missing_recomputed_count"`
	MaxScoreDelta          float64          `json:"max_score_delta"`
	ParityStatus           string           `json:"parity_status"`
	ParityExplanation      string           `json:"parity_explanation"`
	CommonMismatchCauses   []string         `json:"common_mismatch_causes"`
	MismatchSamples        []MismatchSample `json:"mismatch_samples"`
	AsScoredFingerprint    string           `json:"as_scored_fingerprint"`
	RecomputedFingerprint  string           `json:"recomputed_fingerprint"`
	DeterminismStatus      string           `json:"determinism_status"`
	DeterminismExplanation string           `json:"determinism_explanation"`
	PreviousReplayID       string           `json:"previous_replay_id,omitempty"`
}
