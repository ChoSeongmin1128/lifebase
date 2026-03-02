import { useMemo } from "react";
import { ManageTodoUseCase } from "../../usecase/ManageTodo";
import { HttpTodoMobileRepository } from "../../infrastructure/httpTodoMobileRepository";

export function useTodoActions() {
  const useCase = useMemo(() => {
    return new ManageTodoUseCase(new HttpTodoMobileRepository());
  }, []);

  return useMemo(
    () => ({
      listLists: () => useCase.listLists(),
      listTodos: (listId: string) => useCase.listTodos(listId),
      updateDone: (todoId: string, done: boolean) => useCase.updateDone(todoId, done),
    }),
    [useCase],
  );
}
