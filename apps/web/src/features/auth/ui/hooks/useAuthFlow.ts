"use client";

import { useMemo } from "react";
import { AuthFlowUseCase } from "@/features/auth/usecase/AuthFlow";
import { HttpAuthRepository } from "@/features/auth/infrastructure/httpAuthRepository";
import type { AuthCallbackInput } from "@/features/auth/domain/AuthSession";

export function useAuthFlow() {
  const useCase = useMemo(() => {
    return new AuthFlowUseCase(new HttpAuthRepository());
  }, []);

  return useMemo(
    () => ({
      requestAuthUrl: () => useCase.requestAuthUrl(),
      exchangeCode: (input: AuthCallbackInput) => useCase.exchangeCode(input),
    }),
    [useCase],
  );
}
