package postgres

import (
	"context"
	"testing"

	"lifebase/internal/testutil/dbtest"
)

func TestSettingsRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)

	ctx := context.Background()
	repo := NewSettingsRepo(db)

	if err := repo.Set(ctx, "u1", "calendar_theme", "compact"); err != nil {
		t.Fatalf("set setting: %v", err)
	}
	if err := repo.Set(ctx, "u1", "calendar_theme", "timeline"); err != nil {
		t.Fatalf("update setting: %v", err)
	}
	if err := repo.Set(ctx, "u1", "todo_view", "list"); err != nil {
		t.Fatalf("set second setting: %v", err)
	}

	items, err := repo.ListByUser(ctx, "u1")
	if err != nil {
		t.Fatalf("list by user: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 settings, got %d", len(items))
	}

	kv := map[string]string{}
	for _, it := range items {
		kv[it.Key] = it.Value
	}
	if kv["calendar_theme"] != "timeline" {
		t.Fatalf("expected updated value timeline, got %q", kv["calendar_theme"])
	}
	if kv["todo_view"] != "list" {
		t.Fatalf("expected todo_view=list, got %q", kv["todo_view"])
	}

	empty, err := repo.ListByUser(ctx, "u2")
	if err != nil {
		t.Fatalf("list by user (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected no settings for other user, got %d", len(empty))
	}
}

func TestSettingsRepoErrorPaths(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	repo := NewSettingsRepo(db)
	db.Close()

	if _, err := repo.ListByUser(ctx, "u1"); err == nil {
		t.Fatal("expected list error on closed pool")
	}
	if err := repo.Set(ctx, "u1", "k", "v"); err == nil {
		t.Fatal("expected set error on closed pool")
	}
}
