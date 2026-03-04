import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "./token-auth";
import type { AuthUrlResponse, GoogleAccountSummary, TriggerGoogleSyncInput } from "../domain/AuthSession";
import type { AuthRepository } from "../repository/AuthRepository";

interface GoogleAccountsResponse {
  accounts?: GoogleAccountSummary[];
}

interface TriggerGoogleSyncResponse {
  scheduled_accounts?: number;
}

export class HttpAuthRepository implements AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse> {
    return api<AuthUrlResponse>("/auth/url");
  }

  private async getToken(): Promise<string> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listGoogleAccounts(): Promise<GoogleAccountSummary[]> {
    const token = await this.getToken();
    const data = await api<GoogleAccountsResponse>("/auth/google-accounts", { token });
    return data.accounts || [];
  }

  async triggerGoogleSync(input: TriggerGoogleSyncInput): Promise<number> {
    const token = await this.getToken();
    const data = await api<TriggerGoogleSyncResponse>("/auth/google-sync/trigger", {
      method: "POST",
      body: input,
      token,
    });
    return data.scheduled_accounts || 0;
  }
}
