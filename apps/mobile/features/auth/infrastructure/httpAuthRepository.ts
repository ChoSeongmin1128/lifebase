import { api } from "../../shared/infrastructure/http-api";
import type { AuthUrlResponse } from "../domain/AuthSession";
import type { AuthRepository } from "../repository/AuthRepository";

export class HttpAuthRepository implements AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse> {
    return api<AuthUrlResponse>("/auth/url");
  }
}
