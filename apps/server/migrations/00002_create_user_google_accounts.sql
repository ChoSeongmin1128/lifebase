-- +goose Up
CREATE TABLE user_google_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    google_email TEXT NOT NULL,
    google_id TEXT NOT NULL,
    access_token TEXT NOT NULL DEFAULT '',
    refresh_token TEXT NOT NULL DEFAULT '',
    token_expires_at TIMESTAMPTZ,
    scopes TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active', -- active, reauth_required, revoked
    is_primary BOOLEAN NOT NULL DEFAULT false,
    connected_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_google_accounts_user_id ON user_google_accounts (user_id);
CREATE UNIQUE INDEX idx_user_google_accounts_google_id ON user_google_accounts (google_id);

-- +goose Down
DROP TABLE IF EXISTS user_google_accounts;
