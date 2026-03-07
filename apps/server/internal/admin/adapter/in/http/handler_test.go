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

	admindomain "lifebase/internal/admin/domain"
	portin "lifebase/internal/admin/port/in"
	authdomain "lifebase/internal/auth/domain"
	"lifebase/internal/shared/middleware"
)

type mockAdminUC struct {
	listUsersResult []*authdomain.User
	listUsersCursor string
	listUsersErr    error

	userDetail    *portin.UserDetail
	userDetailErr error

	updateQuotaErr       error
	recalculateResult    int64
	recalculateErr       error
	resetStorageErr      error
	updateGoogleStatusErr error

	listAdminsResult []*portin.AdminUserView
	listAdminsErr    error
	createAdminRes   *portin.AdminUserView
	createAdminErr   error
	updateAdminErr   error
	deactivateErr    error
}

func (m *mockAdminUC) ListUsers(context.Context, string, string, string, int) ([]*authdomain.User, string, error) {
	return m.listUsersResult, m.listUsersCursor, m.listUsersErr
}
func (m *mockAdminUC) GetUserDetail(context.Context, string, string) (*portin.UserDetail, error) {
	return m.userDetail, m.userDetailErr
}
func (m *mockAdminUC) UpdateStorageQuota(context.Context, string, string, int64) error { return m.updateQuotaErr }
func (m *mockAdminUC) RecalculateStorageUsed(context.Context, string, string) (int64, error) {
	return m.recalculateResult, m.recalculateErr
}
func (m *mockAdminUC) ResetUserStorage(context.Context, string, string, string) error { return m.resetStorageErr }
func (m *mockAdminUC) UpdateGoogleAccountStatus(context.Context, string, string, string, string) error {
	return m.updateGoogleStatusErr
}
func (m *mockAdminUC) ListAdmins(context.Context, string) ([]*portin.AdminUserView, error) {
	return m.listAdminsResult, m.listAdminsErr
}
func (m *mockAdminUC) CreateAdmin(context.Context, string, string, admindomain.Role) (*portin.AdminUserView, error) {
	return m.createAdminRes, m.createAdminErr
}
func (m *mockAdminUC) UpdateAdminRole(context.Context, string, string, admindomain.Role) error {
	return m.updateAdminErr
}
func (m *mockAdminUC) DeactivateAdmin(context.Context, string, string) error { return m.deactivateErr }

func adminReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "actor-1")
	return req.WithContext(ctx)
}

func adminReqWithParam(req *http.Request, key, value string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func TestAdminHandlerUsersAndStorage(t *testing.T) {
	now := time.Now()
	uc := &mockAdminUC{
		listUsersResult: []*authdomain.User{{ID: "u1", Email: "u1@example.com", CreatedAt: now, UpdatedAt: now}},
		listUsersCursor: "next",
		userDetail: &portin.UserDetail{
			User: &authdomain.User{ID: "u1", Email: "u1@example.com"},
			GoogleAccounts: []portin.GoogleAccountSummary{{ID: "ga1", Status: "active"}},
		},
		recalculateResult: 123,
	}
	h := NewAdminHandler(uc)

	rec := httptest.NewRecorder()
	h.ListUsers(rec, adminReq(http.MethodGet, "/admin/users?q=a&cursor=c&limit=999", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.listUsersErr = errors.New("denied")
	rec = httptest.NewRecorder()
	h.ListUsers(rec, adminReq(http.MethodGet, "/admin/users", ""))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.listUsersErr = nil

	rec = httptest.NewRecorder()
	req := adminReqWithParam(adminReq(http.MethodGet, "/admin/users/u1", ""), "userID", "u1")
	h.GetUser(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.userDetailErr = errors.New("not found")
	rec = httptest.NewRecorder()
	h.GetUser(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.userDetailErr = nil

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/quota", `{"quota_bytes":2048}`), "userID", "u1")
	h.UpdateQuota(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/quota", `{`), "userID", "u1")
	h.UpdateQuota(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.updateQuotaErr = errors.New("invalid quota")
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/quota", `{"quota_bytes":0}`), "userID", "u1")
	h.UpdateQuota(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateQuotaErr = nil

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPost, "/admin/users/u1/recalculate-storage", ""), "userID", "u1")
	h.RecalculateStorage(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.recalculateErr = errors.New("db fail")
	rec = httptest.NewRecorder()
	h.RecalculateStorage(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.recalculateErr = nil

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPost, "/admin/users/u1/reset-storage", `{"confirm":"DELETE u1@example.com"}`), "userID", "u1")
	h.ResetStorage(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPost, "/admin/users/u1/reset-storage", `{`), "userID", "u1")
	h.ResetStorage(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.resetStorageErr = errors.New("mismatch")
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPost, "/admin/users/u1/reset-storage", `{"confirm":"wrong"}`), "userID", "u1")
	h.ResetStorage(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestAdminHandlerAdminManagement(t *testing.T) {
	now := time.Now()
	uc := &mockAdminUC{
		listAdminsResult: []*portin.AdminUserView{{ID: "a1", UserID: "u1", Role: admindomain.RoleAdmin, CreatedAt: now, UpdatedAt: now}},
		createAdminRes:   &portin.AdminUserView{ID: "a2", UserID: "u2", Role: admindomain.RoleAdmin, CreatedAt: now, UpdatedAt: now},
	}
	h := NewAdminHandler(uc)

	rec := httptest.NewRecorder()
	h.ListAdmins(rec, adminReq(http.MethodGet, "/admin/admins", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listAdminsErr = errors.New("required")
	rec = httptest.NewRecorder()
	h.ListAdmins(rec, adminReq(http.MethodGet, "/admin/admins", ""))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.listAdminsErr = nil

	rec = httptest.NewRecorder()
	h.CreateAdmin(rec, adminReq(http.MethodPost, "/admin/admins", `{"email":"u2@example.com","role":"admin"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateAdmin(rec, adminReq(http.MethodPost, "/admin/admins", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.createAdminErr = errors.New("not found")
	rec = httptest.NewRecorder()
	h.CreateAdmin(rec, adminReq(http.MethodPost, "/admin/admins", `{"email":"x@example.com"}`))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	uc.createAdminErr = nil

	rec = httptest.NewRecorder()
	req := adminReqWithParam(adminReq(http.MethodPatch, "/admin/admins/a1", `{"role":"super_admin"}`), "adminID", "a1")
	h.UpdateAdminRole(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/admins/a1", `{`), "adminID", "a1")
	h.UpdateAdminRole(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateAdminErr = errors.New("cannot")
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/admins/a1", `{"role":"admin"}`), "adminID", "a1")
	h.UpdateAdminRole(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.updateAdminErr = nil

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodDelete, "/admin/admins/a1", ""), "adminID", "a1")
	h.DeactivateAdmin(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.deactivateErr = errors.New("cannot deactivate yourself")
	rec = httptest.NewRecorder()
	h.DeactivateAdmin(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}

func TestAdminHandlerGoogleStatusAndHelpers(t *testing.T) {
	uc := &mockAdminUC{}
	h := NewAdminHandler(uc)

	rec := httptest.NewRecorder()
	req := adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/google-accounts/ga1/status", `{"status":"active"}`), "userID", "u1")
	req = adminReqWithParam(req, "accountID", "ga1")
	h.UpdateGoogleAccountStatus(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/google-accounts/ga1/status", `{`), "userID", "u1")
	req = adminReqWithParam(req, "accountID", "ga1")
	h.UpdateGoogleAccountStatus(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.updateGoogleStatusErr = errors.New("denied")
	rec = httptest.NewRecorder()
	req = adminReqWithParam(adminReq(http.MethodPatch, "/admin/users/u1/google-accounts/ga1/status", `{"status":"revoked"}`), "userID", "u1")
	req = adminReqWithParam(req, "accountID", "ga1")
	h.UpdateGoogleAccountStatus(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}

	if got := parseLimit("", 20, 100); got != 20 {
		t.Fatalf("expected default limit, got %d", got)
	}
	if got := parseLimit("bad", 20, 100); got != 20 {
		t.Fatalf("expected default for bad limit, got %d", got)
	}
	if got := parseLimit("0", 20, 100); got != 20 {
		t.Fatalf("expected default for zero limit, got %d", got)
	}
	if got := parseLimit("999", 20, 100); got != 100 {
		t.Fatalf("expected max limit, got %d", got)
	}
	if got := parseLimit("50", 20, 100); got != 50 {
		t.Fatalf("expected parsed limit, got %d", got)
	}

	for _, tc := range []struct {
		err  error
		code int
	}{
		{errors.New("denied"), http.StatusForbidden},
		{errors.New("required"), http.StatusForbidden},
		{errors.New("cannot change own role"), http.StatusForbidden},
		{errors.New("cannot deactivate yourself"), http.StatusForbidden},
		{errors.New("not found"), http.StatusNotFound},
		{errors.New("invalid"), http.StatusBadRequest},
		{errors.New("must be positive"), http.StatusBadRequest},
		{errors.New("cannot"), http.StatusBadRequest},
		{errors.New("mismatch"), http.StatusBadRequest},
		{errors.New("unknown"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		writeUseCaseError(rec, tc.err)
		if rec.Code != tc.code {
			t.Fatalf("err=%q expected %d got %d", tc.err, tc.code, rec.Code)
		}
	}
}

