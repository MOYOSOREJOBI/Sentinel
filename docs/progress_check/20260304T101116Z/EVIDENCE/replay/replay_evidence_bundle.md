Replay + Governance evidence bundle (2026-03-04 UTC).

Primary replay jobs:

- One-second replay A: `21002bc2-0e83-4e26-8147-e78a74c166a9`
- One-second replay B: `4647e7ec-2a60-4448-93dd-879047a22cba` (same window + inputs as A)

Key proofs:

- Replay B determinism: `determinism_status = MATCH`
  - Source: `replay_one_second_b.json`
- Replay diff summary endpoint includes:
  - `matched_count`
  - `mismatched_count`
  - `max_score_delta`
  - `common_mismatch_causes`
  - Source: `diff_summary_one_second_b.txt`
- Replay brief export contains provenance:
  - window
  - modelVersion
  - replayId
  - parityStatus
  - pinnedEvidence
  - timestamps
  - Source: `brief_export_one_second_b.json`

Governance + Trust lineage proofs:

- Deployment action created a deployment record:
  - Source: `model_deploy.txt`
- Governance summary now returns model deployment history and replay jobs:
  - Source: `governance_summary_after_fix.txt`
- Trust snapshot returns active model lineage fields:
  - `active_model_version`
  - `last_trained_at`
  - `active_artifact_hash`
  - Source: `trust_snapshot.txt`
