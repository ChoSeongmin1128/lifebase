export interface AuthUrlResponse {
  url: string;
}

export interface GoogleAccountSummary {
  id: string;
  google_email: string;
  status: "active" | "reauth_required" | "revoked" | string;
  is_primary: boolean;
  connected_at: string;
}

export interface TriggerGoogleSyncInput {
  area?: "calendar" | "todo" | "both";
  reason?: "page_enter" | "page_action" | "tab_heartbeat" | "manual";
}
