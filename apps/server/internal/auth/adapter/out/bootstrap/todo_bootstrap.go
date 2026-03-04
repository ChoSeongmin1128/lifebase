package bootstrap

import (
	"context"
	"time"

	todoportout "lifebase/internal/todo/port/out"
)

type TodoBootstrapper struct {
	lists todoportout.TodoListRepository
}

func NewTodoBootstrapper(lists todoportout.TodoListRepository) *TodoBootstrapper {
	return &TodoBootstrapper{lists: lists}
}

func (b *TodoBootstrapper) BootstrapUser(ctx context.Context, userID string, now time.Time) error {
	_ = ctx
	_ = userID
	_ = now
	// 신규 사용자 기본 Todo 목록 자동 생성 비활성화.
	return nil
}
