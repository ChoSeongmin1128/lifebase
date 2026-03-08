export interface Todo {
  id: string;
  listId: string;
  userId: string;
  parentId: string | null;
  title: string;
  notes: string;
  dueDate: string | null;
  dueTime: string | null;
  isDone: boolean;
  isPinned: boolean;
  starredAt?: string | null;
  sortOrder: number;
  createdAt: string;
  updatedAt: string;
}

export function normalizeTodoTitle(raw: string): string {
  const title = raw.trim();
  if (!title) {
    throw new Error("title is required");
  }
  return title;
}
