export interface AuthUrlResponse {
  url: string;
  state: string;
}

export type AuthApp = "web" | "admin";

export interface AuthCallbackInput {
  code: string;
  state?: string;
  app: AuthApp;
}

export interface AuthTokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}

export interface GoogleAccountSummary {
  id: string;
  google_email: string;
  status: "active" | "reauth_required" | "revoked" | string;
  is_primary: boolean;
  connected_at: string;
}

export interface SyncGoogleAccountInput {
  sync_calendar: boolean;
  sync_todo: boolean;
}

export interface TriggerGoogleSyncInput {
  area?: "calendar" | "todo" | "both";
  reason?: "page_enter" | "page_action" | "tab_heartbeat" | "manual";
}

export interface TriggerGoogleSyncResponse {
  scheduled_accounts: number;
}
