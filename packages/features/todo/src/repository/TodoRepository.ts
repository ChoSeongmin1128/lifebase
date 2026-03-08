import type { Todo } from "../domain/Todo";

export interface CreateTodoParams {
  listId: string;
  parentId?: string;
  title: string;
  notes?: string;
  dueDate?: string | null;
  dueTime?: string | null;
}

export interface CreateTodoRequestContext {
  requestId: string;
  requestedAt: string;
}

export interface TodoRepository {
  createTodo(params: CreateTodoParams, ctx: CreateTodoRequestContext): Promise<Todo>;
}
