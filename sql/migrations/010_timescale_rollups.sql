-- +goose Up
DROP MATERIALIZED VIEW IF EXISTS scores_1d CASCADE;
DROP MATERIALIZED VIEW IF EXISTS scores_1h CASCADE;
DROP MATERIALIZED VIEW IF EXISTS candles_1d CASCADE;
DROP MATERIALIZED VIEW IF EXISTS candles_1h CASCADE;

CREATE MATERIALIZED VIEW candles_1h
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 hour', bucket) AS bucket,
  symbol,
  first(open, bucket) AS open,
  max(high) AS high,
  min(low) AS low,
  last(close, bucket) AS close,
  sum(volume) AS volume
FROM candles
WHERE interval = '1m'
GROUP BY 1, 2
WITH NO DATA;

CREATE MATERIALIZED VIEW candles_1d
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 day', bucket) AS bucket,
  symbol,
  first(open, bucket) AS open,
  max(high) AS high,
  min(low) AS low,
  last(close, bucket) AS close,
  sum(volume) AS volume
FROM candles
WHERE interval = '1m'
GROUP BY 1, 2
WITH NO DATA;

CREATE MATERIALIZED VIEW scores_1h
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 hour', ts) AS bucket,
  symbol,
  avg(COALESCE(normalized_anomaly_score, score, 0)) AS anomaly_score,
  avg(COALESCE(normalized_anomaly_score, score, 0)) AS score_norm,
  avg(COALESCE(raw_anomaly_score, score, 0)) AS score_raw,
  avg(COALESCE(escalation_probability, 0)) AS escalation_prob,
  avg(COALESCE(composite_risk, score, 0)) AS composite,
  avg(COALESCE(composite_risk, score, 0)) AS composite_risk,
  last(COALESCE(model_version, 'baseline-v1'), COALESCE(produced_at, ts)) AS model_version,
  max(COALESCE(produced_at, ts)) AS produced_at,
  last(COALESCE(scoring_run_id::text, ''), COALESCE(produced_at, ts)) AS scoring_run_id,
  last(COALESCE(feature_snapshot_hash, ''), COALESCE(produced_at, ts)) AS feature_snapshot_hash,
  last(COALESCE(artifact_hash, ''), COALESCE(produced_at, ts)) AS artifact_hash,
  last(COALESCE(model_artifact_hash, ''), COALESCE(produced_at, ts)) AS model_artifact_hash,
  last(COALESCE(severity, ''), COALESCE(produced_at, ts)) AS severity
FROM scores
GROUP BY 1, 2
WITH NO DATA;

CREATE MATERIALIZED VIEW scores_1d
WITH (timescaledb.continuous) AS
SELECT
  time_bucket('1 day', ts) AS bucket,
  symbol,
  avg(COALESCE(normalized_anomaly_score, score, 0)) AS anomaly_score,
  avg(COALESCE(normalized_anomaly_score, score, 0)) AS score_norm,
  avg(COALESCE(raw_anomaly_score, score, 0)) AS score_raw,
  avg(COALESCE(escalation_probability, 0)) AS escalation_prob,
  avg(COALESCE(composite_risk, score, 0)) AS composite,
  avg(COALESCE(composite_risk, score, 0)) AS composite_risk,
  last(COALESCE(model_version, 'baseline-v1'), COALESCE(produced_at, ts)) AS model_version,
  max(COALESCE(produced_at, ts)) AS produced_at,
  last(COALESCE(scoring_run_id::text, ''), COALESCE(produced_at, ts)) AS scoring_run_id,
  last(COALESCE(feature_snapshot_hash, ''), COALESCE(produced_at, ts)) AS feature_snapshot_hash,
  last(COALESCE(artifact_hash, ''), COALESCE(produced_at, ts)) AS artifact_hash,
  last(COALESCE(model_artifact_hash, ''), COALESCE(produced_at, ts)) AS model_artifact_hash,
  last(COALESCE(severity, ''), COALESCE(produced_at, ts)) AS severity
FROM scores
GROUP BY 1, 2
WITH NO DATA;

SELECT add_continuous_aggregate_policy('candles_1h',
  start_offset => INTERVAL '45 days',
  end_offset => INTERVAL '5 minutes',
  schedule_interval => INTERVAL '5 minutes');
SELECT add_continuous_aggregate_policy('candles_1d',
  start_offset => INTERVAL '6 years',
  end_offset => INTERVAL '1 hour',
  schedule_interval => INTERVAL '1 hour');
SELECT add_continuous_aggregate_policy('scores_1h',
  start_offset => INTERVAL '45 days',
  end_offset => INTERVAL '5 minutes',
  schedule_interval => INTERVAL '5 minutes');
SELECT add_continuous_aggregate_policy('scores_1d',
  start_offset => INTERVAL '6 years',
  end_offset => INTERVAL '1 hour',
  schedule_interval => INTERVAL '1 hour');

CALL refresh_continuous_aggregate('candles_1h', NULL, now());
CALL refresh_continuous_aggregate('candles_1d', NULL, now());
CALL refresh_continuous_aggregate('scores_1h', NULL, now());
CALL refresh_continuous_aggregate('scores_1d', NULL, now());

-- +goose Down
SELECT remove_continuous_aggregate_policy('scores_1d');
SELECT remove_continuous_aggregate_policy('scores_1h');
SELECT remove_continuous_aggregate_policy('candles_1d');
SELECT remove_continuous_aggregate_policy('candles_1h');

DROP MATERIALIZED VIEW IF EXISTS scores_1d;
DROP MATERIALIZED VIEW IF EXISTS scores_1h;
DROP MATERIALIZED VIEW IF EXISTS candles_1d;
DROP MATERIALIZED VIEW IF EXISTS candles_1h;
