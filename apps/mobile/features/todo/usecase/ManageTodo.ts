import type { MobileCreateListInput } from "../repository/TodoMobileRepository";
import type { TodoMobileRepository } from "../repository/TodoMobileRepository";

export class ManageTodoUseCase {
  constructor(private readonly repo: TodoMobileRepository) {}

  listLists() {
    return this.repo.listLists();
  }

  createList(input: MobileCreateListInput) {
    const name = input.name.trim();
    if (!name) {
      throw new Error("목록 이름이 비어 있습니다.");
    }
    return this.repo.createList({ ...input, name });
  }

  listTodos(listId: string) {
    return this.repo.listTodos(listId);
  }

  updateDone(todoId: string, done: boolean) {
    return this.repo.updateDone(todoId, done);
  }
}
