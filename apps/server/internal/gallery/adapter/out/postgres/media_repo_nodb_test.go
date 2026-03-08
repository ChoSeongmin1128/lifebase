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

func TestMediaRepoNoDBErrorBranch(t *testing.T) {
	pool := newNoDBPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	repo := NewMediaRepo(pool)
	if repo == nil {
		t.Fatal("expected media repo to be constructed")
	}
	if _, err := repo.ListMedia(ctx, "u1", "image", "taken_at", "asc", "", 10); err == nil {
		t.Fatal("expected list media error")
	}
}

