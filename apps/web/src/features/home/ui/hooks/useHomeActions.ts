"use client";

import { useMemo } from "react";
import { BrowseHomeUseCase } from "@/features/home/usecase/BrowseHome";
import { HttpHomeRepository } from "@/features/home/infrastructure/httpHomeRepository";
import type { GetHomeSummaryInput } from "@/features/home/domain/HomeSummary";

export function useHomeActions() {
  const useCase = useMemo(() => {
    return new BrowseHomeUseCase(new HttpHomeRepository());
  }, []);

  return useMemo(
    () => ({
      getSummary: (input: GetHomeSummaryInput) => useCase.getSummary(input),
    }),
    [useCase],
  );
}
