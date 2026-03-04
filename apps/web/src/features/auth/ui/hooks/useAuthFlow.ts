"use client";

import { useMemo } from "react";
import { AuthFlowUseCase } from "@/features/auth/usecase/AuthFlow";
import { HttpAuthRepository } from "@/features/auth/infrastructure/httpAuthRepository";
import type { AuthApp, AuthCallbackInput, SyncGoogleAccountInput, TriggerGoogleSyncInput } from "@/features/auth/domain/AuthSession";

export function useAuthFlow() {
  const useCase = useMemo(() => {
    return new AuthFlowUseCase(new HttpAuthRepository());
  }, []);

  return useMemo(
    () => ({
      requestAuthUrl: (app?: AuthApp) => useCase.requestAuthUrl(app),
      exchangeCode: (input: AuthCallbackInput) => useCase.exchangeCode(input),
      listGoogleAccounts: () => useCase.listGoogleAccounts(),
      linkGoogleAccount: (input: AuthCallbackInput) => useCase.linkGoogleAccount(input),
      syncGoogleAccount: (accountID: string, input: SyncGoogleAccountInput) => useCase.syncGoogleAccount(accountID, input),
      triggerGoogleSync: (input: TriggerGoogleSyncInput) => useCase.triggerGoogleSync(input),
    }),
    [useCase],
  );
}
