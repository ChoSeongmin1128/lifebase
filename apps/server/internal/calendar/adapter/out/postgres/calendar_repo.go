package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/calendar/domain"
)

type calendarRepo struct {
	db *pgxpool.Pool
}

func NewCalendarRepo(db *pgxpool.Pool) *calendarRepo {
	return &calendarRepo{db: db}
}

func (r *calendarRepo) Create(ctx context.Context, cal *domain.Calendar) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO calendars (id, user_id, google_id, name, color_id, is_primary, is_visible, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		cal.ID, cal.UserID, cal.GoogleID, cal.Name, cal.ColorID, cal.IsPrimary, cal.IsVisible, cal.CreatedAt, cal.UpdatedAt,
	)
	return err
}

func (r *calendarRepo) FindByID(ctx context.Context, userID, id string) (*domain.Calendar, error) {
	var c domain.Calendar
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, google_id, name, color_id, is_primary, is_visible, sync_token, created_at, updated_at
		 FROM calendars WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&c.ID, &c.UserID, &c.GoogleID, &c.Name, &c.ColorID, &c.IsPrimary, &c.IsVisible, &c.SyncToken, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("calendar not found")
	}
	return &c, err
}

func (r *calendarRepo) ListByUser(ctx context.Context, userID string) ([]*domain.Calendar, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, google_id, name, color_id, is_primary, is_visible, sync_token, created_at, updated_at
		 FROM calendars WHERE user_id = $1 ORDER BY is_primary DESC, name ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var calendars []*domain.Calendar
	for rows.Next() {
		var c domain.Calendar
		if err := rows.Scan(&c.ID, &c.UserID, &c.GoogleID, &c.Name, &c.ColorID, &c.IsPrimary, &c.IsVisible, &c.SyncToken, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		calendars = append(calendars, &c)
	}
	return calendars, nil
}

func (r *calendarRepo) Update(ctx context.Context, cal *domain.Calendar) error {
	_, err := r.db.Exec(ctx,
		`UPDATE calendars SET name = $3, color_id = $4, is_visible = $5, updated_at = $6
		 WHERE id = $1 AND user_id = $2`,
		cal.ID, cal.UserID, cal.Name, cal.ColorID, cal.IsVisible, cal.UpdatedAt,
	)
	return err
}

func (r *calendarRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM calendars WHERE id = $1`, id)
	return err
}
