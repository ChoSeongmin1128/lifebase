import type { CalendarRepository } from "../repository/CalendarRepository";

export class BrowseCalendarUseCase {
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

  backfillEvents(start: string, end: string, calendarIDs?: string[]) {
    return this.repo.backfillEvents(start, end, calendarIDs);
  }
}
