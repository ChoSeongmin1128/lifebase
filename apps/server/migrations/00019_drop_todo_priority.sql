-- +goose Up
ALTER TABLE todos DROP COLUMN IF EXISTS priority;

-- +goose Down
ALTER TABLE todos
ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'normal';
