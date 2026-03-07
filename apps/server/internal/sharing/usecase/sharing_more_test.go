package usecase

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"lifebase/internal/sharing/domain"
)

type errShareRepo struct {
	createErr error
	findByID  *domain.Share
	findErr   error
	listErr   error
	deleteErr error
}

func (m *errShareRepo) Create(context.Context, *domain.Share) error { return m.createErr }
func (m *errShareRepo) FindByID(context.Context, string) (*domain.Share, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.findByID == nil {
		return nil, errors.New("not found")
	}
	return m.findByID, nil
}
func (m *errShareRepo) ListByFolder(context.Context, string) ([]*domain.Share, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []*domain.Share{{ID: "s1", FolderID: "f1"}}, nil
}
func (m *errShareRepo) ListByUser(context.Context, string) ([]*domain.Share, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []*domain.Share{{ID: "s2", SharedWith: "u1"}}, nil
}
func (m *errShareRepo) Delete(context.Context, string) error { return m.deleteErr }

type errInviteRepo struct {
	createErr error
	find      *domain.ShareInvite
	findErr   error
	markErr   error
}

func (m *errInviteRepo) Create(context.Context, *domain.ShareInvite) error { return m.createErr }
func (m *errInviteRepo) FindByToken(context.Context, string) (*domain.ShareInvite, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	if m.find == nil {
		return nil, errors.New("not found")
	}
	return m.find, nil
}
func (m *errInviteRepo) MarkAccepted(context.Context, string) error { return m.markErr }

func TestSharingUseCaseErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("create invite invalid role and success", func(t *testing.T) {
		uc := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{})
		if _, err := uc.CreateInvite(ctx, "owner1", "folder1", "admin"); err == nil {
			t.Fatal("expected invalid role error")
		}
		link, err := uc.CreateInvite(ctx, "owner1", "folder1", "viewer")
		if err != nil || link == nil || link.Token == "" || link.ExpiresAt == "" {
			t.Fatalf("expected successful invite creation, link=%#v err=%v", link, err)
		}
	})

	t.Run("create invite token generation error", func(t *testing.T) {
		prev := shareRandRead
		shareRandRead = func(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
		t.Cleanup(func() { shareRandRead = prev })

		uc := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{})
		if _, err := uc.CreateInvite(ctx, "owner1", "folder1", "viewer"); err == nil {
			t.Fatal("expected token generation error")
		}
	})

	t.Run("create invite repository error", func(t *testing.T) {
		uc := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{createErr: errors.New("db down")})
		if _, err := uc.CreateInvite(ctx, "owner1", "folder1", "viewer"); err == nil {
			t.Fatal("expected create invite error")
		}
	})

	t.Run("accept invite invalid token and expired", func(t *testing.T) {
		ucInvalid := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{findErr: errors.New("not found")})
		if _, err := ucInvalid.AcceptInvite(ctx, "u1", "bad"); err == nil {
			t.Fatal("expected invalid token error")
		}

		expired := &domain.ShareInvite{
			ID:        "i1",
			FolderID:  "f1",
			OwnerID:   "owner",
			Token:     "tok",
			Role:      "viewer",
			ExpiresAt: time.Now().Add(-time.Minute),
		}
		ucExpired := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{find: expired})
		if _, err := ucExpired.AcceptInvite(ctx, "u1", "tok"); err == nil {
			t.Fatal("expected invite expired error")
		}
	})

	t.Run("accept invite create share and mark accepted errors", func(t *testing.T) {
		valid := &domain.ShareInvite{
			ID:        "i2",
			FolderID:  "f1",
			OwnerID:   "owner",
			Token:     "tok2",
			Role:      "viewer",
			ExpiresAt: time.Now().Add(time.Minute),
		}

		ucCreateErr := NewSharingUseCase(&errShareRepo{createErr: errors.New("create fail")}, &errInviteRepo{find: valid})
		if _, err := ucCreateErr.AcceptInvite(ctx, "u1", "tok2"); err == nil {
			t.Fatal("expected create share error")
		}

		ucMarkErr := NewSharingUseCase(&errShareRepo{}, &errInviteRepo{find: valid, markErr: errors.New("mark fail")})
		if _, err := ucMarkErr.AcceptInvite(ctx, "u1", "tok2"); err == nil {
			t.Fatal("expected mark accepted error")
		}
	})

	t.Run("list and remove share branches", func(t *testing.T) {
		repo := &errShareRepo{
			findByID: &domain.Share{ID: "s1", OwnerID: "owner1", SharedWith: "u2"},
		}
		uc := NewSharingUseCase(repo, &errInviteRepo{})

		listByFolder, err := uc.ListShares(ctx, "owner1", "f1")
		if err != nil || len(listByFolder) != 1 {
			t.Fatalf("list shares failed: %v, len=%d", err, len(listByFolder))
		}
		listByUser, err := uc.ListSharedWithMe(ctx, "u1")
		if err != nil || len(listByUser) != 1 {
			t.Fatalf("list shared with me failed: %v, len=%d", err, len(listByUser))
		}

		repo.findErr = errors.New("missing")
		if err := uc.RemoveShare(ctx, "owner1", "none"); err == nil {
			t.Fatal("expected share not found")
		}

		repo.findErr = nil
		repo.deleteErr = errors.New("delete failed")
		if err := uc.RemoveShare(ctx, "owner1", "s1"); err == nil {
			t.Fatal("expected delete error")
		}
	})
}

func TestGenerateTokenError(t *testing.T) {
	prev := shareRandRead
	shareRandRead = func(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
	t.Cleanup(func() { shareRandRead = prev })

	if _, err := generateToken(); err == nil {
		t.Fatal("expected generateToken error")
	}
}
