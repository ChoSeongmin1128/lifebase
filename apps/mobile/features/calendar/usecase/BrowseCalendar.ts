import type { CalendarRepository } from "../repository/CalendarRepository";

export class BrowseCalendarUseCase {
  constructor(private readonly repo: CalendarRepository) {}

  listCalendars() {
    return this.repo.listCalendars();
  }

  getSettings() {
    return this.repo.getSettings();
  }

  updateSettings(values: Record<string, string>) {
    return this.repo.updateSettings(values);
  }

  listEvents(start: string, end: string, calendarIDs?: string[]) {
    return this.repo.listEvents(start, end, calendarIDs);
  }

  getDaySummary(date: string, timezone: string, calendarIDs?: string[], includeDoneTodos: boolean = false) {
    return this.repo.getDaySummary(date, timezone, calendarIDs, includeDoneTodos);
  }

  backfillEvents(start: string, end: string, calendarIDs?: string[]) {
    return this.repo.backfillEvents(start, end, calendarIDs);
  }
}
