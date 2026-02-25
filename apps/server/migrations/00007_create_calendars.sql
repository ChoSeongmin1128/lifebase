-- +goose Up
CREATE TABLE calendars (
    id              TEXT PRIMARY KEY,
    user_id         TEXT NOT NULL,
    google_id       TEXT,
    name            TEXT NOT NULL,
    color_id        TEXT,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    is_visible      BOOLEAN NOT NULL DEFAULT TRUE,
    sync_token      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_calendars_user_id ON calendars (user_id);
CREATE UNIQUE INDEX idx_calendars_google_id ON calendars (user_id, google_id) WHERE google_id IS NOT NULL;

-- +goose Down
DROP TABLE IF EXISTS calendars;
