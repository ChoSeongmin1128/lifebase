package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/holiday/domain"
)

func newUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://lifebase:lifebase@127.0.0.1:1/lifebase?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestHolidayRepoDBErrorBranchesWithoutIntegrationDB(t *testing.T) {
	repo := NewHolidayRepo(newUnreachablePool(t))
	ctx := context.Background()
	now := time.Now().UTC()

	if _, err := repo.ListByDateRange(ctx, now.AddDate(0, 0, -1), now); err == nil {
		t.Fatal("expected ListByDateRange error")
	}
	if _, err := repo.GetMonthSyncState(ctx, 2026, 3); err == nil {
		t.Fatal("expected GetMonthSyncState error")
	}
	if err := repo.ReplaceMonth(ctx, 2026, 3, []domain.Holiday{
		{
			Date:      now,
			Name:      "holiday",
			Year:      2026,
			Month:     3,
			DateKind:  "01",
			IsHoliday: true,
		},
	}, now, "00"); err == nil {
		t.Fatal("expected ReplaceMonth error")
	}
	if _, _, err := repo.TryAdvisoryMonthLock(ctx, 2026, 3); err == nil {
		t.Fatal("expected TryAdvisoryMonthLock error")
	}
}

func TestHolidayRepoHelperBranches(t *testing.T) {
	if key := advisoryMonthLockKey(2026, 3); key == 0 {
		t.Fatal("expected non-zero advisory lock key")
	}

	if err := validateYearMonth(1899, 1); err == nil {
		t.Fatal("expected invalid year")
	}
	if err := validateYearMonth(2201, 1); err == nil {
		t.Fatal("expected invalid year upper bound")
	}
	if err := validateYearMonth(2026, 0); err == nil {
		t.Fatal("expected invalid month lower bound")
	}
	if err := validateYearMonth(2026, 13); err == nil {
		t.Fatal("expected invalid month upper bound")
	}
	if err := validateYearMonth(2026, 12); err != nil {
		t.Fatalf("expected valid year/month: %v", err)
	}
}
