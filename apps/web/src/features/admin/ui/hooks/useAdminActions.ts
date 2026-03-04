"use client";

import { useMemo } from "react";
import { ManageAdminUseCase } from "@/features/admin/usecase/ManageAdmin";
import { HttpAdminRepository } from "@/features/admin/infrastructure/httpAdminRepository";
import type { AdminUser, GoogleAccountStatus } from "@/features/admin/domain/AdminEntities";

export function useAdminActions() {
  const useCase = useMemo(() => {
    return new ManageAdminUseCase(new HttpAdminRepository());
  }, []);

  return useMemo(
    () => ({
      requestAuthUrl: () => useCase.requestAuthUrl(),
      exchangeCode: (code: string, state?: string) => useCase.exchangeCode(code, state),
      listUsers: (cursor?: string, query?: string) => useCase.listUsers(cursor, query),
      getUser: (userId: string) => useCase.getUser(userId),
      updateUserQuota: (userId: string, quotaBytes: number) => useCase.updateUserQuota(userId, quotaBytes),
      recalculateStorage: (userId: string) => useCase.recalculateStorage(userId),
      resetStorage: (userId: string, confirm: string) => useCase.resetStorage(userId, confirm),
      updateGoogleAccountStatus: (userId: string, accountId: string, status: GoogleAccountStatus) =>
        useCase.updateGoogleAccountStatus(userId, accountId, status),
      refreshHolidays: (fromYear?: number, toYear?: number) => useCase.refreshHolidays(fromYear, toYear),
      listAdmins: () => useCase.listAdmins(),
      addAdmin: (email: string, role: AdminUser["Role"]) => useCase.addAdmin(email, role),
      updateAdminRole: (adminId: string, role: AdminUser["Role"]) => useCase.updateAdminRole(adminId, role),
      deactivateAdmin: (adminId: string) => useCase.deactivateAdmin(adminId),
    }),
    [useCase],
  );
}
