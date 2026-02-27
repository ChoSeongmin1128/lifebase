package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/cloud/domain"
)

type sharedRepo struct {
	db *pgxpool.Pool
}

func NewSharedRepo(db *pgxpool.Pool) *sharedRepo {
	return &sharedRepo{db: db}
}

func (r *sharedRepo) ListSharedFolders(ctx context.Context, userID string) ([]*domain.Folder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT ON (f.id)
		 f.id, f.user_id, f.parent_id, f.name, f.created_at, f.updated_at, f.deleted_at
		 FROM shares s
		 JOIN folders f ON f.id = s.folder_id
		 WHERE s.shared_with = $1 AND f.deleted_at IS NULL
		 ORDER BY f.id, s.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}
