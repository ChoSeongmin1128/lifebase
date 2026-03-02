import { useMemo } from "react";
import { BrowseCalendarUseCase } from "../../usecase/BrowseCalendar";
import { HttpCalendarRepository } from "../../infrastructure/httpCalendarRepository";

export function useCalendarActions() {
  const useCase = useMemo(() => {
    return new BrowseCalendarUseCase(new HttpCalendarRepository());
  }, []);

  return useMemo(
    () => ({
      getSettings: () => useCase.getSettings(),
      listEvents: (start: string, end: string) => useCase.listEvents(start, end),
    }),
    [useCase],
  );
}
