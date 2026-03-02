import type { AuthUrlResponse } from "../domain/AuthSession";

export interface AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse>;
}
