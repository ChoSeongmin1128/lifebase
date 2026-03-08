package usecase

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"lifebase/internal/calendar/domain"
	portin "lifebase/internal/calendar/port/in"
	portout "lifebase/internal/calendar/port/out"
)

type mockDaySummaryEventRepo struct {
	events []*domain.Event
	err    error

	gotUserID      string
	gotCalendarIDs []string
	gotStart       string
	gotEnd         string
}

func (m *mockDaySummaryEventRepo) Create(_ context.Context, _ *domain.Event) error {
	return nil
}

func (m *mockDaySummaryEventRepo) FindByID(_ context.Context, _, _ string) (*domain.Event, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockDaySummaryEventRepo) ListByRange(_ context.Context, userID string, calendarIDs []string, start, end string) ([]*domain.Event, error) {
	m.gotUserID = userID
	m.gotCalendarIDs = append([]string(nil), calendarIDs...)
	m.gotStart = start
	m.gotEnd = end
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}

func (m *mockDaySummaryEventRepo) Update(_ context.Context, _ *domain.Event) error {
	return nil
}

func (m *mockDaySummaryEventRepo) SoftDelete(_ context.Context, _, _ string) error {
	return nil
}

type mockDaySummaryHolidayRepo struct {
	rows []portout.DaySummaryHoliday
	err  error

	gotStart time.Time
	gotEnd   time.Time
}

func (m *mockDaySummaryHolidayRepo) ListByDateRange(_ context.Context, start, end time.Time) ([]portout.DaySummaryHoliday, error) {
	m.gotStart = start
	m.gotEnd = end
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

type mockDaySummaryTodoRepo struct {
	rows []portout.DaySummaryTodo
	err  error

	gotUserID      string
	gotDate        string
	gotIncludeDone bool
}

func (m *mockDaySummaryTodoRepo) ListByDueDate(_ context.Context, userID, date string, includeDone bool) ([]portout.DaySummaryTodo, error) {
	m.gotUserID = userID
	m.gotDate = date
	m.gotIncludeDone = includeDone
	if m.err != nil {
		return nil, m.err
	}
	return m.rows, nil
}

func TestGetDaySummary_InvalidDate(t *testing.T) {
	uc := &calendarUseCase{}

	_, err := uc.GetDaySummary(context.Background(), "user-1", portin.DaySummaryInput{
		Date: "2026/03/01",
	})
	if err == nil {
		t.Fatal("expected invalid date error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid date format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDaySummary_InvalidTimezone(t *testing.T) {
	uc := &calendarUseCase{}

	_, err := uc.GetDaySummary(context.Background(), "user-1", portin.DaySummaryInput{
		Date:     "2026-03-01",
		Timezone: "Invalid/Timezone",
	})
	if err == nil {
		t.Fatal("expected invalid timezone error, got nil")
	}
	if !strings.Contains(err.Error(), "invalid timezone") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDaySummary_Success(t *testing.T) {
	due := "2026-03-01"
	eventsRepo := &mockDaySummaryEventRepo{
		events: []*domain.Event{
			{
				ID:         "event-1",
				CalendarID: "cal-1",
				UserID:     "user-1",
				Title:      "회의",
				StartTime:  time.Date(2026, 3, 1, 2, 0, 0, 0, time.UTC),
				EndTime:    time.Date(2026, 3, 1, 3, 0, 0, 0, time.UTC),
				Timezone:   "Asia/Seoul",
			},
		},
	}
	holidayRepo := &mockDaySummaryHolidayRepo{
		rows: []portout.DaySummaryHoliday{
			{Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Name: "삼일절"},
			{Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Name: "삼일절"},
			{Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Name: "A Holiday"},
			{Date: time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC), Name: "Earlier"},
		},
	}
	todoRepo := &mockDaySummaryTodoRepo{
		rows: []portout.DaySummaryTodo{
			{
				ID:      "todo-1",
				ListID:  "list-1",
				Title:   "서류 제출",
				DueDate: &due,
				IsDone:  false,
			},
		},
	}
	uc := &calendarUseCase{
		events:   eventsRepo,
		holidays: holidayRepo,
		todos:    todoRepo,
	}

	result, err := uc.GetDaySummary(context.Background(), "user-1", portin.DaySummaryInput{
		Date:             "2026-03-01",
		Timezone:         "Asia/Seoul",
		CalendarIDs:      []string{"cal-1"},
		IncludeDoneTodos: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Date != "2026-03-01" {
		t.Fatalf("expected date 2026-03-01, got %s", result.Date)
	}
	if result.Timezone != "Asia/Seoul" {
		t.Fatalf("expected timezone Asia/Seoul, got %s", result.Timezone)
	}

	if eventsRepo.gotUserID != "user-1" {
		t.Fatalf("unexpected user id: %s", eventsRepo.gotUserID)
	}
	if !reflect.DeepEqual(eventsRepo.gotCalendarIDs, []string{"cal-1"}) {
		t.Fatalf("unexpected calendar ids: %#v", eventsRepo.gotCalendarIDs)
	}
	if eventsRepo.gotStart != "2026-02-28T15:00:00Z" || eventsRepo.gotEnd != "2026-03-01T15:00:00Z" {
		t.Fatalf("unexpected range: %s - %s", eventsRepo.gotStart, eventsRepo.gotEnd)
	}

	if len(result.Holidays) != 3 {
		t.Fatalf("expected deduped 3 holidays, got %d", len(result.Holidays))
	}
	if result.Holidays[0].Name != "Earlier" || result.Holidays[1].Name != "A Holiday" || result.Holidays[2].Name != "삼일절" {
		t.Fatalf("unexpected holiday order: %#v", result.Holidays)
	}

	if todoRepo.gotUserID != "user-1" || todoRepo.gotDate != "2026-03-01" || !todoRepo.gotIncludeDone {
		t.Fatalf("unexpected todo query args: user=%s date=%s includeDone=%v", todoRepo.gotUserID, todoRepo.gotDate, todoRepo.gotIncludeDone)
	}
	if len(result.Todos) != 1 || result.Todos[0].ID != "todo-1" {
		t.Fatalf("unexpected todo result: %#v", result.Todos)
	}
}

func TestGetDaySummary_EventRepoError(t *testing.T) {
	uc := &calendarUseCase{
		events: &mockDaySummaryEventRepo{err: errors.New("event repo failed")},
	}

	_, err := uc.GetDaySummary(context.Background(), "user-1", portin.DaySummaryInput{
		Date: "2026-03-01",
	})
	if err == nil {
		t.Fatal("expected event repo error, got nil")
	}
	if !strings.Contains(err.Error(), "event repo failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
