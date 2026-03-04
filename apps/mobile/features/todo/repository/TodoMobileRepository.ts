import type { MobileTodoItem, MobileTodoList } from "../domain/TodoEntities";

export interface MobileCreateListInput {
  name: string;
  target?: "local" | "google";
  google_account_id?: string | null;
}

export interface TodoMobileRepository {
  listLists(): Promise<MobileTodoList[]>;
  createList(input: MobileCreateListInput): Promise<MobileTodoList>;
  listTodos(listId: string): Promise<MobileTodoItem[]>;
  updateDone(todoId: string, done: boolean): Promise<void>;
}
