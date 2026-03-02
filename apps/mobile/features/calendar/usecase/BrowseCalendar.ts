import type { CalendarRepository } from "../repository/CalendarRepository";

export class BrowseCalendarUseCase {
  constructor(private readonly repo: CalendarRepository) {}

  getSettings() {
    return this.repo.getSettings();
  }

  listEvents(start: string, end: string) {
    return this.repo.listEvents(start, end);
  }
}
