-- +goose Up
CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    role TEXT NOT NULL DEFAULT 'admin', -- admin, super_admin
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_admin_users_user_id ON admin_users (user_id);
CREATE INDEX idx_admin_users_role ON admin_users (role);
CREATE INDEX idx_admin_users_is_active ON admin_users (is_active);

-- +goose Down
DROP TABLE IF EXISTS admin_users;
