package in

import (
	"context"

	"lifebase/internal/todo/domain"
)

type CreateTodoInput struct {
	ListID   string  `json:"list_id"`
	ParentID *string `json:"parent_id"`
	Title    string  `json:"title"`
	Notes    string  `json:"notes"`
	Due      *string `json:"due"`
	Priority string  `json:"priority"`
}

type UpdateTodoInput struct {
	Title    *string `json:"title"`
	Notes    *string `json:"notes"`
	Due      *string `json:"due"`
	Priority *string `json:"priority"`
	IsDone   *bool   `json:"is_done"`
	IsPinned *bool   `json:"is_pinned"`
	ParentID *string `json:"parent_id"`
}

type TodoUseCase interface {
	// Lists
	CreateList(ctx context.Context, userID, name string) (*domain.TodoList, error)
	ListLists(ctx context.Context, userID string) ([]*domain.TodoList, error)
	UpdateList(ctx context.Context, userID, listID, name string) error
	DeleteList(ctx context.Context, userID, listID string) error

	// Todos
	CreateTodo(ctx context.Context, userID string, input CreateTodoInput) (*domain.Todo, error)
	GetTodo(ctx context.Context, userID, todoID string) (*domain.Todo, error)
	ListTodos(ctx context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error)
	UpdateTodo(ctx context.Context, userID, todoID string, input UpdateTodoInput) (*domain.Todo, error)
	DeleteTodo(ctx context.Context, userID, todoID string) error
}
