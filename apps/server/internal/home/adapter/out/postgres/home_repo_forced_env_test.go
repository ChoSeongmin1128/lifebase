package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"lifebase/internal/testutil/dbtest"
)

func ensureHomeTestDSN(t *testing.T) {
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

func TestHomeRepoCoverageWithForcedDBEnv(t *testing.T) {
	ensureHomeTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	repo := NewHomeRepo(db)
	userID := "11111111-1111-1111-1111-111111111111"
	now := time.Now().UTC().Truncate(time.Second)
	today := now.Format("2006-01-02")

	_, err := db.Exec(ctx,
		`INSERT INTO users (id, email, name, storage_quota_bytes, storage_used_bytes, created_at, updated_at)
		 VALUES ($1, 'home-coverage@example.com', 'home-coverage', 1000, 200, $2, $2)`,
		userID, now,
	)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}

	events, total, err := repo.ListEventsInRange(
		ctx,
		userID,
		now.Add(-time.Hour).Format(time.RFC3339),
		now.Add(time.Hour).Format(time.RFC3339),
		10,
	)
	if err != nil || total != 0 || len(events) != 0 {
		t.Fatalf("events should be empty success path: total=%d len=%d err=%v", total, len(events), err)
	}

	overdue, overdueTotal, err := repo.ListOverdueTodos(ctx, userID, today, 10)
	if err != nil || overdueTotal != 0 || len(overdue) != 0 {
		t.Fatalf("overdue todos should be empty success path: total=%d len=%d err=%v", overdueTotal, len(overdue), err)
	}

	todayTodos, todayTotal, err := repo.ListTodayTodos(ctx, userID, today, 10)
	if err != nil || todayTotal != 0 || len(todayTodos) != 0 {
		t.Fatalf("today todos should be empty success path: total=%d len=%d err=%v", todayTotal, len(todayTodos), err)
	}

	recent, recentTotal, err := repo.ListRecentFiles(ctx, userID, 10)
	if err != nil || recentTotal != 0 || len(recent) != 0 {
		t.Fatalf("recent files should be empty success path: total=%d len=%d err=%v", recentTotal, len(recent), err)
	}

	summary, err := repo.GetStorageSummary(ctx, userID)
	if err != nil {
		t.Fatalf("get storage summary: %v", err)
	}
	if summary.UsedBytes != 200 || summary.QuotaBytes != 1000 {
		t.Fatalf("unexpected storage summary: %#v", summary)
	}

	if _, err := repo.GetStorageSummary(ctx, "22222222-2222-2222-2222-222222222222"); err == nil {
		t.Fatal("expected not found error for missing user")
	}

	usage, err := repo.ListStorageTypeUsage(ctx, userID)
	if err != nil {
		t.Fatalf("list storage type usage: %v", err)
	}
	if len(usage) != 0 {
		t.Fatalf("expected empty usage rows, got %#v", usage)
	}
}

