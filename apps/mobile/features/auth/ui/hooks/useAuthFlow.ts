import { useMemo } from "react";
import { AuthFlowUseCase } from "../../usecase/AuthFlow";
import { HttpAuthRepository } from "../../infrastructure/httpAuthRepository";

export function useAuthFlow() {
  const useCase = useMemo(() => {
    return new AuthFlowUseCase(new HttpAuthRepository());
  }, []);

  return useMemo(
    () => ({
      requestAuthUrl: () => useCase.requestAuthUrl(),
    }),
    [useCase],
  );
}
