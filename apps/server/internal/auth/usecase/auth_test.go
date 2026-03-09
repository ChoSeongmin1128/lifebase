package usecase

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"lifebase/internal/auth/domain"
	portin "lifebase/internal/auth/port/in"
	portout "lifebase/internal/auth/port/out"
)

type mockUserRepo struct {
	findByEmailUser *domain.User
	findByEmailErr  error
	createErr       error
	updateErr       error
	created         *domain.User
	updated         *domain.User
}

func (m *mockUserRepo) FindByEmail(context.Context, string) (*domain.User, error) {
	return m.findByEmailUser, m.findByEmailErr
}
func (m *mockUserRepo) FindByID(context.Context, string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (m *mockUserRepo) ListUsers(context.Context, string, string, int) ([]*domain.User, string, error) {
	return nil, "", errors.New("not implemented")
}
func (m *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.created = user
	return nil
}
func (m *mockUserRepo) Update(_ context.Context, user *domain.User) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updated = user
	return nil
}
func (m *mockUserRepo) UpdateStorageQuota(context.Context, string, int64) error {
	return errors.New("not implemented")
}
func (m *mockUserRepo) UpdateStorageUsed(context.Context, string, int64) error {
	return errors.New("not implemented")
}

type mockGoogleAccountRepo struct {
	byGoogle       *domain.GoogleAccount
	byGoogleErr    error
	byID           *domain.GoogleAccount
	byIDErr        error
	byUser         []*domain.GoogleAccount
	byUserErr      error
	createErr      error
	updateErr      error
	createdAccount *domain.GoogleAccount
	updatedAccount *domain.GoogleAccount
}

func (m *mockGoogleAccountRepo) FindByGoogleID(context.Context, string) (*domain.GoogleAccount, error) {
	if m.byGoogleErr != nil {
		return nil, m.byGoogleErr
	}
	return m.byGoogle, nil
}
func (m *mockGoogleAccountRepo) FindByID(context.Context, string, string) (*domain.GoogleAccount, error) {
	if m.byIDErr != nil {
		return nil, m.byIDErr
	}
	return m.byID, nil
}
func (m *mockGoogleAccountRepo) FindByUserID(context.Context, string) ([]*domain.GoogleAccount, error) {
	if m.byUserErr != nil {
		return nil, m.byUserErr
	}
	return m.byUser, nil
}
func (m *mockGoogleAccountRepo) Create(_ context.Context, account *domain.GoogleAccount) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdAccount = account
	return nil
}
func (m *mockGoogleAccountRepo) Update(_ context.Context, account *domain.GoogleAccount) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.updatedAccount = account
	return nil
}

type mockRefreshRepo struct {
	createErr       error
	findToken       *domain.RefreshToken
	findErr         error
	deleteByUserErr error
	deleteByHashErr error
	createdToken    *domain.RefreshToken
	deletedByHash   string
	deletedByUser   string
}

func (m *mockRefreshRepo) Create(_ context.Context, token *domain.RefreshToken) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.createdToken = token
	return nil
}
func (m *mockRefreshRepo) FindByHash(context.Context, string) (*domain.RefreshToken, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.findToken, nil
}
func (m *mockRefreshRepo) DeleteByUserID(_ context.Context, userID string) error {
	m.deletedByUser = userID
	return m.deleteByUserErr
}
func (m *mockRefreshRepo) DeleteByHash(_ context.Context, tokenHash string) error {
	m.deletedByHash = tokenHash
	return m.deleteByHashErr
}
func (m *mockRefreshRepo) DeleteExpired(context.Context) error { return nil }

type mockAdminAccessRepo struct {
	ok  bool
	err error
}

func (m *mockAdminAccessRepo) IsActiveAdmin(context.Context, string) (bool, error) {
	return m.ok, m.err
}

type mockGoogleAuthClient struct {
	url           string
	exchangeToken *portout.OAuthToken
	exchangeErr   error
	userInfo      *portout.OAuthUserInfo
	userInfoErr   error
}

func (m *mockGoogleAuthClient) AuthURL(state string) string { return m.url + state }
func (m *mockGoogleAuthClient) AuthURLForApp(state, app string) string {
	return m.url + app + "/" + state
}
func (m *mockGoogleAuthClient) ExchangeCode(context.Context, string) (*portout.OAuthToken, error) {
	return m.exchangeToken, m.exchangeErr
}
func (m *mockGoogleAuthClient) ExchangeCodeForApp(context.Context, string, string) (*portout.OAuthToken, error) {
	return m.exchangeToken, m.exchangeErr
}
func (m *mockGoogleAuthClient) FetchUserInfo(context.Context, portout.OAuthToken) (*portout.OAuthUserInfo, error) {
	return m.userInfo, m.userInfoErr
}
func (m *mockGoogleAuthClient) ListCalendars(context.Context, portout.OAuthToken) ([]portout.OAuthCalendar, error) {
	return nil, nil
}
func (m *mockGoogleAuthClient) ListTaskLists(context.Context, portout.OAuthToken) ([]portout.OAuthTaskList, error) {
	return nil, nil
}
func (m *mockGoogleAuthClient) ListCalendarEvents(context.Context, portout.OAuthToken, string, string, string, *time.Time, *time.Time) (*portout.OAuthCalendarEventsPage, error) {
	return nil, nil
}
func (m *mockGoogleAuthClient) ListTasks(context.Context, portout.OAuthToken, string, string) (*portout.OAuthTasksPage, error) {
	return nil, nil
}
func (m *mockGoogleAuthClient) CreateCalendarEvent(context.Context, portout.OAuthToken, string, portout.CalendarEventUpsertInput) (string, *string, error) {
	return "", nil, nil
}
func (m *mockGoogleAuthClient) UpdateCalendarEvent(context.Context, portout.OAuthToken, string, string, portout.CalendarEventUpsertInput) (*string, error) {
	return nil, nil
}
func (m *mockGoogleAuthClient) DeleteCalendarEvent(context.Context, portout.OAuthToken, string, string) error {
	return nil
}
func (m *mockGoogleAuthClient) CreateTaskList(context.Context, portout.OAuthToken, string) (string, error) {
	return "", nil
}
func (m *mockGoogleAuthClient) DeleteTaskList(context.Context, portout.OAuthToken, string) error {
	return nil
}
func (m *mockGoogleAuthClient) CreateTask(context.Context, portout.OAuthToken, string, portout.TodoUpsertInput) (string, error) {
	return "", nil
}
func (m *mockGoogleAuthClient) UpdateTask(context.Context, portout.OAuthToken, string, string, portout.TodoUpsertInput) error {
	return nil
}
func (m *mockGoogleAuthClient) MoveTask(context.Context, portout.OAuthToken, string, string, *string, *string) error {
	return nil
}
func (m *mockGoogleAuthClient) DeleteTask(context.Context, portout.OAuthToken, string, string) error {
	return nil
}

type mockSyncer struct {
	err    error
	called bool
}

func (m *mockSyncer) SyncAccount(context.Context, string, *domain.GoogleAccount, portout.GoogleSyncOptions) error {
	m.called = true
	return m.err
}

type mockSyncCoordinator struct {
	triggerCount int
	triggerErr   error
	hourlyCount  int
	hourlyErr    error
	area         string
	reason       string
}

func (m *mockSyncCoordinator) TriggerUserSync(_ context.Context, _ string, area, reason string) (int, error) {
	m.area, m.reason = area, reason
	return m.triggerCount, m.triggerErr
}
func (m *mockSyncCoordinator) RunHourlySync(context.Context) (int, error) {
	return m.hourlyCount, m.hourlyErr
}

type mockPushProcessor struct {
	count int
	err   error
	limit int
}

func (m *mockPushProcessor) ProcessPending(context.Context, int) (int, error) {
	return m.count, m.err
}

type mockBootstrapper struct {
	called bool
}

func (m *mockBootstrapper) BootstrapUser(context.Context, string, time.Time) error {
	m.called = true
	return nil
}

func baseUC() *authUseCase {
	return &authUseCase{
		jwt: JWTOptions{
			Secret:        "test-secret",
			AccessExpiry:  time.Hour,
			RefreshExpiry: 24 * time.Hour,
		},
		users:         &mockUserRepo{},
		admins:        &mockAdminAccessRepo{ok: true},
		googleAccts:   &mockGoogleAccountRepo{},
		refreshTokens: &mockRefreshRepo{},
		googleAuth:    &mockGoogleAuthClient{url: "https://auth/"},
	}
}

func TestAuthUseCaseBasics(t *testing.T) {
	uc := baseUC()
	if got := uc.GetAuthURL("state"); !strings.Contains(got, "web/state") {
		t.Fatalf("unexpected auth url: %s", got)
	}
	if got := uc.GetAuthURLForApp("state", "admin"); !strings.Contains(got, "admin/state") {
		t.Fatalf("unexpected app url: %s", got)
	}
}

func TestListGoogleAccounts(t *testing.T) {
	uc := baseUC()
	if _, err := uc.ListGoogleAccounts(context.Background(), ""); err == nil {
		t.Fatal("expected user id validation error")
	}

	repo := uc.googleAccts.(*mockGoogleAccountRepo)
	now := time.Now()
	repo.byUser = []*domain.GoogleAccount{{ID: "a1", GoogleEmail: "a@b.com", Status: "active", IsPrimary: true, ConnectedAt: now}}
	items, err := uc.ListGoogleAccounts(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	repo.byUserErr = errors.New("list failed")
	if _, err := uc.ListGoogleAccounts(context.Background(), "u1"); err == nil {
		t.Fatal("expected list google accounts repo error")
	}
	repo.byUserErr = nil
	if len(items) != 1 || items[0].ID != "a1" {
		t.Fatalf("unexpected accounts: %#v", items)
	}
}

func TestTriggerRunProcessSyncDefaults(t *testing.T) {
	uc := baseUC()

	count, err := uc.TriggerGoogleSync(context.Background(), "u1", portin.TriggerGoogleSyncInput{})
	if err != nil || count != 0 {
		t.Fatalf("expected no-op sync trigger, got count=%d err=%v", count, err)
	}
	count, err = uc.RunHourlyGoogleSync(context.Background())
	if err != nil || count != 0 {
		t.Fatalf("expected no-op hourly sync, got count=%d err=%v", count, err)
	}
	count, err = uc.ProcessGooglePushOutbox(context.Background(), 0)
	if err != nil || count != 0 {
		t.Fatalf("expected no-op push outbox, got count=%d err=%v", count, err)
	}

	coord := &mockSyncCoordinator{triggerCount: 2, hourlyCount: 3}
	uc.syncCoordinator = coord
	count, err = uc.TriggerGoogleSync(context.Background(), "u1", portin.TriggerGoogleSyncInput{})
	if err != nil || count != 2 {
		t.Fatalf("expected trigger count 2, got count=%d err=%v", count, err)
	}
	if coord.area != "both" || coord.reason != "manual" {
		t.Fatalf("expected default area/reason, got %s/%s", coord.area, coord.reason)
	}
	count, err = uc.RunHourlyGoogleSync(context.Background())
	if err != nil || count != 3 {
		t.Fatalf("expected hourly count 3, got count=%d err=%v", count, err)
	}

	processor := &mockPushProcessor{count: 4}
	uc.pushProcessor = processor
	count, err = uc.ProcessGooglePushOutbox(context.Background(), 0)
	if err != nil || count != 4 {
		t.Fatalf("expected processed count 4, got count=%d err=%v", count, err)
	}
}

func TestRefreshAccessTokenAndLogout(t *testing.T) {
	uc := baseUC()
	refreshRepo := uc.refreshTokens.(*mockRefreshRepo)

	refreshRepo.findErr = errors.New("not found")
	if _, err := uc.RefreshAccessToken(context.Background(), "bad"); err == nil {
		t.Fatal("expected invalid refresh token error")
	}
	refreshRepo.findErr = nil

	refreshRepo.findToken = &domain.RefreshToken{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(-time.Hour),
	}
	if _, err := uc.RefreshAccessToken(context.Background(), "expired"); err == nil {
		t.Fatal("expected expired token error")
	}

	refreshRepo.findToken = &domain.RefreshToken{
		UserID:    "u1",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	res, err := uc.RefreshAccessToken(context.Background(), "ok")
	if err != nil {
		t.Fatalf("unexpected refresh error: %v", err)
	}
	if res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("expected issued tokens, got %#v", res)
	}

	if err := uc.Logout(context.Background(), "u1"); err != nil {
		t.Fatalf("unexpected logout error: %v", err)
	}
	if refreshRepo.deletedByUser != "u1" {
		t.Fatalf("expected delete by user u1, got %q", refreshRepo.deletedByUser)
	}
}

func TestFindOrCreateUser(t *testing.T) {
	uc := baseUC()
	users := uc.users.(*mockUserRepo)

	existing := &domain.User{ID: "u1", Email: "a@b.com", Name: "Old"}
	users.findByEmailUser = existing
	info := &portout.OAuthUserInfo{Email: "a@b.com", Name: "New", Picture: "pic"}
	user, err := uc.findOrCreateUser(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != "u1" || users.updated == nil {
		t.Fatalf("expected existing user update, user=%#v updated=%#v", user, users.updated)
	}

	users.findByEmailUser = nil
	users.findByEmailErr = errors.New("not found")
	bootstrap := &mockBootstrapper{}
	uc.bootstrapper = bootstrap
	user, err = uc.findOrCreateUser(context.Background(), info)
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if user.ID == "" || users.created == nil {
		t.Fatalf("expected created user, got %#v", user)
	}
	if !bootstrap.called {
		t.Fatal("expected bootstrap call")
	}
}

func TestUpsertGoogleAccount(t *testing.T) {
	uc := baseUC()
	repo := uc.googleAccts.(*mockGoogleAccountRepo)
	info := &portout.OAuthUserInfo{Email: "a@b.com", GoogleID: "gid-1"}
	token := portout.OAuthToken{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)}

	repo.byGoogle = &domain.GoogleAccount{ID: "g1", UserID: "other", GoogleID: "gid-1"}
	if err := uc.upsertGoogleAccount(context.Background(), "u1", info, token); err == nil {
		t.Fatal("expected cross-user link error")
	}

	repo.byGoogle = &domain.GoogleAccount{ID: "g1", UserID: "u1", GoogleID: "gid-1", RefreshToken: "keep"}
	token.RefreshToken = ""
	if err := uc.upsertGoogleAccount(context.Background(), "u1", info, token); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}
	if repo.updatedAccount == nil || repo.updatedAccount.RefreshToken != "keep" {
		t.Fatalf("expected refresh token preserved, got %#v", repo.updatedAccount)
	}

	repo.byGoogle = nil
	repo.byUser = nil
	token.RefreshToken = "new-refresh"
	if err := uc.upsertGoogleAccount(context.Background(), "u1", info, token); err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if repo.createdAccount == nil || !repo.createdAccount.IsPrimary {
		t.Fatalf("expected primary account create, got %#v", repo.createdAccount)
	}
}

func TestSyncGoogleAccount(t *testing.T) {
	uc := baseUC()
	if err := uc.SyncGoogleAccount(context.Background(), "", "a1", portin.SyncGoogleAccountInput{}); err == nil {
		t.Fatal("expected user validation error")
	}
	if err := uc.SyncGoogleAccount(context.Background(), "u1", "", portin.SyncGoogleAccountInput{}); err == nil {
		t.Fatal("expected account validation error")
	}
	if err := uc.SyncGoogleAccount(context.Background(), "u1", "a1", portin.SyncGoogleAccountInput{}); err == nil {
		t.Fatal("expected syncer missing error")
	}

	repo := uc.googleAccts.(*mockGoogleAccountRepo)
	repo.byIDErr = errors.New("not found")
	uc.googleSyncer = &mockSyncer{}
	if err := uc.SyncGoogleAccount(context.Background(), "u1", "a1", portin.SyncGoogleAccountInput{}); err == nil {
		t.Fatal("expected account not found error")
	}

	repo.byIDErr = nil
	repo.byID = &domain.GoogleAccount{ID: "a1", UserID: "u1"}
	syncer := &mockSyncer{}
	uc.googleSyncer = syncer
	if err := uc.SyncGoogleAccount(context.Background(), "u1", "a1", portin.SyncGoogleAccountInput{SyncCalendar: true}); err != nil {
		t.Fatalf("unexpected sync error: %v", err)
	}
	if !syncer.called {
		t.Fatal("expected sync call")
	}
}

func TestHandleCallbackForAppAndLinkGoogleAccount(t *testing.T) {
	uc := baseUC()
	auth := uc.googleAuth.(*mockGoogleAuthClient)
	users := uc.users.(*mockUserRepo)
	repo := uc.googleAccts.(*mockGoogleAccountRepo)

	auth.exchangeToken = &portout.OAuthToken{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)}
	auth.userInfo = &portout.OAuthUserInfo{GoogleID: "gid", Email: "x@y.com", Name: "Name"}
	users.findByEmailErr = errors.New("not found")

	res, err := uc.HandleCallbackForApp(context.Background(), "code", "web")
	if err != nil {
		t.Fatalf("unexpected callback error: %v", err)
	}
	if res.AccessToken == "" {
		t.Fatalf("expected access token, got %#v", res)
	}

	auth.exchangeErr = errors.New("oauth failed")
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "web"); err == nil {
		t.Fatal("expected oauth exchange error")
	}
	auth.exchangeErr = nil
	auth.userInfoErr = errors.New("userinfo failed")
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "web"); err == nil {
		t.Fatal("expected user info error")
	}

	auth.userInfoErr = nil
	repo.byGoogle = &domain.GoogleAccount{ID: "a1", UserID: "u1", GoogleID: "gid"}
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err != nil {
		t.Fatalf("unexpected link error: %v", err)
	}
	if repo.updatedAccount == nil {
		t.Fatal("expected existing account update")
	}
	if err := uc.LinkGoogleAccount(context.Background(), "", "code", "web"); err == nil {
		t.Fatal("expected user validation error")
	}
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "", "web"); err == nil {
		t.Fatal("expected code validation error")
	}

	users.createErr = errors.New("create failed")
	repo.byGoogle = nil
	users.findByEmailErr = errors.New("not found")
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "web"); err == nil {
		t.Fatal("expected callback create user error")
	}
	users.createErr = nil

	repo.byGoogle = &domain.GoogleAccount{ID: "a1", UserID: "u1", GoogleID: "gid"}
	repo.updateErr = errors.New("update failed")
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected link update error")
	}
	repo.updateErr = nil

	repo.byGoogle = &domain.GoogleAccount{ID: "a1", UserID: "other", GoogleID: "gid"}
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "web"); err == nil {
		t.Fatal("expected callback upsert google account error")
	}

	repo.byGoogle = nil
	repo.byUser = nil
	uc.googleSyncer = &mockSyncer{}
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err != nil {
		t.Fatalf("expected create-and-sync link path, got %v", err)
	}
}

func TestIssueTokenAndHashHelpers(t *testing.T) {
	uc := baseUC()
	refresh := uc.refreshTokens.(*mockRefreshRepo)

	res, err := uc.issueTokens(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected issue token error: %v", err)
	}
	if refresh.createdToken == nil || refresh.createdToken.UserID != "u1" {
		t.Fatalf("expected refresh token persisted, got %#v", refresh.createdToken)
	}
	if res.ExpiresIn != int(uc.jwt.AccessExpiry.Seconds()) {
		t.Fatalf("unexpected expires in: %d", res.ExpiresIn)
	}

	token, err := uc.generateAccessToken("u1")
	if err != nil || token == "" {
		t.Fatalf("expected signed token, got %q err=%v", token, err)
	}

	rnd, err := generateRandomToken()
	if err != nil || rnd == "" {
		t.Fatalf("expected random token, got %q err=%v", rnd, err)
	}
	if hashToken("abc") == hashToken("abcd") {
		t.Fatal("expected different hash outputs")
	}

	refresh.createErr = errors.New("store failed")
	if _, err := uc.issueTokens(context.Background(), "u1"); err == nil {
		t.Fatal("expected store refresh token error")
	}

	refresh.createErr = nil
	prev := randRead
	randRead = func(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
	t.Cleanup(func() { randRead = prev })
	if _, err := uc.issueTokens(context.Background(), "u1"); err == nil {
		t.Fatal("expected random refresh token generation error")
	}

	prevSign := signJWTToken
	signJWTToken = func(token *jwt.Token, secret string) (string, error) { return "", errors.New("sign failed") }
	t.Cleanup(func() { signJWTToken = prevSign })
	randRead = prev
	if _, err := uc.issueTokens(context.Background(), "u1"); err == nil {
		t.Fatal("expected access token generation error")
	}
}

func TestGenerateRandomTokenError(t *testing.T) {
	prev := randRead
	randRead = func(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
	t.Cleanup(func() { randRead = prev })

	if _, err := generateRandomToken(); err == nil {
		t.Fatal("expected generateRandomToken error")
	}
}

func TestAuthUseCaseAdditionalBranches(t *testing.T) {
	uc := baseUC()
	auth := uc.googleAuth.(*mockGoogleAuthClient)
	users := uc.users.(*mockUserRepo)
	repo := uc.googleAccts.(*mockGoogleAccountRepo)

	constructed := NewAuthUseCase(
		uc.jwt,
		uc.users,
		uc.admins,
		uc.googleAccts,
		uc.refreshTokens,
		uc.googleAuth,
		nil,
		nil,
		nil,
		nil,
	)
	if constructed == nil {
		t.Fatal("expected constructor to return usecase instance")
	}

	auth.exchangeToken = &portout.OAuthToken{AccessToken: "a", RefreshToken: "r", Expiry: time.Now().Add(time.Hour)}
	auth.userInfo = &portout.OAuthUserInfo{GoogleID: "gid", Email: "x@y.com", Name: "Name"}
	users.findByEmailErr = errors.New("not found")
	if _, err := uc.HandleCallback(context.Background(), "code"); err != nil {
		t.Fatalf("HandleCallback wrapper failed: %v", err)
	}

	admins := uc.admins.(*mockAdminAccessRepo)
	admins.ok = false
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "admin"); !errors.Is(err, portin.ErrAdminAccessDenied) {
		t.Fatalf("expected admin access denied, got %v", err)
	}
	admins.ok = true
	users.findByEmailUser = nil
	users.findByEmailErr = errors.New("not found")
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "admin"); !errors.Is(err, portin.ErrAdminAccessDenied) {
		t.Fatalf("expected admin access denied when user missing, got %v", err)
	}
	users.findByEmailErr = nil
	users.findByEmailUser = &domain.User{ID: "u-admin", Email: "x@y.com", Name: "Name"}
	admins.err = errors.New("admin lookup failed")
	if _, err := uc.HandleCallbackForApp(context.Background(), "code", "admin"); !errors.Is(err, portin.ErrAdminAccessCheckFailed) {
		t.Fatalf("expected admin access check failure, got %v", err)
	}
	admins.err = nil

	auth.exchangeErr = errors.New("exchange failed")
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected link oauth exchange error")
	}
	auth.exchangeErr = nil
	auth.userInfoErr = errors.New("userinfo failed")
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected link userinfo error")
	}
	auth.userInfoErr = nil

	repo.byGoogle = &domain.GoogleAccount{ID: "g1", UserID: "other", GoogleID: "gid"}
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected cross-user link error")
	}

	repo.byGoogle = nil
	repo.byUserErr = errors.New("list failed")
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected list google accounts error")
	}
	repo.byUserErr = nil
	repo.createErr = errors.New("create failed")
	if err := uc.LinkGoogleAccount(context.Background(), "u1", "code", "web"); err == nil {
		t.Fatal("expected link create error")
	}

	uc.googleSyncer = nil
	if err := uc.syncGoogleAccountByGoogleID(context.Background(), "u1", "gid", portout.GoogleSyncOptions{}); err != nil {
		t.Fatalf("expected nil syncer no-op, got %v", err)
	}
	uc.googleSyncer = &mockSyncer{err: errors.New("sync failed")}
	repo.byGoogle = nil
	if err := uc.syncGoogleAccountByGoogleID(context.Background(), "u1", "gid", portout.GoogleSyncOptions{}); err == nil {
		t.Fatal("expected sync failure when account lookup fails")
	}
	repo.byGoogle = &domain.GoogleAccount{ID: "g1", UserID: "u1", GoogleID: "gid"}
	if err := uc.syncGoogleAccountByGoogleID(context.Background(), "u1", "gid", portout.GoogleSyncOptions{}); err == nil {
		t.Fatal("expected syncer error")
	}

	repo.byGoogleErr = errors.New("lookup failed")
	if err := uc.syncGoogleAccountByGoogleID(context.Background(), "u1", "gid", portout.GoogleSyncOptions{}); err == nil {
		t.Fatal("expected google account lookup error")
	}
	repo.byGoogleErr = nil

	existing := &domain.GoogleAccount{ID: "g2", UserID: "u1", GoogleID: "gid", RefreshToken: "keep"}
	repo.byGoogle = existing
	if err := uc.upsertGoogleAccount(context.Background(), "u1", &portout.OAuthUserInfo{GoogleID: "gid", Email: "x@y.com"}, portout.OAuthToken{
		AccessToken: "new-at",
		Expiry:      time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("unexpected upsert without refresh token: %v", err)
	}
	if existing.RefreshToken != "keep" {
		t.Fatalf("expected refresh token to be preserved, got %q", existing.RefreshToken)
	}

	if err := uc.upsertGoogleAccount(context.Background(), "u1", &portout.OAuthUserInfo{GoogleID: "gid", Email: "x@y.com"}, portout.OAuthToken{
		AccessToken:  "new-at-2",
		RefreshToken: "new-rt",
		Expiry:       time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("unexpected upsert with refresh token: %v", err)
	}
	if existing.RefreshToken != "new-rt" {
		t.Fatalf("expected refresh token update, got %q", existing.RefreshToken)
	}

	repo.byGoogle = nil
	repo.byUserErr = errors.New("list failed")
	if err := uc.upsertGoogleAccount(context.Background(), "u1", &portout.OAuthUserInfo{GoogleID: "gid", Email: "x@y.com"}, portout.OAuthToken{
		AccessToken: "new-at",
		Expiry:      time.Now().Add(time.Hour),
	}); err == nil {
		t.Fatal("expected upsert list accounts error")
	}
}

func TestEnsureAdminLoginAllowed(t *testing.T) {
	ctx := context.Background()
	userInfo := &portout.OAuthUserInfo{Email: "admin@example.com"}

	t.Run("non_admin_app_bypasses_check", func(t *testing.T) {
		uc := baseUC()
		if err := uc.ensureAdminLoginAllowed(ctx, "web", userInfo); err != nil {
			t.Fatalf("expected non-admin app to bypass check, got %v", err)
		}
	})

	t.Run("missing_admin_repository", func(t *testing.T) {
		uc := baseUC()
		uc.admins = nil
		if err := uc.ensureAdminLoginAllowed(ctx, "admin", userInfo); !errors.Is(err, portin.ErrAdminAccessCheckFailed) {
			t.Fatalf("expected admin access check failed, got %v", err)
		}
	})

	t.Run("missing_user_without_repo_error", func(t *testing.T) {
		uc := baseUC()
		users := uc.users.(*mockUserRepo)
		users.findByEmailUser = nil
		users.findByEmailErr = nil
		if err := uc.ensureAdminLoginAllowed(ctx, "admin", userInfo); !errors.Is(err, portin.ErrAdminAccessDenied) {
			t.Fatalf("expected admin access denied, got %v", err)
		}
	})

	t.Run("existing_admin_passes", func(t *testing.T) {
		uc := baseUC()
		users := uc.users.(*mockUserRepo)
		admins := uc.admins.(*mockAdminAccessRepo)
		users.findByEmailUser = &domain.User{ID: "u-admin", Email: userInfo.Email}
		admins.ok = true
		if err := uc.ensureAdminLoginAllowed(ctx, "admin", userInfo); err != nil {
			t.Fatalf("expected admin access allowed, got %v", err)
		}
	})

	t.Run("existing_non_admin_is_denied", func(t *testing.T) {
		uc := baseUC()
		users := uc.users.(*mockUserRepo)
		admins := uc.admins.(*mockAdminAccessRepo)
		users.findByEmailUser = &domain.User{ID: "u-admin", Email: userInfo.Email}
		admins.ok = false
		if err := uc.ensureAdminLoginAllowed(ctx, "admin", userInfo); !errors.Is(err, portin.ErrAdminAccessDenied) {
			t.Fatalf("expected admin access denied, got %v", err)
		}
	})
}

type errorReader struct{}
