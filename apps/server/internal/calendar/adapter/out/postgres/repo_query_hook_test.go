package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestCalendarRepoQueryHooksWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	repo := NewCalendarRepo(pool)

	prevQuery := queryCalendarRowsFn
	t.Cleanup(func() { queryCalendarRowsFn = prevQuery })

	queryCalendarRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakeCalendarRows{}, nil
	}
	items, err := repo.ListByUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("ListByUser success hook: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected empty list, got %d", len(items))
	}

	queryCalendarRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query fail")
	}
	if _, err := repo.ListByUser(context.Background(), "u1"); err == nil {
		t.Fatal("expected ListByUser query error")
	}
}

func TestDaySummaryRepoQueryHooksWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	holidayRepo := NewDaySummaryHolidayRepo(pool)
	todoRepo := NewDaySummaryTodoRepo(pool)

	prevQuery := queryDaySummaryRowsFn
	t.Cleanup(func() { queryDaySummaryRowsFn = prevQuery })

	queryDaySummaryRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakeCalendarRows{}, nil
	}
	if _, err := holidayRepo.ListByDateRange(context.Background(), time.Now(), time.Now()); err != nil {
		t.Fatalf("holiday ListByDateRange success hook: %v", err)
	}
	if _, err := todoRepo.ListByDueDate(context.Background(), "u1", "2026-03-09", true); err != nil {
		t.Fatalf("todo ListByDueDate success hook: %v", err)
	}

	queryDaySummaryRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query fail")
	}
	if _, err := holidayRepo.ListByDateRange(context.Background(), time.Now(), time.Now()); err == nil {
		t.Fatal("expected holiday ListByDateRange query error")
	}
	if _, err := todoRepo.ListByDueDate(context.Background(), "u1", "2026-03-09", false); err == nil {
		t.Fatal("expected todo ListByDueDate query error")
	}
}

func TestEventAndReminderRepoQueryHooksWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	eventRepo := NewEventRepo(pool)
	reminderRepo := NewReminderRepo(pool)

	prevQuery := queryEventRowsFn
	t.Cleanup(func() { queryEventRowsFn = prevQuery })

	queryEventRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakeCalendarRows{}, nil
	}
	if _, err := eventRepo.ListByRange(context.Background(), "u1", []string{"c1", "c2"}, time.Now().Add(-time.Hour).Format(time.RFC3339), time.Now().Format(time.RFC3339)); err != nil {
		t.Fatalf("event ListByRange success hook: %v", err)
	}
	if _, err := reminderRepo.ListByEvent(context.Background(), "e1"); err != nil {
		t.Fatalf("reminder ListByEvent success hook: %v", err)
	}

	queryEventRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query fail")
	}
	if _, err := eventRepo.ListByRange(context.Background(), "u1", nil, time.Now().Add(-time.Hour).Format(time.RFC3339), time.Now().Format(time.RFC3339)); err == nil {
		t.Fatal("expected event ListByRange query error")
	}
	if _, err := reminderRepo.ListByEvent(context.Background(), "e1"); err == nil {
		t.Fatal("expected reminder ListByEvent query error")
	}
}

func TestEventPushOutboxRepoHookBranchesWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	repo := NewEventPushOutboxRepo(pool)

	prevQuery := queryCalendarOutboxAccountFn
	prevInsert := insertCalendarOutboxFn
	t.Cleanup(func() {
		queryCalendarOutboxAccountFn = prevQuery
		insertCalendarOutboxFn = prevInsert
	})

	queryCalendarOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return nil, pgx.ErrNoRows
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "e1", time.Now()); err != nil {
		t.Fatalf("ErrNoRows should be ignored: %v", err)
	}

	queryCalendarOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return nil, errors.New("query fail")
	}
	if err := repo.EnqueueUpdate(context.Background(), "u1", "e1", time.Now()); err == nil {
		t.Fatal("expected outbox account query error")
	}

	empty := ""
	queryCalendarOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return &empty, nil
	}
	if err := repo.EnqueueDelete(context.Background(), "u1", "e1", time.Now()); err != nil {
		t.Fatalf("empty account id should be ignored: %v", err)
	}

	accountID := "22222222-2222-2222-2222-222222222222"
	queryCalendarOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return &accountID, nil
	}
	insertCalendarOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return errors.New("insert fail")
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "e1", time.Now()); err == nil {
		t.Fatal("expected outbox insert error")
	}

	insertCalendarOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return nil
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "e1", time.Now()); err != nil {
		t.Fatalf("expected outbox enqueue success: %v", err)
	}
}
