package in

import (
	"context"
	"time"

	"lifebase/internal/calendar/domain"
)

type CreateEventInput struct {
	CalendarID     string          `json:"calendar_id"`
	Title          string          `json:"title"`
	Description    string          `json:"description"`
	Location       string          `json:"location"`
	StartTime      string          `json:"start_time"`
	EndTime        string          `json:"end_time"`
	Timezone       string          `json:"timezone"`
	IsAllDay       bool            `json:"is_all_day"`
	ColorID        *string         `json:"color_id"`
	RecurrenceRule *string         `json:"recurrence_rule"`
	Reminders      []ReminderInput `json:"reminders"`
}

type UpdateEventInput struct {
	Title          *string         `json:"title"`
	Description    *string         `json:"description"`
	Location       *string         `json:"location"`
	StartTime      *string         `json:"start_time"`
	EndTime        *string         `json:"end_time"`
	Timezone       *string         `json:"timezone"`
	IsAllDay       *bool           `json:"is_all_day"`
	ColorID        *string         `json:"color_id"`
	RecurrenceRule *string         `json:"recurrence_rule"`
	Reminders      []ReminderInput `json:"reminders"`
}

type ReminderInput struct {
	Method  string `json:"method"`
	Minutes int    `json:"minutes"`
}

type BackfillEventsInput struct {
	Start       string   `json:"start"`
	End         string   `json:"end"`
	CalendarIDs []string `json:"calendar_ids"`
	Reason      string   `json:"reason"`
}

type BackfillEventsResult struct {
	FetchedEvents int       `json:"fetched_events"`
	UpdatedEvents int       `json:"updated_events"`
	DeletedEvents int       `json:"deleted_events"`
	CoveredStart  time.Time `json:"covered_start"`
	CoveredEnd    time.Time `json:"covered_end"`
}

type CalendarUseCase interface {
	// Calendars
	CreateCalendar(ctx context.Context, userID, name string, colorID *string) (*domain.Calendar, error)
	ListCalendars(ctx context.Context, userID string) ([]*domain.Calendar, error)
	UpdateCalendar(ctx context.Context, userID, calID, name string, colorID *string, isVisible *bool) error
	DeleteCalendar(ctx context.Context, userID, calID string) error

	// Events
	CreateEvent(ctx context.Context, userID string, input CreateEventInput) (*domain.Event, error)
	GetEvent(ctx context.Context, userID, eventID string) (*domain.Event, error)
	ListEvents(ctx context.Context, userID string, calendarIDs []string, start, end string) ([]*domain.Event, error)
	BackfillEvents(ctx context.Context, userID string, input BackfillEventsInput) (*BackfillEventsResult, error)
	UpdateEvent(ctx context.Context, userID, eventID string, input UpdateEventInput) (*domain.Event, error)
	DeleteEvent(ctx context.Context, userID, eventID string) error
}
