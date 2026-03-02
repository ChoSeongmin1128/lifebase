import { useState, useEffect, useCallback } from "react";
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  TextInput,
  StyleSheet,
  RefreshControl,
} from "react-native";
import { api } from "../../lib/api";
import { getAccessToken } from "../../lib/auth";
import { useCreateTodo } from "../../features/todo/ui/hooks/useCreateTodo";

type TodoItem = {
  id: string;
  title: string;
  done: boolean;
  priority: string;
  due_date?: string;
  is_pinned: boolean;
};

type TodoList = {
  id: string;
  name: string;
};

export default function TodoScreen() {
  const [lists, setLists] = useState<TodoList[]>([]);
  const [selectedList, setSelectedList] = useState<string | null>(null);
  const [todos, setTodos] = useState<TodoItem[]>([]);
  const [newTitle, setNewTitle] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const { createTodo, creating } = useCreateTodo();

  const loadLists = useCallback(async () => {
    const token = await getAccessToken();
    if (!token) return;
    try {
      const data = await api<{ lists: TodoList[] }>("/todo/lists", { token });
      const ls = data.lists || [];
      setLists(ls);
      if (!selectedList && ls.length > 0) setSelectedList(ls[0].id);
    } catch {
      setLists([]);
    }
  }, [selectedList]);

  const loadTodos = useCallback(async () => {
    if (!selectedList) return;
    const token = await getAccessToken();
    if (!token) return;
    try {
      const data = await api<{ todos: TodoItem[] }>(
        `/todo?list_id=${selectedList}`,
        { token }
      );
      setTodos(data.todos || []);
    } catch {
      setTodos([]);
    }
  }, [selectedList]);

  useEffect(() => {
    loadLists();
  }, [loadLists]);

  useEffect(() => {
    loadTodos();
  }, [loadTodos]);

  const onRefresh = async () => {
    setRefreshing(true);
    await loadTodos();
    setRefreshing(false);
  };

  const toggleDone = async (id: string, done: boolean) => {
    const token = await getAccessToken();
    if (!token) return;
    setTodos((prev) =>
      prev.map((t) => (t.id === id ? { ...t, done: !done } : t))
    );
    try {
      await api(`/todo/${id}`, {
        method: "PATCH",
        body: { done: !done },
        token,
      });
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

  const priorityColor: Record<string, string> = {
    urgent: "#DC2626",
    high: "#F97316",
    normal: "#666",
    low: "#9CA3AF",
  };

  const pinned = todos.filter((t) => t.is_pinned && !t.done);
  const active = todos.filter((t) => !t.is_pinned && !t.done);
  const done = todos.filter((t) => t.done);

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
          </TouchableOpacity>
        )}
      />

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
                <Text style={styles.dueDate}>{item.due_date}</Text>
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
