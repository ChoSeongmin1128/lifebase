"use client";

import { useMemo } from "react";
import { AuthFlowUseCase } from "@/features/auth/usecase/AuthFlow";
import { HttpAuthRepository } from "@/features/auth/infrastructure/httpAuthRepository";
import type { AuthApp, AuthCallbackInput } from "@/features/auth/domain/AuthSession";

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
    }),
    [useCase],
  );
}
