-- +goose Up
CREATE TABLE google_sync_state (
    account_id               TEXT PRIMARY KEY,
    user_id                  TEXT NOT NULL,
    last_hourly_sync_at      TIMESTAMPTZ,
    last_tab_sync_at         TIMESTAMPTZ,
    last_nav_sync_at         TIMESTAMPTZ,
    last_action_sync_at      TIMESTAMPTZ,
    last_success_at          TIMESTAMPTZ,
    last_error               TEXT,
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_google_sync_state_user_id ON google_sync_state (user_id);
CREATE INDEX idx_google_sync_state_updated_at ON google_sync_state (updated_at);

-- +goose Down
DROP TABLE IF EXISTS google_sync_state;
