package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUserRepoListUsersHookedScanError(t *testing.T) {
	prev := queryUserRowsFn
	t.Cleanup(func() { queryUserRowsFn = prev })

	queryUserRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakePushRows{next: []bool{true, false}, scan: errors.New("scan users fail")}, nil
	}

	repo := NewUserRepo(nil)
	if _, _, err := repo.ListUsers(context.Background(), "", "", 10); err == nil || !strings.Contains(err.Error(), "scan users fail") {
		t.Fatalf("expected scan users fail, got %v", err)
	}
}
