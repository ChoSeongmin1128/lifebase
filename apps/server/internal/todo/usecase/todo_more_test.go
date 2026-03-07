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

type todoListRepoStub struct {
	lists     map[string]*domain.TodoList
	createErr error
	findErr   error
	listErr   error
	updateErr error
	deleteErr error
}

func newTodoListRepoStub() *todoListRepoStub {
	return &todoListRepoStub{lists: map[string]*domain.TodoList{}}
}

func (m *todoListRepoStub) Create(_ context.Context, list *domain.TodoList) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.lists[list.ID] = list
	return nil
}

func (m *todoListRepoStub) FindByID(_ context.Context, userID, id string) (*domain.TodoList, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	l, ok := m.lists[id]
	if !ok || l.UserID != userID {
		return nil, errors.New("not found")
	}
	return l, nil
}

func (m *todoListRepoStub) ListByUser(_ context.Context, userID string) ([]*domain.TodoList, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	out := make([]*domain.TodoList, 0, len(m.lists))
	for _, l := range m.lists {
		if l.UserID == userID {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *todoListRepoStub) Update(_ context.Context, list *domain.TodoList) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.lists[list.ID] = list
	return nil
}

func (m *todoListRepoStub) Delete(_ context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.lists, id)
	return nil
}

type todoRepoStub struct {
	todos          map[string]*domain.Todo
	createErr      error
	findErr        error
	updateErr      error
	softDeleteErr  error
	countPinned    int
	countPinnedErr error
	findChildren   []*domain.Todo
	findChildErr   error
	updateBatchErr error
	nextSortErr    error
	listByListErr  error
}

func newTodoRepoStub() *todoRepoStub {
	return &todoRepoStub{todos: map[string]*domain.Todo{}}
}

func (m *todoRepoStub) Create(_ context.Context, todo *domain.Todo) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.todos[todo.ID] = todo
	return nil
}

func (m *todoRepoStub) FindByID(_ context.Context, userID, id string) (*domain.Todo, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	t, ok := m.todos[id]
	if !ok || t.UserID != userID {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (m *todoRepoStub) ListByList(_ context.Context, userID, listID string, includeDone bool) ([]*domain.Todo, error) {
	if m.listByListErr != nil {
		return nil, m.listByListErr
	}
	out := make([]*domain.Todo, 0, len(m.todos))
	for _, t := range m.todos {
		if t.UserID == userID && t.ListID == listID {
			if !includeDone && t.IsDone {
				continue
			}
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *todoRepoStub) Update(_ context.Context, todo *domain.Todo) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if existing, ok := m.todos[todo.ID]; ok {
		if todo.ListID == "" {
			todo.ListID = existing.ListID
		}
		if todo.UserID == "" {
			todo.UserID = existing.UserID
		}
	}
	m.todos[todo.ID] = todo
	return nil
}

func (m *todoRepoStub) SoftDelete(_ context.Context, userID, id string) error {
	if m.softDeleteErr != nil {
		return m.softDeleteErr
	}
	t, ok := m.todos[id]
	if !ok || t.UserID != userID {
		return errors.New("not found")
	}
	delete(m.todos, id)
	return nil
}

func (m *todoRepoStub) CountPinned(_ context.Context, _, _ string) (int, error) {
	if m.countPinnedErr != nil {
		return 0, m.countPinnedErr
	}
	return m.countPinned, nil
}

func (m *todoRepoStub) FindChildrenByParentID(_ context.Context, _ string, parentID string) ([]*domain.Todo, error) {
	if m.findChildErr != nil {
		return nil, m.findChildErr
	}
	if m.findChildren != nil {
		return m.findChildren, nil
	}
	var out []*domain.Todo
	for _, t := range m.todos {
		if t.ParentID != nil && *t.ParentID == parentID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (m *todoRepoStub) SoftDeleteByParentID(_ context.Context, userID, parentID string) error {
	for id, t := range m.todos {
		if t.UserID == userID && t.ParentID != nil && *t.ParentID == parentID {
			delete(m.todos, id)
		}
	}
	return nil
}

func (m *todoRepoStub) UpdateBatch(_ context.Context, todos []*domain.Todo) error {
	if m.updateBatchErr != nil {
		return m.updateBatchErr
	}
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

func (m *todoRepoStub) NextSortOrder(_ context.Context, _, _ string, _ *string) (int, error) {
	if m.nextSortErr != nil {
		return 0, m.nextSortErr
	}
	return len(m.todos) + 1, nil
}

type todoOutboxStub struct {
	createCalls int
	updateCalls int
	deleteCalls int
}

func (m *todoOutboxStub) EnqueueCreate(context.Context, string, string, time.Time) error {
	m.createCalls++
	return nil
}
func (m *todoOutboxStub) EnqueueUpdate(context.Context, string, string, time.Time) error {
	m.updateCalls++
	return nil
}
func (m *todoOutboxStub) EnqueueDelete(context.Context, string, string, time.Time) error {
	m.deleteCalls++
	return nil
}

type googleAccountRepoStub struct {
	findByIDFn func(context.Context, string, string) (*authdomain.GoogleAccount, error)
}

func (m *googleAccountRepoStub) FindByGoogleID(context.Context, string) (*authdomain.GoogleAccount, error) {
	return nil, errors.New("not implemented")
}
func (m *googleAccountRepoStub) FindByID(ctx context.Context, userID, id string) (*authdomain.GoogleAccount, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, userID, id)
	}
	return nil, errors.New("not found")
}
func (m *googleAccountRepoStub) FindByUserID(context.Context, string) ([]*authdomain.GoogleAccount, error) {
	return nil, errors.New("not implemented")
}
func (m *googleAccountRepoStub) Create(context.Context, *authdomain.GoogleAccount) error { return nil }
func (m *googleAccountRepoStub) Update(context.Context, *authdomain.GoogleAccount) error { return nil }

type googleClientStub struct {
	createTaskListID string
	createTaskListErr error
	deleteTaskListErr error
}

func (m *googleClientStub) AuthURL(string) string { return "" }
func (m *googleClientStub) AuthURLForApp(string, string) string { return "" }
func (m *googleClientStub) ExchangeCode(context.Context, string) (*authportout.OAuthToken, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) ExchangeCodeForApp(context.Context, string, string) (*authportout.OAuthToken, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) FetchUserInfo(context.Context, authportout.OAuthToken) (*authportout.OAuthUserInfo, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) ListCalendars(context.Context, authportout.OAuthToken) ([]authportout.OAuthCalendar, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) ListTaskLists(context.Context, authportout.OAuthToken) ([]authportout.OAuthTaskList, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) ListCalendarEvents(context.Context, authportout.OAuthToken, string, string, string, *time.Time, *time.Time) (*authportout.OAuthCalendarEventsPage, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) ListTasks(context.Context, authportout.OAuthToken, string, string) (*authportout.OAuthTasksPage, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) CreateCalendarEvent(context.Context, authportout.OAuthToken, string, authportout.CalendarEventUpsertInput) (string, *string, error) {
	return "", nil, errors.New("not implemented")
}
func (m *googleClientStub) UpdateCalendarEvent(context.Context, authportout.OAuthToken, string, string, authportout.CalendarEventUpsertInput) (*string, error) {
	return nil, errors.New("not implemented")
}
func (m *googleClientStub) DeleteCalendarEvent(context.Context, authportout.OAuthToken, string, string) error {
	return errors.New("not implemented")
}
func (m *googleClientStub) CreateTaskList(context.Context, authportout.OAuthToken, string) (string, error) {
	if m.createTaskListErr != nil {
		return "", m.createTaskListErr
	}
	if m.createTaskListID != "" {
		return m.createTaskListID, nil
	}
	return "google-list-1", nil
}
func (m *googleClientStub) DeleteTaskList(context.Context, authportout.OAuthToken, string) error {
	return m.deleteTaskListErr
}
func (m *googleClientStub) CreateTask(context.Context, authportout.OAuthToken, string, authportout.TodoUpsertInput) (string, error) {
	return "", errors.New("not implemented")
}
func (m *googleClientStub) UpdateTask(context.Context, authportout.OAuthToken, string, string, authportout.TodoUpsertInput) error {
	return errors.New("not implemented")
}
func (m *googleClientStub) DeleteTask(context.Context, authportout.OAuthToken, string, string) error {
	return errors.New("not implemented")
}

func TestCreateListWithTargetGoogleBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("validation and invalid target", func(t *testing.T) {
		uc := NewTodoUseCase(newTodoListRepoStub(), newTodoRepoStub(), nil)
		if _, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: " ", Target: "local"}); err == nil {
			t.Fatal("expected required name error")
		}
		if _, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "x", Target: "invalid"}); err == nil {
			t.Fatal("expected invalid target error")
		}
	})

	t.Run("google missing account id rolls back local list", func(t *testing.T) {
		lists := newTodoListRepoStub()
		uc := NewTodoUseCase(lists, newTodoRepoStub(), nil)
		_, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google"})
		if err == nil || !strings.Contains(err.Error(), "google_account_id is required") {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(lists.lists) != 0 {
			t.Fatalf("expected rollback delete, got %d lists", len(lists.lists))
		}
	})

	t.Run("google integration not configured", func(t *testing.T) {
		lists := newTodoListRepoStub()
		accountID := "acct-1"
		uc := NewTodoUseCase(lists, newTodoRepoStub(), nil)
		_, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID})
		if err == nil || !strings.Contains(err.Error(), "not configured") {
			t.Fatalf("unexpected err: %v", err)
		}
		if len(lists.lists) != 0 {
			t.Fatalf("expected rollback delete, got %d lists", len(lists.lists))
		}
	})

	t.Run("google account status and api errors", func(t *testing.T) {
		accountID := "acct-1"
		lists := newTodoListRepoStub()
		repo := &googleAccountRepoStub{
			findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
				return &authdomain.GoogleAccount{ID: accountID, Status: "revoked"}, nil
			},
		}
		uc := NewTodoUseCase(lists, newTodoRepoStub(), nil, TodoExternalDeps{GoogleAccounts: repo, GoogleClient: &googleClientStub{}})
		_, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID})
		if err == nil || !strings.Contains(err.Error(), "not active") {
			t.Fatalf("unexpected err: %v", err)
		}
	})

	t.Run("google create list failure and link update failure", func(t *testing.T) {
		accountID := "acct-1"
		activeRepo := &googleAccountRepoStub{
			findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
				return &authdomain.GoogleAccount{
					ID:           accountID,
					Status:       "active",
					AccessToken:  "a",
					RefreshToken: "r",
				}, nil
			},
		}

		ucFailCreate := NewTodoUseCase(newTodoListRepoStub(), newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: activeRepo,
			GoogleClient:   &googleClientStub{createTaskListErr: errors.New("upstream")},
		})
		if _, err := ucFailCreate.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID}); err == nil {
			t.Fatal("expected create google task list error")
		}

		lists := newTodoListRepoStub()
		lists.updateErr = errors.New("update failed")
		ucFailUpdate := NewTodoUseCase(lists, newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: activeRepo,
			GoogleClient:   &googleClientStub{createTaskListID: "glist-1"},
		})
		if _, err := ucFailUpdate.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID}); err == nil {
			t.Fatal("expected link google list error")
		}
	})

	t.Run("google success", func(t *testing.T) {
		accountID := "acct-1"
		activeRepo := &googleAccountRepoStub{
			findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
				return &authdomain.GoogleAccount{ID: accountID, Status: "active"}, nil
			},
		}
		uc := NewTodoUseCase(newTodoListRepoStub(), newTodoRepoStub(), nil, TodoExternalDeps{
			GoogleAccounts: activeRepo,
			GoogleClient:   &googleClientStub{createTaskListID: "glist-ok"},
		})
		list, err := uc.CreateListWithTarget(ctx, "u1", portin.CreateListInput{Name: "g", Target: "google", GoogleAccountID: &accountID})
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if list.GoogleID == nil || *list.GoogleID != "glist-ok" {
			t.Fatalf("unexpected google list link: %#v", list)
		}
	})
}

func TestTodoUseCaseListGetUpdateDeleteAndErrors(t *testing.T) {
	ctx := context.Background()
	lists := newTodoListRepoStub()
	todos := newTodoRepoStub()
	outbox := &todoOutboxStub{}
	uc := NewTodoUseCase(lists, todos, outbox)

	list, err := uc.CreateList(ctx, "u1", "inbox")
	if err != nil {
		t.Fatalf("create list: %v", err)
	}
	if err := uc.UpdateList(ctx, "u1", "missing", "x"); err == nil {
		t.Fatal("expected list not found")
	}
	if err := uc.UpdateList(ctx, "u1", list.ID, "renamed"); err != nil {
		t.Fatalf("update list: %v", err)
	}

	todo, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list.ID, Title: "t1"})
	if err != nil {
		t.Fatalf("create todo: %v", err)
	}
	if outbox.createCalls != 1 {
		t.Fatalf("expected outbox create call")
	}

	got, err := uc.GetTodo(ctx, "u1", todo.ID)
	if err != nil || got.ID != todo.ID {
		t.Fatalf("get todo failed: %v", err)
	}
	items, err := uc.ListTodos(ctx, "u1", list.ID, true)
	if err != nil || len(items) != 1 {
		t.Fatalf("list todos failed: %v, len=%d", err, len(items))
	}

	todos.softDeleteErr = errors.New("delete failed")
	if err := uc.DeleteTodo(ctx, "u1", todo.ID); err == nil {
		t.Fatal("expected delete error")
	}
	todos.softDeleteErr = nil
	if err := uc.DeleteTodo(ctx, "u1", todo.ID); err != nil {
		t.Fatalf("delete todo: %v", err)
	}
	if outbox.deleteCalls != 1 {
		t.Fatalf("expected outbox delete call")
	}
}

func TestTodoUseCaseCreateUpdateAndReorderBranches(t *testing.T) {
	ctx := context.Background()
	lists := newTodoListRepoStub()
	todos := newTodoRepoStub()
	outbox := &todoOutboxStub{}
	uc := NewTodoUseCase(lists, todos, outbox)

	list1, _ := uc.CreateList(ctx, "u1", "l1")
	list2, _ := uc.CreateList(ctx, "u1", "l2")

	if _, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: "missing", Title: "x"}); err == nil {
		t.Fatal("expected list not found")
	}

	missingParent := "missing"
	if _, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "x", ParentID: &missingParent}); err == nil {
		t.Fatal("expected parent not found")
	}

	parent, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "parent"})
	if _, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list2.ID, Title: "x", ParentID: &parent.ID}); err == nil {
		t.Fatal("expected parent same list validation")
	}

	todos.nextSortErr = errors.New("next sort fail")
	created, err := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "fallback-order"})
	if err != nil {
		t.Fatalf("create todo with fallback sort order: %v", err)
	}
	if created.SortOrder != 0 {
		t.Fatalf("expected fallback sort order 0, got %d", created.SortOrder)
	}
	todos.nextSortErr = nil

	if _, err := uc.UpdateTodo(ctx, "u1", "missing", portin.UpdateTodoInput{}); err == nil {
		t.Fatal("expected todo not found")
	}

	self := parent.ID
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{ParentID: &self}); err == nil {
		t.Fatal("expected self-parent error")
	}

	unknownParent := "unknown"
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{ParentID: &unknownParent}); err == nil {
		t.Fatal("expected parent not found")
	}

	otherListTodo, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list2.ID, Title: "other"})
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{ParentID: &otherListTodo.ID}); err == nil {
		t.Fatal("expected parent same list error")
	}

	missingList := "none"
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{ListID: &missingList}); err == nil {
		t.Fatal("expected target list not found")
	}

	todos.updateErr = errors.New("update fail")
	title := "new"
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{Title: &title}); err == nil {
		t.Fatal("expected update failure")
	}
	todos.updateErr = nil

	todos.countPinnedErr = errors.New("count failed")
	pinned := true
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{IsPinned: &pinned}); err != nil {
		t.Fatalf("pin should proceed on count error, got %v", err)
	}

	done := true
	todos.findChildErr = errors.New("child query failed")
	if _, err := uc.UpdateTodo(ctx, "u1", parent.ID, portin.UpdateTodoInput{IsDone: &done}); err != nil {
		t.Fatalf("mark done with child lookup error should not fail: %v", err)
	}
	if outbox.updateCalls == 0 {
		t.Fatal("expected outbox update call")
	}

	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: "missing", SortOrder: 1}}); err == nil {
		t.Fatal("expected reorder todo not found")
	}
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: parent.ID, ParentID: &parent.ID, SortOrder: 1}}); err == nil {
		t.Fatal("expected reorder self-parent")
	}
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: parent.ID, ParentID: &unknownParent, SortOrder: 1}}); err == nil {
		t.Fatal("expected reorder parent not found")
	}
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: parent.ID, ParentID: &otherListTodo.ID, SortOrder: 1}}); err == nil {
		t.Fatal("expected reorder parent list mismatch")
	}

	child, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "child", ParentID: &parent.ID})
	another, _ := uc.CreateTodo(ctx, "u1", portin.CreateTodoInput{ListID: list1.ID, Title: "another"})
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: another.ID, ParentID: &child.ID, SortOrder: 1}}); err == nil {
		t.Fatal("expected reorder max depth error")
	}

	todos.updateBatchErr = errors.New("batch fail")
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: another.ID, ParentID: &parent.ID, SortOrder: 2}}); err == nil {
		t.Fatal("expected reorder batch failure")
	}
	todos.updateBatchErr = nil

	rootSort := 10
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: another.ID, SortOrder: rootSort}}); err != nil {
		t.Fatalf("expected reorder root item success: %v", err)
	}

	deepSort := 11
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{{ID: child.ID, ParentID: &parent.ID, SortOrder: deepSort}}); err != nil {
		t.Fatalf("expected reorder with parent outside items success: %v", err)
	}

	parentRootSort := 12
	childSort := 13
	if err := uc.ReorderTodos(ctx, "u1", []portin.ReorderItem{
		{ID: parent.ID, SortOrder: parentRootSort},
		{ID: child.ID, ParentID: &parent.ID, SortOrder: childSort},
	}); err != nil {
		t.Fatalf("expected reorder with parent included in batch success: %v", err)
	}

	if err := uc.ReorderTodos(ctx, "u1", nil); err != nil {
		t.Fatalf("expected empty reorder batch success: %v", err)
	}
}

func TestTodoUseCaseDeleteListGoogleBranchesAndMapError(t *testing.T) {
	ctx := context.Background()
	lists := newTodoListRepoStub()
	todos := newTodoRepoStub()

	googleID := "g-list"
	accountID := "g-account"
	lists.lists["l1"] = &domain.TodoList{ID: "l1", UserID: "u1", Name: "G", GoogleID: &googleID, GoogleAccountID: &accountID}

	ucNoDeps := NewTodoUseCase(lists, todos, nil)
	if err := ucNoDeps.DeleteList(ctx, "u1", "missing"); err == nil {
		t.Fatal("expected list not found")
	}

	lists.lists["l2"] = &domain.TodoList{ID: "l2", UserID: "u1", Name: "G", GoogleID: &googleID}
	if err := ucNoDeps.DeleteList(ctx, "u1", "l2"); err == nil {
		t.Fatal("expected missing account id")
	}
	if err := ucNoDeps.DeleteList(ctx, "u1", "l1"); err == nil {
		t.Fatal("expected integration not configured")
	}

	activeRepo := &googleAccountRepoStub{
		findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
			return &authdomain.GoogleAccount{ID: accountID, Status: "active"}, nil
		},
	}
	notFoundRepo := &googleAccountRepoStub{
		findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
			return nil, errors.New("not found")
		},
	}
	ucNoAccount := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{GoogleAccounts: notFoundRepo, GoogleClient: &googleClientStub{}})
	if err := ucNoAccount.DeleteList(ctx, "u1", "l1"); err == nil {
		t.Fatal("expected google account not found")
	}

	inactiveRepo := &googleAccountRepoStub{
		findByIDFn: func(context.Context, string, string) (*authdomain.GoogleAccount, error) {
			return &authdomain.GoogleAccount{ID: accountID, Status: "revoked"}, nil
		},
	}
	ucInactive := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{GoogleAccounts: inactiveRepo, GoogleClient: &googleClientStub{}})
	if err := ucInactive.DeleteList(ctx, "u1", "l1"); err == nil {
		t.Fatal("expected inactive account error")
	}

	uc404 := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{
		GoogleAccounts: activeRepo,
		GoogleClient: &googleClientStub{
			deleteTaskListErr: &authportout.GoogleAPIError{StatusCode: 404, Message: "not found"},
		},
	})
	if err := uc404.DeleteList(ctx, "u1", "l1"); err != nil {
		t.Fatalf("404 google delete should be ignored, got %v", err)
	}

	lists.lists["l1"] = &domain.TodoList{ID: "l1", UserID: "u1", Name: "G", GoogleID: &googleID, GoogleAccountID: &accountID}
	uc400 := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{
		GoogleAccounts: activeRepo,
		GoogleClient: &googleClientStub{
			deleteTaskListErr: &authportout.GoogleAPIError{StatusCode: 400, Reason: "invalid", Message: "bad"},
		},
	})
	if err := uc400.DeleteList(ctx, "u1", "l1"); err == nil || !strings.Contains(err.Error(), "기본 Tasks 목록") {
		t.Fatalf("expected default tasks message, got %v", err)
	}

	uc403 := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{
		GoogleAccounts: activeRepo,
		GoogleClient: &googleClientStub{
			deleteTaskListErr: &authportout.GoogleAPIError{StatusCode: 403, Message: "denied"},
		},
	})
	if err := uc403.DeleteList(ctx, "u1", "l1"); err == nil || !strings.Contains(err.Error(), "권한") {
		t.Fatalf("expected permission message, got %v", err)
	}

	ucUnknown := NewTodoUseCase(lists, todos, nil, TodoExternalDeps{
		GoogleAccounts: activeRepo,
		GoogleClient: &googleClientStub{
			deleteTaskListErr: errors.New("network"),
		},
	})
	if err := ucUnknown.DeleteList(ctx, "u1", "l1"); err == nil || !strings.Contains(err.Error(), "delete google task list") {
		t.Fatalf("expected wrapped delete error, got %v", err)
	}

	if err := mapDeleteGoogleTaskListError(nil); err != nil {
		t.Fatalf("nil map error should remain nil: %v", err)
	}
}
