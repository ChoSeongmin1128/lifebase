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

var randRead = rand.Read
var signJWTToken = func(token *jwt.Token, secret string) (string, error) {
	return token.SignedString([]byte(secret))
}

type JWTOptions struct {
	Secret        string
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
}

const defaultStorageQuotaBytes int64 = 15 * (1 << 30) // 15GB

type authUseCase struct {
	jwt             JWTOptions
	users           portout.UserRepository
	admins          portout.AdminAccessRepository
	googleAccts     portout.GoogleAccountRepository
	refreshTokens   portout.RefreshTokenRepository
	googleAuth      portout.GoogleAuthClient
	googleSyncer    portout.GoogleAccountSyncer
	syncCoordinator portout.GoogleSyncCoordinator
	pushProcessor   portout.GooglePushProcessor
	bootstrapper    portout.UserBootstrapper
}

func NewAuthUseCase(
	jwt JWTOptions,
	users portout.UserRepository,
	admins portout.AdminAccessRepository,
	googleAccts portout.GoogleAccountRepository,
	refreshTokens portout.RefreshTokenRepository,
	googleAuth portout.GoogleAuthClient,
	googleSyncer portout.GoogleAccountSyncer,
	syncCoordinator portout.GoogleSyncCoordinator,
	pushProcessor portout.GooglePushProcessor,
	bootstrapper portout.UserBootstrapper,
) portin.AuthUseCase {
	return &authUseCase{
		jwt:             jwt,
		users:           users,
		admins:          admins,
		googleAccts:     googleAccts,
		refreshTokens:   refreshTokens,
		googleAuth:      googleAuth,
		googleSyncer:    googleSyncer,
		syncCoordinator: syncCoordinator,
		pushProcessor:   pushProcessor,
		bootstrapper:    bootstrapper,
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

	if err := uc.ensureAdminLoginAllowed(ctx, app, userInfo); err != nil {
		return nil, err
	}

	user, err := uc.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("find or create user: %w", err)
	}

	if err := uc.upsertGoogleAccount(ctx, user.ID, userInfo, *token); err != nil {
		return nil, fmt.Errorf("upsert google account: %w", err)
	}
	// Login should not fail just because background sync fails.
	_ = uc.syncGoogleAccountByGoogleID(ctx, user.ID, userInfo.GoogleID, portout.GoogleSyncOptions{
		SyncCalendar: true,
		SyncTodo:     true,
	})

	return uc.issueTokens(ctx, user.ID)
}

func (uc *authUseCase) ensureAdminLoginAllowed(ctx context.Context, app string, userInfo *portout.OAuthUserInfo) error {
	if app != "admin" {
		return nil
	}
	if uc.admins == nil {
		return fmt.Errorf("%w: admin repository is not configured", portin.ErrAdminAccessCheckFailed)
	}

	user, err := uc.users.FindByEmail(ctx, userInfo.Email)
	if err != nil || user == nil {
		return portin.ErrAdminAccessDenied
	}

	ok, err := uc.admins.IsActiveAdmin(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("%w: %v", portin.ErrAdminAccessCheckFailed, err)
	}
	if !ok {
		return portin.ErrAdminAccessDenied
	}
	return nil
}

func (uc *authUseCase) ListGoogleAccounts(ctx context.Context, userID string) ([]portin.GoogleAccountSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user id is required")
	}

	accounts, err := uc.googleAccts.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list google accounts: %w", err)
	}

	summaries := make([]portin.GoogleAccountSummary, 0, len(accounts))
	for _, account := range accounts {
		summaries = append(summaries, portin.GoogleAccountSummary{
			ID:          account.ID,
			GoogleEmail: account.GoogleEmail,
			Status:      account.Status,
			IsPrimary:   account.IsPrimary,
			ConnectedAt: account.ConnectedAt,
		})
	}
	return summaries, nil
}

func (uc *authUseCase) LinkGoogleAccount(ctx context.Context, userID, code, app string) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	if code == "" {
		return fmt.Errorf("authorization code is required")
	}

	token, err := uc.googleAuth.ExchangeCodeForApp(ctx, code, app)
	if err != nil {
		return fmt.Errorf("oauth exchange: %w", err)
	}

	userInfo, err := uc.googleAuth.FetchUserInfo(ctx, *token)
	if err != nil {
		return fmt.Errorf("fetch user info: %w", err)
	}

	existing, _ := uc.googleAccts.FindByGoogleID(ctx, userInfo.GoogleID)
	now := time.Now()
	if existing != nil {
		if existing.UserID != userID {
			return fmt.Errorf("google account already linked to another user")
		}

		existing.AccessToken = token.AccessToken
		if token.RefreshToken != "" {
			existing.RefreshToken = token.RefreshToken
		}
		existing.TokenExpiresAt = &token.Expiry
		existing.Status = "active"
		existing.UpdatedAt = now
		if err := uc.googleAccts.Update(ctx, existing); err != nil {
			return err
		}
		return uc.syncGoogleAccountByGoogleID(ctx, userID, userInfo.GoogleID, portout.GoogleSyncOptions{
			SyncCalendar: true,
			SyncTodo:     true,
		})
	}

	userAccounts, err := uc.googleAccts.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list google accounts: %w", err)
	}

	account := &domain.GoogleAccount{
		ID:             uuid.New().String(),
		UserID:         userID,
		GoogleEmail:    userInfo.Email,
		GoogleID:       userInfo.GoogleID,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		TokenExpiresAt: &token.Expiry,
		Scopes:         "openid email profile calendar tasks",
		Status:         "active",
		IsPrimary:      len(userAccounts) == 0,
		ConnectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := uc.googleAccts.Create(ctx, account); err != nil {
		return err
	}
	return uc.syncGoogleAccountByGoogleID(ctx, userID, userInfo.GoogleID, portout.GoogleSyncOptions{
		SyncCalendar: true,
		SyncTodo:     true,
	})
}

func (uc *authUseCase) SyncGoogleAccount(
	ctx context.Context,
	userID, accountID string,
	input portin.SyncGoogleAccountInput,
) error {
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	if accountID == "" {
		return fmt.Errorf("account id is required")
	}
	if uc.googleSyncer == nil {
		return fmt.Errorf("google sync is not configured")
	}

	account, err := uc.googleAccts.FindByID(ctx, userID, accountID)
	if err != nil {
		return fmt.Errorf("google account not found")
	}

	return uc.googleSyncer.SyncAccount(ctx, userID, account, portout.GoogleSyncOptions{
		SyncCalendar: input.SyncCalendar,
		SyncTodo:     input.SyncTodo,
	})
}

func (uc *authUseCase) TriggerGoogleSync(
	ctx context.Context,
	userID string,
	input portin.TriggerGoogleSyncInput,
) (int, error) {
	if uc.syncCoordinator == nil {
		return 0, nil
	}
	area := input.Area
	if area == "" {
		area = "both"
	}
	reason := input.Reason
	if reason == "" {
		reason = "manual"
	}
	return uc.syncCoordinator.TriggerUserSync(ctx, userID, area, reason)
}

func (uc *authUseCase) RunHourlyGoogleSync(ctx context.Context) (int, error) {
	if uc.syncCoordinator == nil {
		return 0, nil
	}
	return uc.syncCoordinator.RunHourlySync(ctx)
}

func (uc *authUseCase) ProcessGooglePushOutbox(ctx context.Context, limit int) (int, error) {
	if uc.pushProcessor == nil {
		return 0, nil
	}
	if limit <= 0 {
		limit = 50
	}
	return uc.pushProcessor.ProcessPending(ctx, limit)
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
		if existing.UserID != userID {
			return fmt.Errorf("google account already linked to another user")
		}
		existing.AccessToken = token.AccessToken
		if token.RefreshToken != "" {
			existing.RefreshToken = token.RefreshToken
		}
		existing.TokenExpiresAt = &token.Expiry
		existing.Status = "active"
		existing.UpdatedAt = now
		return uc.googleAccts.Update(ctx, existing)
	}

	userAccounts, err := uc.googleAccts.FindByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("list google accounts: %w", err)
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
		IsPrimary:      len(userAccounts) == 0,
		ConnectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	return uc.googleAccts.Create(ctx, account)
}

func (uc *authUseCase) syncGoogleAccountByGoogleID(
	ctx context.Context,
	userID, googleID string,
	options portout.GoogleSyncOptions,
) error {
	if uc.googleSyncer == nil {
		return nil
	}

	account, err := uc.googleAccts.FindByGoogleID(ctx, googleID)
	if err != nil {
		return err
	}

	return uc.googleSyncer.SyncAccount(ctx, userID, account, options)
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
	return signJWTToken(token, uc.jwt.Secret)
}

func generateRandomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := randRead(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.URLEncoding.EncodeToString(h[:])
}
