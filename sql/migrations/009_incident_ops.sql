-- +goose Up
CREATE TABLE IF NOT EXISTS incident_notifications (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  notification_type TEXT NOT NULL DEFAULT 'on_call',
  target TEXT NOT NULL DEFAULT '',
  channel TEXT NOT NULL DEFAULT 'webhook_stub',
  status TEXT NOT NULL DEFAULT 'queued',
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_by TEXT NOT NULL DEFAULT 'system',
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_incident_notifications_incident_created ON incident_notifications (incident_id, created_at DESC);

CREATE TABLE IF NOT EXISTS forecast_snapshots (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  generated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  window_label TEXT NOT NULL DEFAULT '24h',
  incident_band_low INT NOT NULL DEFAULT 0,
  incident_band_base INT NOT NULL DEFAULT 0,
  incident_band_high INT NOT NULL DEFAULT 0,
  velocity DOUBLE PRECISION NOT NULL DEFAULT 0,
  dq_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  dispersion DOUBLE PRECISION NOT NULL DEFAULT 0,
  tail_risk_p95 DOUBLE PRECISION NOT NULL DEFAULT 0,
  watchlist_risks TEXT[] NOT NULL DEFAULT '{}',
  why_moved TEXT[] NOT NULL DEFAULT '{}',
  created_by TEXT NOT NULL DEFAULT 'system'
);
CREATE INDEX IF NOT EXISTS idx_forecast_snapshots_generated_at ON forecast_snapshots (generated_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_forecast_snapshots_generated_at;
DROP TABLE IF EXISTS forecast_snapshots;

DROP INDEX IF EXISTS idx_incident_notifications_incident_created;
DROP TABLE IF EXISTS incident_notifications;
