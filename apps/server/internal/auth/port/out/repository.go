package out

import (
	"context"
	"time"

	"lifebase/internal/auth/domain"
)

type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
}

type GoogleAccountRepository interface {
	FindByGoogleID(ctx context.Context, googleID string) (*domain.GoogleAccount, error)
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

type GoogleAuthClient interface {
	AuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*OAuthToken, error)
	FetchUserInfo(ctx context.Context, token OAuthToken) (*OAuthUserInfo, error)
}

type UserBootstrapper interface {
	BootstrapUser(ctx context.Context, userID string, now time.Time) error
}
