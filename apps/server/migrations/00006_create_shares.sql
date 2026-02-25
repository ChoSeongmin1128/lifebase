-- +goose Up
CREATE TABLE shares (
    id          TEXT PRIMARY KEY,
    folder_id   TEXT NOT NULL,
    owner_id    TEXT NOT NULL,
    shared_with TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'viewer',  -- viewer, editor
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shares_folder_id ON shares (folder_id);
CREATE INDEX idx_shares_owner_id ON shares (owner_id);
CREATE INDEX idx_shares_shared_with ON shares (shared_with);

CREATE TABLE share_invites (
    id          TEXT PRIMARY KEY,
    folder_id   TEXT NOT NULL,
    owner_id    TEXT NOT NULL,
    token       TEXT NOT NULL UNIQUE,
    role        TEXT NOT NULL DEFAULT 'viewer',
    expires_at  TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_share_invites_token ON share_invites (token);
CREATE INDEX idx_share_invites_folder_id ON share_invites (folder_id);

-- +goose Down
DROP TABLE IF EXISTS share_invites;
DROP TABLE IF EXISTS shares;
