package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"lifebase/internal/sharing/domain"
)

// Mock repositories

type mockShareRepo struct {
	shares map[string]*domain.Share
}

func newMockShareRepo() *mockShareRepo {
	return &mockShareRepo{shares: make(map[string]*domain.Share)}
}

func (m *mockShareRepo) Create(_ context.Context, share *domain.Share) error {
	m.shares[share.ID] = share
	return nil
}

func (m *mockShareRepo) FindByID(_ context.Context, id string) (*domain.Share, error) {
	s, ok := m.shares[id]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return s, nil
}

func (m *mockShareRepo) ListByFolder(_ context.Context, folderID string) ([]*domain.Share, error) {
	var result []*domain.Share
	for _, s := range m.shares {
		if s.FolderID == folderID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockShareRepo) ListByUser(_ context.Context, userID string) ([]*domain.Share, error) {
	var result []*domain.Share
	for _, s := range m.shares {
		if s.SharedWith == userID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockShareRepo) Delete(_ context.Context, id string) error {
	delete(m.shares, id)
	return nil
}

type mockInviteRepo struct {
	invites map[string]*domain.ShareInvite
}

func newMockInviteRepo() *mockInviteRepo {
	return &mockInviteRepo{invites: make(map[string]*domain.ShareInvite)}
}

func (m *mockInviteRepo) Create(_ context.Context, invite *domain.ShareInvite) error {
	m.invites[invite.ID] = invite
	return nil
}

func (m *mockInviteRepo) FindByToken(_ context.Context, token string) (*domain.ShareInvite, error) {
	for _, inv := range m.invites {
		if inv.Token == token {
			return inv, nil
		}
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockInviteRepo) MarkAccepted(_ context.Context, id string) error {
	inv, ok := m.invites[id]
	if !ok {
		return fmt.Errorf("not found")
	}
	now := time.Now()
	inv.AcceptedAt = &now
	return nil
}

// Tests

func TestCreateInvite_InvalidRole(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	_, err := uc.CreateInvite(context.Background(), "owner1", "folder1", "admin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestCreateInvite_ValidRoles(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	for _, role := range []string{"viewer", "editor"} {
		link, err := uc.CreateInvite(context.Background(), "owner1", "folder1", role)
		if err != nil {
			t.Fatalf("unexpected error for role %s: %v", role, err)
		}
		if link.Token == "" {
			t.Errorf("expected non-empty token for role %s", role)
		}
	}
}

func TestAcceptInvite_SelfShare(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	_, err := uc.AcceptInvite(context.Background(), "owner1", link.Token)
	if err == nil {
		t.Fatal("expected error when sharing with self")
	}
}

func TestAcceptInvite_Success(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "editor")
	share, err := uc.AcceptInvite(context.Background(), "user2", link.Token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if share.SharedWith != "user2" {
		t.Errorf("expected shared_with 'user2', got '%s'", share.SharedWith)
	}
	if share.Role != "editor" {
		t.Errorf("expected role 'editor', got '%s'", share.Role)
	}
}

func TestAcceptInvite_AlreadyAccepted(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	uc.AcceptInvite(context.Background(), "user2", link.Token)

	_, err := uc.AcceptInvite(context.Background(), "user3", link.Token)
	if err == nil {
		t.Fatal("expected error for already accepted invite")
	}
}

func TestRemoveShare_NotOwner(t *testing.T) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	uc := NewSharingUseCase(shareRepo, inviteRepo)

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	share, _ := uc.AcceptInvite(context.Background(), "user2", link.Token)

	err := uc.RemoveShare(context.Background(), "user2", share.ID)
	if err == nil {
		t.Fatal("expected error when non-owner tries to remove share")
	}
}
