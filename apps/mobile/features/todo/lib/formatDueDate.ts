export function formatDueYYMMDD(value: string | null | undefined): string {
  if (!value) return "";

  const candidate = value.includes("T") ? value.slice(0, 10) : value;
  if (/^\d{4}-\d{2}-\d{2}$/.test(candidate)) {
    return `${candidate.slice(2, 4)}.${candidate.slice(5, 7)}.${candidate.slice(8, 10)}`;
  }

  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return "";
  const year = String(parsed.getFullYear() % 100).padStart(2, "0");
  const month = String(parsed.getMonth() + 1).padStart(2, "0");
  const day = String(parsed.getDate()).padStart(2, "0");
  return `${year}.${month}.${day}`;
}
