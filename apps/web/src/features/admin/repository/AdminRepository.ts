import type {
  AdminAuthTokenPair,
  AdminAuthUrlResponse,
  AdminUser,
  GoogleAccount,
  GoogleAccountStatus,
  UserDetail,
  UserItem,
} from "@/features/admin/domain/AdminEntities";

export interface ListUsersResult {
  users: UserItem[];
  nextCursor: string;
}

export interface AdminRepository {
  requestAuthUrl(): Promise<AdminAuthUrlResponse>;
  exchangeCode(code: string, state?: string): Promise<AdminAuthTokenPair>;
  listUsers(cursor?: string, query?: string): Promise<ListUsersResult>;
  getUser(userId: string): Promise<{ user: UserDetail; googleAccounts: GoogleAccount[] }>;
  updateUserQuota(userId: string, quotaBytes: number): Promise<void>;
  recalculateStorage(userId: string): Promise<void>;
  resetStorage(userId: string, confirm: string): Promise<void>;
  updateGoogleAccountStatus(userId: string, accountId: string, status: GoogleAccountStatus): Promise<void>;
  listAdmins(): Promise<AdminUser[]>;
  addAdmin(email: string, role: AdminUser["Role"]): Promise<void>;
  updateAdminRole(adminId: string, role: AdminUser["Role"]): Promise<void>;
  deactivateAdmin(adminId: string): Promise<void>;
}
