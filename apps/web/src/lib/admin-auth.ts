const TOKEN_KEY = "lifebase_admin_access_token";
const REFRESH_KEY = "lifebase_admin_refresh_token";
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:38117";

export function getAdminAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getAdminRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function setAdminTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_KEY, refreshToken);
}

export function clearAdminTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export function isAdminAuthenticated(): boolean {
  return !!getAdminAccessToken();
}

function getTokenExpiry(token: string): number | null {
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
  const refreshToken = getAdminRefreshToken();
  if (!refreshToken) return null;

  try {
    const res = await fetch(`${API_URL}/api/v1/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) {
      clearAdminTokens();
      return null;
    }
    const data = await res.json();
    setAdminTokens(data.access_token, data.refresh_token);
    return data.access_token;
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

