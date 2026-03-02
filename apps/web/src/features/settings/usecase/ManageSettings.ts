import type { SettingsRepository } from "@/features/settings/repository/SettingsRepository";

export class ManageSettingsUseCase {
  constructor(private readonly repo: SettingsRepository) {}

  getSettings() {
    return this.repo.getSettings();
  }

  updateSetting(key: string, value: string) {
    const normalizedKey = key.trim();
    if (!normalizedKey) {
      throw new Error("설정 키가 비어 있습니다.");
    }

    return this.repo.updateSetting(normalizedKey, value);
  }
}
