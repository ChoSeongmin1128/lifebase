package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/gallery/domain"
)

type mediaRepo struct {
	db *pgxpool.Pool
}

var scanMediaFilesFn = scanFiles

func NewMediaRepo(db *pgxpool.Pool) *mediaRepo {
	return &mediaRepo{db: db}
}

func (r *mediaRepo) ListMedia(ctx context.Context, userID string, mimePrefix string, sortBy string, sortDir string, cursor string, limit int) ([]*domain.Media, error) {
	column := "created_at"
	switch sortBy {
	case "taken_at":
		column = "COALESCE(taken_at, created_at)"
	case "name":
		column = "name"
	case "size":
		column = "size_bytes"
	case "created_at":
		column = "created_at"
	}

	dir := "DESC"
	if sortDir == "asc" {
		dir = "ASC"
	}

	orderClause := fmt.Sprintf("%s %s, id ASC", column, dir)

	var mimeFilter string
	args := []any{userID, limit}
	argIdx := 3

	switch mimePrefix {
	case "image":
		mimeFilter = " AND mime_type LIKE 'image/%'"
	case "video":
		mimeFilter = " AND mime_type LIKE 'video/%'"
	default:
		mimeFilter = " AND (mime_type LIKE 'image/%' OR mime_type LIKE 'video/%')"
	}

	cursorFilter := ""
	if cursor != "" {
		cursorFilter = fmt.Sprintf(" AND id > $%d", argIdx)
		args = append(args, cursor)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
		 FROM files
		 WHERE user_id = $1 AND deleted_at IS NULL%s%s
		 ORDER BY %s
		 LIMIT $2`,
		mimeFilter, cursorFilter, orderClause,
	)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMediaFilesFn(rows)
}

func scanFiles(rows pgx.Rows) ([]*domain.Media, error) {
	var files []*domain.Media
	for rows.Next() {
		var f domain.Media
		if err := rows.Scan(&f.ID, &f.UserID, &f.FolderID, &f.Name, &f.MimeType, &f.SizeBytes,
			&f.StoragePath, &f.ThumbStatus, &f.TakenAt, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return files, nil
}
