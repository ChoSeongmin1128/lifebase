package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lifebase/internal/gallery/domain"
)

type mockMediaRepo struct {
	items []*domain.Media
	item  *domain.Media
	err   error
	limit int
}

func (m *mockMediaRepo) ListMedia(_ context.Context, _ string, _ string, _ string, _ string, _ string, limit int) ([]*domain.Media, error) {
	m.limit = limit
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func (m *mockMediaRepo) GetMedia(_ context.Context, _ string, _ string) (*domain.Media, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.item, nil
}

func TestListMediaDefaultLimitAndCursor(t *testing.T) {
	now := time.Now()
	repo := &mockMediaRepo{
		items: []*domain.Media{
			{ID: "m1", CreatedAt: now, UpdatedAt: now},
			{ID: "m2", CreatedAt: now, UpdatedAt: now},
			{ID: "m3", CreatedAt: now, UpdatedAt: now},
		},
	}
	uc := NewGalleryUseCase(repo)

	files, next, err := uc.ListMedia(context.Background(), "u1", "image", "taken_at", "desc", "", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if next != "m2" {
		t.Fatalf("expected next cursor m2, got %q", next)
	}
}

func TestListMediaInvalidLimitUsesDefault(t *testing.T) {
	repo := &mockMediaRepo{}
	uc := NewGalleryUseCase(repo)

	_, _, err := uc.ListMedia(context.Background(), "u1", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.limit != 51 {
		t.Fatalf("expected internal fetch limit 51 (default 50 + 1), got %d", repo.limit)
	}
}

func TestListMediaRepoError(t *testing.T) {
	repo := &mockMediaRepo{err: errors.New("db failed")}
	uc := NewGalleryUseCase(repo)
	if _, _, err := uc.ListMedia(context.Background(), "u1", "", "", "", "", 10); err == nil {
		t.Fatal("expected repo error")
	}
}

func TestListMediaNoNextCursorWhenWithinLimit(t *testing.T) {
	now := time.Now()
	repo := &mockMediaRepo{
		items: []*domain.Media{
			{ID: "m1", CreatedAt: now, UpdatedAt: now},
		},
	}
	uc := NewGalleryUseCase(repo)

	files, next, err := uc.ListMedia(context.Background(), "u1", "", "", "", "", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if next != "" {
		t.Fatalf("expected empty next cursor, got %q", next)
	}
}

func TestGetMedia(t *testing.T) {
	now := time.Now()
	repo := &mockMediaRepo{item: &domain.Media{ID: "m1", CreatedAt: now, UpdatedAt: now}}
	uc := NewGalleryUseCase(repo)

	item, err := uc.GetMedia(context.Background(), "u1", "m1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item == nil || item.ID != "m1" {
		t.Fatalf("unexpected media: %#v", item)
	}
}
