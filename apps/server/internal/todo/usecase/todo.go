package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"lifebase/internal/todo/domain"
	portin "lifebase/internal/todo/port/in"
	portout "lifebase/internal/todo/port/out"
)

type todoUseCase struct {
	lists portout.TodoListRepository
	todos portout.TodoRepository
}

func NewTodoUseCase(lists portout.TodoListRepository, todos portout.TodoRepository) portin.TodoUseCase {
	return &todoUseCase{lists: lists, todos: todos}
}

// Lists

func (uc *todoUseCase) CreateList(ctx context.Context, userID, name string) (*domain.TodoList, error) {
	now := time.Now()
	list := &domain.TodoList{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      name,
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := uc.lists.Create(ctx, list); err != nil {
		return nil, fmt.Errorf("create list: %w", err)
	}
	return list, nil
}

func (uc *todoUseCase) ListLists(ctx context.Context, userID string) ([]*domain.TodoList, error) {
	return uc.lists.ListByUser(ctx, userID)
}

func (uc *todoUseCase) UpdateList(ctx context.Context, userID, listID, name string) error {
	list, err := uc.lists.FindByID(ctx, userID, listID)
	if err != nil {
		return fmt.Errorf("list not found")
	}
	list.Name = name
	list.UpdatedAt = time.Now()
	return uc.lists.Update(ctx, list)
}

func (uc *todoUseCase) DeleteList(ctx context.Context, userID, listID string) error {
	_, err := uc.lists.FindByID(ctx, userID, listID)
	if err != nil {
		return fmt.Errorf("list not found")
	}

	// Protect default list (first list by sort_order)
	allLists, err := uc.lists.ListByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("list lookup: %w", err)
	}
	if len(allLists) > 0 && allLists[0].ID == listID {
		return fmt.Errorf("cannot delete default list")
	}

	return uc.lists.Delete(ctx, listID)
}

// Todos

func (uc *todoUseCase) CreateTodo(ctx context.Context, userID string, input portin.CreateTodoInput) (*domain.Todo, error) {
	// Verify list
	_, err := uc.lists.FindByID(ctx, userID, input.ListID)
	if err != nil {
		return nil, fmt.Errorf("list not found")
	}

	// Verify parent exists if specified
	if input.ParentID != nil {
		parent, err := uc.todos.FindByID(ctx, userID, *input.ParentID)
		if err != nil {
			return nil, fmt.Errorf("parent todo not found")
		}
		// Max 2 levels: parent cannot have a parent
		if parent.ParentID != nil {
			return nil, fmt.Errorf("maximum nesting depth is 2 levels")
		}
	}

	priority := input.Priority
	if priority == "" {
		priority = "normal"
	}

	sortOrder, err := uc.todos.NextSortOrder(ctx, userID, input.ListID, input.ParentID)
	if err != nil {
		sortOrder = 0
	}

	now := time.Now()
	todo := &domain.Todo{
		ID:        uuid.New().String(),
		ListID:    input.ListID,
		UserID:    userID,
		ParentID:  input.ParentID,
		Title:     input.Title,
		Notes:     input.Notes,
		Due:       input.Due,
		Priority:  priority,
		IsDone:    false,
		IsPinned:  false,
		SortOrder: sortOrder,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := uc.todos.Create(ctx, todo); err != nil {
		return nil, fmt.Errorf("create todo: %w", err)
	}
	return todo, nil
}

func (uc *todoUseCase) GetTodo(ctx context.Context, userID, todoID string) (*domain.Todo, error) {
	return uc.todos.FindByID(ctx, userID, todoID)
}

func (uc *todoUseCase) ListTodos(ctx context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error) {
	return uc.todos.ListByList(ctx, userID, listID, includeDone)
}

func (uc *todoUseCase) UpdateTodo(ctx context.Context, userID, todoID string, input portin.UpdateTodoInput) (*domain.Todo, error) {
	todo, err := uc.todos.FindByID(ctx, userID, todoID)
	if err != nil {
		return nil, fmt.Errorf("todo not found")
	}

	if input.Title != nil {
		todo.Title = *input.Title
	}
	if input.Notes != nil {
		todo.Notes = *input.Notes
	}
	if input.Due != nil {
		todo.Due = input.Due
	}
	if input.Priority != nil {
		todo.Priority = *input.Priority
	}
	if input.IsDone != nil {
		todo.IsDone = *input.IsDone
		if *input.IsDone {
			now := time.Now()
			todo.DoneAt = &now

			// Cascade: mark all children as done
			children, err := uc.todos.FindChildrenByParentID(ctx, userID, todoID)
			if err == nil {
				for _, child := range children {
					if !child.IsDone {
						child.IsDone = true
						child.DoneAt = &now
						child.UpdatedAt = now
						_ = uc.todos.Update(ctx, child)
					}
				}
			}
		} else {
			todo.DoneAt = nil
		}
	}
	if input.IsPinned != nil {
		if *input.IsPinned {
			count, err := uc.todos.CountPinned(ctx, userID, todo.ListID)
			if err == nil && count >= 5 {
				return nil, fmt.Errorf("maximum 5 pinned todos per list")
			}
		}
		todo.IsPinned = *input.IsPinned
	}
	if input.ParentID != nil {
		if *input.ParentID == "" {
			// Empty string means "move to root"
			todo.ParentID = nil
		} else {
			todo.ParentID = input.ParentID
		}
	}
	if input.ListID != nil {
		// Verify target list exists and belongs to user
		_, err := uc.lists.FindByID(ctx, userID, *input.ListID)
		if err != nil {
			return nil, fmt.Errorf("target list not found")
		}
		todo.ListID = *input.ListID
	}
	if input.SortOrder != nil {
		todo.SortOrder = *input.SortOrder
	}

	todo.UpdatedAt = time.Now()

	if err := uc.todos.Update(ctx, todo); err != nil {
		return nil, fmt.Errorf("update todo: %w", err)
	}
	return todo, nil
}

func (uc *todoUseCase) DeleteTodo(ctx context.Context, userID, todoID string) error {
	// Cascade: soft-delete children first
	_ = uc.todos.SoftDeleteByParentID(ctx, userID, todoID)
	return uc.todos.SoftDelete(ctx, userID, todoID)
}

func (uc *todoUseCase) ReorderTodos(ctx context.Context, userID string, items []portin.ReorderItem) error {
	now := time.Now()
	var todos []*domain.Todo
	for _, item := range items {
		todos = append(todos, &domain.Todo{
			ID:        item.ID,
			UserID:    userID,
			ParentID:  item.ParentID,
			SortOrder: item.SortOrder,
			UpdatedAt: now,
		})
	}
	return uc.todos.UpdateBatch(ctx, todos)
}
