package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/calendar/domain"
)

func newNoDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://localhost:1/lifebase_test?sslmode=disable")
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestCalendarReposNoDBErrorBranches(t *testing.T) {
	pool := newNoDBPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	now := time.Now().UTC()
	start := now.Format(time.RFC3339)
	end := now.Add(time.Hour).Format(time.RFC3339)

	calendars := NewCalendarRepo(pool)
	events := NewEventRepo(pool)
	reminders := NewReminderRepo(pool)
	holidays := NewDaySummaryHolidayRepo(pool)
	todos := NewDaySummaryTodoRepo(pool)
	outbox := NewEventPushOutboxRepo(pool)

	if calendars == nil || events == nil || reminders == nil || holidays == nil || todos == nil || outbox == nil {
		t.Fatal("expected repos to be constructed")
	}

	if err := calendars.Create(ctx, &domain.Calendar{ID: "c1", UserID: "u1", Name: "C1", Kind: "local", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected calendar create error")
	}
	if _, err := calendars.FindByID(ctx, "u1", "c1"); err == nil {
		t.Fatal("expected calendar find error")
	}
	if _, err := calendars.ListByUser(ctx, "u1"); err == nil {
		t.Fatal("expected calendar list error")
	}
	if err := calendars.Update(ctx, &domain.Calendar{ID: "c1", UserID: "u1", Name: "C1", Kind: "local", UpdatedAt: now}); err == nil {
		t.Fatal("expected calendar update error")
	}
	if err := calendars.Delete(ctx, "c1"); err == nil {
		t.Fatal("expected calendar delete error")
	}

	if err := events.Create(ctx, &domain.Event{
		ID: "e1", CalendarID: "c1", UserID: "u1", Title: "E1", StartTime: now, EndTime: now.Add(time.Hour), Timezone: "Asia/Seoul", CreatedAt: now, UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected event create error")
	}
	if _, err := events.FindByID(ctx, "u1", "e1"); err == nil {
		t.Fatal("expected event find error")
	}
	if _, err := events.ListByRange(ctx, "u1", nil, start, end); err == nil {
		t.Fatal("expected event list range error")
	}
	if err := events.Update(ctx, &domain.Event{
		ID: "e1", CalendarID: "c1", UserID: "u1", Title: "E1", StartTime: now, EndTime: now.Add(time.Hour), Timezone: "Asia/Seoul", UpdatedAt: now,
	}); err == nil {
		t.Fatal("expected event update error")
	}
	if err := events.SoftDelete(ctx, "u1", "e1"); err == nil {
		t.Fatal("expected event soft delete error")
	}

	if err := reminders.CreateBatch(ctx, []domain.EventReminder{{ID: "r1", EventID: "e1", Method: "popup", Minutes: 10, CreatedAt: now}}); err == nil {
		t.Fatal("expected reminder create batch error")
	}
	if _, err := reminders.ListByEvent(ctx, "e1"); err == nil {
		t.Fatal("expected reminder list error")
	}
	if err := reminders.DeleteByEvent(ctx, "e1"); err == nil {
		t.Fatal("expected reminder delete error")
	}

	if _, err := holidays.ListByDateRange(ctx, now, now.AddDate(0, 0, 1)); err == nil {
		t.Fatal("expected holiday list error")
	}
	if _, err := todos.ListByDueDate(ctx, "u1", now.Format("2006-01-02"), true); err == nil {
		t.Fatal("expected due-date todo list error")
	}

	if err := outbox.EnqueueCreate(ctx, "u1", "e1", now); err == nil {
		t.Fatal("expected outbox enqueue create error")
	}
	if err := outbox.EnqueueUpdate(ctx, "u1", "e1", now); err == nil {
		t.Fatal("expected outbox enqueue update error")
	}
	if err := outbox.EnqueueDelete(ctx, "u1", "e1", now); err == nil {
		t.Fatal("expected outbox enqueue delete error")
	}
}

