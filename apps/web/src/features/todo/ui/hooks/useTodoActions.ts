"use client";

import { useMemo } from "react";
import { ManageTodoUseCase } from "@/features/todo/usecase/ManageTodo";
import { HttpTodoAppRepository } from "@/features/todo/infrastructure/httpTodoAppRepository";
import type { ReorderItem } from "@/features/todo/repository/TodoAppRepository";

export function useTodoActions() {
  const useCase = useMemo(() => {
    return new ManageTodoUseCase(new HttpTodoAppRepository());
  }, []);

  return useMemo(
    () => ({
      listLists: () => useCase.listLists(),
      createList: (name: string) => useCase.createList(name),
      deleteList: (listId: string) => useCase.deleteList(listId),
      listTodos: (listId: string, includeDone: boolean) => useCase.listTodos(listId, includeDone),
      updateTodo: (todoId: string, updates: Record<string, unknown>) => useCase.updateTodo(todoId, updates),
      deleteTodo: (todoId: string) => useCase.deleteTodo(todoId),
      reorder: (items: ReorderItem[]) => useCase.reorder(items),
    }),
    [useCase],
  );
}
