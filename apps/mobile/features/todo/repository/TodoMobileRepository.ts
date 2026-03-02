import type { MobileTodoItem, MobileTodoList } from "../domain/TodoEntities";

export interface TodoMobileRepository {
  listLists(): Promise<MobileTodoList[]>;
  listTodos(listId: string): Promise<MobileTodoItem[]>;
  updateDone(todoId: string, done: boolean): Promise<void>;
}
