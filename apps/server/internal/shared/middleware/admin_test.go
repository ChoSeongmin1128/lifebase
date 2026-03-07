package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockAdminChecker struct {
	ok   bool
	err  error
	last string
}

func (m *mockAdminChecker) IsActiveAdmin(_ context.Context, userID string) (bool, error) {
	m.last = userID
	return m.ok, m.err
}

func codeFromRecorder(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	return body.Error.Code
}

func TestAdminUnauthorizedWithoutUserID(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	checker := &mockAdminChecker{ok: true}
	Admin(checker)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if code := codeFromRecorder(t, rec); code != "UNAUTHORIZED" {
		t.Fatalf("expected UNAUTHORIZED, got %s", code)
	}
}

func TestAdminCheckFailed(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, "user-1"))

	checker := &mockAdminChecker{err: errors.New("db failed")}
	Admin(checker)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if code := codeFromRecorder(t, rec); code != "ADMIN_CHECK_FAILED" {
		t.Fatalf("expected ADMIN_CHECK_FAILED, got %s", code)
	}
	if checker.last != "user-1" {
		t.Fatalf("expected checker to receive user-1, got %q", checker.last)
	}
}

func TestAdminForbiddenWhenNotAdmin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, "user-2"))

	checker := &mockAdminChecker{ok: false}
	Admin(checker)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next should not be called")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	if code := codeFromRecorder(t, rec); code != "FORBIDDEN" {
		t.Fatalf("expected FORBIDDEN, got %s", code)
	}
}

func TestAdminSuccess(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), UserIDKey, "admin-user"))

	checker := &mockAdminChecker{ok: true}
	called := false
	Admin(checker)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatal("next was not called")
	}
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	if checker.last != "admin-user" {
		t.Fatalf("expected checker to receive admin-user, got %q", checker.last)
	}
}
