import type { BackfillEventsResult, CalendarData, CalendarEvent, DaySummaryData, SettingsResponse } from "../domain/CalendarEntities";

export interface CalendarRepository {
  listCalendars(): Promise<CalendarData[]>;
  getSettings(): Promise<SettingsResponse>;
  updateSettings(values: Record<string, string>): Promise<void>;
  listEvents(start: string, end: string, calendarIDs?: string[]): Promise<CalendarEvent[]>;
  getDaySummary(date: string, timezone: string, calendarIDs?: string[], includeDoneTodos?: boolean): Promise<DaySummaryData>;
  backfillEvents(start: string, end: string, calendarIDs?: string[]): Promise<BackfillEventsResult>;
}
