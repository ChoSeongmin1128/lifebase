import type { AuthCallbackInput, AuthTokenPair, AuthUrlResponse } from "@/features/auth/domain/AuthSession";

export interface AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse>;
  exchangeCode(input: AuthCallbackInput): Promise<AuthTokenPair>;
}
