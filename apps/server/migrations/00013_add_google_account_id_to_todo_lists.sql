-- +goose Up
ALTER TABLE todo_lists
    ADD COLUMN google_account_id TEXT;

CREATE INDEX idx_todo_lists_google_account_id ON todo_lists (google_account_id);
CREATE INDEX idx_todo_lists_user_account ON todo_lists (user_id, google_account_id);
CREATE UNIQUE INDEX idx_todo_lists_user_google_id ON todo_lists (user_id, google_id) WHERE google_id IS NOT NULL;
CREATE UNIQUE INDEX idx_todos_user_google_id ON todos (user_id, google_id) WHERE google_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_todos_user_google_id;
DROP INDEX IF EXISTS idx_todo_lists_user_google_id;
DROP INDEX IF EXISTS idx_todo_lists_user_account;
DROP INDEX IF EXISTS idx_todo_lists_google_account_id;

ALTER TABLE todo_lists
    DROP COLUMN IF EXISTS google_account_id;
