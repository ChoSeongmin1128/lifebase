package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	authdomain "lifebase/internal/auth/domain"
	portout "lifebase/internal/auth/port/out"
)

func openBrokenPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://127.0.0.1:1/lifebase?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestGooglePushProcessorNoDBBranches(t *testing.T) {
	pool := openBrokenPool(t)
	processor := NewGooglePushProcessor(pool, &googleAuthStub{})

	if _, err := processor.ProcessPending(context.Background(), 5); err == nil {
		t.Fatal("expected ProcessPending query error on broken pool")
	}

	err := processor.processOne(context.Background(), pushOutboxItem{
		ID:                "out-1",
		AccountID:         "acc-1",
		UserID:            "user-1",
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-1",
		ExpectedUpdatedAt: time.Now().UTC(),
	})
	if err == nil {
		t.Fatal("expected processOne acquire error on broken pool")
	}

	if err := processor.processCalendarPush(context.Background(), portout.OAuthToken{}, pushOutboxItem{
		AccountID:         "acc-1",
		UserID:            "user-1",
		Domain:            "calendar",
		Op:                "update",
		LocalResourceID:   "evt-1",
		ExpectedUpdatedAt: time.Now().UTC(),
	}); err == nil {
		t.Fatal("expected processCalendarPush load error on broken pool")
	}

	if err := processor.processTodoPush(context.Background(), portout.OAuthToken{}, pushOutboxItem{
		AccountID:         "acc-1",
		UserID:            "user-1",
		Domain:            "todo",
		Op:                "update",
		LocalResourceID:   "todo-1",
		ExpectedUpdatedAt: time.Now().UTC(),
	}); err == nil {
		t.Fatal("expected processTodoPush load error on broken pool")
	}
}

func TestGoogleSyncCoordinatorNoDBBranches(t *testing.T) {
	pool := openBrokenPool(t)
	coordinator := NewGoogleSyncCoordinator(pool, &syncerStub{})

	if _, err := coordinator.TriggerUserSync(context.Background(), "", "both", "manual"); err == nil {
		t.Fatal("expected required user id error")
	}
	if _, err := coordinator.TriggerUserSync(context.Background(), "user-1", "both", "manual"); err == nil {
		t.Fatal("expected TriggerUserSync query error on broken pool")
	}
	if _, err := coordinator.RunHourlySync(context.Background()); err == nil {
		t.Fatal("expected RunHourlySync query error on broken pool")
	}

	performed, err := coordinator.syncAccountIfDue(context.Background(), "user-1", nil, portout.GoogleSyncOptions{}, "manual")
	if err != nil || performed {
		t.Fatalf("nil account should no-op: performed=%v err=%v", performed, err)
	}
	performed, err = coordinator.syncAccountIfDue(context.Background(), "user-1", &authdomain.GoogleAccount{ID: "acc-1"}, portout.GoogleSyncOptions{}, "manual")
	if err != nil || performed {
		t.Fatalf("disabled options should no-op: performed=%v err=%v", performed, err)
	}
	if _, err := coordinator.syncAccountIfDue(context.Background(), "user-1", &authdomain.GoogleAccount{ID: "acc-1"}, portout.GoogleSyncOptions{SyncCalendar: true}, "manual"); err == nil {
		t.Fatal("expected syncAccountIfDue acquire error on broken pool")
	}

	prevGet := coordinatorGetSettingBoolFn
	t.Cleanup(func() { coordinatorGetSettingBoolFn = prevGet })
	coordinatorGetSettingBoolFn = func(*googleSyncCoordinator, context.Context, string, string, bool) (bool, error) {
		return true, nil
	}
	options, enabled, err := coordinator.resolveSyncOptions(context.Background(), "user-1", "acc-1", "unknown")
	if err != nil {
		t.Fatalf("resolveSyncOptions unknown area err: %v", err)
	}
	if enabled || !options.SyncCalendar || !options.SyncTodo {
		t.Fatalf("unknown area should disable execution but preserve options, got enabled=%v options=%+v", enabled, options)
	}
}

func TestGoogleSyncerNoDBBranches(t *testing.T) {
	pool := openBrokenPool(t)
	syncer := NewGoogleAccountSyncer(pool, &googleAuthStub{
		listCalendarsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
			return nil, nil
		},
		listTaskListsFn: func(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
			return nil, nil
		},
	})

	if err := syncer.SyncAccount(context.Background(), "user-1", nil, portout.GoogleSyncOptions{SyncCalendar: true}); err == nil {
		t.Fatal("expected nil account error")
	}

	account := &authdomain.GoogleAccount{ID: "acc-1", UserID: "user-1", AccessToken: "at", RefreshToken: "rt"}
	if err := syncer.SyncAccount(context.Background(), "user-1", account, portout.GoogleSyncOptions{SyncCalendar: true}); err != nil {
		t.Fatalf("sync calendar empty should pass: %v", err)
	}
	if err := syncer.SyncAccount(context.Background(), "user-1", account, portout.GoogleSyncOptions{SyncTodo: true}); err != nil {
		t.Fatalf("sync todo empty should pass: %v", err)
	}

	_, err := syncer.BackfillEvents(context.Background(), "user-1", time.Now().UTC(), time.Now().UTC(), nil)
	if err == nil || !strings.Contains(err.Error(), "invalid backfill range") {
		t.Fatalf("expected invalid backfill range error, got %v", err)
	}
	if _, err := syncer.BackfillEvents(context.Background(), "user-1", time.Now().UTC(), time.Now().UTC().Add(time.Hour), nil); err == nil {
		t.Fatal("expected backfill query error on broken pool")
	}

	if _, err := syncer.loadLocalTodoIDsByGoogleID(context.Background(), "user-1", "list-1"); err == nil {
		t.Fatal("expected loadLocalTodoIDsByGoogleID query error on broken pool")
	}
	if _, err := syncer.loadPendingDeleteTodoIDs(context.Background(), "user-1", "list-1"); err == nil {
		t.Fatal("expected loadPendingDeleteTodoIDs query error on broken pool")
	}

	now := time.Now().UTC()
	cutoff := syncer.resolveTodoDoneRetentionCutoff(context.Background(), "user-1", now)
	if cutoff == nil {
		t.Fatal("default retention cutoff should not be nil")
	}
	if err := syncer.expandCalendarCoverage(context.Background(), "user-1", "cal-1", now.Add(-time.Hour), now, now); err != nil {
		t.Fatalf("expandCalendarCoverage should swallow exec errors: %v", err)
	}
}
