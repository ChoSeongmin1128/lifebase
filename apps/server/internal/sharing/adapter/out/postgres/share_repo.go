package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/sharing/domain"
)

type shareRepo struct {
	db *pgxpool.Pool
}

func NewShareRepo(db *pgxpool.Pool) *shareRepo {
	return &shareRepo{db: db}
}

func (r *shareRepo) Create(ctx context.Context, share *domain.Share) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO shares (id, folder_id, owner_id, shared_with, role, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		share.ID, share.FolderID, share.OwnerID, share.SharedWith, share.Role, share.CreatedAt, share.UpdatedAt,
	)
	return err
}

func (r *shareRepo) FindByID(ctx context.Context, id string) (*domain.Share, error) {
	var s domain.Share
	err := r.db.QueryRow(ctx,
		`SELECT id, folder_id, owner_id, shared_with, role, created_at, updated_at
		 FROM shares WHERE id = $1`, id,
	).Scan(&s.ID, &s.FolderID, &s.OwnerID, &s.SharedWith, &s.Role, &s.CreatedAt, &s.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("share not found")
	}
	return &s, err
}

func (r *shareRepo) ListByFolder(ctx context.Context, folderID string) ([]*domain.Share, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, folder_id, owner_id, shared_with, role, created_at, updated_at
		 FROM shares WHERE folder_id = $1 ORDER BY created_at`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanShares(rows)
}

func (r *shareRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Share, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, folder_id, owner_id, shared_with, role, created_at, updated_at
		 FROM shares WHERE shared_with = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanShares(rows)
}

func (r *shareRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM shares WHERE id = $1`, id)
	return err
}

func scanShares(rows pgx.Rows) ([]*domain.Share, error) {
	var shares []*domain.Share
	for rows.Next() {
		var s domain.Share
		if err := rows.Scan(&s.ID, &s.FolderID, &s.OwnerID, &s.SharedWith, &s.Role, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, nil
}

// Invite repo

type inviteRepo struct {
	db *pgxpool.Pool
}

func NewInviteRepo(db *pgxpool.Pool) *inviteRepo {
	return &inviteRepo{db: db}
}

func (r *inviteRepo) Create(ctx context.Context, invite *domain.ShareInvite) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO share_invites (id, folder_id, owner_id, token, role, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		invite.ID, invite.FolderID, invite.OwnerID, invite.Token, invite.Role, invite.ExpiresAt, invite.CreatedAt,
	)
	return err
}

func (r *inviteRepo) FindByToken(ctx context.Context, token string) (*domain.ShareInvite, error) {
	var inv domain.ShareInvite
	err := r.db.QueryRow(ctx,
		`SELECT id, folder_id, owner_id, token, role, expires_at, accepted_at, created_at
		 FROM share_invites WHERE token = $1`, token,
	).Scan(&inv.ID, &inv.FolderID, &inv.OwnerID, &inv.Token, &inv.Role, &inv.ExpiresAt, &inv.AcceptedAt, &inv.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("invite not found")
	}
	return &inv, err
}

func (r *inviteRepo) MarkAccepted(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE share_invites SET accepted_at = $2 WHERE id = $1`,
		id, time.Now(),
	)
	return err
}
