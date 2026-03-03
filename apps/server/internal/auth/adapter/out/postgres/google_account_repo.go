package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/auth/domain"
)

type googleAccountRepo struct {
	db *pgxpool.Pool
}

func NewGoogleAccountRepo(db *pgxpool.Pool) *googleAccountRepo {
	return &googleAccountRepo{db: db}
}

func (r *googleAccountRepo) FindByGoogleID(ctx context.Context, googleID string) (*domain.GoogleAccount, error) {
	var a domain.GoogleAccount
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at,
		        scopes, status, is_primary, connected_at, created_at, updated_at
		 FROM user_google_accounts WHERE google_id = $1`, googleID,
	).Scan(&a.ID, &a.UserID, &a.GoogleEmail, &a.GoogleID, &a.AccessToken, &a.RefreshToken,
		&a.TokenExpiresAt, &a.Scopes, &a.Status, &a.IsPrimary, &a.ConnectedAt, &a.CreatedAt, &a.UpdatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *googleAccountRepo) FindByID(ctx context.Context, userID, id string) (*domain.GoogleAccount, error) {
	var a domain.GoogleAccount
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at,
		        scopes, status, is_primary, connected_at, created_at, updated_at
		 FROM user_google_accounts WHERE user_id = $1 AND id = $2`,
		userID, id,
	).Scan(&a.ID, &a.UserID, &a.GoogleEmail, &a.GoogleID, &a.AccessToken, &a.RefreshToken,
		&a.TokenExpiresAt, &a.Scopes, &a.Status, &a.IsPrimary, &a.ConnectedAt, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *googleAccountRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.GoogleAccount, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at,
		        scopes, status, is_primary, connected_at, created_at, updated_at
		 FROM user_google_accounts WHERE user_id = $1 ORDER BY is_primary DESC, connected_at ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*domain.GoogleAccount
	for rows.Next() {
		var a domain.GoogleAccount
		if err := rows.Scan(&a.ID, &a.UserID, &a.GoogleEmail, &a.GoogleID, &a.AccessToken, &a.RefreshToken,
			&a.TokenExpiresAt, &a.Scopes, &a.Status, &a.IsPrimary, &a.ConnectedAt, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, &a)
	}
	return accounts, nil
}

func (r *googleAccountRepo) Create(ctx context.Context, account *domain.GoogleAccount) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO user_google_accounts (id, user_id, google_email, google_id, access_token, refresh_token,
		 token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		account.ID, account.UserID, account.GoogleEmail, account.GoogleID, account.AccessToken,
		account.RefreshToken, account.TokenExpiresAt, account.Scopes, account.Status,
		account.IsPrimary, account.ConnectedAt, account.CreatedAt, account.UpdatedAt,
	)
	return err
}

func (r *googleAccountRepo) Update(ctx context.Context, account *domain.GoogleAccount) error {
	_, err := r.db.Exec(ctx,
		`UPDATE user_google_accounts
		 SET access_token = $2, refresh_token = $3, token_expires_at = $4, status = $5, updated_at = $6
		 WHERE id = $1`,
		account.ID, account.AccessToken, account.RefreshToken, account.TokenExpiresAt, account.Status, account.UpdatedAt,
	)
	return err
}
