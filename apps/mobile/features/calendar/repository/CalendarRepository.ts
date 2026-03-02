import type { CalendarEvent, SettingsResponse } from "../domain/CalendarEntities";

export interface CalendarRepository {
  getSettings(): Promise<SettingsResponse>;
  listEvents(start: string, end: string): Promise<CalendarEvent[]>;
}
