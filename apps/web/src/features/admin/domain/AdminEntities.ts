export type UserItem = {
  ID: string;
  Email: string;
  Name: string;
  StorageQuotaBytes: number;
  StorageUsedBytes: number;
};

export type GoogleAccountStatus = "active" | "reauth_required" | "revoked";

export type GoogleAccount = {
  ID: string;
  GoogleEmail: string;
  GoogleID: string;
  Status: GoogleAccountStatus;
  IsPrimary: boolean;
  ConnectedAt: string;
};

export type UserDetail = {
  ID: string;
  Email: string;
  Name: string;
  StorageQuotaBytes: number;
  StorageUsedBytes: number;
};

export type AdminUser = {
  ID: string;
  UserID: string;
  Email: string;
  Name: string;
  Role: "admin" | "super_admin";
  IsActive: boolean;
};

export type HolidayRefreshResult = {
  months_total: number;
  months_refreshed: number;
  items_upserted: number;
  refreshed_at: string;
};

export interface AdminAuthUrlResponse {
  url: string;
  state: string;
}

export interface AdminAuthTokenPair {
  access_token: string;
  refresh_token: string;
  expires_in: number;
}
