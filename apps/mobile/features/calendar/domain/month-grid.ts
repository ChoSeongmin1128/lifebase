export interface MonthCell {
  date: Date;
  dateKey: string;
  day: number;
  inCurrentMonth: boolean;
  weekIndex: number;
  dayIndex: number;
}

const FIXED_GRID_CELL_COUNT = 42;

function normalizeWeekStartsOn(value: number): number {
  if (!Number.isInteger(value)) return 0;
  const normalized = value % 7;
  return normalized < 0 ? normalized + 7 : normalized;
}

function toDateKey(date: Date): string {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, "0");
  const d = String(date.getDate()).padStart(2, "0");
  return `${y}-${m}-${d}`;
}

function startOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 0, 0, 0, 0);
}

function endOfDay(date: Date): Date {
  return new Date(date.getFullYear(), date.getMonth(), date.getDate(), 23, 59, 59, 999);
}

export function buildFixedMonthGrid(baseDate: Date): MonthCell[] {
  return buildFixedMonthGridWithWeekStart(baseDate, 0);
}

export function buildFixedMonthGridWithWeekStart(baseDate: Date, weekStartsOn: number): MonthCell[] {
  const year = baseDate.getFullYear();
  const month = baseDate.getMonth();
  const startsOn = normalizeWeekStartsOn(weekStartsOn);

  const firstOfMonth = new Date(year, month, 1);
  const daysInMonth = new Date(year, month + 1, 0).getDate();
  const offset = (firstOfMonth.getDay() - startsOn + 7) % 7;
  const start = new Date(firstOfMonth);
  start.setDate(firstOfMonth.getDate() - offset);
  // Keep one leading outside week when month fits exactly 4 rows.
  if (offset + daysInMonth === 28) {
    start.setDate(start.getDate() - 7);
  }

  return Array.from({ length: FIXED_GRID_CELL_COUNT }, (_, index) => {
    const date = new Date(start);
    date.setDate(start.getDate() + index);

    return {
      date,
      dateKey: toDateKey(date),
      day: date.getDate(),
      inCurrentMonth: date.getFullYear() === year && date.getMonth() === month,
      weekIndex: Math.floor(index / 7),
      dayIndex: index % 7,
    };
  });
}

export function getFixedMonthFetchRange(baseDate: Date): { start: string; end: string } {
  return getFixedMonthFetchRangeWithWeekStart(baseDate, 0);
}

export function getFixedMonthFetchRangeWithWeekStart(baseDate: Date, weekStartsOn: number): { start: string; end: string } {
  const cells = buildFixedMonthGridWithWeekStart(baseDate, weekStartsOn);
  const start = startOfDay(cells[0].date);
  const end = endOfDay(cells[cells.length - 1].date);
  return { start: start.toISOString(), end: end.toISOString() };
}
