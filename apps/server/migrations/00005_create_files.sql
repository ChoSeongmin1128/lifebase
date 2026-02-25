-- +goose Up
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    folder_id UUID,              -- NULL = 루트
    name TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT 'application/octet-stream',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_path TEXT NOT NULL,   -- 물리 파일 경로 (UUID 기반)
    thumb_status TEXT NOT NULL DEFAULT 'pending', -- pending, processing, done, failed
    taken_at TIMESTAMPTZ,         -- EXIF 촬영 날짜
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ        -- soft delete (휴지통)
);

CREATE INDEX idx_files_user_id ON files (user_id);
CREATE INDEX idx_files_folder_id ON files (folder_id);
CREATE INDEX idx_files_user_folder ON files (user_id, folder_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_deleted_at ON files (deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_files_mime_type ON files (user_id, mime_type) WHERE deleted_at IS NULL;
CREATE INDEX idx_files_name_trgm ON files USING gin (name gin_trgm_ops);

-- +goose Down
DROP TABLE IF EXISTS files;
