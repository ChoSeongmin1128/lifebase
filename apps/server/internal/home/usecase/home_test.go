package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lifebase/internal/home/domain"
	portin "lifebase/internal/home/port/in"
)

type mockHomeRepo struct {
	events       []domain.EventSummary
	eventTotal   int
	eventErr     error
	overdue      []domain.TodoSummary
	overdueCount int
	overdueErr   error
	today        []domain.TodoSummary
	todayCount   int
	todayErr     error
	recent       []domain.RecentFileSummary
	recentTotal  int
	recentErr    error
	storage      domain.StorageSummary
	storageErr   error
	typeUsage    []domain.StorageTypeUsage
	typeErr      error

	lastEventLimit  int
	lastTodoLimit   int
	lastRecentLimit int
}

func (m *mockHomeRepo) ListEventsInRange(_ context.Context, _ string, _, _ string, limit int) ([]domain.EventSummary, int, error) {
	m.lastEventLimit = limit
	return m.events, m.eventTotal, m.eventErr
}

func (m *mockHomeRepo) ListOverdueTodos(_ context.Context, _ string, _ string, limit int) ([]domain.TodoSummary, int, error) {
	m.lastTodoLimit = limit
	return m.overdue, m.overdueCount, m.overdueErr
}

func (m *mockHomeRepo) ListTodayTodos(_ context.Context, _ string, _ string, limit int) ([]domain.TodoSummary, int, error) {
	m.lastTodoLimit = limit
	return m.today, m.todayCount, m.todayErr
}

func (m *mockHomeRepo) ListRecentFiles(_ context.Context, _ string, limit int) ([]domain.RecentFileSummary, int, error) {
	m.lastRecentLimit = limit
	return m.recent, m.recentTotal, m.recentErr
}

func (m *mockHomeRepo) GetStorageSummary(context.Context, string) (domain.StorageSummary, error) {
	return m.storage, m.storageErr
}

func (m *mockHomeRepo) ListStorageTypeUsage(context.Context, string) ([]domain.StorageTypeUsage, error) {
	return m.typeUsage, m.typeErr
}

func fixedInput() portin.GetSummaryInput {
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.Local)
	end := start.Add(24 * time.Hour)
	return portin.GetSummaryInput{
		Start:       start,
		End:         end,
		EventLimit:  10,
		TodoLimit:   10,
		RecentLimit: 10,
	}
}

func TestGetSummaryValidation(t *testing.T) {
	uc := NewHomeUseCase(&mockHomeRepo{})
	input := fixedInput()

	if _, err := uc.GetSummary(context.Background(), "", input); err == nil {
		t.Fatal("expected user id validation error")
	}

	input.Start = time.Time{}
	if _, err := uc.GetSummary(context.Background(), "u1", input); err == nil {
		t.Fatal("expected start required error")
	}

	input = fixedInput()
	input.End = input.Start
	if _, err := uc.GetSummary(context.Background(), "u1", input); err == nil {
		t.Fatal("expected start-before-end error")
	}
}

func TestGetSummarySuccessAndClamp(t *testing.T) {
	repo := &mockHomeRepo{
		eventTotal:   2,
		events:       []domain.EventSummary{{ID: "e1"}, {ID: "e2"}},
		overdue:      []domain.TodoSummary{{ID: "o1"}},
		overdueCount: 1,
		today:        []domain.TodoSummary{{ID: "t1"}},
		todayCount:   1,
		recent:       []domain.RecentFileSummary{{ID: "f1"}},
		recentTotal:  1,
		storage: domain.StorageSummary{
			UsedBytes:  50,
			QuotaBytes: 200,
		},
		typeUsage: []domain.StorageTypeUsage{
			{Type: "image", Bytes: 20},
			{Type: "unknown", Bytes: 10},
		},
	}
	uc := NewHomeUseCase(repo)

	input := fixedInput()
	input.EventLimit = 999
	input.TodoLimit = -1
	input.RecentLimit = 0
	got, err := uc.GetSummary(context.Background(), "u1", input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.lastEventLimit != 20 {
		t.Fatalf("expected event limit clamp 20, got %d", repo.lastEventLimit)
	}
	if repo.lastTodoLimit != 7 {
		t.Fatalf("expected todo limit default 7, got %d", repo.lastTodoLimit)
	}
	if repo.lastRecentLimit != 8 {
		t.Fatalf("expected recent limit default 8, got %d", repo.lastRecentLimit)
	}
	if got.Storage.UsagePercent != 25 {
		t.Fatalf("expected usage percent 25, got %f", got.Storage.UsagePercent)
	}
	if len(got.Storage.Breakdown) != 4 {
		t.Fatalf("expected 4 breakdown items, got %d", len(got.Storage.Breakdown))
	}
}

func TestGetSummaryStorageNoQuota(t *testing.T) {
	repo := &mockHomeRepo{
		storage: domain.StorageSummary{
			UsedBytes:  100,
			QuotaBytes: 0,
		},
	}
	uc := NewHomeUseCase(repo)
	got, err := uc.GetSummary(context.Background(), "u1", fixedInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Storage.UsagePercent != 0 {
		t.Fatalf("expected zero usage percent, got %f", got.Storage.UsagePercent)
	}
}

func TestGetSummaryRepoErrors(t *testing.T) {
	tests := []struct {
		name string
		repo *mockHomeRepo
		want string
	}{
		{name: "events", repo: &mockHomeRepo{eventErr: errors.New("e")}, want: "list events"},
		{name: "overdue", repo: &mockHomeRepo{overdueErr: errors.New("e")}, want: "list overdue todos"},
		{name: "today", repo: &mockHomeRepo{todayErr: errors.New("e")}, want: "list today todos"},
		{name: "recent", repo: &mockHomeRepo{recentErr: errors.New("e")}, want: "list recent files"},
		{name: "storage", repo: &mockHomeRepo{storageErr: errors.New("e")}, want: "get storage summary"},
		{name: "typeUsage", repo: &mockHomeRepo{typeErr: errors.New("e")}, want: "list storage type usage"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			uc := NewHomeUseCase(tc.repo)
			_, err := uc.GetSummary(context.Background(), "u1", fixedInput())
			if err == nil {
				t.Fatalf("expected error for %s", tc.name)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error to contain %q, got %q", tc.want, err.Error())
			}
		})
	}
}

func TestClampLimit(t *testing.T) {
	if got := clampLimit(0, 5, 20); got != 5 {
		t.Fatalf("expected default 5, got %d", got)
	}
	if got := clampLimit(30, 5, 20); got != 20 {
		t.Fatalf("expected max 20, got %d", got)
	}
	if got := clampLimit(7, 5, 20); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestFillStorageBreakdown(t *testing.T) {
	raw := []domain.StorageTypeUsage{
		{Type: "image", Bytes: 40},
		{Type: "video", Bytes: 10},
		{Type: "other-x", Bytes: 5},
	}
	got := fillStorageBreakdown(raw, 0)
	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d", len(got))
	}
	if got[0].Type != "image" || got[0].Bytes != 40 {
		t.Fatalf("unexpected image item: %#v", got[0])
	}
	if got[3].Type != "other" || got[3].Bytes != 5 {
		t.Fatalf("unexpected other item: %#v", got[3])
	}
	if got[0].Percent <= 0 {
		t.Fatalf("expected percent > 0, got %f", got[0].Percent)
	}
}
