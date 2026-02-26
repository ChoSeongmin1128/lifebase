package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"lifebase/internal/auth/domain"
	portin "lifebase/internal/auth/port/in"
	portout "lifebase/internal/auth/port/out"
	"lifebase/internal/shared/config"
	tododomain "lifebase/internal/todo/domain"
	todoportout "lifebase/internal/todo/port/out"
)

type googleUserInfo struct {
	Sub     string `json:"sub"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

type authUseCase struct {
	cfg           *config.Config
	oauthConfig   *oauth2.Config
	users         portout.UserRepository
	googleAccts   portout.GoogleAccountRepository
	refreshTokens portout.RefreshTokenRepository
	todoLists     todoportout.TodoListRepository
}

func NewAuthUseCase(
	cfg *config.Config,
	users portout.UserRepository,
	googleAccts portout.GoogleAccountRepository,
	refreshTokens portout.RefreshTokenRepository,
	todoLists todoportout.TodoListRepository,
) portin.AuthUseCase {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.Google.ClientID,
		ClientSecret: cfg.Google.ClientSecret,
		Scopes: []string{
			"openid",
			"email",
			"profile",
			"https://www.googleapis.com/auth/calendar",
			"https://www.googleapis.com/auth/tasks",
		},
		Endpoint:    google.Endpoint,
		RedirectURL: cfg.Server.WebOrigin + "/auth/callback",
	}

	return &authUseCase{
		cfg:           cfg,
		oauthConfig:   oauthConfig,
		users:         users,
		googleAccts:   googleAccts,
		refreshTokens: refreshTokens,
		todoLists:     todoLists,
	}
}

func (uc *authUseCase) GetAuthURL(state string) string {
	return uc.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
}

func (uc *authUseCase) HandleCallback(ctx context.Context, code string) (*portin.LoginResult, error) {
	token, err := uc.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("oauth exchange: %w", err)
	}

	userInfo, err := uc.fetchGoogleUserInfo(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("fetch user info: %w", err)
	}

	user, err := uc.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	if err := uc.upsertGoogleAccount(ctx, user.ID, userInfo, token); err != nil {
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

func (uc *authUseCase) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*googleUserInfo, error) {
	client := uc.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned %d", resp.StatusCode)
	}

	var info googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (uc *authUseCase) findOrCreateUser(ctx context.Context, info *googleUserInfo) (*domain.User, error) {
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
		StorageQuotaBytes: 1099511627776, // 1TB
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	// Create default todo list for new user
	if uc.todoLists != nil {
		defaultList := &tododomain.TodoList{
			ID:        uuid.New().String(),
			UserID:    user.ID,
			Name:      "할 일",
			SortOrder: 0,
			CreatedAt: now,
			UpdatedAt: now,
		}
		_ = uc.todoLists.Create(ctx, defaultList)
	}

	return user, nil
}

func (uc *authUseCase) upsertGoogleAccount(ctx context.Context, userID string, info *googleUserInfo, token *oauth2.Token) error {
	existing, _ := uc.googleAccts.FindByGoogleID(ctx, info.Sub)
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
		GoogleID:       info.Sub,
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
		ExpiresAt: time.Now().Add(uc.cfg.JWT.RefreshExpiry),
		CreatedAt: time.Now(),
	}

	if err := uc.refreshTokens.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}

	return &portin.LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenStr,
		ExpiresIn:    int(uc.cfg.JWT.AccessExpiry.Seconds()),
	}, nil
}

func (uc *authUseCase) generateAccessToken(userID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": userID,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(uc.cfg.JWT.AccessExpiry).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(uc.cfg.JWT.Secret))
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
