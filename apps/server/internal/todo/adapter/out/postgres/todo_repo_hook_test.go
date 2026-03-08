package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/testutil/dbtest"
	"lifebase/internal/todo/domain"
)

func TestTodoReposScanHookErrorBranches(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"

	lists := NewListRepo(db)
	todos := NewTodoRepo(db)

	list := &domain.TodoList{
		ID: "hook-list", UserID: userID, Name: "Hook", SortOrder: 0, CreatedAt: now, UpdatedAt: now,
	}
	if err := lists.Create(ctx, list); err != nil {
		t.Fatalf("create list: %v", err)
	}
	if err := todos.Create(ctx, &domain.Todo{
		ID: "hook-todo", ListID: list.ID, UserID: userID, Title: "Hook", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create todo: %v", err)
	}

	prevListScan := scanTodoListRowsFn
	prevTodoScan := scanTodosRowsFn
	t.Cleanup(func() {
		scanTodoListRowsFn = prevListScan
		scanTodosRowsFn = prevTodoScan
	})

	scanTodoListRowsFn = func(pgx.Rows) ([]*domain.TodoList, error) {
		return nil, errors.New("list scan fail")
	}
	if _, err := lists.ListByUser(ctx, userID); err == nil {
		t.Fatal("expected ListByUser scan hook error")
	}
	scanTodoListRowsFn = prevListScan

	scanTodosRowsFn = func(pgx.Rows) ([]*domain.Todo, error) {
		return nil, errors.New("todo scan fail")
	}
	if _, err := todos.ListByList(ctx, userID, list.ID, false); err == nil {
		t.Fatal("expected ListByList scan hook error")
	}
	if _, err := todos.FindChildrenByParentID(ctx, userID, "missing-parent"); err == nil {
		t.Fatal("expected FindChildrenByParentID scan hook error")
	}
}

func TestTodoPushOutboxRepoInvalidAccountIDInsertError(t *testing.T) {
	prevQuery := queryTodoOutboxAccountFn
	prevInsert := insertTodoOutboxFn
	t.Cleanup(func() {
		queryTodoOutboxAccountFn = prevQuery
		insertTodoOutboxFn = prevInsert
	})

	queryTodoOutboxAccountFn = func(context.Context, *pgxpool.Pool, string, string) (*string, error) {
		return strPtr("not-a-uuid"), nil
	}
	insertTodoOutboxFn = func(context.Context, *pgxpool.Pool, string, string, string, string, time.Time, time.Time) error {
		return errors.New("insert fail")
	}

	repo := NewTodoPushOutboxRepo(nil)
	if err := repo.EnqueueCreate(context.Background(), "user-1", "todo-bad-account", time.Now().UTC()); err == nil {
		t.Fatal("expected enqueue insert error")
	}
}
