import type { AuthRepository } from "../repository/AuthRepository";
import type { TriggerGoogleSyncInput } from "../domain/AuthSession";

export class AuthFlowUseCase {
  constructor(private readonly repo: AuthRepository) {}

  requestAuthUrl() {
    return this.repo.requestAuthUrl();
  }

  listGoogleAccounts() {
    return this.repo.listGoogleAccounts();
  }

  triggerGoogleSync(input: TriggerGoogleSyncInput) {
    return this.repo.triggerGoogleSync(input);
  }
}
