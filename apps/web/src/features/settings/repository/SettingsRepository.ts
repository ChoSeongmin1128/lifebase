import type { UserSettings } from "@/features/settings/domain/Settings";

export interface SettingsRepository {
  getSettings(): Promise<UserSettings>;
  updateSetting(key: string, value: string): Promise<void>;
}
