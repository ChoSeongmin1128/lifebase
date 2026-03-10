-- +goose Up
ALTER TABLE users
    ALTER COLUMN storage_quota_bytes SET DEFAULT 16106127360; -- 15GB

-- +goose Down
ALTER TABLE users
    ALTER COLUMN storage_quota_bytes SET DEFAULT 1099511627776; -- 1TB
