package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
)

type fakeAuthRow struct {
	values []any
	err    error
}

func (r *fakeAuthRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if i >= len(r.values) || r.values[i] == nil {
			dv.Elem().Set(reflect.Zero(dv.Elem().Type()))
			continue
		}
		dv.Elem().Set(reflect.ValueOf(r.values[i]))
	}
	return nil
}

type fakeAuthConn struct {
	rowFn    func(string, ...any) googleSyncRow
	execFn   func(string, ...any) (pgconn.CommandTag, error)
	released bool
}

func (c *fakeAuthConn) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	if c.rowFn != nil {
		return c.rowFn(sql, args...)
	}
	return &fakeAuthRow{}
}

func (c *fakeAuthConn) Exec(_ context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	if c.execFn != nil {
		return c.execFn(sql, args...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (c *fakeAuthConn) Release() {
	c.released = true
}

func TestGooglePushProcessorHelperUnitBranches(t *testing.T) {
	prevRow := queryGooglePushRowFn
	prevExec := execGooglePushFn
	t.Cleanup(func() {
		queryGooglePushRowFn = prevRow
		execGooglePushFn = prevExec
	})

	now := time.Now().UTC().Truncate(time.Second)
	expiry := now.Add(time.Hour)

	t.Run("loaders_and_exec_helpers", func(t *testing.T) {
		queryGooglePushRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
			switch {
			case strings.Contains(sql, "FROM user_google_accounts"):
				return &fakeAuthRow{values: []any{"at", "rt", &expiry, "active"}}
			case strings.Contains(sql, "FROM events"):
				return &fakeAuthRow{values: []any{
					"evt-1", strPtr("g-evt-1"), "Event", "desc", "Seoul", now, now.Add(time.Hour), "Asia/Seoul",
					false, strPtr("1"), strPtr("RRULE"), strPtr("etag"), now, (*time.Time)(nil), strPtr("g-cal-1"), strPtr("acc-1"),
				}}
			case strings.Contains(sql, "FROM todos t"):
				return &fakeAuthRow{values: []any{
					"todo-1", "user-1", "list-1", strPtr("parent-1"), strPtr("g-todo-1"), "Todo", "memo",
					strPtr("2026-03-09"), false, 3, now, (*time.Time)(nil), strPtr("g-list-1"), strPtr("acc-1"),
				}}
			default:
				return &fakeAuthRow{err: pgx.ErrNoRows}
			}
		}

		p := NewGooglePushProcessor(nil, &googleAuthStub{})
		token, err := p.loadAccountToken(context.Background(), "user-1", "acc-1")
		if err != nil || token.AccessToken != "at" || token.Status != "active" || token.ExpiresAt == nil {
			t.Fatalf("loadAccountToken mismatch: token=%+v err=%v", token, err)
		}

		event, err := p.loadCalendarEvent(context.Background(), "user-1", "evt-1")
		if err != nil || event == nil || event.ID != "evt-1" || event.CalendarAccount == nil || *event.CalendarAccount != "acc-1" {
			t.Fatalf("loadCalendarEvent mismatch: event=%+v err=%v", event, err)
		}

		todo, err := p.loadTodo(context.Background(), "user-1", "todo-1")
		if err != nil || todo == nil || todo.ID != "todo-1" || todo.ListAccountID == nil || *todo.ListAccountID != "acc-1" {
			t.Fatalf("loadTodo mismatch: todo=%+v err=%v", todo, err)
		}

		var execSQL []string
		execGooglePushFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			execSQL = append(execSQL, sql)
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}
		if err := p.setEventGoogleMeta(context.Background(), "user-1", "evt-1", "g-evt-2", strPtr("etag-2")); err != nil {
			t.Fatalf("setEventGoogleMeta: %v", err)
		}
		if err := p.setTodoGoogleID(context.Background(), "user-1", "todo-1", "g-todo-2"); err != nil {
			t.Fatalf("setTodoGoogleID: %v", err)
		}
		if err := p.markDone(context.Background(), "out-1"); err != nil {
			t.Fatalf("markDone: %v", err)
		}
		if err := p.markRetry(context.Background(), "out-2", 30*time.Second, "retry"); err != nil {
			t.Fatalf("markRetry: %v", err)
		}
		if err := p.reschedule(context.Background(), "out-3", 15*time.Second, "busy"); err != nil {
			t.Fatalf("reschedule: %v", err)
		}
		if err := p.markDead(context.Background(), "out-4", "dead"); err != nil {
			t.Fatalf("markDead: %v", err)
		}
		if err := p.markAccountReauthRequired(context.Background(), "acc-1"); err != nil {
			t.Fatalf("markAccountReauthRequired: %v", err)
		}
		if len(execSQL) != 7 {
			t.Fatalf("expected 7 helper execs, got %d", len(execSQL))
		}
	})

	t.Run("resolve_todo_move_targets_and_sync", func(t *testing.T) {
		var moveCalls []moveTaskCall
		queryGooglePushRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
			switch {
			case strings.Contains(sql, "FROM todos\n\t\t\t  WHERE user_id") && strings.Contains(sql, "id = $2"):
				parentID := args[1].(string)
				switch parentID {
				case "parent-missing":
					return &fakeAuthRow{err: pgx.ErrNoRows}
				case "parent-nil-google":
					return &fakeAuthRow{values: []any{(*string)(nil)}}
				default:
					return &fakeAuthRow{values: []any{strPtr("g-parent")}}
				}
			case strings.Contains(sql, "ORDER BY sort_order DESC"):
				if args[2] == "todo-prev" {
					return &fakeAuthRow{values: []any{strPtr("g-prev")}}
				}
				if args[2] == "todo-prev-error" {
					return &fakeAuthRow{err: errors.New("previous lookup failed")}
				}
				return &fakeAuthRow{err: pgx.ErrNoRows}
			default:
				return &fakeAuthRow{err: pgx.ErrNoRows}
			}
		}

		p := NewGooglePushProcessor(nil, &googleAuthStub{
			moveTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, taskID string, parentTaskID, previousTaskID *string) error {
				moveCalls = append(moveCalls, moveTaskCall{taskID: taskID, parentTaskID: parentTaskID, previousTaskID: previousTaskID})
				return nil
			},
		})

		parentMissing := &localTodo{ID: "todo-1", UserID: "user-1", ListID: "list-1", ParentID: strPtr("parent-missing"), SortOrder: 1}
		if _, _, err := p.resolveTodoMoveTargets(context.Background(), parentMissing); err == nil || !strings.Contains(err.Error(), "parent todo missing") {
			t.Fatalf("expected parent missing error, got %v", err)
		}

		parentNoGoogle := &localTodo{ID: "todo-2", UserID: "user-1", ListID: "list-1", ParentID: strPtr("parent-nil-google"), SortOrder: 1}
		if _, _, err := p.resolveTodoMoveTargets(context.Background(), parentNoGoogle); err == nil || !strings.Contains(err.Error(), "google id is not ready") {
			t.Fatalf("expected parent google id error, got %v", err)
		}

		prevErr := &localTodo{ID: "todo-prev-error", UserID: "user-1", ListID: "list-1", SortOrder: 2}
		if _, _, err := p.resolveTodoMoveTargets(context.Background(), prevErr); err == nil || !strings.Contains(err.Error(), "previous lookup failed") {
			t.Fatalf("expected previous lookup error, got %v", err)
		}

		root := &localTodo{ID: "todo-root", UserID: "user-1", ListID: "list-1", GoogleID: strPtr("g-root"), ListGoogleID: strPtr("g-list"), SortOrder: 0}
		if err := p.syncTodoPlacement(context.Background(), portout.OAuthToken{}, "create", root); err != nil {
			t.Fatalf("create root placement should noop: %v", err)
		}

		movable := &localTodo{
			ID: "todo-prev", UserID: "user-1", ListID: "list-1", ParentID: strPtr("parent-1"),
			GoogleID: strPtr("g-task"), ListGoogleID: strPtr("g-list"), SortOrder: 4,
		}
		parentID, prevID, err := p.resolveTodoMoveTargets(context.Background(), movable)
		if err != nil || parentID == nil || *parentID != "g-parent" || prevID == nil || *prevID != "g-prev" {
			t.Fatalf("resolveTodoMoveTargets mismatch: parent=%v prev=%v err=%v", parentID, prevID, err)
		}
		if err := p.syncTodoPlacement(context.Background(), portout.OAuthToken{}, "update", movable); err != nil {
			t.Fatalf("syncTodoPlacement update: %v", err)
		}
		if len(moveCalls) != 1 || moveCalls[0].parentTaskID == nil || *moveCalls[0].parentTaskID != "g-parent" || moveCalls[0].previousTaskID == nil || *moveCalls[0].previousTaskID != "g-prev" {
			t.Fatalf("unexpected move calls: %#v", moveCalls)
		}
	})
}

func TestGooglePushProcessorProcessAndProcessPendingUnitBranches(t *testing.T) {
	prevAcquire := acquireGooglePushConnFn
	prevLock := googlePushTryAdvisoryLockFn
	prevLoad := googlePushLoadAccountTokenFn
	prevRow := queryGooglePushRowFn
	prevExec := execGooglePushFn
	prevRows := queryGooglePushRowsFn
	t.Cleanup(func() {
		acquireGooglePushConnFn = prevAcquire
		googlePushTryAdvisoryLockFn = prevLock
		googlePushLoadAccountTokenFn = prevLoad
		queryGooglePushRowFn = prevRow
		execGooglePushFn = prevExec
		queryGooglePushRowsFn = prevRows
	})

	now := time.Now().UTC().Truncate(time.Second)
	fakeConn := &fakeAuthConn{}
	acquireGooglePushConnFn = func(context.Context, *pgxpool.Pool) (googlePushConn, error) { return fakeConn, nil }
	googlePushTryAdvisoryLockFn = func(context.Context, googlePushConn, int64) (bool, error) { return true, nil }

	execStatuses := make([]string, 0, 8)
	execGooglePushFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
		execStatuses = append(execStatuses, sql)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}

	queryGooglePushRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
		if strings.Contains(sql, "FROM todos t") {
			return &fakeAuthRow{values: []any{
				"todo-1", "user-1", "list-1", (*string)(nil), strPtr("g-todo-1"), "Todo", "memo",
				strPtr("2026-03-09"), false, 1, now, (*time.Time)(nil), strPtr("g-list-1"), strPtr("acc-1"),
			}}
		}
		return &fakeAuthRow{err: pgx.ErrNoRows}
	}

	processor := NewGooglePushProcessor(nil, &googleAuthStub{
		updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, _ string, _ portout.TodoUpsertInput) error {
			return nil
		},
	})
	googlePushLoadAccountTokenFn = func(*googlePushProcessor, context.Context, string, string) (*accountToken, error) {
		return &accountToken{AccessToken: "at", RefreshToken: "rt", Status: "active"}, nil
	}

	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-done", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("processOne success: %v", err)
	}

	googlePushTryAdvisoryLockFn = func(context.Context, googlePushConn, int64) (bool, error) { return false, nil }
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-busy", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("processOne busy should reschedule: %v", err)
	}
	googlePushTryAdvisoryLockFn = func(context.Context, googlePushConn, int64) (bool, error) { return true, nil }

	googlePushLoadAccountTokenFn = func(*googlePushProcessor, context.Context, string, string) (*accountToken, error) {
		return nil, pgx.ErrNoRows
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-missing", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("processOne missing account should mark dead: %v", err)
	}

	googlePushLoadAccountTokenFn = func(*googlePushProcessor, context.Context, string, string) (*accountToken, error) {
		return &accountToken{Status: "revoked"}, nil
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-inactive", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("processOne inactive account should mark dead: %v", err)
	}

	googlePushLoadAccountTokenFn = func(*googlePushProcessor, context.Context, string, string) (*accountToken, error) {
		return &accountToken{AccessToken: "at", RefreshToken: "rt", Status: "active"}, nil
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-unsupported", AccountID: "acc-1", UserID: "user-1", Domain: "weird", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("unsupported domain should mark dead: %v", err)
	}

	processor.googleAuth = &googleAuthStub{
		updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, _ string, _ portout.TodoUpsertInput) error {
			return &portout.GoogleAPIError{StatusCode: 401, Reason: "authError", Message: "expired"}
		},
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-auth", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("auth error should mark dead after reauth: %v", err)
	}

	processor.googleAuth = &googleAuthStub{
		updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, _ string, _ portout.TodoUpsertInput) error {
			return &portout.GoogleAPIError{StatusCode: 503, Reason: "backendError", Message: "retry"}
		},
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-retry", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", AttemptCount: 0, ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("retryable error should schedule retry: %v", err)
	}
	if err := processor.processOne(context.Background(), pushOutboxItem{
		ID: "out-dead", AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "todo-1", AttemptCount: pushOutboxMaxAttempts - 1, ExpectedUpdatedAt: now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("max attempts should mark dead: %v", err)
	}

	queryGooglePushRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakePushRows{data: [][]any{
			{"out-pending", "acc-1", "user-1", "todo", "update", "todo-1", now, 0},
		}}, nil
	}
	processor.googleAuth = &googleAuthStub{
		updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, _ string, _ portout.TodoUpsertInput) error {
			return errors.New("boom")
		},
	}
	processed, err := processor.ProcessPending(context.Background(), 0)
	if err != nil || processed != 1 {
		t.Fatalf("ProcessPending mismatch: processed=%d err=%v", processed, err)
	}
	if len(execStatuses) == 0 {
		t.Fatal("expected exec statuses to be recorded")
	}
}

func TestGoogleSyncCoordinatorUnitBranches(t *testing.T) {
	prevRow := queryGoogleSyncCoordinatorRowFn
	prevRows := queryGoogleSyncCoordinatorRowsFn
	prevExec := execGoogleSyncCoordinatorFn
	prevAcquire := acquireGoogleSyncConnFn
	prevLock := coordinatorTryAdvisoryLockFn
	t.Cleanup(func() {
		queryGoogleSyncCoordinatorRowFn = prevRow
		queryGoogleSyncCoordinatorRowsFn = prevRows
		execGoogleSyncCoordinatorFn = prevExec
		acquireGoogleSyncConnFn = prevAcquire
		coordinatorTryAdvisoryLockFn = prevLock
	})

	now := time.Now().UTC().Truncate(time.Second)
	rowBySQL := func(sql string, args ...any) googleSyncRow {
		switch {
		case strings.Contains(sql, "SELECT value FROM user_settings"):
			key, _ := args[1].(string)
			switch {
			case key == "k_true":
				return &fakeAuthRow{values: []any{"true"}}
			case key == "k_false":
				return &fakeAuthRow{values: []any{"FALSE"}}
			case key == "k_other":
				return &fakeAuthRow{values: []any{"other"}}
			default:
				return &fakeAuthRow{err: pgx.ErrNoRows}
			}
		case strings.Contains(sql, "last_hourly_sync_at"):
			return &fakeAuthRow{values: []any{&now}}
		case strings.Contains(sql, "last_tab_sync_at"):
			return &fakeAuthRow{values: []any{(*time.Time)(nil)}}
		case strings.Contains(sql, "last_action_sync_at"):
			return &fakeAuthRow{err: pgx.ErrNoRows}
		default:
			return &fakeAuthRow{err: pgx.ErrNoRows}
		}
	}
	queryGoogleSyncCoordinatorRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
		return rowBySQL(sql, args...)
	}

	coordinator := NewGoogleSyncCoordinator(nil, &syncerStub{})
	if got, err := coordinator.getSettingBool(context.Background(), "user-1", "k_true", false); err != nil || !got {
		t.Fatalf("getSettingBool true mismatch: got=%v err=%v", got, err)
	}
	if got, err := coordinator.getSettingBool(context.Background(), "user-1", "k_false", true); err != nil || got {
		t.Fatalf("getSettingBool false mismatch: got=%v err=%v", got, err)
	}
	if got, err := coordinator.getSettingBool(context.Background(), "user-1", "k_other", true); err != nil || !got {
		t.Fatalf("getSettingBool fallback mismatch: got=%v err=%v", got, err)
	}
	if last, err := coordinator.lastSyncAt(context.Background(), "acc-1", "hourly"); err != nil || last.IsZero() {
		t.Fatalf("lastSyncAt hourly mismatch: last=%v err=%v", last, err)
	}
	if last, err := coordinator.lastSyncAt(context.Background(), "acc-1", "tab_heartbeat"); err != nil || !last.IsZero() {
		t.Fatalf("lastSyncAt tab mismatch: last=%v err=%v", last, err)
	}
	if last, err := coordinator.lastSyncAt(context.Background(), "acc-1", "page_action"); err != nil || !last.IsZero() {
		t.Fatalf("lastSyncAt no rows mismatch: last=%v err=%v", last, err)
	}

	execSQL := make([]string, 0, 8)
	execGoogleSyncCoordinatorFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
		execSQL = append(execSQL, sql)
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	for _, reason := range []string{"hourly", "tab_heartbeat", "page_enter", "manual"} {
		if err := coordinator.touchSyncReason(context.Background(), "acc-1", "user-1", reason, now); err != nil {
			t.Fatalf("touchSyncReason(%s): %v", reason, err)
		}
	}
	if err := coordinator.updateSyncSuccess(context.Background(), "acc-1", now); err != nil {
		t.Fatalf("updateSyncSuccess: %v", err)
	}
	if err := coordinator.updateSyncError(context.Background(), "acc-1", "boom", now); err != nil {
		t.Fatalf("updateSyncError: %v", err)
	}
	if err := coordinator.markAccountReauthRequired(context.Background(), "acc-1"); err != nil {
		t.Fatalf("markAccountReauthRequired: %v", err)
	}
	if len(execSQL) != 7 {
		t.Fatalf("expected 7 coordinator execs, got %d", len(execSQL))
	}

	queryGoogleSyncCoordinatorRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakePushRows{data: [][]any{
			{"acc-1", "user-1", "u1@gmail.com", "gid-1", "at", "rt", (*time.Time)(nil), "scope", "active", true, now, now, now},
		}}, nil
	}
	byUser, err := coordinator.listActiveAccountsByUser(context.Background(), "user-1")
	if err != nil || len(byUser) != 1 || byUser[0].ID != "acc-1" {
		t.Fatalf("listActiveAccountsByUser mismatch: items=%+v err=%v", byUser, err)
	}
	all, err := coordinator.listActiveAccounts(context.Background())
	if err != nil || len(all) != 1 || all[0].UserID != "user-1" {
		t.Fatalf("listActiveAccounts mismatch: items=%+v err=%v", all, err)
	}

	fakeConn := &fakeAuthConn{}
	acquireGoogleSyncConnFn = func(context.Context, *pgxpool.Pool) (googleSyncConn, error) { return fakeConn, nil }
	coordinatorTryAdvisoryLockFn = func(context.Context, googleSyncConn, int64) (bool, error) { return true, nil }
	stub := &syncerStub{}
	coordinator.syncer = stub
	performed, err := coordinator.syncAccountIfDue(context.Background(), "user-1", &authdomain.GoogleAccount{ID: "acc-1", UserID: "user-1"}, portout.GoogleSyncOptions{SyncCalendar: true}, "manual")
	if err != nil || !performed || stub.calls != 1 {
		t.Fatalf("syncAccountIfDue success mismatch: performed=%v calls=%d err=%v", performed, stub.calls, err)
	}

	stub = &syncerStub{errByAccount: map[string]error{"acc-1": &portout.GoogleAPIError{StatusCode: 401, Reason: "authError", Message: "expired"}}}
	coordinator.syncer = stub
	if performed, err = coordinator.syncAccountIfDue(context.Background(), "user-1", &authdomain.GoogleAccount{ID: "acc-1", UserID: "user-1"}, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil || !performed {
		t.Fatalf("syncAccountIfDue auth error mismatch: performed=%v err=%v", performed, err)
	}
}

func TestGoogleAccountSyncerUnitBranches(t *testing.T) {
	prevRows := queryGoogleSyncRowsFn
	prevRow := queryGoogleSyncRowFn
	prevExec := execGoogleSyncFn
	prevAcquire := acquireGoogleSyncConnFn
	prevLock := googleSyncTryAdvisoryLockFn
	t.Cleanup(func() {
		queryGoogleSyncRowsFn = prevRows
		queryGoogleSyncRowFn = prevRow
		execGoogleSyncFn = prevExec
		acquireGoogleSyncConnFn = prevAcquire
		googleSyncTryAdvisoryLockFn = prevLock
	})

	now := time.Now().UTC().Truncate(time.Second)
	queryGoogleSyncRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
		switch {
		case strings.Contains(sql, "SELECT id FROM calendars"):
			return &fakeAuthRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "SELECT sync_token FROM calendars"):
			return &fakeAuthRow{values: []any{strPtr("sync-1")}}
		case strings.Contains(sql, "SELECT id FROM todo_lists"):
			return &fakeAuthRow{err: pgx.ErrNoRows}
		case strings.Contains(sql, "todo_done_retention_period"):
			return &fakeAuthRow{values: []any{"1m"}}
		default:
			return &fakeAuthRow{err: pgx.ErrNoRows}
		}
	}
	queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
		switch {
		case strings.Contains(sql, "SELECT id, google_id\n\t\t   FROM todos"):
			return &fakePushRows{data: [][]any{{"local-parent", strPtr("g-parent")}, {"local-child", strPtr("g-child")}}}, nil
		case strings.Contains(sql, "SELECT DISTINCT t.id"):
			return &fakePushRows{data: [][]any{{"blocked-local"}}}, nil
		default:
			return &fakePushRows{}, nil
		}
	}
	execGoogleSyncFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
		switch {
		case strings.Contains(sql, "UPDATE events"):
			return pgconn.NewCommandTag("UPDATE 0"), nil
		case strings.Contains(sql, "UPDATE todos\n\t\t\t\t SET title"):
			if args[2] == "g-parent" {
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
			return pgconn.NewCommandTag("UPDATE 0"), nil
		default:
			return pgconn.NewCommandTag("INSERT 1"), nil
		}
	}

	listCalendarCalls := 0
	syncer := NewGoogleAccountSyncer(nil, &googleAuthStub{
		listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
			return []portout.OAuthCalendar{
				{GoogleID: "g-cal-1", Name: "Main", Kind: "google"},
				{GoogleID: "g-cal-special", Name: "Holiday", Kind: "holiday", IsSpecial: true},
			}, nil
		},
		listCalendarEventsFn: func(_ context.Context, _ portout.OAuthToken, _ string, pageToken, syncToken string, _, _ *time.Time) (*portout.OAuthCalendarEventsPage, error) {
			listCalendarCalls++
			if syncToken != "" && listCalendarCalls == 1 {
				return nil, &portout.GoogleAPIError{StatusCode: 410, Reason: "updatedMinTooLongAgo", Message: "expired"}
			}
			if pageToken == "" {
				start := now.Add(time.Hour)
				end := start.Add(time.Hour)
				return &portout.OAuthCalendarEventsPage{
					Events: []portout.OAuthCalendarEvent{
						{GoogleID: "evt-1", Title: "Event", StartTime: &start, EndTime: &end, Timezone: "UTC"},
						{GoogleID: "evt-del", Status: "cancelled"},
					},
					NextSyncToken: "next-sync",
				}, nil
			}
			return &portout.OAuthCalendarEventsPage{}, nil
		},
		listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
			return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
		},
		listTasksFn: func(_ context.Context, _ portout.OAuthToken, _ string, pageToken string) (*portout.OAuthTasksPage, error) {
			if pageToken != "" {
				return &portout.OAuthTasksPage{}, nil
			}
			parentGoogleID := "g-parent"
			return &portout.OAuthTasksPage{
				Items: []portout.OAuthTask{
					{GoogleID: "g-parent", Title: "Parent"},
					{GoogleID: "g-child", Title: "Child", ParentGoogleID: &parentGoogleID},
					{GoogleID: "g-deleted", IsDeleted: true},
					{GoogleID: "g-done", Title: "Done", IsDone: true},
				},
			}, nil
		},
	})

	if err := syncer.syncCalendarsAndEvents(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err != nil {
		t.Fatalf("syncCalendarsAndEvents success mismatch: %v", err)
	}
	if err := syncer.syncTaskListsAndTodos(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err != nil {
		t.Fatalf("syncTaskListsAndTodos success mismatch: %v", err)
	}

	ids, err := syncer.loadLocalTodoIDsByGoogleID(context.Background(), "user-1", "list-1")
	if err != nil || ids["g-parent"] != "local-parent" || ids["g-child"] != "local-child" {
		t.Fatalf("loadLocalTodoIDsByGoogleID mismatch: ids=%v err=%v", ids, err)
	}

	root := normalizeGoogleTaskParent(portout.OAuthTask{GoogleID: "g-root"}, map[string]portout.OAuthTask{}, map[string]string{})
	if root != nil {
		t.Fatalf("expected nil root parent, got %v", root)
	}
	cycleParent := "g-cycle"
	if got := normalizeGoogleTaskParent(
		portout.OAuthTask{GoogleID: "g-cycle", ParentGoogleID: &cycleParent},
		map[string]portout.OAuthTask{"g-cycle": {GoogleID: "g-cycle", ParentGoogleID: &cycleParent}},
		map[string]string{"g-cycle": "local-cycle"},
	); got != nil {
		t.Fatalf("expected cycle to return nil, got %v", got)
	}
	parentGoogle := "g-parent"
	if got := normalizeGoogleTaskParent(
		portout.OAuthTask{GoogleID: "g-child", ParentGoogleID: &parentGoogle},
		map[string]portout.OAuthTask{"g-parent": {GoogleID: "g-parent"}},
		map[string]string{"g-parent": "local-parent"},
	); got == nil || *got != "local-parent" {
		t.Fatalf("expected normalized parent id, got %v", got)
	}

	cutoff := syncer.resolveTodoDoneRetentionCutoff(context.Background(), "user-1", now)
	if cutoff == nil || cutoff.After(now) {
		t.Fatalf("unexpected cutoff: %v", cutoff)
	}

	fakeConn := &fakeAuthConn{}
	acquireGoogleSyncConnFn = func(context.Context, *pgxpool.Pool) (googleSyncConn, error) { return fakeConn, nil }
	googleSyncTryAdvisoryLockFn = func(context.Context, googleSyncConn, int64) (bool, error) { return true, nil }
	queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
		if strings.Contains(sql, "FROM calendars c") {
			return &fakePushRows{data: [][]any{{"cal-1", "g-cal-1", "acc-1", "at", "rt", (*time.Time)(nil)}}}, nil
		}
		return &fakePushRows{}, nil
	}
	result, err := syncer.BackfillEvents(context.Background(), "user-1", now, now.Add(2*time.Hour), nil)
	if err != nil || result == nil {
		t.Fatalf("BackfillEvents success mismatch: result=%+v err=%v", result, err)
	}
}

func TestGoogleAuthRemainingUnitCoverageBranches(t *testing.T) {
	t.Run("default advisory lock helpers return scan errors", func(t *testing.T) {
		pushConn := &fakeAuthConn{
			rowFn: func(string, ...any) googleSyncRow {
				return &fakeAuthRow{err: errors.New("push lock scan fail")}
			},
		}
		if _, err := googlePushTryAdvisoryLockFn(context.Background(), pushConn, 1); err == nil || err.Error() != "push lock scan fail" {
			t.Fatalf("expected push lock scan error, got %v", err)
		}

		syncConn := &fakeAuthConn{
			rowFn: func(string, ...any) googleSyncRow {
				return &fakeAuthRow{err: errors.New("sync lock scan fail")}
			},
		}
		if _, err := googleSyncTryAdvisoryLockFn(context.Background(), syncConn, 2); err == nil || err.Error() != "sync lock scan fail" {
			t.Fatalf("expected sync lock scan error, got %v", err)
		}
	})

	t.Run("processTodoPush setTodoGoogleID and placement branches", func(t *testing.T) {
		prevRow := queryGooglePushRowFn
		prevExec := execGooglePushFn
		t.Cleanup(func() {
			queryGooglePushRowFn = prevRow
			execGooglePushFn = prevExec
		})

		now := time.Now().UTC().Truncate(time.Second)
		queryGooglePushRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
			switch {
			case strings.Contains(sql, "FROM todos t"):
				todoID := args[1].(string)
				switch todoID {
				case "create-missing-google":
					return &fakeAuthRow{values: []any{
						todoID, "user-1", "list-1", (*string)(nil), (*string)(nil), "Todo", "memo",
						(*string)(nil), false, 0, now, (*time.Time)(nil), strPtr("g-list"), strPtr("acc-1"),
					}}
				case "update-missing-google":
					return &fakeAuthRow{values: []any{
						todoID, "user-1", "list-1", (*string)(nil), (*string)(nil), "Todo", "memo",
						(*string)(nil), false, 0, now, (*time.Time)(nil), strPtr("g-list"), strPtr("acc-1"),
					}}
				case "update-404":
					return &fakeAuthRow{values: []any{
						todoID, "user-1", "list-1", (*string)(nil), strPtr("g-existing"), "Todo", "memo",
						(*string)(nil), false, 0, now, (*time.Time)(nil), strPtr("g-list"), strPtr("acc-1"),
					}}
				case "resolve-error":
					return &fakeAuthRow{values: []any{
						todoID, "user-1", "list-1", strPtr("parent-1"), strPtr("g-resolve"), "Todo", "memo",
						(*string)(nil), false, 1, now, (*time.Time)(nil), strPtr("g-list"), strPtr("acc-1"),
					}}
				default:
					return &fakeAuthRow{err: pgx.ErrNoRows}
				}
			case strings.Contains(sql, "FROM todos") && strings.Contains(sql, "id = $2"):
				if args[1] == "parent-1" {
					return &fakeAuthRow{err: errors.New("parent lookup fail")}
				}
			}
			return &fakeAuthRow{err: pgx.ErrNoRows}
		}

		execGooglePushFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "SET google_id = $3") {
				return pgconn.NewCommandTag("UPDATE 0"), errors.New("set google id fail")
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}

		processor := NewGooglePushProcessor(nil, &googleAuthStub{
			createTaskFn: func(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error) {
				return "g-created", nil
			},
			updateTaskFn: func(_ context.Context, _ portout.OAuthToken, _ string, taskID string, _ portout.TodoUpsertInput) error {
				if taskID == "g-existing" {
					return &portout.GoogleAPIError{StatusCode: 404}
				}
				return nil
			},
		})

		for _, tc := range []struct {
			name string
			item pushOutboxItem
			want string
		}{
			{name: "create_set_google_id_error", item: pushOutboxItem{AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "create", LocalResourceID: "create-missing-google", ExpectedUpdatedAt: now}, want: "set google id fail"},
			{name: "update_create_set_google_id_error", item: pushOutboxItem{AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "update-missing-google", ExpectedUpdatedAt: now}, want: "set google id fail"},
			{name: "update_404_set_google_id_error", item: pushOutboxItem{AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "update-404", ExpectedUpdatedAt: now}, want: "set google id fail"},
			{name: "sync_placement_resolve_error", item: pushOutboxItem{AccountID: "acc-1", UserID: "user-1", Domain: "todo", Op: "update", LocalResourceID: "resolve-error", ExpectedUpdatedAt: now}, want: "parent lookup fail"},
		} {
			err := processor.processTodoPush(context.Background(), portout.OAuthToken{}, tc.item)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("%s: expected %q, got %v", tc.name, tc.want, err)
			}
		}

		if err := processor.syncTodoPlacement(context.Background(), portout.OAuthToken{}, "update", nil); err != nil {
			t.Fatalf("expected nil todo placement to noop, got %v", err)
		}
	})

	t.Run("syncTaskListsAndTodos error branches and helpers", func(t *testing.T) {
		prevRows := queryGoogleSyncRowsFn
		prevRow := queryGoogleSyncRowFn
		prevExec := execGoogleSyncFn
		t.Cleanup(func() {
			queryGoogleSyncRowsFn = prevRows
			queryGoogleSyncRowFn = prevRow
			execGoogleSyncFn = prevExec
		})

		now := time.Now().UTC().Truncate(time.Second)
		syncer := NewGoogleAccountSyncer(nil, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
			},
			listTasksFn: func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
				parent := "g-parent"
				return &portout.OAuthTasksPage{Items: []portout.OAuthTask{
					{GoogleID: "g-parent", Title: "Parent"},
					{GoogleID: "g-child", Title: "Child", ParentGoogleID: &parent},
				}}, nil
			},
		})

		queryGoogleSyncRowFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) googleSyncRow {
			if strings.Contains(sql, "SELECT value FROM user_settings") {
				return &fakeGoogleSyncRow{err: pgx.ErrNoRows}
			}
			if strings.Contains(sql, "SELECT id FROM todo_lists") {
				return &fakeGoogleSyncRow{values: []any{"list-1"}}
			}
			return &fakeGoogleSyncRow{err: pgx.ErrNoRows}
		}

		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			switch {
			case strings.Contains(sql, "FROM todos") && strings.Contains(sql, "google_id IS NOT NULL"):
				return &fakePushRows{data: [][]any{{"local-parent", strPtr("g-parent")}, {"local-child", strPtr("g-child")}}}, nil
			case strings.Contains(sql, "FROM google_push_outbox"):
				return &fakePushRows{next: []bool{false}}, nil
			default:
				return &fakePushRows{next: []bool{false}}, nil
			}
		}

		execGoogleSyncFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			switch {
			case strings.Contains(sql, "UPDATE todos") && strings.Contains(sql, "SET title ="):
				return pgconn.NewCommandTag("UPDATE 0"), errors.New("update todo fail")
			default:
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
		}
		if err := syncer.syncTaskListsAndTodos(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err == nil || !strings.Contains(err.Error(), "update todo fail") {
			t.Fatalf("expected update todo fail, got %v", err)
		}

		execGoogleSyncFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "SET title =") {
				return pgconn.NewCommandTag("UPDATE 1"), nil
			}
			if strings.Contains(sql, "SET parent_id =") {
				return pgconn.NewCommandTag("UPDATE 0"), errors.New("hierarchy fail")
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}
		if err := syncer.syncTaskListsAndTodos(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err == nil || !strings.Contains(err.Error(), "hierarchy fail") {
			t.Fatalf("expected hierarchy fail, got %v", err)
		}

		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			if strings.Contains(sql, "FROM todos") && strings.Contains(sql, "google_id IS NOT NULL") {
				return nil, errors.New("query todo ids fail")
			}
			return &fakePushRows{next: []bool{false}}, nil
		}
		if _, err := syncer.loadLocalTodoIDsByGoogleID(context.Background(), "user-1", "list-1"); err == nil || !strings.Contains(err.Error(), "query todo ids fail") {
			t.Fatalf("expected query todo ids fail, got %v", err)
		}

		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{true, false}, scan: errors.New("scan todo ids fail")}, nil
		}
		if _, err := syncer.loadLocalTodoIDsByGoogleID(context.Background(), "user-1", "list-1"); err == nil || !strings.Contains(err.Error(), "scan todo ids fail") {
			t.Fatalf("expected scan todo ids fail, got %v", err)
		}

		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{false}, err: errors.New("rows todo ids fail")}, nil
		}
		if _, err := syncer.loadLocalTodoIDsByGoogleID(context.Background(), "user-1", "list-1"); err == nil || !strings.Contains(err.Error(), "rows todo ids fail") {
			t.Fatalf("expected rows todo ids fail, got %v", err)
		}

		parent := "g-parent"
		if got := normalizeGoogleTaskParent(
			portout.OAuthTask{GoogleID: "child", ParentGoogleID: &parent},
			map[string]portout.OAuthTask{"g-parent": {GoogleID: "g-parent", IsDeleted: true}},
			map[string]string{"g-parent": "local-parent"},
		); got != nil {
			t.Fatalf("expected deleted parent to normalize nil, got %v", got)
		}
		if got := normalizeGoogleTaskParent(
			portout.OAuthTask{GoogleID: "child", ParentGoogleID: &parent},
			map[string]portout.OAuthTask{"g-parent": {GoogleID: "g-parent"}},
			map[string]string{},
		); got != nil {
			t.Fatalf("expected missing local parent to normalize nil, got %v", got)
		}

		rowsCall := 0
		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			if strings.Contains(sql, "FROM todos") && strings.Contains(sql, "google_id IS NOT NULL") {
				rowsCall++
				return &fakePushRows{data: [][]any{{"local-parent", strPtr("g-parent")}}}, nil
			}
			if strings.Contains(sql, "FROM google_push_outbox") {
				return nil, errors.New("pending delete fail")
			}
			return &fakePushRows{next: []bool{false}}, nil
		}
		if err := syncer.syncTaskListsAndTodos(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err == nil || !strings.Contains(err.Error(), "pending delete fail") {
			t.Fatalf("expected pending delete fail, got %v", err)
		}
		if rowsCall != 1 {
			t.Fatalf("expected one local todo id load before pending delete error, got %d", rowsCall)
		}

		queryGoogleSyncRowsFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgx.Rows, error) {
			if strings.Contains(sql, "FROM todos") && strings.Contains(sql, "google_id IS NOT NULL") {
				rowsCall++
				if rowsCall >= 3 {
					return nil, errors.New("reload todo ids fail")
				}
				return &fakePushRows{data: [][]any{{"local-parent", strPtr("g-parent")}}}, nil
			}
			if strings.Contains(sql, "FROM google_push_outbox") {
				return &fakePushRows{next: []bool{false}}, nil
			}
			return &fakePushRows{next: []bool{false}}, nil
		}
		rowsCall = 1
		execGoogleSyncFn = func(_ context.Context, _ *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "SET title =") {
				return pgconn.NewCommandTag("UPDATE 0"), nil
			}
			return pgconn.NewCommandTag("UPDATE 1"), nil
		}
		if err := syncer.syncTaskListsAndTodos(context.Background(), "user-1", "acc-1", portout.OAuthToken{}, now); err == nil || !strings.Contains(err.Error(), "reload todo ids fail") {
			t.Fatalf("expected reload todo ids fail, got %v", err)
		}
	})
}
