import { api } from "@/features/shared/infrastructure/http-api";
import type { AuthRepository } from "@/features/auth/repository/AuthRepository";
import type { AuthCallbackInput, AuthTokenPair, AuthUrlResponse } from "@/features/auth/domain/AuthSession";

export class HttpAuthRepository implements AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse> {
    return api<AuthUrlResponse>("/auth/url");
  }

  exchangeCode(input: AuthCallbackInput): Promise<AuthTokenPair> {
    return api<AuthTokenPair>("/auth/callback", {
      method: "POST",
      body: input,
    });
  }
}
