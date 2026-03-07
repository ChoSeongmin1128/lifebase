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

type mockHolidayRepo struct {
	listResult []domain.Holiday
	listErr    error

	stateByMonth map[[2]int]*domain.MonthSyncState
	stateErr     error

	lockOK  bool
	lockErr error
	unlocked bool

	replaceErr error
	replaced   struct {
		year       int
		month      int
		holidays   []domain.Holiday
		fetchedAt  time.Time
		resultCode string
	}
}

func (m *mockHolidayRepo) ListByDateRange(context.Context, time.Time, time.Time) ([]domain.Holiday, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockHolidayRepo) GetMonthSyncState(_ context.Context, year, month int) (*domain.MonthSyncState, error) {
	if m.stateErr != nil {
		return nil, m.stateErr
	}
	if m.stateByMonth == nil {
		return nil, errors.New("not found")
	}
	if st, ok := m.stateByMonth[[2]int{year, month}]; ok {
		return st, nil
	}
	return nil, errors.New("not found")
}

func (m *mockHolidayRepo) ReplaceMonth(_ context.Context, year, month int, holidays []domain.Holiday, fetchedAt time.Time, resultCode string) error {
	if m.replaceErr != nil {
		return m.replaceErr
	}
	m.replaced.year = year
	m.replaced.month = month
	m.replaced.holidays = holidays
	m.replaced.fetchedAt = fetchedAt
	m.replaced.resultCode = resultCode
	return nil
}

func (m *mockHolidayRepo) TryAdvisoryMonthLock(context.Context, int, int) (bool, portout.UnlockFunc, error) {
	if m.lockErr != nil {
		return false, nil, m.lockErr
	}
	unlock := func() { m.unlocked = true }
	return m.lockOK, unlock, nil
}

type mockHolidayProvider struct {
	holidays   []domain.Holiday
	resultCode string
	err        error
}

func (m *mockHolidayProvider) FetchMonth(context.Context, int, int) ([]domain.Holiday, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return append([]domain.Holiday(nil), m.holidays...), m.resultCode, nil
}

func intPtr(v int) *int { return &v }

func TestListRangeValidation(t *testing.T) {
	uc := &holidayUseCase{}
	start := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if _, err := uc.ListRange(context.Background(), start, end); err == nil {
		t.Fatal("expected invalid range error")
	}
}

func TestListRangeReturnsCachedDataWhenRefreshFails(t *testing.T) {
	repo := &mockHolidayRepo{
		listResult: []domain.Holiday{{Name: "x"}},
		lockOK:     true,
		stateErr:   errors.New("no cache"),
	}
	provider := &mockHolidayProvider{err: errors.New("provider failed")}
	uc := &holidayUseCase{
		repo:     repo,
		provider: provider,
		nowFn: func() time.Time {
			return time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
		},
	}

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	items, err := uc.ListRange(context.Background(), start, end)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 || items[0].Name != "x" {
		t.Fatalf("unexpected holidays: %#v", items)
	}
}

func TestListRangeRepoError(t *testing.T) {
	repo := &mockHolidayRepo{
		listErr: errors.New("db failed"),
		lockOK:  false,
	}
	uc := &holidayUseCase{
		repo:     repo,
		provider: &mockHolidayProvider{},
		nowFn:    time.Now,
	}
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	if _, err := uc.ListRange(context.Background(), start, end); err == nil {
		t.Fatal("expected list error")
	}
}

func TestRefreshRangeValidation(t *testing.T) {
	uc := &holidayUseCase{
		repo:     &mockHolidayRepo{},
		provider: &mockHolidayProvider{},
		nowFn:    time.Now,
	}
	_, err := uc.RefreshRange(context.Background(), portin.RefreshRangeInput{
		FromYear: intPtr(2027),
		ToYear:   intPtr(2026),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestRefreshRangeSuccess(t *testing.T) {
	repo := &mockHolidayRepo{
		lockOK: true,
		stateByMonth: map[[2]int]*domain.MonthSyncState{},
	}
	provider := &mockHolidayProvider{
		holidays: []domain.Holiday{{Name: "new-year", Year: 2026, Month: 1}},
	}
	fixedNow := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	uc := &holidayUseCase{
		repo:     repo,
		provider: provider,
		nowFn:    func() time.Time { return fixedNow },
	}

	res, err := uc.RefreshRange(context.Background(), portin.RefreshRangeInput{
		FromYear: intPtr(2026),
		ToYear:   intPtr(2026),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.MonthsTotal != 12 {
		t.Fatalf("expected 12 months total, got %d", res.MonthsTotal)
	}
	if res.MonthsRefreshed != 12 {
		t.Fatalf("expected 12 refreshed, got %d", res.MonthsRefreshed)
	}
	if res.ItemsUpserted != 12 {
		t.Fatalf("expected 12 upserted items, got %d", res.ItemsUpserted)
	}
	if !repo.unlocked {
		t.Fatal("expected advisory lock to unlock")
	}
}

func TestRefreshMonthValidation(t *testing.T) {
	uc := &holidayUseCase{}
	if _, _, err := uc.refreshMonth(context.Background(), 2026, 0, false); err == nil {
		t.Fatal("expected invalid month error")
	}
	if _, _, err := uc.refreshMonth(context.Background(), 1800, 1, false); err == nil {
		t.Fatal("expected invalid year error")
	}
}

func TestRefreshMonthSkipsWhenFreshAndNotForced(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := &mockHolidayRepo{
		stateByMonth: map[[2]int]*domain.MonthSyncState{
			{2026, 3}: {Year: 2026, Month: 3, LastSyncedAt: now.Add(-time.Hour)},
		},
	}
	uc := &holidayUseCase{
		repo:     repo,
		provider: &mockHolidayProvider{},
		nowFn:    func() time.Time { return now },
	}

	count, refreshed, err := uc.refreshMonth(context.Background(), 2026, 3, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 || refreshed {
		t.Fatalf("expected skipped refresh, got count=%d refreshed=%v", count, refreshed)
	}
}

func TestRefreshMonthNoLock(t *testing.T) {
	uc := &holidayUseCase{
		repo: &mockHolidayRepo{
			lockOK: false,
		},
		provider: &mockHolidayProvider{},
		nowFn:    time.Now,
	}
	count, refreshed, err := uc.refreshMonth(context.Background(), 2026, 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 || refreshed {
		t.Fatalf("expected no refresh, got count=%d refreshed=%v", count, refreshed)
	}
}

func TestRefreshMonthProviderAndReplaceErrors(t *testing.T) {
	ucProviderErr := &holidayUseCase{
		repo: &mockHolidayRepo{lockOK: true},
		provider: &mockHolidayProvider{
			err: errors.New("provider failed"),
		},
		nowFn: time.Now,
	}
	if _, _, err := ucProviderErr.refreshMonth(context.Background(), 2026, 3, true); err == nil {
		t.Fatal("expected provider error")
	}

	ucReplaceErr := &holidayUseCase{
		repo: &mockHolidayRepo{
			lockOK:     true,
			replaceErr: errors.New("replace failed"),
		},
		provider: &mockHolidayProvider{
			holidays: []domain.Holiday{{Name: "h"}},
		},
		nowFn: time.Now,
	}
	if _, _, err := ucReplaceErr.refreshMonth(context.Background(), 2026, 3, true); err == nil {
		t.Fatal("expected replace error")
	}
}

func TestRefreshMonthDefaultsResultCodeAndFetchedAt(t *testing.T) {
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	repo := &mockHolidayRepo{lockOK: true}
	uc := &holidayUseCase{
		repo:     repo,
		provider: &mockHolidayProvider{holidays: []domain.Holiday{{Name: "h"}}},
		nowFn:    func() time.Time { return now },
	}

	count, refreshed, err := uc.refreshMonth(context.Background(), 2026, 3, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 || !refreshed {
		t.Fatalf("expected refreshed with one item, got count=%d refreshed=%v", count, refreshed)
	}
	if repo.replaced.resultCode != "00" {
		t.Fatalf("expected default result code 00, got %q", repo.replaced.resultCode)
	}
	if repo.replaced.holidays[0].FetchedAt != now {
		t.Fatalf("expected fetched_at set to now, got %v", repo.replaced.holidays[0].FetchedAt)
	}
}

func TestIsStale(t *testing.T) {
	now := time.Now()
	if !isStale(time.Time{}, now) {
		t.Fatal("zero synced_at should be stale")
	}
	if isStale(now.Add(-time.Hour), now) {
		t.Fatal("recent sync should not be stale")
	}
	if !isStale(now.Add(-4*time.Hour), now) {
		t.Fatal("old sync should be stale")
	}
}
