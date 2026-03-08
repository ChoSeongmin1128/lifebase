package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTodoRepoQueryHooksWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	lists := NewListRepo(pool)
	todos := NewTodoRepo(pool)

	prevQuery := queryTodoRowsFn
	t.Cleanup(func() { queryTodoRowsFn = prevQuery })

	queryTodoRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return &fakeScanRows{}, nil
	}
	if _, err := lists.ListByUser(context.Background(), "u1"); err != nil {
		t.Fatalf("ListByUser success hook: %v", err)
	}
	if _, err := todos.ListByList(context.Background(), "u1", "l1", false); err != nil {
		t.Fatalf("ListByList success hook: %v", err)
	}
	if _, err := todos.FindChildrenByParentID(context.Background(), "u1", "t1"); err != nil {
		t.Fatalf("FindChildrenByParentID success hook: %v", err)
	}

	queryTodoRowsFn = func(context.Context, *pgxpool.Pool, string, ...any) (pgx.Rows, error) {
		return nil, errors.New("query fail")
	}
	if _, err := lists.ListByUser(context.Background(), "u1"); err == nil {
		t.Fatal("expected ListByUser query error")
	}
	if _, err := todos.ListByList(context.Background(), "u1", "l1", true); err == nil {
		t.Fatal("expected ListByList query error")
	}
	if _, err := todos.FindChildrenByParentID(context.Background(), "u1", "t1"); err == nil {
		t.Fatal("expected FindChildrenByParentID query error")
	}
}

func TestTodoPushOutboxRepoHookBranchesWithoutDB(t *testing.T) {
	pool := newNoDBPool(t)
	repo := NewTodoPushOutboxRepo(pool)

	prevQuery := queryTodoOutboxAccountFn
	prevInsert := insertTodoOutboxFn
	t.Cleanup(func() {
		queryTodoOutboxAccountFn = prevQuery
		insertTodoOutboxFn = prevInsert
	})

	queryTodoOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return nil, pgx.ErrNoRows
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "t1", time.Now()); err != nil {
		t.Fatalf("ErrNoRows should be ignored: %v", err)
	}

	queryTodoOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return nil, errors.New("query fail")
	}
	if err := repo.EnqueueUpdate(context.Background(), "u1", "t1", time.Now()); err == nil {
		t.Fatal("expected outbox account query error")
	}

	empty := ""
	queryTodoOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return &empty, nil
	}
	if err := repo.EnqueueDelete(context.Background(), "u1", "t1", time.Now()); err != nil {
		t.Fatalf("empty account id should be ignored: %v", err)
	}

	accountID := "22222222-2222-2222-2222-222222222222"
	queryTodoOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return &accountID, nil
	}
	insertTodoOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return errors.New("insert fail")
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "t1", time.Now()); err == nil {
		t.Fatal("expected outbox insert error")
	}

	insertTodoOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return nil
	}
	if err := repo.EnqueueCreate(context.Background(), "u1", "t1", time.Now()); err != nil {
		t.Fatalf("expected outbox enqueue success: %v", err)
	}
}
