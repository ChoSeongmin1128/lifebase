package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/gallery/domain"
	"lifebase/internal/shared/middleware"
)

type mockGalleryUC struct {
	items      []*domain.Media
	nextCursor string
	err        error
}

func (m *mockGalleryUC) ListMedia(context.Context, string, string, string, string, string, int) ([]*domain.Media, string, error) {
	if m.err != nil {
		return nil, "", m.err
	}
	return m.items, m.nextCursor, nil
}

func galleryReq(method, target string) *http.Request {
	req := httptest.NewRequest(method, target, nil)
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func galleryReqWithParam(method, target, key, value string) *http.Request {
	req := galleryReq(method, target)
	rc := chi.NewRouteContext()
	rc.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func galleryReqWithParams(method, target, fileID, size string) *http.Request {
	req := galleryReq(method, target)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("fileID", fileID)
	rc.URLParams.Add("size", size)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
}

func TestGalleryHandlerListMedia(t *testing.T) {
	now := time.Now()
	uc := &mockGalleryUC{
		items: []*domain.Media{{ID: "m1", CreatedAt: now, UpdatedAt: now}},
	}
	h := NewGalleryHandler(uc, t.TempDir())

	rec := httptest.NewRecorder()
	h.ListMedia(rec, galleryReq(http.MethodGet, "/gallery"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.nextCursor = "m1"
	rec = httptest.NewRecorder()
	h.ListMedia(rec, galleryReq(http.MethodGet, "/gallery?limit=20"))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.err = errors.New("db failed")
	rec = httptest.NewRecorder()
	h.ListMedia(rec, galleryReq(http.MethodGet, "/gallery?limit=bad"))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestGalleryHandlerGetThumbnail(t *testing.T) {
	root := t.TempDir()
	h := NewGalleryHandler(&mockGalleryUC{}, root)

	rec := httptest.NewRecorder()
	req := galleryReqWithParams(http.MethodGet, "/gallery/thumbnails/f1/large", "f1", "large")
	h.GetThumbnail(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = galleryReqWithParams(http.MethodGet, "/gallery/thumbnails/f1/small", "f1", "small")
	h.GetThumbnail(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	userDir := filepath.Join(root, "user-1")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := []byte("webp-data")
	if err := os.WriteFile(filepath.Join(userDir, "f1_small.webp"), content, 0o600); err != nil {
		t.Fatalf("write thumbnail: %v", err)
	}

	rec = httptest.NewRecorder()
	req = galleryReqWithParams(http.MethodGet, "/gallery/thumbnails/f1/small", "f1", "small")
	h.GetThumbnail(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "image/webp" {
		t.Fatalf("expected image/webp, got %s", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("expected immutable cache header, got %s", cc)
	}
	if got := rec.Body.String(); got != string(content) {
		t.Fatalf("unexpected body: %q", got)
	}
}
