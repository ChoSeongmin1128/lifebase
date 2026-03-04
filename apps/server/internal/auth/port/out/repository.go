package out

import (
	"context"
	"time"

	"lifebase/internal/auth/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id string) (*domain.User, error)
	ListUsers(ctx context.Context, search, cursor string, limit int) ([]*domain.User, string, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	UpdateStorageQuota(ctx context.Context, userID string, quotaBytes int64) error
	UpdateStorageUsed(ctx context.Context, userID string, usedBytes int64) error
}

type GoogleAccountRepository interface {
	FindByGoogleID(ctx context.Context, googleID string) (*domain.GoogleAccount, error)
	FindByID(ctx context.Context, userID, id string) (*domain.GoogleAccount, error)
	FindByUserID(ctx context.Context, userID string) ([]*domain.GoogleAccount, error)
	Create(ctx context.Context, account *domain.GoogleAccount) error
	Update(ctx context.Context, account *domain.GoogleAccount) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, token *domain.RefreshToken) error
	FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	DeleteByUserID(ctx context.Context, userID string) error
	DeleteByHash(ctx context.Context, tokenHash string) error
	DeleteExpired(ctx context.Context) error
}

type OAuthToken struct {
	AccessToken  string
	RefreshToken string
	Expiry       time.Time
}

type OAuthUserInfo struct {
	GoogleID string
	Email    string
	Name     string
	Picture  string
}

type OAuthCalendar struct {
	GoogleID  string
	Name      string
	ColorID   *string
	IsPrimary bool
	IsVisible bool
}

type OAuthTaskList struct {
	GoogleID string
	Name     string
}

type OAuthCalendarEvent struct {
	GoogleID       string
	Status         string
	Title          string
	Description    string
	Location       string
	StartTime      *time.Time
	EndTime        *time.Time
	Timezone       string
	IsAllDay       bool
	ColorID        *string
	RecurrenceRule *string
	ETag           *string
}

type OAuthCalendarEventsPage struct {
	Events        []OAuthCalendarEvent
	NextPageToken string
	NextSyncToken string
}

type OAuthTask struct {
	GoogleID    string
	Title       string
	Notes       string
	DueDate     *string
	IsDone      bool
	IsDeleted   bool
	CompletedAt *time.Time
}

type OAuthTasksPage struct {
	Items         []OAuthTask
	NextPageToken string
}

type CalendarEventUpsertInput struct {
	Title          string
	Description    string
	Location       string
	StartTime      time.Time
	EndTime        time.Time
	Timezone       string
	IsAllDay       bool
	ColorID        *string
	RecurrenceRule *string
	ETag           *string
}

type TodoUpsertInput struct {
	Title   string
	Notes   string
	DueDate *string
	IsDone  bool
}

type GoogleAPIError struct {
	StatusCode int
	Message    string
}

func (e *GoogleAPIError) Error() string {
	if e == nil {
		return "google api error"
	}
	return e.Message
}

type GoogleAuthClient interface {
	AuthURL(state string) string
	AuthURLForApp(state, app string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthToken, error)
	ExchangeCodeForApp(ctx context.Context, code, app string) (*OAuthToken, error)
	FetchUserInfo(ctx context.Context, token OAuthToken) (*OAuthUserInfo, error)
	ListCalendars(ctx context.Context, token OAuthToken) ([]OAuthCalendar, error)
	ListTaskLists(ctx context.Context, token OAuthToken) ([]OAuthTaskList, error)
	ListCalendarEvents(ctx context.Context, token OAuthToken, calendarID, pageToken, syncToken string, timeMin, timeMax *time.Time) (*OAuthCalendarEventsPage, error)
	ListTasks(ctx context.Context, token OAuthToken, taskListID, pageToken string) (*OAuthTasksPage, error)
	CreateCalendarEvent(ctx context.Context, token OAuthToken, calendarID string, input CalendarEventUpsertInput) (googleID string, etag *string, err error)
	UpdateCalendarEvent(ctx context.Context, token OAuthToken, calendarID, eventID string, input CalendarEventUpsertInput) (etag *string, err error)
	DeleteCalendarEvent(ctx context.Context, token OAuthToken, calendarID, eventID string) error
	CreateTask(ctx context.Context, token OAuthToken, taskListID string, input TodoUpsertInput) (googleID string, err error)
	UpdateTask(ctx context.Context, token OAuthToken, taskListID, taskID string, input TodoUpsertInput) error
	DeleteTask(ctx context.Context, token OAuthToken, taskListID, taskID string) error
}

type GoogleSyncOptions struct {
	SyncCalendar bool
	SyncTodo     bool
}

type GoogleAccountSyncer interface {
	SyncAccount(ctx context.Context, userID string, account *domain.GoogleAccount, options GoogleSyncOptions) error
}

type GoogleSyncCoordinator interface {
	TriggerUserSync(ctx context.Context, userID, area, reason string) (int, error)
	RunHourlySync(ctx context.Context) (int, error)
}

type GooglePushProcessor interface {
	ProcessPending(ctx context.Context, limit int) (int, error)
}

type UserBootstrapper interface {
	BootstrapUser(ctx context.Context, userID string, now time.Time) error
}
