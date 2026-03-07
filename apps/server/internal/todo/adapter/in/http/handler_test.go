package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"lifebase/internal/shared/middleware"
	"lifebase/internal/todo/domain"
	portin "lifebase/internal/todo/port/in"
)

type mockTodoUC struct {
	createListResult *domain.TodoList
	createListErr    error
	listListsResult  []*domain.TodoList
	listListsErr     error
	updateListErr    error
	deleteListErr    error

	createTodoResult *domain.Todo
	createTodoErr    error
	getTodoResult    *domain.Todo
	getTodoErr       error
	listTodosResult  []*domain.Todo
	listTodosErr     error
	updateTodoResult *domain.Todo
	updateTodoErr    error
	deleteTodoErr    error
	reorderErr       error
}

func (m *mockTodoUC) CreateList(context.Context, string, string) (*domain.TodoList, error) {
	return nil, nil
}
func (m *mockTodoUC) CreateListWithTarget(context.Context, string, portin.CreateListInput) (*domain.TodoList, error) {
	return m.createListResult, m.createListErr
}
func (m *mockTodoUC) ListLists(context.Context, string) ([]*domain.TodoList, error) {
	return m.listListsResult, m.listListsErr
}
func (m *mockTodoUC) UpdateList(context.Context, string, string, string) error { return m.updateListErr }
func (m *mockTodoUC) DeleteList(context.Context, string, string) error           { return m.deleteListErr }

func (m *mockTodoUC) CreateTodo(context.Context, string, portin.CreateTodoInput) (*domain.Todo, error) {
	return m.createTodoResult, m.createTodoErr
}
func (m *mockTodoUC) GetTodo(context.Context, string, string) (*domain.Todo, error) {
	return m.getTodoResult, m.getTodoErr
}
func (m *mockTodoUC) ListTodos(context.Context, string, string, bool) ([]*domain.Todo, error) {
	return m.listTodosResult, m.listTodosErr
}
func (m *mockTodoUC) UpdateTodo(context.Context, string, string, portin.UpdateTodoInput) (*domain.Todo, error) {
	return m.updateTodoResult, m.updateTodoErr
}
func (m *mockTodoUC) DeleteTodo(context.Context, string, string) error                 { return m.deleteTodoErr }
func (m *mockTodoUC) ReorderTodos(context.Context, string, []portin.ReorderItem) error { return m.reorderErr }

func reqWithTodoUser(method, target, body string) *http.Request {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := context.WithValue(req.Context(), middleware.UserIDKey, "user-1")
	return req.WithContext(ctx)
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add(key, value)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestTodoHandlerCreateList(t *testing.T) {
	now := time.Now()
	uc := &mockTodoUC{createListResult: &domain.TodoList{ID: "l1", Name: "Inbox", CreatedAt: now, UpdatedAt: now}}
	h := NewTodoHandler(uc)

	rec := httptest.NewRecorder()
	h.CreateList(rec, reqWithTodoUser(http.MethodPost, "/todo/lists", `{"name":"Inbox","target":"local"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateList(rec, reqWithTodoUser(http.MethodPost, "/todo/lists", `{}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.createListErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.CreateList(rec, reqWithTodoUser(http.MethodPost, "/todo/lists", `{"name":"Inbox"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTodoHandlerListLists(t *testing.T) {
	now := time.Now()
	uc := &mockTodoUC{listListsResult: []*domain.TodoList{{ID: "l1", CreatedAt: now, UpdatedAt: now}}}
	h := NewTodoHandler(uc)
	rec := httptest.NewRecorder()
	h.ListLists(rec, reqWithTodoUser(http.MethodGet, "/todo/lists", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.listListsErr = errors.New("db")
	rec = httptest.NewRecorder()
	h.ListLists(rec, reqWithTodoUser(http.MethodGet, "/todo/lists", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}

func TestTodoHandlerUpdateAndDeleteList(t *testing.T) {
	uc := &mockTodoUC{}
	h := NewTodoHandler(uc)

	rec := httptest.NewRecorder()
	req := withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/lists/l1", `{"name":"Renamed"}`), "listID", "l1")
	h.UpdateList(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/lists/l1", `{}`), "listID", "l1")
	h.UpdateList(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.updateListErr = errors.New("fail")
	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/lists/l1", `{"name":"x"}`), "listID", "l1")
	h.UpdateList(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodDelete, "/todo/lists/l1", ""), "listID", "l1")
	uc.updateListErr = nil
	h.DeleteList(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	uc.deleteListErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.DeleteList(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTodoHandlerCreateGetListUpdateDeleteTodo(t *testing.T) {
	now := time.Now()
	uc := &mockTodoUC{
		createTodoResult: &domain.Todo{ID: "t1", ListID: "l1", Title: "todo", CreatedAt: now, UpdatedAt: now},
		getTodoResult:    &domain.Todo{ID: "t1", ListID: "l1", Title: "todo", CreatedAt: now, UpdatedAt: now},
		updateTodoResult: &domain.Todo{ID: "t1", ListID: "l1", Title: "updated", CreatedAt: now, UpdatedAt: now},
	}
	h := NewTodoHandler(uc)

	rec := httptest.NewRecorder()
	h.CreateTodo(rec, reqWithTodoUser(http.MethodPost, "/todo", `{"list_id":"l1","title":"todo"}`))
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateTodo(rec, reqWithTodoUser(http.MethodPost, "/todo", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.CreateTodo(rec, reqWithTodoUser(http.MethodPost, "/todo", `{"list_id":"","title":""}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.createTodoErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.CreateTodo(rec, reqWithTodoUser(http.MethodPost, "/todo", `{"list_id":"l1","title":"todo"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := withURLParam(reqWithTodoUser(http.MethodGet, "/todo/t1", ""), "todoID", "t1")
	h.GetTodo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.getTodoErr = errors.New("not found")
	rec = httptest.NewRecorder()
	h.GetTodo(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/t1", `{"title":"new"}`), "todoID", "t1")
	h.UpdateTodo(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/t1", `{`), "todoID", "t1")
	h.UpdateTodo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.updateTodoErr = errors.New("fail")
	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodPatch, "/todo/t1", `{"title":"new"}`), "todoID", "t1")
	h.UpdateTodo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req = withURLParam(reqWithTodoUser(http.MethodDelete, "/todo/t1", ""), "todoID", "t1")
	uc.updateTodoErr = nil
	h.DeleteTodo(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	uc.deleteTodoErr = errors.New("fail")
	rec = httptest.NewRecorder()
	h.DeleteTodo(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestTodoHandlerListTodosAndReorder(t *testing.T) {
	now := time.Now()
	uc := &mockTodoUC{
		listTodosResult: []*domain.Todo{{ID: "t1", ListID: "l1", Title: "todo", CreatedAt: now, UpdatedAt: now}},
	}
	h := NewTodoHandler(uc)

	rec := httptest.NewRecorder()
	h.ListTodos(rec, reqWithTodoUser(http.MethodGet, "/todo", ""))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ListTodos(rec, reqWithTodoUser(http.MethodGet, "/todo?list_id=l1&include_done=true", ""))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	uc.listTodosErr = errors.New("db failed")
	rec = httptest.NewRecorder()
	h.ListTodos(rec, reqWithTodoUser(http.MethodGet, "/todo?list_id=l1", ""))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ReorderTodos(rec, reqWithTodoUser(http.MethodPatch, "/todo/reorder", `{"items":[{"id":"t1","sort_order":0}]}`))
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ReorderTodos(rec, reqWithTodoUser(http.MethodPatch, "/todo/reorder", `{`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ReorderTodos(rec, reqWithTodoUser(http.MethodPatch, "/todo/reorder", `{"items":[]}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}

	uc.reorderErr = errors.New("failed")
	rec = httptest.NewRecorder()
	h.ReorderTodos(rec, reqWithTodoUser(http.MethodPatch, "/todo/reorder", `{"items":[{"id":"t1","sort_order":0}]}`))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
