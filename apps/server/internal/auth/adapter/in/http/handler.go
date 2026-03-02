package http

import (
	"encoding/json"
	"net/http"

	portin "lifebase/internal/auth/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/oauthstate"
	"lifebase/internal/shared/response"
)

type AuthHandler struct {
	auth         portin.AuthUseCase
	stateHMACKey string
}

func NewAuthHandler(auth portin.AuthUseCase, stateHMACKey string) *AuthHandler {
	return &AuthHandler{
		auth:         auth,
		stateHMACKey: stateHMACKey,
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

	app := "web"
	if req.State != "" {
		verifiedApp, err := oauthstate.Verify(req.State, h.stateHMACKey)
		if err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_STATE", "invalid oauth state")
			return
		}
		app = verifiedApp
	} else if req.App == "admin" {
		// Backward-compatible fallback for gradual client rollout.
		app = "admin"
	}

	result, err := h.auth.HandleCallbackForApp(r.Context(), req.Code, app)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "AUTH_FAILED", "authentication failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_TOKEN", "refresh token is required")
		return
	}

	result, err := h.auth.RefreshAccessToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "REFRESH_FAILED", "token refresh failed")
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_in":    result.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	if err := h.auth.Logout(r.Context(), userID); err != nil {
		response.Error(w, http.StatusInternalServerError, "LOGOUT_FAILED", "logout failed")
		return
	}
	response.NoContent(w)
}
