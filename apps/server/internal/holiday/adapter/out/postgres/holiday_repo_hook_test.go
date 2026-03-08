package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"lifebase/internal/holiday/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestHolidayRepoScanHookErrorBranch(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	repo := NewHolidayRepo(db)

	if err := repo.ReplaceMonth(ctx, 2026, 6, []domain.Holiday{
		{
			Date:      time.Date(2026, 6, 6, 0, 0, 0, 0, time.UTC),
			Name:      "현충일",
			Year:      2026,
			Month:     6,
			DateKind:  "01",
			IsHoliday: true,
		},
	}, now, "00"); err != nil {
		t.Fatalf("replace month: %v", err)
	}

	prev := scanHolidayRowsFn
	scanHolidayRowsFn = func(pgx.Rows) ([]domain.Holiday, error) {
		return nil, errors.New("scan fail")
	}
	t.Cleanup(func() { scanHolidayRowsFn = prev })

	if _, err := repo.ListByDateRange(ctx, now.AddDate(0, 0, -1), now.AddDate(0, 0, 1)); err == nil {
		t.Fatal("expected ListByDateRange scan hook error")
	}
}
