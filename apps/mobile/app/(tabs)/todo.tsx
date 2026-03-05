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
import { formatDueYYMMDD } from "../../features/todo/lib/formatDueDate";

export default function TodoScreen() {
  const [lists, setLists] = useState<TodoList[]>([]);
  const [selectedList, setSelectedList] = useState<string | null>(null);
  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [newTitle, setNewTitle] = useState("");
  const [newListName, setNewListName] = useState("");
  const [newListTarget, setNewListTarget] = useState<"local" | "google">("local");
  const [newListGoogleAccountID, setNewListGoogleAccountID] = useState("");
  const [googleAccounts, setGoogleAccounts] = useState<GoogleAccountSummary[]>([]);
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
      prev.map((t) => (t.id === id ? { ...t, done: !done } : t))
    );
    try {
      await updateDone(id, !done);
    } catch {
      setTodos((prev) =>
        prev.map((t) => (t.id === id ? { ...t, done } : t))
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

  const priorityColor: Record<string, string> = {
    urgent: "#DC2626",
    high: "#F97316",
    normal: "#666",
    low: "#9CA3AF",
  };

  const pinned = todos.filter((t) => t.is_pinned && !t.done);
  const active = todos.filter((t) => !t.is_pinned && !t.done);
  const done = todos.filter((t) => t.done);

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
        data={[...pinned, ...active, ...done]}
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
              {item.due_date && (
                <Text style={styles.dueDate}>{formatDueYYMMDD(item.due_date)}</Text>
              )}
            </View>
            <View
              style={[
                styles.priorityDot,
                { backgroundColor: priorityColor[item.priority] || "#666" },
              ]}
            />
          </TouchableOpacity>
        )}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: "#fff" },
  listBar: {
    maxHeight: 48,
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
  },
  listChip: {
    paddingHorizontal: 14,
    paddingVertical: 6,
    borderRadius: 16,
    backgroundColor: "#f0f0f0",
    marginRight: 8,
  },
  listChipActive: { backgroundColor: "#000" },
  listChipText: { fontSize: 13, color: "#666" },
  listChipTextActive: { color: "#fff", fontWeight: "600" },
  listChipMeta: { fontSize: 10, color: "#9ca3af", marginTop: 1 },
  listChipMetaActive: { color: "#e5e7eb" },
  inputRow: {
    padding: 12,
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
  },
  input: {
    backgroundColor: "#f5f5f5",
    borderRadius: 8,
    padding: 12,
    fontSize: 14,
  },
  newListRow: {
    paddingHorizontal: 12,
    paddingTop: 10,
    paddingBottom: 12,
    borderBottomWidth: 1,
    borderBottomColor: "#eee",
    gap: 8,
  },
  newListInput: {
    backgroundColor: "#f5f5f5",
    borderRadius: 8,
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
    borderColor: "#d1d5db",
    backgroundColor: "#f9fafb",
  },
  targetChipActive: {
    backgroundColor: "#111827",
    borderColor: "#111827",
  },
  targetChipText: {
    fontSize: 12,
    color: "#4b5563",
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
    borderColor: "#d1d5db",
    backgroundColor: "#fff",
    marginRight: 8,
    maxWidth: 240,
  },
  googleAccountChipActive: {
    borderColor: "#2563eb",
    backgroundColor: "#dbeafe",
  },
  googleAccountChipText: {
    fontSize: 12,
    color: "#374151",
  },
  googleAccountChipTextActive: {
    color: "#1d4ed8",
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
    borderRadius: 8,
    backgroundColor: "#111827",
  },
  createListButtonDisabled: {
    backgroundColor: "#9ca3af",
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
    borderBottomColor: "#f0f0f0",
  },
  todoDone: { opacity: 0.5 },
  check: { fontSize: 20, marginRight: 12 },
  todoContent: { flex: 1 },
  todoTitle: { fontSize: 15 },
  todoTitleDone: { textDecorationLine: "line-through", color: "#999" },
  dueDate: { fontSize: 11, color: "#999", marginTop: 2 },
  priorityDot: { width: 8, height: 8, borderRadius: 4 },
  empty: { textAlign: "center", marginTop: 60, color: "#999", fontSize: 14 },
});
