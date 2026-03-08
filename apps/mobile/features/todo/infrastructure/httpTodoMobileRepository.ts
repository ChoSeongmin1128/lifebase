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
  notes?: string;
  due?: string | null;
  due_date?: string | null;
  due_time?: string | null;
  priority: string;
  is_done?: boolean;
  done?: boolean;
  is_pinned: boolean;
  starred_at?: string | null;
  sort_order?: number;
  created_at?: string;
  updated_at?: string;
}

function normalizeDueDate(value: string | null | undefined): string | null {
  if (!value) return null;
  const candidate = value.includes("T") ? value.slice(0, 10) : value;
  return /^\d{4}-\d{2}-\d{2}$/.test(candidate) ? candidate : null;
}

function normalizeDueTime(value: string | null | undefined): string | null {
  if (!value) return null;
  if (/^\d{2}:\d{2}/.test(value)) return value.slice(0, 5);
  const match = value.match(/T(\d{2}:\d{2})/);
  if (match) return match[1];
  return null;
}

function toMobileTodoItem(item: ApiTodoItemResponse): MobileTodoItem {
  const done = item.done ?? item.is_done ?? false;
  const dueDate = item.due_date ?? normalizeDueDate(item.due);
  const dueTime = item.due_time ?? normalizeDueTime(item.due);
  const due = dueDate ? (dueTime ? `${dueDate}T${dueTime}` : dueDate) : null;
  return {
    id: item.id,
    list_id: item.list_id,
    title: item.title,
    notes: item.notes || "",
    done,
    is_done: done,
    priority: item.priority || "normal",
    due,
    due_date: dueDate,
    due_time: dueTime,
    is_pinned: item.is_pinned,
    starred_at: item.starred_at ?? null,
    sort_order: item.sort_order ?? 0,
    created_at: item.created_at,
    updated_at: item.updated_at,
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
    const params = new URLSearchParams({
      list_id: listId,
      include_done: "true",
    });
    const data = await api<TodoResponse>(`/todo?${params.toString()}`, { token });
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
