import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { CalendarEvent, SettingsResponse } from "../domain/CalendarEntities";
import type { CalendarRepository } from "../repository/CalendarRepository";

interface EventListResponse {
  events?: CalendarEvent[];
}

export class HttpCalendarRepository implements CalendarRepository {
  private async getToken(): Promise<string> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async getSettings(): Promise<SettingsResponse> {
    const token = await this.getToken();
    return api<SettingsResponse>("/settings", { token });
  }

  async listEvents(start: string, end: string): Promise<CalendarEvent[]> {
    const token = await this.getToken();
    const data = await api<EventListResponse>(
      `/events?start=${encodeURIComponent(start)}&end=${encodeURIComponent(end)}`,
      { token },
    );
    return data.events || [];
  }
}
