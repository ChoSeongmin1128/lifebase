package postgres

import (
	"context"
	"testing"
	"time"

	"lifebase/internal/testutil/dbtest"
	"lifebase/internal/todo/domain"
)

func TestTodoListAndTodoRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "22222222-2222-2222-2222-222222222222"

	lists := NewListRepo(db)
	todos := NewTodoRepo(db)

	list := &domain.TodoList{
		ID:              "list-1",
		UserID:          userID,
		Name:            "My List",
		SortOrder:       1,
		GoogleID:        strPtr("g-list"),
		GoogleAccountID: strPtr(accountID),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := lists.Create(ctx, list); err != nil {
		t.Fatalf("create list: %v", err)
	}

	gotList, err := lists.FindByID(ctx, userID, list.ID)
	if err != nil {
		t.Fatalf("find list by id: %v", err)
	}
	if gotList.Name != "My List" || gotList.GoogleAccountID == nil {
		t.Fatalf("unexpected list row: %#v", gotList)
	}
	if _, err := lists.FindByID(ctx, userID, "missing"); err == nil {
		t.Fatal("expected list not found")
	}
	list.Name = "My List Updated"
	list.SortOrder = 2
	list.UpdatedAt = now.Add(30 * time.Second)
	if err := lists.Update(ctx, list); err != nil {
		t.Fatalf("update list: %v", err)
	}
	updatedList, err := lists.FindByID(ctx, userID, list.ID)
	if err != nil || updatedList.Name != "My List Updated" || updatedList.SortOrder != 2 {
		t.Fatalf("updated list mismatch: %v %#v", err, updatedList)
	}

	// row for google email join path
	_, err = db.Exec(ctx,
		`INSERT INTO user_google_accounts
		    (id, user_id, google_email, google_id, access_token, refresh_token, token_expires_at, scopes, status, is_primary, connected_at, created_at, updated_at)
		 VALUES
		    ($2,$3,'u1@gmail.com','gid','at','rt',$1,'s','active',true,$1,$1,$1)`,
		now, accountID, userID,
	)
	if err != nil {
		t.Fatalf("insert google account for join: %v", err)
	}

	rootTodo := &domain.Todo{
		ID:        "todo-1",
		ListID:    list.ID,
		UserID:    userID,
		Title:     "Root",
		Notes:     "N",
		Due:       strPtr("2026-03-07"),
		Priority:  "high",
		IsDone:    false,
		IsPinned:  true,
		SortOrder: 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := todos.Create(ctx, rootTodo); err != nil {
		t.Fatalf("create todo: %v", err)
	}
	child := &domain.Todo{
		ID:        "todo-2",
		ListID:    list.ID,
		UserID:    userID,
		ParentID:  &rootTodo.ID,
		Title:     "Child",
		Notes:     "",
		Priority:  "normal",
		IsDone:    false,
		IsPinned:  false,
		SortOrder: 2,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := todos.Create(ctx, child); err != nil {
		t.Fatalf("create child todo: %v", err)
	}

	allLists, err := lists.ListByUser(ctx, userID)
	if err != nil {
		t.Fatalf("list lists by user: %v", err)
	}
	if len(allLists) != 1 || allLists[0].TotalCount != 2 {
		t.Fatalf("unexpected list stats: %#v", allLists)
	}
	emptyLists, err := lists.ListByUser(ctx, "other-user")
	if err != nil {
		t.Fatalf("list lists by other user: %v", err)
	}
	if len(emptyLists) != 0 {
		t.Fatalf("expected no lists for other user, got %#v", emptyLists)
	}

	foundTodo, err := todos.FindByID(ctx, userID, rootTodo.ID)
	if err != nil || foundTodo.ID != rootTodo.ID || foundTodo.Due == nil {
		t.Fatalf("find todo by id failed: %v %#v", err, foundTodo)
	}
	if _, err := todos.FindByID(ctx, userID, "missing"); err == nil {
		t.Fatal("expected todo not found")
	}

	listed, err := todos.ListByList(ctx, userID, list.ID, true)
	if err != nil || len(listed) != 2 {
		t.Fatalf("list by list include done failed: %v len=%d", err, len(listed))
	}
	listedUndone, err := todos.ListByList(ctx, userID, list.ID, false)
	if err != nil || len(listedUndone) != 2 {
		t.Fatalf("list by list includeDone=false failed: %v len=%d", err, len(listedUndone))
	}
	emptyTodos, err := todos.ListByList(ctx, userID, "missing-list", true)
	if err != nil {
		t.Fatalf("list by list empty failed: %v", err)
	}
	if len(emptyTodos) != 0 {
		t.Fatalf("expected empty todos for missing list, got %#v", emptyTodos)
	}

	rootTodo.IsDone = true
	rootTodo.DoneAt = &now
	rootTodo.Priority = "urgent"
	rootTodo.Title = "Updated"
	rootTodo.ParentID = nil
	rootTodo.UpdatedAt = now.Add(time.Minute)
	if err := todos.Update(ctx, rootTodo); err != nil {
		t.Fatalf("update todo: %v", err)
	}
	listedUndone, err = todos.ListByList(ctx, userID, list.ID, false)
	if err != nil || len(listedUndone) != 1 || listedUndone[0].ID != child.ID {
		t.Fatalf("list by list exclude done mismatch: %v %#v", err, listedUndone)
	}

	pinnedCount, err := todos.CountPinned(ctx, userID, list.ID)
	if err != nil || pinnedCount != 1 {
		t.Fatalf("count pinned mismatch: count=%d err=%v", pinnedCount, err)
	}

	children, err := todos.FindChildrenByParentID(ctx, userID, rootTodo.ID)
	if err != nil || len(children) != 1 || children[0].ID != child.ID {
		t.Fatalf("find children failed: %v %#v", err, children)
	}
	noChildren, err := todos.FindChildrenByParentID(ctx, userID, "missing-parent")
	if err != nil {
		t.Fatalf("find children empty failed: %v", err)
	}
	if len(noChildren) != 0 {
		t.Fatalf("expected no children for missing parent, got %#v", noChildren)
	}

	nextRootSort, err := todos.NextSortOrder(ctx, userID, list.ID, nil)
	if err != nil || nextRootSort < 0 {
		t.Fatalf("next root sort order failed: %d %v", nextRootSort, err)
	}
	nextChildSort, err := todos.NextSortOrder(ctx, userID, list.ID, &rootTodo.ID)
	if err != nil || nextChildSort < 0 {
		t.Fatalf("next child sort order failed: %d %v", nextChildSort, err)
	}

	updates := []*domain.Todo{
		{ID: rootTodo.ID, UserID: userID, ParentID: nil, SortOrder: 5, UpdatedAt: now.Add(2 * time.Minute)},
		{ID: child.ID, UserID: userID, ParentID: &rootTodo.ID, SortOrder: 6, UpdatedAt: now.Add(2 * time.Minute)},
	}
	if err := todos.UpdateBatch(ctx, updates); err != nil {
		t.Fatalf("update batch: %v", err)
	}

	if err := todos.SoftDeleteByParentID(ctx, userID, rootTodo.ID); err != nil {
		t.Fatalf("soft delete by parent id: %v", err)
	}
	afterChildDelete, err := todos.FindChildrenByParentID(ctx, userID, rootTodo.ID)
	if err != nil {
		t.Fatalf("find children after soft delete: %v", err)
	}
	if len(afterChildDelete) != 0 {
		t.Fatalf("child should be soft deleted, got %#v", afterChildDelete)
	}

	if err := todos.SoftDelete(ctx, userID, rootTodo.ID); err != nil {
		t.Fatalf("soft delete todo: %v", err)
	}
	if _, err := todos.FindByID(ctx, userID, rootTodo.ID); err == nil {
		t.Fatal("expected deleted todo not found")
	}

	if err := lists.Delete(ctx, list.ID); err != nil {
		t.Fatalf("delete list: %v", err)
	}
}

func TestTodoPushOutboxRepoIntegration(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)
	const userID = "11111111-1111-1111-1111-111111111111"
	const accountID = "22222222-2222-2222-2222-222222222222"

	repo := NewTodoPushOutboxRepo(db)

	// Missing todo should be ignored.
	if err := repo.EnqueueCreate(ctx, userID, "missing", now); err != nil {
		t.Fatalf("enqueue create on missing todo should be nil, got %v", err)
	}

	_, err := db.Exec(ctx,
		`INSERT INTO todo_lists (id, user_id, name, sort_order, created_at, updated_at)
		 VALUES ('list-1',$2,'L1',0,$1,$1)`,
		now, userID,
	)
	if err != nil {
		t.Fatalf("insert list: %v", err)
	}
	_, err = db.Exec(ctx,
		`INSERT INTO todos (id, list_id, user_id, title, priority, is_done, is_pinned, sort_order, created_at, updated_at)
		 VALUES ('todo-1','list-1',$2,'T','normal',false,false,0,$1,$1)`,
		now, userID,
	)
	if err != nil {
		t.Fatalf("insert todo: %v", err)
	}

	// No google account id should be ignored.
	if err := repo.EnqueueUpdate(ctx, userID, "todo-1", now); err != nil {
		t.Fatalf("enqueue update with nil google account should be nil: %v", err)
	}

	_, err = db.Exec(ctx, `UPDATE todo_lists SET google_account_id=$1 WHERE id='list-1'`, accountID)
	if err != nil {
		t.Fatalf("set google account id: %v", err)
	}

	if err := repo.EnqueueCreate(ctx, userID, "todo-1", now); err != nil {
		t.Fatalf("enqueue create: %v", err)
	}
	if err := repo.EnqueueUpdate(ctx, userID, "todo-1", now.Add(time.Second)); err != nil {
		t.Fatalf("enqueue update: %v", err)
	}
	if err := repo.EnqueueDelete(ctx, userID, "todo-1", now.Add(2*time.Second)); err != nil {
		t.Fatalf("enqueue delete: %v", err)
	}

	var count int
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox`).Scan(&count); err != nil {
		t.Fatalf("count outbox rows: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 outbox rows, got %d", count)
	}

	// Dedup on (domain, op, local_resource_id, expected_updated_at)
	if err := repo.EnqueueCreate(ctx, userID, "todo-1", now); err != nil {
		t.Fatalf("enqueue duplicate create: %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT COUNT(*) FROM google_push_outbox`).Scan(&count); err != nil {
		t.Fatalf("count outbox after dedup: %v", err)
	}
	if count != 3 {
		t.Fatalf("dedup should keep row count 3, got %d", count)
	}
}

func TestTodoReposErrorBranchesOnClosedPool(t *testing.T) {
	db := dbtest.Open(t)
	dbtest.Reset(t, db)
	ctx := context.Background()

	lists := NewListRepo(db)
	todos := NewTodoRepo(db)
	outbox := NewTodoPushOutboxRepo(db)
	db.Close()

	if _, err := lists.ListByUser(ctx, "u1"); err == nil {
		t.Fatal("expected list repo error on closed pool")
	}
	if _, err := todos.ListByList(ctx, "u1", "list-1", true); err == nil {
		t.Fatal("expected ListByList error on closed pool")
	}
	if _, err := todos.FindChildrenByParentID(ctx, "u1", "todo-1"); err == nil {
		t.Fatal("expected FindChildrenByParentID error on closed pool")
	}
	if err := todos.UpdateBatch(ctx, []*domain.Todo{{ID: "todo-1", UserID: "u1", SortOrder: 1, UpdatedAt: time.Now()}}); err == nil {
		t.Fatal("expected UpdateBatch error on closed pool")
	}
	if err := outbox.EnqueueCreate(ctx, "u1", "todo-1", time.Now()); err == nil {
		t.Fatal("expected outbox enqueue error on closed pool")
	}
}

func strPtr(s string) *string { return &s }
