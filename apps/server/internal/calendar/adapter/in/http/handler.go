package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/calendar/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type CalendarHandler struct {
	cal portin.CalendarUseCase
}

func NewCalendarHandler(cal portin.CalendarUseCase) *CalendarHandler {
	return &CalendarHandler{cal: cal}
}

// Calendars

func (h *CalendarHandler) CreateCalendar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		Name    string  `json:"name"`
		ColorID *string `json:"color_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	cal, err := h.cal.CreateCalendar(r.Context(), userID, req.Name, req.ColorID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, cal)
}

func (h *CalendarHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	calendars, err := h.cal.ListCalendars(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"calendars": calendars})
}

func (h *CalendarHandler) UpdateCalendar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	calID := chi.URLParam(r, "calendarID")
	var req struct {
		Name      string  `json:"name"`
		ColorID   *string `json:"color_id"`
		IsVisible *bool   `json:"is_visible"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	if err := h.cal.UpdateCalendar(r.Context(), userID, calID, req.Name, req.ColorID, req.IsVisible); err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *CalendarHandler) DeleteCalendar(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	calID := chi.URLParam(r, "calendarID")

	if err := h.cal.DeleteCalendar(r.Context(), userID, calID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

// Events

func (h *CalendarHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var input portin.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if input.CalendarID == "" || input.StartTime == "" || input.EndTime == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "calendar_id, start_time, end_time are required")
		return
	}

	event, err := h.cal.CreateEvent(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, event)
}

func (h *CalendarHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "eventID")

	event, err := h.cal.GetEvent(r.Context(), userID, eventID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "event not found")
		return
	}
	response.JSON(w, http.StatusOK, event)
}

func (h *CalendarHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	start := r.URL.Query().Get("start")
	end := r.URL.Query().Get("end")
	calIDs := r.URL.Query().Get("calendar_ids")

	if start == "" || end == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start and end are required")
		return
	}

	var calendarIDs []string
	if calIDs != "" {
		calendarIDs = strings.Split(calIDs, ",")
	}

	events, err := h.cal.ListEvents(r.Context(), userID, calendarIDs, start, end)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *CalendarHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "eventID")
	var input portin.UpdateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	event, err := h.cal.UpdateEvent(r.Context(), userID, eventID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, event)
}

func (h *CalendarHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "eventID")

	if err := h.cal.DeleteEvent(r.Context(), userID, eventID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}
