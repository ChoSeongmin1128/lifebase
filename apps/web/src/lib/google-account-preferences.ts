export const MULTI_ACCOUNT_FALLBACK_COLORS = [
  "#2563eb",
  "#16a34a",
  "#dc2626",
  "#9333ea",
  "#ea580c",
  "#0891b2",
  "#ca8a04",
  "#0f766e",
];

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
