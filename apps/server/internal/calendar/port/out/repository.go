package out

import (
	"context"
	"time"

	"lifebase/internal/calendar/domain"
)

type CalendarRepository interface {
	Create(ctx context.Context, cal *domain.Calendar) error
	FindByID(ctx context.Context, userID, id string) (*domain.Calendar, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.Calendar, error)
	Update(ctx context.Context, cal *domain.Calendar) error
	Delete(ctx context.Context, id string) error
}

type EventRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	FindByID(ctx context.Context, userID, id string) (*domain.Event, error)
	ListByRange(ctx context.Context, userID string, calendarIDs []string, start, end string) ([]*domain.Event, error)
	Update(ctx context.Context, event *domain.Event) error
	SoftDelete(ctx context.Context, userID, id string) error
}

type ReminderRepository interface {
	CreateBatch(ctx context.Context, reminders []domain.EventReminder) error
	ListByEvent(ctx context.Context, eventID string) ([]domain.EventReminder, error)
	DeleteByEvent(ctx context.Context, eventID string) error
}

type EventPushOutbox interface {
	EnqueueCreate(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error
	EnqueueUpdate(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error
	EnqueueDelete(ctx context.Context, userID, eventID string, expectedUpdatedAt time.Time) error
}

type CalendarBackfillResult struct {
	FetchedEvents int
	UpdatedEvents int
	DeletedEvents int
	CoveredStart  time.Time
	CoveredEnd    time.Time
}

type CalendarBackfillService interface {
	BackfillEvents(ctx context.Context, userID string, start, end time.Time, calendarIDs []string) (*CalendarBackfillResult, error)
}

type DaySummaryHoliday struct {
	Date time.Time
	Name string
}

type DaySummaryTodo struct {
	ID       string
	ListID   string
	Title    string
	Due      *string
	Priority string
	IsDone   bool
}

type DaySummaryHolidayRepository interface {
	ListByDateRange(ctx context.Context, start, end time.Time) ([]DaySummaryHoliday, error)
}

type DaySummaryTodoRepository interface {
	ListByDueDate(ctx context.Context, userID, date string, includeDone bool) ([]DaySummaryTodo, error)
}
