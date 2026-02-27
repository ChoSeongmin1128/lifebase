package http

import (
	"encoding/json"
	"net/http"

	portin "lifebase/internal/settings/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type SettingsHandler struct {
	settings portin.SettingsUseCase
}

func NewSettingsHandler(settings portin.SettingsUseCase) *SettingsHandler {
	return &SettingsHandler{settings: settings}
}

func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	settings, err := h.settings.GetAll(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "GET_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"settings": settings})
}

func (h *SettingsHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.settings.Update(r.Context(), userID, req); err != nil {
		response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}
