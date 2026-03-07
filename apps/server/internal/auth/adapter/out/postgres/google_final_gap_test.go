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
	"lifebase/internal/testutil/dbtest"
)

type fakeGoogleSyncRow struct {
	values []any
	err    error
}

func (r *fakeGoogleSyncRow) Scan(dest ...any) error {
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

func TestGooglePushProcessorFinalBranches(t *testing.T) {
	prevLock := googlePushTryAdvisoryLockFn
	prevLoad := googlePushLoadAccountTokenFn
	t.Cleanup(func() {
		googlePushTryAdvisoryLockFn = prevLock
		googlePushLoadAccountTokenFn = prevLoad
	})

	t.Run("process_pending_marks_retry_when_process_one_returns_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)

		_, err := db.Exec(ctx, `INSERT INTO google_push_outbox
			(id, account_id, user_id, domain, op, local_resource_id, expected_updated_at, payload_json, status, attempt_count, created_at, updated_at)
		 VALUES ('out-lock-err', 'acc-1', 'user-1', 'todo', 'update', 'todo-1', $1, '{}'::jsonb, 'pending', 0, $1, $1)`, now)
		if err != nil {
			t.Fatalf("insert outbox: %v", err)
		}

		googlePushTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return false, errors.New("lock scan fail")
		}

		processed, err := NewGooglePushProcessor(db, &googleAuthStub{}).ProcessPending(ctx, 1)
		if err != nil {
			t.Fatalf("ProcessPending: %v", err)
		}
		if processed != 1 {
			t.Fatalf("expected processed=1, got %d", processed)
		}
		assertOutboxStatus(t, db, "out-lock-err", "retry")
	})

	t.Run("process_one_returns_unexpected_load_account_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)

		googlePushTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return true, nil
		}
		googlePushLoadAccountTokenFn = func(*googlePushProcessor, context.Context, string, string) (*accountToken, error) {
			return nil, errors.New("load account failed")
		}

		err := NewGooglePushProcessor(db, &googleAuthStub{}).processOne(context.Background(), pushOutboxItem{
			ID:                "out-load-error",
			AccountID:         "acc-1",
			UserID:            "user-1",
			Domain:            "todo",
			Op:                "update",
			LocalResourceID:   "todo-1",
			ExpectedUpdatedAt: time.Now().UTC(),
		})
		if err == nil || !strings.Contains(err.Error(), "load account failed") {
			t.Fatalf("expected load account error, got %v", err)
		}
	})

	t.Run("calendar_update_non_404_error_returns_directly", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		const userID = "11111111-1111-1111-1111-111111111111"
		const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

		_, err := db.Exec(ctx, `INSERT INTO calendars
			(id, user_id, google_id, google_account_id, name, kind, is_primary, is_visible, is_readonly, is_special, created_at, updated_at)
		 VALUES ('cal-1', $1, 'g-cal-1', $2, 'Main', 'google', true, true, false, false, $3, $3)`,
			userID, accountID, now)
		if err != nil {
			t.Fatalf("insert calendar: %v", err)
		}
		_, err = db.Exec(ctx, `INSERT INTO events
			(id, calendar_id, user_id, google_id, title, description, location, start_time, end_time, timezone, is_all_day, created_at, updated_at)
		 VALUES ('evt-1', 'cal-1', $1, 'g-evt-1', 'E1', '', '', $2, $3, 'Asia/Seoul', false, $2, $2)`,
			userID, now, now.Add(time.Hour))
		if err != nil {
			t.Fatalf("insert event: %v", err)
		}

		processor := NewGooglePushProcessor(db, &googleAuthStub{
			updateCalendarEventFn: func(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
				return nil, errors.New("update exploded")
			},
		})
		err = processor.processCalendarPush(ctx, portout.OAuthToken{}, pushOutboxItem{
			AccountID:         accountID,
			UserID:            userID,
			Domain:            "calendar",
			Op:                "update",
			LocalResourceID:   "evt-1",
			ExpectedUpdatedAt: now.Add(time.Minute),
		})
		if err == nil || !strings.Contains(err.Error(), "update exploded") {
			t.Fatalf("expected direct update error, got %v", err)
		}
	})
}

func TestGoogleSyncCoordinatorFinalBranches(t *testing.T) {
	prevGetSetting := coordinatorGetSettingBoolFn
	prevTryLock := coordinatorTryAdvisoryLockFn
	t.Cleanup(func() {
		coordinatorGetSettingBoolFn = prevGetSetting
		coordinatorTryAdvisoryLockFn = prevTryLock
	})

	t.Run("trigger_user_sync_skips_resolve_errors", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		const userID = "11111111-1111-1111-1111-111111111111"
		const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

		_, err := db.Exec(ctx, `INSERT INTO user_google_accounts
			(id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
			accountID, userID, now)
		if err != nil {
			t.Fatalf("insert account: %v", err)
		}

		coordinatorGetSettingBoolFn = func(*googleSyncCoordinator, context.Context, string, string, bool) (bool, error) {
			return false, errors.New("settings exploded")
		}

		stub := &syncerStub{}
		scheduled, err := NewGoogleSyncCoordinator(db, stub).TriggerUserSync(ctx, userID, "both", "manual")
		if err != nil {
			t.Fatalf("TriggerUserSync: %v", err)
		}
		if scheduled != 0 || stub.calls != 0 {
			t.Fatalf("expected skip on resolve error, scheduled=%d calls=%d", scheduled, stub.calls)
		}
	})

	t.Run("hourly_sync_skips_resolve_errors", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		const userID = "11111111-1111-1111-1111-111111111111"
		const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

		_, err := db.Exec(ctx, `INSERT INTO user_google_accounts
			(id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
			accountID, userID, now)
		if err != nil {
			t.Fatalf("insert account: %v", err)
		}

		coordinatorGetSettingBoolFn = func(*googleSyncCoordinator, context.Context, string, string, bool) (bool, error) {
			return false, errors.New("settings exploded")
		}

		stub := &syncerStub{}
		scheduled, err := NewGoogleSyncCoordinator(db, stub).RunHourlySync(ctx)
		if err != nil {
			t.Fatalf("RunHourlySync: %v", err)
		}
		if scheduled != 0 || stub.calls != 0 {
			t.Fatalf("expected skip on resolve error, scheduled=%d calls=%d", scheduled, stub.calls)
		}
	})

	t.Run("sync_account_if_due_returns_lock_scan_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)

		coordinatorTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return false, errors.New("lock scan failed")
		}

		_, err := NewGoogleSyncCoordinator(db, &syncerStub{}).syncAccountIfDue(
			context.Background(),
			"user-1",
			&authdomain.GoogleAccount{ID: "acc-1", UserID: "user-1"},
			portout.GoogleSyncOptions{SyncCalendar: true},
			"manual",
		)
		if err == nil || !strings.Contains(err.Error(), "lock scan failed") {
			t.Fatalf("expected lock scan error, got %v", err)
		}
	})
}

func TestGoogleAccountSyncerFinalBranches(t *testing.T) {
	prevRows := queryGoogleSyncRowsFn
	prevRow := queryGoogleSyncRowFn
	prevExec := execGoogleSyncFn
	prevLock := googleSyncTryAdvisoryLockFn
	t.Cleanup(func() {
		queryGoogleSyncRowsFn = prevRows
		queryGoogleSyncRowFn = prevRow
		execGoogleSyncFn = prevExec
		googleSyncTryAdvisoryLockFn = prevLock
	})

	t.Run("calendar_query_lookup_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{err: errors.New("query calendar id failed")}
		}
		execGoogleSyncFn = prevExec

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main", Kind: "google"}}, nil
			},
		}).syncCalendarsAndEvents(ctx, "user-1", "acc-1", portout.OAuthToken{}, time.Now().UTC())
		if err == nil || !strings.Contains(err.Error(), "query calendar id") {
			t.Fatalf("expected query calendar id error, got %v", err)
		}
	})

	t.Run("calendar_insert_error_after_missing_lookup", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{err: pgx.ErrNoRows}
		}
		execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "INSERT INTO calendars") {
				return pgconn.CommandTag{}, errors.New("insert calendar failed")
			}
			return prevExec(ctx, db, sql, args...)
		}

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main", Kind: "google"}}, nil
			},
		}).syncCalendarsAndEvents(ctx, "user-1", "acc-1", portout.OAuthToken{}, time.Now().UTC())
		if err == nil || !strings.Contains(err.Error(), "insert calendar") {
			t.Fatalf("expected insert calendar error, got %v", err)
		}
	})

	t.Run("calendar_apply_event_error_bubbles", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		start := now.Add(time.Hour)
		end := start.Add(time.Hour)

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{values: []any{"local-cal-1"}}
		}
		execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "UPDATE events") {
				return pgconn.CommandTag{}, errors.New("apply event failed")
			}
			return prevExec(ctx, db, sql, args...)
		}

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
				return []portout.OAuthCalendar{{GoogleID: "g-cal-1", Name: "Main", Kind: "google"}}, nil
			},
			listCalendarEventsFn: func(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				return &portout.OAuthCalendarEventsPage{
					Events: []portout.OAuthCalendarEvent{{
						GoogleID: "evt-1", Title: "Event", StartTime: &start, EndTime: &end, Timezone: "UTC",
					}},
				}, nil
			},
		}).syncCalendarsAndEvents(ctx, "user-1", "acc-1", portout.OAuthToken{}, now)
		if err == nil || !strings.Contains(err.Error(), "apply event failed") {
			t.Fatalf("expected apply event error, got %v", err)
		}
	})

	t.Run("todo_query_lookup_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{err: errors.New("query todo list id failed")}
		}
		execGoogleSyncFn = prevExec

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
			},
		}).syncTaskListsAndTodos(ctx, "user-1", "acc-1", portout.OAuthToken{}, time.Now().UTC())
		if err == nil || !strings.Contains(err.Error(), "query todo list id") {
			t.Fatalf("expected query todo list id error, got %v", err)
		}
	})

	t.Run("todo_list_insert_error_after_missing_lookup", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{err: pgx.ErrNoRows}
		}
		execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "INSERT INTO todo_lists") {
				return pgconn.CommandTag{}, errors.New("insert todo list failed")
			}
			return prevExec(ctx, db, sql, args...)
		}

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
			},
		}).syncTaskListsAndTodos(ctx, "user-1", "acc-1", portout.OAuthToken{}, time.Now().UTC())
		if err == nil || !strings.Contains(err.Error(), "insert todo list") {
			t.Fatalf("expected insert todo list error, got %v", err)
		}
	})

	t.Run("todo_insert_error_for_new_task", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{values: []any{"local-list-1"}}
		}
		execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			switch {
			case strings.Contains(sql, "UPDATE todos"):
				return pgconn.NewCommandTag("UPDATE 0"), nil
			case strings.Contains(sql, "INSERT INTO todos"):
				return pgconn.CommandTag{}, errors.New("insert todo failed")
			default:
				return prevExec(ctx, db, sql, args...)
			}
		}

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
			},
			listTasksFn: func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
				return &portout.OAuthTasksPage{
					Items: []portout.OAuthTask{{GoogleID: "g-task-1", Title: "Todo"}},
				}, nil
			},
		}).syncTaskListsAndTodos(ctx, "user-1", "acc-1", portout.OAuthToken{}, now)
		if err == nil || !strings.Contains(err.Error(), "insert todo") {
			t.Fatalf("expected insert todo error, got %v", err)
		}
	})

	t.Run("todo_full_list_empty_marks_unseen_deleted", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()

		queryGoogleSyncRowFn = func(context.Context, *pgxpool.Pool, string, ...any) googleSyncRow {
			return &fakeGoogleSyncRow{values: []any{"local-list-1"}}
		}
		execGoogleSyncFn = prevExec

		err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
				return []portout.OAuthTaskList{{GoogleID: "g-list-1", Name: "List"}}, nil
			},
			listTasksFn: func(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
				return &portout.OAuthTasksPage{}, nil
			},
		}).syncTaskListsAndTodos(ctx, "user-1", "acc-1", portout.OAuthToken{}, time.Now().UTC())
		if err != nil {
			t.Fatalf("syncTaskListsAndTodos empty list: %v", err)
		}
	})

	t.Run("backfill_lock_scan_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC()

		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"cal-1", "g-cal-1", "acc-1", "at", "rt", (*time.Time)(nil)}}}, nil
		}
		googleSyncTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return false, errors.New("backfill lock scan failed")
		}

		_, err := NewGoogleAccountSyncer(db, &googleAuthStub{}).BackfillEvents(ctx, "user-1", now, now.Add(time.Hour), nil)
		if err == nil || !strings.Contains(err.Error(), "backfill lock scan failed") {
			t.Fatalf("expected backfill lock scan error, got %v", err)
		}
	})

	t.Run("backfill_apply_event_error", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC().Truncate(time.Second)
		start := now.Add(time.Hour)
		end := start.Add(time.Hour)

		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"cal-1", "g-cal-1", "acc-1", "at", "rt", (*time.Time)(nil)}}}, nil
		}
		googleSyncTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return true, nil
		}
		execGoogleSyncFn = func(ctx context.Context, db *pgxpool.Pool, sql string, args ...any) (pgconn.CommandTag, error) {
			if strings.Contains(sql, "UPDATE events") {
				return pgconn.CommandTag{}, errors.New("backfill apply failed")
			}
			return prevExec(ctx, db, sql, args...)
		}

		_, err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarEventsFn: func(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				return &portout.OAuthCalendarEventsPage{
					Events: []portout.OAuthCalendarEvent{{
						GoogleID: "evt-1", Title: "Event", StartTime: &start, EndTime: &end, Timezone: "UTC",
					}},
				}, nil
			},
		}).BackfillEvents(ctx, "user-1", now, now.Add(2*time.Hour), nil)
		if err == nil || !strings.Contains(err.Error(), "backfill apply failed") {
			t.Fatalf("expected backfill apply error, got %v", err)
		}
	})

	t.Run("backfill_pagination_uses_next_page_token", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		ctx := context.Background()
		now := time.Now().UTC()
		call := 0

		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"cal-1", "g-cal-1", "acc-1", "at", "rt", (*time.Time)(nil)}}}, nil
		}
		googleSyncTryAdvisoryLockFn = func(context.Context, *pgxpool.Conn, int64) (bool, error) {
			return true, nil
		}
		execGoogleSyncFn = prevExec

		result, err := NewGoogleAccountSyncer(db, &googleAuthStub{
			listCalendarEventsFn: func(_ context.Context, _ portout.OAuthToken, _ string, pageToken, _ string, _, _ *time.Time) (*portout.OAuthCalendarEventsPage, error) {
				call++
				if call == 1 {
					if pageToken != "" {
						t.Fatalf("expected empty first page token, got %q", pageToken)
					}
					return &portout.OAuthCalendarEventsPage{NextPageToken: "p2"}, nil
				}
				if pageToken != "p2" {
					t.Fatalf("expected second page token p2, got %q", pageToken)
				}
				return &portout.OAuthCalendarEventsPage{}, nil
			},
		}).BackfillEvents(ctx, "user-1", now, now.Add(time.Hour), nil)
		if err != nil {
			t.Fatalf("BackfillEvents pagination: %v", err)
		}
		if result == nil || call != 2 {
			t.Fatalf("expected two page calls, result=%v calls=%d", result, call)
		}
	})
}
