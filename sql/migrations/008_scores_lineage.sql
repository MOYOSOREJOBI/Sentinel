-- +goose Up
ALTER TABLE scores
  ADD COLUMN IF NOT EXISTS scoring_run_id UUID NOT NULL DEFAULT gen_random_uuid(),
  ADD COLUMN IF NOT EXISTS produced_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  ADD COLUMN IF NOT EXISTS artifact_hash TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS model_artifact_hash TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_scores_scoring_run_id ON scores (scoring_run_id);
CREATE INDEX IF NOT EXISTS idx_scores_produced_at_desc ON scores (produced_at DESC);
CREATE INDEX IF NOT EXISTS idx_scores_model_version ON scores (model_version);
CREATE INDEX IF NOT EXISTS idx_scores_artifact_hash ON scores (artifact_hash);

-- Timescale hypertables cannot add a UNIQUE/PRIMARY KEY on id alone because the
-- partitioning column (ts) must be included. Enforce the intended score_id
-- reference semantics with triggers instead of a native FK.
CREATE OR REPLACE FUNCTION ensure_alert_score_reference()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF NEW.score_id IS NULL THEN
    RETURN NEW;
  END IF;
  PERFORM 1 FROM scores WHERE id = NEW.score_id LIMIT 1;
  IF NOT FOUND THEN
    RAISE EXCEPTION 'alerts.score_id % does not reference an existing score', NEW.score_id
      USING ERRCODE = '23503';
  END IF;
  RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS trg_alerts_score_reference ON alerts;
CREATE TRIGGER trg_alerts_score_reference
BEFORE INSERT OR UPDATE OF score_id ON alerts
FOR EACH ROW
EXECUTE FUNCTION ensure_alert_score_reference();

CREATE OR REPLACE FUNCTION restrict_score_delete_if_alerted()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
  IF EXISTS (SELECT 1 FROM alerts WHERE score_id = OLD.id) THEN
    RAISE EXCEPTION 'score % is still referenced by alerts', OLD.id
      USING ERRCODE = '23503';
  END IF;
  RETURN OLD;
END
$$;

DROP TRIGGER IF EXISTS trg_scores_restrict_delete ON scores;
CREATE TRIGGER trg_scores_restrict_delete
BEFORE DELETE ON scores
FOR EACH ROW
EXECUTE FUNCTION restrict_score_delete_if_alerted();

-- +goose Down
DROP TRIGGER IF EXISTS trg_scores_restrict_delete ON scores;
DROP FUNCTION IF EXISTS restrict_score_delete_if_alerted();

DROP TRIGGER IF EXISTS trg_alerts_score_reference ON alerts;
DROP FUNCTION IF EXISTS ensure_alert_score_reference();

DROP INDEX IF EXISTS idx_scores_artifact_hash;
DROP INDEX IF EXISTS idx_scores_model_version;
DROP INDEX IF EXISTS idx_scores_produced_at_desc;
DROP INDEX IF EXISTS idx_scores_scoring_run_id;

ALTER TABLE scores
  DROP COLUMN IF EXISTS model_artifact_hash,
  DROP COLUMN IF EXISTS artifact_hash,
  DROP COLUMN IF EXISTS produced_at,
  DROP COLUMN IF EXISTS scoring_run_id;
