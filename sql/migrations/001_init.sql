-- +goose Up
DO $$
BEGIN
  PERFORM pg_advisory_lock(947531);
  BEGIN
    CREATE EXTENSION IF NOT EXISTS timescaledb;
  EXCEPTION
    WHEN duplicate_object THEN
      NULL;
  END;
  PERFORM pg_advisory_unlock(947531);
END $$;

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  role TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS raw_ticks (
  event_id TEXT PRIMARY KEY,
  symbol TEXT NOT NULL,
  price DOUBLE PRECISION NOT NULL,
  volume DOUBLE PRECISION NOT NULL,
  event_time TIMESTAMPTZ NOT NULL,
  replay_run_id TEXT DEFAULT '',
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS candles (
  id BIGSERIAL,
  idempotency_key TEXT NOT NULL,
  symbol TEXT NOT NULL,
  bucket TIMESTAMPTZ NOT NULL,
  interval TEXT NOT NULL,
  open DOUBLE PRECISION NOT NULL,
  high DOUBLE PRECISION NOT NULL,
  low DOUBLE PRECISION NOT NULL,
  close DOUBLE PRECISION NOT NULL,
  volume DOUBLE PRECISION NOT NULL,
  replay_run_id TEXT DEFAULT ''
);
SELECT create_hypertable('candles', by_range('bucket'), if_not_exists=>TRUE);
CREATE UNIQUE INDEX IF NOT EXISTS candles_idempotency_bucket_uq ON candles(idempotency_key, bucket);
CREATE INDEX IF NOT EXISTS candles_id_idx ON candles(id);

CREATE TABLE IF NOT EXISTS features (
  id BIGSERIAL,
  idempotency_key TEXT NOT NULL,
  symbol TEXT NOT NULL,
  ts TIMESTAMPTZ NOT NULL,
  feature_hash TEXT NOT NULL,
  payload JSONB NOT NULL,
  replay_run_id TEXT DEFAULT ''
);
SELECT create_hypertable('features', by_range('ts'), if_not_exists=>TRUE);
CREATE UNIQUE INDEX IF NOT EXISTS features_idempotency_ts_uq ON features(idempotency_key, ts);
CREATE INDEX IF NOT EXISTS features_id_idx ON features(id);

CREATE TABLE IF NOT EXISTS scores (
  id BIGSERIAL,
  idempotency_key TEXT NOT NULL,
  symbol TEXT NOT NULL,
  ts TIMESTAMPTZ NOT NULL,
  score DOUBLE PRECISION NOT NULL,
  severity TEXT NOT NULL,
  explanation TEXT NOT NULL,
  replay_run_id TEXT DEFAULT ''
);
SELECT create_hypertable('scores', by_range('ts'), if_not_exists=>TRUE);
CREATE UNIQUE INDEX IF NOT EXISTS scores_idempotency_ts_uq ON scores(idempotency_key, ts);
CREATE INDEX IF NOT EXISTS scores_id_idx ON scores(id);

CREATE TABLE IF NOT EXISTS alerts (
  id BIGSERIAL PRIMARY KEY,
  score_id BIGINT,
  symbol TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'ack', 'resolved', 'suppressed')),
  justification TEXT DEFAULT '',
  created_at TIMESTAMPTZ DEFAULT now(),
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS circuit_suppression (
  symbol TEXT PRIMARY KEY,
  suppressed BOOLEAN NOT NULL,
  reason TEXT NOT NULL,
  updated_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS model_registry (
  id SERIAL PRIMARY KEY,
  model_name TEXT UNIQUE NOT NULL,
  version TEXT NOT NULL,
  state TEXT NOT NULL,
  metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE TABLE IF NOT EXISTS replay_runs (
  id TEXT PRIMARY KEY,
  incident_id TEXT NOT NULL,
  status TEXT NOT NULL,
  started_at TIMESTAMPTZ DEFAULT now(),
  completed_at TIMESTAMPTZ,
  diff_summary JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS audit_log (
  id BIGSERIAL PRIMARY KEY,
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  details TEXT NOT NULL,
  prev_hash TEXT NOT NULL,
  entry_hash TEXT NOT NULL,
  created_at TIMESTAMPTZ DEFAULT now()
);

CREATE OR REPLACE FUNCTION block_audit_mutation() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'audit_log is append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_audit_no_update ON audit_log;
DROP TRIGGER IF EXISTS trg_audit_no_delete ON audit_log;
CREATE TRIGGER trg_audit_no_update BEFORE UPDATE ON audit_log FOR EACH ROW EXECUTE FUNCTION block_audit_mutation();
CREATE TRIGGER trg_audit_no_delete BEFORE DELETE ON audit_log FOR EACH ROW EXECUTE FUNCTION block_audit_mutation();

