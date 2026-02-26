"use client";

import { useState, useEffect, useCallback, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { api } from "@/lib/api";
import { getAccessToken } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Separator } from "@/components/ui/separator";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Select, SelectTrigger, SelectValue, SelectContent, SelectItem } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { TodoToolbar, type TodoSortBy, type TodoFilterMode } from "@/components/todo/TodoToolbar";
import { TodoRow } from "@/components/todo/TodoRow";
import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";

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

function TodoPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const listFromUrl = searchParams.get("list") || "";

  const [lists, setLists] = useState<TodoList[]>([]);
  const [activeListId, setActiveListIdState] = useState<string>(listFromUrl);

  const setActiveListId = useCallback((id: string) => {
    setActiveListIdState(id);
    if (id) {
      router.replace(`/todo?list=${id}`, { scroll: false });
    }
  }, [router]);

  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [newTodoTitle, setNewTodoTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [showNewList, setShowNewList] = useState(false);
  const [editingTodo, setEditingTodo] = useState<TodoItem | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<TodoSortBy>("due");
  const [filter, setFilter] = useState<TodoFilterMode>("all");

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
        include_done: filter === "done" ? "true" : "false",
      });
      const data = await api<{ todos: TodoItem[] }>(`/todo?${params}`, { token });
      setTodos(data.todos || []);
    } catch {
      setTodos([]);
    } finally {
      setLoading(false);
    }
  }, [token, activeListId, filter]);

  useEffect(() => { loadLists(); }, [loadLists]);
  useEffect(() => { loadTodos(); }, [loadTodos]);

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
      await api(`/todo/${todo.id}`, { method: "PATCH", body: { is_done: !todo.is_done }, token });
      loadTodos();
    } catch (err) {
      console.error("Toggle failed:", err);
    }
  };

  const handleTogglePin = async (todo: TodoItem) => {
    if (!token) return;
    try {
      await api(`/todo/${todo.id}`, { method: "PATCH", body: { is_pinned: !todo.is_pinned }, token });
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

  const isOverdue = (due: string | null) => {
    if (!due) return false;
    return new Date(due) < new Date(new Date().toDateString());
  };

  // Filter and search
  let filteredTodos = todos;
  if (searchQuery) {
    const q = searchQuery.toLowerCase();
    filteredTodos = filteredTodos.filter((t) => t.title.toLowerCase().includes(q));
  }
  if (filter === "has_due") filteredTodos = filteredTodos.filter((t) => t.due);
  if (filter === "has_priority") filteredTodos = filteredTodos.filter((t) => t.priority !== "normal");
  if (filter === "done") filteredTodos = filteredTodos.filter((t) => t.is_done);

  const pinnedTodos = filteredTodos.filter((t) => t.is_pinned && !t.is_done);
  const activeTodos = filteredTodos.filter((t) => !t.is_pinned && !t.is_done);
  const doneTodos = filteredTodos.filter((t) => t.is_done);

  return (
    <div className="flex h-full flex-col md:flex-row">
      {/* Left: Lists — desktop */}
      <div className="hidden md:block w-56 shrink-0 border-r border-border overflow-auto">
        <div className="p-3">
          <h2 className="mb-2 text-xs font-medium text-text-muted uppercase tracking-wider">목록</h2>
          {lists.map((list) => (
            <button
              key={list.id}
              onClick={() => setActiveListId(list.id)}
              className={cn(
                "mb-0.5 flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors",
                activeListId === list.id
                  ? "border-l-2 border-primary bg-surface-accent font-medium text-text-strong"
                  : "text-text-secondary hover:bg-surface-accent/50"
              )}
            >
              <span className="truncate">{list.name}</span>
            </button>
          ))}
          {showNewList ? (
            <div className="mt-1 flex gap-1">
              <Input
                autoFocus
                placeholder="목록 이름"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleCreateList();
                  if (e.key === "Escape") { setShowNewList(false); setNewListName(""); }
                }}
                className="h-8 flex-1"
              />
            </div>
          ) : (
            <button
              onClick={() => setShowNewList(true)}
              className="mt-1 w-full rounded-lg px-3 py-2 text-left text-sm text-text-muted hover:bg-surface-accent/50 transition-colors"
            >
              + 새 목록
            </button>
          )}
        </div>
      </div>

      {/* Mobile: Horizontal chip bar */}
      <div className="flex md:hidden overflow-x-auto gap-2 px-4 py-2 border-b border-border">
        {lists.map((list) => (
          <button
            key={list.id}
            onClick={() => setActiveListId(list.id)}
            className={cn(
              "flex shrink-0 items-center gap-1.5 rounded-full px-3 py-1 text-sm transition-colors",
              activeListId === list.id
                ? "bg-primary text-white font-medium"
                : "bg-surface-accent text-text-secondary"
            )}
          >
            {list.name}
          </button>
        ))}
        <button
          onClick={() => setShowNewList(true)}
          className="shrink-0 rounded-full bg-surface-accent px-3 py-1 text-sm text-text-muted"
        >
          +
        </button>
      </div>

      {/* Right: Todos */}
      <div className="flex flex-1 flex-col min-w-0">
        <TodoToolbar
          listName={lists.find((l) => l.id === activeListId)?.name || "Todo"}
          searchQuery={searchQuery}
          onSearchChange={setSearchQuery}
          sortBy={sortBy}
          onSortChange={setSortBy}
          filter={filter}
          onFilterChange={setFilter}
        />

        {/* New todo input */}
        <div className="flex items-center gap-2 border-b border-border px-4 py-2">
          <Plus size={14} className="text-text-muted" />
          <Input
            placeholder="새 Todo 추가..."
            value={newTodoTitle}
            onChange={(e) => setNewTodoTitle(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreateTodo()}
            className="h-8 border-0 bg-transparent px-0 focus:border-0"
          />
        </div>

        {/* Todo list */}
        <div className="flex-1 overflow-auto">
          {loading ? (
            <div className="flex items-center justify-center py-20 text-text-muted">
              불러오는 중...
            </div>
          ) : !activeListId ? (
            <div className="flex flex-col items-center justify-center py-20 text-text-muted">
              <p>목록을 선택하거나 만들어 주세요</p>
            </div>
          ) : (
            <div>
              {/* Pinned */}
              {pinnedTodos.length > 0 && (
                <>
                  <div className="px-4 pt-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-muted">
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
                      onDelete={() => handleDeleteTodo(todo.id)}
                      onChangePriority={(p) => handleUpdateTodo(todo.id, { priority: p })}
                    />
                  ))}
                  <div className="mx-4"><Separator /></div>
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
                  onDelete={() => handleDeleteTodo(todo.id)}
                  onChangePriority={(p) => handleUpdateTodo(todo.id, { priority: p })}
                />
              ))}

              {/* Done */}
              {doneTodos.length > 0 && (
                <>
                  <div className="px-4 pt-4 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-muted">
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
                      onDelete={() => handleDeleteTodo(todo.id)}
                      onChangePriority={(p) => handleUpdateTodo(todo.id, { priority: p })}
                    />
                  ))}
                </>
              )}

              {pinnedTodos.length === 0 && activeTodos.length === 0 && doneTodos.length === 0 && (
                <div className="flex flex-col items-center justify-center py-20 text-text-muted">
                  <p>Todo가 없습니다</p>
                  <p className="mt-1 text-sm">위 입력란에서 추가해 보세요</p>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Edit Modal */}
      <Dialog open={!!editingTodo} onOpenChange={(v) => !v && setEditingTodo(null)}>
        {editingTodo && (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Todo 수정</DialogTitle>
            </DialogHeader>
            <div className="space-y-3">
              <Input
                defaultValue={editingTodo.title}
                onBlur={(e) => {
                  if (e.target.value !== editingTodo.title) {
                    handleUpdateTodo(editingTodo.id, { title: e.target.value });
                  }
                }}
              />
              <Textarea
                defaultValue={editingTodo.notes}
                placeholder="메모"
                rows={3}
                onBlur={(e) => {
                  if (e.target.value !== editingTodo.notes) {
                    handleUpdateTodo(editingTodo.id, { notes: e.target.value });
                  }
                }}
              />
              <div className="flex gap-2">
                <div className="flex-1">
                  <label className="mb-1 block text-xs text-text-muted">마감일</label>
                  <Input
                    type="date"
                    defaultValue={editingTodo.due || ""}
                    onChange={(e) => handleUpdateTodo(editingTodo.id, { due: e.target.value || null })}
                  />
                </div>
                <div className="flex-1">
                  <label className="mb-1 block text-xs text-text-muted">우선순위</label>
                  <Select
                    defaultValue={editingTodo.priority}
                    onValueChange={(v) => handleUpdateTodo(editingTodo.id, { priority: v })}
                  >
                    <SelectTrigger className="h-9">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="urgent">긴급</SelectItem>
                      <SelectItem value="high">높음</SelectItem>
                      <SelectItem value="normal">보통</SelectItem>
                      <SelectItem value="low">낮음</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
            <DialogFooter className="justify-between">
              <Button variant="danger" size="sm" onClick={() => handleDeleteTodo(editingTodo.id)}>
                삭제
              </Button>
              <Button variant="ghost" size="sm" onClick={() => setEditingTodo(null)}>
                닫기
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  );
}

export default function TodoPage() {
  return (
    <Suspense fallback={<div className="flex items-center justify-center h-full text-text-muted">불러오는 중...</div>}>
      <TodoPageInner />
    </Suspense>
  );
}
