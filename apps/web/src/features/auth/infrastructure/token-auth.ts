import { getApiUrl } from "@/features/shared/infrastructure/api-url";

const SESSION_KEY = "lifebase_web_session";
const SESSION_TOKEN = "__cookie_session__";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(SESSION_KEY) === "1" ? SESSION_TOKEN : null;
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(SESSION_KEY) === "1" ? SESSION_TOKEN : null;
}

export function setTokens(_accessToken: string, _refreshToken: string) {
  void _accessToken;
  void _refreshToken;
  localStorage.setItem(SESSION_KEY, "1");
}

export function clearTokens() {
  localStorage.removeItem(SESSION_KEY);
}

export function isAuthenticated(): boolean {
  return !!getAccessToken();
}

// JWT 만료 시각 추출 (seconds since epoch)
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

// 토큰이 곧 만료되는지 확인 (2분 이내)
export function isTokenExpiringSoon(): boolean {
  const token = getAccessToken();
  if (!token) return true;
  if (token === SESSION_TOKEN) return false;
  const exp = getTokenExpiry(token);
  if (!exp) return true;
  const now = Math.floor(Date.now() / 1000);
  return exp - now < 120; // 2분 이내면 만료 임박
}

// 동시 refresh 호출 방지를 위한 진행 중 Promise
let refreshPromise: Promise<string | null> | null = null;

/**
 * Access token을 갱신한다.
 * 동시 호출 시 하나의 요청만 실행하고 나머지는 결과를 공유한다.
 * 실패 시 null을 반환하며, 호출자가 로그아웃 처리해야 한다.
 */
export async function refreshAccessToken(): Promise<string | null> {
  // 이미 진행 중이면 같은 Promise 반환 (중복 호출 방지)
  if (refreshPromise) return refreshPromise;

  refreshPromise = doRefresh();
  try {
    return await refreshPromise;
  } finally {
    refreshPromise = null;
  }
}

async function doRefresh(): Promise<string | null> {
  if (!getRefreshToken()) return null;

  try {
    const res = await fetch(getApiUrl("/auth/refresh"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ app: "web" }),
    });

    if (!res.ok) {
      // refresh token 만료 또는 무효 → 토큰 삭제
      clearTokens();
      return null;
    }

    await res.json().catch(() => null);
    setTokens("", "");
    return SESSION_TOKEN;
  } catch {
    return null;
  }
}

/**
 * 유효한 access token을 반환한다.
 * 만료 임박 시 자동으로 갱신을 시도한다.
 * 갱신 실패 시 null을 반환한다.
 */
export async function getValidToken(): Promise<string | null> {
  const token = getAccessToken();
  if (!token) return null;

  if (!isTokenExpiringSoon()) return token;
  return refreshAccessToken();
}

export async function logout(): Promise<void> {
  try {
    await fetch(getApiUrl("/auth/logout"), {
      method: "POST",
      credentials: "include",
    });
  } finally {
    clearTokens();
  }
}

export function isSessionMarkerToken(token?: string | null): boolean {
  return token === SESSION_TOKEN;
}
