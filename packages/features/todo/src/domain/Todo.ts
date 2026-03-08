export type TodoPriority = "urgent" | "high" | "normal" | "low";

export interface Todo {
  id: string;
  listId: string;
  userId: string;
  parentId: string | null;
  title: string;
  notes: string;
  dueDate: string | null;
  dueTime: string | null;
  priority: TodoPriority;
  isDone: boolean;
  isPinned: boolean;
  starredAt?: string | null;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

const TODO_PRIORITIES: TodoPriority[] = ["urgent", "high", "normal", "low"];

export function normalizeTodoTitle(raw: string): string {
  const title = raw.trim();
  if (!title) {
    throw new Error("title is required");
  }
  return title;
}

export function normalizeTodoPriority(raw: string | undefined): TodoPriority {
  if (!raw) return "normal";
  if (TODO_PRIORITIES.includes(raw as TodoPriority)) {
    return raw as TodoPriority;
  }
  throw new Error("invalid priority");
}
