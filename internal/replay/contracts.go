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

type DiffSummary struct {
	Result         Result     `json:"result"`
	AsScored       ScoreStats `json:"as_scored"`
	Recomputed     ScoreStats `json:"recomputed"`
	AvgScoreDelta  float64    `json:"avg_score_delta"`
	HighCountDelta int        `json:"high_count_delta"`
}
