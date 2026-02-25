package domain

import "time"

type Event struct {
	ID             string     `json:"id"`
	CalendarID     string     `json:"calendar_id"`
	UserID         string     `json:"user_id"`
	GoogleID       *string    `json:"google_id,omitempty"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Location       string     `json:"location"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        time.Time  `json:"end_time"`
	Timezone       string     `json:"timezone"`
	IsAllDay       bool       `json:"is_all_day"`
	ColorID        *string    `json:"color_id,omitempty"`
	RecurrenceRule *string    `json:"recurrence_rule,omitempty"`
	ETag           *string    `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`

	Reminders []EventReminder `json:"reminders,omitempty"`
}

type EventReminder struct {
	ID        string    `json:"id"`
	EventID   string    `json:"event_id"`
	Method    string    `json:"method"` // popup, email
	Minutes   int       `json:"minutes"`
	CreatedAt time.Time `json:"created_at"`
}

type EventException struct {
	ID               string     `json:"id"`
	RecurringEventID string     `json:"recurring_event_id"`
	OriginalStart    time.Time  `json:"original_start"`
	IsCancelled      bool       `json:"is_cancelled"`
	Title            *string    `json:"title,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Location         *string    `json:"location,omitempty"`
	StartTime        *time.Time `json:"start_time,omitempty"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}
