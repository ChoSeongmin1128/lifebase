import type { BackfillEventsResult, CalendarData, CalendarEvent, SettingsResponse } from "../domain/CalendarEntities";

export interface CalendarRepository {
  listCalendars(): Promise<CalendarData[]>;
  getSettings(): Promise<SettingsResponse>;
  updateSettings(values: Record<string, string>): Promise<void>;
  listEvents(start: string, end: string, calendarIDs?: string[]): Promise<CalendarEvent[]>;
  backfillEvents(start: string, end: string, calendarIDs?: string[]): Promise<BackfillEventsResult>;
}
