package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"

	portin "lifebase/internal/auth/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type AuthHandler struct {
	auth portin.AuthUseCase
}

func NewAuthHandler(auth portin.AuthUseCase) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) GetAuthURL(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	url := h.auth.GetAuthURL(state)
	response.JSON(w, http.StatusOK, map[string]string{
		"url":   url,
		"state": state,
	})
}

func (h *AuthHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if req.Code == "" {
		response.Error(w, http.StatusBadRequest, "MISSING_CODE", "authorization code is required")
		return
	}

	result, err := h.auth.HandleCallback(r.Context(), req.Code)
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

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
