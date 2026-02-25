package postgres

import (
	"context"
	"errors"

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
