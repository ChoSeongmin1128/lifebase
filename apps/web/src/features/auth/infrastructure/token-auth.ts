import { getApiUrl } from "@/features/shared/infrastructure/api-url";

const TOKEN_KEY = "lifebase_access_token";
const REFRESH_KEY = "lifebase_refresh_token";

export function getAccessToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem(REFRESH_KEY);
}

export function setTokens(accessToken: string, refreshToken: string) {
  localStorage.setItem(TOKEN_KEY, accessToken);
  localStorage.setItem(REFRESH_KEY, refreshToken);
}

export function clearTokens() {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

export function isAuthenticated(): boolean {
  return !!getAccessToken();
}

// JWT 만료 시각 추출 (seconds since epoch)
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

// 토큰이 곧 만료되는지 확인 (2분 이내)
export function isTokenExpiringSoon(): boolean {
  const token = getAccessToken();
  if (!token) return true;
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
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;

  try {
    const res = await fetch(getApiUrl("/auth/refresh"), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) {
      // refresh token 만료 또는 무효 → 토큰 삭제
      clearTokens();
      return null;
    }

    const data = await res.json();
    setTokens(data.access_token, data.refresh_token);
    return data.access_token;
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

  // 선제적 갱신
  return refreshAccessToken();
}
