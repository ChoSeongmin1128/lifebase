package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/auth/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/oauthstate"
)

type mockAuthUC struct {
	callbackResult *portin.LoginResult
	callbackErr    error
	refreshResult  *portin.LoginResult
	refreshErr     error
	accounts       []portin.GoogleAccountSummary
	accountsErr    error
	linkErr        error
	syncErr        error
	triggerCount   int
	triggerErr     error
	logoutErr      error
}

func (m *mockAuthUC) GetAuthURL(state string) string { return "unused://" + state }
func (m *mockAuthUC) GetAuthURLForApp(state, app string) string {
	return "https://auth.example.com/" + app + "?state=" + state
}
func (m *mockAuthUC) HandleCallback(context.Context, string) (*portin.LoginResult, error) {
	return m.callbackResult, m.callbackErr
}
func (m *mockAuthUC) HandleCallbackForApp(context.Context, string, string) (*portin.LoginResult, error) {
	return m.callbackResult, m.callbackErr
}
func (m *mockAuthUC) ListGoogleAccounts(context.Context, string) ([]portin.GoogleAccountSummary, error) {
	return m.accounts, m.accountsErr
}
func (m *mockAuthUC) LinkGoogleAccount(context.Context, string, string, string) error {
	return m.linkErr
}
func (m *mockAuthUC) SyncGoogleAccount(context.Context, string, string, portin.SyncGoogleAccountInput) error {
	return m.syncErr
}
func (m *mockAuthUC) TriggerGoogleSync(context.Context, string, portin.TriggerGoogleSyncInput) (int, error) {
	return m.triggerCount, m.triggerErr
}
func (m *mockAuthUC) RunHourlyGoogleSync(context.Context) (int, error) {
	return 0, nil
}
func (m *mockAuthUC) ProcessGooglePushOutbox(context.Context, int) (int, error) {
	return 0, nil
}
func (m *mockAuthUC) RefreshAccessToken(context.Context, string) (*portin.LoginResult, error) {
	return m.refreshResult, m.refreshErr
}
func (m *mockAuthUC) Logout(context.Context, string) error { return m.logoutErr }

func authReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func authReqWithParam(method, target, body, key, value string) *http.Request {
	req := authReq(method, target, body)
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func newTestAuthHandler(uc *mockAuthUC) *AuthHandler {
	return NewAuthHandler(uc, "test-key", SessionCookieConfig{
		AccessExpiry:  time.Hour,
		RefreshExpiry: 24 * time.Hour,
	})
}

func TestAuthHandlerGetAuthURL(t *testing.T) {
	h := newTestAuthHandler(&mockAuthUC{})

	rec := httptest.NewRecorder()
	h.GetAuthURL(rec, authReq(http.MethodGet, "/auth/url?app=web", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.GetAuthURL(rec, authReq(http.MethodGet, "/auth/url?app=mobile", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid app, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.GetAuthURL(rec, authReq(http.MethodGet, "/auth/url", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 when app is omitted (default web), got %d", rec.Code)
	}
}

func TestAuthHandlerHandleCallback(t *testing.T) {
	uc := &mockAuthUC{callbackResult: &portin.LoginResult{AccessToken: "a", RefreshToken: "r", ExpiresIn: 3600}}
	h := newTestAuthHandler(uc)
	validState, _ := oauthstate.Generate("web", "test-key")
	validAdminState, _ := oauthstate.Generate("admin", "test-key")

	rec := httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{"code":"ok","state":"`+validState+`"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 2 {
		t.Fatalf("expected 2 session cookies, got %d", len(cookies))
	}

	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{"code":"ok","state":"bad"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid state, got %d", rec.Code)
	}

	uc.callbackErr = errors.New("auth failed")
	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{"code":"ok","state":"`+validAdminState+`"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	uc.callbackErr = portin.ErrAdminAccessDenied
	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{"code":"ok","state":"`+validAdminState+`"}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	uc.callbackErr = portin.ErrAdminAccessCheckFailed
	rec = httptest.NewRecorder()
	h.HandleCallback(rec, authReq(http.MethodPost, "/auth/callback", `{"code":"ok","state":"`+validAdminState+`"}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestAuthHandlerRefreshToken(t *testing.T) {
	uc := &mockAuthUC{refreshResult: &portin.LoginResult{AccessToken: "a", RefreshToken: "r", ExpiresIn: 3600}}
	h := newTestAuthHandler(uc)

	rec := httptest.NewRecorder()
	h.RefreshToken(rec, authReq(http.MethodPost, "/auth/refresh", `{"refresh_token":"token"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.RefreshToken(rec, authReq(http.MethodPost, "/auth/refresh", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.RefreshToken(rec, authReq(http.MethodPost, "/auth/refresh", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := authReq(http.MethodPost, "/auth/refresh", `{"app":"web"}`)
	req.AddCookie(&http.Cookie{Name: "lifebase_refresh_token", Value: "cookie-refresh"})
	h.RefreshToken(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 with refresh cookie, got %d", rec.Code)
	}
	if len(rec.Result().Cookies()) != 2 {
		t.Fatalf("expected 2 session cookies on cookie refresh, got %d", len(rec.Result().Cookies()))
	}

	uc.refreshErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.RefreshToken(rec, authReq(http.MethodPost, "/auth/refresh", `{"refresh_token":"token"}`))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuthHandlerGoogleAccountFlows(t *testing.T) {
	now := time.Now()
	uc := &mockAuthUC{
		accounts: []portin.GoogleAccountSummary{{ID: "g1", GoogleEmail: "a@b.com", ConnectedAt: now}},
	}
	h := newTestAuthHandler(uc)
	validState, _ := oauthstate.Generate("admin", "test-key")

	rec := httptest.NewRecorder()
	h.GetGoogleAccounts(rec, authReq(http.MethodGet, "/auth/google-accounts", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.accountsErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.GetGoogleAccounts(rec, authReq(http.MethodGet, "/auth/google-accounts", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	uc.accountsErr = nil
	rec = httptest.NewRecorder()
	h.LinkGoogleAccount(rec, authReq(http.MethodPost, "/auth/google-accounts/link", `{"code":"ok","state":"`+validState+`"}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.LinkGoogleAccount(rec, authReq(http.MethodPost, "/auth/google-accounts/link", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.LinkGoogleAccount(rec, authReq(http.MethodPost, "/auth/google-accounts/link", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.LinkGoogleAccount(rec, authReq(http.MethodPost, "/auth/google-accounts/link", `{"code":"ok","state":"bad"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 invalid state, got %d", rec.Code)
	}
	uc.linkErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.LinkGoogleAccount(rec, authReq(http.MethodPost, "/auth/google-accounts/link", `{"code":"ok","state":"`+validState+`"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAuthHandlerSyncTriggerAndLogout(t *testing.T) {
	uc := &mockAuthUC{triggerCount: 2}
	h := newTestAuthHandler(uc)

	rec := httptest.NewRecorder()
	req := authReqWithParam(http.MethodPost, "/auth/google-accounts/acc/sync", `{"sync_calendar":true}`, "accountID", "acc")
	h.SyncGoogleAccount(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = authReqWithParam(http.MethodPost, "/auth/google-accounts//sync", `{}`, "accountID", "")
	h.SyncGoogleAccount(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = authReqWithParam(http.MethodPost, "/auth/google-accounts/acc/sync", `{`, "accountID", "acc")
	h.SyncGoogleAccount(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.syncErr = errors.New("fail")
	rec = httptest.NewRecorder()
	req = authReqWithParam(http.MethodPost, "/auth/google-accounts/acc/sync", `{"sync_calendar":true}`, "accountID", "acc")
	h.SyncGoogleAccount(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.TriggerGoogleSync(rec, authReq(http.MethodPost, "/auth/google-sync/trigger", `{"area":"calendar","reason":"manual"}`))
	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.TriggerGoogleSync(rec, authReq(http.MethodPost, "/auth/google-sync/trigger", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.triggerErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.TriggerGoogleSync(rec, authReq(http.MethodPost, "/auth/google-sync/trigger", `{"area":"calendar","reason":"manual"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = authReq(http.MethodPost, "/auth/logout", "")
	req = req.WithContext(context.WithValue(req.Context(), middleware.AuthAppKey, "admin"))
	h.Logout(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	foundClearedAdmin := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "lifebase_admin_access_token" && cookie.MaxAge < 0 {
			foundClearedAdmin = true
		}
	}
	if !foundClearedAdmin {
		t.Fatal("expected cleared admin session cookie")
	}
	uc.logoutErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.Logout(rec, authReq(http.MethodPost, "/auth/logout", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
