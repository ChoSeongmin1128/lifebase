package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

func TestGooglePushProcessorProcessPendingDefaultLimitClaimsEligibleStatuses(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO google_push_outbox
		    (id, account_id, user_id, domain, op, local_resource_id, expected_updated_at, payload_json, status, attempt_count, next_retry_at, created_at, updated_at)
		 VALUES
		    ('out-pending', $1, $2, 'todo', 'update', 'missing-1', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-retry-ready', $1, $2, 'todo', 'update', 'missing-2', $3, '{}'::jsonb, 'retry', 1, $5, $4, $4),
		    ('out-stale-processing', $1, $2, 'todo', 'update', 'missing-3', $3, '{}'::jsonb, 'processing', 0, NULL, $4, $6),
		    ('out-retry-wait', $1, $2, 'todo', 'update', 'missing-4', $3, '{}'::jsonb, 'retry', 0, $7, $4, $4),
		    ('out-processing-fresh', $1, $2, 'todo', 'update', 'missing-5', $3, '{}'::jsonb, 'processing', 0, NULL, $4, $8)`,
		accountID,
		userID,
		now.Add(time.Minute),
		now,
		now.Add(-time.Minute),
		now.Add(-6*time.Minute),
		now.Add(10*time.Minute),
		now.Add(-2*time.Minute),
	)
	if err != nil {
		t.Fatalf("insert outbox rows: %v", err)
	}

	processor := NewGooglePushProcessor(db, &googleAuthStub{})
	processed, err := processor.ProcessPending(ctx, 0)
	if err != nil {
		t.Fatalf("ProcessPending limit<=0: %v", err)
	}
	if processed != 3 {
		t.Fatalf("expected processed=3, got %d", processed)
	}

	assertOutboxStatus(t, db, "out-pending", "done")
	assertOutboxStatus(t, db, "out-retry-ready", "done")
	assertOutboxStatus(t, db, "out-stale-processing", "done")
	assertOutboxStatus(t, db, "out-retry-wait", "retry")
	assertOutboxStatus(t, db, "out-processing-fresh", "processing")
}

func TestGooglePushProcessorClaimPendingErrorWhenPoolClosed(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	processor := NewGooglePushProcessor(db, &googleAuthStub{})
	db.Close()

	if _, err := processor.claimPending(context.Background(), 1); err == nil {
		t.Fatal("expected claimPending error on closed pool")
	}
}

func TestGooglePushProcessorProcessOneInactiveAndMaxAttemptsBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'reauth_required', true, $3, $3, $3),
		    ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', $2, 'u2@gmail.com', 'gid2', 'at', 'rt', $3, 'scope', 'active', false, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1', $1, 'g-list-1', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'L1', 0, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
		 VALUES ('todo-1', 'list-1', $1, 'g-todo-1', 'Todo', '', NULL, 'normal', false, false, 0, $2, $2, NULL)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert todo: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO google_push_outbox
		    (id, account_id, user_id, domain, op, local_resource_id, expected_updated_at, payload_json, status, attempt_count, next_retry_at, created_at, updated_at)
		 VALUES
		    ('out-inactive', $1, $2, 'todo', 'update', 'todo-1', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4),
		    ('out-max-attempt', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', $2, 'todo', 'update', 'todo-1', $5, '{}'::jsonb, 'pending', 7, NULL, $4, $4)`,
		accountID, userID, now.Add(time.Minute), now, now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("insert outbox rows: %v", err)
	}

	processor := NewGooglePushProcessor(db, &googleAuthStub{
		updateTaskFn: func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
			return &portout.GoogleAPIError{StatusCode: 500, Reason: "backendError", Message: "retryable"}
		},
	})

	err = processor.processOne(ctx, pushOutboxItem{
		ID:                "out-inactive",
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-1",
		ExpectedUpdatedAt: now.Add(time.Minute),
		AttemptCount:      0,
	})
	if err != nil {
		t.Fatalf("processOne inactive account: %v", err)
	}
	assertOutboxStatus(t, db, "out-inactive", "dead")
	assertOutboxLastErrorContains(t, db, "out-inactive", "not active")

	err = processor.processOne(ctx, pushOutboxItem{
		ID:                "out-max-attempt",
		AccountID:         "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:            userID,
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-1",
		ExpectedUpdatedAt: now.Add(time.Minute),
		AttemptCount:      7,
	})
	if err != nil {
		t.Fatalf("processOne max attempts: %v", err)
	}
	assertOutboxStatus(t, db, "out-max-attempt", "dead")
}

func TestGooglePushProcessorProcessCalendarPushAdditionalBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	const otherAccountID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3),
		    ($4, $2, 'u2@gmail.com', 'gid2', 'at', 'rt', $3, 'scope', 'active', false, $3, $3, $3)`,
		accountID, userID, now, otherAccountID,
	)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES
		    ('cal-main', $1, 'g-cal-main', $2, 'Main', 'google', true, true, false, false, $3, $3),
		    ('cal-other', $1, 'g-cal-other', $4, 'Other', 'google', false, true, false, false, $3, $3),
		    ('cal-no-google', $1, NULL, $2, 'NoGoogle', 'google', false, true, false, false, $3, $3)`,
		userID, accountID, now, otherAccountID,
	)
	if err != nil {
		t.Fatalf("insert calendars: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, created_at, updated_at, deleted_at)
		 VALUES
		    ('evt-main', 'cal-main', $1, 'g-evt-main', 'Main', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-other', 'cal-other', $1, 'g-evt-other', 'Other', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-no-cal-gid', 'cal-no-google', $1, 'g-evt-x', 'NoCalGoogle', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL)`,
		userID, now, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}

	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}
	stub := &googleAuthStub{}
	processor := NewGooglePushProcessor(db, stub)

	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "missing", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("calendar missing event should no-op: %v", err)
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-other", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("calendar account mismatch should no-op: %v", err)
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-no-cal-gid", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("calendar missing calendar google id should no-op: %v", err)
	}

	stub.updateCalendarEventFn = func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
		return nil, errors.New("update failed")
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "calendar", Op: "create", LocalResourceID: "evt-main", ExpectedUpdatedAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("expected non-404 update error on calendar create")
	}

	stub.updateCalendarEventFn = func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
		return nil, &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
	}
	stub.createCalendarEventFn = func(context.Context, portout.OAuthToken, string, portout.CalendarEventUpsertInput) (string, *string, error) {
		return "", nil, errors.New("create failed")
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "calendar", Op: "update", LocalResourceID: "evt-main", ExpectedUpdatedAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("expected create fallback error on calendar update")
	}
}

func TestGooglePushProcessorProcessTodoPushAdditionalBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	const otherAccountID = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1, $2, 'u1@gmail.com', 'gid', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3),
		    ($4, $2, 'u2@gmail.com', 'gid2', 'at', 'rt', $3, 'scope', 'active', false, $3, $3, $3)`,
		accountID, userID, now, otherAccountID,
	)
	if err != nil {
		t.Fatalf("insert accounts: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES
		    ('list-main', $1, 'g-list-main', $2, 'Main', 0, $3, $3),
		    ('list-other', $1, 'g-list-other', $4, 'Other', 1, $3, $3),
		    ('list-no-google', $1, NULL, $2, 'NoGoogle', 2, $3, $3)`,
		userID, accountID, now, otherAccountID,
	)
	if err != nil {
		t.Fatalf("insert lists: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
		 VALUES
		    ('todo-main', 'list-main', $1, 'g-todo-main', 'Main', '', NULL, 'normal', false, false, 0, $2, $2, NULL),
		    ('todo-other', 'list-other', $1, 'g-todo-other', 'Other', '', NULL, 'normal', false, false, 1, $2, $2, NULL),
		    ('todo-no-list-gid', 'list-no-google', $1, 'g-todo-x', 'NoListGoogle', '', NULL, 'normal', false, false, 2, $2, $2, NULL)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}
	stub := &googleAuthStub{}
	processor := NewGooglePushProcessor(db, stub)

	if err := processor.processTodoPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "missing", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("todo missing row should no-op: %v", err)
	}
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-other", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("todo account mismatch should no-op: %v", err)
	}
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-no-list-gid", ExpectedUpdatedAt: now.Add(time.Minute)}); err != nil {
		t.Fatalf("todo missing list google id should no-op: %v", err)
	}

	stub.updateTaskFn = func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
		return errors.New("update failed")
	}
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "todo", Op: "create", LocalResourceID: "todo-main", ExpectedUpdatedAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("expected non-404 update error on todo create")
	}

	stub.updateTaskFn = func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
		return &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
	}
	stub.createTaskFn = func(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error) {
		return "", errors.New("create failed")
	}
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{AccountID: accountID, UserID: userID, Domain: "todo", Op: "update", LocalResourceID: "todo-main", ExpectedUpdatedAt: now.Add(time.Minute)}); err == nil {
		t.Fatal("expected create fallback error on todo update")
	}
}

func assertOutboxStatus(t *testing.T, db *pgxpool.Pool, id, want string) {
	t.Helper()
	var got string
	if err := db.QueryRow(context.Background(), `SELECT status FROM google_push_outbox WHERE id = $1`, id).Scan(&got); err != nil {
		t.Fatalf("read outbox status %s: %v", id, err)
	}
	if got != want {
		t.Fatalf("outbox %s status mismatch: want=%s got=%s", id, want, got)
	}
}

func assertOutboxLastErrorContains(t *testing.T, db *pgxpool.Pool, id, wantPart string) {
	t.Helper()
	var lastErr *string
	if err := db.QueryRow(context.Background(), `SELECT last_error FROM google_push_outbox WHERE id = $1`, id).Scan(&lastErr); err != nil {
		t.Fatalf("read outbox error %s: %v", id, err)
	}
	if lastErr == nil || !strings.Contains(*lastErr, wantPart) {
		t.Fatalf("outbox %s error mismatch: want contains=%q got=%v", id, wantPart, lastErr)
	}
}
