import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { MobileTodoItem, MobileTodoList } from "../domain/TodoEntities";
import type { MobileCreateListInput, TodoMobileRepository } from "../repository/TodoMobileRepository";

interface TodoListResponse {
  lists?: MobileTodoList[];
}

interface TodoResponse {
  todos?: MobileTodoItem[];
}

export class HttpTodoMobileRepository implements TodoMobileRepository {
  private async getToken(): Promise<string> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listLists(): Promise<MobileTodoList[]> {
    const token = await this.getToken();
    const data = await api<TodoListResponse>("/todo/lists", { token });
    return data.lists || [];
  }

  async createList(input: MobileCreateListInput): Promise<MobileTodoList> {
    const token = await this.getToken();
    return api<MobileTodoList>("/todo/lists", {
      method: "POST",
      body: {
        name: input.name,
        target: input.target || "local",
        google_account_id: input.google_account_id || undefined,
      },
      token,
    });
  }

  async listTodos(listId: string): Promise<MobileTodoItem[]> {
    const token = await this.getToken();
    const data = await api<TodoResponse>(`/todo?list_id=${encodeURIComponent(listId)}`, { token });
    return data.todos || [];
  }

  async updateDone(todoId: string, done: boolean): Promise<void> {
    const token = await this.getToken();
    await api(`/todo/${todoId}`, {
      method: "PATCH",
      body: { is_done: done },
      token,
    });
  }
}
