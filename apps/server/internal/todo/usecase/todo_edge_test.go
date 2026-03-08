package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	authdomain "lifebase/internal/auth/domain"
	authportout "lifebase/internal/auth/port/out"
	"lifebase/internal/todo/domain"
	portin "lifebase/internal/todo/port/in"
)

type googleClientSpy struct {
	googleClientStub
	lastCreateToken authportout.OAuthToken
	lastDeleteToken authportout.OAuthToken
}

func todoStringPtr(v string) *string { return &v }

func (m *googleClientSpy) CreateTaskList(ctx context.Context, token authportout.OAuthToken, title string) (string, error) {
	m.lastCreateToken = token
	return m.googleClientStub.CreateTaskList(ctx, token, title)
}

func (m *googleClientSpy) DeleteTaskList(ctx context.Context, token authportout.OAuthToken, id string) error {
	m.lastDeleteToken = token
	return m.googleClientStub.DeleteTaskList(ctx, token, id)
}

func TestTodoUseCaseAdditionalBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("create list default local and create error", func(t *testing.T) {
		lists := newTodoListRepoStub()
		uc := NewTodoUseCase(lists, newTodoRepoStub(), nil)
		list, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: " inbox ", Target: ""})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if list.Name != "inbox" {
			t.Fatalf("expected trimmed list name, got %q", list.Name)
		}

		lists.createErr = errors.New("boom")
		if _, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "x"}); err == nil || !strings.Contains(err.Error(), "create list") {
			t.Fatalf("expected wrapped create error, got %v", err)
		}
	})

	t.Run("create list google account not found and expiry passed", func(t *testing.T) {
		accountID := "acct-1"
		expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
		repoErr := &googleAccountRepoStub{
			findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
				return nil, errors.New("missing")
			},
		}
		ucErr := NewTodoUseCase(newTodoListRepoStub(), newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: repoErr,
			GoogleClient:   &googleClientStub{},
		})
		if _, err := ucErr.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID}); err == nil || !strings.Contains(err.Error(), "google account not found") {
			t.Fatalf("expected account not found error, got %v", err)
		}

		repoOK := &googleAccountRepoStub{
			findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
				return &authdomain.GoogleAccount{
					ID:             accountID,
					Status:         "active",
					AccessToken:    "at",
					RefreshToken:   "rt",
					TokenExpiresAt: &expiry,
				}, nil
			},
		}
		client := &googleClientSpy{googleClientStub: googleClientStub{createTaskListID: "g-list"}}
		ucOK := NewTodoUseCase(newTodoListRepoStub(), newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: repoOK,
			GoogleClient:   client,
		})
		if _, err := ucOK.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID}); err != nil {
			t.Fatalf("unexpected create google list err: %v", err)
		}
		if !client.lastCreateToken.Expiry.Equal(expiry) {
			t.Fatalf("expected token expiry propagated, got %v", client.lastCreateToken.Expiry)
		}
	})

	t.Run("delete list passes expiry token", func(t *testing.T) {
		expiry := time.Now().UTC().Add(time.Hour).Truncate(time.Second)
		lists := newTodoListRepoStub()
		lists.lists["l1"] = &domain.TodoList{
			ID:              "l1",
			UserID:          "u1",
			Name:            "Google",
			GoogleID:        todoStringPtr("gid"),
			GoogleAccountID: todoStringPtr("acct-1"),
		}
		client := &googleClientSpy{}
		uc := NewTodoUseCase(lists, newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: &googleAccountRepoStub{
				findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
					return &authdomain.GoogleAccount{
						ID:             "acct-1",
						Status:         "active",
						AccessToken:    "at",
						RefreshToken:   "rt",
						TokenExpiresAt: &expiry,
					}, nil
				},
			},
			GoogleClient: client,
		})
		if err := uc.DeleteList(ctx, "u1", "l1"); err != nil {
			t.Fatalf("unexpected delete list err: %v", err)
		}
		if !client.lastDeleteToken.Expiry.Equal(expiry) {
			t.Fatalf("expected delete token expiry propagated, got %v", client.lastDeleteToken.Expiry)
		}

		if err := mapDeleteGoogleTaskListError(&authportout.GoogleAPIError{StatusCode: 500, Message: "boom"}); err == nil || !strings.Contains(err.Error(), "delete google task list") {
			t.Fatalf("expected wrapped google api error, got %v", err)
		}
	})

	t.Run("create todo and update todo edge branches", func(t *testing.T) {
		lists := newTodoListRepoStub()
		todos := newTodoRepoStub()
		outbox := &todoOutboxStub{}
		uc := NewTodoUseCase(lists, todos, outbox)
		list1, _ := uc.CreateList(ctx, "u1", "l1")
		list2, _ := uc.CreateList(ctx, "u1", "l2")

		todos.createErr = errors.New("create fail")
		if _, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "x"}); err == nil || !strings.Contains(err.Error(), "create todo") {
			t.Fatalf("expected wrapped create todo error, got %v", err)
		}
		todos.createErr = nil

		parent, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "parent"})
		child, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "child", ParentID: &parent.ID})
		target, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "target"})

		title := "renamed"
		notes := "notes"
		dueDate := "2026-03-09"
		dueTime := "14:30"
		priority := "high"
		done := true
		pinned := false
		sortOrder := 7
		if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{
			Title:     &title,
			Notes:     &notes,
			DueDate:   &dueDate,
			DueTime:   &dueTime,
			Priority:  &priority,
			IsDone:    &done,
			IsPinned:  &pinned,
			SortOrder: &sortOrder,
		}); err != nil {
			t.Fatalf("unexpected update err: %v", err)
		}
		if !todos.todos[child.ID].IsDone || todos.todos[child.ID].DoneAt == nil {
			t.Fatalf("expected child done cascade, got %#v", todos.todos[child.ID])
		}

		badList := "missing"
		if _, err := uc.UpdateTodo(ctx, "u1", target.ID, portin.UpdateTodoInput{ListID: &badList}); err == nil || !strings.Contains(err.Error(), "target list not found") {
			t.Fatalf("expected target list error, got %v", err)
		}

		done = false
		clearParent := ""
		nextList := list2.ID
		if _, err := uc.UpdateTodo(ctx, "u1", target.ID, portin.UpdateTodoInput{
			ParentID:  &clearParent,
			ListID:    &nextList,
			IsDone:    &done,
			SortOrder: &sortOrder,
		}); err != nil {
			t.Fatalf("unexpected move update err: %v", err)
		}
		if todos.todos[target.ID].DoneAt != nil || todos.todos[target.ID].ListID != list2.ID || todos.todos[target.ID].SortOrder != sortOrder {
			t.Fatalf("unexpected moved todo state: %#v", todos.todos[target.ID])
		}

		todos.updateErr = errors.New("update fail")
		if _, err := uc.UpdateTodo(ctx, "u1", target.ID, portin.UpdateTodoInput{Title: &title}); err == nil || !strings.Contains(err.Error(), "update todo") {
			t.Fatalf("expected wrapped update error, got %v", err)
		}
	})

	t.Run("reorder parent lookup errors", func(t *testing.T) {
		lists := newTodoListRepoStub()
		todos := newTodoRepoStub()
		uc := NewTodoUseCase(lists, todos, nil)
		list, _ := uc.CreateList(ctx, "u1", "l1")
		parent, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list.ID, Title: "parent"})
		child, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list.ID, Title: "child", ParentID: &parent.ID})
		target, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list.ID, Title: "target"})

		missingParent := "missing"
		if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: target.ID, ParentID: &missingParent, SortOrder: 1}}); err == nil || !strings.Contains(err.Error(), "parent todo not found") {
			t.Fatalf("expected missing parent error, got %v", err)
		}

		if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: target.ID, ParentID: &child.ID, SortOrder: 1}}); err == nil || !strings.Contains(err.Error(), "maximum nesting depth") {
			t.Fatalf("expected depth error, got %v", err)
		}
	})
}
