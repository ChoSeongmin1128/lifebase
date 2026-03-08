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

type folderRepo struct {
	db *pgxpool.Pool
}

func NewFolderRepo(db *pgxpool.Pool) *folderRepo {
	return &folderRepo{db: db}
}

func (r *folderRepo) Create(ctx context.Context, folder *domain.Folder) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO folders (id, user_id, parent_id, name, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		folder.ID, folder.UserID, folder.ParentID, folder.Name, folder.CreatedAt, folder.UpdatedAt,
	)
	return err
}

func (r *folderRepo) FindByID(ctx context.Context, userID, id string) (*domain.Folder, error) {
	var f domain.Folder
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, parent_id, name, created_at, updated_at, deleted_at
		 FROM folders WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID,
	).Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("folder not found")
	}
	return &f, err
}

func (r *folderRepo) FindTrashedByID(ctx context.Context, userID, id string) (*domain.Folder, error) {
	var f domain.Folder
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, parent_id, name, created_at, updated_at, deleted_at
		 FROM folders WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`, id, userID,
	).Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("folder not found")
	}
	return &f, err
}

func (r *folderRepo) ListByParent(ctx context.Context, userID string, parentID *string) ([]*domain.Folder, error) {
	var rows pgx.Rows
	var err error

	if parentID == nil {
		rows, err = r.db.Query(ctx,
			`SELECT id, user_id, parent_id, name, created_at, updated_at, deleted_at
			 FROM folders WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
			 ORDER BY name ASC`, userID)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, user_id, parent_id, name, created_at, updated_at, deleted_at
			 FROM folders WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL
			 ORDER BY name ASC`, userID, *parentID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

func (r *folderRepo) Update(ctx context.Context, folder *domain.Folder) error {
	_, err := r.db.Exec(ctx,
		`UPDATE folders SET parent_id = $2, name = $3, updated_at = $4 WHERE id = $1`,
		folder.ID, folder.ParentID, folder.Name, folder.UpdatedAt,
	)
	return err
}

func (r *folderRepo) SoftDelete(ctx context.Context, userID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE folders SET deleted_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID, time.Now(),
	)
	return err
}

func (r *folderRepo) Restore(ctx context.Context, userID, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE folders SET deleted_at = NULL, updated_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NOT NULL`,
		id, userID, time.Now(),
	)
	return err
}

func (r *folderRepo) HardDelete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM folders WHERE id = $1`, id)
	return err
}

func (r *folderRepo) ListTrashed(ctx context.Context, userID string) ([]*domain.Folder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, parent_id, name, created_at, updated_at, deleted_at
		 FROM folders WHERE user_id = $1 AND deleted_at IS NOT NULL
		 ORDER BY deleted_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanFolders(rows)
}

func (r *folderRepo) ExistsByName(ctx context.Context, userID string, parentID *string, name string) (bool, error) {
	var count int
	var err error
	if parentID == nil {
		err = r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM folders WHERE user_id = $1 AND parent_id IS NULL AND name = $2 AND deleted_at IS NULL`,
			userID, name,
		).Scan(&count)
	} else {
		err = r.db.QueryRow(ctx,
			`SELECT COUNT(*) FROM folders WHERE user_id = $1 AND parent_id = $2 AND name = $3 AND deleted_at IS NULL`,
			userID, *parentID, name,
		).Scan(&count)
	}
	return count > 0, err
}

func scanFolders(rows pgx.Rows) ([]*domain.Folder, error) {
	var folders []*domain.Folder
	for rows.Next() {
		var f domain.Folder
		if err := rows.Scan(&f.ID, &f.UserID, &f.ParentID, &f.Name, &f.CreatedAt, &f.UpdatedAt, &f.DeletedAt); err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}
