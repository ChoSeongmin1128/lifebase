package http

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	portin "lifebase/internal/todo/port/in"
	"lifebase/internal/shared/middleware"
	"lifebase/internal/shared/response"
)

type TodoHandler struct {
	todo portin.TodoUseCase
}

func NewTodoHandler(todo portin.TodoUseCase) *TodoHandler {
	return &TodoHandler{todo: todo}
}

// Lists

func (h *TodoHandler) CreateList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	list, err := h.todo.CreateList(r.Context(), userID, req.Name)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, list)
}

func (h *TodoHandler) ListLists(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	lists, err := h.todo.ListLists(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"lists": lists})
}

func (h *TodoHandler) UpdateList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	listID := chi.URLParam(r, "listID")
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "name is required")
		return
	}

	if err := h.todo.UpdateList(r.Context(), userID, listID, req.Name); err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

func (h *TodoHandler) DeleteList(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	listID := chi.URLParam(r, "listID")

	if err := h.todo.DeleteList(r.Context(), userID, listID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}

// Todos

func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	var input portin.CreateTodoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}
	if input.ListID == "" || input.Title == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "list_id and title are required")
		return
	}

	todo, err := h.todo.CreateTodo(r.Context(), userID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "CREATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, todo)
}

func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID := chi.URLParam(r, "todoID")

	todo, err := h.todo.GetTodo(r.Context(), userID, todoID)
	if err != nil {
		response.Error(w, http.StatusNotFound, "NOT_FOUND", "todo not found")
		return
	}
	response.JSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) ListTodos(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	listID := r.URL.Query().Get("list_id")
	if listID == "" {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "list_id is required")
		return
	}

	includeDone := r.URL.Query().Get("include_done") == "true"

	todos, err := h.todo.ListTodos(r.Context(), userID, listID, includeDone)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "LIST_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"todos": todos})
}

func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID := chi.URLParam(r, "todoID")
	var input portin.UpdateTodoInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		response.Error(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid request body")
		return
	}

	todo, err := h.todo.UpdateTodo(r.Context(), userID, todoID, input)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "UPDATE_FAILED", err.Error())
		return
	}
	response.JSON(w, http.StatusOK, todo)
}

func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	todoID := chi.URLParam(r, "todoID")

	if err := h.todo.DeleteTodo(r.Context(), userID, todoID); err != nil {
		response.Error(w, http.StatusBadRequest, "DELETE_FAILED", err.Error())
		return
	}
	response.NoContent(w)
}
