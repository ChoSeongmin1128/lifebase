package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"lifebase/internal/holiday/domain"
	portin "lifebase/internal/holiday/port/in"
	portout "lifebase/internal/holiday/port/out"
)

const (
	cacheFreshTTL   = 3 * time.Hour
	defaultYearSpan = 2
)

type holidayUseCase struct {
	repo     portout.HolidayCacheRepository
	provider portout.HolidayProvider
	nowFn    func() time.Time
}

func NewHolidayUseCase(repo portout.HolidayCacheRepository, provider portout.HolidayProvider) portin.HolidayUseCase {
	return &holidayUseCase{
		repo:     repo,
		provider: provider,
		nowFn:    time.Now,
	}
}

func (uc *holidayUseCase) ListRange(ctx context.Context, start, end time.Time) ([]domain.Holiday, error) {
	if !end.After(start) {
		return nil, fmt.Errorf("end must be after start")
	}

	for _, month := range domain.MonthKeysBetween(start, end) {
		if err := uc.ensureMonthFresh(ctx, month.Year, month.Month, false); err != nil {
			// 외부 API 실패 시 캐시 우선. 캐시가 없어도 빈 결과로 degrade 한다.
			if _, cacheErr := uc.repo.GetMonthSyncState(ctx, month.Year, month.Month); cacheErr != nil {
				slog.Warn("holiday month refresh skipped without cache",
					"year", month.Year,
					"month", month.Month,
					"error", err,
				)
			}
		}
	}

	holidays, err := uc.repo.ListByDateRange(ctx, start, end)
	if err != nil {
		return nil, err
	}
	return holidays, nil
}

func (uc *holidayUseCase) RefreshRange(ctx context.Context, input portin.RefreshRangeInput) (*portin.RefreshRangeResult, error) {
	now := uc.nowFn().In(time.Local)
	fromYear := now.Year() - defaultYearSpan
	toYear := now.Year() + defaultYearSpan
	if input.FromYear != nil {
		fromYear = *input.FromYear
	}
	if input.ToYear != nil {
		toYear = *input.ToYear
	}
	if fromYear > toYear {
		return nil, fmt.Errorf("from_year must be <= to_year")
	}

	months := domain.MonthKeysInYearRange(fromYear, toYear)
	result := &portin.RefreshRangeResult{
		MonthsTotal: len(months),
		RefreshedAt: uc.nowFn(),
	}

	for _, month := range months {
		count, refreshed, err := uc.refreshMonth(ctx, month.Year, month.Month, true)
		if err != nil {
			return nil, err
		}
		if refreshed {
			result.MonthsRefreshed += 1
			result.ItemsUpserted += count
		}
	}

	return result, nil
}

func (uc *holidayUseCase) ensureMonthFresh(ctx context.Context, year, month int, force bool) error {
	_, _, err := uc.refreshMonth(ctx, year, month, force)
	return err
}

func (uc *holidayUseCase) refreshMonth(ctx context.Context, year, month int, force bool) (int, bool, error) {
	if month < 1 || month > 12 {
		return 0, false, fmt.Errorf("invalid month")
	}
	if year < 1900 || year > 2200 {
		return 0, false, fmt.Errorf("invalid year")
	}

	if !force {
		state, err := uc.repo.GetMonthSyncState(ctx, year, month)
		if err == nil && !isStale(state.LastSyncedAt, uc.nowFn()) {
			return 0, false, nil
		}
	}

	locked, unlock, err := uc.repo.TryAdvisoryMonthLock(ctx, year, month)
	if err != nil {
		return 0, false, err
	}
	if !locked {
		return 0, false, nil
	}
	defer unlock()

	if !force {
		state, err := uc.repo.GetMonthSyncState(ctx, year, month)
		if err == nil && !isStale(state.LastSyncedAt, uc.nowFn()) {
			return 0, false, nil
		}
	}

	holidays, resultCode, err := uc.provider.FetchMonth(ctx, year, month)
	if err != nil {
		return 0, false, err
	}
	if resultCode == "" {
		resultCode = "00"
	}

	fetchedAt := uc.nowFn()
	for i := range holidays {
		holidays[i].FetchedAt = fetchedAt
	}

	if err := uc.repo.ReplaceMonth(ctx, year, month, holidays, fetchedAt, resultCode); err != nil {
		return 0, false, err
	}
	return len(holidays), true, nil
}

func isStale(lastSyncedAt time.Time, now time.Time) bool {
	if lastSyncedAt.IsZero() {
		return true
	}
	return now.Sub(lastSyncedAt) >= cacheFreshTTL
}

var ErrProviderNotConfigured = errors.New("holiday provider is not configured")
