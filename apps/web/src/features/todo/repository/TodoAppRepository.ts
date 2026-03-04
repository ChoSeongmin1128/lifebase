import type { TodoItem } from "@/features/todo/domain/dnd-tree";

export interface TodoListItem {
  id: string;
  name: string;
  sort_order: number;
  google_account_id?: string | null;
  active_count?: number;
  done_count?: number;
  total_count?: number;
  source?: "google" | "local" | string;
}

export interface ReorderItem {
  id: string;
  parent_id: string | null;
  sort_order: number;
}

export interface TodoAppRepository {
  listLists(): Promise<TodoListItem[]>;
  createList(name: string): Promise<TodoListItem>;
  deleteList(listId: string): Promise<void>;
  listTodos(listId: string, includeDone: boolean): Promise<TodoItem[]>;
  updateTodo(todoId: string, updates: Record<string, unknown>): Promise<void>;
  deleteTodo(todoId: string): Promise<void>;
  reorder(items: ReorderItem[]): Promise<void>;
}
