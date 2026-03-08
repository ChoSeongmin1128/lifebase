package usecase

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"lifebase/internal/todo/domain"
	portin "lifebase/internal/todo/port/in"
)

// Mock repositories

type mockListRepo struct {
	lists map[string]*domain.TodoList
}

func newMockListRepo() *mockListRepo {
	return &mockListRepo{lists: make(map[string]*domain.TodoList)}
}

func (m *mockListRepo) Create(_ context.Context, list *domain.TodoList) error {
	m.lists[list.ID] = list
	return nil
}

func (m *mockListRepo) FindByID(_ context.Context, userID, id string) (*domain.TodoList, error) {
	l, ok := m.lists[id]
	if !ok || l.UserID != userID {
		return nil, fmt.Errorf("not found")
	}
	return l, nil
}

func (m *mockListRepo) ListByUser(_ context.Context, userID string) ([]*domain.TodoList, error) {
	var result []*domain.TodoList
	for _, l := range m.lists {
		if l.UserID == userID {
			result = append(result, l)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder == result[j].SortOrder {
			return result[i].Name < result[j].Name
		}
		return result[i].SortOrder < result[j].SortOrder
	})
	return result, nil
}

func (m *mockListRepo) Update(_ context.Context, list *domain.TodoList) error {
	m.lists[list.ID] = list
	return nil
}

func (m *mockListRepo) Delete(_ context.Context, id string) error {
	delete(m.lists, id)
	return nil
}

type mockTodoRepo struct {
	todos map[string]*domain.Todo
}

func newMockTodoRepo() *mockTodoRepo {
	return &mockTodoRepo{todos: make(map[string]*domain.Todo)}
}

func (m *mockTodoRepo) Create(_ context.Context, todo *domain.Todo) error {
	m.todos[todo.ID] = todo
	return nil
}

func (m *mockTodoRepo) FindByID(_ context.Context, userID, id string) (*domain.Todo, error) {
	t, ok := m.todos[id]
	if !ok || t.UserID != userID {
		return nil, fmt.Errorf("not found")
	}
	return t, nil
}

func (m *mockTodoRepo) ListByList(_ context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error) {
	var result []*domain.Todo
	for _, t := range m.todos {
		if t.UserID == userID && t.ListID == listID && t.DeletedAt == nil {
			if !includeDone && t.IsDone {
				continue
			}
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTodoRepo) Update(_ context.Context, todo *domain.Todo) error {
	m.todos[todo.ID] = todo
	return nil
}

func (m *mockTodoRepo) SoftDelete(_ context.Context, userID, id string) error {
	t, ok := m.todos[id]
	if !ok || t.UserID != userID {
		return fmt.Errorf("not found")
	}
	delete(m.todos, id)
	return nil
}

func (m *mockTodoRepo) CountPinned(_ context.Context, userID, listID string) (int, error) {
	count := 0
	for _, t := range m.todos {
		if t.UserID == userID && t.ListID == listID && t.IsPinned && !t.IsDone {
			count++
		}
	}
	return count, nil
}

func (m *mockTodoRepo) FindChildrenByParentID(_ context.Context, userID, parentID string) ([]*domain.Todo, error) {
	var result []*domain.Todo
	for _, t := range m.todos {
		if t.UserID == userID && t.ParentID != nil && *t.ParentID == parentID {
			result = append(result, t)
		}
	}
	return result, nil
}

func (m *mockTodoRepo) SoftDeleteByParentID(_ context.Context, userID, parentID string) error {
	for id, t := range m.todos {
		if t.UserID == userID && t.ParentID != nil && *t.ParentID == parentID {
			delete(m.todos, id)
		}
	}
	return nil
}

func (m *mockTodoRepo) UpdateBatch(_ context.Context, todos []*domain.Todo) error {
	for _, t := range todos {
		existing, ok := m.todos[t.ID]
		if !ok {
			continue
		}
		existing.ParentID = t.ParentID
		existing.SortOrder = t.SortOrder
		existing.UpdatedAt = t.UpdatedAt
	}
	return nil
}

func (m *mockTodoRepo) NextSortOrder(_ context.Context, userID, listID string, parentID *string) (int, error) {
	max := -1
	for _, t := range m.todos {
		if t.UserID != userID || t.ListID != listID {
			continue
		}
		if parentID == nil && t.ParentID != nil {
			continue
		}
		if parentID != nil && (t.ParentID == nil || *t.ParentID != *parentID) {
			continue
		}
		if t.SortOrder > max {
			max = t.SortOrder
		}
	}
	return max + 1, nil
}

// Tests

func TestCreateTodo_CreatesWithoutLegacyPriorityField(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	todo, err := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Test Todo",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if todo.Title != "Test Todo" {
		t.Errorf("expected created todo title, got '%s'", todo.Title)
	}
}

func TestCreateTodo_MaxNestingDepth(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	parent, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Parent",
	})

	child, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID:   list.ID,
		Title:    "Child",
		ParentID: &parent.ID,
	})

	// Grandchild should fail (max 1 child depth)
	_, err := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID:   list.ID,
		Title:    "Grandchild",
		ParentID: &child.ID,
	})
	if err == nil {
		t.Fatal("expected error for nesting deeper than 1 level, got nil")
	}
}

func TestUpdateTodo_MaxNestingDepth(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	parent, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Parent",
	})
	child, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID:   list.ID,
		Title:    "Child",
		ParentID: &parent.ID,
	})
	target, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Target",
	})

	_, err := uc.UpdateTodo(ctx, "user1", target.ID, portin.UpdateTodoInput{
		ParentID: &child.ID,
	})
	if err == nil {
		t.Fatal("expected error for nesting deeper than 1 level, got nil")
	}
}

func TestReorderTodos_MaxNestingDepth(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	parent, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Parent",
	})
	child, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID:   list.ID,
		Title:    "Child",
		ParentID: &parent.ID,
	})
	target, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Target",
	})

	err := uc.ReorderTodos(ctx, "user1", []portin.ReorderItem{
		{ID: target.ID, ParentID: &child.ID, SortOrder: 0},
	})
	if err == nil {
		t.Fatal("expected error for nesting deeper than 1 level, got nil")
	}
}

func TestUpdateTodo_MaxPinned(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	// Create 5 pinned todos
	for i := 0; i < 5; i++ {
		todo, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
			ListID: list.ID,
			Title:  fmt.Sprintf("Pinned %d", i),
		})
		pinned := true
		uc.UpdateTodo(ctx, "user1", todo.ID, portin.UpdateTodoInput{
			IsPinned: &pinned,
		})
	}

	// 6th pin should fail
	extra, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Extra",
	})
	pinned := true
	_, err := uc.UpdateTodo(ctx, "user1", extra.ID, portin.UpdateTodoInput{
		IsPinned: &pinned,
	})
	if err == nil {
		t.Fatal("expected error for 6th pinned todo, got nil")
	}
}

func TestUpdateTodo_MarkDone(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	list, _ := uc.CreateList(ctx, "user1", "My List")

	todo, _ := uc.CreateTodo(ctx, "user1", portin.CreateTodoInput{
		ListID: list.ID,
		Title:  "Test",
	})

	done := true
	updated, err := uc.UpdateTodo(ctx, "user1", todo.ID, portin.UpdateTodoInput{
		IsDone: &done,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updated.IsDone {
		t.Error("expected todo to be done")
	}
	if updated.DoneAt == nil {
		t.Error("expected DoneAt to be set")
	}

	// Unmark
	undone := false
	updated2, _ := uc.UpdateTodo(ctx, "user1", todo.ID, portin.UpdateTodoInput{
		IsDone: &undone,
	})
	if updated2.IsDone {
		t.Error("expected todo to be undone")
	}
	if updated2.DoneAt != nil {
		t.Error("expected DoneAt to be nil")
	}
}

func TestCreateList_And_Delete(t *testing.T) {
	listRepo := newMockListRepo()
	todoRepo := newMockTodoRepo()
	uc := NewTodoUseCase(listRepo, todoRepo, nil)

	ctx := context.Background()
	_, err := uc.CreateList(ctx, "user1", "Default")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	list, err := uc.CreateList(ctx, "user1", "Shopping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if list.Name != "Shopping" {
		t.Errorf("expected name 'Shopping', got '%s'", list.Name)
	}

	lists, _ := uc.ListLists(ctx, "user1")
	if len(lists) != 2 {
		t.Errorf("expected 2 lists, got %d", len(lists))
	}

	err = uc.DeleteList(ctx, "user1", list.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lists, _ = uc.ListLists(ctx, "user1")
	if len(lists) != 1 {
		t.Errorf("expected 1 list after delete, got %d", len(lists))
	}
}
