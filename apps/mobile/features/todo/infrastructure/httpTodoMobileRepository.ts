import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { MobileTodoItem, MobileTodoList } from "../domain/TodoEntities";
import type { MobileCreateListInput, TodoMobileRepository } from "../repository/TodoMobileRepository";

interface TodoListResponse {
  lists?: MobileTodoList[];
}

interface TodoResponse {
  todos?: ApiTodoItemResponse[];
}

interface ApiTodoItemResponse {
  id: string;
  list_id?: string;
  title: string;
  due?: string | null;
  due_date?: string | null;
  priority: string;
  is_done?: boolean;
  done?: boolean;
  is_pinned: boolean;
}

function toMobileTodoItem(item: ApiTodoItemResponse): MobileTodoItem {
  const done = item.done ?? item.is_done ?? false;
  const due = item.due ?? item.due_date ?? null;
  return {
    id: item.id,
    list_id: item.list_id,
    title: item.title,
    done,
    is_done: done,
    priority: item.priority || "normal",
    due,
    due_date: due || undefined,
    is_pinned: item.is_pinned,
  };
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
    return (data.todos || []).map(toMobileTodoItem);
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
