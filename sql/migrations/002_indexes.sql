-- +goose Up
CREATE INDEX IF NOT EXISTS idx_alerts_symbol_created_at ON alerts(symbol, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_alerts_status_created_at ON alerts(status, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scores_symbol_ts ON scores(symbol, ts DESC);
CREATE INDEX IF NOT EXISTS idx_features_symbol_ts ON features(symbol, ts DESC);
CREATE INDEX IF NOT EXISTS idx_candles_symbol_bucket_interval ON candles(symbol, bucket DESC, interval);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_log(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_alerts_symbol_created_at;
DROP INDEX IF EXISTS idx_alerts_status_created_at;
DROP INDEX IF EXISTS idx_scores_symbol_ts;
DROP INDEX IF EXISTS idx_features_symbol_ts;
DROP INDEX IF EXISTS idx_candles_symbol_bucket_interval;
DROP INDEX IF EXISTS idx_audit_created_at;
