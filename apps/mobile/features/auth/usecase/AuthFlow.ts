import type { AuthRepository } from "../repository/AuthRepository";

export class AuthFlowUseCase {
  constructor(private readonly repo: AuthRepository) {}

  requestAuthUrl() {
    return this.repo.requestAuthUrl();
  }
}
