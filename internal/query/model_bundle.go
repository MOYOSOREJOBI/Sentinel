package query

import (
	"encoding/json"
	"os"
)

type modelBundleMetadata struct {
	ModelVersion       string
	TrainedAt          string
	ArtifactHash       string
	FallbackMode       bool
	CalibrationVersion string
	Metrics            struct {
		AUC              float64
		Brier            float64
		PRAUC            float64
		AnomalyStability float64
	}
}

func readModelBundleMetadata() modelBundleMetadata {
	paths := []string{
		os.Getenv("INFERENCE_MODEL_METADATA_PATH"),
		"/app/inference-models/metadata.json",
		"services/inference/models/metadata.json",
	}
	for _, path := range paths {
		if path == "" {
			continue
		}
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var raw struct {
			ModelVersion       string `json:"model_version"`
			TrainedAt          string `json:"trained_at"`
			ArtifactHash       string `json:"artifact_hash"`
			FallbackMode       bool   `json:"fallback_mode"`
			CalibrationVersion string `json:"calibration_version"`
			Metrics            struct {
				AUC              float64 `json:"auc"`
				Brier            float64 `json:"brier"`
				PRAUC            float64 `json:"pr_auc"`
				AnomalyStability float64 `json:"anomaly_stability"`
			} `json:"metrics"`
		}
		if err := json.Unmarshal(b, &raw); err != nil {
			continue
		}
		return modelBundleMetadata{
			ModelVersion:       raw.ModelVersion,
			TrainedAt:          raw.TrainedAt,
			ArtifactHash:       raw.ArtifactHash,
			FallbackMode:       raw.FallbackMode,
			CalibrationVersion: raw.CalibrationVersion,
			Metrics: struct {
				AUC              float64
				Brier            float64
				PRAUC            float64
				AnomalyStability float64
			}{
				AUC:              raw.Metrics.AUC,
				Brier:            raw.Metrics.Brier,
				PRAUC:            raw.Metrics.PRAUC,
				AnomalyStability: raw.Metrics.AnomalyStability,
			},
		}
	}
	return modelBundleMetadata{}
}
