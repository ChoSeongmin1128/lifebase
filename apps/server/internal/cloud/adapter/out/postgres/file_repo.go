package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/cloud/domain"
)

type fileRepo struct {
	db *pgxpool.Pool
}

func NewFileRepo(db *pgxpool.Pool) *fileRepo {
	return &fileRepo{db: db}
}

func (r *fileRepo) Create(ctx context.Context, file *domain.File) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		file.ID, file.UserID, file.FolderID, file.Name, file.MimeType, file.SizeBytes,
		file.StoragePath, file.ThumbStatus, file.TakenAt, file.CreatedAt, file.UpdatedAt,
	)
	return err
}

func (r *fileRepo) FindByID(ctx context.Context, userID, id string) (*domain.File, error) {
	var f domain.File
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
		 FROM files WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID,
	).Scan(&f.ID, &f.UserID, &f.FolderID, &f.Name, &f.MimeType, &f.SizeBytes,
		&f.StoragePath, &f.ThumbStatus, &f.TakenAt, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("file not found")
	}
	return &f, err
}

func (r *fileRepo) ListByFolder(ctx context.Context, userID string, folderID *string, sortBy, sortDir string) ([]*domain.File, error) {
	orderClause := buildOrderClause(sortBy, sortDir)

	var rows pgx.Rows
	var err error

	if folderID == nil {
		rows, err = r.db.Query(ctx, fmt.Sprintf(
			`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
			 FROM files WHERE user_id = $1 AND folder_id IS NULL AND deleted_at IS NULL
			 ORDER BY %s`, orderClause), userID)
	} else {
		rows, err = r.db.Query(ctx, fmt.Sprintf(
			`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
			 FROM files WHERE user_id = $1 AND folder_id = $2 AND deleted_at IS NULL
			 ORDER BY %s`, orderClause), userID, *folderID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

func (r *fileRepo) Update(ctx context.Context, file *domain.File) error {
	_, err := r.db.Exec(ctx,
		`UPDATE files SET folder_id = $2, name = $3, updated_at = $4 WHERE id = $1`,
		file.ID, file.FolderID, file.Name, file.UpdatedAt,
	)
	return err
}

func (r *fileRepo) SoftDelete(ctx context.Context, userID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE files SET deleted_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID, time.Now(),
	)
	return err
}

func (r *fileRepo) Restore(ctx context.Context, userID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE files SET deleted_at = NULL, updated_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`,
		id, userID, time.Now(),
	)
	return err
}

func (r *fileRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM files WHERE id = $1`, id)
	return err
}

func (r *fileRepo) ListTrashed(ctx context.Context, userID string) ([]*domain.File, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
		 FROM files WHERE user_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

func (r *fileRepo) UpdateStorageUsed(ctx context.Context, userID string, delta int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET storage_used_bytes = storage_used_bytes + $2, updated_at = $3 WHERE id = $1`,
		userID, delta, time.Now(),
	)
	return err
}

func (r *fileRepo) Search(ctx context.Context, userID, query string, limit int) ([]*domain.File, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, taken_at, created_at, updated_at, deleted_at
		 FROM files WHERE user_id = $1 AND deleted_at IS NULL AND name % $2
		 ORDER BY similarity(name, $2) DESC
		 LIMIT $3`, userID, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFiles(rows)
}

func scanFiles(rows pgx.Rows) ([]*domain.File, error) {
	var files []*domain.File
	for rows.Next() {
		var f domain.File
		if err := rows.Scan(&f.ID, &f.UserID, &f.FolderID, &f.Name, &f.MimeType, &f.SizeBytes,
			&f.StoragePath, &f.ThumbStatus, &f.TakenAt, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

func buildOrderClause(sortBy, sortDir string) string {
	column := "name"
	switch sortBy {
	case "name":
		column = "name"
	case "size":
		column = "size_bytes"
	case "updated_at":
		column = "updated_at"
	case "created_at":
		column = "created_at"
	}

	dir := "ASC"
	if sortDir == "desc" {
		dir = "DESC"
	}

	return fmt.Sprintf("%s %s, name ASC", column, dir)
}
