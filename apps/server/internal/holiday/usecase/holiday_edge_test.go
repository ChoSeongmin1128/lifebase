package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lifebase/internal/holiday/domain"
	portin "lifebase/internal/holiday/port/in"
	portout "lifebase/internal/holiday/port/out"
)

type stagedHolidayRepo struct {
	mockHolidayRepo
	states []*domain.MonthSyncState
	calls  int
}

func (m *stagedHolidayRepo) GetMonthSyncState(_ context.Context, year, month int) (*domain.MonthSyncState, error) {
	if m.calls >= len(m.states) {
		return nil, errors.New("not found")
	}
	state := m.states[m.calls]
	m.calls++
	if state == nil {
		return nil, errors.New("not found")
	}
	return state, nil
}

func (m *stagedHolidayRepo) TryAdvisoryMonthLock(ctx context.Context, year, month int) (bool, portout.UnlockFunc, error) {
	return m.mockHolidayRepo.TryAdvisoryMonthLock(ctx, year, month)
}

func TestHolidayUseCaseAdditionalBranches(t *testing.T) {
	t.Run("refresh range propagates refresh month error", func(t *testing.T) {
		uc := &holidayUseCase{
			repo: &mockHolidayRepo{lockErr: errors.New("lock fail")},
			provider: &mockHolidayProvider{},
			nowFn: func() time.Time {
				return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			},
		}
		_, err := uc.RefreshRange(context.Background(), portin.RefreshRangeInput{
			FromYear: intPtr(2026),
			ToYear:   intPtr(2026),
		})
		if err == nil || err.Error() != "lock fail" {
			t.Fatalf("expected lock error, got %v", err)
		}
	})

	t.Run("refresh month lock error and fresh after lock", func(t *testing.T) {
		now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		ucErr := &holidayUseCase{
			repo:     &mockHolidayRepo{lockErr: errors.New("lock fail")},
			provider: &mockHolidayProvider{},
			nowFn:    func() time.Time { return now },
		}
		if _, _, err := ucErr.refreshMonth(context.Background(), 2026, 3, true); err == nil {
			t.Fatal("expected lock error")
		}

		stale := &domain.MonthSyncState{Year: 2026, Month: 3, LastSyncedAt: now.Add(-5 * time.Hour)}
		fresh := &domain.MonthSyncState{Year: 2026, Month: 3, LastSyncedAt: now.Add(-time.Hour)}
		repo := &stagedHolidayRepo{
			mockHolidayRepo: mockHolidayRepo{lockOK: true},
			states:          []*domain.MonthSyncState{stale, fresh},
		}
		ucFresh := &holidayUseCase{
			repo:     repo,
			provider: &mockHolidayProvider{holidays: []domain.Holiday{{Name: "should-not-fetch"}}},
			nowFn:    func() time.Time { return now },
		}
		count, refreshed, err := ucFresh.refreshMonth(context.Background(), 2026, 3, false)
		if err != nil {
			t.Fatalf("unexpected refreshMonth error: %v", err)
		}
		if count != 0 || refreshed {
			t.Fatalf("expected second freshness check to skip, got count=%d refreshed=%v", count, refreshed)
		}
	})
}
