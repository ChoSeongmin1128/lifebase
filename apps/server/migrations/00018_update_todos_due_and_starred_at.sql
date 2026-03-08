-- +goose Up
ALTER TABLE todos
    RENAME COLUMN due TO due_date;

ALTER TABLE todos
    ADD COLUMN due_time TIME,
    ADD COLUMN starred_at TIMESTAMPTZ;

ALTER TABLE todos
    ADD CONSTRAINT todos_due_time_requires_due_date
    CHECK (due_time IS NULL OR due_date IS NOT NULL);

DROP INDEX IF EXISTS idx_todos_due;
CREATE INDEX idx_todos_due_date ON todos (user_id, due_date) WHERE due_date IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_todos_starred_at ON todos (user_id, starred_at DESC) WHERE starred_at IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_todos_starred_at;
DROP INDEX IF EXISTS idx_todos_due_date;

ALTER TABLE todos
    DROP CONSTRAINT IF EXISTS todos_due_time_requires_due_date;

ALTER TABLE todos
    DROP COLUMN starred_at,
    DROP COLUMN due_time;

ALTER TABLE todos
    RENAME COLUMN due_date TO due;

CREATE INDEX idx_todos_due ON todos (user_id, due) WHERE due IS NOT NULL AND deleted_at IS NULL;
