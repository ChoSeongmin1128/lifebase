package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"lifebase/internal/todo/domain"
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

func TestTodoReposNoDBErrorBranches(t *testing.T) {
	pool := newNoDBPool(t)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	now := time.Now().UTC()

	lists := NewListRepo(pool)
	todos := NewTodoRepo(pool)
	outbox := NewTodoPushOutboxRepo(pool)

	if lists == nil || todos == nil || outbox == nil {
		t.Fatal("expected repos to be constructed")
	}

	if err := lists.Create(ctx, &domain.TodoList{ID: "l1", UserID: "u1", Name: "L1", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected list create error")
	}
	if _, err := lists.FindByID(ctx, "u1", "l1"); err == nil {
		t.Fatal("expected list find error")
	}
	if _, err := lists.ListByUser(ctx, "u1"); err == nil {
		t.Fatal("expected list list error")
	}
	if err := lists.Update(ctx, &domain.TodoList{ID: "l1", UserID: "u1", Name: "L1", UpdatedAt: now}); err == nil {
		t.Fatal("expected list update error")
	}
	if err := lists.Delete(ctx, "l1"); err == nil {
		t.Fatal("expected list delete error")
	}

	if err := todos.Create(ctx, &domain.Todo{ID: "t1", ListID: "l1", UserID: "u1", Title: "T1", CreatedAt: now, UpdatedAt: now}); err == nil {
		t.Fatal("expected todo create error")
	}
	if _, err := todos.FindByID(ctx, "u1", "t1"); err == nil {
		t.Fatal("expected todo find error")
	}
	if _, err := todos.ListByList(ctx, "u1", "l1", true); err == nil {
		t.Fatal("expected todo list error")
	}
	if err := todos.Update(ctx, &domain.Todo{ID: "t1", UserID: "u1", ListID: "l1", Title: "T1", UpdatedAt: now}); err == nil {
		t.Fatal("expected todo update error")
	}
	if err := todos.SoftDelete(ctx, "u1", "t1"); err == nil {
		t.Fatal("expected todo soft delete error")
	}
	if _, err := todos.CountPinned(ctx, "u1", "l1"); err == nil {
		t.Fatal("expected todo count pinned error")
	}
	if _, err := todos.FindChildrenByParentID(ctx, "u1", "t1"); err == nil {
		t.Fatal("expected todo find children error")
	}
	if err := todos.SoftDeleteByParentID(ctx, "u1", "t1"); err == nil {
		t.Fatal("expected todo soft delete by parent error")
	}
	if err := todos.UpdateBatch(ctx, []*domain.Todo{{ID: "t1", UserID: "u1", UpdatedAt: now}}); err == nil {
		t.Fatal("expected todo update batch error")
	}
	if _, err := todos.NextSortOrder(ctx, "u1", "l1", nil); err == nil {
		t.Fatal("expected todo next sort order(root) error")
	}
	parentID := "t1"
	if _, err := todos.NextSortOrder(ctx, "u1", "l1", &parentID); err == nil {
		t.Fatal("expected todo next sort order(child) error")
	}

	if err := outbox.EnqueueCreate(ctx, "u1", "t1", now); err == nil {
		t.Fatal("expected outbox enqueue create error")
	}
	if err := outbox.EnqueueUpdate(ctx, "u1", "t1", now); err == nil {
		t.Fatal("expected outbox enqueue update error")
	}
	if err := outbox.EnqueueDelete(ctx, "u1", "t1", now); err == nil {
		t.Fatal("expected outbox enqueue delete error")
	}
}

