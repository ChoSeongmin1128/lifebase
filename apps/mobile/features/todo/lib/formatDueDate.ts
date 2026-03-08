function normalizeDueDate(value: string | null | undefined): string {
  if (!value) return "";

  const candidate = value.includes("T") ? value.slice(0, 10) : value;
  if (/^\d{4}-\d{2}-\d{2}$/.test(candidate)) {
    return candidate;
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "";
  const year = String(parsed.getFullYear()).padStart(4, "0");
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function normalizeDueTime(value: string | null | undefined): string {
  if (!value) return "";
  if (/^\d{2}:\d{2}/.test(value)) return value.slice(0, 5);
  const match = value.match(/T(\d{2}:\d{2})/);
  return match ? match[1] : "";
}

export function formatDueYYMMDD(value: string | null | undefined): string {
  const date = normalizeDueDate(value);
  if (!date) return "";
  return `${date.slice(2, 4)}.${date.slice(5, 7)}.${date.slice(8, 10)}`;
}

export function formatDueLabel(dueDate: string | null | undefined, dueTime?: string | null | undefined): string {
  const date = normalizeDueDate(dueDate);
  if (!date) return "";
  const time = normalizeDueTime(dueTime);
  const dateLabel = `${date.slice(2, 4)}.${date.slice(5, 7)}.${date.slice(8, 10)}`;
  return time ? `${dateLabel} ${time}` : dateLabel;
}
