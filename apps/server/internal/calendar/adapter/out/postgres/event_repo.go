package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/calendar/domain"
)

type eventRepo struct {
	db *pgxpool.Pool
}

var (
	scanEventRowsFn    = scanEventRows
	scanReminderRowsFn = scanReminderRows
	queryEventRowsFn   = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
		return db.Query(ctx, sql, args...)
	}
)

func NewEventRepo(db *pgxpool.Pool) *eventRepo {
	return &eventRepo{db: db}
}

func (r *eventRepo) Create(ctx context.Context, event *domain.Event) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, color_id, recurrence_rule, etag, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)`,
		event.ID, event.CalendarID, event.UserID, event.GoogleID, event.Title, event.Description,
		event.Location, event.StartTime, event.EndTime, event.Timezone, event.IsAllDay,
		event.ColorID, event.RecurrenceRule, event.ETag, event.CreatedAt, event.UpdatedAt,
	)
	return err
}

func (r *eventRepo) FindByID(ctx context.Context, userID, id string) (*domain.Event, error) {
	var e domain.Event
	err := r.db.QueryRow(ctx,
		`SELECT id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, color_id, recurrence_rule, etag, created_at, updated_at, deleted_at
		 FROM events WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`, id, userID,
	).Scan(&e.ID, &e.CalendarID, &e.UserID, &e.GoogleID, &e.Title, &e.Description,
		&e.Location, &e.StartTime, &e.EndTime, &e.Timezone, &e.IsAllDay,
		&e.ColorID, &e.RecurrenceRule, &e.ETag, &e.CreatedAt, &e.UpdatedAt, &e.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("event not found")
	}
	return &e, err
}

func (r *eventRepo) ListByRange(ctx context.Context, userID string, calendarIDs []string, start, end string) ([]*domain.Event, error) {
	query := `SELECT id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, color_id, recurrence_rule, etag, created_at, updated_at, deleted_at
		 FROM events WHERE user_id = $1 AND deleted_at IS NULL
		 AND start_time < $3 AND end_time > $2`

	args := []any{userID, start, end}

	if len(calendarIDs) > 0 {
		placeholders := make([]string, len(calendarIDs))
		for i, cid := range calendarIDs {
			args = append(args, cid)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		query += fmt.Sprintf(" AND calendar_id IN (%s)", strings.Join(placeholders, ","))
	}

	query += " ORDER BY start_time ASC, end_time DESC"

	rows, err := queryEventRowsFn(ctx, r.db, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEventRowsFn(rows)
}

func (r *eventRepo) Update(ctx context.Context, event *domain.Event) error {
	_, err := r.db.Exec(ctx,
		`UPDATE events SET title = $3, description = $4, location = $5, start_time = $6, end_time = $7,
		 timezone = $8, is_all_day = $9, color_id = $10, recurrence_rule = $11, updated_at = $12
		 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		event.ID, event.UserID, event.Title, event.Description, event.Location,
		event.StartTime, event.EndTime, event.Timezone, event.IsAllDay,
		event.ColorID, event.RecurrenceRule, event.UpdatedAt,
	)
	return err
}

func (r *eventRepo) SoftDelete(ctx context.Context, userID, id string) error {
	now := time.Now()
	_, err := r.db.Exec(ctx,
		`UPDATE events SET deleted_at = $3, updated_at = $3 WHERE id = $1 AND user_id = $2 AND deleted_at IS NULL`,
		id, userID, now,
	)
	return err
}

// Reminder repo

type reminderRepo struct {
	db *pgxpool.Pool
}

func NewReminderRepo(db *pgxpool.Pool) *reminderRepo {
	return &reminderRepo{db: db}
}

func (r *reminderRepo) CreateBatch(ctx context.Context, reminders []domain.EventReminder) error {
	for _, rem := range reminders {
		_, err := r.db.Exec(ctx,
			`INSERT INTO event_reminders (id, event_id, method, minutes, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			rem.ID, rem.EventID, rem.Method, rem.Minutes, rem.CreatedAt,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *reminderRepo) ListByEvent(ctx context.Context, eventID string) ([]domain.EventReminder, error) {
	rows, err := queryEventRowsFn(ctx, r.db,
		`SELECT id, event_id, method, minutes, created_at
		 FROM event_reminders WHERE event_id = $1 ORDER BY minutes ASC`, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanReminderRowsFn(rows)
}

func (r *reminderRepo) DeleteByEvent(ctx context.Context, eventID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM event_reminders WHERE event_id = $1`, eventID)
	return err
}

func scanReminderRows(rows pgx.Rows) ([]domain.EventReminder, error) {
	var reminders []domain.EventReminder
	for rows.Next() {
		var rem domain.EventReminder
		if err := rows.Scan(&rem.ID, &rem.EventID, &rem.Method, &rem.Minutes, &rem.CreatedAt); err != nil {
			return nil, err
		}
		reminders = append(reminders, rem)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reminders, nil
}

func scanEventRows(rows pgx.Rows) ([]*domain.Event, error) {
	var events []*domain.Event
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.CalendarID, &e.UserID, &e.GoogleID, &e.Title, &e.Description,
			&e.Location, &e.StartTime, &e.EndTime, &e.Timezone, &e.IsAllDay,
			&e.ColorID, &e.RecurrenceRule, &e.ETag, &e.CreatedAt, &e.UpdatedAt, &e.DeletedAt); err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
