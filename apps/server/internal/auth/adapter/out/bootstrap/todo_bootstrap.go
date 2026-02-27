package bootstrap

import (
	"context"
	"time"

	"github.com/google/uuid"

	tododomain "lifebase/internal/todo/domain"
	todoportout "lifebase/internal/todo/port/out"
)

type TodoBootstrapper struct {
	lists todoportout.TodoListRepository
}

func NewTodoBootstrapper(lists todoportout.TodoListRepository) *TodoBootstrapper {
	return &TodoBootstrapper{lists: lists}
}

func (b *TodoBootstrapper) BootstrapUser(ctx context.Context, userID string, now time.Time) error {
	defaultList := &tododomain.TodoList{
		ID:        uuid.New().String(),
		UserID:    userID,
		Name:      "할 일",
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return b.lists.Create(ctx, defaultList)
}
