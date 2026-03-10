package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"lifebase/internal/auth/domain"
	"lifebase/internal/testutil/dbtest"
)

func ensureAuthTestDSN(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgres://seongmin@localhost:5432/lifebase_test?sslmode=disable"
	}
	t.Setenv("LIFEBASE_TEST_DATABASE_URL", dsn)
}

func TestAuthReposCRUDCoverageWithDBTest(t *testing.T) {
	ensureAuthTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	userRepo := NewUserRepo(db)
	googleRepo := NewGoogleAccountRepo(db)
	refreshRepo := NewRefreshTokenRepo(db)

	user := &domain.User{
		ID:                "11111111-1111-1111-1111-111111111111",
		Email:             "crud-coverage@example.com",
		Name:              "Coverage",
		Picture:           "p1",
		StorageQuotaBytes: 100,
		StorageUsedBytes:  1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if _, err := userRepo.FindByEmail(ctx, user.Email); err != nil {
		t.Fatalf("find by email: %v", err)
	}
	if _, err := userRepo.FindByID(ctx, user.ID); err != nil {
		t.Fatalf("find by id: %v", err)
	}
	user.Name = "Coverage Updated"
	user.Picture = "p2"
	user.UpdatedAt = now.Add(time.Minute)
	if err := userRepo.Update(ctx, user); err != nil {
		t.Fatalf("update user: %v", err)
	}
	if err := userRepo.UpdateStorageQuota(ctx, user.ID, 999); err != nil {
		t.Fatalf("update storage quota: %v", err)
	}
	if err := userRepo.UpdateStorageUsed(ctx, user.ID, 777); err != nil {
		t.Fatalf("update storage used: %v", err)
	}
	if list, next, err := userRepo.ListUsers(ctx, "coverage", "", 1); err != nil || len(list) != 1 || next != "" {
		t.Fatalf("list users mismatch: len=%d next=%q err=%v", len(list), next, err)
	}

	account := &domain.GoogleAccount{
		ID:             "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		UserID:         user.ID,
		GoogleEmail:    "coverage@gmail.com",
		GoogleID:       "gid-coverage",
		AccessToken:    "at",
		RefreshToken:   "rt",
		TokenExpiresAt: ptrTime(now.Add(time.Hour)),
		Scopes:         "scope",
		Status:         "active",
		IsPrimary:      true,
		ConnectedAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := googleRepo.Create(ctx, account); err != nil {
		t.Fatalf("create google account: %v", err)
	}
	if _, err := googleRepo.FindByGoogleID(ctx, account.GoogleID); err != nil {
		t.Fatalf("find by google id: %v", err)
	}
	if _, err := googleRepo.FindByID(ctx, user.ID, account.ID); err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if rows, err := googleRepo.FindByUserID(ctx, user.ID); err != nil || len(rows) != 1 {
		t.Fatalf("find by user mismatch: len=%d err=%v", len(rows), err)
	}
	account.AccessToken = "at2"
	account.RefreshToken = "rt2"
	account.Status = "reauth_required"
	account.UpdatedAt = now.Add(2 * time.Minute)
	if err := googleRepo.Update(ctx, account); err != nil {
		t.Fatalf("update google account: %v", err)
	}

	rt := &domain.RefreshToken{
		ID:        "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		UserID:    user.ID,
		TokenHash: "coverage-hash",
		ExpiresAt: now.Add(time.Hour),
		CreatedAt: now,
	}
	if err := refreshRepo.Create(ctx, rt); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}
	if _, err := refreshRepo.FindByHash(ctx, rt.TokenHash); err != nil {
		t.Fatalf("find by hash: %v", err)
	}
	if err := refreshRepo.DeleteByHash(ctx, rt.TokenHash); err != nil {
		t.Fatalf("delete by hash: %v", err)
	}
	if err := refreshRepo.DeleteExpired(ctx); err != nil {
		t.Fatalf("delete expired: %v", err)
	}
	if err := refreshRepo.DeleteByUserID(ctx, user.ID); err != nil {
		t.Fatalf("delete by user id: %v", err)
	}
}

func ptrTime(v time.Time) *time.Time { return &v }
