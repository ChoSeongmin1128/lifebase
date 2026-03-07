package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"lifebase/internal/holiday/domain"
	portin "lifebase/internal/holiday/port/in"
)

type mockHolidayUC struct {
	listResult []domain.Holiday
	listErr    error
	refreshRes *portin.RefreshRangeResult
	refreshErr error
}

func (m *mockHolidayUC) ListRange(context.Context, time.Time, time.Time) ([]domain.Holiday, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listResult, nil
}

func (m *mockHolidayUC) RefreshRange(context.Context, portin.RefreshRangeInput) (*portin.RefreshRangeResult, error) {
	if m.refreshErr != nil {
		return nil, m.refreshErr
	}
	return m.refreshRes, nil
}

func TestHolidayHandlerListValidation(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{})
	targets := []string{
		"/holidays",
		"/holidays?start=bad&end=2026-03-05",
		"/holidays?start=2026-03-04&end=bad",
	}
	for _, target := range targets {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, target, nil)
		h.ListHolidays(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("target=%s expected 400, got %d", target, rec.Code)
		}
	}
}

func TestHolidayHandlerListError(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{listErr: errors.New("failed")})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/holidays?start=2026-03-04&end=2026-03-05", nil)

	h.ListHolidays(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHolidayHandlerListSuccess(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{
		listResult: []domain.Holiday{
			{Name: "어린이날", Date: time.Date(2026, 5, 5, 0, 0, 0, 0, time.UTC)},
		},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/holidays?start=2026-05-01&end=2026-05-31", nil)

	h.ListHolidays(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string][]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["holidays"][0]["name"] != "어린이날" {
		t.Fatalf("unexpected payload: %#v", body)
	}
}

func TestHolidayHandlerRefreshValidation(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/holidays/refresh", strings.NewReader("{"))

	h.RefreshHolidays(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHolidayHandlerRefreshError(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{refreshErr: errors.New("failed")})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/holidays/refresh", strings.NewReader(`{}`))

	h.RefreshHolidays(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestHolidayHandlerRefreshSuccess(t *testing.T) {
	h := NewHolidayHandler(&mockHolidayUC{
		refreshRes: &portin.RefreshRangeResult{MonthsTotal: 12, MonthsRefreshed: 12, ItemsUpserted: 20},
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/holidays/refresh", strings.NewReader(`{}`))

	h.RefreshHolidays(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var out portin.RefreshRangeResult
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.MonthsTotal != 12 {
		t.Fatalf("unexpected response: %#v", out)
	}
}
