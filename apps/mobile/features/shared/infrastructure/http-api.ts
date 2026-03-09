import Constants from "expo-constants";
import { clearTokens, refreshAccessToken } from "../../auth/infrastructure/token-auth";

const API_BASE =
  (Constants.expoConfig?.extra?.apiUrl as string) ||
  "https://lifebase.cc/api/v1";

export async function api<T = unknown>(
  path: string,
  opts: { method?: string; body?: unknown; token?: string } = {}
): Promise<T> {
  const { method = "GET", body } = opts;
  let { token } = opts;

  const res = await doFetch(path, method, body, token);

  if (res.status === 401 && token) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      await clearTokens();
      throw new Error("인증이 만료되었습니다. 다시 로그인해 주세요.");
    }

    const retryRes = await doFetch(path, method, body, newToken);
    return parseResponse<T>(retryRes);
  }

  return parseResponse<T>(res);
}

async function doFetch(path: string, method: string, body: unknown, token?: string): Promise<Response> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (token) headers["Authorization"] = `Bearer ${token}`;

  return fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
}

async function parseResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`API ${res.status}: ${text}`);
  }

  if (res.status === 204) {
    return {} as T;
  }

  const text = await res.text();
  if (!text) {
    return {} as T;
  }

  return JSON.parse(text) as T;
}
