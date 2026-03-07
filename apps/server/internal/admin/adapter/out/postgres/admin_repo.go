package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	admindomain "lifebase/internal/admin/domain"
	adminout "lifebase/internal/admin/port/out"
)

type AdminRepo struct {
	db *pgxpool.Pool
}

func NewAdminRepo(db *pgxpool.Pool) *AdminRepo {
	return &AdminRepo{db: db}
}

func (r *AdminRepo) IsActiveAdmin(ctx context.Context, userID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM admin_users WHERE user_id = $1 AND is_active = true)`,
		userID,
	).Scan(&exists)
	return exists, err
}

func (r *AdminRepo) FindByUserID(ctx context.Context, userID string) (*admindomain.AdminUser, error) {
	return r.findOne(ctx, `SELECT id, user_id, role, is_active, created_by, created_at, updated_at FROM admin_users WHERE user_id = $1`, userID)
}

func (r *AdminRepo) FindByID(ctx context.Context, adminID string) (*admindomain.AdminUser, error) {
	return r.findOne(ctx, `SELECT id, user_id, role, is_active, created_by, created_at, updated_at FROM admin_users WHERE id = $1`, adminID)
}

func (r *AdminRepo) List(ctx context.Context) ([]*admindomain.AdminUser, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, role, is_active, created_by, created_at, updated_at
		 FROM admin_users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdminRows(rows)
}

func (r *AdminRepo) Create(ctx context.Context, admin *admindomain.AdminUser) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO admin_users (id, user_id, role, is_active, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		admin.ID, admin.UserID, string(admin.Role), admin.IsActive, admin.CreatedBy, admin.CreatedAt, admin.UpdatedAt,
	)
	return err
}

func (r *AdminRepo) Update(ctx context.Context, admin *admindomain.AdminUser) error {
	_, err := r.db.Exec(ctx,
		`UPDATE admin_users
		 SET role = $2, is_active = $3, updated_at = $4
		 WHERE id = $1`,
		admin.ID, string(admin.Role), admin.IsActive, admin.UpdatedAt,
	)
	return err
}

func (r *AdminRepo) CountActiveSuperAdmins(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM admin_users WHERE role = 'super_admin' AND is_active = true`,
	).Scan(&n)
	return n, err
}

func (r *AdminRepo) ListByUserID(ctx context.Context, userID string) ([]adminout.GoogleAccountRecord, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, google_email, google_id, status, is_primary, connected_at
		 FROM user_google_accounts
		 WHERE user_id = $1
		 ORDER BY is_primary DESC, connected_at ASC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGoogleAccountRows(rows)
}

func (r *AdminRepo) UpdateStatus(ctx context.Context, accountID, userID, status string) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE user_google_accounts
		 SET status = $3, updated_at = now()
		 WHERE id = $1 AND user_id = $2`,
		accountID, userID, status,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *AdminRepo) findOne(ctx context.Context, query, value string) (*admindomain.AdminUser, error) {
	row := &admindomain.AdminUser{}
	var role string
	err := r.db.QueryRow(ctx, query, value).Scan(
		&row.ID, &row.UserID, &role, &row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	row.Role = admindomain.Role(role)
	return row, nil
}

func scanAdminRows(rows pgx.Rows) ([]*admindomain.AdminUser, error) {
	out := make([]*admindomain.AdminUser, 0)
	for rows.Next() {
		row := &admindomain.AdminUser{}
		var role string
		if err := rows.Scan(&row.ID, &row.UserID, &role, &row.IsActive, &row.CreatedBy, &row.CreatedAt, &row.UpdatedAt); err != nil {
			return nil, err
		}
		row.Role = admindomain.Role(role)
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func scanGoogleAccountRows(rows pgx.Rows) ([]adminout.GoogleAccountRecord, error) {
	out := make([]adminout.GoogleAccountRecord, 0)
	for rows.Next() {
		var row adminout.GoogleAccountRecord
		if err := rows.Scan(&row.ID, &row.UserID, &row.GoogleEmail, &row.GoogleID, &row.Status, &row.IsPrimary, &row.ConnectedAt); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
