import type { AdminUser, GoogleAccountStatus } from "@/features/admin/domain/AdminEntities";
import type { AdminRepository } from "@/features/admin/repository/AdminRepository";

export class ManageAdminUseCase {
  constructor(private readonly repo: AdminRepository) {}

  requestAuthUrl() {
    return this.repo.requestAuthUrl();
  }

  exchangeCode(code: string, state?: string) {
    const normalized = code.trim();
    if (!normalized) {
      throw new Error("인증 코드가 없습니다.");
    }
    return this.repo.exchangeCode(normalized, state);
  }

  listUsers(cursor?: string, query?: string) {
    return this.repo.listUsers(cursor, query?.trim());
  }

  getUser(userId: string) {
    return this.repo.getUser(userId);
  }

  updateUserQuota(userId: string, quotaBytes: number) {
    if (!Number.isFinite(quotaBytes) || quotaBytes <= 0) {
      throw new Error("할당량은 0보다 커야 합니다.");
    }
    return this.repo.updateUserQuota(userId, quotaBytes);
  }

  recalculateStorage(userId: string) {
    return this.repo.recalculateStorage(userId);
  }

  resetStorage(userId: string, confirm: string) {
    return this.repo.resetStorage(userId, confirm);
  }

  updateGoogleAccountStatus(userId: string, accountId: string, status: GoogleAccountStatus) {
    return this.repo.updateGoogleAccountStatus(userId, accountId, status);
  }

  refreshHolidays(fromYear?: number, toYear?: number) {
    return this.repo.refreshHolidays(fromYear, toYear);
  }

  listAdmins() {
    return this.repo.listAdmins();
  }

  addAdmin(email: string, role: AdminUser["Role"]) {
    const normalized = email.trim();
    if (!normalized) {
      throw new Error("이메일을 입력해 주세요.");
    }
    return this.repo.addAdmin(normalized, role);
  }

  updateAdminRole(adminId: string, role: AdminUser["Role"]) {
    return this.repo.updateAdminRole(adminId, role);
  }

  deactivateAdmin(adminId: string) {
    return this.repo.deactivateAdmin(adminId);
  }
}
