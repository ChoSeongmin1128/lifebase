import { adminApi } from "@/features/admin/infrastructure/http-admin-api";
import { getAdminAccessToken } from "@/features/admin/infrastructure/admin-auth";
import type {
  AdminAuthTokenPair,
  AdminAuthUrlResponse,
  AdminUser,
  GoogleAccount,
  GoogleAccountStatus,
  UserDetail,
  UserItem,
} from "@/features/admin/domain/AdminEntities";
import type { AdminRepository, ListUsersResult } from "@/features/admin/repository/AdminRepository";

interface ListUsersResponse {
  users: UserItem[];
  next_cursor?: string;
}

interface UserDetailResponse {
  user: UserDetail;
  google_accounts: GoogleAccount[];
}

interface AdminListResponse {
  admins: AdminUser[];
}

export class HttpAdminRepository implements AdminRepository {
  private getToken(): string {
    const token = getAdminAccessToken();
    if (!token) {
      throw new Error("관리자 인증이 필요합니다.");
    }
    return token;
  }

  requestAuthUrl(): Promise<AdminAuthUrlResponse> {
    return adminApi<AdminAuthUrlResponse>("/auth/url?app=admin");
  }

  exchangeCode(code: string, state?: string): Promise<AdminAuthTokenPair> {
    return adminApi<AdminAuthTokenPair>("/auth/callback", {
      method: "POST",
      body: { code, state, app: "admin" },
    });
  }

  async listUsers(cursor?: string, query?: string): Promise<ListUsersResult> {
    const token = this.getToken();
    const params = new URLSearchParams();
    params.set("limit", "20");
    if (query) {
      params.set("q", query);
    }
    if (cursor) {
      params.set("cursor", cursor);
    }

    const data = await adminApi<ListUsersResponse>(`/admin/users?${params.toString()}`, { token });
    return {
      users: data.users,
      nextCursor: data.next_cursor || "",
    };
  }

  async getUser(userId: string): Promise<{ user: UserDetail; googleAccounts: GoogleAccount[] }> {
    const token = this.getToken();
    const data = await adminApi<UserDetailResponse>(`/admin/users/${userId}`, { token });
    return {
      user: data.user,
      googleAccounts: data.google_accounts,
    };
  }

  async updateUserQuota(userId: string, quotaBytes: number): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/users/${userId}/quota`, {
      method: "PATCH",
      token,
      body: { quota_bytes: quotaBytes },
    });
  }

  async recalculateStorage(userId: string): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/users/${userId}/recalculate-storage`, {
      method: "POST",
      token,
    });
  }

  async resetStorage(userId: string, confirm: string): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/users/${userId}/reset-storage`, {
      method: "POST",
      token,
      body: { confirm },
    });
  }

  async updateGoogleAccountStatus(userId: string, accountId: string, status: GoogleAccountStatus): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/users/${userId}/google-accounts/${accountId}/status`, {
      method: "PATCH",
      token,
      body: { status },
    });
  }

  async listAdmins(): Promise<AdminUser[]> {
    const token = this.getToken();
    const data = await adminApi<AdminListResponse>("/admin/admins", { token });
    return data.admins;
  }

  async addAdmin(email: string, role: AdminUser["Role"]): Promise<void> {
    const token = this.getToken();
    await adminApi("/admin/admins", {
      method: "POST",
      token,
      body: { email, role },
    });
  }

  async updateAdminRole(adminId: string, role: AdminUser["Role"]): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/admins/${adminId}/role`, {
      method: "PATCH",
      token,
      body: { role },
    });
  }

  async deactivateAdmin(adminId: string): Promise<void> {
    const token = this.getToken();
    await adminApi(`/admin/admins/${adminId}/deactivate`, {
      method: "PATCH",
      token,
    });
  }
}
