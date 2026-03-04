type EventDateLike = {
  start_time: string;
  end_time: string;
  timezone?: string | null;
  is_all_day?: boolean;
};

function formatDateKey(date: Date, timeZone?: string | null): string {
  if (Number.isNaN(date.getTime())) return "";

  const formatter = new Intl.DateTimeFormat("en-CA", {
    timeZone: timeZone || "Asia/Seoul",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  });
  const parts = formatter.formatToParts(date);
  const year = parts.find((part) => part.type === "year")?.value ?? "";
  const month = parts.find((part) => part.type === "month")?.value ?? "";
  const day = parts.find((part) => part.type === "day")?.value ?? "";
  if (!year || !month || !day) return "";
  return `${year}-${month}-${day}`;
}

function parseISO(value: string): Date | null {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return null;
  return parsed;
}

function isUtcMidnight(date: Date): boolean {
  return (
    date.getUTCHours() === 0 &&
    date.getUTCMinutes() === 0 &&
    date.getUTCSeconds() === 0 &&
    date.getUTCMilliseconds() === 0
  );
}

export function getEventStartDateKey(event: EventDateLike): string {
  const parsed = parseISO(event.start_time);
  if (!parsed) return "";
  return formatDateKey(parsed, event.timezone);
}

export function getEventEndDateKey(event: EventDateLike): string {
  const parsed = parseISO(event.end_time);
  if (!parsed) return "";

  // Backward compatibility:
  // Legacy synced all-day events may store exclusive end at UTC midnight.
  // Convert to inclusive date key for rendering.
  if (event.is_all_day && isUtcMidnight(parsed)) {
    const inclusive = new Date(parsed.getTime() - 1);
    return formatDateKey(inclusive, event.timezone);
  }

  return formatDateKey(parsed, event.timezone);
}

