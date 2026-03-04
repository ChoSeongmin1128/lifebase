import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type { TodoItem } from "@/features/todo/domain/dnd-tree";
import type {
  CreateListInput,
  ReorderItem,
  TodoAppRepository,
  TodoListItem,
} from "@/features/todo/repository/TodoAppRepository";

interface ListsResponse {
  lists?: TodoListItem[];
}

interface TodosResponse {
  todos?: TodoItem[];
}

export class HttpTodoAppRepository implements TodoAppRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listLists(): Promise<TodoListItem[]> {
    const token = this.getToken();
    const data = await api<ListsResponse>("/todo/lists", { token });
    return data.lists || [];
  }

  createList(input: CreateListInput): Promise<TodoListItem> {
    const token = this.getToken();
    return api<TodoListItem>("/todo/lists", {
      method: "POST",
      body: {
        name: input.name,
        target: input.target || "local",
        google_account_id: input.google_account_id || undefined,
      },
      token,
    });
  }

  async deleteList(listId: string): Promise<void> {
    const token = this.getToken();
    await api(`/todo/lists/${listId}`, { method: "DELETE", token });
  }

  async listTodos(listId: string, includeDone: boolean): Promise<TodoItem[]> {
    const token = this.getToken();
    const params = new URLSearchParams({
      list_id: listId,
      include_done: includeDone ? "true" : "false",
    });
    const data = await api<TodosResponse>(`/todo?${params.toString()}`, { token });
    return data.todos || [];
  }

  async updateTodo(todoId: string, updates: Record<string, unknown>): Promise<void> {
    const token = this.getToken();
    await api(`/todo/${todoId}`, {
      method: "PATCH",
      body: updates,
      token,
    });
  }

  async deleteTodo(todoId: string): Promise<void> {
    const token = this.getToken();
    await api(`/todo/${todoId}`, { method: "DELETE", token });
  }

  async reorder(items: ReorderItem[]): Promise<void> {
    const token = this.getToken();
    await api("/todo/reorder", {
      method: "PATCH",
      body: { items },
      token,
    });
  }
}
