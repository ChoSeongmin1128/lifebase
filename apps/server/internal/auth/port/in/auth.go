package in

import (
	"context"
	"time"
)

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type GoogleAccountSummary struct {
	ID          string    `json:"id"`
	GoogleEmail string    `json:"google_email"`
	Status      string    `json:"status"`
	IsPrimary   bool      `json:"is_primary"`
	ConnectedAt time.Time `json:"connected_at"`
}

type SyncGoogleAccountInput struct {
	SyncCalendar bool `json:"sync_calendar"`
	SyncTodo     bool `json:"sync_todo"`
}

type AuthUseCase interface {
	GetAuthURL(state string) string
	GetAuthURLForApp(state, app string) string
	HandleCallback(ctx context.Context, code string) (*LoginResult, error)
	HandleCallbackForApp(ctx context.Context, code, app string) (*LoginResult, error)
	ListGoogleAccounts(ctx context.Context, userID string) ([]GoogleAccountSummary, error)
	LinkGoogleAccount(ctx context.Context, userID, code, app string) error
	SyncGoogleAccount(ctx context.Context, userID, accountID string, input SyncGoogleAccountInput) error
	RefreshAccessToken(ctx context.Context, refreshToken string) (*LoginResult, error)
	Logout(ctx context.Context, userID string) error
}
