import { api } from "../../shared/infrastructure/http-api";
import { getAccessToken } from "../../auth/infrastructure/token-auth";
import type { BackfillEventsResult, CalendarData, CalendarEvent, SettingsResponse } from "../domain/CalendarEntities";
import type { CalendarRepository } from "../repository/CalendarRepository";

interface EventListResponse {
  events?: CalendarEvent[];
}

interface CalendarListResponse {
  calendars?: CalendarData[];
}

export class HttpCalendarRepository implements CalendarRepository {
  private async getToken(): Promise<string> {
    const token = await getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listCalendars(): Promise<CalendarData[]> {
    const token = await this.getToken();
    const data = await api<CalendarListResponse>("/calendars", { token });
    return data.calendars || [];
  }

  async getSettings(): Promise<SettingsResponse> {
    const token = await this.getToken();
    return api<SettingsResponse>("/settings", { token });
  }

  async listEvents(start: string, end: string, calendarIDs?: string[]): Promise<CalendarEvent[]> {
    const token = await this.getToken();
    const params = new URLSearchParams({
      start,
      end,
    });
    if (calendarIDs && calendarIDs.length > 0) {
      params.set("calendar_ids", calendarIDs.join(","));
    }
    const data = await api<EventListResponse>(`/events?${params.toString()}`, { token });
    return data.events || [];
  }

  async backfillEvents(start: string, end: string, calendarIDs?: string[]): Promise<BackfillEventsResult> {
    const token = await this.getToken();
    return api<BackfillEventsResult>("/events/backfill", {
      method: "POST",
      body: {
        start,
        end,
        calendar_ids: calendarIDs || [],
        reason: "range_backfill",
      },
      token,
    });
  }
}
