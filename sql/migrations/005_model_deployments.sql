-- +goose Up
CREATE TABLE IF NOT EXISTS model_deployments (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  model_name TEXT NOT NULL,
  model_version TEXT NOT NULL,
  artifact_hash TEXT NOT NULL,
  artifact_path TEXT NOT NULL,
  feature_set_version TEXT NOT NULL,
  calibration_version TEXT,
  status TEXT NOT NULL CHECK (status IN ('draft','pending','approved','deployed','retired')),
  training_window_start TIMESTAMPTZ,
  training_window_end TIMESTAMPTZ,
  created_by TEXT NOT NULL,
  approved_by TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  approved_at TIMESTAMPTZ,
  deployed_at TIMESTAMPTZ,
  retired_at TIMESTAMPTZ,
  change_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_model_deployments_status
  ON model_deployments (status, deployed_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_model_deployments_status;
DROP TABLE IF EXISTS model_deployments;
