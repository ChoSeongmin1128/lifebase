package postgres

import (
	"context"
	"testing"
	"time"

	"lifebase/internal/auth/domain"
	"lifebase/internal/testutil/dbtest"
)

func TestUserRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	repo := NewUserRepo(db)

	u1 := &domain.User{
		ID:                "11111111-1111-1111-1111-111111111111",
		Email:             "alpha@example.com",
		Name:              "Alpha",
		Picture:           "p1",
		StorageQuotaBytes: 1000,
		StorageUsedBytes:  10,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	u2 := &domain.User{
		ID:                "22222222-2222-2222-2222-222222222222",
		Email:             "beta@example.com",
		Name:              "Beta",
		Picture:           "p2",
		StorageQuotaBytes: 2000,
		StorageUsedBytes:  20,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	u3 := &domain.User{
		ID:                "33333333-3333-3333-3333-333333333333",
		Email:             "gamma@example.com",
		Name:              "Gamma",
		Picture:           "p3",
		StorageQuotaBytes: 3000,
		StorageUsedBytes:  30,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	for _, u := range []*domain.User{u1, u2, u3} {
		if err := repo.Create(ctx, u); err != nil {
			t.Fatalf("create user %s: %v", u.Email, err)
		}
	}

	gotByEmail, err := repo.FindByEmail(ctx, u1.Email)
	if err != nil || gotByEmail.ID != u1.ID {
		t.Fatalf("find by email failed: err=%v user=%#v", err, gotByEmail)
	}
	if _, err := repo.FindByEmail(ctx, "missing@example.com"); err == nil {
		t.Fatal("expected missing email error")
	}

	gotByID, err := repo.FindByID(ctx, u2.ID)
	if err != nil || gotByID.Email != u2.Email {
		t.Fatalf("find by id failed: err=%v user=%#v", err, gotByID)
	}
	if _, err := repo.FindByID(ctx, "44444444-4444-4444-4444-444444444444"); err == nil {
		t.Fatal("expected missing id error")
	}

	limited, nextCursor, err := repo.ListUsers(ctx, "", "", 2)
	if err != nil || len(limited) != 2 || nextCursor == "" {
		t.Fatalf("list users with cursor failed: err=%v users=%d next=%q", err, len(limited), nextCursor)
	}
	searchResult, _, err := repo.ListUsers(ctx, "beta", "", -1)
	if err != nil || len(searchResult) != 1 || searchResult[0].Email != u2.Email {
		t.Fatalf("list users with search failed: err=%v users=%#v", err, searchResult)
	}
	afterCursor, _, err := repo.ListUsers(ctx, "", u1.ID, 10)
	if err != nil || len(afterCursor) == 0 {
		t.Fatalf("list users with cursor condition failed: err=%v users=%#v", err, afterCursor)
	}
	combined, _, err := repo.ListUsers(ctx, "a", u1.ID, 999)
	if err != nil {
		t.Fatalf("list users with search+cursor failed: %v", err)
	}
	if len(combined) == 0 {
		t.Fatal("expected at least one user for combined search+cursor path")
	}

	u1.Name = "Alpha Updated"
	u1.Picture = "p1-updated"
	u1.UpdatedAt = now.Add(time.Minute)
	if err := repo.Update(ctx, u1); err != nil {
		t.Fatalf("update user: %v", err)
	}
	updated, err := repo.FindByID(ctx, u1.ID)
	if err != nil || updated.Name != "Alpha Updated" || updated.Picture != "p1-updated" {
		t.Fatalf("updated user mismatch: err=%v user=%#v", err, updated)
	}

	if err := repo.UpdateStorageQuota(ctx, u1.ID, 9876); err != nil {
		t.Fatalf("update storage quota: %v", err)
	}
	if err := repo.UpdateStorageUsed(ctx, u1.ID, 5432); err != nil {
		t.Fatalf("update storage used: %v", err)
	}
	finalUser, err := repo.FindByID(ctx, u1.ID)
	if err != nil {
		t.Fatalf("find user after storage updates: %v", err)
	}
	if finalUser.StorageQuotaBytes != 9876 || finalUser.StorageUsedBytes != 5432 {
		t.Fatalf("storage fields mismatch: %#v", finalUser)
	}
}

func TestGoogleAccountRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	repo := NewGoogleAccountRepo(db)

	const userID = "11111111-1111-1111-1111-111111111111"
	acc1 := &domain.GoogleAccount{
		ID:             "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:         userID,
		GoogleEmail:    "u1@gmail.com",
		GoogleID:       "gid-1",
		AccessToken:    "at-1",
		RefreshToken:   "rt-1",
		TokenExpiresAt: timePtr(now.Add(time.Hour)),
		Scopes:         "scope",
		Status:         "active",
		IsPrimary:      true,
		ConnectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	acc2 := &domain.GoogleAccount{
		ID:             "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:         userID,
		GoogleEmail:    "u1+2@gmail.com",
		GoogleID:       "gid-2",
		AccessToken:    "at-2",
		RefreshToken:   "rt-2",
		TokenExpiresAt: timePtr(now.Add(2 * time.Hour)),
		Scopes:         "scope",
		Status:         "active",
		IsPrimary:      false,
		ConnectedAt:    now.Add(time.Minute),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := repo.Create(ctx, acc1); err != nil {
		t.Fatalf("create acc1: %v", err)
	}
	if err := repo.Create(ctx, acc2); err != nil {
		t.Fatalf("create acc2: %v", err)
	}

	gotByGoogleID, err := repo.FindByGoogleID(ctx, acc1.GoogleID)
	if err != nil || gotByGoogleID.ID != acc1.ID {
		t.Fatalf("find by google id failed: err=%v account=%#v", err, gotByGoogleID)
	}
	if _, err := repo.FindByGoogleID(ctx, "missing"); err == nil {
		t.Fatal("expected missing google id error")
	}

	gotByID, err := repo.FindByID(ctx, userID, acc2.ID)
	if err != nil || gotByID.GoogleID != acc2.GoogleID {
		t.Fatalf("find by id failed: err=%v account=%#v", err, gotByID)
	}
	if _, err := repo.FindByID(ctx, userID, "cccccccc-cccc-cccc-cccc-cccccccccccc"); err == nil {
		t.Fatal("expected missing account id error")
	}

	all, err := repo.FindByUserID(ctx, userID)
	if err != nil || len(all) != 2 {
		t.Fatalf("find by user failed: err=%v accounts=%#v", err, all)
	}
	if all[0].ID != acc1.ID {
		t.Fatalf("expected primary account first, got %#v", all)
	}

	acc2.AccessToken = "new-at"
	acc2.RefreshToken = "new-rt"
	acc2.Status = "reauth_required"
	acc2.UpdatedAt = now.Add(2 * time.Minute)
	if err := repo.Update(ctx, acc2); err != nil {
		t.Fatalf("update account: %v", err)
	}
	updated, err := repo.FindByID(ctx, userID, acc2.ID)
	if err != nil || updated.AccessToken != "new-at" || updated.Status != "reauth_required" {
		t.Fatalf("updated account mismatch: err=%v account=%#v", err, updated)
	}

	if accounts, err := repo.FindByUserID(ctx, "99999999-9999-9999-9999-999999999999"); err != nil || len(accounts) != 0 {
		t.Fatalf("expected empty account list for other user: err=%v accounts=%#v", err, accounts)
	}
}

func TestRefreshTokenRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	repo := NewRefreshTokenRepo(db)
	const userID = "11111111-1111-1111-1111-111111111111"

	rt1 := &domain.RefreshToken{
		ID:        "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:    userID,
		TokenHash: "hash-1",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	rt2 := &domain.RefreshToken{
		ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:    userID,
		TokenHash: "hash-2",
		ExpiresAt: now.Add(-time.Hour),
		CreatedAt: now,
	}
	if err := repo.Create(ctx, rt1); err != nil {
		t.Fatalf("create rt1: %v", err)
	}
	if err := repo.Create(ctx, rt2); err != nil {
		t.Fatalf("create rt2: %v", err)
	}

	got, err := repo.FindByHash(ctx, "hash-1")
	if err != nil || got.ID != rt1.ID {
		t.Fatalf("find by hash failed: err=%v token=%#v", err, got)
	}
	if _, err := repo.FindByHash(ctx, "missing"); err == nil {
		t.Fatal("expected missing hash error")
	}

	if err := repo.DeleteByHash(ctx, "hash-1"); err != nil {
		t.Fatalf("delete by hash: %v", err)
	}
	if _, err := repo.FindByHash(ctx, "hash-1"); err == nil {
		t.Fatal("expected hash-1 deleted")
	}

	if err := repo.DeleteExpired(ctx); err != nil {
		t.Fatalf("delete expired: %v", err)
	}
	if _, err := repo.FindByHash(ctx, "hash-2"); err == nil {
		t.Fatal("expected hash-2 expired and deleted")
	}

	rt3 := &domain.RefreshToken{
		ID:        "cccccccc-cccc-cccc-cccc-cccccccccccc",
		UserID:    userID,
		TokenHash: "hash-3",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := repo.Create(ctx, rt3); err != nil {
		t.Fatalf("create rt3: %v", err)
	}
	if err := repo.DeleteByUserID(ctx, userID); err != nil {
		t.Fatalf("delete by user id: %v", err)
	}
	if _, err := repo.FindByHash(ctx, "hash-3"); err == nil {
		t.Fatal("expected hash-3 deleted by user id")
	}
}

func TestAuthPostgresReposErrorBranchesOnClosedPool(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	userRepo := NewUserRepo(db)
	googleRepo := NewGoogleAccountRepo(db)
	refreshRepo := NewRefreshTokenRepo(db)

	db.Close()

	if _, err := userRepo.FindByEmail(ctx, "alpha@example.com"); err == nil {
		t.Fatal("expected FindByEmail error on closed pool")
	}
	if _, err := userRepo.FindByID(ctx, "11111111-1111-1111-1111-111111111111"); err == nil {
		t.Fatal("expected FindByID error on closed pool")
	}
	if _, _, err := userRepo.ListUsers(ctx, "", "", 10); err == nil {
		t.Fatal("expected ListUsers error on closed pool")
	}

	if _, err := googleRepo.FindByGoogleID(ctx, "gid-1"); err == nil {
		t.Fatal("expected FindByGoogleID error on closed pool")
	}
	if _, err := googleRepo.FindByID(ctx, "11111111-1111-1111-1111-111111111111", "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"); err == nil {
		t.Fatal("expected FindByID error on closed pool")
	}
	if _, err := googleRepo.FindByUserID(ctx, "11111111-1111-1111-1111-111111111111"); err == nil {
		t.Fatal("expected FindByUserID error on closed pool")
	}

	if _, err := refreshRepo.FindByHash(ctx, "hash-1"); err == nil {
		t.Fatal("expected FindByHash error on closed pool")
	}
}

func timePtr(v time.Time) *time.Time { return &v }
