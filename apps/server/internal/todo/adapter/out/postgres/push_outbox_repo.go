package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type todoPushOutboxRepo struct {
	db *pgxpool.Pool
}

func NewTodoPushOutboxRepo(db *pgxpool.Pool) *todoPushOutboxRepo {
	return &todoPushOutboxRepo{db: db}
}

func (r *todoPushOutboxRepo) EnqueueCreate(ctx context.Context, userID, todoID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, todoID, "create", expectedUpdatedAt)
}

func (r *todoPushOutboxRepo) EnqueueUpdate(ctx context.Context, userID, todoID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, todoID, "update", expectedUpdatedAt)
}

func (r *todoPushOutboxRepo) EnqueueDelete(ctx context.Context, userID, todoID string, expectedUpdatedAt time.Time) error {
	return r.enqueue(ctx, userID, todoID, "delete", expectedUpdatedAt)
}

func (r *todoPushOutboxRepo) enqueue(
	ctx context.Context,
	userID, todoID, op string,
	expectedUpdatedAt time.Time,
) error {
	var accountID *string
	err := r.db.QueryRow(ctx,
		`SELECT l.google_account_id
		 FROM todos t
		 JOIN todo_lists l ON l.id = t.list_id AND l.user_id = t.user_id
		 WHERE t.user_id = $1 AND t.id = $2`,
		userID,
		todoID,
	).Scan(&accountID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	}
	if accountID == nil || *accountID == "" {
		return nil
	}

	now := time.Now()
	_, err = r.db.Exec(ctx,
		`INSERT INTO google_push_outbox (
		   id, account_id, user_id, domain, op, local_resource_id, expected_updated_at,
		   payload_json, status, attempt_count, next_retry_at, created_at, updated_at
		 ) VALUES (
		   $1, $2, $3, 'todo', $4, $5, $6,
		   '{}'::jsonb, 'pending', 0, $7, $7, $7
		 )
		 ON CONFLICT (domain, op, local_resource_id, expected_updated_at) DO NOTHING`,
		uuid.New().String(),
		*accountID,
		userID,
		op,
		todoID,
		expectedUpdatedAt,
		now,
	)
	return err
}
