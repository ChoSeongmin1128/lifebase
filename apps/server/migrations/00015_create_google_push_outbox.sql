-- +goose Up
CREATE TABLE google_push_outbox (
    id                  TEXT PRIMARY KEY,
    account_id          TEXT NOT NULL,
    user_id             TEXT NOT NULL,
    domain              TEXT NOT NULL, -- calendar | todo
    op                  TEXT NOT NULL, -- create | update | delete
    local_resource_id   TEXT NOT NULL,
    expected_updated_at TIMESTAMPTZ NOT NULL,
    payload_json        JSONB,
    status              TEXT NOT NULL DEFAULT 'pending', -- pending | processing | done | retry | dead
    attempt_count       INT NOT NULL DEFAULT 0,
    next_retry_at       TIMESTAMPTZ,
    last_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_google_push_outbox_status_retry
    ON google_push_outbox (status, next_retry_at, created_at);

CREATE INDEX idx_google_push_outbox_account_status
    ON google_push_outbox (account_id, status, created_at);

CREATE UNIQUE INDEX idx_google_push_outbox_dedup
    ON google_push_outbox (domain, op, local_resource_id, expected_updated_at);

-- +goose Down
DROP TABLE IF EXISTS google_push_outbox;
