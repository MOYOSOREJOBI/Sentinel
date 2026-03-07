-- +goose Up
ALTER TABLE user_preferences
  ADD COLUMN IF NOT EXISTS preferred_locale TEXT NOT NULL DEFAULT 'en';

-- +goose Down
ALTER TABLE user_preferences
  DROP COLUMN IF EXISTS preferred_locale;
