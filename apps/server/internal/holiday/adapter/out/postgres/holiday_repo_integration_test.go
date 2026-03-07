package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"lifebase/internal/holiday/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestHolidayRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	ctx := context.Background()
	repo := NewHolidayRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	holidays := []domain.Holiday{
		{
			Date:      time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Name:      "삼일절",
			Year:      2026,
			Month:     3,
			DateKind:  "01",
			IsHoliday: true,
		},
	}

	if err := repo.ReplaceMonth(ctx, 2026, 3, holidays, now, "00"); err != nil {
		t.Fatalf("replace month: %v", err)
	}

	items, err := repo.ListByDateRange(
		ctx,
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("list by date range: %v", err)
	}
	if len(items) != 1 || items[0].Name != "삼일절" {
		t.Fatalf("unexpected holiday rows: %#v", items)
	}

	state, err := repo.GetMonthSyncState(ctx, 2026, 3)
	if err != nil {
		t.Fatalf("get month sync state: %v", err)
	}
	if state.ResultCode != "00" {
		t.Fatalf("unexpected sync state: %#v", state)
	}
	if _, err := repo.GetMonthSyncState(ctx, 2026, 4); err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows for missing month, got %v", err)
	}
}

func TestHolidayRepoAdvisoryLockAndHelpers(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	ctx := context.Background()
	repo := NewHolidayRepo(db)

	locked, unlock, err := repo.TryAdvisoryMonthLock(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	if !locked || unlock == nil {
		t.Fatalf("expected first lock success, locked=%v unlock=nil?%v", locked, unlock == nil)
	}

	locked2, unlock2, err := repo.TryAdvisoryMonthLock(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("second lock: %v", err)
	}
	if locked2 || unlock2 != nil {
		t.Fatalf("expected second lock contention, locked=%v unlock2=%v", locked2, unlock2)
	}

	unlock()
	locked3, unlock3, err := repo.TryAdvisoryMonthLock(ctx, 2026, 5)
	if err != nil {
		t.Fatalf("third lock after unlock: %v", err)
	}
	if !locked3 || unlock3 == nil {
		t.Fatalf("expected lock after unlock")
	}
	unlock3()

	if err := validateYearMonth(1800, 1); err == nil {
		t.Fatal("expected invalid year")
	}
	if err := validateYearMonth(2026, 0); err == nil {
		t.Fatal("expected invalid month")
	}
	if err := validateYearMonth(2026, 12); err != nil {
		t.Fatalf("valid year/month should pass: %v", err)
	}

	if key := advisoryMonthLockKey(2026, 5); key == 0 {
		t.Fatal("expected non-zero advisory lock key")
	}
}

func TestHolidayRepoErrorPaths(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewHolidayRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	// Duplicate names for same date trigger insert error path in ReplaceMonth loop.
	dup := []domain.Holiday{
		{
			Date:      time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
			Name:      "어린이날",
			Year:      2026,
			Month:     5,
			DateKind:  "01",
			IsHoliday: true,
		},
		{
			Date:      time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC),
			Name:      "어린이날",
			Year:      2026,
			Month:     5,
			DateKind:  "01",
			IsHoliday: true,
		},
	}
	if err := repo.ReplaceMonth(ctx, 2026, 5, dup, now, "00"); err == nil {
		t.Fatal("expected duplicate holiday insert error")
	}

	db.Close()

	if _, err := repo.ListByDateRange(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1)); err == nil {
		t.Fatal("expected ListByDateRange query error on closed pool")
	}
	if _, err := repo.GetMonthSyncState(ctx, 2026, 5); err == nil {
		t.Fatal("expected GetMonthSyncState query error on closed pool")
	}
	if err := repo.ReplaceMonth(ctx, 2026, 6, nil, now, "00"); err == nil {
		t.Fatal("expected ReplaceMonth begin tx error on closed pool")
	}
	if _, _, err := repo.TryAdvisoryMonthLock(ctx, 2026, 6); err == nil {
		t.Fatal("expected TryAdvisoryMonthLock acquire error on closed pool")
	}
}
