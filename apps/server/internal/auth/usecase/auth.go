package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"lifebase/internal/auth/domain"
	portin "lifebase/internal/auth/port/in"
	portout "lifebase/internal/auth/port/out"
)

type JWTOptions struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

const defaultStorageQuotaBytes int64 = 1 << 40 // 1TB

type authUseCase struct {
	jwt           JWTOptions
	users         portout.UserRepository
	googleAccts   portout.GoogleAccountRepository
	refreshTokens portout.RefreshTokenRepository
	googleAuth    portout.GoogleAuthClient
	bootstrapper  portout.UserBootstrapper
}

func NewAuthUseCase(
	jwt JWTOptions,
	users portout.UserRepository,
	googleAccts portout.GoogleAccountRepository,
	refreshTokens portout.RefreshTokenRepository,
	googleAuth portout.GoogleAuthClient,
	bootstrapper portout.UserBootstrapper,
) portin.AuthUseCase {
	return &authUseCase{
		jwt:           jwt,
		users:         users,
		googleAccts:   googleAccts,
		refreshTokens: refreshTokens,
		googleAuth:    googleAuth,
		bootstrapper:  bootstrapper,
	}
}

func (uc *authUseCase) GetAuthURL(state string) string {
	return uc.GetAuthURLForApp(state, "web")
}

func (uc *authUseCase) GetAuthURLForApp(state, app string) string {
	return uc.googleAuth.AuthURLForApp(state, app)
}

func (uc *authUseCase) HandleCallback(ctx context.Context, code string) (*portin.LoginResult, error) {
	return uc.HandleCallbackForApp(ctx, code, "web")
}

func (uc *authUseCase) HandleCallbackForApp(ctx context.Context, code, app string) (*portin.LoginResult, error) {
	token, err := uc.googleAuth.ExchangeCodeForApp(ctx, code, app)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}

	userInfo, err := uc.googleAuth.FetchUserInfo(ctx, *token)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}

	user, err := uc.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	if err := uc.upsertGoogleAccount(ctx, user.ID, userInfo, *token); err != nil {
		return nil, fmt.Errorf("upsert google account: %w", err)
	}

	return uc.issueTokens(ctx, user.ID)
}

func (uc *authUseCase) RefreshAccessToken(ctx context.Context, refreshTokenStr string) (*portin.LoginResult, error) {
	hash := hashToken(refreshTokenStr)

	stored, err := uc.refreshTokens.FindByHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	if time.Now().After(stored.ExpiresAt) {
		_ = uc.refreshTokens.DeleteByHash(ctx, hash)
		return nil, fmt.Errorf("refresh token expired")
	}

	_ = uc.refreshTokens.DeleteByHash(ctx, hash)

	return uc.issueTokens(ctx, stored.UserID)
}

func (uc *authUseCase) Logout(ctx context.Context, userID string) error {
	return uc.refreshTokens.DeleteByUserID(ctx, userID)
}

func (uc *authUseCase) findOrCreateUser(ctx context.Context, info *portout.OAuthUserInfo) (*domain.User, error) {
	user, err := uc.users.FindByEmail(ctx, info.Email)
	if err == nil && user != nil {
		user.Name = info.Name
		user.Picture = info.Picture
		user.UpdatedAt = time.Now()
		_ = uc.users.Update(ctx, user)
		return user, nil
	}

	now := time.Now()
	user = &domain.User{
		ID:                uuid.New().String(),
		Email:             info.Email,
		Name:              info.Name,
		Picture:           info.Picture,
		StorageQuotaBytes: defaultStorageQuotaBytes,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	if uc.bootstrapper != nil {
		_ = uc.bootstrapper.BootstrapUser(ctx, user.ID, now)
	}

	return user, nil
}

func (uc *authUseCase) upsertGoogleAccount(ctx context.Context, userID string, info *portout.OAuthUserInfo, token portout.OAuthToken) error {
	existing, _ := uc.googleAccts.FindByGoogleID(ctx, info.GoogleID)
	now := time.Now()

	if existing != nil {
		existing.AccessToken = token.AccessToken
		existing.RefreshToken = token.RefreshToken
		existing.TokenExpiresAt = &token.Expiry
		existing.Status = "active"
		existing.UpdatedAt = now
		return uc.googleAccts.Update(ctx, existing)
	}

	account := &domain.GoogleAccount{
		ID:             uuid.New().String(),
		UserID:         userID,
		GoogleEmail:    info.Email,
		GoogleID:       info.GoogleID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		TokenExpiresAt: &token.Expiry,
		Scopes:         "openid email profile calendar tasks",
		Status:         "active",
		IsPrimary:      true,
		ConnectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return uc.googleAccts.Create(ctx, account)
}

func (uc *authUseCase) issueTokens(ctx context.Context, userID string) (*portin.LoginResult, error) {
	accessToken, err := uc.generateAccessToken(userID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshTokenStr, err := generateRandomToken()
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}

	refreshToken := &domain.RefreshToken{
		ID:        uuid.New().String(),
		UserID:    userID,
		TokenHash: hashToken(refreshTokenStr),
		ExpiresAt: time.Now().Add(uc.jwt.RefreshExpiry),
		CreatedAt: time.Now(),
	}

	if err := uc.refreshTokens.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &portin.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(uc.jwt.AccessExpiry.Seconds()),
	}, nil
}

func (uc *authUseCase) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(uc.jwt.AccessExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.jwt.Secret))
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(h[:])
}
