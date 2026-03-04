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
      updateSettings: (values: Record<string, string>) => useCase.updateSettings(values),
      listEvents: (start: string, end: string, calendarIDs?: string[]) => useCase.listEvents(start, end, calendarIDs),
      getDaySummary: (date: string, timezone: string, calendarIDs?: string[], includeDoneTodos: boolean = false) =>
        useCase.getDaySummary(date, timezone, calendarIDs, includeDoneTodos),
      backfillEvents: (start: string, end: string, calendarIDs?: string[]) =>
        useCase.backfillEvents(start, end, calendarIDs),
    }),
    [useCase],
  );
}
