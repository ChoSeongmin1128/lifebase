import type {
  CreateEventInput,
  EventPayload,
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

  listEvents(start: string, end: string) {
    return this.repo.listEvents(start, end);
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
