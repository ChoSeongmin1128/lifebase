package out

import (
	"context"

	"lifebase/internal/todo/domain"
)

type TodoListRepository interface {
	Create(ctx context.Context, list *domain.TodoList) error
	FindByID(ctx context.Context, userID, id string) (*domain.TodoList, error)
	ListByUser(ctx context.Context, userID string) ([]*domain.TodoList, error)
	Update(ctx context.Context, list *domain.TodoList) error
	Delete(ctx context.Context, id string) error
}

type TodoRepository interface {
	Create(ctx context.Context, todo *domain.Todo) error
	FindByID(ctx context.Context, userID, id string) (*domain.Todo, error)
	ListByList(ctx context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error)
	Update(ctx context.Context, todo *domain.Todo) error
	SoftDelete(ctx context.Context, userID, id string) error
	CountPinned(ctx context.Context, userID, listID string) (int, error)
}
