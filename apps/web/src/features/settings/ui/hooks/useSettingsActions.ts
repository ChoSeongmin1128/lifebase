"use client";

import { useMemo } from "react";
import { ManageSettingsUseCase } from "@/features/settings/usecase/ManageSettings";
import { HttpSettingsRepository } from "@/features/settings/infrastructure/httpSettingsRepository";

export function useSettingsActions() {
  const useCase = useMemo(() => {
    return new ManageSettingsUseCase(new HttpSettingsRepository());
  }, []);

  return {
    getSettings: () => useCase.getSettings(),
    updateSetting: (key: string, value: string) => useCase.updateSetting(key, value),
  };
}
