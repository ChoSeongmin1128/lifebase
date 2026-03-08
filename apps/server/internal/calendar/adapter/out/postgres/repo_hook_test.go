package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/calendar/domain"
	calendarportout "lifebase/internal/calendar/port/out"
	"lifebase/internal/testutil/dbtest"
)

func TestCalendarReposScanHookErrorBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"

	calendarRepo := NewCalendarRepo(db)
	eventRepo := NewEventRepo(db)
	reminderRepo := NewReminderRepo(db)
	holidayRepo := NewDaySummaryHolidayRepo(db)
	todoRepo := NewDaySummaryTodoRepo(db)

	if err := calendarRepo.Create(ctx, &domain.Calendar{
		ID: "cal-hook", UserID: userID, Name: "Hook", Kind: "local", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create calendar: %v", err)
	}
	if err := eventRepo.Create(ctx, &domain.Event{
		ID: "evt-hook", CalendarID: "cal-hook", UserID: userID, Title: "Hook",
		StartTime: now, EndTime: now.Add(time.Hour), Timezone: "UTC", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create event: %v", err)
	}

	prevCal := scanCalendarRowsFn
	prevEvt := scanEventRowsFn
	prevRem := scanReminderRowsFn
	prevHoliday := scanDaySummaryHolidayRowsFn
	prevTodo := scanDaySummaryTodoRowsFn
	t.Cleanup(func() {
		scanCalendarRowsFn = prevCal
		scanEventRowsFn = prevEvt
		scanReminderRowsFn = prevRem
		scanDaySummaryHolidayRowsFn = prevHoliday
		scanDaySummaryTodoRowsFn = prevTodo
	})

	scanCalendarRowsFn = func(pgx.Rows) ([]*domain.Calendar, error) {
		return nil, errors.New("calendar scan fail")
	}
	if _, err := calendarRepo.ListByUser(ctx, userID); err == nil {
		t.Fatal("expected ListByUser scan hook error")
	}
	scanCalendarRowsFn = prevCal

	scanEventRowsFn = func(pgx.Rows) ([]*domain.Event, error) {
		return nil, errors.New("event scan fail")
	}
	if _, err := eventRepo.ListByRange(ctx, userID, []string{"cal-hook", "cal-2"}, now.Add(-time.Hour).Format(time.RFC3339), now.Add(time.Hour).Format(time.RFC3339)); err == nil {
		t.Fatal("expected ListByRange scan hook error")
	}
	scanEventRowsFn = prevEvt

	scanReminderRowsFn = func(pgx.Rows) ([]domain.EventReminder, error) {
		return nil, errors.New("reminder scan fail")
	}
	if _, err := reminderRepo.ListByEvent(ctx, "evt-hook"); err == nil {
		t.Fatal("expected ListByEvent scan hook error")
	}
	scanReminderRowsFn = prevRem

	scanDaySummaryHolidayRowsFn = func(pgx.Rows) ([]calendarportout.DaySummaryHoliday, error) {
		return nil, errors.New("holiday summary scan fail")
	}
	if _, err := holidayRepo.ListByDateRange(ctx, now, now); err == nil {
		t.Fatal("expected ListByDateRange scan hook error")
	}
	scanDaySummaryHolidayRowsFn = prevHoliday

	scanDaySummaryTodoRowsFn = func(pgx.Rows) ([]calendarportout.DaySummaryTodo, error) {
		return nil, errors.New("todo summary scan fail")
	}
	if _, err := todoRepo.ListByDueDate(ctx, userID, now.Format("2006-01-02"), false); err == nil {
		t.Fatal("expected ListByDueDate scan hook error")
	}
}

func TestEventPushOutboxRepoInvalidAccountIDInsertError(t *testing.T) {
	prevQuery := queryCalendarOutboxAccountFn
	prevInsert := insertCalendarOutboxFn
	t.Cleanup(func() {
		queryCalendarOutboxAccountFn = prevQuery
		insertCalendarOutboxFn = prevInsert
	})

	queryCalendarOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return strPtr("not-a-uuid"), nil
	}
	insertCalendarOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return errors.New("insert fail")
	}

	outboxRepo := NewEventPushOutboxRepo(nil)
	if err := outboxRepo.EnqueueCreate(context.Background(), "user-1", "evt-bad-account", time.Now().UTC()); err == nil {
		t.Fatal("expected enqueue insert error")
	}
}
