package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"lifebase/internal/calendar/domain"
	portin "lifebase/internal/calendar/port/in"
)

func TestCalendarUseCaseAdditionalBranches(t *testing.T) {
	t.Run("backfill error branches", func(t *testing.T) {
		uc := &calendarUseCase{backfill: &mockBackfill{err: errors.New("backfill failed")}}
		if _, err := uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
			Start: "2026-03-01T00:00:00Z",
			End:   "bad",
		}); err == nil || !strings.Contains(err.Error(), "invalid end format") {
			t.Fatalf("expected invalid end error, got %v", err)
		}
		if _, err := uc.BackfillEvents(context.Background(), "u1", portin.BackfillEventsInput{
			Start: "2026-03-01T00:00:00Z",
			End:   "2026-03-02T00:00:00Z",
		}); err == nil || !strings.Contains(err.Error(), "backfill failed") {
			t.Fatalf("expected propagated backfill error, got %v", err)
		}
	})

	t.Run("day summary default timezone and repo errors", func(t *testing.T) {
		eventsRepo := &mockDaySummaryEventRepo{}
		uc := &calendarUseCase{events: eventsRepo}
		result, err := uc.GetDaySummary(context.Background(), "u1", portin.DaySummaryInput{Date: "2026-03-01"})
		if err != nil {
			t.Fatalf("unexpected default timezone err: %v", err)
		}
		if result.Timezone != "Asia/Seoul" {
			t.Fatalf("expected default timezone, got %s", result.Timezone)
		}

		uc.holidays = &mockDaySummaryHolidayRepo{err: errors.New("holiday fail")}
		if _, err := uc.GetDaySummary(context.Background(), "u1", portin.DaySummaryInput{Date: "2026-03-01"}); err == nil || !strings.Contains(err.Error(), "holiday fail") {
			t.Fatalf("expected holiday repo error, got %v", err)
		}

		uc.holidays = &mockDaySummaryHolidayRepo{}
		uc.todos = &mockDaySummaryTodoRepo{err: errors.New("todo fail")}
		if _, err := uc.GetDaySummary(context.Background(), "u1", portin.DaySummaryInput{Date: "2026-03-01"}); err == nil || !strings.Contains(err.Error(), "todo fail") {
			t.Fatalf("expected todo repo error, got %v", err)
		}
	})

	t.Run("update event optional fields and reminders", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		eventRepo := &mockEventRepo{
			event: &domain.Event{
				ID:         "e1",
				CalendarID: "c1",
				Title:      "old",
				StartTime:  now,
				EndTime:    now.Add(time.Hour),
			},
		}
		reminders := &mockReminderRepo{}
		uc := &calendarUseCase{
			calendars: &mockCalendarRepo{cal: &domain.Calendar{ID: "c1"}},
			events:    eventRepo,
			reminders: reminders,
			outbox:    &mockOutbox{},
		}

		title := "new"
		description := "desc"
		location := "room"
		start := now.Add(2 * time.Hour).Format(time.RFC3339)
		end := now.Add(3 * time.Hour).Format(time.RFC3339)
		timezone := "UTC"
		allDay := true
		color := "5"
		rrule := "FREQ=WEEKLY"
		updated, err := uc.UpdateEvent(context.Background(), "u1", "e1", portin.UpdateEventInput{
			Title:          &title,
			Description:    &description,
			Location:       &location,
			StartTime:      &start,
			EndTime:        &end,
			Timezone:       &timezone,
			IsAllDay:       &allDay,
			ColorID:        &color,
			RecurrenceRule: &rrule,
			Reminders:      []portin.ReminderInput{{Minutes: 10}},
		})
		if err != nil {
			t.Fatalf("unexpected update err: %v", err)
		}
		if updated.Description != description || updated.Location != location || updated.Timezone != timezone || !updated.IsAllDay {
			t.Fatalf("unexpected updated event: %#v", updated)
		}
		if updated.ColorID == nil || *updated.ColorID != color || updated.RecurrenceRule == nil || *updated.RecurrenceRule != rrule {
			t.Fatalf("expected color and recurrence to update: %#v", updated)
		}
		if len(reminders.created) != 1 || reminders.created[0].Method != "popup" {
			t.Fatalf("expected default popup reminder, got %#v", reminders.created)
		}
	})
}
