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
import { CreateTodoDialog } from "@/components/todo/CreateTodoDialog";
import { DndContext, closestCenter, type DragEndEvent, PointerSensor, useSensor, useSensors } from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
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

interface TodoNode extends TodoItem {
  children: TodoNode[];
}

function buildTree(todos: TodoItem[]): TodoNode[] {
  const map = new Map<string, TodoNode>();
  const roots: TodoNode[] = [];

  for (const todo of todos) {
    map.set(todo.id, { ...todo, children: [] });
  }

  for (const todo of todos) {
    const node = map.get(todo.id)!;
    if (todo.parent_id && map.has(todo.parent_id)) {
      map.get(todo.parent_id)!.children.push(node);
    } else {
      roots.push(node);
    }
  }

  return roots;
}

function flattenTree(
  nodes: TodoNode[],
  collapsed: Set<string>,
): { todo: TodoNode; depth: number }[] {
  const result: { todo: TodoNode; depth: number }[] = [];

  function walk(items: TodoNode[], depth: number) {
    for (const node of items) {
      result.push({ todo: node, depth });
      if (node.children.length > 0 && !collapsed.has(node.id)) {
        walk(node.children, depth + 1);
      }
    }
  }

  walk(nodes, 0);
  return result;
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
  const [newListName, setNewListName] = useState("");
  const [showNewList, setShowNewList] = useState(false);
  const [editingTodo, setEditingTodo] = useState<TodoItem | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [sortBy, setSortBy] = useState<TodoSortBy>("due");
  const [filter, setFilter] = useState<TodoFilterMode>("all");
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [creating, setCreating] = useState(false);
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createParentId, setCreateParentId] = useState<string | undefined>();

  const token = getAccessToken();

  // First list is the default list (undeletable)
  const defaultListId = lists.length > 0 ? lists[0].id : null;

  const loadLists = useCallback(async () => {
    if (!token) return;
    try {
      const data = await api<{ lists: TodoList[] }>("/todo/lists", { token });
      const fetchedLists = data.lists || [];

      // Auto-create default list if none exist
      if (fetchedLists.length === 0) {
        try {
          const list = await api<TodoList>("/todo/lists", {
            method: "POST",
            body: { name: "할 일" },
            token,
          });
          setLists([list]);
          setActiveListId(list.id);
          return;
        } catch {
          // ignore
        }
      }

      setLists(fetchedLists);
      setActiveListIdState((prev) => {
        if (!prev && fetchedLists.length > 0) {
          const firstId = fetchedLists[0].id;
          router.replace(`/todo?list=${firstId}`, { scroll: false });
          return firstId;
        }
        return prev;
      });
    } catch {
      setLists([]);
    }
  }, [token, setActiveListId, router]);

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

  const handleCreateTodo = async (data: {
    title: string;
    due: string | null;
    priority: string;
    notes: string;
    parentId?: string;
  }) => {
    if (!token || !activeListId || creating) return;
    setCreating(true);
    try {
      await api("/todo", {
        method: "POST",
        body: {
          list_id: activeListId,
          title: data.title,
          notes: data.notes,
          due: data.due,
          priority: data.priority,
          ...(data.parentId ? { parent_id: data.parentId } : {}),
        },
        token,
      });
      if (data.parentId) {
        setCollapsed((prev) => {
          const next = new Set(prev);
          next.delete(data.parentId!);
          return next;
        });
      }
      setShowCreateDialog(false);
      setCreateParentId(undefined);
      loadTodos();
    } catch (err) {
      console.error("Create todo failed:", err);
    } finally {
      setCreating(false);
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

  const handleDeleteList = async (listId: string) => {
    if (!token || listId === defaultListId) return;
    try {
      await api(`/todo/lists/${listId}`, { method: "DELETE", token });
      setLists((prev) => prev.filter((l) => l.id !== listId));
      if (activeListId === listId && lists.length > 1) {
        const remaining = lists.filter((l) => l.id !== listId);
        if (remaining.length > 0) setActiveListId(remaining[0].id);
      }
    } catch (err) {
      console.error("Delete list failed:", err);
    }
  };

  const toggleCollapse = (todoId: string) => {
    setCollapsed((prev) => {
      const next = new Set(prev);
      if (next.has(todoId)) next.delete(todoId);
      else next.add(todoId);
      return next;
    });
  };

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 5 } }),
  );

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

  // Build tree structure
  const tree = buildTree(filteredTodos);

  // Separate pinned/active/done at root level, then flatten with children
  const pinnedRoots = tree.filter((t) => t.is_pinned && !t.is_done);
  const activeRoots = tree.filter((t) => !t.is_pinned && !t.is_done);
  const doneRoots = tree.filter((t) => t.is_done);

  const pinnedFlat = flattenTree(pinnedRoots, collapsed);
  const activeFlat = flattenTree(activeRoots, collapsed);
  const doneFlat = flattenTree(doneRoots, collapsed);

  const allFlatIds = [...pinnedFlat, ...activeFlat, ...doneFlat].map((f) => f.todo.id);

  const handleDragEnd = async (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id || !token) return;

    const allFlat = [...pinnedFlat, ...activeFlat, ...doneFlat];
    const oldIndex = allFlat.findIndex((f) => f.todo.id === active.id);
    const newIndex = allFlat.findIndex((f) => f.todo.id === over.id);
    if (oldIndex === -1 || newIndex === -1) return;

    const activeItem = allFlat[oldIndex];
    const overItem = allFlat[newIndex];

    // Determine target parent based on drop position's depth
    // - dropping onto a depth-0 item → becomes root (parent_id = null)
    // - dropping onto a depth-1 item → same parent as that item
    let newParentId: string | null = null;
    if (overItem.depth === 1) {
      newParentId = overItem.todo.parent_id;
    } else if (overItem.depth === 0) {
      // If active is a child and dropping right after a root that has children,
      // check the item below: if it's depth-1 child of overItem, stay as child
      if (newIndex < allFlat.length - 1) {
        const nextItem = allFlat[newIndex + 1];
        if (nextItem.depth === 1 && nextItem.todo.parent_id === overItem.todo.id && activeItem.depth === 1) {
          newParentId = overItem.todo.id;
        }
      }
    }

    // Don't allow nesting beyond 2 levels (child can't become parent's parent)
    // A root item dropped onto a child position: only if it has no children itself
    if (newParentId && activeItem.todo.children && activeItem.todo.children.length > 0) {
      newParentId = null; // Can't nest a parent under another parent
    }

    const parentChanged = (activeItem.todo.parent_id ?? null) !== newParentId;

    // Reorder: compute new sort_order among siblings
    const sorted = allFlat.map((f) => f.todo);
    const [moved] = sorted.splice(oldIndex, 1);
    sorted.splice(newIndex, 0, moved);

    // Optimistic UI update
    const updatedTodos = todos.map((t) => {
      const idx = sorted.findIndex((s) => s.id === t.id);
      if (t.id === String(active.id)) {
        return {
          ...t,
          sort_order: idx !== -1 ? idx : t.sort_order,
          parent_id: newParentId,
        };
      }
      return idx !== -1 ? { ...t, sort_order: idx } : t;
    });
    setTodos(updatedTodos);

    // Persist to server
    try {
      const body: Record<string, unknown> = { sort_order: newIndex };
      if (parentChanged) {
        // Send "" for root (Go *string nil = no change, "" = clear parent)
        body.parent_id = newParentId ?? "";
      }
      await api(`/todo/${active.id}`, {
        method: "PATCH",
        body,
        token,
      });
      // Reload to get consistent tree state
      if (parentChanged) loadTodos();
    } catch (err) {
      console.error("Reorder failed:", err);
      loadTodos();
    }
  };

  // Count children for collapsed parents
  const childCountMap = new Map<string, { total: number; done: number }>();
  for (const todo of todos) {
    if (todo.parent_id) {
      const existing = childCountMap.get(todo.parent_id) || { total: 0, done: 0 };
      existing.total++;
      if (todo.is_done) existing.done++;
      childCountMap.set(todo.parent_id, existing);
    }
  }

  const renderTodoRow = ({ todo, depth }: { todo: TodoNode; depth: number }) => {
    const hasChildren = todo.children.length > 0 || childCountMap.has(todo.id);
    const isCollapsed = collapsed.has(todo.id);
    const childCount = childCountMap.get(todo.id);

    return (
      <div key={todo.id}>
        <TodoRow
          todo={todo}
          depth={depth}
          isOverdue={isOverdue(todo.due)}
          hasChildren={hasChildren}
          isCollapsed={isCollapsed}
          childCount={childCount}
          showDragHandle={sortBy === "manual"}
          lists={lists}
          onToggleCollapse={() => toggleCollapse(todo.id)}
          onToggleDone={() => handleToggleDone(todo)}
          onTogglePin={() => handleTogglePin(todo)}
          onEdit={() => setEditingTodo(todo)}
          onDelete={() => handleDeleteTodo(todo.id)}
          onChangePriority={(p) => handleUpdateTodo(todo.id, { priority: p })}
          onAddSubtask={depth < 1 ? () => { setCreateParentId(todo.id); setShowCreateDialog(true); } : undefined}
          onMoveToList={(listId) => handleUpdateTodo(todo.id, { list_id: listId })}
        />
      </div>
    );
  };

  return (
    <div className="flex h-full flex-col md:flex-row">
      {/* Left: Lists — desktop */}
      <div className="hidden md:block w-56 shrink-0 border-r border-border overflow-auto">
        <div className="p-3">
          <h2 className="mb-2 text-xs font-medium text-text-muted uppercase tracking-wider">목록</h2>
          {lists.map((list, index) => (
            <div key={list.id} className="group/list relative">
              <button
                onClick={() => setActiveListId(list.id)}
                className={cn(
                  "mb-0.5 flex w-full items-center justify-between rounded-lg px-3 py-2 text-sm transition-colors",
                  activeListId === list.id
                    ? "border-l-2 border-primary bg-surface-accent font-medium text-text-strong"
                    : "text-text-secondary hover:bg-surface-accent/50"
                )}
              >
                <span className="truncate">{list.name}</span>
                {/* Delete button (not for default list) */}
                {index !== 0 && (
                  <span
                    onClick={(e) => {
                      e.stopPropagation();
                      if (confirm(`"${list.name}" 목록을 삭제하시겠습니까?`)) {
                        handleDeleteList(list.id);
                      }
                    }}
                    className="ml-1 hidden text-text-muted hover:text-error group-hover/list:inline-block text-xs"
                  >
                    ×
                  </span>
                )}
              </button>
            </div>
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

        {/* Add todo button */}
        <button
          onClick={() => { setCreateParentId(undefined); setShowCreateDialog(true); }}
          className="flex items-center gap-2 border-b border-border px-4 py-2.5 text-sm text-text-muted hover:bg-surface-accent/50 transition-colors w-full"
        >
          <Plus size={14} />
          새 Todo 추가...
        </button>

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
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={handleDragEnd}>
              <SortableContext items={allFlatIds} strategy={verticalListSortingStrategy}>
                <div>
                  {/* Pinned */}
                  {pinnedFlat.length > 0 && (
                    <>
                      <div className="px-4 pt-3 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-muted">
                        고정됨
                      </div>
                      {pinnedFlat.map(renderTodoRow)}
                      <div className="mx-4"><Separator /></div>
                    </>
                  )}

                  {/* Active */}
                  {activeFlat.map(renderTodoRow)}

                  {/* Done */}
                  {doneFlat.length > 0 && (
                    <>
                      <div className="px-4 pt-4 pb-1 text-[10px] font-medium uppercase tracking-wider text-text-muted">
                        완료됨 ({doneFlat.length})
                      </div>
                      {doneFlat.map(renderTodoRow)}
                    </>
                  )}

                  {pinnedFlat.length === 0 && activeFlat.length === 0 && doneFlat.length === 0 && (
                    <div className="flex flex-col items-center justify-center py-20 text-text-muted">
                      <p>Todo가 없습니다</p>
                      <p className="mt-1 text-sm">위 버튼으로 추가해 보세요</p>
                    </div>
                  )}
                </div>
              </SortableContext>
            </DndContext>
          )}
        </div>
      </div>

      {/* Create Modal */}
      <CreateTodoDialog
        open={showCreateDialog}
        onOpenChange={(v) => {
          setShowCreateDialog(v);
          if (!v) setCreateParentId(undefined);
        }}
        onSubmit={handleCreateTodo}
        parentId={createParentId}
        disabled={creating}
      />

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
