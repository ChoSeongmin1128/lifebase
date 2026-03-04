package http

import (
	"encoding/json"
	"net/http"
	"time"

	portin "lifebase/internal/holiday/port/in"
	"lifebase/internal/shared/response"
)

type HolidayHandler struct {
	useCase portin.HolidayUseCase
}

func NewHolidayHandler(useCase portin.HolidayUseCase) *HolidayHandler {
	return &HolidayHandler{useCase: useCase}
}

func (h *HolidayHandler) ListHolidays(w http.ResponseWriter, r *http.Request) {
	startRaw := r.URL.Query().Get("start")
	endRaw := r.URL.Query().Get("end")
	if startRaw == "" || endRaw == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "start and end are required")
		return
	}

	start, err := time.Parse("2006-01-02", startRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid start format")
		return
	}
	end, err := time.Parse("2006-01-02", endRaw)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid end format")
		return
	}
	end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	holidays, err := h.useCase.ListRange(r.Context(), start, end)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "LIST_FAILED", err.Error())
		return
	}

	type holidayItem struct {
		Date string `json:"date"`
		Name string `json:"name"`
	}
	out := make([]holidayItem, 0, len(holidays))
	for _, item := range holidays {
		out = append(out, holidayItem{Date: item.Date.Format("2006-01-02"), Name: item.Name})
	}
	response.JSON(w, http.StatusOK, map[string]any{"holidays": out})
}

func (h *HolidayHandler) RefreshHolidays(w http.ResponseWriter, r *http.Request) {
	var input portin.RefreshRangeInput
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
			return
		}
	}

	result, err := h.useCase.RefreshRange(r.Context(), input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "REFRESH_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, result)
}
