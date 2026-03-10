package postgres

import (
	"context"
	"os"
	"strings"
	"testing"

	"lifebase/internal/testutil/dbtest"
)

func ensureSettingsTestDSN(t *testing.T) {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("LIFEBASE_TEST_DATABASE_URL"))
	if dsn == "" {
		dsn = "postgres://seongmin@localhost:5432/lifebase_test?sslmode=disable"
	}
	t.Setenv("LIFEBASE_TEST_DATABASE_URL", dsn)
}

func TestSettingsRepoCRUDCoverageWithDBTest(t *testing.T) {
	ensureSettingsTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	repo := NewSettingsRepo(db)
	if err := repo.Set(ctx, "coverage-user", "todo_default_sort", "due"); err != nil {
		t.Fatalf("set first key: %v", err)
	}
	if err := repo.Set(ctx, "coverage-user", "todo_default_sort", "manual"); err != nil {
		t.Fatalf("upsert key: %v", err)
	}
	if err := repo.Set(ctx, "coverage-user", "theme", "dark"); err != nil {
		t.Fatalf("set second key: %v", err)
	}

	items, err := repo.ListByUser(ctx, "coverage-user")
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(items))
	}
}

func TestSettingsRepoClosedPoolErrorCoverageWithDBTest(t *testing.T) {
	ensureSettingsTestDSN(t)
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewSettingsRepo(db)

	db.Close()
	if _, err := repo.ListByUser(ctx, "coverage-user"); err == nil {
		t.Fatal("expected list error on closed pool")
	}
	if err := repo.Set(ctx, "coverage-user", "k", "v"); err == nil {
		t.Fatal("expected set error on closed pool")
	}
}
