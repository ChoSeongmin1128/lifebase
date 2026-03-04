import type {
  BackfillEventsInput,
  CreateEventInput,
  DaySummaryData,
  EventPayload,
  HolidayData,
} from "@/features/calendar/domain/CalendarEntities";
import type { CalendarRepository } from "@/features/calendar/repository/CalendarRepository";

export class ManageCalendarUseCase {
  constructor(private readonly repo: CalendarRepository) {}

  listCalendars() {
    return this.repo.listCalendars();
  }

  getSettings() {
    return this.repo.getSettings();
  }

  listEvents(start: string, end: string, calendarIDs?: string[]) {
    return this.repo.listEvents(start, end, calendarIDs);
  }

  listHolidays(startDate: string, endDate: string): Promise<HolidayData[]> {
    return this.repo.listHolidays(startDate, endDate);
  }

  getDaySummary(
    date: string,
    timezone: string,
    calendarIDs?: string[],
    includeDoneTodos: boolean = false
  ): Promise<DaySummaryData> {
    return this.repo.getDaySummary(date, timezone, calendarIDs, includeDoneTodos);
  }

  backfillEvents(input: BackfillEventsInput) {
    return this.repo.backfillEvents(input);
  }

  createEvent(input: CreateEventInput) {
    if (!input.title.trim()) {
      throw new Error("일정 제목이 비어 있습니다.");
    }
    return this.repo.createEvent({
      ...input,
      title: input.title.trim(),
    });
  }

  updateEvent(eventId: string, payload: EventPayload) {
    if (!payload.title.trim()) {
      throw new Error("일정 제목이 비어 있습니다.");
    }
    return this.repo.updateEvent(eventId, {
      ...payload,
      title: payload.title.trim(),
    });
  }

  deleteEvent(eventId: string) {
    return this.repo.deleteEvent(eventId);
  }
}
