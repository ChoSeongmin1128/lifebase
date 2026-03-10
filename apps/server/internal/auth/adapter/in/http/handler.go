package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/auth/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/oauthstate"
	"lifebase/internal/shared/response"
)

type AuthHandler struct {
	auth         portin.AuthUseCase
	stateHMACKey string
	cookies      SessionCookieConfig
}

type SessionCookieConfig struct {
	AccessExpiry  time.Duration
	RefreshExpiry time.Duration
	Secure        bool
}

func NewAuthHandler(auth portin.AuthUseCase, stateHMACKey string, cookies SessionCookieConfig) *AuthHandler {
	return &AuthHandler{
		auth:         auth,
		stateHMACKey: stateHMACKey,
		cookies:      cookies,
	}
}

func (h *AuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	app := r.URL.Query().Get("app")
	if app == "" {
		app = "web"
	}

	state, err := oauthstate.Generate(app, h.stateHMACKey)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_APP", "invalid app")
		return
	}

	url := h.auth.GetAuthURLForApp(state, app)
	response.JSON(w, http.StatusOK, map[string]string{
		"url":   url,
		"state": state,
	})
}

func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code  string `json:"code"`
		State string `json:"state"`
		App   string `json:"app"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CODE", "authorization code is required")
		return
	}
	if req.State == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_STATE", "invalid oauth state")
		return
	}

	app, err := oauthstate.Verify(req.State, h.stateHMACKey)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_STATE", "invalid oauth state")
		return
	}

	result, err := h.auth.HandleCallbackForApp(r.Context(), req.Code, app)
	if err != nil {
		if errors.Is(err, portin.ErrAdminAccessDenied) {
			response.Error(w, http.StatusForbidden, "ADMIN_FORBIDDEN", "admin access denied")
			return
		}
		if errors.Is(err, portin.ErrAdminAccessCheckFailed) {
			response.Error(w, http.StatusInternalServerError, "ADMIN_CHECK_FAILED", "admin check failed")
			return
		}
		response.Error(w, http.StatusUnauthorized, "AUTH_FAILED", "authentication failed")
		return
	}
	h.setSessionCookies(w, app, result)

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
		App          string `json:"app"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	refreshToken := req.RefreshToken
	if refreshToken == "" && (req.App == "web" || req.App == "admin") {
		if cookie, err := r.Cookie(refreshCookieName(req.App)); err == nil {
			refreshToken = cookie.Value
		}
	}
	if refreshToken == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_TOKEN", "refresh token is required")
		return
	}

	result, err := h.auth.RefreshAccessToken(r.Context(), refreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "REFRESH_FAILED", "token refresh failed")
		return
	}
	h.setSessionCookies(w, req.App, result)

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

func (h *AuthHandler) GetGoogleAccounts(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	accounts, err := h.auth.ListGoogleAccounts(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", "failed to list google accounts")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"accounts": accounts,
	})
}

func (h *AuthHandler) LinkGoogleAccount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		Code  string `json:"code"`
		State string `json:"state"`
		App   string `json:"app"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CODE", "authorization code is required")
		return
	}
	if req.State == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_STATE", "invalid oauth state")
		return
	}

	app, err := oauthstate.Verify(req.State, h.stateHMACKey)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_STATE", "invalid oauth state")
		return
	}

	if err := h.auth.LinkGoogleAccount(r.Context(), userID, req.Code, app); err != nil {
		response.Error(w, http.StatusBadRequest, "LINK_FAILED", "failed to link google account")
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) SyncGoogleAccount(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	accountID := chi.URLParam(r, "accountID")
	if accountID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "account id is required")
		return
	}

	var req portin.SyncGoogleAccountInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.auth.SyncGoogleAccount(r.Context(), userID, accountID, req); err != nil {
		response.Error(w, http.StatusBadRequest, "SYNC_FAILED", "failed to sync google account")
		return
	}

	response.NoContent(w)
}

func (h *AuthHandler) TriggerGoogleSync(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req portin.TriggerGoogleSyncInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	count, err := h.auth.TriggerGoogleSync(r.Context(), userID, req)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "SYNC_TRIGGER_FAILED", "failed to trigger google sync")
		return
	}

	response.JSON(w, http.StatusAccepted, map[string]any{
		"scheduled_accounts": count,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if err := h.auth.Logout(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "LOGOUT_FAILED", "logout failed")
		return
	}
	h.clearSessionCookies(w, middleware.GetAuthApp(r.Context()))
	response.NoContent(w)
}

func (h *AuthHandler) setSessionCookies(w http.ResponseWriter, app string, result *portin.LoginResult) {
	if app != "web" && app != "admin" {
		return
	}

	now := time.Now()
	http.SetCookie(w, &http.Cookie{
		Name:     accessCookieName(app),
		Value:    result.AccessToken,
		Path:     "/api/v1",
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  now.Add(time.Duration(result.ExpiresIn) * time.Second),
		MaxAge:   result.ExpiresIn,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName(app),
		Value:    result.RefreshToken,
		Path:     "/api/v1",
		HttpOnly: true,
		Secure:   h.cookies.Secure,
		SameSite: http.SameSiteLaxMode,
		Expires:  now.Add(h.cookies.RefreshExpiry),
		MaxAge:   int(h.cookies.RefreshExpiry.Seconds()),
	})
}

func (h *AuthHandler) clearSessionCookies(w http.ResponseWriter, app string) {
	clearCookie := func(name string) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/api/v1",
			HttpOnly: true,
			Secure:   h.cookies.Secure,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}

	switch app {
	case "admin":
		clearCookie(accessCookieName("admin"))
		clearCookie(refreshCookieName("admin"))
	case "web":
		clearCookie(accessCookieName("web"))
		clearCookie(refreshCookieName("web"))
	default:
		clearCookie(accessCookieName("web"))
		clearCookie(refreshCookieName("web"))
		clearCookie(accessCookieName("admin"))
		clearCookie(refreshCookieName("admin"))
	}
}

func accessCookieName(app string) string {
	if app == "admin" {
		return "lifebase_admin_access_token"
	}
	return "lifebase_access_token"
}

func refreshCookieName(app string) string {
	if app == "admin" {
		return "lifebase_admin_refresh_token"
	}
	return "lifebase_refresh_token"
}
