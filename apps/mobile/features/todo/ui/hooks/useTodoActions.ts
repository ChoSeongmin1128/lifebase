import { useMemo } from "react";
import { ManageTodoUseCase } from "../../usecase/ManageTodo";
import { HttpTodoMobileRepository } from "../../infrastructure/httpTodoMobileRepository";
import type { MobileCreateListInput } from "../../repository/TodoMobileRepository";

export function useTodoActions() {
  const useCase = useMemo(() => {
    return new ManageTodoUseCase(new HttpTodoMobileRepository());
  }, []);

  return useMemo(
    () => ({
      listLists: () => useCase.listLists(),
      createList: (input: MobileCreateListInput) => useCase.createList(input),
      listTodos: (listId: string) => useCase.listTodos(listId),
      updateDone: (todoId: string, done: boolean) => useCase.updateDone(todoId, done),
    }),
    [useCase],
  );
}
