-- +goose Up
CREATE TABLE IF NOT EXISTS replay_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  requested_by TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued','running','completed','failed')),
  requested_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  started_at TIMESTAMPTZ,
  completed_at TIMESTAMPTZ,
  error_message TEXT,
  time_window_start TIMESTAMPTZ NOT NULL,
  time_window_end TIMESTAMPTZ NOT NULL,
  replay_mode TEXT NOT NULL DEFAULT 'recompute',
  watermark_policy_id TEXT NOT NULL DEFAULT 'wm_v1',
  allowed_lateness_ms INT NOT NULL DEFAULT 5000,
  model_version TEXT NOT NULL,
  feature_set_version TEXT NOT NULL,
  calibration_version TEXT,
  source_type TEXT NOT NULL DEFAULT 'raw_ticks'
);

CREATE INDEX IF NOT EXISTS idx_replay_jobs_status_req
  ON replay_jobs (status, requested_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_replay_jobs_status_req;
DROP TABLE IF EXISTS replay_jobs;
