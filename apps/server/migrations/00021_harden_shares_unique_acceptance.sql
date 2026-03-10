-- +goose Up
CREATE UNIQUE INDEX idx_shares_folder_shared_with_unique ON shares (folder_id, shared_with);

-- +goose Down
DROP INDEX IF EXISTS idx_shares_folder_shared_with_unique;
