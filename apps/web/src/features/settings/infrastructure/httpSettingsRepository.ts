import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type { SettingsResponse } from "@/features/settings/domain/Settings";
import type { SettingsRepository } from "@/features/settings/repository/SettingsRepository";

export class HttpSettingsRepository implements SettingsRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async getSettings() {
    const token = this.getToken();
    const data = await api<SettingsResponse>("/settings", { token });
    return data.settings || {};
  }

  async updateSetting(key: string, value: string): Promise<void> {
    const token = this.getToken();
    await api("/settings", {
      method: "PATCH",
      body: { [key]: value },
      token,
    });
  }
}
