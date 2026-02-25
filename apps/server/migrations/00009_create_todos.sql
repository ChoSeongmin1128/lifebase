-- +goose Up
CREATE TABLE todo_lists (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    google_id   TEXT,
    name        TEXT NOT NULL,
    sort_order  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_todo_lists_user_id ON todo_lists (user_id);

CREATE TABLE todos (
    id              TEXT PRIMARY KEY,
    list_id         TEXT NOT NULL,
    user_id         TEXT NOT NULL,
    parent_id       TEXT,
    google_id       TEXT,
    title           TEXT NOT NULL DEFAULT '',
    notes           TEXT NOT NULL DEFAULT '',
    due             DATE,
    priority        TEXT NOT NULL DEFAULT 'normal',  -- urgent, high, normal, low
    is_done         BOOLEAN NOT NULL DEFAULT FALSE,
    is_pinned       BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order      INT NOT NULL DEFAULT 0,
    done_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_todos_list_id ON todos (list_id);
CREATE INDEX idx_todos_user_id ON todos (user_id);
CREATE INDEX idx_todos_parent_id ON todos (parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_todos_due ON todos (user_id, due) WHERE due IS NOT NULL AND deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS todos;
DROP TABLE IF EXISTS todo_lists;
