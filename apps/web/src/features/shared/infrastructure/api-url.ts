const API_PREFIX = "/api/v1";

function normalizeApiOrigin(value?: string): string {
  if (!value) return "";

  const trimmed = value.trim().replace(/\/+$/, "");
  if (!trimmed) return "";

  return trimmed.replace(/\/api\/v1$/i, "");
}

export function getConfiguredApiOrigin(): string {
  return normalizeApiOrigin(process.env.NEXT_PUBLIC_API_URL);
}

export function getApiPath(path: string): string {
  const normalizedPath = path.startsWith("/") ? path : `/${path}`;
  return `${API_PREFIX}${normalizedPath}`;
}

export function getApiUrl(path: string): string {
  return `${getConfiguredApiOrigin()}${getApiPath(path)}`;
}
