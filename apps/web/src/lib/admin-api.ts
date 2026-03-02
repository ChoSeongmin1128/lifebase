const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:38117";

import { clearAdminTokens, getValidAdminToken, refreshAdminAccessToken } from "@/lib/admin-auth";

interface ApiOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

export async function adminApi<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const { method = "GET", body } = options;
  let { token } = options;

  if (token) {
    const valid = await getValidAdminToken();
    if (valid) token = valid;
  }

  const res = await doFetch(path, method, body, token);
  if (res.status === 401 && token) {
    const newToken = await refreshAdminAccessToken();
    if (!newToken) {
      handleAuthFailure();
      throw new Error("관리자 인증이 만료되었습니다.");
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

  return fetch(`${API_URL}/api/v1${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  });
}

async function parseResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const error = await res
      .json()
      .catch(() => ({ error: { code: "UNKNOWN", message: "Unknown error" } }));
    throw new Error(error.error?.message || "Request failed");
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

function handleAuthFailure() {
  clearAdminTokens();
  if (typeof window !== "undefined") {
    window.location.href = "/admin/login";
  }
}

