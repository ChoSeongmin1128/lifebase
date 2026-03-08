import {
  type Todo,
  type TodoPriority,
  type TodoRepository,
  type CreateTodoParams,
  type CreateTodoRequestContext,
} from "@lifebase/features-todo";
import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";

interface ApiTodoResponse {
  id: string;
  list_id: string;
  user_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due_date: string | null;
  due_time: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
  starred_at?: string | null;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

function toDomainTodo(data: ApiTodoResponse): Todo {
  return {
    id: data.id,
    listId: data.list_id,
    userId: data.user_id,
    parentId: data.parent_id,
    title: data.title,
    notes: data.notes,
    dueDate: data.due_date,
    dueTime: data.due_time,
    priority: data.priority as TodoPriority,
    isDone: data.is_done,
    isPinned: data.is_pinned,
    starredAt: data.starred_at ?? null,
    sortOrder: data.sort_order,
    createdAt: data.created_at,
    updatedAt: data.updated_at,
  };
}

export class HttpTodoRepository implements TodoRepository {
  async createTodo(params: CreateTodoParams, ctx: CreateTodoRequestContext): Promise<Todo> {
    void ctx;
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const payload = {
      list_id: params.listId,
      title: params.title,
      notes: params.notes || "",
      due_date: params.dueDate ?? null,
      due_time: params.dueDate ? (params.dueTime ?? null) : null,
      priority: params.priority || "normal",
      ...(params.parentId ? { parent_id: params.parentId } : {}),
    };

    const created = await api<ApiTodoResponse>("/todo", {
      method: "POST",
      body: payload,
      token,
    });
    return toDomainTodo(created);
  }
}
