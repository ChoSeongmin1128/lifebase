package out

import (
	"context"
	"time"

	"lifebase/internal/holiday/domain"
)

type UnlockFunc func()

type HolidayCacheRepository interface {
	ListByDateRange(ctx context.Context, start, end time.Time) ([]domain.Holiday, error)
	GetMonthSyncState(ctx context.Context, year, month int) (*domain.MonthSyncState, error)
	ReplaceMonth(ctx context.Context, year, month int, holidays []domain.Holiday, fetchedAt time.Time, resultCode string) error
	TryAdvisoryMonthLock(ctx context.Context, year, month int) (bool, UnlockFunc, error)
}

type HolidayProvider interface {
	FetchMonth(ctx context.Context, year, month int) ([]domain.Holiday, string, error)
}
