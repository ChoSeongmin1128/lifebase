package postgres

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/testutil/dbtest"
)

type fakePushRows struct {
	next []bool
	err  error
	data [][]any
	i    int
	scan error
}

func (r *fakePushRows) Close()                                       {}
func (r *fakePushRows) Err() error                                   { return r.err }
func (r *fakePushRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (r *fakePushRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (r *fakePushRows) Values() ([]any, error)                       { return nil, nil }
func (r *fakePushRows) RawValues() [][]byte                          { return nil }
func (r *fakePushRows) Conn() *pgx.Conn                              { return nil }
func (r *fakePushRows) Next() bool {
	if len(r.data) > 0 {
		if r.i >= len(r.data) {
			return false
		}
		r.i++
		return true
	}
	if len(r.next) == 0 {
		return false
	}
	v := r.next[0]
	r.next = r.next[1:]
	return v
}
func (r *fakePushRows) Scan(dest ...any) error {
	if r.scan != nil {
		return r.scan
	}
	if len(r.data) == 0 {
		return errors.New("scan fail")
	}
	row := r.data[r.i-1]
	for i := range dest {
		dv := reflect.ValueOf(dest[i])
		if i >= len(row) || row[i] == nil {
			dv.Elem().Set(reflect.Zero(dv.Elem().Type()))
			continue
		}
		dv.Elem().Set(reflect.ValueOf(row[i]))
	}
	return nil
}

func TestGooglePushProcessorClaimPendingRowBranches(t *testing.T) {
	prev := queryGooglePushRowsFn
	t.Cleanup(func() { queryGooglePushRowsFn = prev })

	processor := NewGooglePushProcessor(nil, &googleAuthStub{})

	t.Run("scan_error", func(t *testing.T) {
		queryGooglePushRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{true, false}}, nil
		}
		if _, err := processor.claimPending(context.Background(), 1); err == nil {
			t.Fatal("expected claimPending scan error")
		}
	})

	t.Run("rows_err", func(t *testing.T) {
		queryGooglePushRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{false}, err: errors.New("rows fail")}, nil
		}
		if _, err := processor.claimPending(context.Background(), 1); err == nil {
			t.Fatal("expected claimPending rows error")
		}
	})
}

func TestGoogleSyncCoordinatorHookedErrorBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', NOW(), 'scope', 'active', true, NOW(), NOW(), NOW())`,
		accountID, userID,
	)
	if err != nil {
		t.Fatalf("insert account: %v", err)
	}

	coordinator := NewGoogleSyncCoordinator(db, &syncerStub{})
	account := &authdomain.GoogleAccount{ID: accountID, UserID: userID}

	prevGetSetting := coordinatorGetSettingBoolFn
	prevLastSync := coordinatorLastSyncAtFn
	prevTouch := coordinatorTouchSyncReasonFn
	prevUpdateSuccess := coordinatorUpdateSyncSuccessFn
	t.Cleanup(func() {
		coordinatorGetSettingBoolFn = prevGetSetting
		coordinatorLastSyncAtFn = prevLastSync
		coordinatorTouchSyncReasonFn = prevTouch
		coordinatorUpdateSyncSuccessFn = prevUpdateSuccess
	})

	t.Run("resolve_calendar_setting_error", func(t *testing.T) {
		coordinatorGetSettingBoolFn = func(*googleSyncCoordinator, context.Context, string, string, bool) (bool, error) {
			return false, errors.New("calendar setting fail")
		}
		if _, _, err := coordinator.resolveSyncOptions(ctx, userID, accountID, "both"); err == nil {
			t.Fatal("expected resolveSyncOptions calendar setting error")
		}
		coordinatorGetSettingBoolFn = prevGetSetting
	})

	t.Run("resolve_todo_setting_error", func(t *testing.T) {
		call := 0
		coordinatorGetSettingBoolFn = func(*googleSyncCoordinator, context.Context, string, string, bool) (bool, error) {
			call++
			if call == 1 {
				return true, nil
			}
			return false, errors.New("todo setting fail")
		}
		if _, _, err := coordinator.resolveSyncOptions(ctx, userID, accountID, "both"); err == nil {
			t.Fatal("expected resolveSyncOptions todo setting error")
		}
		coordinatorGetSettingBoolFn = prevGetSetting
	})

	t.Run("last_sync_error", func(t *testing.T) {
		coordinatorLastSyncAtFn = func(*googleSyncCoordinator, context.Context, string, string) (time.Time, error) {
			return time.Time{}, errors.New("last sync fail")
		}
		if _, err := coordinator.syncAccountIfDue(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil {
			t.Fatal("expected lastSyncAt error")
		}
		coordinatorLastSyncAtFn = prevLastSync
	})

	t.Run("touch_sync_error", func(t *testing.T) {
		coordinatorTouchSyncReasonFn = func(*googleSyncCoordinator, context.Context, string, string, string, time.Time) error {
			return errors.New("touch fail")
		}
		if _, err := coordinator.syncAccountIfDue(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil {
			t.Fatal("expected touchSyncReason error")
		}
		coordinatorTouchSyncReasonFn = prevTouch
	})

	t.Run("update_success_error", func(t *testing.T) {
		coordinatorUpdateSyncSuccessFn = func(*googleSyncCoordinator, context.Context, string, time.Time) error {
			return errors.New("update success fail")
		}
		if performed, err := coordinator.syncAccountIfDue(ctx, userID, account, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil || !performed {
			t.Fatalf("expected updateSyncSuccess error after performed sync, performed=%v err=%v", performed, err)
		}
		coordinatorUpdateSyncSuccessFn = prevUpdateSuccess
	})
}

func TestGoogleAccountSyncerBackfillRowsAndAcquireBranches(t *testing.T) {
	prev := queryGoogleSyncRowsFn
	t.Cleanup(func() { queryGoogleSyncRowsFn = prev })

	syncer := NewGoogleAccountSyncer(nil, &googleAuthStub{})
	ctx := context.Background()
	now := time.Now().UTC()

	t.Run("rows_scan_error", func(t *testing.T) {
		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"cal", "gcal", "acc", "at", "rt", (*time.Time)(nil)}}, scan: errors.New("scan fail")}, nil
		}
		if _, err := syncer.BackfillEvents(ctx, "user", now, now.Add(time.Hour), nil); err == nil {
			t.Fatal("expected backfill rows scan error")
		}
	})

	t.Run("rows_err", func(t *testing.T) {
		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{false}, err: errors.New("rows fail")}, nil
		}
		if _, err := syncer.BackfillEvents(ctx, "user", now, now.Add(time.Hour), nil); err == nil {
			t.Fatal("expected backfill rows error")
		}
	})

	t.Run("acquire_error_after_rows", func(t *testing.T) {
		db := dbtest.Open(t)
		dbtest.Reset(t, db)
		db.Close()

		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"cal", "gcal", "acc", "at", "rt", (*time.Time)(nil)}}}, nil
		}

		if _, err := NewGoogleAccountSyncer(db, &googleAuthStub{}).BackfillEvents(ctx, "user", now, now.Add(time.Hour), nil); err == nil {
			t.Fatal("expected acquire error after successful row scan")
		}
	})
}

func TestGoogleAccountSyncerLoadPendingDeleteTodoIDsBranches(t *testing.T) {
	prev := queryGoogleSyncRowsFn
	t.Cleanup(func() { queryGoogleSyncRowsFn = prev })

	syncer := NewGoogleAccountSyncer(nil, &googleAuthStub{})
	ctx := context.Background()

	t.Run("scan_error", func(t *testing.T) {
		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{true}, scan: errors.New("scan fail")}, nil
		}
		if _, err := syncer.loadPendingDeleteTodoIDs(ctx, "user", "list"); err == nil {
			t.Fatal("expected loadPendingDeleteTodoIDs scan error")
		}
	})

	t.Run("rows_error", func(t *testing.T) {
		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{next: []bool{false}, err: errors.New("rows fail")}, nil
		}
		if _, err := syncer.loadPendingDeleteTodoIDs(ctx, "user", "list"); err == nil {
			t.Fatal("expected loadPendingDeleteTodoIDs rows error")
		}
	})

	t.Run("success", func(t *testing.T) {
		queryGoogleSyncRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
			return &fakePushRows{data: [][]any{{"todo-1"}, {"todo-2"}}}, nil
		}
		ids, err := syncer.loadPendingDeleteTodoIDs(ctx, "user", "list")
		if err != nil {
			t.Fatalf("loadPendingDeleteTodoIDs: %v", err)
		}
		if _, ok := ids["todo-1"]; !ok {
			t.Fatal("expected todo-1 in pending delete ids")
		}
		if _, ok := ids["todo-2"]; !ok {
			t.Fatal("expected todo-2 in pending delete ids")
		}
		if len(ids) != 2 {
			t.Fatalf("expected 2 pending delete ids, got %d", len(ids))
		}
	})
}

func TestAuthScannerHelpers(t *testing.T) {
	now := time.Now().UTC()

	t.Run("scan_google_accounts_rows_scan_error", func(t *testing.T) {
		_, err := scanGoogleAccountsRows(&fakePushRows{
			data: [][]any{{"acc", "user", "u1@gmail.com", "gid", "at", "rt", &now, "scope", "active", true, now, now, now}},
			scan: errors.New("scan fail"),
		})
		if err == nil {
			t.Fatal("expected google accounts scan error")
		}
	})

	t.Run("scan_google_accounts_rows_err", func(t *testing.T) {
		_, err := scanGoogleAccountsRows(&fakePushRows{next: []bool{false}, err: errors.New("rows fail")})
		if err == nil {
			t.Fatal("expected google accounts rows.Err")
		}
	})

	t.Run("scan_user_rows_scan_error", func(t *testing.T) {
		_, err := scanUserRows(&fakePushRows{
			data: [][]any{{"u1", "u1@example.com", "U1", "pic", int64(1), int64(2), now, now}},
			scan: errors.New("scan fail"),
		})
		if err == nil {
			t.Fatal("expected user rows scan error")
		}
	})

	t.Run("scan_user_rows_err", func(t *testing.T) {
		_, err := scanUserRows(&fakePushRows{next: []bool{false}, err: errors.New("rows fail")})
		if err == nil {
			t.Fatal("expected user rows.Err")
		}
	})
}

func TestGoogleAccountSyncerApplyOAuthEventInsertErrorBranch(t *testing.T) {
	prev := execGoogleSyncFn
	t.Cleanup(func() { execGoogleSyncFn = prev })

	call := 0
	execGoogleSyncFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgconn.CommandTag, error) {
		call++
		if call == 1 {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		}
		return pgconn.CommandTag{}, errors.New("insert fail")
	}

	now := time.Now().UTC()
	start := now.Add(time.Hour)
	end := start.Add(time.Hour)
	_, _, err := NewGoogleAccountSyncer(nil, &googleAuthStub{}).applyOAuthEvent(
		context.Background(),
		"user",
		"calendar",
		portout.OAuthCalendarEvent{
			GoogleID:    "event-1",
			Title:       "title",
			StartTime:   &start,
			EndTime:     &end,
			Timezone:    "UTC",
			IsAllDay:    false,
			Description: "desc",
		},
		now,
	)
	if err == nil {
		t.Fatal("expected insert event error")
	}
}
