import type { AuthUrlResponse, GoogleAccountSummary, TriggerGoogleSyncInput } from "../domain/AuthSession";

export interface AuthRepository {
  requestAuthUrl(): Promise<AuthUrlResponse>;
  listGoogleAccounts(): Promise<GoogleAccountSummary[]>;
  triggerGoogleSync(input: TriggerGoogleSyncInput): Promise<number>;
}
