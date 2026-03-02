package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	adminout "lifebase/internal/admin/port/out"
)

type StorageResetRepo struct {
	db *pgxpool.Pool
}

func NewStorageResetRepo(db *pgxpool.Pool) *StorageResetRepo {
	return &StorageResetRepo{db: db}
}

func (r *StorageResetRepo) ListFilesByUser(ctx context.Context, userID string) ([]adminout.FileRef, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, storage_path FROM files WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	files := make([]adminout.FileRef, 0)
	for rows.Next() {
		var file adminout.FileRef
		if err := rows.Scan(&file.ID, &file.StoragePath); err != nil {
			return nil, err
		}
		files = append(files, file)
	}
	return files, nil
}

func (r *StorageResetRepo) DeleteAllFilesByUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM files WHERE user_id = $1`, userID)
	return err
}

func (r *StorageResetRepo) DeleteAllFoldersByUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM folders WHERE user_id = $1`, userID)
	return err
}

func (r *StorageResetRepo) DeleteAllStarsByUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM user_settings WHERE user_id = $1 AND key LIKE 'cloud_star:%'`,
		userID,
	)
	return err
}

func (r *StorageResetRepo) DeleteSharesByOwner(ctx context.Context, ownerID string) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM shares
		 WHERE owner_id = $1
		    OR folder_id IN (SELECT id::text FROM folders WHERE user_id = $1)`,
		ownerID,
	)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(ctx,
		`DELETE FROM share_invites
		 WHERE owner_id = $1
		    OR folder_id IN (SELECT id::text FROM folders WHERE user_id = $1)`,
		ownerID,
	)
	return err
}

func (r *StorageResetRepo) SumStorageUsed(ctx context.Context, userID string) (int64, error) {
	var used int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(size_bytes), 0) FROM files WHERE user_id = $1 AND deleted_at IS NULL`,
		userID,
	).Scan(&used)
	return used, err
}
