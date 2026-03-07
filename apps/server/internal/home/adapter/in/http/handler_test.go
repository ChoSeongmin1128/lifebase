package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"lifebase/internal/home/domain"
	portin "lifebase/internal/home/port/in"
	"lifebase/internal/shared/middleware"
)

type mockHomeUC struct {
	result *domain.Summary
	err    error
	input  portin.GetSummaryInput
	userID string
}

func (m *mockHomeUC) GetSummary(_ context.Context, userID string, input portin.GetSummaryInput) (*domain.Summary, error) {
	m.userID = userID
	m.input = input
	if m.err != nil {
		return nil, m.err
	}
	return m.result, nil
}

func requestWithUser(target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func TestHomeHandlerGetSummaryValidation(t *testing.T) {
	h := NewHomeHandler(&mockHomeUC{})
	cases := []string{
		"/home/summary",
		"/home/summary?start=bad&end=2026-03-05T00:00:00Z",
		"/home/summary?start=2026-03-04T00:00:00Z&end=bad",
		"/home/summary?start=2026-03-05T00:00:00Z&end=2026-03-04T00:00:00Z",
	}
	for _, target := range cases {
		rec := httptest.NewRecorder()
		h.GetSummary(rec, requestWithUser(target))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("target=%s expected 400, got %d", target, rec.Code)
		}
	}
}

func TestHomeHandlerGetSummarySuccess(t *testing.T) {
	summary := &domain.Summary{}
	uc := &mockHomeUC{result: summary}
	h := NewHomeHandler(uc)
	rec := httptest.NewRecorder()
	target := "/home/summary?start=2026-03-04T00:00:00Z&end=2026-03-05T00:00:00Z&event_limit=50&todo_limit=1&recent_limit=0"

	h.GetSummary(rec, requestWithUser(target))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if uc.userID != "user-1" {
		t.Fatalf("expected user-1, got %q", uc.userID)
	}
	if uc.input.EventLimit != 20 {
		t.Fatalf("expected clamped event limit 20, got %d", uc.input.EventLimit)
	}
	if uc.input.TodoLimit != 1 {
		t.Fatalf("expected todo limit 1, got %d", uc.input.TodoLimit)
	}
	if uc.input.RecentLimit != 8 {
		t.Fatalf("expected default recent limit 8, got %d", uc.input.RecentLimit)
	}
	var out domain.Summary
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode summary: %v", err)
	}
}

func TestHomeHandlerGetSummaryUseCaseErrors(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: errors.New("start and end are required"), want: http.StatusBadRequest},
		{err: errors.New("item not found"), want: http.StatusNotFound},
		{err: errors.New("db failed"), want: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		uc := &mockHomeUC{err: tc.err}
		h := NewHomeHandler(uc)
		h.GetSummary(rec, requestWithUser("/home/summary?start=2026-03-04T00:00:00Z&end=2026-03-05T00:00:00Z"))
		if rec.Code != tc.want {
			t.Fatalf("err=%v expected %d, got %d", tc.err, tc.want, rec.Code)
		}
	}
}

func TestParseLimit(t *testing.T) {
	if got := parseLimit("", 5, 20); got != 5 {
		t.Fatalf("expected default 5, got %d", got)
	}
	if got := parseLimit("abc", 5, 20); got != 5 {
		t.Fatalf("expected default for invalid, got %d", got)
	}
	if got := parseLimit("-1", 5, 20); got != 5 {
		t.Fatalf("expected default for non-positive, got %d", got)
	}
	if got := parseLimit("30", 5, 20); got != 20 {
		t.Fatalf("expected max 20, got %d", got)
	}
	if got := parseLimit("7", 5, 20); got != 7 {
		t.Fatalf("expected 7, got %d", got)
	}
}

func TestWriteUseCaseError(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{err: errors.New("required"), want: http.StatusBadRequest},
		{err: errors.New("must be before"), want: http.StatusBadRequest},
		{err: errors.New("not found"), want: http.StatusNotFound},
		{err: errors.New("other"), want: http.StatusInternalServerError},
	}
	for _, tc := range tests {
		rec := httptest.NewRecorder()
		writeUseCaseError(rec, tc.err)
		if rec.Code != tc.want {
			t.Fatalf("err=%v expected %d, got %d", tc.err, tc.want, rec.Code)
		}
	}
}

func TestHomeHandlerParsesRFC3339(t *testing.T) {
	uc := &mockHomeUC{result: &domain.Summary{}}
	h := NewHomeHandler(uc)
	rec := httptest.NewRecorder()
	start := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	target := "/home/summary?start=" + start.Format(time.RFC3339) + "&end=" + end.Format(time.RFC3339)

	h.GetSummary(rec, requestWithUser(target))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !uc.input.Start.Equal(start) || !uc.input.End.Equal(end) {
		t.Fatalf("unexpected parsed time: %+v", uc.input)
	}
}
