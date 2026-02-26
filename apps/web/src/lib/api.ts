const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:38117";

import { getValidToken, refreshAccessToken, clearTokens } from "@/lib/auth";

interface ApiOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

/**
 * 인증이 필요한 API 호출.
 * - token을 전달하면 자동 갱신 대상이 된다.
 * - 401 응답 시 refresh → 재시도를 1회 수행한다.
 * - refresh도 실패하면 로그아웃 후 로그인 페이지로 이동한다.
 */
export async function api<T>(path: string, options: ApiOptions = {}): Promise<T> {
  const { method = "GET", body } = options;
  let { token } = options;

  // token이 전달된 경우 만료 임박이면 선제 갱신
  if (token) {
    const valid = await getValidToken();
    if (valid) token = valid;
  }

  const res = await doFetch(path, method, body, token);

  // 401이면 refresh 후 재시도
  if (res.status === 401 && token) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      handleAuthFailure();
      throw new Error("인증이 만료되었습니다. 다시 로그인해 주세요.");
    }
    const retryRes = await doFetch(path, method, body, newToken);
    return parseResponse<T>(retryRes);
  }

  return parseResponse<T>(res);
}

export async function apiUpload<T>(path: string, formData: FormData, token?: string): Promise<T> {
  if (token) {
    const valid = await getValidToken();
    if (valid) token = valid;
  }

  const res = await doUploadFetch(path, formData, token);

  if (res.status === 401 && token) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      handleAuthFailure();
      throw new Error("인증이 만료되었습니다. 다시 로그인해 주세요.");
    }
    const retryRes = await doUploadFetch(path, formData, newToken);
    return parseResponse<T>(retryRes);
  }

  return parseResponse<T>(res);
}

export async function apiDownload(path: string, token?: string): Promise<{ blob: Blob; filename: string }> {
  if (token) {
    const valid = await getValidToken();
    if (valid) token = valid;
  }

  const res = await doDownloadFetch(path, token);

  if (res.status === 401 && token) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      handleAuthFailure();
      throw new Error("인증이 만료되었습니다. 다시 로그인해 주세요.");
    }
    const retryRes = await doDownloadFetch(path, newToken);
    if (!retryRes.ok) throw new Error("Download failed");
    return parseDownloadResponse(retryRes);
  }

  if (!res.ok) throw new Error("Download failed");
  return parseDownloadResponse(res);
}

/* ===== Internal helpers ===== */

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

async function doUploadFetch(path: string, formData: FormData, token?: string): Promise<Response> {
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  return fetch(`${API_URL}/api/v1${path}`, {
    method: "POST",
    headers,
    body: formData,
  });
}

async function doDownloadFetch(path: string, token?: string): Promise<Response> {
  const headers: Record<string, string> = {};
  if (token) headers["Authorization"] = `Bearer ${token}`;

  return fetch(`${API_URL}/api/v1${path}`, {
    method: "GET",
    headers,
  });
}

async function parseResponse<T>(res: Response): Promise<T> {
  if (!res.ok) {
    const error = await res.json().catch(() => ({ error: { code: "UNKNOWN", message: "Unknown error" } }));
    throw new Error(error.error?.message || "Request failed");
  }
  if (res.status === 204) return undefined as T;
  return res.json();
}

async function parseDownloadResponse(res: Response): Promise<{ blob: Blob; filename: string }> {
  const disposition = res.headers.get("Content-Disposition") || "";
  const match = disposition.match(/filename="(.+?)"/);
  const filename = match ? match[1] : "download";
  return { blob: await res.blob(), filename };
}

function handleAuthFailure() {
  clearTokens();
  if (typeof window !== "undefined") {
    window.location.href = "/";
  }
}
