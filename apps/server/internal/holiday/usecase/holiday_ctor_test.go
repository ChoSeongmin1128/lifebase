package usecase

import "testing"

func TestNewHolidayUseCase(t *testing.T) {
	repo := &mockHolidayRepo{}
	provider := &mockHolidayProvider{}

	uc := NewHolidayUseCase(repo, provider)
	impl, ok := uc.(*holidayUseCase)
	if !ok {
		t.Fatalf("expected concrete holiday use case, got %T", uc)
	}
	if impl.repo != repo || impl.provider != provider || impl.nowFn == nil {
		t.Fatalf("constructor did not wire dependencies correctly: %#v", impl)
	}
}
