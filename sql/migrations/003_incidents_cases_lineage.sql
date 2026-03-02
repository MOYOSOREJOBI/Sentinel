-- +goose Up
ALTER TABLE candles
  ADD COLUMN IF NOT EXISTS open_event_time TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS close_event_time TIMESTAMPTZ;

ALTER TABLE scores
  ADD COLUMN IF NOT EXISTS raw_anomaly_score DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS normalized_anomaly_score DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS escalation_probability DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS priority_score DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS composite_risk DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS feature_snapshot_hash TEXT,
  ADD COLUMN IF NOT EXISTS feature_set_version TEXT DEFAULT 'v1',
  ADD COLUMN IF NOT EXISTS model_version TEXT DEFAULT 'baseline-v1',
  ADD COLUMN IF NOT EXISTS calibration_version TEXT,
  ADD COLUMN IF NOT EXISTS explanation_payload JSONB DEFAULT '{}'::jsonb;

CREATE TABLE IF NOT EXISTS incidents (
  id BIGSERIAL PRIMARY KEY,
  primary_symbol TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','ack','resolved','suppressed','hibernating')),
  severity_band TEXT NOT NULL DEFAULT 'elevated',
  priority_score DOUBLE PRECISION NOT NULL DEFAULT 0,
  composite_risk DOUBLE PRECISION NOT NULL DEFAULT 0,
  escalation_probability DOUBLE PRECISION NOT NULL DEFAULT 0,
  confidence DOUBLE PRECISION NOT NULL DEFAULT 1,
  trust_state TEXT NOT NULL DEFAULT 'stable',
  feature_snapshot_hash TEXT NOT NULL DEFAULT '',
  feature_set_version TEXT NOT NULL DEFAULT 'v1',
  model_version TEXT NOT NULL DEFAULT 'baseline-v1',
  calibration_version TEXT,
  top_driver_1 TEXT NOT NULL DEFAULT '',
  top_driver_2 TEXT NOT NULL DEFAULT '',
  top_driver_3 TEXT NOT NULL DEFAULT '',
  driver_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  owner_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS incident_alert_links (
  incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
  alert_id BIGINT NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
  linked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  PRIMARY KEY (incident_id, alert_id)
);

CREATE TABLE IF NOT EXISTS cases (
  id BIGSERIAL PRIMARY KEY,
  incident_id BIGINT NOT NULL REFERENCES incidents(id) ON DELETE RESTRICT,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','investigating','escalated','closed')),
  reason TEXT NOT NULL,
  owner_name TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS case_notes (
  id BIGSERIAL PRIMARY KEY,
  case_id BIGINT NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  actor TEXT NOT NULL,
  note TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS case_actions (
  id BIGSERIAL PRIMARY KEY,
  case_id BIGINT NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS case_evidence (
  id BIGSERIAL PRIMARY KEY,
  case_id BIGINT NOT NULL REFERENCES cases(id) ON DELETE CASCADE,
  evidence_type TEXT NOT NULL,
  reference_id TEXT NOT NULL,
  metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS instrument_metadata (
  instrument_id TEXT PRIMARY KEY,
  venue TEXT NOT NULL DEFAULT '',
  asset_class TEXT NOT NULL DEFAULT '',
  country TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  state_province TEXT NOT NULL DEFAULT '',
  sector TEXT NOT NULL DEFAULT '',
  industry TEXT NOT NULL DEFAULT '',
  benchmark_id TEXT NOT NULL DEFAULT ''
);

ALTER TABLE alerts ADD COLUMN IF NOT EXISTS incident_id BIGINT;
CREATE INDEX IF NOT EXISTS alerts_score_id_idx ON alerts (score_id);
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_incident_id_fkey;
ALTER TABLE alerts ADD CONSTRAINT alerts_incident_id_fkey FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_incidents_queue ON incidents(status, severity_band, priority_score DESC, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_incidents_symbol_time ON incidents(primary_symbol, last_activity_at DESC);
CREATE INDEX IF NOT EXISTS idx_instrument_metadata_geo ON instrument_metadata(country, region, sector, industry);

-- +goose Down
DROP INDEX IF EXISTS idx_instrument_metadata_geo;
DROP INDEX IF EXISTS alerts_score_id_idx;
DROP INDEX IF EXISTS idx_incidents_symbol_time;
DROP INDEX IF EXISTS idx_incidents_queue;
ALTER TABLE alerts DROP CONSTRAINT IF EXISTS alerts_incident_id_fkey;
ALTER TABLE alerts DROP COLUMN IF EXISTS incident_id;
DROP TABLE IF EXISTS instrument_metadata;
DROP TABLE IF EXISTS case_evidence;
DROP TABLE IF EXISTS case_actions;
DROP TABLE IF EXISTS case_notes;
DROP TABLE IF EXISTS cases;
DROP TABLE IF EXISTS incident_alert_links;
DROP TABLE IF EXISTS incidents;
ALTER TABLE candles
  ADD COLUMN IF NOT EXISTS open_event_time TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS close_event_time TIMESTAMPTZ;

ALTER TABLE scores
  DROP COLUMN IF EXISTS explanation_payload,
  DROP COLUMN IF EXISTS calibration_version,
  DROP COLUMN IF EXISTS model_version,
  DROP COLUMN IF EXISTS feature_set_version,
  DROP COLUMN IF EXISTS feature_snapshot_hash,
  DROP COLUMN IF EXISTS composite_risk,
  DROP COLUMN IF EXISTS priority_score,
  DROP COLUMN IF EXISTS escalation_probability,
  DROP COLUMN IF EXISTS normalized_anomaly_score,
  DROP COLUMN IF EXISTS raw_anomaly_score;
