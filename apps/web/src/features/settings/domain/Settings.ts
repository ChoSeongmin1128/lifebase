export type UserSettings = Record<string, string>;

export interface SettingsResponse {
  settings: UserSettings;
}
