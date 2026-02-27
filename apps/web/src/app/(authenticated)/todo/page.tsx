"use client";

import { useState, useEffect, useCallback, useMemo, useRef, Suspense } from "react";
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
import {
  DndContext,
  closestCenter,
  DragOverlay,
  type DragStartEvent,
  type DragMoveEvent,
  type DragOverEvent,
  type DragEndEvent,
  type DragCancelEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { Plus } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  buildTree,
  flattenTree,
  getProjection,
  computeReorderChanges,
  type TodoItem,
  type TodoNode,
  type FlattenedItem,
} from "@/lib/todo/dnd-tree";

interface TodoList {
  id: string;
  name: string;
  sort_order: number;
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

  // DnD state
  const [dragActiveId, setDragActiveId] = useState<string | null>(null);
  const [dragOverId, setDragOverId] = useState<string | null>(null);
  const [offsetLeft, setOffsetLeft] = useState(0);
  const dragSnapshotRef = useRef<TodoItem[]>([]);

  const token = getAccessToken();
  const defaultListId = lists.length > 0 ? lists[0].id : null;

  const loadLists = useCallback(async () => {
    if (!token) return;
    try {
      const data = await api<{ lists: TodoList[] }>("/todo/lists", { token });
      const fetchedLists = data.lists || [];

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

  // Separate pinned/active/done at root level
  const pinnedRoots = tree.filter((t) => t.is_pinned && !t.is_done);
  const activeRoots = tree.filter((t) => !t.is_pinned && !t.is_done);
  const doneRoots = tree.filter((t) => t.is_done);

  const pinnedFlat = flattenTree(pinnedRoots, collapsed, dragActiveId);
  const activeFlat = flattenTree(activeRoots, collapsed, dragActiveId);
  const doneFlat = flattenTree(doneRoots, collapsed, dragActiveId);

  const allFlat = useMemo(
    () => [...pinnedFlat, ...activeFlat, ...doneFlat],
    [pinnedFlat, activeFlat, doneFlat],
  );
  const allFlatIds = useMemo(() => allFlat.map((f) => f.id), [allFlat]);

  // Projection for current drag
  const projection = useMemo(() => {
    if (!dragActiveId || !dragOverId) return null;
    return getProjection(allFlat, dragActiveId, dragOverId, offsetLeft);
  }, [allFlat, dragActiveId, dragOverId, offsetLeft]);

  // DnD event handlers
  const handleDragStart = useCallback((event: DragStartEvent) => {
    const id = String(event.active.id);
    setDragActiveId(id);
    setOffsetLeft(0);
    dragSnapshotRef.current = [...todos];
  }, [todos]);

  const handleDragMove = useCallback((event: DragMoveEvent) => {
    setOffsetLeft(event.delta.x);
  }, []);

  const handleDragOver = useCallback((event: DragOverEvent) => {
    setDragOverId(event.over ? String(event.over.id) : null);
  }, []);

  const handleDragEnd = useCallback(async (event: DragEndEvent) => {
    const { active, over } = event;
    const currentActiveId = String(active.id);
    const currentOverId = over ? String(over.id) : null;

    // Reset drag state
    setDragActiveId(null);
    setDragOverId(null);
    setOffsetLeft(0);

    if (!currentOverId || currentActiveId === currentOverId || !token) return;

    const currentProjection = getProjection(allFlat, currentActiveId, currentOverId, event.delta.x);
    const { changes } = computeReorderChanges(allFlat, currentActiveId, currentOverId, currentProjection);

    if (changes.length === 0) return;

    // Optimistic UI update
    const changeMap = new Map(changes.map((c) => [c.id, c]));
    const updatedTodos = todos.map((t) => {
      const change = changeMap.get(t.id);
      if (change) {
        return { ...t, parent_id: change.parent_id, sort_order: change.sort_order };
      }
      return t;
    });
    setTodos(updatedTodos);

    // Persist to server
    try {
      await api("/todo/reorder", {
        method: "PATCH",
        body: { items: changes },
        token,
      });
    } catch (err) {
      console.error("Reorder failed:", err);
      setTodos(dragSnapshotRef.current);
      loadTodos();
    }
  }, [token, allFlat, todos, loadTodos]);

  const handleDragCancel = useCallback((_event: DragCancelEvent) => {
    setDragActiveId(null);
    setDragOverId(null);
    setOffsetLeft(0);
    if (dragSnapshotRef.current.length > 0) {
      setTodos(dragSnapshotRef.current);
    }
  }, []);

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

  // Find active todo for DragOverlay
  const activeTodo = useMemo(() => {
    if (!dragActiveId) return null;
    return allFlat.find((f) => f.id === dragActiveId)?.todo ?? null;
  }, [dragActiveId, allFlat]);

  const projectedDepth = projection?.depth ?? 0;

  const isDndEnabled = sortBy === "manual";

  const renderTodoRow = (item: FlattenedItem) => {
    const { todo, depth } = item;
    const hasChildren = todo.children.length > 0 || childCountMap.has(todo.id);
    const isCollapsed = collapsed.has(todo.id);
    const childCount = childCountMap.get(todo.id);
    const isDragging = todo.id === dragActiveId;

    // Show drop indicator when this is the over item during drag
    const isDropTarget = isDndEnabled && dragActiveId && dragOverId === todo.id && dragActiveId !== todo.id;

    return (
      <div key={todo.id}>
        {isDropTarget && (
          <div
            className="h-0.5 bg-primary rounded-full mx-4 my-0"
            style={{ marginLeft: `${(projection?.depth ?? depth) * 24 + 16}px` }}
          />
        )}
        <TodoRow
          todo={todo}
          depth={depth}
          isOverdue={isOverdue(todo.due)}
          hasChildren={hasChildren}
          isCollapsed={isCollapsed}
          childCount={childCount}
          showDragHandle={isDndEnabled}
          isDragging={isDragging}
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

  const todoListContent = (
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
  );

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
          ) : isDndEnabled ? (
            <DndContext
              sensors={sensors}
              collisionDetection={closestCenter}
              onDragStart={handleDragStart}
              onDragMove={handleDragMove}
              onDragOver={handleDragOver}
              onDragEnd={handleDragEnd}
              onDragCancel={handleDragCancel}
            >
              <SortableContext items={allFlatIds} strategy={verticalListSortingStrategy}>
                {todoListContent}
              </SortableContext>
              <DragOverlay dropAnimation={null}>
                {activeTodo && (
                  <TodoRow
                    todo={activeTodo}
                    depth={projectedDepth}
                    isOverdue={isOverdue(activeTodo.due)}
                    hasChildren={activeTodo.children.length > 0}
                    isCollapsed={false}
                    showDragHandle
                    isOverlay
                    onToggleCollapse={() => {}}
                    onToggleDone={() => {}}
                    onTogglePin={() => {}}
                    onEdit={() => {}}
                    onDelete={() => {}}
                    onChangePriority={() => {}}
                  />
                )}
              </DragOverlay>
            </DndContext>
          ) : (
            todoListContent
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
