package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"lifebase/internal/sharing/domain"
)

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
	shares  map[string]*domain.Share
}

func newMockInviteRepo() *mockInviteRepo {
	return &mockInviteRepo{
		invites: make(map[string]*domain.ShareInvite),
		shares:  make(map[string]*domain.Share),
	}
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

func (m *mockInviteRepo) AcceptWithShare(_ context.Context, id string, share *domain.Share, acceptedAt time.Time) (bool, error) {
	inv, ok := m.invites[id]
	if !ok {
		return false, fmt.Errorf("not found")
	}
	if inv.AcceptedAt != nil {
		return false, nil
	}
	inv.AcceptedAt = &acceptedAt
	m.shares[share.ID] = share
	return true, nil
}

type mockFolderAccessRepo struct {
	owners map[string]string
}

func newMockFolderAccessRepo() *mockFolderAccessRepo {
	return &mockFolderAccessRepo{owners: map[string]string{}}
}

func (m *mockFolderAccessRepo) IsOwner(_ context.Context, userID, folderID string) (bool, error) {
	return m.owners[folderID] == userID, nil
}

func newSharingUseCaseForTest() (*mockShareRepo, *mockInviteRepo, *mockFolderAccessRepo, *sharingUseCase) {
	shareRepo := newMockShareRepo()
	inviteRepo := newMockInviteRepo()
	folderRepo := newMockFolderAccessRepo()
	folderRepo.owners["folder1"] = "owner1"
	uc := NewSharingUseCase(shareRepo, inviteRepo, folderRepo).(*sharingUseCase)
	return shareRepo, inviteRepo, folderRepo, uc
}

func TestCreateInvite_InvalidRole(t *testing.T) {
	_, _, _, uc := newSharingUseCaseForTest()

	_, err := uc.CreateInvite(context.Background(), "owner1", "folder1", "admin")
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestCreateInvite_RequiresOwner(t *testing.T) {
	_, _, _, uc := newSharingUseCaseForTest()

	_, err := uc.CreateInvite(context.Background(), "user2", "folder1", "viewer")
	if err == nil {
		t.Fatal("expected access denied for non-owner")
	}
}

func TestCreateInvite_ValidRoles(t *testing.T) {
	_, _, _, uc := newSharingUseCaseForTest()

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
	_, _, _, uc := newSharingUseCaseForTest()

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	_, err := uc.AcceptInvite(context.Background(), "owner1", link.Token)
	if err == nil {
		t.Fatal("expected error when sharing with self")
	}
}

func TestAcceptInvite_Success(t *testing.T) {
	_, _, _, uc := newSharingUseCaseForTest()

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
	_, _, _, uc := newSharingUseCaseForTest()

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	_, _ = uc.AcceptInvite(context.Background(), "user2", link.Token)

	_, err := uc.AcceptInvite(context.Background(), "user3", link.Token)
	if err == nil {
		t.Fatal("expected error for already accepted invite")
	}
}

func TestListShares_RequiresOwner(t *testing.T) {
	_, _, _, uc := newSharingUseCaseForTest()
	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	_, _ = uc.AcceptInvite(context.Background(), "user2", link.Token)

	if _, err := uc.ListShares(context.Background(), "user2", "folder1"); err == nil {
		t.Fatal("expected access denied for non-owner")
	}
}

func TestRemoveShare_NotOwner(t *testing.T) {
	shareRepo, _, _, uc := newSharingUseCaseForTest()

	link, _ := uc.CreateInvite(context.Background(), "owner1", "folder1", "viewer")
	share, _ := uc.AcceptInvite(context.Background(), "user2", link.Token)
	shareRepo.shares[share.ID] = share

	err := uc.RemoveShare(context.Background(), "user2", share.ID)
	if err == nil {
		t.Fatal("expected error when non-owner tries to remove share")
	}
}
