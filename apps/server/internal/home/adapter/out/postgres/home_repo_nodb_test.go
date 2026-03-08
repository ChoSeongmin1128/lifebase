package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newNoDBPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://localhost:1/lifebase_test?sslmode=disable")
	if err != nil {
		t.Fatalf("new pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestHomeRepoNoDBErrorBranches(t *testing.T) {
	pool := newNoDBPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	repo := NewHomeRepo(pool)
	if repo == nil {
		t.Fatal("expected home repo to be constructed")
	}

	start := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	end := time.Now().UTC().Format(time.RFC3339)
	today := time.Now().UTC().Format("2006-01-02")

	if _, _, err := repo.ListEventsInRange(ctx, "u1", start, end, 10); err == nil {
		t.Fatal("expected ListEventsInRange error")
	}
	if _, _, err := repo.ListOverdueTodos(ctx, "u1", today, 10); err == nil {
		t.Fatal("expected ListOverdueTodos error")
	}
	if _, _, err := repo.ListTodayTodos(ctx, "u1", today, 10); err == nil {
		t.Fatal("expected ListTodayTodos error")
	}
	if _, _, err := repo.ListRecentFiles(ctx, "u1", 10); err == nil {
		t.Fatal("expected ListRecentFiles error")
	}
	if _, err := repo.GetStorageSummary(ctx, "u1"); err == nil {
		t.Fatal("expected GetStorageSummary error")
	}
	if _, err := repo.ListStorageTypeUsage(ctx, "u1"); err == nil {
		t.Fatal("expected ListStorageTypeUsage error")
	}
}

