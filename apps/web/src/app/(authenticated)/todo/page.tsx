"use client";

import { useState, useEffect, useCallback, useMemo, useRef, Suspense } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { TodoToolbar, type TodoSortBy, type TodoFilterMode } from "@/components/todo/TodoToolbar";
import { TodoRow } from "@/components/todo/TodoRow";
import { TodoInlineEditor } from "@/components/todo/TodoInlineEditor";
import { CreateTodoDialog } from "@/components/todo/CreateTodoDialog";
import { useCreateTodo } from "@/features/todo/ui/hooks/useCreateTodo";
import { useTodoActions } from "@/features/todo/ui/hooks/useTodoActions";
import { useSettingsActions } from "@/features/settings/ui/hooks/useSettingsActions";
import { useAuthFlow } from "@/features/auth/ui/hooks/useAuthFlow";
import type { GoogleAccountSummary } from "@/features/auth/domain/AuthSession";
import { useToast } from "@/components/providers/ToastProvider";
import {
  DndContext,
  closestCenter,
  DragOverlay,
  type DragStartEvent,
  type DragMoveEvent,
  type DragOverEvent,
  type DragEndEvent,
  PointerSensor,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import { SortableContext, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { ChevronRight, Plus } from "lucide-react";
import { cn } from "@/lib/utils";
import { getGoogleAccountDisplayName } from "@/lib/google-account-preferences";
import { normalizeDueDate, normalizeDueTime } from "@/features/todo/lib/formatDueDate";
import {
  buildTree,
  flattenTree,
  getProjection,
  computeReorderChanges,
  type TodoItem,
  type FlattenedItem,
} from "@/lib/todo/dnd-tree";

interface TodoList {
  id: string;
  name: string;
  sort_order: number;
  is_virtual?: boolean;
  google_account_id?: string | null;
  google_account_email?: string | null;
  active_count?: number;
  done_count?: number;
  total_count?: number;
  source?: "google" | "local" | string;
}

const PAGE_ACTION_SYNC_COOLDOWN_MS = 15_000;
const ALL_LIST_ID = "__all__";
const TODO_LAST_SYNC_AT_SETTING_KEY = "todo_last_sync_at";
const TODO_DONE_COLLAPSED_SETTING_KEY = "todo_done_section_collapsed";
const TODO_LAST_ACTIVE_LIST_ID_SETTING_KEY = "todo_last_active_list_id";
const TODO_SORT_VALUES: TodoSortBy[] = ["manual", "due", "recent_starred", "title"];

function toErrorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error && err.message.trim()) {
    return err.message;
  }
  return fallback;
}

function isTodoSortBy(value: string | null | undefined): value is TodoSortBy {
  return !!value && TODO_SORT_VALUES.includes(value as TodoSortBy);
}

function resolveTodoSort(value: string | null | undefined): TodoSortBy | null {
  if (value === "date") return "due";
  return isTodoSortBy(value) ? value : null;
}

function compareStrings(a: string, b: string): number {
  return a.localeCompare(b, "ko");
}

function compareDatesDesc(a?: string | null, b?: string | null): number {
  const aTime = a ? new Date(a).getTime() : Number.NEGATIVE_INFINITY;
  const bTime = b ? new Date(b).getTime() : Number.NEGATIVE_INFINITY;
  return bTime - aTime;
}

function compareDue(a: TodoItem, b: TodoItem): number {
  if (!a.due_date && !b.due_date) return compareDatesDesc(a.created_at, b.created_at);
  if (!a.due_date) return 1;
  if (!b.due_date) return -1;

  const dateCmp = a.due_date.localeCompare(b.due_date);
  if (dateCmp !== 0) return dateCmp;

  if (!a.due_time && !b.due_time) return compareDatesDesc(a.created_at, b.created_at);
  if (!a.due_time) return 1;
  if (!b.due_time) return -1;

  const timeCmp = a.due_time.localeCompare(b.due_time);
  if (timeCmp !== 0) return timeCmp;
  return compareDatesDesc(a.created_at, b.created_at);
}

function compareTodos(a: TodoItem, b: TodoItem, sortBy: TodoSortBy): number {
  if (sortBy === "manual") {
    const orderCmp = a.sort_order - b.sort_order;
    if (orderCmp !== 0) return orderCmp;
    return a.created_at.localeCompare(b.created_at);
  }

  if (sortBy === "due") {
    const dueCmp = compareDue(a, b);
    if (dueCmp !== 0) return dueCmp;
    return compareStrings(a.title, b.title);
  }

  if (sortBy === "recent_starred") {
    const aStar = a.starred_at ? new Date(a.starred_at).getTime() : Number.NEGATIVE_INFINITY;
    const bStar = b.starred_at ? new Date(b.starred_at).getTime() : Number.NEGATIVE_INFINITY;
    if (aStar !== bStar) return bStar - aStar;
    const createdCmp = compareDatesDesc(a.created_at, b.created_at);
    if (createdCmp !== 0) return createdCmp;
    return compareStrings(a.title, b.title);
  }

  const titleCmp = compareStrings(a.title, b.title);
  if (titleCmp !== 0) return titleCmp;
  return compareDatesDesc(a.created_at, b.created_at);
}

function sortTodoNodes(items: ReturnType<typeof buildTree>, sortBy: TodoSortBy): ReturnType<typeof buildTree> {
  return [...items]
    .sort((a, b) => compareTodos(a, b, sortBy))
    .map((item) => ({
      ...item,
      children: sortTodoNodes(item.children, sortBy),
    }));
}

function isOverdueTodo(todo: TodoItem): boolean {
  const dueDate = normalizeDueDate(todo.due_date);
  const dueTime = normalizeDueTime(todo.due_time);
  if (!dueDate || todo.is_done) return false;
  if (dueTime) {
    return new Date(`${dueDate}T${dueTime}:00`) < new Date();
  }
  const now = new Date();
  const todayKey = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
  return dueDate < todayKey;
}

function TodoPageInner() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const listFromUrl = searchParams.get("list") || "";
  const sortFromUrl = searchParams.get("sort") || "";
  const quickAction = searchParams.get("quick");

  const [lists, setLists] = useState<TodoList[]>([]);
  const [settings, setSettings] = useState<Record<string, string>>({});
  const [settingsLoaded, setSettingsLoaded] = useState(false);
  const [activeListId, setActiveListIdState] = useState<string>(listFromUrl || ALL_LIST_ID);
  const [sortBy, setSortBy] = useState<TodoSortBy>("due");
  const activeListIdRef = useRef<string>(listFromUrl || ALL_LIST_ID);

  const setActiveListId = useCallback((id: string) => {
    activeListIdRef.current = id;
    setActiveListIdState(id);
  }, []);

  useEffect(() => {
    activeListIdRef.current = activeListId;
  }, [activeListId]);

  useEffect(() => {
    if (!activeListId) return;
    const params = new URLSearchParams();
    if (activeListId !== ALL_LIST_ID) {
      params.set("list", activeListId);
    }
    if (sortBy) {
      params.set("sort", sortBy);
    }
    const next = params.toString();
    const target = next ? `/todo?${next}` : "/todo";
    const current = searchParams.toString() ? `/todo?${searchParams.toString()}` : "/todo";
    if (target !== current) {
      router.replace(target, { scroll: false });
    }
  }, [activeListId, router, searchParams, sortBy]);

  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [hasLoadedTodos, setHasLoadedTodos] = useState(false);
  const [newListName, setNewListName] = useState("");
  const [newListTarget, setNewListTarget] = useState<"local" | "google">("local");
  const [newListGoogleAccountID, setNewListGoogleAccountID] = useState("");
  const [googleAccounts, setGoogleAccounts] = useState<GoogleAccountSummary[]>([]);
  const [showNewList, setShowNewList] = useState(false);
  const [editingTodoId, setEditingTodoId] = useState<string | null>(null);
  const [closingTodoId, setClosingTodoId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState("");
  const [filter, setFilter] = useState<TodoFilterMode>("all");
  const [doneCollapsed, setDoneCollapsed] = useState(true);
  const [collapsed, setCollapsed] = useState<Set<string>>(new Set());
  const [showCreateDialog, setShowCreateDialog] = useState(false);
  const [createParentId, setCreateParentId] = useState<string | undefined>();
  const [listDeleteTarget, setListDeleteTarget] = useState<TodoList | null>(null);
  const [deletingList, setDeletingList] = useState(false);
  const [clearingCompleted, setClearingCompleted] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState("");
  const [syncingNow, setSyncingNow] = useState(false);
  const quickActionHandledRef = useRef(false);
  const syncThrottleRef = useRef<Record<string, number>>({});
  const editingTodoIdRef = useRef<string | null>(null);
  const closeAnimationTimerRef = useRef<number | null>(null);

  // DnD state
  const [dragActiveId, setDragActiveId] = useState<string | null>(null);
  const [dragOverId, setDragOverId] = useState<string | null>(null);
  const [offsetLeft, setOffsetLeft] = useState(0);
  const dragSnapshotRef = useRef<TodoItem[]>([]);

  const { createTodo, creating } = useCreateTodo();
  const { listLists, createList, deleteList, listTodos, updateTodo, deleteTodo, reorder } = useTodoActions();
  const { getSettings, updateSetting } = useSettingsActions();
  const { listGoogleAccounts, triggerGoogleSync } = useAuthFlow();
  const toast = useToast();
  const isAllView = activeListId === ALL_LIST_ID;
  const realLists = useMemo(() => lists.filter((list) => !list.is_virtual), [lists]);
  const realListsRef = useRef<TodoList[]>([]);
  const realListIDsKey = useMemo(() => realLists.map((list) => list.id).join(","), [realLists]);

  useEffect(() => {
    editingTodoIdRef.current = editingTodoId;
  }, [editingTodoId]);

  useEffect(() => {
    return () => {
      if (closeAnimationTimerRef.current !== null) {
        window.clearTimeout(closeAnimationTimerRef.current);
      }
    };
  }, []);

  const scheduleClosingTodo = useCallback((todoId: string | null) => {
    if (!todoId) return;
    setClosingTodoId(todoId);
    if (closeAnimationTimerRef.current !== null) {
      window.clearTimeout(closeAnimationTimerRef.current);
    }
    closeAnimationTimerRef.current = window.setTimeout(() => {
      setClosingTodoId((prev) => (prev === todoId ? null : prev));
      closeAnimationTimerRef.current = null;
    }, 220);
  }, []);

  const closeEditingTodo = useCallback((todoId?: string | null) => {
    const targetId = todoId ?? editingTodoIdRef.current;
    if (targetId) {
      scheduleClosingTodo(targetId);
    }
    setEditingTodoId((prev) => (targetId && prev === targetId ? null : prev));
  }, [scheduleClosingTodo]);

  const toggleEditingTodo = useCallback((todoId: string) => {
    const currentEditingId = editingTodoIdRef.current;
    if (currentEditingId === todoId) {
      closeEditingTodo(todoId);
      return;
    }
    if (currentEditingId) {
      scheduleClosingTodo(currentEditingId);
    }
    setClosingTodoId((prev) => (prev === todoId ? null : prev));
    setEditingTodoId(todoId);
  }, [closeEditingTodo, scheduleClosingTodo]);
  const listNameByID = useMemo(
    () => new Map(realLists.map((list) => [list.id, list.name])),
    [realLists],
  );
  useEffect(() => {
    realListsRef.current = realLists;
  }, [realLists]);

  const getListActiveCount = useCallback((list: TodoList) => {
    if (typeof list.active_count === "number") return list.active_count;
    const total = typeof list.total_count === "number" ? list.total_count : 0;
    const done = typeof list.done_count === "number" ? list.done_count : 0;
    return Math.max(total - done, 0);
  }, []);

  const getListDoneCount = useCallback((list: TodoList) => {
    if (typeof list.done_count === "number") return list.done_count;
    return 0;
  }, []);

  const getListSourceLabel = useCallback((list: TodoList) => {
    if (list.is_virtual) return "통합";
    const displayName = getGoogleAccountDisplayName(
      settings,
      list.google_account_id ?? null,
      list.google_account_email ?? null,
    );
    if (list.source === "google") {
      return `Google · ${displayName}`;
    }
    if (list.source === "local") return "로컬";
    if (list.google_account_id) {
      return `Google · ${displayName}`;
    }
    return "로컬";
  }, [settings]);

  const isTodoAccountEnabled = useCallback((accountID: string | null | undefined) => {
    if (!accountID) return true;
    return settings[`google_account_sync_todo_${accountID}`] !== "false";
  }, [settings]);

  const filterVisibleLists = useCallback((items: TodoList[]): TodoList[] => {
    return items.filter((list) => isTodoAccountEnabled(list.google_account_id ?? null));
  }, [isTodoAccountEnabled]);

  const loadSettings = useCallback(async () => {
    setSettingsLoaded(false);
    try {
      const next = await getSettings();
      setSettings(next || {});
      const preferredListID = (listFromUrl || next?.[TODO_LAST_ACTIVE_LIST_ID_SETTING_KEY] || ALL_LIST_ID).trim();
      const preferredSort =
        resolveTodoSort(sortFromUrl) ??
        resolveTodoSort(next?.todo_default_sort) ??
        "due";
      setActiveListId(preferredListID || ALL_LIST_ID);
      setSortBy(preferredSort);
      setLastSyncedAt(next?.[TODO_LAST_SYNC_AT_SETTING_KEY] || "");
      setDoneCollapsed(next?.[TODO_DONE_COLLAPSED_SETTING_KEY] !== "false");
    } catch {
      setSettings({});
      setLastSyncedAt("");
      setDoneCollapsed(true);
      if (listFromUrl) {
        setActiveListId(listFromUrl);
      }
      setSortBy(resolveTodoSort(sortFromUrl) ?? "due");
    } finally {
      setSettingsLoaded(true);
    }
  }, [getSettings, listFromUrl, setActiveListId, sortFromUrl]);

  const loadGoogleAccounts = useCallback(async () => {
    try {
      const accounts = await listGoogleAccounts();
      setGoogleAccounts((accounts || []).filter((account) => account.status === "active"));
    } catch {
      setGoogleAccounts([]);
    }
  }, [listGoogleAccounts]);

  const loadLists = useCallback(async () => {
    try {
      const fetchedLists = await listLists();

      const visibleLists = filterVisibleLists(fetchedLists).map((list) => ({ ...list, is_virtual: false }));
      const allList: TodoList = {
        id: ALL_LIST_ID,
        is_virtual: true,
        name: "전체",
        sort_order: -1,
        active_count: visibleLists.reduce((sum, list) => sum + (list.active_count || 0), 0),
        done_count: visibleLists.reduce((sum, list) => sum + (list.done_count || 0), 0),
        total_count: visibleLists.reduce((sum, list) => sum + (list.total_count || 0), 0),
        source: "local",
      };
      const nextLists = [allList, ...visibleLists];
      setLists(nextLists);
      setActiveListIdState((prev) => {
        if (!prev) {
          return ALL_LIST_ID;
        }
        if (prev && !nextLists.some((list) => list.id === prev)) {
          return ALL_LIST_ID;
        }
        return prev;
      });
    } catch {
      setLists([]);
      setActiveListIdState(ALL_LIST_ID);
    }
  }, [filterVisibleLists, listLists]);

  const loadTodos = useCallback(async (listID?: string, options?: { silent?: boolean }) => {
    const silent = options?.silent === true;
    const targetListID = listID ?? activeListIdRef.current;
    if (!targetListID) {
      setTodos([]);
      setLoading(false);
      setRefreshing(false);
      return;
    }
    if (!silent) {
      const showInitialLoading = !hasLoadedTodos;
      setLoading(showInitialLoading);
      setRefreshing(!showInitialLoading);
    }
    try {
      // 완료 섹션 렌더링을 위해 완료 항목 포함 조회
      if (targetListID === ALL_LIST_ID) {
        const todoGroups = await Promise.all(
          realListsRef.current.map(async (list) => {
            try {
              return await listTodos(list.id, true);
            } catch {
              return [];
            }
          }),
        );
        setTodos(todoGroups.flat());
      } else {
        const next = await listTodos(targetListID, true);
        setTodos(next || []);
      }
    } catch {
      setTodos([]);
    } finally {
      if (!silent) {
        setLoading(false);
        setRefreshing(false);
      }
      setHasLoadedTodos(true);
    }
  }, [hasLoadedTodos, listTodos]);

  const applyListDoneDelta = useCallback((listID: string, nextDone: boolean) => {
    const doneDelta = nextDone ? 1 : -1;
    const activeDelta = -doneDelta;
    setLists((prev) =>
      prev.map((list) => {
        if (list.id !== listID && list.id !== ALL_LIST_ID) return list;
        const doneCount = typeof list.done_count === "number" ? list.done_count : 0;
        const totalCount =
          typeof list.total_count === "number"
            ? list.total_count
            : doneCount + (typeof list.active_count === "number" ? list.active_count : 0);
        const activeCount =
          typeof list.active_count === "number"
            ? list.active_count
            : Math.max(totalCount - doneCount, 0);
        const nextDoneCount = Math.max(doneCount + doneDelta, 0);
        const nextActiveCount = Math.max(activeCount + activeDelta, 0);
        return {
          ...list,
          done_count: nextDoneCount,
          active_count: nextActiveCount,
          total_count: nextDoneCount + nextActiveCount,
        };
      }),
    );
  }, []);

  const triggerTodoSync = useCallback(
    async (reason: "page_enter" | "page_action" | "tab_heartbeat" | "manual", throttleMs = 0) => {
      const key = `todo:${reason}`;
      const now = Date.now();
      const last = syncThrottleRef.current[key] || 0;
      if (throttleMs > 0 && now - last < throttleMs) {
        return 0;
      }
      syncThrottleRef.current[key] = now;
      const scheduled = await triggerGoogleSync({ area: "todo", reason });
      if (scheduled > 0) {
        const stamp = new Date().toISOString();
        setLastSyncedAt(stamp);
        updateSetting(TODO_LAST_SYNC_AT_SETTING_KEY, stamp).catch(() => undefined);
      }
      return scheduled;
    },
    [triggerGoogleSync, updateSetting]
  );

  const triggerTodoSyncAndRefresh = useCallback(
    async (reason: "page_enter" | "tab_heartbeat", throttleMs = 0) => {
      try {
        const scheduled = await triggerTodoSync(reason, throttleMs);
        if (scheduled > 0) {
          await loadLists();
          await loadTodos(activeListIdRef.current, { silent: true });
        }
      } catch {
        // ignore sync refresh failures
      }
    },
    [loadLists, loadTodos, triggerTodoSync]
  );

  useEffect(() => { loadSettings(); }, [loadSettings]);
  useEffect(() => {
    if (!settingsLoaded) return;
    if (listFromUrl && listFromUrl !== activeListIdRef.current) {
      setActiveListId(listFromUrl);
    }
  }, [listFromUrl, setActiveListId, settingsLoaded]);
  useEffect(() => {
    if (!settingsLoaded) return;
    const nextSort = resolveTodoSort(sortFromUrl);
    if (!nextSort) return;
    setSortBy((prev) => (prev === nextSort ? prev : nextSort));
  }, [settingsLoaded, sortFromUrl]);
  useEffect(() => {
    if (!settingsLoaded) return;
    if (!activeListId) return;
    if (settings[TODO_LAST_ACTIVE_LIST_ID_SETTING_KEY] === activeListId) return;

    setSettings((prev) => ({ ...prev, [TODO_LAST_ACTIVE_LIST_ID_SETTING_KEY]: activeListId }));
    updateSetting(TODO_LAST_ACTIVE_LIST_ID_SETTING_KEY, activeListId).catch((err) => {
      console.error("Persist active todo list failed:", err);
      toast.warning("마지막 목록 저장 실패", "다음 진입 시 마지막 목록 복원이 되지 않을 수 있습니다.");
    });
  }, [activeListId, settings, settingsLoaded, toast, updateSetting]);
  useEffect(() => { loadGoogleAccounts(); }, [loadGoogleAccounts]);
  useEffect(() => {
    if (!settingsLoaded) return;
    loadLists();
  }, [loadLists, settingsLoaded, settings]);
  useEffect(() => {
    if (!settingsLoaded) return;
    loadTodos(activeListId);
  }, [activeListId, loadTodos, settingsLoaded]);
  useEffect(() => {
    if (!settingsLoaded) return;
    if (activeListId !== ALL_LIST_ID) return;
    if (!realListIDsKey) {
      setTodos([]);
      return;
    }
    loadTodos(ALL_LIST_ID, { silent: true });
  }, [activeListId, loadTodos, realListIDsKey, settingsLoaded]);
  useEffect(() => {
    if (!settingsLoaded) return;
    void triggerTodoSyncAndRefresh("page_enter", 5_000);
  }, [settingsLoaded, triggerTodoSyncAndRefresh]);
  useEffect(() => {
    if (!settingsLoaded) return;
    const timer = window.setInterval(() => {
      void triggerTodoSyncAndRefresh("tab_heartbeat", 9 * 60_000);
    }, 10 * 60_000);
    return () => window.clearInterval(timer);
  }, [settingsLoaded, triggerTodoSyncAndRefresh]);
  useEffect(() => {
    if (quickAction !== "create") return;
    if (quickActionHandledRef.current) return;
    if (!activeListId) return;

    if (activeListId === ALL_LIST_ID) {
      if (realLists.length === 0) return;
      setActiveListId(realLists[0].id);
      return;
    }

    quickActionHandledRef.current = true;
    setCreateParentId(undefined);
    setShowCreateDialog(true);

    const params = new URLSearchParams();
    if (activeListId) {
      params.set("list", activeListId);
    }
    params.set("sort", sortBy);
    const next = params.toString();
    router.replace(next ? `/todo?${next}` : "/todo", { scroll: false });
  }, [quickAction, activeListId, realLists, router, setActiveListId, sortBy]);

  useEffect(() => {
    if (!editingTodoId) return;

    const handlePointerDown = (event: PointerEvent) => {
      const target = event.target;
      if (!(target instanceof HTMLElement)) return;
      const eventPath = typeof event.composedPath === "function" ? event.composedPath() : [];
      const matchesInPath = (predicate: (node: HTMLElement) => boolean) =>
        eventPath.some((node) => node instanceof HTMLElement && predicate(node));

      if (document.body.dataset.todoEditOverlayOpen === "true") return;
      if (
        target.closest(`[data-todo-row-id="${editingTodoId}"]`) ||
        matchesInPath((node) => Boolean(node.closest(`[data-todo-row-id="${editingTodoId}"]`)))
      ) {
        return;
      }
      if (
        target.closest("[data-todo-row-id]") ||
        matchesInPath((node) => Boolean(node.closest("[data-todo-row-id]")))
      ) {
        return;
      }
      if (
        target.closest('[data-todo-edit-layer="true"]') ||
        matchesInPath((node) => Boolean(node.closest('[data-todo-edit-layer="true"]')))
      ) {
        return;
      }
      if (
        target.closest("[data-radix-popper-content-wrapper]") ||
        matchesInPath((node) => Boolean(node.closest("[data-radix-popper-content-wrapper]")))
      ) {
        return;
      }
      closeEditingTodo(editingTodoId);
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        closeEditingTodo(editingTodoId);
      }
    };

    document.addEventListener("pointerdown", handlePointerDown);
    document.addEventListener("keydown", handleEscape);

    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      document.removeEventListener("keydown", handleEscape);
    };
  }, [closeEditingTodo, editingTodoId]);

  const handleCreateList = async () => {
    if (!newListName.trim()) return;
    if (newListTarget === "google" && !newListGoogleAccountID) return;
    try {
      const list = await createList({
        name: newListName,
        target: newListTarget,
        google_account_id: newListTarget === "google" ? newListGoogleAccountID : null,
      });
      setNewListName("");
      setNewListTarget("local");
      setNewListGoogleAccountID("");
      setShowNewList(false);
      setActiveListId(list.id);
      await loadLists();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Create list failed:", err);
    }
  };

  const handleCreateTodo = async (data: {
    title: string;
    dueDate: string | null;
    dueTime: string | null;
    priority: string;
    notes: string;
    parentId?: string;
  }) => {
    if (!activeListId || activeListId === ALL_LIST_ID || creating) return;
    try {
      await createTodo({
        listId: activeListId,
        title: data.title,
        notes: data.notes,
        dueDate: data.dueDate,
        dueTime: data.dueTime,
        priority: data.priority as "urgent" | "high" | "normal" | "low",
        parentId: data.parentId,
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
      await loadTodos();
      await loadLists();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Create todo failed:", err);
    }
  };

  const handleToggleDone = async (todo: TodoItem) => {
    const nextDone = !todo.is_done;
    const nextDoneAt = nextDone ? new Date().toISOString() : null;
    setTodos((prev) =>
      prev.map((item) =>
        item.id === todo.id
          ? { ...item, is_done: nextDone, done_at: nextDoneAt }
          : item,
      ),
    );
    applyListDoneDelta(todo.list_id, nextDone);
    try {
      await updateTodo(todo.id, { is_done: nextDone });
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      setTodos((prev) =>
        prev.map((item) =>
          item.id === todo.id
            ? { ...item, is_done: todo.is_done, done_at: todo.done_at }
            : item,
        ),
      );
      applyListDoneDelta(todo.list_id, !nextDone);
      console.error("Toggle failed:", err);
    }
  };

  const handleTogglePin = async (todo: TodoItem) => {
    try {
      await updateTodo(todo.id, { is_pinned: !todo.is_pinned });
      await loadTodos();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Pin toggle failed:", err);
    }
  };

  const handleDeleteTodo = async (todoId: string) => {
    try {
      await deleteTodo(todoId);
      closeEditingTodo(todoId);
      await loadTodos();
      await loadLists();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Delete failed:", err);
    }
  };

  const handleUpdateTodo = async (todoId: string, updates: Record<string, unknown>) => {
    const previousTodo = todos.find((todo) => todo.id === todoId);
    if (previousTodo) {
      patchTodoLocally(todoId, updates);
    }
    try {
      await updateTodo(todoId, updates);
      const needsListRefresh =
        Object.prototype.hasOwnProperty.call(updates, "is_done") ||
        Object.prototype.hasOwnProperty.call(updates, "list_id");
      if (needsListRefresh) {
        await loadTodos(activeListIdRef.current, { silent: true });
        await loadLists();
      }
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      if (previousTodo) {
        setTodos((prev) => prev.map((todo) => (todo.id === todoId ? previousTodo : todo)));
      }
      console.error("Update failed:", err);
    }
  };

  const handleDeleteList = async (listId: string): Promise<boolean> => {
    if (listId === ALL_LIST_ID) return false;
    try {
      await deleteList(listId);
      await loadLists();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
      return true;
    } catch (err) {
      console.error("Delete list failed:", err);
      toast.error("목록 삭제 실패", toErrorMessage(err, "목록을 삭제하지 못했습니다."));
      return false;
    }
  };

  const handleConfirmDeleteList = async () => {
    if (!listDeleteTarget || deletingList) return;
    setDeletingList(true);
    try {
      const ok = await handleDeleteList(listDeleteTarget.id);
      if (ok) {
        setListDeleteTarget(null);
      }
    } finally {
      setDeletingList(false);
    }
  };

  const handleToggleDoneSection = async () => {
    if (filter === "done") return;
    const next = !doneCollapsed;
    setDoneCollapsed(next);
    setSettings((prev) => ({ ...prev, [TODO_DONE_COLLAPSED_SETTING_KEY]: next ? "true" : "false" }));
    try {
      await updateSetting(TODO_DONE_COLLAPSED_SETTING_KEY, next ? "true" : "false");
    } catch (err) {
      console.error("Persist done section state failed:", err);
      toast.warning("완료 섹션 상태 저장 실패", "다음 새로고침 시 상태가 초기화될 수 있습니다.");
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

  // Filter and search
  let filteredTodos = todos;
  if (searchQuery) {
    const q = searchQuery.toLowerCase();
    filteredTodos = filteredTodos.filter(
      (t) => t.title.toLowerCase().includes(q) || t.notes.toLowerCase().includes(q),
    );
  }
  if (filter === "has_due") filteredTodos = filteredTodos.filter((t) => t.due_date);
  if (filter === "has_priority") filteredTodos = filteredTodos.filter((t) => t.priority !== "normal");
  if (filter === "done") filteredTodos = filteredTodos.filter((t) => t.is_done);

  const buildSortedRoots = (items: TodoItem[]) => {
    const roots = buildTree(items);
    if (sortBy === "manual" && isAllView) {
      return roots;
    }
    return sortTodoNodes(roots, sortBy);
  };

  // 루트 기준으로 섹션을 나눠 부모-자식 체인이 완료 상태 분리로 끊어지지 않게 유지한다.
  const sectionRoots = buildSortedRoots(filteredTodos);
  const activeRoots = sectionRoots.filter((root) => !root.is_done);
  const doneRoots = sectionRoots.filter((root) => root.is_done);
  const showCompletedSection = filter === "done" || !doneCollapsed;

  const activeFlat = flattenTree(activeRoots, collapsed, dragActiveId);
  const doneFlat = flattenTree(doneRoots, collapsed, dragActiveId);
  const doneDeleteRootIDs = useMemo(() => {
    const doneIDSet = new Set(todos.filter((todo) => todo.is_done).map((todo) => todo.id));
    return todos
      .filter((todo) => {
        if (!todo.is_done) return false;
        if (!todo.parent_id) return true;
        return !doneIDSet.has(todo.parent_id);
      })
      .map((todo) => todo.id);
  }, [todos]);

  const handleClearCompleted = useCallback(async () => {
    if (clearingCompleted) return;
    if (doneDeleteRootIDs.length === 0) return;

    setClearingCompleted(true);
    try {
      const results = await Promise.allSettled(doneDeleteRootIDs.map((todoID) => deleteTodo(todoID)));
      const failed = results.filter((result) => result.status === "rejected").length;

      await loadTodos(activeListIdRef.current, { silent: true });
      await loadLists();
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);

      if (failed === 0) {
        toast.success("완료 항목 정리 완료");
      } else {
        toast.warning("일부 완료 항목 정리 실패", `${failed}개 항목을 삭제하지 못했습니다.`);
      }
    } catch (err) {
      console.error("Clear completed failed:", err);
      toast.error("완료 항목 정리 실패", toErrorMessage(err, "완료 항목 삭제 중 오류가 발생했습니다."));
    } finally {
      setClearingCompleted(false);
    }
  }, [activeListIdRef, clearingCompleted, deleteTodo, doneDeleteRootIDs, loadLists, loadTodos, toast, triggerTodoSync]);

  const allFlat = useMemo(() => [...activeFlat, ...doneFlat], [activeFlat, doneFlat]);
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

    if (!currentOverId || currentActiveId === currentOverId) return;

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
      await reorder(changes);
      void triggerTodoSync("page_action", PAGE_ACTION_SYNC_COOLDOWN_MS);
    } catch (err) {
      console.error("Reorder failed:", err);
      setTodos(dragSnapshotRef.current);
      loadTodos(undefined, { silent: true });
    }
  }, [allFlat, loadTodos, reorder, todos, triggerTodoSync]);

  const handleManualSync = useCallback(async () => {
    if (syncingNow) return;
    setSyncingNow(true);
    try {
      await triggerTodoSync("manual", 0);
      await loadLists();
      await loadTodos(activeListIdRef.current, { silent: true });
    } catch {
      // noop
    } finally {
      setSyncingNow(false);
    }
  }, [loadLists, loadTodos, syncingNow, triggerTodoSync]);

  const handleDragCancel = useCallback(() => {
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

  const isDndEnabled = sortBy === "manual" && !isAllView;

  const patchTodoLocally = useCallback((todoId: string, updates: Record<string, unknown>) => {
    setTodos((prev) =>
      prev.map((todo) => {
        if (todo.id !== todoId) return todo;
        return {
          ...todo,
          ...updates,
          due_date: Object.prototype.hasOwnProperty.call(updates, "due_date")
            ? ((updates.due_date as string) || null)
            : todo.due_date,
          due_time: Object.prototype.hasOwnProperty.call(updates, "due_time")
            ? ((updates.due_time as string) || null)
            : todo.due_time,
        };
      }),
    );
  }, []);

  const renderTodoRow = (item: FlattenedItem) => {
    const { todo, depth } = item;
    const hasChildren = todo.children.length > 0 || childCountMap.has(todo.id);
    const isCollapsed = collapsed.has(todo.id);
    const childCount = childCountMap.get(todo.id);
    const isDragging = todo.id === dragActiveId;
    const isExpanded = editingTodoId === todo.id;
    const isClosing = closingTodoId === todo.id;
    const shouldKeepExpandedContent = isExpanded || isClosing;

    // Show drop indicator when this is the over item during drag
    const isDropTarget = isDndEnabled && dragActiveId && dragOverId === todo.id && dragActiveId !== todo.id;

    return (
      <div
        key={todo.id}
        className={cn(
          "mx-2 rounded-xl transition-all",
          isExpanded && "bg-surface-accent/70 ring-1 ring-border/70"
        )}
      >
        {isDropTarget && (
          <div
            className="h-0.5 bg-primary rounded-full mx-4 my-0"
            style={{ marginLeft: `${(projection?.depth ?? depth) * 24 + 16}px` }}
          />
        )}
        <TodoRow
          todo={todo}
          listLabel={isAllView ? listNameByID.get(todo.list_id) : undefined}
          depth={depth}
          isOverdue={isOverdueTodo(todo)}
          hasChildren={hasChildren}
          isCollapsed={isCollapsed}
          childCount={childCount}
          showDragHandle={isDndEnabled}
          isDragging={isDragging}
          isExpanded={isExpanded}
          lists={realLists}
          onToggleCollapse={() => toggleCollapse(todo.id)}
          onToggleDone={() => handleToggleDone(todo)}
          onTogglePin={() => handleTogglePin(todo)}
          onEdit={() => toggleEditingTodo(todo.id)}
          onDelete={() => handleDeleteTodo(todo.id)}
          onChangePriority={(p) => handleUpdateTodo(todo.id, { priority: p })}
          onUpdateTitle={(title) => handleUpdateTodo(todo.id, { title })}
          expandedContent={
            shouldKeepExpandedContent ? (
              <TodoInlineEditor
                todo={todo}
                listName={listNameByID.get(todo.list_id)}
                onUpdate={(updates) => handleUpdateTodo(todo.id, updates)}
              />
            ) : undefined
          }
          onAddSubtask={!isAllView && depth < 1 ? () => { setCreateParentId(todo.id); setShowCreateDialog(true); } : undefined}
          onMoveToList={(listId) => handleUpdateTodo(todo.id, { list_id: listId })}
        />
      </div>
    );
  };

  const todoListContent = (
    <div>
      {/* Active */}
      {activeFlat.map(renderTodoRow)}

      {/* Done */}
      {doneFlat.length > 0 && (
        <>
          <div className="mt-3 flex items-center justify-between gap-2 px-4 pb-1">
            <button
              type="button"
              className="flex items-center gap-1 text-[10px] font-medium uppercase tracking-wider text-text-muted hover:text-text-secondary"
              onClick={handleToggleDoneSection}
            >
              <ChevronRight
                size={12}
                className={cn(
                  "transition-transform",
                  showCompletedSection ? "rotate-90" : "rotate-0",
                )}
              />
              <span>완료됨 ({doneFlat.length})</span>
            </button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              className="h-6 px-2 text-[11px] text-text-muted hover:text-text-strong"
              onClick={handleClearCompleted}
              disabled={clearingCompleted || doneDeleteRootIDs.length === 0}
            >
              {clearingCompleted ? "정리 중..." : "완료 항목 모두 지우기"}
            </Button>
          </div>
          {showCompletedSection && doneFlat.map(renderTodoRow)}
        </>
      )}

      {activeFlat.length === 0 && doneFlat.length === 0 && (
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
          {lists.map((list) => (
            <div key={list.id} className="group/list relative">
              <div
                role="button"
                tabIndex={0}
                onClick={() => setActiveListId(list.id)}
                onKeyDown={(e) => {
                  if (e.key === "Enter" || e.key === " ") {
                    e.preventDefault();
                    setActiveListId(list.id);
                  }
                }}
                className={cn(
                  "mb-0.5 flex w-full items-start justify-between rounded-lg px-3 py-2 transition-colors",
                  activeListId === list.id
                    ? "border-l-2 border-primary bg-surface-accent text-text-strong"
                    : "text-text-secondary hover:bg-surface-accent/50"
                )}
              >
                <div className="min-w-0 flex-1 text-left">
                  <p className="truncate text-sm font-medium">{list.name}</p>
                  <div className={cn(
                    "mt-0.5 flex items-center gap-1 text-[11px]",
                    activeListId === list.id ? "text-text-secondary" : "text-text-muted",
                  )}>
                    <span className="tabular-nums">진행 {getListActiveCount(list)}</span>
                    <span>·</span>
                    <span className="tabular-nums">완료 {getListDoneCount(list)}</span>
                  </div>
                  <p
                    className={cn(
                      "mt-0.5 truncate text-[11px]",
                      activeListId === list.id ? "text-text-secondary" : "text-text-muted",
                    )}
                  >
                    {getListSourceLabel(list)}
                  </p>
                </div>
                {!list.is_virtual && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setListDeleteTarget(list);
                    }}
                    className="ml-1 hidden text-text-muted hover:text-error group-hover/list:inline-block text-xs"
                    aria-label={`${list.name} 목록 삭제`}
                  >
                    ×
                  </button>
                )}
              </div>
            </div>
          ))}
          {showNewList ? (
            <div className="mt-1 space-y-1">
              <Input
                autoFocus
                placeholder="목록 이름"
                value={newListName}
                onChange={(e) => setNewListName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleCreateList();
                  if (e.key === "Escape") {
                    setShowNewList(false);
                    setNewListName("");
                    setNewListTarget("local");
                    setNewListGoogleAccountID("");
                  }
                }}
                className="h-8 flex-1"
              />
              <div className="flex gap-1">
                <select
                  value={newListTarget}
                  onChange={(e) => setNewListTarget(e.target.value as "local" | "google")}
                  className="h-8 flex-1 rounded-md border border-border bg-background px-2 text-xs"
                >
                  <option value="local">로컬 목록</option>
                  <option value="google">Google 목록</option>
                </select>
                {newListTarget === "google" && (
                  <select
                    value={newListGoogleAccountID}
                    onChange={(e) => setNewListGoogleAccountID(e.target.value)}
                    className="h-8 flex-1 rounded-md border border-border bg-background px-2 text-xs"
                  >
                    <option value="">계정 선택</option>
                    {googleAccounts.map((account) => (
                      <option key={account.id} value={account.id}>
                        {account.google_email}
                      </option>
                    ))}
                  </select>
                )}
              </div>
              <div className="flex gap-1">
                <Button
                  size="sm"
                  className="h-8 flex-1"
                  onClick={handleCreateList}
                  disabled={!newListName.trim() || (newListTarget === "google" && !newListGoogleAccountID)}
                >
                  생성
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-8"
                  onClick={() => {
                    setShowNewList(false);
                    setNewListName("");
                    setNewListTarget("local");
                    setNewListGoogleAccountID("");
                  }}
                >
                  취소
                </Button>
              </div>
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
            <span className="max-w-[9rem] truncate">{list.name}</span>
            <span className={cn(
              "rounded-full px-1.5 py-0.5 text-[10px] tabular-nums",
              activeListId === list.id ? "bg-white/20 text-white" : "bg-surface text-text-muted",
            )}>
              진행 {getListActiveCount(list)}
            </span>
          </button>
        ))}
        <button
          onClick={() => setShowNewList(true)}
          className="shrink-0 rounded-full bg-surface-accent px-3 py-1 text-sm text-text-muted"
        >
          +
        </button>
      </div>
      {showNewList && (
        <div className="md:hidden border-b border-border px-4 py-2 space-y-2">
          <Input
            placeholder="목록 이름"
            value={newListName}
            onChange={(e) => setNewListName(e.target.value)}
            className="h-9"
          />
          <div className="flex gap-2">
            <select
              value={newListTarget}
              onChange={(e) => setNewListTarget(e.target.value as "local" | "google")}
              className="h-9 flex-1 rounded-md border border-border bg-background px-2 text-xs"
            >
              <option value="local">로컬 목록</option>
              <option value="google">Google 목록</option>
            </select>
            {newListTarget === "google" && (
              <select
                value={newListGoogleAccountID}
                onChange={(e) => setNewListGoogleAccountID(e.target.value)}
                className="h-9 flex-1 rounded-md border border-border bg-background px-2 text-xs"
              >
                <option value="">계정 선택</option>
                {googleAccounts.map((account) => (
                  <option key={account.id} value={account.id}>
                    {account.google_email}
                  </option>
                ))}
              </select>
            )}
          </div>
          <div className="flex gap-2">
            <Button
              size="sm"
              className="flex-1"
              onClick={handleCreateList}
              disabled={!newListName.trim() || (newListTarget === "google" && !newListGoogleAccountID)}
            >
              생성
            </Button>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setShowNewList(false);
                setNewListName("");
                setNewListTarget("local");
                setNewListGoogleAccountID("");
              }}
            >
              취소
            </Button>
          </div>
        </div>
      )}

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
          lastSyncedAt={lastSyncedAt}
          syncingNow={syncingNow}
          refreshing={refreshing}
          onManualSync={handleManualSync}
        />

        {/* Add todo button */}
        <button
          onClick={() => {
            if (isAllView) return;
            setCreateParentId(undefined);
            setShowCreateDialog(true);
          }}
          disabled={isAllView}
          className={cn(
            "flex w-full items-center gap-2 border-b border-border px-4 py-2.5 text-sm text-text-muted transition-colors",
            isAllView ? "cursor-not-allowed opacity-60" : "hover:bg-surface-accent/50",
          )}
        >
          <Plus size={14} />
          {isAllView ? "전체 뷰에서는 Todo 추가 불가" : "새 Todo 추가..."}
        </button>

        {/* Todo list */}
        <div className="relative flex-1 overflow-auto">
          {!settingsLoaded || loading ? (
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
                    listLabel={isAllView ? listNameByID.get(activeTodo.list_id) : undefined}
                    depth={projectedDepth}
                    isOverdue={isOverdueTodo(activeTodo)}
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

      {/* List Delete Confirm Modal */}
      <Dialog open={!!listDeleteTarget} onOpenChange={(v) => !v && setListDeleteTarget(null)}>
        {listDeleteTarget && (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>목록 삭제</DialogTitle>
            </DialogHeader>
            <p className="text-sm text-text-secondary">
              &quot;{listDeleteTarget.name}&quot; 목록을 삭제하시겠습니까?
            </p>
            <DialogFooter className="justify-end gap-2">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setListDeleteTarget(null)}
                disabled={deletingList}
              >
                취소
              </Button>
              <Button
                type="button"
                variant="danger"
                size="sm"
                onClick={handleConfirmDeleteList}
                disabled={deletingList}
              >
                {deletingList ? "삭제 중..." : "삭제"}
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
