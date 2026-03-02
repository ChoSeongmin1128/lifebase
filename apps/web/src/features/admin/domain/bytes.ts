export type ByteUnit = "B" | "KB" | "MB" | "GB" | "TB";

const UNIT_FACTORS: Record<ByteUnit, number> = {
  B: 1,
  KB: 1024,
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
};

const UNIT_ORDER: ByteUnit[] = ["TB", "GB", "MB", "KB", "B"];

function trimTrailingZeros(value: string): string {
  return value.replace(/\.?0+$/, "");
}

export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes <= 0) return "0 B";

  const abs = Math.abs(bytes);
  const unit = UNIT_ORDER.find((u) => abs >= UNIT_FACTORS[u]) || "B";
  const value = bytes / UNIT_FACTORS[unit];

  if (unit === "B") return `${Math.round(value).toLocaleString()} B`;

  const hasDecimal = Math.abs(value - Math.round(value)) > Number.EPSILON;
  const fixed = hasDecimal ? value.toFixed(value >= 100 ? 0 : value >= 10 ? 1 : 2) : value.toFixed(0);
  return `${trimTrailingZeros(fixed)} ${unit}`;
}

export function splitBytes(bytes: number): { value: string; unit: ByteUnit } {
  if (!Number.isFinite(bytes) || bytes <= 0) {
    return { value: "1", unit: "GB" };
  }

  for (const unit of UNIT_ORDER) {
    const factor = UNIT_FACTORS[unit];
    if (bytes >= factor) {
      const value = bytes / factor;
      const valueStr = Number.isInteger(value) ? String(value) : trimTrailingZeros(value.toFixed(value >= 100 ? 0 : 2));
      return { value: valueStr, unit };
    }
  }

  return { value: String(Math.max(1, Math.round(bytes))), unit: "B" };
}

export function toBytes(value: string, unit: ByteUnit): number | null {
  const n = Number(value);
  if (!Number.isFinite(n) || n <= 0) return null;

  const bytes = n * UNIT_FACTORS[unit];
  if (!Number.isFinite(bytes) || bytes <= 0 || bytes > Number.MAX_SAFE_INTEGER) return null;

  return Math.round(bytes);
}
