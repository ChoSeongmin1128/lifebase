package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/calendar/domain"
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
		if errors.Is(err, domain.ErrReadOnlyCalendar) {
			response.Error(w, http.StatusForbidden, "READ_ONLY_CALENDAR", "read-only calendar")
			return
		}
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, event)
}

func (h *CalendarHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "eventID")
	if eventID == "day-summary" {
		h.GetDaySummary(w, r)
		return
	}

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

func (h *CalendarHandler) GetDaySummary(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "date is required")
		return
	}

	tz := strings.TrimSpace(r.URL.Query().Get("tz"))
	calIDsRaw := strings.TrimSpace(r.URL.Query().Get("calendar_ids"))
	calendarIDs := make([]string, 0)
	if calIDsRaw != "" {
		for _, item := range strings.Split(calIDsRaw, ",") {
			trimmed := strings.TrimSpace(item)
			if trimmed != "" {
				calendarIDs = append(calendarIDs, trimmed)
			}
		}
	}

	includeDoneTodos := false
	includeDoneRaw := strings.TrimSpace(r.URL.Query().Get("include_done_todos"))
	if includeDoneRaw != "" {
		if includeDoneRaw == "true" {
			includeDoneTodos = true
		} else if includeDoneRaw != "false" {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "include_done_todos must be true or false")
			return
		}
	}

	result, err := h.cal.GetDaySummary(r.Context(), userID, portin.DaySummaryInput{
		Date:             date,
		Timezone:         tz,
		CalendarIDs:      calendarIDs,
		IncludeDoneTodos: includeDoneTodos,
	})
	if err != nil {
		response.Error(w, http.StatusBadRequest, "SUMMARY_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (h *CalendarHandler) BackfillEvents(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var input portin.BackfillEventsInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if input.Start == "" || input.End == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start and end are required")
		return
	}

	result, err := h.cal.BackfillEvents(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "BACKFILL_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
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
		if errors.Is(err, domain.ErrReadOnlyCalendar) {
			response.Error(w, http.StatusForbidden, "READ_ONLY_CALENDAR", "read-only calendar")
			return
		}
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, event)
}

func (h *CalendarHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	eventID := chi.URLParam(r, "eventID")

	if err := h.cal.DeleteEvent(r.Context(), userID, eventID); err != nil {
		if errors.Is(err, domain.ErrReadOnlyCalendar) {
			response.Error(w, http.StatusForbidden, "READ_ONLY_CALENDAR", "read-only calendar")
			return
		}
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}
