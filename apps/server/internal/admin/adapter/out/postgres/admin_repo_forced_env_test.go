package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	admindomain "lifebase/internal/admin/domain"
	"lifebase/internal/testutil/dbtest"
)

func ensureAdminTestDSN(t *testing.T) {
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

func TestAdminRepoCRUDCoverageWithDBTest(t *testing.T) {
	ensureAdminTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	userID := "11111111-1111-1111-1111-111111111111"
	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1, 'admin-coverage@example.com', 'admin', $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	repo := NewAdminRepo(db)
	admin := &admindomain.AdminUser{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:    userID,
		Role:      admindomain.RoleSuperAdmin,
		IsActive:  true,
		CreatedBy: userID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(ctx, admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}
	if ok, err := repo.IsActiveAdmin(ctx, userID); err != nil || !ok {
		t.Fatalf("is active admin mismatch: ok=%v err=%v", ok, err)
	}
	if _, err := repo.FindByUserID(ctx, userID); err != nil {
		t.Fatalf("find by user id: %v", err)
	}
	if _, err := repo.FindByID(ctx, admin.ID); err != nil {
		t.Fatalf("find by id: %v", err)
	}
	admin.Role = admindomain.RoleAdmin
	admin.IsActive = false
	admin.UpdatedAt = now.Add(time.Minute)
	if err := repo.Update(ctx, admin); err != nil {
		t.Fatalf("update admin: %v", err)
	}
	if rows, err := repo.List(ctx); err != nil || len(rows) != 1 {
		t.Fatalf("list admins mismatch: len=%d err=%v", len(rows), err)
	}
	if n, err := repo.CountActiveSuperAdmins(ctx); err != nil || n != 0 {
		t.Fatalf("count active super admins mismatch: n=%d err=%v", n, err)
	}

	accountID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'admin@gmail.com', 'gid-admin', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert google account: %v", err)
	}
	if rows, err := repo.ListByUserID(ctx, userID); err != nil || len(rows) != 1 {
		t.Fatalf("list google account rows mismatch: len=%d err=%v", len(rows), err)
	}
	if err := repo.UpdateStatus(ctx, accountID, userID, "revoked"); err != nil {
		t.Fatalf("update status: %v", err)
	}
}

func TestAdminRepoMissingAndStorageResetCoverageWithDBTest(t *testing.T) {
	ensureAdminTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	user1 := "11111111-1111-1111-1111-111111111111"
	user2 := "22222222-2222-2222-2222-222222222222"
	folderID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	fileID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	accountID := "cccccccc-cccc-cccc-cccc-cccccccccccc"

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1, 'u1@example.com', 'u1', $3, $3), ($2, 'u2@example.com', 'u2', $3, $3)`,
		user1, user2, now,
	)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO folders (id, user_id, name, created_at, updated_at)
		 VALUES ($1,$2,'F',$3,$3)`,
		folderID, user1, now,
	)
	if err != nil {
		t.Fatalf("insert folder: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, created_at, updated_at)
		 VALUES ($1,$2,$3,'a.txt','text/plain',123,'u1/a','done',$4,$4)`,
		fileID, user1, folderID, now,
	)
	if err != nil {
		t.Fatalf("insert file: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO shares (id, folder_id, owner_id, shared_with, role, created_at, updated_at)
		 VALUES ('share-1',$1,$2,'user-x','viewer',$3,$3)`,
		folderID, user1, now,
	)
	if err != nil {
		t.Fatalf("insert share: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO share_invites (id, folder_id, owner_id, token, role, expires_at, created_at)
		 VALUES ('invite-1',$1,$2,'tok','viewer',$3,$3)`,
		folderID, user1, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert invite: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES ($1, $2, 'u1@gmail.com', 'gid-1', 'at', 'rt', $3, 'scope', 'active', true, $3, $3, $3)`,
		accountID, user1, now,
	)
	if err != nil {
		t.Fatalf("insert google account: %v", err)
	}

	adminRepo := NewAdminRepo(db)
	if _, err := adminRepo.FindByUserID(ctx, user2); err == nil {
		t.Fatal("expected not found for missing admin by user")
	}
	if _, err := adminRepo.FindByID(ctx, "dddddddd-dddd-dddd-dddd-dddddddddddd"); err == nil {
		t.Fatal("expected not found for missing admin by id")
	}
	if err := adminRepo.UpdateStatus(ctx, accountID, user2, "active"); err == nil {
		t.Fatal("expected no rows for mismatched owner")
	}
	if rows, err := adminRepo.ListByUserID(ctx, user2); err != nil || len(rows) != 0 {
		t.Fatalf("expected empty list for user2: len=%d err=%v", len(rows), err)
	}

	storageRepo := NewStorageResetRepo(db)
	files, err := storageRepo.ListFilesByUser(ctx, user1)
	if err != nil || len(files) != 1 {
		t.Fatalf("list files by user mismatch: len=%d err=%v", len(files), err)
	}
	if err := storageRepo.DeleteSharesByOwner(ctx, user1); err != nil {
		t.Fatalf("delete shares by owner: %v", err)
	}
	if err := storageRepo.DeleteAllFilesByUser(ctx, user1); err != nil {
		t.Fatalf("delete files by user: %v", err)
	}
	if err := storageRepo.DeleteAllFoldersByUser(ctx, user1); err != nil {
		t.Fatalf("delete folders by user: %v", err)
	}
}

func TestAdminRepoClosedPoolErrorCoverageWithDBTest(t *testing.T) {
	ensureAdminTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	adminRepo := NewAdminRepo(db)
	storageRepo := NewStorageResetRepo(db)
	db.Close()

	if _, err := adminRepo.List(ctx); err == nil {
		t.Fatal("expected list query error on closed pool")
	}
	if _, err := adminRepo.ListByUserID(ctx, "u1"); err == nil {
		t.Fatal("expected list by user query error on closed pool")
	}
	if err := adminRepo.UpdateStatus(ctx, "a1", "u1", "active"); err == nil {
		t.Fatal("expected update status exec error on closed pool")
	}
	if _, err := adminRepo.FindByID(ctx, "a1"); err == nil {
		t.Fatal("expected find by id query error on closed pool")
	}
	if _, err := storageRepo.ListFilesByUser(ctx, "u1"); err == nil {
		t.Fatal("expected list files query error on closed pool")
	}
}
