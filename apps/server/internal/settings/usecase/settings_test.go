package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"lifebase/internal/settings/domain"
)

type mockSettingsRepo struct {
	items  []domain.Setting
	listErr error
	setErr  error
	setCalls []struct {
		userID string
		key    string
		value  string
	}
}

func (m *mockSettingsRepo) ListByUser(context.Context, string) ([]domain.Setting, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.items, nil
}

func (m *mockSettingsRepo) Set(_ context.Context, userID, key, value string) error {
	if m.setErr != nil {
		return m.setErr
	}
	m.setCalls = append(m.setCalls, struct {
		userID string
		key    string
		value  string
	}{userID: userID, key: key, value: value})
	return nil
}

func TestGetAll(t *testing.T) {
	repo := &mockSettingsRepo{
		items: []domain.Setting{
			{UserID: "u1", Key: "theme", Value: "dark", UpdatedAt: time.Now()},
			{UserID: "u1", Key: "lang", Value: "ko", UpdatedAt: time.Now()},
		},
	}
	uc := NewSettingsUseCase(repo)

	got, err := uc.GetAll(context.Background(), "u1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["theme"] != "dark" || got["lang"] != "ko" {
		t.Fatalf("unexpected map: %#v", got)
	}
}

func TestGetAllRepoError(t *testing.T) {
	repo := &mockSettingsRepo{listErr: errors.New("boom")}
	uc := NewSettingsUseCase(repo)

	if _, err := uc.GetAll(context.Background(), "u1"); err == nil {
		t.Fatal("expected error")
	}
}

func TestUpdate(t *testing.T) {
	repo := &mockSettingsRepo{}
	uc := NewSettingsUseCase(repo)

	err := uc.Update(context.Background(), "u1", map[string]string{
		"theme": "light",
		"lang":  "en",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.setCalls) != 2 {
		t.Fatalf("expected 2 set calls, got %d", len(repo.setCalls))
	}
}

func TestUpdateRepoError(t *testing.T) {
	repo := &mockSettingsRepo{setErr: errors.New("set failed")}
	uc := NewSettingsUseCase(repo)

	if err := uc.Update(context.Background(), "u1", map[string]string{"theme": "light"}); err == nil {
		t.Fatal("expected error")
	}
}
