package out

import (
	"context"

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
