package postgres

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func newUnreachablePool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), "postgres://lifebase:lifebase@127.0.0.1:1/lifebase?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestStorageResetRepoMethodsReturnErrorsWhenDBUnavailable(t *testing.T) {
	repo := NewStorageResetRepo(newUnreachablePool(t))
	ctx := context.Background()

	if _, err := repo.ListFilesByUser(ctx, "u1"); err == nil {
		t.Fatal("expected ListFilesByUser error")
	}
	if err := repo.DeleteAllFilesByUser(ctx, "u1"); err == nil {
		t.Fatal("expected DeleteAllFilesByUser error")
	}
	if err := repo.DeleteAllFoldersByUser(ctx, "u1"); err == nil {
		t.Fatal("expected DeleteAllFoldersByUser error")
	}
	if err := repo.DeleteAllStarsByUser(ctx, "u1"); err == nil {
		t.Fatal("expected DeleteAllStarsByUser error")
	}
	if err := repo.DeleteSharesByOwner(ctx, "u1"); err == nil {
		t.Fatal("expected DeleteSharesByOwner error")
	}
	if _, err := repo.SumStorageUsed(ctx, "u1"); err == nil {
		t.Fatal("expected SumStorageUsed error")
	}
}
