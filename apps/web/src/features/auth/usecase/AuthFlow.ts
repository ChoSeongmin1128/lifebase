import type { AuthCallbackInput, AuthTokenPair, AuthUrlResponse } from "@/features/auth/domain/AuthSession";
import type { AuthRepository } from "@/features/auth/repository/AuthRepository";

export class AuthFlowUseCase {
  constructor(private readonly repo: AuthRepository) {}

  requestAuthUrl(): Promise<AuthUrlResponse> {
    return this.repo.requestAuthUrl();
  }

  exchangeCode(input: AuthCallbackInput): Promise<AuthTokenPair> {
    const code = input.code.trim();
    if (!code) {
      throw new Error("인증 코드가 없습니다.");
    }

    return this.repo.exchangeCode({
      ...input,
      code,
    });
  }
}
