package postgres

import (
	"context"
	"strings"
	"testing"
	"time"

	"lifebase/internal/sharing/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestShareRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	ctx := context.Background()
	repo := NewShareRepo(db)

	now := time.Now().UTC().Truncate(time.Second)
	share := &domain.Share{
		ID:         "share-1",
		FolderID:   "folder-1",
		OwnerID:    "owner-1",
		SharedWith: "user-2",
		Role:       "viewer",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := repo.Create(ctx, share); err != nil {
		t.Fatalf("create share: %v", err)
	}

	got, err := repo.FindByID(ctx, share.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if got.SharedWith != "user-2" || got.Role != "viewer" {
		t.Fatalf("unexpected share: %#v", got)
	}

	byFolder, err := repo.ListByFolder(ctx, "folder-1")
	if err != nil {
		t.Fatalf("list by folder: %v", err)
	}
	if len(byFolder) != 1 || byFolder[0].ID != share.ID {
		t.Fatalf("unexpected by-folder list: %#v", byFolder)
	}

	byUser, err := repo.ListByUser(ctx, "user-2")
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(byUser) != 1 || byUser[0].ID != share.ID {
		t.Fatalf("unexpected by-user list: %#v", byUser)
	}

	if err := repo.Delete(ctx, share.ID); err != nil {
		t.Fatalf("delete share: %v", err)
	}
	if _, err := repo.FindByID(ctx, share.ID); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestInviteRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	ctx := context.Background()
	repo := NewInviteRepo(db)
	now := time.Now().UTC().Truncate(time.Second)

	invite := &domain.ShareInvite{
		ID:        "invite-1",
		FolderID:  "folder-1",
		OwnerID:   "owner-1",
		Token:     "token-123",
		Role:      "editor",
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	}
	if err := repo.Create(ctx, invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}

	got, err := repo.FindByToken(ctx, invite.Token)
	if err != nil {
		t.Fatalf("find by token: %v", err)
	}
	if got.ID != invite.ID || got.Role != "editor" {
		t.Fatalf("unexpected invite: %#v", got)
	}
	if got.AcceptedAt != nil {
		t.Fatalf("accepted_at should be nil before mark accepted: %#v", got.AcceptedAt)
	}

	if err := repo.MarkAccepted(ctx, invite.ID); err != nil {
		t.Fatalf("mark accepted: %v", err)
	}
	got, err = repo.FindByToken(ctx, invite.Token)
	if err != nil {
		t.Fatalf("find after mark accepted: %v", err)
	}
	if got.AcceptedAt == nil {
		t.Fatal("accepted_at should be set")
	}

	if _, err := repo.FindByToken(ctx, "missing-token"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected invite not found error, got %v", err)
	}
}

func TestShareRepoErrorBranchesOnClosedPool(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	repo := NewShareRepo(db)
	db.Close()

	if _, err := repo.ListByFolder(ctx, "folder-1"); err == nil {
		t.Fatal("expected ListByFolder error on closed pool")
	}
	if _, err := repo.ListByUser(ctx, "user-1"); err == nil {
		t.Fatal("expected ListByUser error on closed pool")
	}
}
