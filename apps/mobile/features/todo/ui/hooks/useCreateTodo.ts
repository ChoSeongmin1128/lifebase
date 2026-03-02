import { useCallback, useMemo, useState } from "react";
import {
  CreateTodoUseCase,
  type Clock,
  type IdGenerator,
  type CreateTodoParams,
} from "@lifebase/features-todo";
import { HttpTodoRepository } from "../../infrastructure/httpTodoRepository";

function createMobileIdGenerator(): IdGenerator {
  return {
    nextId: () => `${Date.now()}-${Math.random().toString(16).slice(2)}`,
  };
}

export function useCreateTodo() {
  const [creating, setCreating] = useState(false);

  const useCase = useMemo(() => {
    const clock: Clock = { now: () => new Date() };
    const idGen = createMobileIdGenerator();
    const todoRepo = new HttpTodoRepository();
    return new CreateTodoUseCase({ todoRepo, clock, idGen });
  }, []);

  const createTodo = useCallback(
    async (input: CreateTodoParams) => {
      setCreating(true);
      try {
        return await useCase.execute(input);
      } finally {
        setCreating(false);
      }
    },
    [useCase],
  );

  return { createTodo, creating };
}
