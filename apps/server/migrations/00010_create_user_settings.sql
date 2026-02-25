-- +goose Up
CREATE TABLE user_settings (
    user_id     TEXT NOT NULL,
    key         TEXT NOT NULL,
    value       TEXT NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, key)
);

CREATE INDEX idx_user_settings_user_id ON user_settings (user_id);

-- +goose Down
DROP TABLE IF EXISTS user_settings;
