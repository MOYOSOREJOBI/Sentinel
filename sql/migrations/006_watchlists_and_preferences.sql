-- +goose Up
CREATE TABLE IF NOT EXISTS watchlists (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_user TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS watchlist_items (
  watchlist_id UUID NOT NULL REFERENCES watchlists(id) ON DELETE CASCADE,
  instrument_id TEXT NOT NULL,
  PRIMARY KEY (watchlist_id, instrument_id)
);

CREATE TABLE IF NOT EXISTS user_preferences (
  owner_user TEXT PRIMARY KEY,
  large_text BOOLEAN NOT NULL DEFAULT FALSE,
  preferred_mode TEXT NOT NULL DEFAULT 'viewer',
  preferred_region TEXT,
  preferred_industry TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE instrument_metadata
  ADD COLUMN IF NOT EXISTS country_code TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS country_name TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS timezone TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE instrument_metadata
  DROP COLUMN IF EXISTS timezone,
  DROP COLUMN IF EXISTS country_name,
  DROP COLUMN IF EXISTS country_code;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS watchlist_items;
DROP TABLE IF EXISTS watchlists;
