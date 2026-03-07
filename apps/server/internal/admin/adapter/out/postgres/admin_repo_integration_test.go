package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	admindomain "lifebase/internal/admin/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestAdminRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	user1 := "11111111-1111-1111-1111-111111111111"
	user2 := "22222222-2222-2222-2222-222222222222"
	accountID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1,'u1@example.com','u1',$3,$3),($2,'u2@example.com','u2',$3,$3)`,
		user1, user2, now,
	)
	if err != nil {
		t.Fatalf("insert users: %v", err)
	}

	repo := NewAdminRepo(db)
	admin := &admindomain.AdminUser{
		ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:    user1,
		Role:      admindomain.RoleSuperAdmin,
		IsActive:  true,
		CreatedBy: user1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := repo.Create(ctx, admin); err != nil {
		t.Fatalf("create admin: %v", err)
	}

	active, err := repo.IsActiveAdmin(ctx, user1)
	if err != nil || !active {
		t.Fatalf("is active admin failed: active=%v err=%v", active, err)
	}
	active, err = repo.IsActiveAdmin(ctx, user2)
	if err != nil || active {
		t.Fatalf("is active admin for non-admin mismatch: active=%v err=%v", active, err)
	}

	foundByUser, err := repo.FindByUserID(ctx, user1)
	if err != nil || foundByUser.ID != admin.ID {
		t.Fatalf("find by user failed: %v %#v", err, foundByUser)
	}
	if _, err := repo.FindByUserID(ctx, user2); err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows for missing admin by user, got %v", err)
	}

	foundByID, err := repo.FindByID(ctx, admin.ID)
	if err != nil || foundByID.UserID != user1 {
		t.Fatalf("find by id failed: %v %#v", err, foundByID)
	}
	if _, err := repo.FindByID(ctx, "cccccccc-cccc-cccc-cccc-cccccccccccc"); err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows for missing admin by id, got %v", err)
	}

	admin.Role = admindomain.RoleAdmin
	admin.IsActive = false
	admin.UpdatedAt = now.Add(time.Minute)
	if err := repo.Update(ctx, admin); err != nil {
		t.Fatalf("update admin: %v", err)
	}

	all, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("list admins: %v", err)
	}
	if len(all) != 1 || all[0].Role != admindomain.RoleAdmin || all[0].IsActive {
		t.Fatalf("unexpected admin rows after update: %#v", all)
	}

	count, err := repo.CountActiveSuperAdmins(ctx)
	if err != nil {
		t.Fatalf("count super admins: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 active super admins, got %d", count)
	}

	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($1,$2,'u1@gmail.com','gid-1','at','rt',$3,'scope','active',true,$3,$3,$3)`,
		accountID, user1, now,
	)
	if err != nil {
		t.Fatalf("insert google account: %v", err)
	}

	records, err := repo.ListByUserID(ctx, user1)
	if err != nil {
		t.Fatalf("list google accounts by user: %v", err)
	}
	if len(records) != 1 || records[0].GoogleEmail != "u1@gmail.com" {
		t.Fatalf("unexpected google account records: %#v", records)
	}
	emptyRecords, err := repo.ListByUserID(ctx, user2)
	if err != nil {
		t.Fatalf("list google accounts by non-linked user: %v", err)
	}
	if len(emptyRecords) != 0 {
		t.Fatalf("expected no google account rows for user2, got %#v", emptyRecords)
	}

	if err := repo.UpdateStatus(ctx, accountID, user1, "revoked"); err != nil {
		t.Fatalf("update google account status: %v", err)
	}
	if err := repo.UpdateStatus(ctx, accountID, user2, "active"); err != pgx.ErrNoRows {
		t.Fatalf("expected pgx.ErrNoRows when account owner mismatch, got %v", err)
	}
}

func TestStorageResetRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	userID := "11111111-1111-1111-1111-111111111111"
	folderID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	fileID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, created_at, updated_at)
		 VALUES ($1,'u1@example.com','u1',$2,$2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO folders (id, user_id, name, created_at, updated_at)
		 VALUES ($1,$2,'F',$3,$3)`,
		folderID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert folder: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO files (id, user_id, folder_id, name, mime_type, size_bytes, storage_path, thumb_status, created_at, updated_at)
		 VALUES ($1,$2,$3,'a.txt','text/plain',123,'u1/a','done',$4,$4)`,
		fileID, userID, folderID, now,
	)
	if err != nil {
		t.Fatalf("insert file: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO shares (id, folder_id, owner_id, shared_with, role, created_at, updated_at)
		 VALUES ('share-1',$1,$2,'user-x','viewer',$3,$3)`,
		folderID, userID, now,
	)
	if err != nil {
		t.Fatalf("insert share: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO share_invites (id, folder_id, owner_id, token, role, expires_at, created_at)
		 VALUES ('invite-1',$1,$2,'tok','viewer',$3,$3)`,
		folderID, userID, now.Add(time.Hour),
	)
	if err != nil {
		t.Fatalf("insert invite: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO user_settings (user_id, key, value, updated_at)
		 VALUES ($1, 'cloud_star:file:' || $2, '1', $3)`,
		userID, fileID, now,
	)
	if err != nil {
		t.Fatalf("insert star setting: %v", err)
	}

	repo := NewStorageResetRepo(db)

	files, err := repo.ListFilesByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list files by user: %v", err)
	}
	if len(files) != 1 || files[0].ID != fileID {
		t.Fatalf("unexpected files by user: %#v", files)
	}

	used, err := repo.SumStorageUsed(ctx, userID)
	if err != nil || used != 123 {
		t.Fatalf("sum storage used mismatch: used=%d err=%v", used, err)
	}

	if err := repo.DeleteAllStarsByUser(ctx, userID); err != nil {
		t.Fatalf("delete stars by user: %v", err)
	}
	if err := repo.DeleteSharesByOwner(ctx, userID); err != nil {
		t.Fatalf("delete shares by owner: %v", err)
	}
	if err := repo.DeleteAllFilesByUser(ctx, userID); err != nil {
		t.Fatalf("delete files by user: %v", err)
	}
	if err := repo.DeleteAllFoldersByUser(ctx, userID); err != nil {
		t.Fatalf("delete folders by user: %v", err)
	}

	used, err = repo.SumStorageUsed(ctx, userID)
	if err != nil || used != 0 {
		t.Fatalf("storage used should be zero after delete: used=%d err=%v", used, err)
	}
}

func TestAdminAndStorageReposErrorBranches(t *testing.T) {
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
