package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/calendar/domain"
	portin "lifebase/internal/calendar/port/in"
	"lifebase/internal/shared/middleware"
)

type mockCalendarUC struct {
	createCalendarResult *domain.Calendar
	createCalendarErr    error
	listCalendarsResult  []*domain.Calendar
	listCalendarsErr     error
	updateCalendarErr    error
	deleteCalendarErr    error

	createEventResult *domain.Event
	createEventErr    error
	getEventResult    *domain.Event
	getEventErr       error
	listEventsResult  []*domain.Event
	listEventsErr     error
	backfillResult    *portin.BackfillEventsResult
	backfillErr       error
	daySummaryResult  *portin.DaySummaryResult
	daySummaryErr     error
	updateEventResult *domain.Event
	updateEventErr    error
	deleteEventErr    error
}

func (m *mockCalendarUC) CreateCalendar(context.Context, string, string, *string) (*domain.Calendar, error) {
	return m.createCalendarResult, m.createCalendarErr
}
func (m *mockCalendarUC) ListCalendars(context.Context, string) ([]*domain.Calendar, error) {
	return m.listCalendarsResult, m.listCalendarsErr
}
func (m *mockCalendarUC) UpdateCalendar(context.Context, string, string, string, *string, *bool) error {
	return m.updateCalendarErr
}
func (m *mockCalendarUC) DeleteCalendar(context.Context, string, string) error {
	return m.deleteCalendarErr
}
func (m *mockCalendarUC) CreateEvent(context.Context, string, portin.CreateEventInput) (*domain.Event, error) {
	return m.createEventResult, m.createEventErr
}
func (m *mockCalendarUC) GetEvent(context.Context, string, string) (*domain.Event, error) {
	return m.getEventResult, m.getEventErr
}
func (m *mockCalendarUC) ListEvents(context.Context, string, []string, string, string) ([]*domain.Event, error) {
	return m.listEventsResult, m.listEventsErr
}
func (m *mockCalendarUC) BackfillEvents(context.Context, string, portin.BackfillEventsInput) (*portin.BackfillEventsResult, error) {
	return m.backfillResult, m.backfillErr
}
func (m *mockCalendarUC) GetDaySummary(context.Context, string, portin.DaySummaryInput) (*portin.DaySummaryResult, error) {
	return m.daySummaryResult, m.daySummaryErr
}
func (m *mockCalendarUC) UpdateEvent(context.Context, string, string, portin.UpdateEventInput) (*domain.Event, error) {
	return m.updateEventResult, m.updateEventErr
}
func (m *mockCalendarUC) DeleteEvent(context.Context, string, string) error { return m.deleteEventErr }

func calendarReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func withParam(req *http.Request, key, value string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func TestCalendarHandlersCalendars(t *testing.T) {
	now := time.Now()
	uc := &mockCalendarUC{
		createCalendarResult: &domain.Calendar{ID: "c1", Name: "My", CreatedAt: now, UpdatedAt: now},
		listCalendarsResult:  []*domain.Calendar{{ID: "c1", Name: "My", CreatedAt: now, UpdatedAt: now}},
	}
	h := NewCalendarHandler(uc)

	rec := httptest.NewRecorder()
	h.CreateCalendar(rec, calendarReq(http.MethodPost, "/calendars", `{"name":"My"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateCalendar(rec, calendarReq(http.MethodPost, "/calendars", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.createCalendarErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.CreateCalendar(rec, calendarReq(http.MethodPost, "/calendars", `{"name":"My"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ListCalendars(rec, calendarReq(http.MethodGet, "/calendars", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listCalendarsErr = errors.New("db")
	rec = httptest.NewRecorder()
	h.ListCalendars(rec, calendarReq(http.MethodGet, "/calendars", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := withParam(calendarReq(http.MethodPatch, "/calendars/c1", `{"name":"N"}`), "calendarID", "c1")
	h.UpdateCalendar(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/calendars/c1", `{`), "calendarID", "c1")
	h.UpdateCalendar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.updateCalendarErr = errors.New("fail")
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/calendars/c1", `{"name":"N"}`), "calendarID", "c1")
	h.UpdateCalendar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodDelete, "/calendars/c1", ""), "calendarID", "c1")
	h.DeleteCalendar(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.deleteCalendarErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.DeleteCalendar(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestCalendarHandlersEventsCore(t *testing.T) {
	now := time.Now()
	uc := &mockCalendarUC{
		createEventResult: &domain.Event{ID: "e1", CalendarID: "c1", Title: "A", StartTime: now, EndTime: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now},
		getEventResult:    &domain.Event{ID: "e1", CalendarID: "c1", Title: "A", StartTime: now, EndTime: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now},
		listEventsResult:  []*domain.Event{{ID: "e1", CalendarID: "c1", Title: "A", StartTime: now, EndTime: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now}},
		updateEventResult: &domain.Event{ID: "e1", CalendarID: "c1", Title: "B", StartTime: now, EndTime: now.Add(time.Hour), CreatedAt: now, UpdatedAt: now},
		backfillResult:    &portin.BackfillEventsResult{FetchedEvents: 1, UpdatedEvents: 1, DeletedEvents: 0, CoveredStart: now, CoveredEnd: now},
		daySummaryResult:  &portin.DaySummaryResult{Date: "2026-03-05", Timezone: "Asia/Seoul"},
	}
	h := NewCalendarHandler(uc)

	// Create event
	rec := httptest.NewRecorder()
	h.CreateEvent(rec, calendarReq(http.MethodPost, "/events", `{"calendar_id":"c1","start_time":"2026-03-05T10:00:00Z","end_time":"2026-03-05T11:00:00Z"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateEvent(rec, calendarReq(http.MethodPost, "/events", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateEvent(rec, calendarReq(http.MethodPost, "/events", `{"calendar_id":"","start_time":"","end_time":""}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.createEventErr = domain.ErrReadOnlyCalendar
	rec = httptest.NewRecorder()
	h.CreateEvent(rec, calendarReq(http.MethodPost, "/events", `{"calendar_id":"c1","start_time":"2026-03-05T10:00:00Z","end_time":"2026-03-05T11:00:00Z"}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	uc.createEventErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.CreateEvent(rec, calendarReq(http.MethodPost, "/events", `{"calendar_id":"c1","start_time":"2026-03-05T10:00:00Z","end_time":"2026-03-05T11:00:00Z"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Get event
	rec = httptest.NewRecorder()
	req := withParam(calendarReq(http.MethodGet, "/events/e1", ""), "eventID", "e1")
	uc.createEventErr = nil
	h.GetEvent(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.getEventErr = errors.New("not found")
	rec = httptest.NewRecorder()
	h.GetEvent(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	// day-summary path through GetEvent
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodGet, "/events/day-summary?date=2026-03-05", ""), "eventID", "day-summary")
	uc.getEventErr = nil
	h.GetEvent(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for day-summary path, got %d", rec.Code)
	}

	// List events
	rec = httptest.NewRecorder()
	h.ListEvents(rec, calendarReq(http.MethodGet, "/events", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ListEvents(rec, calendarReq(http.MethodGet, "/events?start=2026-03-01&end=2026-03-31&calendar_ids=c1,c2", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.listEventsErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.ListEvents(rec, calendarReq(http.MethodGet, "/events?start=2026-03-01&end=2026-03-31", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	// Day summary
	rec = httptest.NewRecorder()
	h.GetDaySummary(rec, calendarReq(http.MethodGet, "/events/day-summary", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.GetDaySummary(rec, calendarReq(http.MethodGet, "/events/day-summary?date=2026-03-05&include_done_todos=maybe", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.GetDaySummary(rec, calendarReq(http.MethodGet, "/events/day-summary?date=2026-03-05&calendar_ids=c1,%20,%20c2&include_done_todos=true", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.daySummaryErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.GetDaySummary(rec, calendarReq(http.MethodGet, "/events/day-summary?date=2026-03-05", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Backfill
	rec = httptest.NewRecorder()
	h.BackfillEvents(rec, calendarReq(http.MethodPost, "/events/backfill", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.BackfillEvents(rec, calendarReq(http.MethodPost, "/events/backfill", `{"start":"","end":""}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.BackfillEvents(rec, calendarReq(http.MethodPost, "/events/backfill", `{"start":"2026-03-01","end":"2026-03-31"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.backfillErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.BackfillEvents(rec, calendarReq(http.MethodPost, "/events/backfill", `{"start":"2026-03-01","end":"2026-03-31"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	// Update event
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/events/e1", `{"title":"x"}`), "eventID", "e1")
	h.UpdateEvent(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/events/e1", `{`), "eventID", "e1")
	h.UpdateEvent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateEventErr = domain.ErrReadOnlyCalendar
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/events/e1", `{"title":"x"}`), "eventID", "e1")
	h.UpdateEvent(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.updateEventErr = errors.New("fail")
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodPatch, "/events/e1", `{"title":"x"}`), "eventID", "e1")
	h.UpdateEvent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "UPDATE_FAILED") {
		t.Fatalf("expected UPDATE_FAILED response, body=%s", rec.Body.String())
	}

	// Delete event
	rec = httptest.NewRecorder()
	req = withParam(calendarReq(http.MethodDelete, "/events/e1", ""), "eventID", "e1")
	uc.updateEventErr = nil
	h.DeleteEvent(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.deleteEventErr = domain.ErrReadOnlyCalendar
	rec = httptest.NewRecorder()
	h.DeleteEvent(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.deleteEventErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.DeleteEvent(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
