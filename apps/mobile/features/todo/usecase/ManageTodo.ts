import type { TodoMobileRepository } from "../repository/TodoMobileRepository";

export class ManageTodoUseCase {
  constructor(private readonly repo: TodoMobileRepository) {}

  listLists() {
    return this.repo.listLists();
  }

  listTodos(listId: string) {
    return this.repo.listTodos(listId);
  }

  updateDone(todoId: string, done: boolean) {
    return this.repo.updateDone(todoId, done);
  }
}
