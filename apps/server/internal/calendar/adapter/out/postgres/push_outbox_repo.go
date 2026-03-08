package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type eventPushOutboxRepo struct {
	db *pgxpool.Pool
}

var (
	queryCalendarOutboxAccountFn = func(ctx context.Context, db *pgxpool.Pool, userID, eventID string) (*string, error) {
		var accountID *string
		err := db.QueryRow(ctx,
			`SELECT c.google_account_id
			 FROM events e
			 JOIN calendars c ON c.id = e.calendar_id AND c.user_id = e.user_id
			 WHERE e.user_id = $1 AND e.id = $2`,
			userID,
			eventID,
		).Scan(&accountID)
		return accountID, err
	}
	insertCalendarOutboxFn = func(ctx context.Context, db *pgxpool.Pool, accountID, userID, op, eventID string, expectedUpdatedAt, now time.Time) error {
		_, err := db.Exec(ctx,
			`INSERT INTO google_push_outbox (
			   id, account_id, user_id, domain, op, local_resource_id, expected_updated_at,
			   payload_json, status, attempt_count, next_retry_at, created_at, updated_at
			 ) VALUES (
			   $1, $2, $3, 'calendar', $4, $5, $6,
			   '{}'::jsonb, 'pending', 0, $7, $7, $7
			 )
			 ON CONFLICT (domain, op, local_resource_id, expected_updated_at) DO NOTHING`,
			uuid.New().String(),
			accountID,
			userID,
			op,
			eventID,
			expectedUpdatedAt,
			now,
		)
		return err
	}
)

func NewEventPushOutboxRepo(db *pgxpool.Pool) *eventPushOutboxRepo {
	return &eventPushOutboxRepo{db: db}
}

func (r *eventPushOutboxRepo) EnqueueCreate(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, eventID, "create", expectedUpdatedAt)
}

func (r *eventPushOutboxRepo) EnqueueUpdate(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, eventID, "update", expectedUpdatedAt)
}

func (r *eventPushOutboxRepo) EnqueueDelete(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, eventID, "delete", expectedUpdatedAt)
}

func (r *eventPushOutboxRepo) enqueue(
	ctx context.Context,
	userID, eventID, op string,
	expectedUpdatedAt time.Time,
) error {
	accountID, err := queryCalendarOutboxAccountFn(ctx, r.db, userID, eventID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if accountID == nil || *accountID == "" {
		return nil
	}

	now := time.Now()
	return insertCalendarOutboxFn(ctx, r.db, *accountID, userID, op, eventID, expectedUpdatedAt, now)
}
