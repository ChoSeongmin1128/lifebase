package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	admindomain "lifebase/internal/admin/domain"
	portin "lifebase/internal/admin/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type AdminHandler struct {
	admin portin.AdminUseCase
}

func NewAdminHandler(admin portin.AdminUseCase) *AdminHandler {
	return &AdminHandler{admin: admin}
}

func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	search := r.URL.Query().Get("q")
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)

	users, nextCursor, err := h.admin.ListUsers(r.Context(), actorID, search, cursor, limit)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"users":       users,
		"next_cursor": nextCursor,
	})
}

func (h *AdminHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	userID := chi.URLParam(r, "userID")

	detail, err := h.admin.GetUserDetail(r.Context(), actorID, userID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"user": detail.User, "google_accounts": detail.GoogleAccounts})
}

func (h *AdminHandler) UpdateQuota(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	userID := chi.URLParam(r, "userID")

	var req struct {
		QuotaBytes int64 `json:"quota_bytes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.UpdateStorageQuota(r.Context(), actorID, userID, req.QuotaBytes); err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *AdminHandler) RecalculateStorage(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	userID := chi.URLParam(r, "userID")

	used, err := h.admin.RecalculateStorageUsed(r.Context(), actorID, userID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"storage_used_bytes": used})
}

func (h *AdminHandler) ResetStorage(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	userID := chi.URLParam(r, "userID")

	var req struct {
		Confirm string `json:"confirm"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.ResetUserStorage(r.Context(), actorID, userID, req.Confirm); err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *AdminHandler) UpdateGoogleAccountStatus(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	userID := chi.URLParam(r, "userID")
	accountID := chi.URLParam(r, "accountID")

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.UpdateGoogleAccountStatus(r.Context(), actorID, userID, accountID, req.Status); err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *AdminHandler) ListAdmins(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	admins, err := h.admin.ListAdmins(r.Context(), actorID)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"admins": admins})
}

func (h *AdminHandler) CreateAdmin(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	role := admindomain.Role(req.Role)
	if role == "" {
		role = admindomain.RoleAdmin
	}
	admin, err := h.admin.CreateAdmin(r.Context(), actorID, req.Email, role)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]any{"admin": admin})
}

func (h *AdminHandler) UpdateAdminRole(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	adminID := chi.URLParam(r, "adminID")

	var req struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if err := h.admin.UpdateAdminRole(r.Context(), actorID, adminID, admindomain.Role(req.Role)); err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.NoContent(w)
}

func (h *AdminHandler) DeactivateAdmin(w http.ResponseWriter, r *http.Request) {
	actorID := middleware.GetUserID(r.Context())
	adminID := chi.URLParam(r, "adminID")

	if err := h.admin.DeactivateAdmin(r.Context(), actorID, adminID); err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.NoContent(w)
}

func parseLimit(raw string, def, max int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func writeUseCaseError(w http.ResponseWriter, err error) {
	msg := err.Error()
	switch {
	case strings.Contains(msg, "denied"), strings.Contains(msg, "required"), strings.Contains(msg, "cannot change own"), strings.Contains(msg, "cannot deactivate yourself"):
		response.Error(w, http.StatusForbidden, "FORBIDDEN", msg)
	case strings.Contains(msg, "not found"):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", msg)
	case strings.Contains(msg, "invalid"), strings.Contains(msg, "must"), strings.Contains(msg, "cannot"), strings.Contains(msg, "mismatch"):
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", msg)
	default:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
