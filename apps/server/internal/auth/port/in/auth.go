package in

import "context"

type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

type AuthUseCase interface {
	GetAuthURL(state string) string
	GetAuthURLForApp(state, app string) string
	HandleCallback(ctx context.Context, code string) (*LoginResult, error)
	HandleCallbackForApp(ctx context.Context, code, app string) (*LoginResult, error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (*LoginResult, error)
	Logout(ctx context.Context, userID string) error
}
