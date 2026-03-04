package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	portout "lifebase/internal/auth/port/out"
)

const (
	pushOutboxMaxAttempts = 8
)

type googlePushProcessor struct {
	db         *pgxpool.Pool
	googleAuth portout.GoogleAuthClient
}

type pushOutboxItem struct {
	ID                string
	AccountID         string
	UserID            string
	Domain            string
	Op                string
	LocalResourceID   string
	ExpectedUpdatedAt time.Time
	AttemptCount      int
}

type accountToken struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
	Status       string
}

type localCalendarEvent struct {
	ID               string
	GoogleID         *string
	Title            string
	Description      string
	Location         string
	StartTime        time.Time
	EndTime          time.Time
	Timezone         string
	IsAllDay         bool
	ColorID          *string
	RecurrenceRule   *string
	ETag             *string
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	CalendarGoogleID *string
	CalendarAccount  *string
}

type localTodo struct {
	ID            string
	GoogleID      *string
	Title         string
	Notes         string
	DueDate       *string
	IsDone        bool
	UpdatedAt     time.Time
	DeletedAt     *time.Time
	ListGoogleID  *string
	ListAccountID *string
}

func NewGooglePushProcessor(db *pgxpool.Pool, googleAuth portout.GoogleAuthClient) *googlePushProcessor {
	return &googlePushProcessor{
		db:         db,
		googleAuth: googleAuth,
	}
}

func (p *googlePushProcessor) ProcessPending(ctx context.Context, limit int) (int, error) {
	if p.googleAuth == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 50
	}

	items, err := p.claimPending(ctx, limit)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, item := range items {
		if err := p.processOne(ctx, item); err != nil {
			_ = p.markRetry(ctx, item.ID, nextRetryDelay(item.AttemptCount+1), shortenError(err))
		}
		processed++
	}

	return processed, nil
}

func (p *googlePushProcessor) claimPending(ctx context.Context, limit int) ([]pushOutboxItem, error) {
	rows, err := p.db.Query(ctx,
		`WITH picked AS (
		   SELECT id
		     FROM google_push_outbox
		    WHERE status = 'pending'
		       OR (status = 'retry' AND (next_retry_at IS NULL OR next_retry_at <= NOW()))
		       OR (status = 'processing' AND updated_at < NOW() - INTERVAL '5 minutes')
		    ORDER BY created_at ASC
		    LIMIT $1
		    FOR UPDATE SKIP LOCKED
		 )
		 UPDATE google_push_outbox o
		    SET status = 'processing',
		        updated_at = NOW()
		   FROM picked
		  WHERE o.id = picked.id
		RETURNING o.id, o.account_id, o.user_id, o.domain, o.op, o.local_resource_id,
		          o.expected_updated_at, o.attempt_count`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]pushOutboxItem, 0, limit)
	for rows.Next() {
		var item pushOutboxItem
		if err := rows.Scan(
			&item.ID,
			&item.AccountID,
			&item.UserID,
			&item.Domain,
			&item.Op,
			&item.LocalResourceID,
			&item.ExpectedUpdatedAt,
			&item.AttemptCount,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (p *googlePushProcessor) processOne(ctx context.Context, item pushOutboxItem) error {
	lockKey := advisoryLockKey(item.AccountID)
	var locked bool
	if err := p.db.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return p.reschedule(ctx, item.ID, 15*time.Second, "account lock busy")
	}
	defer p.db.Exec(context.Background(), `SELECT pg_advisory_unlock($1)`, lockKey)

	account, err := p.loadAccountToken(ctx, item.UserID, item.AccountID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return p.markDead(ctx, item.ID, "google account not found")
		}
		return err
	}
	if account.Status != "active" {
		return p.markDead(ctx, item.ID, "google account is not active")
	}

	token := portout.OAuthToken{
		AccessToken:  account.AccessToken,
		RefreshToken: account.RefreshToken,
	}
	if account.ExpiresAt != nil {
		token.Expiry = *account.ExpiresAt
	}

	switch item.Domain {
	case "calendar":
		err = p.processCalendarPush(ctx, token, item)
	case "todo":
		err = p.processTodoPush(ctx, token, item)
	default:
		err = fmt.Errorf("unsupported outbox domain: %s", item.Domain)
	}

	if err == nil {
		return p.markDone(ctx, item.ID)
	}

	if isGoogleStatus(err, 401) || isGoogleStatus(err, 403) {
		_ = p.markAccountReauthRequired(ctx, item.AccountID)
		return p.markDead(ctx, item.ID, shortenError(err))
	}

	nextAttempt := item.AttemptCount + 1
	if nextAttempt >= pushOutboxMaxAttempts || !isRetryableGoogleError(err) {
		return p.markDead(ctx, item.ID, shortenError(err))
	}
	return p.markRetry(ctx, item.ID, nextRetryDelay(nextAttempt), shortenError(err))
}

func (p *googlePushProcessor) processCalendarPush(
	ctx context.Context,
	token portout.OAuthToken,
	item pushOutboxItem,
) error {
	event, err := p.loadCalendarEvent(ctx, item.UserID, item.LocalResourceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if event.CalendarAccount == nil || *event.CalendarAccount != item.AccountID {
		return nil
	}
	if event.CalendarGoogleID == nil || *event.CalendarGoogleID == "" {
		return nil
	}

	if event.UpdatedAt.After(item.ExpectedUpdatedAt) {
		return nil
	}

	if item.Op == "delete" || event.DeletedAt != nil {
		if event.GoogleID == nil || *event.GoogleID == "" {
			return nil
		}
		err := p.googleAuth.DeleteCalendarEvent(ctx, token, *event.CalendarGoogleID, *event.GoogleID)
		if err != nil && !isGoogleStatus(err, 404) {
			return err
		}
		return nil
	}

	input := portout.CalendarEventUpsertInput{
		Title:          event.Title,
		Description:    event.Description,
		Location:       event.Location,
		StartTime:      event.StartTime,
		EndTime:        event.EndTime,
		Timezone:       event.Timezone,
		IsAllDay:       event.IsAllDay,
		ColorID:        event.ColorID,
		RecurrenceRule: event.RecurrenceRule,
		ETag:           event.ETag,
	}

	switch item.Op {
	case "create":
		if event.GoogleID != nil && *event.GoogleID != "" {
			etag, err := p.googleAuth.UpdateCalendarEvent(ctx, token, *event.CalendarGoogleID, *event.GoogleID, input)
			if err == nil {
				return p.setEventGoogleMeta(ctx, item.UserID, event.ID, *event.GoogleID, etag)
			}
			if !isGoogleStatus(err, 404) {
				return err
			}
		}
		googleID, etag, err := p.googleAuth.CreateCalendarEvent(ctx, token, *event.CalendarGoogleID, input)
		if err != nil {
			return err
		}
		return p.setEventGoogleMeta(ctx, item.UserID, event.ID, googleID, etag)
	case "update":
		if event.GoogleID == nil || *event.GoogleID == "" {
			googleID, etag, err := p.googleAuth.CreateCalendarEvent(ctx, token, *event.CalendarGoogleID, input)
			if err != nil {
				return err
			}
			return p.setEventGoogleMeta(ctx, item.UserID, event.ID, googleID, etag)
		}

		etag, err := p.googleAuth.UpdateCalendarEvent(ctx, token, *event.CalendarGoogleID, *event.GoogleID, input)
		if err == nil {
			return p.setEventGoogleMeta(ctx, item.UserID, event.ID, *event.GoogleID, etag)
		}
		if !isGoogleStatus(err, 404) {
			return err
		}

		googleID, etag, createErr := p.googleAuth.CreateCalendarEvent(ctx, token, *event.CalendarGoogleID, input)
		if createErr != nil {
			return createErr
		}
		return p.setEventGoogleMeta(ctx, item.UserID, event.ID, googleID, etag)
	default:
		return fmt.Errorf("unsupported calendar op: %s", item.Op)
	}
}

func (p *googlePushProcessor) processTodoPush(
	ctx context.Context,
	token portout.OAuthToken,
	item pushOutboxItem,
) error {
	todo, err := p.loadTodo(ctx, item.UserID, item.LocalResourceID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}
	if todo.ListAccountID == nil || *todo.ListAccountID != item.AccountID {
		return nil
	}
	if todo.ListGoogleID == nil || *todo.ListGoogleID == "" {
		return nil
	}

	if todo.UpdatedAt.After(item.ExpectedUpdatedAt) {
		return nil
	}

	if item.Op == "delete" || todo.DeletedAt != nil {
		if todo.GoogleID == nil || *todo.GoogleID == "" {
			return nil
		}
		err := p.googleAuth.DeleteTask(ctx, token, *todo.ListGoogleID, *todo.GoogleID)
		if err != nil && !isGoogleStatus(err, 404) {
			return err
		}
		return nil
	}

	input := portout.TodoUpsertInput{
		Title:   todo.Title,
		Notes:   todo.Notes,
		DueDate: todo.DueDate,
		IsDone:  todo.IsDone,
	}

	switch item.Op {
	case "create":
		if todo.GoogleID != nil && *todo.GoogleID != "" {
			err := p.googleAuth.UpdateTask(ctx, token, *todo.ListGoogleID, *todo.GoogleID, input)
			if err == nil {
				return nil
			}
			if !isGoogleStatus(err, 404) {
				return err
			}
		}

		googleID, err := p.googleAuth.CreateTask(ctx, token, *todo.ListGoogleID, input)
		if err != nil {
			return err
		}
		return p.setTodoGoogleID(ctx, item.UserID, todo.ID, googleID)
	case "update":
		if todo.GoogleID == nil || *todo.GoogleID == "" {
			googleID, err := p.googleAuth.CreateTask(ctx, token, *todo.ListGoogleID, input)
			if err != nil {
				return err
			}
			return p.setTodoGoogleID(ctx, item.UserID, todo.ID, googleID)
		}

		err := p.googleAuth.UpdateTask(ctx, token, *todo.ListGoogleID, *todo.GoogleID, input)
		if err == nil {
			return nil
		}
		if !isGoogleStatus(err, 404) {
			return err
		}

		googleID, createErr := p.googleAuth.CreateTask(ctx, token, *todo.ListGoogleID, input)
		if createErr != nil {
			return createErr
		}
		return p.setTodoGoogleID(ctx, item.UserID, todo.ID, googleID)
	default:
		return fmt.Errorf("unsupported todo op: %s", item.Op)
	}
}

func (p *googlePushProcessor) loadAccountToken(ctx context.Context, userID, accountID string) (*accountToken, error) {
	var row accountToken
	err := p.db.QueryRow(ctx,
		`SELECT access_token, refresh_token, token_expires_at, status
		   FROM user_google_accounts
		  WHERE id = $1 AND user_id = $2`,
		accountID,
		userID,
	).Scan(&row.AccessToken, &row.RefreshToken, &row.ExpiresAt, &row.Status)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (p *googlePushProcessor) loadCalendarEvent(ctx context.Context, userID, eventID string) (*localCalendarEvent, error) {
	var row localCalendarEvent
	err := p.db.QueryRow(ctx,
		`SELECT e.id, e.google_id, e.title, e.description, e.location, e.start_time, e.end_time,
		        e.timezone, e.is_all_day, e.color_id, e.recurrence_rule, e.etag,
		        e.updated_at, e.deleted_at, c.google_id, c.google_account_id
		   FROM events e
		   JOIN calendars c ON c.id = e.calendar_id AND c.user_id = e.user_id
		  WHERE e.user_id = $1 AND e.id = $2`,
		userID,
		eventID,
	).Scan(
		&row.ID,
		&row.GoogleID,
		&row.Title,
		&row.Description,
		&row.Location,
		&row.StartTime,
		&row.EndTime,
		&row.Timezone,
		&row.IsAllDay,
		&row.ColorID,
		&row.RecurrenceRule,
		&row.ETag,
		&row.UpdatedAt,
		&row.DeletedAt,
		&row.CalendarGoogleID,
		&row.CalendarAccount,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (p *googlePushProcessor) loadTodo(ctx context.Context, userID, todoID string) (*localTodo, error) {
	var row localTodo
	err := p.db.QueryRow(ctx,
		`SELECT t.id, t.google_id, t.title, t.notes,
		        CASE WHEN t.due IS NULL THEN NULL ELSE to_char(t.due, 'YYYY-MM-DD') END AS due_date,
		        t.is_done, t.updated_at, t.deleted_at, l.google_id, l.google_account_id
		   FROM todos t
		   JOIN todo_lists l ON l.id = t.list_id AND l.user_id = t.user_id
		  WHERE t.user_id = $1 AND t.id = $2`,
		userID,
		todoID,
	).Scan(
		&row.ID,
		&row.GoogleID,
		&row.Title,
		&row.Notes,
		&row.DueDate,
		&row.IsDone,
		&row.UpdatedAt,
		&row.DeletedAt,
		&row.ListGoogleID,
		&row.ListAccountID,
	)
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (p *googlePushProcessor) setEventGoogleMeta(
	ctx context.Context,
	userID, eventID, googleID string,
	etag *string,
) error {
	_, err := p.db.Exec(ctx,
		`UPDATE events
		    SET google_id = $3,
		        etag = COALESCE($4, etag)
		  WHERE user_id = $1 AND id = $2`,
		userID,
		eventID,
		googleID,
		etag,
	)
	return err
}

func (p *googlePushProcessor) setTodoGoogleID(ctx context.Context, userID, todoID, googleID string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE todos
		    SET google_id = $3
		  WHERE user_id = $1 AND id = $2`,
		userID,
		todoID,
		googleID,
	)
	return err
}

func (p *googlePushProcessor) markDone(ctx context.Context, id string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE google_push_outbox
		    SET status = 'done',
		        next_retry_at = NULL,
		        last_error = NULL,
		        updated_at = NOW()
		  WHERE id = $1`,
		id,
	)
	return err
}

func (p *googlePushProcessor) markRetry(ctx context.Context, id string, delay time.Duration, reason string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE google_push_outbox
		    SET status = 'retry',
		        attempt_count = attempt_count + 1,
		        next_retry_at = NOW() + $2::interval,
		        last_error = $3,
		        updated_at = NOW()
		  WHERE id = $1`,
		id,
		pgInterval(delay),
		reason,
	)
	return err
}

func (p *googlePushProcessor) reschedule(ctx context.Context, id string, delay time.Duration, reason string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE google_push_outbox
		    SET status = 'retry',
		        next_retry_at = NOW() + $2::interval,
		        last_error = $3,
		        updated_at = NOW()
		  WHERE id = $1`,
		id,
		pgInterval(delay),
		reason,
	)
	return err
}

func (p *googlePushProcessor) markDead(ctx context.Context, id, reason string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE google_push_outbox
		    SET status = 'dead',
		        attempt_count = attempt_count + 1,
		        next_retry_at = NULL,
		        last_error = $2,
		        updated_at = NOW()
		  WHERE id = $1`,
		id,
		reason,
	)
	return err
}

func (p *googlePushProcessor) markAccountReauthRequired(ctx context.Context, accountID string) error {
	_, err := p.db.Exec(ctx,
		`UPDATE user_google_accounts
		    SET status = 'reauth_required',
		        updated_at = NOW()
		  WHERE id = $1`,
		accountID,
	)
	return err
}

func isRetryableGoogleError(err error) bool {
	var apiErr *portout.GoogleAPIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == 401 || apiErr.StatusCode == 403 {
			return false
		}
		if apiErr.StatusCode == 429 {
			return true
		}
		return apiErr.StatusCode >= 500 && apiErr.StatusCode <= 599
	}
	return true
}

func isGoogleStatus(err error, status int) bool {
	var apiErr *portout.GoogleAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == status
}

func nextRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}

	steps := attempt
	if steps > 8 {
		steps = 8
	}
	delay := (1 << (steps - 1)) * 10
	seconds := time.Duration(delay) * time.Second
	if seconds > 30*time.Minute {
		return 30 * time.Minute
	}
	return seconds
}

func shortenError(err error) string {
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(err.Error())
	if len(text) <= 512 {
		return text
	}
	return text[:512]
}

func pgInterval(d time.Duration) string {
	if d < time.Second {
		d = time.Second
	}
	return fmt.Sprintf("%d seconds", int(d.Seconds()))
}
