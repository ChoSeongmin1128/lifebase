-- +goose Up
CREATE TABLE events (
    id              TEXT PRIMARY KEY,
    calendar_id     TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    google_id       TEXT,
    title           TEXT NOT NULL DEFAULT '',
    description     TEXT NOT NULL DEFAULT '',
    location        TEXT NOT NULL DEFAULT '',
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ NOT NULL,
    timezone        TEXT NOT NULL DEFAULT 'Asia/Seoul',
    is_all_day      BOOLEAN NOT NULL DEFAULT FALSE,
    color_id        TEXT,
    recurrence_rule TEXT,
    etag            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_events_user_id ON events (user_id);
CREATE INDEX idx_events_calendar_id ON events (calendar_id);
CREATE INDEX idx_events_time_range ON events (user_id, start_time, end_time) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_events_google_id ON events (user_id, google_id) WHERE google_id IS NOT NULL;

CREATE TABLE event_reminders (
    id          TEXT PRIMARY KEY,
    event_id    TEXT NOT NULL,
    method      TEXT NOT NULL DEFAULT 'popup',  -- popup, email
    minutes     INT NOT NULL DEFAULT 10,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_reminders_event_id ON event_reminders (event_id);

CREATE TABLE event_exceptions (
    id                  TEXT PRIMARY KEY,
    recurring_event_id  TEXT NOT NULL,
    original_start      TIMESTAMPTZ NOT NULL,
    is_cancelled        BOOLEAN NOT NULL DEFAULT FALSE,
    title               TEXT,
    description         TEXT,
    location            TEXT,
    start_time          TIMESTAMPTZ,
    end_time            TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_event_exceptions_recurring ON event_exceptions (recurring_event_id);

-- +goose Down
DROP TABLE IF EXISTS event_exceptions;
DROP TABLE IF EXISTS event_reminders;
DROP TABLE IF EXISTS events;
