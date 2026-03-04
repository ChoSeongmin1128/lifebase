export const ACCOUNT_COLOR_PALETTE = [
  "#1b998b",
  "#14b8a6",
  "#0ea5e9",
  "#3b82f6",
  "#6366f1",
  "#8b5cf6",
  "#a855f7",
  "#ec4899",
  "#f97316",
  "#f59e0b",
  "#84cc16",
  "#64748b",
];

export const MULTI_ACCOUNT_FALLBACK_COLORS = ACCOUNT_COLOR_PALETTE.slice(0, 8);

export function buildGoogleAccountAliasSettingKey(accountID: string): string {
  return `google_account_alias_${accountID}`;
}

export function buildGoogleAccountColorSettingKey(accountID: string): string {
  return `calendar_account_color_${accountID}`;
}

export function normalizeHexColor(value: string | null | undefined): string | null {
  if (!value) return null;
  const trimmed = value.trim();
  if (/^#[0-9a-fA-F]{6}$/.test(trimmed)) {
    return trimmed.toLowerCase();
  }
  return null;
}

export function getGoogleAccountAlias(settings: Record<string, string>, accountID: string | null | undefined): string {
  if (!accountID) return "";
  return (settings[buildGoogleAccountAliasSettingKey(accountID)] || "").trim();
}

export function getGoogleAccountDisplayName(
  settings: Record<string, string>,
  accountID: string | null | undefined,
  email: string | null | undefined,
  fallback = "계정 미확인",
): string {
  const alias = getGoogleAccountAlias(settings, accountID);
  if (alias) return alias;
  const normalizedEmail = (email || "").trim();
  if (normalizedEmail) return normalizedEmail;
  return fallback;
}

export function getGoogleAccountCustomColor(
  settings: Record<string, string>,
  accountID: string | null | undefined,
): string | null {
  if (!accountID) return null;
  return normalizeHexColor(settings[buildGoogleAccountColorSettingKey(accountID)]);
}

export function isPresetAccountColor(value: string | null | undefined): boolean {
  const normalized = normalizeHexColor(value);
  if (!normalized) return false;
  return ACCOUNT_COLOR_PALETTE.includes(normalized);
}
