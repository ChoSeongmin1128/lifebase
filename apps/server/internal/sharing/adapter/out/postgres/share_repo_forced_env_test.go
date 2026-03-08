package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"lifebase/internal/sharing/domain"
	"lifebase/internal/testutil/dbtest"
)

func ensureSharingTestDSN(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("DATABASE_URL"))
	}
	if dsn == "" {
		dsn = "postgres://seongmin@localhost:5432/lifebase?sslmode=disable"
	}
	t.Setenv("LIFEBASE_TEST_DATABASE_URL", dsn)
}

func TestSharingReposCRUDCoverageWithDBTest(t *testing.T) {
	ensureSharingTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	shareRepo := NewShareRepo(db)
	inviteRepo := NewInviteRepo(db)

	share := &domain.Share{
		ID:         "share-coverage",
		FolderID:   "folder-coverage",
		OwnerID:    "owner-coverage",
		SharedWith: "viewer-coverage",
		Role:       "viewer",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := shareRepo.Create(ctx, share); err != nil {
		t.Fatalf("create share: %v", err)
	}
	if _, err := shareRepo.FindByID(ctx, share.ID); err != nil {
		t.Fatalf("find share by id: %v", err)
	}
	if rows, err := shareRepo.ListByFolder(ctx, share.FolderID); err != nil || len(rows) != 1 {
		t.Fatalf("list by folder mismatch: len=%d err=%v", len(rows), err)
	}
	if rows, err := shareRepo.ListByUser(ctx, share.SharedWith); err != nil || len(rows) != 1 {
		t.Fatalf("list by user mismatch: len=%d err=%v", len(rows), err)
	}
	if err := shareRepo.Delete(ctx, share.ID); err != nil {
		t.Fatalf("delete share: %v", err)
	}

	invite := &domain.ShareInvite{
		ID:        "invite-coverage",
		FolderID:  "folder-coverage",
		OwnerID:   "owner-coverage",
		Token:     "token-coverage",
		Role:      "editor",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := inviteRepo.Create(ctx, invite); err != nil {
		t.Fatalf("create invite: %v", err)
	}
	if _, err := inviteRepo.FindByToken(ctx, invite.Token); err != nil {
		t.Fatalf("find invite by token: %v", err)
	}
	if err := inviteRepo.MarkAccepted(ctx, invite.ID); err != nil {
		t.Fatalf("mark accepted: %v", err)
	}
}

func TestSharingReposMissingAndClosedPoolCoverageWithDBTest(t *testing.T) {
	ensureSharingTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	shareRepo := NewShareRepo(db)
	inviteRepo := NewInviteRepo(db)

	if _, err := shareRepo.FindByID(ctx, "missing-share"); err == nil {
		t.Fatal("expected not found for missing share")
	}
	if _, err := inviteRepo.FindByToken(ctx, "missing-token"); err == nil {
		t.Fatal("expected not found for missing invite")
	}

	share := &domain.Share{
		ID:         "share-error-cover",
		FolderID:   "folder-error-cover",
		OwnerID:    "owner-error-cover",
		SharedWith: "viewer-error-cover",
		Role:       "viewer",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := shareRepo.Create(ctx, share); err != nil {
		t.Fatalf("create share: %v", err)
	}

	db.Close()
	if _, err := shareRepo.FindByID(ctx, share.ID); err == nil {
		t.Fatal("expected find by id query error on closed pool")
	}
	if _, err := shareRepo.ListByFolder(ctx, share.FolderID); err == nil {
		t.Fatal("expected list by folder query error on closed pool")
	}
	if _, err := shareRepo.ListByUser(ctx, share.SharedWith); err == nil {
		t.Fatal("expected list by user query error on closed pool")
	}
	if _, err := inviteRepo.FindByToken(ctx, "token-any"); err == nil {
		t.Fatal("expected find by token query error on closed pool")
	}
}
