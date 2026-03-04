import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type { AuthRepository } from "@/features/auth/repository/AuthRepository";
import type {
  AuthApp,
  AuthCallbackInput,
  AuthTokenPair,
  AuthUrlResponse,
  GoogleAccountSummary,
  SyncGoogleAccountInput,
  TriggerGoogleSyncInput,
  TriggerGoogleSyncResponse,
} from "@/features/auth/domain/AuthSession";

interface GoogleAccountsResponse {
  accounts?: GoogleAccountSummary[];
}

export class HttpAuthRepository implements AuthRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  requestAuthUrl(app: AuthApp = "web"): Promise<AuthUrlResponse> {
    return api<AuthUrlResponse>(`/auth/url?app=${encodeURIComponent(app)}`);
  }

  exchangeCode(input: AuthCallbackInput): Promise<AuthTokenPair> {
    return api<AuthTokenPair>("/auth/callback", {
      method: "POST",
      body: input,
    });
  }

  async listGoogleAccounts(): Promise<GoogleAccountSummary[]> {
    const token = this.getToken();
    const data = await api<GoogleAccountsResponse>("/auth/google-accounts", { token });
    return data.accounts || [];
  }

  async linkGoogleAccount(input: AuthCallbackInput): Promise<void> {
    const token = this.getToken();
    await api("/auth/google-accounts/link", {
      method: "POST",
      body: input,
      token,
    });
  }

  async syncGoogleAccount(accountID: string, input: SyncGoogleAccountInput): Promise<void> {
    const token = this.getToken();
    await api(`/auth/google-accounts/${encodeURIComponent(accountID)}/sync`, {
      method: "POST",
      body: input,
      token,
    });
  }

  async triggerGoogleSync(input: TriggerGoogleSyncInput): Promise<number> {
    const token = this.getToken();
    const data = await api<TriggerGoogleSyncResponse>("/auth/google-sync/trigger", {
      method: "POST",
      body: input,
      token,
    });
    return data.scheduled_accounts || 0;
  }
}
