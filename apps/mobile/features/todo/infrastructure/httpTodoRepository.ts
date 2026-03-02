import {
  type Todo,
  type TodoPriority,
  type TodoRepository,
  type CreateTodoParams,
  type CreateTodoRequestContext,
} from "@lifebase/features-todo";
import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";

interface ApiTodoResponse {
  id: string;
  list_id: string;
  user_id: string;
  parent_id: string | null;
  title: string;
  notes: string;
  due: string | null;
  priority: string;
  is_done: boolean;
  is_pinned: boolean;
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
    due: data.due,
    priority: data.priority as TodoPriority,
    isDone: data.is_done,
    isPinned: data.is_pinned,
    sortOrder: data.sort_order,
    createdAt: data.created_at,
    updatedAt: data.updated_at,
  };
}

export class HttpTodoRepository implements TodoRepository {
  async createTodo(params: CreateTodoParams, ctx: CreateTodoRequestContext): Promise<Todo> {
    void ctx;
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }

    const payload = {
      list_id: params.listId,
      title: params.title,
      notes: params.notes || "",
      due: params.due ?? null,
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
