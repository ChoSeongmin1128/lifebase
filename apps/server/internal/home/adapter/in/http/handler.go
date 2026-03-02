package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	portin "lifebase/internal/home/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type HomeHandler struct {
	home portin.HomeUseCase
}

func NewHomeHandler(home portin.HomeUseCase) *HomeHandler {
	return &HomeHandler{home: home}
}

func (h *HomeHandler) GetSummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	startRaw := r.URL.Query().Get("start")
	endRaw := r.URL.Query().Get("end")
	if startRaw == "" || endRaw == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start and end are required")
		return
	}

	start, err := time.Parse(time.RFC3339, startRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start must be RFC3339")
		return
	}
	end, err := time.Parse(time.RFC3339, endRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "end must be RFC3339")
		return
	}
	if !start.Before(end) {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start must be before end")
		return
	}

	input := portin.GetSummaryInput{
		Start:       start,
		End:         end,
		EventLimit:  parseLimit(r.URL.Query().Get("event_limit"), 5, 20),
		TodoLimit:   parseLimit(r.URL.Query().Get("todo_limit"), 7, 30),
		RecentLimit: parseLimit(r.URL.Query().Get("recent_limit"), 8, 30),
	}

	summary, err := h.home.GetSummary(r.Context(), userID, input)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
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
	case strings.Contains(msg, "required"), strings.Contains(msg, "before"):
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", msg)
	case strings.Contains(msg, "not found"):
		response.Error(w, http.StatusNotFound, "NOT_FOUND", msg)
	default:
		response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	}
}
