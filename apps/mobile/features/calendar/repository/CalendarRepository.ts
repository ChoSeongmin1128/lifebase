import type { BackfillEventsResult, CalendarData, CalendarEvent, SettingsResponse } from "../domain/CalendarEntities";

export interface CalendarRepository {
  listCalendars(): Promise<CalendarData[]>;
  getSettings(): Promise<SettingsResponse>;
  listEvents(start: string, end: string, calendarIDs?: string[]): Promise<CalendarEvent[]>;
  backfillEvents(start: string, end: string, calendarIDs?: string[]): Promise<BackfillEventsResult>;
}
