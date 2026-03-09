import { useState, useEffect, useCallback, useMemo } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  TextInput,
  StyleSheet,
  RefreshControl,
} from "react-native";
import { useCreateTodo } from "../../features/todo/ui/hooks/useCreateTodo";
import { useTodoActions } from "../../features/todo/ui/hooks/useTodoActions";
import type { MobileTodoItem as TodoItem, MobileTodoList as TodoList } from "../../features/todo/domain/TodoEntities";
import { useAuthFlow } from "../../features/auth/ui/hooks/useAuthFlow";
import type { GoogleAccountSummary } from "../../features/auth/domain/AuthSession";
import { formatDueLabel } from "../../features/todo/lib/formatDueDate";

type TodoSortBy = "manual" | "due" | "recent_starred" | "title";

const SORT_OPTIONS: { value: TodoSortBy; label: string }[] = [
  { value: "manual", label: "내 순서" },
  { value: "due", label: "기한" },
  { value: "recent_starred", label: "최근 별표" },
  { value: "title", label: "제목" },
];

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

function sortTodos(items: TodoItem[], sortBy: TodoSortBy): TodoItem[] {
  return [...items].sort((a, b) => {
    const doneCmp = Number(a.done) - Number(b.done);
    if (doneCmp !== 0) return doneCmp;

    if (sortBy === "manual") {
      const pinCmp = Number(b.is_pinned) - Number(a.is_pinned);
      if (pinCmp !== 0) return pinCmp;
      const orderCmp = (a.sort_order ?? 0) - (b.sort_order ?? 0);
      if (orderCmp !== 0) return orderCmp;
      return compareDatesDesc(a.created_at, b.created_at);
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
  });
}

export default function TodoScreen() {
  const [lists, setLists] = useState<TodoList[]>([]);
  const [selectedList, setSelectedList] = useState<string | null>(null);
  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [newTitle, setNewTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [newListTarget, setNewListTarget] = useState<"local" | "google">("local");
  const [newListGoogleAccountID, setNewListGoogleAccountID] = useState("");
  const [googleAccounts, setGoogleAccounts] = useState<GoogleAccountSummary[]>([]);
  const [sortBy, setSortBy] = useState<TodoSortBy>("due");
  const [refreshing, setRefreshing] = useState(false);
  const { createTodo, creating } = useCreateTodo();
  const { listLists, createList, listTodos, updateDone } = useTodoActions();
  const { listGoogleAccounts } = useAuthFlow();
  const googleAccountEmailByID = useMemo(
    () => new Map(googleAccounts.map((account) => [account.id, account.google_email])),
    [googleAccounts]
  );

  const loadLists = useCallback(async () => {
    try {
      const ls = await listLists();
      setLists(ls);
      if (!selectedList && ls.length > 0) setSelectedList(ls[0].id);
    } catch {
      setLists([]);
    }
  }, [listLists, selectedList]);

  const loadTodos = useCallback(async () => {
    if (!selectedList) return;
    try {
      const data = await listTodos(selectedList);
      setTodos(data || []);
    } catch {
      setTodos([]);
    }
  }, [listTodos, selectedList]);

  useEffect(() => {
    loadLists();
  }, [loadLists]);

  useEffect(() => {
    const loadGoogleAccounts = async () => {
      try {
        const items = await listGoogleAccounts();
        setGoogleAccounts(items.filter((account) => account.status === "active"));
      } catch {
        setGoogleAccounts([]);
      }
    };
    loadGoogleAccounts();
  }, [listGoogleAccounts]);

  useEffect(() => {
    loadTodos();
  }, [loadTodos]);

  const onRefresh = async () => {
    setRefreshing(true);
    await loadTodos();
    setRefreshing(false);
  };

  const toggleDone = async (id: string, done: boolean) => {
    setTodos((prev) =>
      prev.map((t) => (t.id === id ? { ...t, done: !done, is_done: !done } : t))
    );
    try {
      await updateDone(id, !done);
    } catch {
      setTodos((prev) =>
        prev.map((t) => (t.id === id ? { ...t, done, is_done: done } : t))
      );
    }
  };

  const addTodo = async () => {
    if (!newTitle.trim() || !selectedList || creating) return;
    try {
      await createTodo({
        listId: selectedList,
        title: newTitle.trim(),
      });
      setNewTitle("");
      loadTodos();
    } catch (err) {
      console.error("Add todo failed:", err);
    }
  };

  const addList = async () => {
    const name = newListName.trim();
    if (!name) return;
    if (newListTarget === "google" && !newListGoogleAccountID) return;

    try {
      await createList({
        name,
        target: newListTarget,
        google_account_id: newListTarget === "google" ? newListGoogleAccountID : null,
      });
      setNewListName("");
      setNewListTarget("local");
      setNewListGoogleAccountID("");
      await loadLists();
    } catch (err) {
      console.error("Create list failed:", err);
    }
  };

  const sortedTodos = useMemo(() => sortTodos(todos, sortBy), [sortBy, todos]);

  const getListSourceLabel = useCallback((list: TodoList) => {
    if (list.source === "local") return "로컬";
    if (list.source === "google" || list.google_account_id) {
      if (list.google_account_email) return `Google · ${list.google_account_email}`;
      if (list.google_account_id && googleAccountEmailByID.has(list.google_account_id)) {
        return `Google · ${googleAccountEmailByID.get(list.google_account_id)}`;
      }
      return "Google · 계정 미확인";
    }
    return "로컬";
  }, [googleAccountEmailByID]);

  return (
    <View style={styles.container}>
      <View style={styles.headerCard}>
        <Text style={styles.screenTitle}>Todo</Text>
        <Text style={styles.screenSubtitle}>목록, 정렬, 입력 흐름을 같은 작업 표면으로 정리합니다.</Text>
      </View>
      <FlatList
        horizontal
        data={lists}
        keyExtractor={(l) => l.id}
        style={styles.listBar}
        showsHorizontalScrollIndicator={false}
        renderItem={({ item }) => (
          <TouchableOpacity
            style={[
              styles.listChip,
              selectedList === item.id && styles.listChipActive,
            ]}
            onPress={() => setSelectedList(item.id)}
          >
            <Text
              style={[
                styles.listChipText,
                selectedList === item.id && styles.listChipTextActive,
              ]}
            >
              {item.name}
            </Text>
            <Text
              style={[
                styles.listChipMeta,
                selectedList === item.id && styles.listChipMetaActive,
              ]}
              numberOfLines={1}
            >
              {getListSourceLabel(item)}
            </Text>
          </TouchableOpacity>
        )}
      />

      <View style={styles.newListRow}>
        <TextInput
          style={styles.newListInput}
          placeholder="새 목록 이름"
          value={newListName}
          onChangeText={setNewListName}
          returnKeyType="done"
          onSubmitEditing={addList}
        />
        <View style={styles.targetToggleRow}>
          <TouchableOpacity
            style={[styles.targetChip, newListTarget === "local" && styles.targetChipActive]}
            onPress={() => setNewListTarget("local")}
          >
            <Text style={[styles.targetChipText, newListTarget === "local" && styles.targetChipTextActive]}>
              로컬
            </Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.targetChip, newListTarget === "google" && styles.targetChipActive]}
            onPress={() => setNewListTarget("google")}
          >
            <Text style={[styles.targetChipText, newListTarget === "google" && styles.targetChipTextActive]}>
              Google
            </Text>
          </TouchableOpacity>
        </View>
        {newListTarget === "google" && (
          <FlatList
            horizontal
            data={googleAccounts}
            keyExtractor={(item) => item.id}
            showsHorizontalScrollIndicator={false}
            style={styles.googleAccountBar}
            renderItem={({ item }) => (
              <TouchableOpacity
                style={[
                  styles.googleAccountChip,
                  newListGoogleAccountID === item.id && styles.googleAccountChipActive,
                ]}
                onPress={() => setNewListGoogleAccountID(item.id)}
              >
                <Text
                  style={[
                    styles.googleAccountChipText,
                    newListGoogleAccountID === item.id && styles.googleAccountChipTextActive,
                  ]}
                  numberOfLines={1}
                >
                  {item.google_email}
                </Text>
              </TouchableOpacity>
            )}
            ListEmptyComponent={<Text style={styles.googleAccountEmpty}>연결된 Google 계정 없음</Text>}
          />
        )}
        <TouchableOpacity
          style={[
            styles.createListButton,
            (!newListName.trim() || (newListTarget === "google" && !newListGoogleAccountID)) && styles.createListButtonDisabled,
          ]}
          onPress={addList}
          disabled={!newListName.trim() || (newListTarget === "google" && !newListGoogleAccountID)}
        >
          <Text style={styles.createListButtonText}>목록 생성</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.inputRow}>
        <TextInput
          style={styles.input}
          placeholder="새 Todo 추가..."
          value={newTitle}
          editable={!creating}
          onChangeText={setNewTitle}
          onSubmitEditing={addTodo}
          returnKeyType="done"
        />
      </View>

      <FlatList
        horizontal
        data={SORT_OPTIONS}
        keyExtractor={(item) => item.value}
        style={styles.sortBar}
        contentContainerStyle={styles.sortBarContent}
        showsHorizontalScrollIndicator={false}
        renderItem={({ item }) => (
          <TouchableOpacity
            style={[styles.sortChip, sortBy === item.value && styles.sortChipActive]}
            onPress={() => setSortBy(item.value)}
          >
            <Text style={[styles.sortChipText, sortBy === item.value && styles.sortChipTextActive]}>
              {item.label}
            </Text>
          </TouchableOpacity>
        )}
      />

      <FlatList
        data={sortedTodos}
        keyExtractor={(item) => item.id}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={onRefresh} />
        }
        ListEmptyComponent={
          <Text style={styles.empty}>Todo가 없습니다</Text>
        }
        renderItem={({ item }) => (
          <TouchableOpacity
            style={[styles.todoRow, item.done && styles.todoDone]}
            onPress={() => toggleDone(item.id, item.done)}
          >
            <Text style={styles.check}>{item.done ? "☑" : "☐"}</Text>
            <View style={styles.todoContent}>
              <Text
                style={[styles.todoTitle, item.done && styles.todoTitleDone]}
                numberOfLines={1}
              >
                {item.is_pinned ? "📌 " : ""}
                {item.title}
              </Text>
              {item.notes ? (
                <Text
                  style={[styles.todoNotes, item.done && styles.todoTitleDone]}
                  numberOfLines={2}
                >
                  {item.notes}
                </Text>
              ) : null}
              {item.due_date ? (
                <Text style={styles.dueDate}>{formatDueLabel(item.due_date, item.due_time)}</Text>
              ) : null}
            </View>
          </TouchableOpacity>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#F7F8F6", padding: 16 },
  headerCard: {
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
    borderRadius: 20,
    paddingHorizontal: 16,
    paddingVertical: 14,
    marginBottom: 12,
  },
  screenTitle: { fontSize: 20, fontWeight: "700", color: "#111111" },
  screenSubtitle: { marginTop: 4, fontSize: 13, color: "#5E6B67" },
  listBar: {
    maxHeight: 48,
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    borderRadius: 18,
    backgroundColor: "#FFFFFF",
  },
  listChip: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 16,
    backgroundColor: "#EEF5F3",
    marginRight: 8,
  },
  listChipActive: { backgroundColor: "#1B998B" },
  listChipText: { fontSize: 13, color: "#5E6B67" },
  listChipTextActive: { color: "#fff", fontWeight: "600" },
  listChipMeta: { fontSize: 10, color: "#9ca3af", marginTop: 1 },
  listChipMetaActive: { color: "#e5e7eb" },
  inputRow: {
    padding: 12,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    borderRadius: 18,
    backgroundColor: "#FFFFFF",
    marginTop: 12,
  },
  input: {
    backgroundColor: "#F7F8F6",
    borderRadius: 12,
    padding: 12,
    fontSize: 14,
  },
  sortBar: {
    maxHeight: 46,
    marginTop: 12,
  },
  sortBarContent: {
    paddingHorizontal: 0,
    paddingVertical: 0,
    gap: 8,
  },
  sortChip: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
    marginRight: 8,
  },
  sortChipActive: {
    backgroundColor: "#1B998B",
    borderColor: "#1B998B",
  },
  sortChipText: {
    fontSize: 12,
    color: "#5E6B67",
    fontWeight: "500",
  },
  sortChipTextActive: {
    color: "#fff",
  },
  newListRow: {
    paddingHorizontal: 12,
    paddingTop: 10,
    paddingBottom: 12,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    borderRadius: 18,
    backgroundColor: "#FFFFFF",
    marginTop: 12,
    gap: 8,
  },
  newListInput: {
    backgroundColor: "#F7F8F6",
    borderRadius: 12,
    padding: 12,
    fontSize: 14,
  },
  targetToggleRow: {
    flexDirection: "row",
    gap: 8,
  },
  targetChip: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#FFFFFF",
  },
  targetChipActive: {
    backgroundColor: "#1B998B",
    borderColor: "#1B998B",
  },
  targetChipText: {
    fontSize: 12,
    color: "#5E6B67",
  },
  targetChipTextActive: {
    color: "#fff",
    fontWeight: "600",
  },
  googleAccountBar: {
    maxHeight: 36,
  },
  googleAccountChip: {
    paddingHorizontal: 10,
    paddingVertical: 6,
    borderRadius: 14,
    borderWidth: 1,
    borderColor: "#D8DFDC",
    backgroundColor: "#fff",
    marginRight: 8,
    maxWidth: 240,
  },
  googleAccountChipActive: {
    borderColor: "#1B998B",
    backgroundColor: "#E7F4F1",
  },
  googleAccountChipText: {
    fontSize: 12,
    color: "#43514D",
  },
  googleAccountChipTextActive: {
    color: "#1B998B",
    fontWeight: "600",
  },
  googleAccountEmpty: {
    color: "#9ca3af",
    fontSize: 12,
  },
  createListButton: {
    alignSelf: "flex-start",
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderRadius: 12,
    backgroundColor: "#1B998B",
  },
  createListButtonDisabled: {
    backgroundColor: "#C7D6D1",
  },
  createListButtonText: {
    color: "#fff",
    fontSize: 12,
    fontWeight: "600",
  },
  todoRow: {
    flexDirection: "row",
    alignItems: "center",
    padding: 14,
    borderBottomWidth: 1,
    borderBottomColor: "#E5ECE9",
    backgroundColor: "#FFFFFF",
  },
  todoDone: { opacity: 0.5, backgroundColor: "#F7FAF9" },
  check: { fontSize: 20, marginRight: 12 },
  todoContent: { flex: 1 },
  todoTitle: { fontSize: 15 },
  todoTitleDone: { textDecorationLine: "line-through", color: "#7B8784" },
  todoNotes: { fontSize: 12, color: "#6B7A76", marginTop: 3, lineHeight: 17 },
  dueDate: { fontSize: 11, color: "#7B8784", marginTop: 2 },
  empty: { textAlign: "center", marginTop: 60, color: "#7B8784", fontSize: 14 },
});
