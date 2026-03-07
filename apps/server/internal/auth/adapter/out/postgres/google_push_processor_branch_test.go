package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

func TestGooglePushProcessorLockBusyReschedule(t *testing.T) {
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
		 VALUES ('out-lock', $1, $2, 'todo', 'update', 'todo-1', $3, '{}'::jsonb, 'pending', 0, NULL, $4, $4)`,
		accountID, userID, now.Add(time.Minute), now,
	)
	if err != nil {
		t.Fatalf("insert outbox row: %v", err)
	}

	processor := NewGooglePushProcessor(db, &googleAuthStub{})
	lockKey := advisoryLockKey(accountID)
	lockConn, err := db.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire lock conn: %v", err)
	}
	defer lockConn.Release()
	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_lock($1)`, lockKey); err != nil {
		t.Fatalf("acquire advisory lock: %v", err)
	}

	item := pushOutboxItem{
		ID:                "out-lock",
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-1",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}
	if err := processor.processOne(ctx, item); err != nil {
		t.Fatalf("processOne lock-busy path: %v", err)
	}
	if _, err := lockConn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, lockKey); err != nil {
		t.Fatalf("unlock advisory lock: %v", err)
	}

	var status string
	var lastErr *string
	if err := db.QueryRow(ctx, `SELECT status, last_error FROM google_push_outbox WHERE id = 'out-lock'`).Scan(&status, &lastErr); err != nil {
		t.Fatalf("read outbox status: %v", err)
	}
	if status != "retry" || lastErr == nil || *lastErr == "" {
		t.Fatalf("expected retry with last_error, got status=%s err=%v", status, lastErr)
	}
}

func TestGooglePushProcessorCalendarAndTodoBranches(t *testing.T) {
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
		`INSERT INTO calendars (id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'g-cal-1', $2, 'C1', 'google', true, true, false, false, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert calendar: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO events (id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, created_at, updated_at, deleted_at)
		 VALUES
		    ('evt-existing', 'cal-1', $1, 'g-evt-existing', 'E', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-nogid', 'cal-1', $1, NULL, 'E2', '', '', $2, $3, 'Asia/Seoul', false, $2, $2, NULL),
		    ('evt-late', 'cal-1', $1, 'g-evt-late', 'Late', '', '', $2, $3, 'Asia/Seoul', false, $2, $4, NULL)`,
		userID, now, now.Add(time.Hour), now.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert events: %v", err)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, google_id, google_account_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1', $1, 'g-list-1', $2, 'L1', 0, $3, $3)`,
		userID, accountID, now,
	)
	if err != nil {
		t.Fatalf("insert todo list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, google_id, title, notes, due, priority, is_done, is_pinned, sort_order, created_at, updated_at, deleted_at)
		 VALUES
		    ('todo-existing', 'list-1', $1, 'g-todo-existing', 'T', '', NULL, 'normal', false, false, 0, $2, $2, NULL),
		    ('todo-nogid', 'list-1', $1, NULL, 'T2', '', NULL, 'normal', false, false, 1, $2, $2, NULL),
		    ('todo-late', 'list-1', $1, 'g-todo-late', 'TL', '', NULL, 'normal', false, false, 2, $2, $3, NULL)`,
		userID, now, now.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatalf("insert todos: %v", err)
	}

	stub := &googleAuthStub{}
	processor := NewGooglePushProcessor(db, stub)
	token := portout.OAuthToken{AccessToken: "at", RefreshToken: "rt"}
	calCreateSeq := 0
	todoCreateSeq := 0

	// calendar create: existing google id + update success path
	stub.updateCalendarEventFn = func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
		etag := "etag-updated"
		return &etag, nil
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "create",
		LocalResourceID:   "evt-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("calendar create with update path: %v", err)
	}

	// calendar create: update 404 -> create fallback path
	stub.updateCalendarEventFn = func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
		return nil, &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
	}
	stub.createCalendarEventFn = func(context.Context, portout.OAuthToken, string, portout.CalendarEventUpsertInput) (string, *string, error) {
		calCreateSeq++
		etag := "etag-created"
		return fmt.Sprintf("g-evt-created-%d", calCreateSeq), &etag, nil
	}
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "create",
		LocalResourceID:   "evt-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("calendar create fallback path: %v", err)
	}

	// calendar update: missing google id -> create path
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "update",
		LocalResourceID:   "evt-nogid",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("calendar update with missing google id: %v", err)
	}

	// calendar delete with missing google id should no-op
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "delete",
		LocalResourceID:   "evt-nogid",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("calendar delete with missing google id should no-op: %v", err)
	}

	// calendar expected update guard should no-op
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "update",
		LocalResourceID:   "evt-late",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("calendar expected-update guard should no-op: %v", err)
	}

	// calendar unsupported op branch
	if err := processor.processCalendarPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "calendar",
		Op:                "unsupported",
		LocalResourceID:   "evt-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err == nil {
		t.Fatal("expected unsupported calendar op error")
	}

	// todo create: existing google id + update success
	stub.updateTaskFn = func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error { return nil }
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "create",
		LocalResourceID:   "todo-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("todo create with update path: %v", err)
	}

	// todo create: update 404 -> create fallback
	stub.updateTaskFn = func(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
		return &portout.GoogleAPIError{StatusCode: 404, Reason: "notFound", Message: "nf"}
	}
	stub.createTaskFn = func(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error) {
		todoCreateSeq++
		return fmt.Sprintf("g-todo-created-%d", todoCreateSeq), nil
	}
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "create",
		LocalResourceID:   "todo-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("todo create fallback path: %v", err)
	}

	// todo update: missing google id -> create
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-nogid",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("todo update with missing google id: %v", err)
	}

	// todo expected update guard no-op
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-late",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("todo expected-update guard should no-op: %v", err)
	}

	// todo unsupported op branch
	if err := processor.processTodoPush(ctx, token, pushOutboxItem{
		AccountID:         accountID,
		UserID:            userID,
		Domain:            "todo",
		Op:                "unsupported",
		LocalResourceID:   "todo-existing",
		ExpectedUpdatedAt: now.Add(time.Minute),
	}); err == nil {
		t.Fatal("expected unsupported todo op error")
	}
}
