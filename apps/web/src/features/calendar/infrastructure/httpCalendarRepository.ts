import { api } from "@/features/shared/infrastructure/http-api";
import { getAccessToken } from "@/features/auth/infrastructure/token-auth";
import type {
  BackfillEventsInput,
  BackfillEventsResult,
  CalendarData,
  CalendarSettingsResponse,
  CreateEventInput,
  EventData,
  EventPayload,
} from "@/features/calendar/domain/CalendarEntities";
import type { CalendarRepository } from "@/features/calendar/repository/CalendarRepository";

interface CalendarsResponse {
  calendars?: CalendarData[];
}

interface EventsResponse {
  events?: EventData[];
}

export class HttpCalendarRepository implements CalendarRepository {
  private getToken(): string {
    const token = getAccessToken();
    if (!token) {
      throw new Error("인증이 필요합니다.");
    }
    return token;
  }

  async listCalendars(): Promise<CalendarData[]> {
    const token = this.getToken();
    const data = await api<CalendarsResponse>("/calendars", { token });
    return data.calendars || [];
  }

  getSettings(): Promise<CalendarSettingsResponse> {
    const token = this.getToken();
    return api<CalendarSettingsResponse>("/settings", { token });
  }

  async listEvents(start: string, end: string, calendarIDs?: string[]): Promise<EventData[]> {
    const token = this.getToken();
    const params = new URLSearchParams({
      start,
      end,
    });
    if (calendarIDs && calendarIDs.length > 0) {
      params.set("calendar_ids", calendarIDs.join(","));
    }
    const data = await api<EventsResponse>(`/events?${params.toString()}`, { token });
    return data.events || [];
  }

  backfillEvents(input: BackfillEventsInput): Promise<BackfillEventsResult> {
    const token = this.getToken();
    return api<BackfillEventsResult>("/events/backfill", {
      method: "POST",
      body: {
        start: input.start,
        end: input.end,
        calendar_ids: input.calendar_ids || [],
        reason: input.reason || "range_backfill",
      },
      token,
    });
  }

  async createEvent(input: CreateEventInput): Promise<void> {
    const token = this.getToken();
    await api("/events", {
      method: "POST",
      body: input,
      token,
    });
  }

  async updateEvent(eventId: string, payload: EventPayload): Promise<void> {
    const token = this.getToken();
    await api(`/events/${eventId}`, {
      method: "PATCH",
      body: payload,
      token,
    });
  }

  async deleteEvent(eventId: string): Promise<void> {
    const token = this.getToken();
    await api(`/events/${eventId}`, {
      method: "DELETE",
      token,
    });
  }
}
