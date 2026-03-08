import { normalizeTodoPriority, normalizeTodoTitle, type Todo } from "../domain/Todo";
import type { CreateTodoParams, TodoRepository } from "../repository/TodoRepository";

export interface Clock {
  now(): Date;
}

export interface IdGenerator {
  nextId(): string;
}

export interface CreateTodoDependencies {
  todoRepo: TodoRepository;
  clock: Clock;
  idGen: IdGenerator;
}

export class CreateTodoUseCase {
  constructor(private readonly deps: CreateTodoDependencies) {}

  async execute(input: CreateTodoParams): Promise<Todo> {
    const listId = input.listId.trim();
    if (!listId) {
      throw new Error("list_id is required");
    }
    if (input.dueTime && !input.dueDate) {
      throw new Error("due_date is required when due_time is set");
    }

    const normalizedInput: CreateTodoParams = {
      ...input,
      listId,
      title: normalizeTodoTitle(input.title),
      priority: normalizeTodoPriority(input.priority),
    };

    return this.deps.todoRepo.createTodo(normalizedInput, {
      requestId: this.deps.idGen.nextId(),
      requestedAt: this.deps.clock.now().toISOString(),
    });
  }
}
