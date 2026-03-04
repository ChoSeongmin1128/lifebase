import type {
  BackfillEventsInput,
  BackfillEventsResult,
  CalendarData,
  CalendarSettingsResponse,
  CreateEventInput,
  EventData,
  EventPayload,
} from "@/features/calendar/domain/CalendarEntities";

export interface CalendarRepository {
  listCalendars(): Promise<CalendarData[]>;
  getSettings(): Promise<CalendarSettingsResponse>;
  listEvents(start: string, end: string, calendarIDs?: string[]): Promise<EventData[]>;
  backfillEvents(input: BackfillEventsInput): Promise<BackfillEventsResult>;
  createEvent(input: CreateEventInput): Promise<void>;
  updateEvent(eventId: string, payload: EventPayload): Promise<void>;
  deleteEvent(eventId: string): Promise<void>;
}
