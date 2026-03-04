import type {
  BackfillEventsInput,
  BackfillEventsResult,
  CalendarData,
  CalendarSettingsResponse,
  CreateEventInput,
  DaySummaryData,
  EventData,
  EventPayload,
  HolidayData,
} from "@/features/calendar/domain/CalendarEntities";

export interface CalendarRepository {
  listCalendars(): Promise<CalendarData[]>;
  getSettings(): Promise<CalendarSettingsResponse>;
  listEvents(start: string, end: string, calendarIDs?: string[]): Promise<EventData[]>;
  listHolidays(startDate: string, endDate: string): Promise<HolidayData[]>;
  getDaySummary(date: string, timezone: string, calendarIDs?: string[], includeDoneTodos?: boolean): Promise<DaySummaryData>;
  backfillEvents(input: BackfillEventsInput): Promise<BackfillEventsResult>;
  createEvent(input: CreateEventInput): Promise<void>;
  updateEvent(eventId: string, payload: EventPayload): Promise<void>;
  deleteEvent(eventId: string): Promise<void>;
}
