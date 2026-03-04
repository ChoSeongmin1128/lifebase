import { useMemo } from "react";
import { BrowseCalendarUseCase } from "../../usecase/BrowseCalendar";
import { HttpCalendarRepository } from "../../infrastructure/httpCalendarRepository";

export function useCalendarActions() {
  const useCase = useMemo(() => {
    return new BrowseCalendarUseCase(new HttpCalendarRepository());
  }, []);

  return useMemo(
    () => ({
      listCalendars: () => useCase.listCalendars(),
      getSettings: () => useCase.getSettings(),
      listEvents: (start: string, end: string, calendarIDs?: string[]) => useCase.listEvents(start, end, calendarIDs),
      backfillEvents: (start: string, end: string, calendarIDs?: string[]) =>
        useCase.backfillEvents(start, end, calendarIDs),
    }),
    [useCase],
  );
}
