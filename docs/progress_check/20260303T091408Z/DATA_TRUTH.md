# Data Truth

## Runtime DB state

Current runtime database truth is `UNKNOWN` because the fresh compose boot never created the `postgres` container.

- Fresh boot failed during image build because every Go service image copies `keys/`, but the repo root did not contain that directory at boot time.
- Evidence:
  - `deploy/docker/Dockerfile-go:9-14`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_up.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/docker/compose_ps.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/tables.txt`

## Static schema truth

The schema required for the pipeline is present in migrations even though runtime rows could not be queried in this run.

- Base tables exist in migration code for `users`, `raw_ticks`, `candles`, `features`, `scores`, `alerts`, and `audit_log`.
  - Evidence: `sql/migrations/001_init.sql:14-130`
- Incident and case lineage tables exist in migration code for `incidents`, `cases`, `case_notes`, `case_actions`, `case_evidence`, and `incident_alert_links`.
  - Evidence: `sql/migrations/003_incidents_cases_lineage.sql:18-105`
- Score lineage columns exist in migration code for `scoring_run_id`, `produced_at`, `artifact_hash`, and `model_artifact_hash`, plus alert-reference triggers.
  - Evidence: `sql/migrations/008_scores_lineage.sql:2-56`

## What could not be proven

The following remained unprovable in this audit run because `postgres` never came up:

- Table list from a live database
- Constraints and indexes from a live database
- Row counts for `raw_ticks`, `candles`, `features`, `scores`, `alerts`, `incidents`, and `cases`
- Live `alerts.score_id -> scores.id` join integrity
- Live non-null lineage values in persisted `scores`

Each attempted runtime query failed with `service "postgres" is not running`.

- Evidence:
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/tables.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/constraints.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/indexes.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/counts.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/alerts_scores_join.txt`
  - `docs/progress_check/20260303T091408Z/EVIDENCE/sql/lineage_columns.txt`

## Practical conclusion

Statically, the repo contains the schema needed for persisted scores, alerts, incidents, and cases.

Operationally, this run cannot prove any live data exists because the required database container never started.
