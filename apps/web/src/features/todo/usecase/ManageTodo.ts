import type {
  ReorderItem,
  TodoAppRepository,
} from "@/features/todo/repository/TodoAppRepository";

export class ManageTodoUseCase {
  constructor(private readonly repo: TodoAppRepository) {}

  listLists() {
    return this.repo.listLists();
  }

  createList(name: string) {
    const normalized = name.trim();
    if (!normalized) {
      throw new Error("목록 이름이 비어 있습니다.");
    }
    return this.repo.createList(normalized);
  }

  deleteList(listId: string) {
    return this.repo.deleteList(listId);
  }

  listTodos(listId: string, includeDone: boolean) {
    return this.repo.listTodos(listId, includeDone);
  }

  updateTodo(todoId: string, updates: Record<string, unknown>) {
    return this.repo.updateTodo(todoId, updates);
  }

  deleteTodo(todoId: string) {
    return this.repo.deleteTodo(todoId);
  }

  reorder(items: ReorderItem[]) {
    return this.repo.reorder(items);
  }
}
