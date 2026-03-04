-- +goose Up
ALTER TABLE calendars
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'custom',
    ADD COLUMN is_readonly BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_special BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN synced_start TIMESTAMPTZ,
    ADD COLUMN synced_end TIMESTAMPTZ;

CREATE TABLE calendar_backfill_state (
    user_id TEXT NOT NULL,
    calendar_id TEXT NOT NULL,
    covered_start TIMESTAMPTZ NOT NULL,
    covered_end TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, calendar_id)
);

CREATE INDEX idx_calendar_backfill_state_user ON calendar_backfill_state (user_id);
CREATE INDEX idx_calendar_backfill_state_calendar ON calendar_backfill_state (calendar_id);

-- +goose Down
DROP INDEX IF EXISTS idx_calendar_backfill_state_calendar;
DROP INDEX IF EXISTS idx_calendar_backfill_state_user;
DROP TABLE IF EXISTS calendar_backfill_state;

ALTER TABLE calendars
    DROP COLUMN IF EXISTS synced_end,
    DROP COLUMN IF EXISTS synced_start,
    DROP COLUMN IF EXISTS is_special,
    DROP COLUMN IF EXISTS is_readonly,
    DROP COLUMN IF EXISTS kind;
