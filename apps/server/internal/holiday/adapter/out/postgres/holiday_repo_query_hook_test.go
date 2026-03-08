package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func newNoDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://localhost:1/lifebase_test?sslmode=disable")
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestHolidayRepoQueryHookBranchesWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	repo := NewHolidayRepo(pool)

	prevQuery := queryHolidayRowsFn
	t.Cleanup(func() { queryHolidayRowsFn = prevQuery })

	queryHolidayRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakeHolidayRows{}, nil
	}
	if _, err := repo.ListByDateRange(context.Background(), time.Now(), time.Now()); err != nil {
		t.Fatalf("ListByDateRange success hook: %v", err)
	}

	queryHolidayRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query fail")
	}
	if _, err := repo.ListByDateRange(context.Background(), time.Now(), time.Now()); err == nil {
		t.Fatal("expected ListByDateRange query error")
	}
}

func TestHolidayRepoAcquireHookBranchWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	repo := NewHolidayRepo(pool)

	prevAcquire := acquireHolidayConnFn
	t.Cleanup(func() { acquireHolidayConnFn = prevAcquire })

	acquireHolidayConnFn = func(context.Context, *pgxpool.Pool) (*pgxpool.Conn, error) {
		return nil, errors.New("acquire fail")
	}
	if _, _, err := repo.TryAdvisoryMonthLock(context.Background(), 2026, 3); err == nil {
		t.Fatal("expected TryAdvisoryMonthLock acquire error")
	}
}
