"use client";

import { useMemo } from "react";
import { ManageCalendarUseCase } from "@/features/calendar/usecase/ManageCalendar";
import { HttpCalendarRepository } from "@/features/calendar/infrastructure/httpCalendarRepository";
import type {
  CreateEventInput,
  EventPayload,
} from "@/features/calendar/domain/CalendarEntities";

export function useCalendarActions() {
  const useCase = useMemo(() => {
    return new ManageCalendarUseCase(new HttpCalendarRepository());
  }, []);

  return useMemo(
    () => ({
      listCalendars: () => useCase.listCalendars(),
      getSettings: () => useCase.getSettings(),
      listEvents: (start: string, end: string, calendarIDs?: string[]) => useCase.listEvents(start, end, calendarIDs),
      createEvent: (input: CreateEventInput) => useCase.createEvent(input),
      updateEvent: (eventId: string, payload: EventPayload) => useCase.updateEvent(eventId, payload),
      deleteEvent: (eventId: string) => useCase.deleteEvent(eventId),
    }),
    [useCase],
  );
}
