package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/auth/domain"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *userRepo {
	return &userRepo{db: db}
}

func (r *userRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, picture, storage_quota_bytes, storage_used_bytes, created_at, updated_at
		 FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.StorageQuotaBytes, &u.StorageUsedBytes, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, name, picture, storage_quota_bytes, storage_used_bytes, created_at, updated_at
		 FROM users WHERE id = $1`, id,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.StorageQuotaBytes, &u.StorageUsedBytes, &u.CreatedAt, &u.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) ListUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.User, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	args := []any{}
	conds := make([]string, 0, 2)
	argPos := 1

	if strings.TrimSpace(search) != "" {
		conds = append(conds, fmt.Sprintf("(email ILIKE $%d OR name ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+strings.TrimSpace(search)+"%")
		argPos++
	}
	if cursor != "" {
		conds = append(conds, fmt.Sprintf("id > $%d", argPos))
		args = append(args, cursor)
		argPos++
	}

	query := `SELECT id, email, name, picture, storage_quota_bytes, storage_used_bytes, created_at, updated_at FROM users`
	if len(conds) > 0 {
		query += " WHERE " + strings.Join(conds, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", argPos)
	args = append(args, limit+1)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	users, err := scanUserRows(rows)
	if err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(users) > limit {
		nextCursor = users[limit].ID
		users = users[:limit]
	}
	return users, nextCursor, nil
}

func (r *userRepo) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, name, picture, storage_quota_bytes, storage_used_bytes, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		user.ID, user.Email, user.Name, user.Picture, user.StorageQuotaBytes, user.StorageUsedBytes, user.CreatedAt, user.UpdatedAt,
	)
	return err
}

func (r *userRepo) Update(ctx context.Context, user *domain.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name = $2, picture = $3, updated_at = $4 WHERE id = $1`,
		user.ID, user.Name, user.Picture, user.UpdatedAt,
	)
	return err
}

func (r *userRepo) UpdateStorageQuota(ctx context.Context, userID string, quotaBytes int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET storage_quota_bytes = $2, updated_at = now() WHERE id = $1`,
		userID, quotaBytes,
	)
	return err
}

func (r *userRepo) UpdateStorageUsed(ctx context.Context, userID string, usedBytes int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET storage_used_bytes = $2, updated_at = now() WHERE id = $1`,
		userID, usedBytes,
	)
	return err
}

func scanUserRows(rows pgx.Rows) ([]*domain.User, error) {
	users := make([]*domain.User, 0)
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Picture, &u.StorageQuotaBytes, &u.StorageUsedBytes, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return users, nil
}
