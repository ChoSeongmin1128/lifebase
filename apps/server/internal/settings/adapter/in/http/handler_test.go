package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lifebase/internal/shared/middleware"
)

type mockSettingsUC struct {
	getAllResult map[string]string
	getAllErr    error
	updateErr    error
	updateInput  map[string]string
	updateUserID string
}

func (m *mockSettingsUC) GetAll(context.Context, string) (map[string]string, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.getAllResult, nil
}

func (m *mockSettingsUC) Update(_ context.Context, userID string, values map[string]string) error {
	m.updateUserID = userID
	m.updateInput = values
	return m.updateErr
}

func reqWithUser(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func TestSettingsHandlerGetAll(t *testing.T) {
	uc := &mockSettingsUC{
		getAllResult: map[string]string{"theme": "dark"},
	}
	h := NewSettingsHandler(uc)
	rec := httptest.NewRecorder()
	req := reqWithUser(http.MethodGet, "/settings", "")

	h.GetAll(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["settings"]["theme"] != "dark" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestSettingsHandlerGetAllError(t *testing.T) {
	uc := &mockSettingsUC{getAllErr: errors.New("db failed")}
	h := NewSettingsHandler(uc)
	rec := httptest.NewRecorder()
	req := reqWithUser(http.MethodGet, "/settings", "")

	h.GetAll(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestSettingsHandlerUpdate(t *testing.T) {
	uc := &mockSettingsUC{}
	h := NewSettingsHandler(uc)
	rec := httptest.NewRecorder()
	req := reqWithUser(http.MethodPatch, "/settings", `{"theme":"light"}`)

	h.Update(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if uc.updateUserID != "user-1" {
		t.Fatalf("expected user-1, got %q", uc.updateUserID)
	}
	if uc.updateInput["theme"] != "light" {
		t.Fatalf("unexpected update input: %#v", uc.updateInput)
	}
}

func TestSettingsHandlerUpdateInvalidBody(t *testing.T) {
	uc := &mockSettingsUC{}
	h := NewSettingsHandler(uc)
	rec := httptest.NewRecorder()
	req := reqWithUser(http.MethodPatch, "/settings", "{")

	h.Update(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestSettingsHandlerUpdateError(t *testing.T) {
	uc := &mockSettingsUC{updateErr: errors.New("update failed")}
	h := NewSettingsHandler(uc)
	rec := httptest.NewRecorder()
	req := reqWithUser(http.MethodPatch, "/settings", `{"theme":"light"}`)

	h.Update(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
