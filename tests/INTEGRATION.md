Integration flow validation: tick->candle->feature->score->alert->audit can be exercised with make demo and querying query/alerts + gateway /audit/verify.
Idempotency: replaying same tick event_id remains deduplicated by DB primary key/unique constraints.
Determinism: feature_hash derives solely from payload JSON.
Audit tamper: update/delete on audit_log blocked by trigger and verify endpoint reports first break.
