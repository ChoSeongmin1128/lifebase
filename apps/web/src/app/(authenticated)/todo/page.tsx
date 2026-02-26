"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";

interface TodoList {
  id: string;
  name: string;
  sort_order: number;
}

interface TodoItem {
  id: string;
  list_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
  sort_order: number;
  done_at: string | null;
  created_at: string;
}

const PRIORITY_COLORS: Record<string, string> = {
  urgent: "text-red-500",
  high: "text-orange-500",
  normal: "text-foreground/50",
  low: "text-foreground/30",
};
const PRIORITY_LABELS: Record<string, string> = {
  urgent: "긴급",
  high: "높음",
  normal: "보통",
  low: "낮음",
};

export default function TodoPage() {
  const [lists, setLists] = useState<TodoList[]>([]);
  const [activeListId, setActiveListId] = useState<string>("");
  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [showDone, setShowDone] = useState(false);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [showNewList, setShowNewList] = useState(false);
  const [editingTodo, setEditingTodo] = useState<TodoItem | null>(null);

  const token = getAccessToken();

  const loadLists = useCallback(async () => {
    if (!token) return;
    try {
      const data = await api<{ lists: TodoList[] }>("/todo/lists", { token });
      setLists(data.lists || []);
      if (data.lists?.length > 0 && !activeListId) {
        setActiveListId(data.lists[0].id);
      }
    } catch {
      setLists([]);
    }
  }, [token, activeListId]);

  const loadTodos = useCallback(async () => {
    if (!token || !activeListId) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({
        list_id: activeListId,
        include_done: showDone ? "true" : "false",
      });
      const data = await api<{ todos: TodoItem[] }>(`/todo?${params}`, { token });
      setTodos(data.todos || []);
    } catch {
      setTodos([]);
    } finally {
      setLoading(false);
    }
  }, [token, activeListId, showDone]);

  useEffect(() => {
    loadLists();
  }, [loadLists]);

  useEffect(() => {
    loadTodos();
  }, [loadTodos]);

  const handleCreateList = async () => {
    if (!token || !newListName.trim()) return;
    try {
      const list = await api<TodoList>("/todo/lists", {
        method: "POST",
        body: { name: newListName },
        token,
      });
      setNewListName("");
      setShowNewList(false);
      setLists((prev) => [...prev, list]);
      setActiveListId(list.id);
    } catch (err) {
      console.error("Create list failed:", err);
    }
  };

  const handleCreateTodo = async () => {
    if (!token || !newTodoTitle.trim() || !activeListId) return;
    try {
      await api("/todo", {
        method: "POST",
        body: { list_id: activeListId, title: newTodoTitle },
        token,
      });
      setNewTodoTitle("");
      loadTodos();
    } catch (err) {
      console.error("Create todo failed:", err);
    }
  };

  const handleToggleDone = async (todo: TodoItem) => {
    if (!token) return;
    try {
      await api(`/todo/${todo.id}`, {
        method: "PATCH",
        body: { is_done: !todo.is_done },
        token,
      });
      loadTodos();
    } catch (err) {
      console.error("Toggle failed:", err);
    }
  };

  const handleTogglePin = async (todo: TodoItem) => {
    if (!token) return;
    try {
      await api(`/todo/${todo.id}`, {
        method: "PATCH",
        body: { is_pinned: !todo.is_pinned },
        token,
      });
      loadTodos();
    } catch (err) {
      console.error("Pin toggle failed:", err);
    }
  };

  const handleDeleteTodo = async (todoId: string) => {
    if (!token) return;
    try {
      await api(`/todo/${todoId}`, { method: "DELETE", token });
      setEditingTodo(null);
      loadTodos();
    } catch (err) {
      console.error("Delete failed:", err);
    }
  };

  const handleUpdateTodo = async (todoId: string, updates: Record<string, unknown>) => {
    if (!token) return;
    try {
      await api(`/todo/${todoId}`, { method: "PATCH", body: updates, token });
      setEditingTodo(null);
      loadTodos();
    } catch (err) {
      console.error("Update failed:", err);
    }
  };

  // Separate pinned, active, done
  const pinnedTodos = todos.filter((t) => t.is_pinned && !t.is_done);
  const activeTodos = todos.filter((t) => !t.is_pinned && !t.is_done);
  const doneTodos = todos.filter((t) => t.is_done);

  const todoCount = (listId: string) =>
    todos.filter((t) => t.list_id === listId && !t.is_done).length;

  const isOverdue = (due: string | null) => {
    if (!due) return false;
    return new Date(due) < new Date(new Date().toDateString());
  };

  return (
    <div className="flex h-full flex-col md:flex-row">
      {/* Left: Lists — desktop only */}
      <div className="hidden md:block w-56 shrink-0 border-r border-foreground/10 overflow-auto">
        <div className="p-3">
          <h2 className="mb-2 text-sm font-medium text-foreground/50">목록</h2>
          {lists.map((list) => (
            <button
              key={list.id}
              onClick={() => setActiveListId(list.id)}
              className={`mb-0.5 flex w-full items-center justify-between rounded-md px-3 py-2 text-sm ${
                activeListId === list.id
                  ? "bg-foreground/10 font-medium"
                  : "hover:bg-foreground/5 text-foreground/70"
              }`}
            >
              <span className="truncate">{list.name}</span>
              {activeListId !== list.id && todoCount(list.id) > 0 && (
                <span className="ml-1 rounded-full bg-foreground/10 px-1.5 text-[10px]">
                  {todoCount(list.id)}
                </span>
              )}
            </button>
          ))}
          {showNewList ? (
            <div className="mt-1 flex gap-1">
              <input
                autoFocus
                placeholder="목록 이름"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleCreateList();
                  if (e.key === "Escape") {
                    setShowNewList(false);
                    setNewListName("");
                  }
                }}
                className="flex-1 rounded border border-foreground/10 bg-background px-2 py-1 text-sm outline-none"
              />
            </div>
          ) : (
            <button
              onClick={() => setShowNewList(true)}
              className="mt-1 w-full rounded-md px-3 py-2 text-left text-sm text-foreground/40 hover:bg-foreground/5"
            >
              + 새 목록
            </button>
          )}
        </div>
      </div>

      {/* Mobile: Horizontal chip bar */}
      <div className="flex md:hidden overflow-x-auto gap-2 px-4 py-2 border-b border-foreground/10">
        {lists.map((list) => (
          <button
            key={list.id}
            onClick={() => setActiveListId(list.id)}
            className={`flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1 text-sm transition-colors ${
              activeListId === list.id
                ? "bg-foreground text-background font-medium"
                : "bg-foreground/5 text-foreground/70"
            }`}
          >
            {list.name}
            {todoCount(list.id) > 0 && (
              <span className={`rounded-full px-1.5 text-[10px] ${
                activeListId === list.id
                  ? "bg-background/20"
                  : "bg-foreground/10"
              }`}>
                {todoCount(list.id)}
              </span>
            )}
          </button>
        ))}
        <button
          onClick={() => setShowNewList(true)}
          className="shrink-0 rounded-full bg-foreground/5 px-3 py-1 text-sm text-foreground/40"
        >
          +
        </button>
      </div>

      {/* Right: Todos */}
      <div className="flex flex-1 flex-col">
        {/* Toolbar */}
        <div className="flex items-center justify-between border-b border-foreground/10 px-4 py-2">
          <h2 className="font-medium">
            {lists.find((l) => l.id === activeListId)?.name || "Todo"}
          </h2>
          <label className="flex items-center gap-1.5 text-xs text-foreground/50">
            <input
              type="checkbox"
              checked={showDone}
              onChange={(e) => setShowDone(e.target.checked)}
              className="rounded"
            />
            완료된 항목 표시
          </label>
        </div>

        {/* New todo input */}
        <div className="flex items-center gap-2 border-b border-foreground/10 px-4 py-2">
          <span className="text-foreground/30">+</span>
          <input
            placeholder="새 Todo 추가..."
            value={newTodoTitle}
            onChange={(e) => setNewTodoTitle(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreateTodo()}
            className="flex-1 bg-transparent text-sm outline-none placeholder:text-foreground/30"
          />
        </div>

        {/* Todo list */}
        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center py-20 text-foreground/40">
              불러오는 중...
            </div>
          ) : !activeListId ? (
            <div className="flex flex-col items-center justify-center py-20 text-foreground/40">
              <p>목록을 선택하거나 만들어 주세요</p>
            </div>
          ) : (
            <div>
              {/* Pinned */}
              {pinnedTodos.length > 0 && (
                <>
                  <div className="px-4 pt-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-foreground/40">
                    고정됨
                  </div>
                  {pinnedTodos.map((todo) => (
                    <TodoRow
                      key={todo.id}
                      todo={todo}
                      isOverdue={isOverdue(todo.due)}
                      onToggleDone={() => handleToggleDone(todo)}
                      onTogglePin={() => handleTogglePin(todo)}
                      onEdit={() => setEditingTodo(todo)}
                    />
                  ))}
                  <div className="mx-4 border-b border-foreground/10" />
                </>
              )}

              {/* Active */}
              {activeTodos.map((todo) => (
                <TodoRow
                  key={todo.id}
                  todo={todo}
                  isOverdue={isOverdue(todo.due)}
                  onToggleDone={() => handleToggleDone(todo)}
                  onTogglePin={() => handleTogglePin(todo)}
                  onEdit={() => setEditingTodo(todo)}
                />
              ))}

              {/* Done */}
              {showDone && doneTodos.length > 0 && (
                <>
                  <div className="px-4 pt-4 pb-1 text-[10px] font-medium uppercase tracking-wider text-foreground/30">
                    완료됨 ({doneTodos.length})
                  </div>
                  {doneTodos.map((todo) => (
                    <TodoRow
                      key={todo.id}
                      todo={todo}
                      isOverdue={false}
                      onToggleDone={() => handleToggleDone(todo)}
                      onTogglePin={() => handleTogglePin(todo)}
                      onEdit={() => setEditingTodo(todo)}
                    />
                  ))}
                </>
              )}

              {pinnedTodos.length === 0 && activeTodos.length === 0 && doneTodos.length === 0 && (
                <div className="flex flex-col items-center justify-center py-20 text-foreground/40">
                  <p>Todo가 없습니다</p>
                  <p className="mt-1 text-sm">위 입력란에서 추가해 보세요</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      {editingTodo && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/30"
          onClick={() => setEditingTodo(null)}
        >
          <div
            className="w-[calc(100vw-2rem)] max-w-96 md:w-96 rounded-lg border border-foreground/10 bg-background p-4 shadow-xl"
            onClick={(e) => e.stopPropagation()}
          >
            <h3 className="text-base font-medium">Todo 수정</h3>
            <div className="mt-3 space-y-3">
              <input
                defaultValue={editingTodo.title}
                onBlur={(e) => {
                  if (e.target.value !== editingTodo.title) {
                    handleUpdateTodo(editingTodo.id, { title: e.target.value });
                  }
                }}
                className="w-full rounded border border-foreground/10 bg-background px-3 py-2 text-sm outline-none"
              />
              <textarea
                defaultValue={editingTodo.notes}
                placeholder="메모"
                rows={3}
                onBlur={(e) => {
                  if (e.target.value !== editingTodo.notes) {
                    handleUpdateTodo(editingTodo.id, { notes: e.target.value });
                  }
                }}
                className="w-full rounded border border-foreground/10 bg-background px-3 py-2 text-sm outline-none resize-none"
              />
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="mb-1 block text-xs text-foreground/50">마감일</label>
                  <input
                    type="date"
                    defaultValue={editingTodo.due || ""}
                    onChange={(e) =>
                      handleUpdateTodo(editingTodo.id, {
                        due: e.target.value || null,
                      })
                    }
                    className="w-full rounded border border-foreground/10 bg-background px-2 py-1.5 text-sm outline-none"
                  />
                </div>
                <div className="flex-1">
                  <label className="mb-1 block text-xs text-foreground/50">우선순위</label>
                  <select
                    defaultValue={editingTodo.priority}
                    onChange={(e) =>
                      handleUpdateTodo(editingTodo.id, { priority: e.target.value })
                    }
                    className="w-full rounded border border-foreground/10 bg-background px-2 py-1.5 text-sm outline-none"
                  >
                    <option value="urgent">긴급</option>
                    <option value="high">높음</option>
                    <option value="normal">보통</option>
                    <option value="low">낮음</option>
                  </select>
                </div>
              </div>
            </div>
            <div className="mt-4 flex justify-between">
              <button
                onClick={() => handleDeleteTodo(editingTodo.id)}
                className="rounded px-3 py-1.5 text-sm text-red-500 hover:bg-red-50"
              >
                삭제
              </button>
              <button
                onClick={() => setEditingTodo(null)}
                className="rounded px-3 py-1.5 text-sm hover:bg-foreground/5"
              >
                닫기
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

function TodoRow({
  todo,
  isOverdue,
  onToggleDone,
  onTogglePin,
  onEdit,
}: {
  todo: TodoItem;
  isOverdue: boolean;
  onToggleDone: () => void;
  onTogglePin: () => void;
  onEdit: () => void;
}) {
  return (
    <div
      className={`group flex items-center gap-2 px-4 py-2 hover:bg-foreground/[0.03] ${
        todo.parent_id ? "pl-10" : ""
      } ${todo.is_pinned && !todo.is_done ? "bg-foreground/[0.02]" : ""}`}
    >
      {/* Checkbox */}
      <button
        onClick={onToggleDone}
        className={`flex h-4 w-4 shrink-0 items-center justify-center rounded-full border ${
          todo.is_done
            ? "border-foreground/20 bg-foreground/10"
            : "border-foreground/30 hover:border-foreground/50"
        }`}
      >
        {todo.is_done && (
          <svg width={10} height={10} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        )}
      </button>

      {/* Priority flag */}
      {todo.priority !== "normal" && (
        <span className={`text-xs ${PRIORITY_COLORS[todo.priority]}`}>
          {PRIORITY_LABELS[todo.priority]}
        </span>
      )}

      {/* Content */}
      <div className="min-w-0 flex-1 cursor-pointer" onClick={onEdit}>
        <span
          className={`text-sm ${
            todo.is_done ? "text-foreground/30 line-through" : ""
          }`}
        >
          {todo.title}
        </span>
      </div>

      {/* Due badge */}
      {todo.due && !todo.is_done && (
        <span
          className={`shrink-0 text-[11px] ${
            isOverdue ? "text-red-500 font-medium" : "text-foreground/40"
          }`}
        >
          {new Date(todo.due).toLocaleDateString("ko-KR", { month: "numeric", day: "numeric" })}
        </span>
      )}

      {/* Pin */}
      <button
        onClick={onTogglePin}
        className={`shrink-0 opacity-0 group-hover:opacity-100 transition-opacity ${
          todo.is_pinned ? "!opacity-100 text-foreground" : "text-foreground/30"
        }`}
      >
        <svg width={14} height={14} viewBox="0 0 24 24" fill={todo.is_pinned ? "currentColor" : "none"} stroke="currentColor" strokeWidth="2">
          <path d="M12 17v5" /><path d="M9 10.76a2 2 0 0 1-1.11 1.79l-1.78.9A2 2 0 0 0 5 15.24V16h14v-.76a2 2 0 0 0-1.11-1.79l-1.78-.9A2 2 0 0 1 15 10.76V7a1 1 0 0 1 1-1 2 2 0 0 0 0-4H8a2 2 0 0 0 0 4 1 1 0 0 1 1 1z" />
        </svg>
      </button>
    </div>
  );
}
