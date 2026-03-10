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

	"lifebase/internal/shared/middleware"
	"lifebase/internal/sharing/domain"
	portin "lifebase/internal/sharing/port/in"
	sharingusecase "lifebase/internal/sharing/usecase"
)

type mockSharingUC struct {
	inviteRes *portin.InviteLink
	inviteErr error

	acceptRes *domain.Share
	acceptErr error

	listRes   []*domain.Share
	listErr   error
	listMeRes []*domain.Share
	listMeErr error

	removeErr error
}

func (m *mockSharingUC) CreateInvite(context.Context, string, string, string) (*portin.InviteLink, error) {
	return m.inviteRes, m.inviteErr
}
func (m *mockSharingUC) AcceptInvite(context.Context, string, string) (*domain.Share, error) {
	return m.acceptRes, m.acceptErr
}
func (m *mockSharingUC) ListShares(context.Context, string, string) ([]*domain.Share, error) {
	return m.listRes, m.listErr
}
func (m *mockSharingUC) ListSharedWithMe(context.Context, string) ([]*domain.Share, error) {
	return m.listMeRes, m.listMeErr
}
func (m *mockSharingUC) RemoveShare(context.Context, string, string) error { return m.removeErr }

func sharingReq(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func sharingReqWithParam(req *http.Request, key, value string) *http.Request {
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func TestSharingHandlers(t *testing.T) {
	now := time.Now()
	uc := &mockSharingUC{
		inviteRes: &portin.InviteLink{Token: "tok", ExpiresAt: now.Format(time.RFC3339)},
		acceptRes: &domain.Share{ID: "s1", FolderID: "f1", OwnerID: "owner", SharedWith: "user-1", Role: "viewer", CreatedAt: now, UpdatedAt: now},
		listRes:   []*domain.Share{{ID: "s1"}},
		listMeRes: []*domain.Share{{ID: "s2"}},
	}
	h := NewSharingHandler(uc)

	rec := httptest.NewRecorder()
	h.CreateInvite(rec, sharingReq(http.MethodPost, "/sharing/invite", `{"folder_id":"f1","role":"editor"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.CreateInvite(rec, sharingReq(http.MethodPost, "/sharing/invite", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.inviteErr = errors.New("invite fail")
	rec = httptest.NewRecorder()
	h.CreateInvite(rec, sharingReq(http.MethodPost, "/sharing/invite", `{"folder_id":"f1"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.inviteErr = sharingusecase.ErrShareAccessDenied
	rec = httptest.NewRecorder()
	h.CreateInvite(rec, sharingReq(http.MethodPost, "/sharing/invite", `{"folder_id":"f1","role":"viewer"}`))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.inviteErr = nil

	rec = httptest.NewRecorder()
	h.AcceptInvite(rec, sharingReq(http.MethodPost, "/sharing/accept", `{"token":"tok"}`))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.AcceptInvite(rec, sharingReq(http.MethodPost, "/sharing/accept", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.acceptErr = errors.New("accept fail")
	rec = httptest.NewRecorder()
	h.AcceptInvite(rec, sharingReq(http.MethodPost, "/sharing/accept", `{"token":"tok"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.acceptErr = nil

	rec = httptest.NewRecorder()
	h.ListShares(rec, sharingReq(http.MethodGet, "/sharing?folder_id=f1", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ListShares(rec, sharingReq(http.MethodGet, "/sharing", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	uc.listErr = errors.New("list fail")
	rec = httptest.NewRecorder()
	h.ListShares(rec, sharingReq(http.MethodGet, "/sharing?folder_id=f1", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listErr = sharingusecase.ErrShareAccessDenied
	rec = httptest.NewRecorder()
	h.ListShares(rec, sharingReq(http.MethodGet, "/sharing?folder_id=f1", ""))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
	uc.listErr = nil

	rec = httptest.NewRecorder()
	h.ListSharedWithMe(rec, sharingReq(http.MethodGet, "/sharing/with-me", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	uc.listMeErr = errors.New("list me fail")
	rec = httptest.NewRecorder()
	h.ListSharedWithMe(rec, sharingReq(http.MethodGet, "/sharing/with-me", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	uc.listMeErr = nil

	rec = httptest.NewRecorder()
	req := sharingReqWithParam(sharingReq(http.MethodDelete, "/sharing/s1", ""), "shareID", "s1")
	h.RemoveShare(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}
	uc.removeErr = errors.New("remove fail")
	rec = httptest.NewRecorder()
	h.RemoveShare(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
