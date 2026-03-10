import { getApiUrl } from "@/features/shared/infrastructure/api-url";

const SESSION_KEY = "lifebase_admin_session";
const SESSION_TOKEN = "__cookie_session__";

export function getAdminAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(SESSION_KEY) === "1" ? SESSION_TOKEN : null;
}

export function getAdminRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(SESSION_KEY) === "1" ? SESSION_TOKEN : null;
}

export function setAdminTokens(_accessToken: string, _refreshToken: string) {
  void _accessToken;
  void _refreshToken;
  localStorage.setItem(SESSION_KEY, "1");
}

export function clearAdminTokens() {
  localStorage.removeItem(SESSION_KEY);
}

export function isAdminAuthenticated(): boolean {
  return !!getAdminAccessToken();
}

function getTokenExpiry(token: string): number | null {
  if (token === SESSION_TOKEN) return null;
  try {
    const payload = token.split(".")[1];
    if (!payload) return null;
    const decoded = JSON.parse(atob(payload));
    return decoded.exp ?? null;
  } catch {
    return null;
  }
}

export function isAdminTokenExpiringSoon(): boolean {
  const token = getAdminAccessToken();
  if (!token) return true;
  if (token === SESSION_TOKEN) return false;
  const exp = getTokenExpiry(token);
  if (!exp) return true;
  const now = Math.floor(Date.now() / 1000);
  return exp - now < 120;
}

let refreshPromise: Promise<string | null> | null = null;

export async function refreshAdminAccessToken(): Promise<string | null> {
  if (refreshPromise) return refreshPromise;
  refreshPromise = doRefresh();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

async function doRefresh(): Promise<string | null> {
  if (!getAdminRefreshToken()) return null;

  try {
    const res = await fetch(getApiUrl("/auth/refresh"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ app: "admin" }),
    });
    if (!res.ok) {
      clearAdminTokens();
      return null;
    }
    await res.json().catch(() => null);
    setAdminTokens("", "");
    return SESSION_TOKEN;
  } catch {
    return null;
  }
}

export async function getValidAdminToken(): Promise<string | null> {
  const token = getAdminAccessToken();
  if (!token) return null;
  if (!isAdminTokenExpiringSoon()) return token;
  return refreshAdminAccessToken();
}

export async function logoutAdmin(): Promise<void> {
  try {
    await fetch(getApiUrl("/auth/logout"), {
      method: "POST",
      credentials: "include",
    });
  } finally {
    clearAdminTokens();
  }
}

export function isAdminSessionMarkerToken(token?: string | null): boolean {
  return token === SESSION_TOKEN;
}
