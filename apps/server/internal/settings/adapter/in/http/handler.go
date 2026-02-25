package http

import (
	"encoding/json"
	"net/http"

	settingspg "lifebase/internal/settings/adapter/out/postgres"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type SettingsHandler struct {
	repo *settingspg.SettingsRepo
}

func NewSettingsHandler(repo *settingspg.SettingsRepo) *SettingsHandler {
	return &SettingsHandler{repo: repo}
}

func (h *SettingsHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	settings, err := h.repo.GetAll(r.Context(), userID)
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

	for key, value := range req {
		if err := h.repo.Set(r.Context(), userID, key, value); err != nil {
			response.Error(w, http.StatusInternalServerError, "UPDATE_FAILED", err.Error())
			return
		}
	}
	response.NoContent(w)
}
