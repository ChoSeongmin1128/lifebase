import type {
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
  createEvent(input: CreateEventInput): Promise<void>;
  updateEvent(eventId: string, payload: EventPayload): Promise<void>;
  deleteEvent(eventId: string): Promise<void>;
}
