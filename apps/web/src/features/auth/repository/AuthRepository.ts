import type {
  AuthApp,
  AuthCallbackInput,
  AuthTokenPair,
  AuthUrlResponse,
  GoogleAccountSummary,
  SyncGoogleAccountInput,
  TriggerGoogleSyncInput,
} from "@/features/auth/domain/AuthSession";

export interface AuthRepository {
  requestAuthUrl(app?: AuthApp): Promise<AuthUrlResponse>;
  exchangeCode(input: AuthCallbackInput): Promise<AuthTokenPair>;
  listGoogleAccounts(): Promise<GoogleAccountSummary[]>;
  linkGoogleAccount(input: AuthCallbackInput): Promise<void>;
  syncGoogleAccount(accountID: string, input: SyncGoogleAccountInput): Promise<void>;
  triggerGoogleSync(input: TriggerGoogleSyncInput): Promise<number>;
}
