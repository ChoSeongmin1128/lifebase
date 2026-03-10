import {
  getValidToken,
  refreshAccessToken,
  clearTokens,
  isSessionMarkerToken,
} from "@/features/auth/infrastructure/token-auth";
import { getApiUrl } from "@/features/shared/infrastructure/api-url";

interface ApiOptions {
  method?: string;
  body?: unknown;
  token?: string;
}

interface UploadProgressOptions {
  token?: string;
  signal?: AbortSignal;
  onProgress?: (loadedBytes: number, totalBytes: number) => void;
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

export async function apiUpload<T>(path: string, formData: FormData, options: UploadProgressOptions = {}): Promise<T> {
  let { token } = options;

  if (token) {
    const valid = await getValidToken();
    if (valid) token = valid;
  }

  let res = await doUploadRequest(path, formData, token, options);

  if (res.status === 401 && token && !options.signal?.aborted) {
    const newToken = await refreshAccessToken();
    if (!newToken) {
      handleAuthFailure();
      throw new Error("인증이 만료되었습니다. 다시 로그인해 주세요.");
    }
    res = await doUploadRequest(path, formData, newToken, options);
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
  if (token && !isSessionMarkerToken(token)) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return fetch(getApiUrl(path), {
    method,
    headers,
    credentials: "include",
    body: body ? JSON.stringify(body) : undefined,
  });
}

async function doUploadRequest(
  path: string,
  formData: FormData,
  token?: string,
  options: UploadProgressOptions = {},
): Promise<Response> {
  return new Promise<Response>((resolve, reject) => {
    if (options.signal?.aborted) {
      reject(new DOMException("Upload aborted", "AbortError"));
      return;
    }

    const xhr = new XMLHttpRequest();
    xhr.open("POST", getApiUrl(path));
    xhr.withCredentials = true;
    if (token && !isSessionMarkerToken(token)) {
      xhr.setRequestHeader("Authorization", `Bearer ${token}`);
    }

    const abortUpload = () => {
      xhr.abort();
    };

    options.signal?.addEventListener("abort", abortUpload, { once: true });

    xhr.upload.onprogress = (event) => {
      if (!event.lengthComputable) return;
      options.onProgress?.(event.loaded, event.total);
    };

    xhr.onerror = () => {
      cleanup();
      reject(new Error("Upload failed"));
    };

    xhr.onabort = () => {
      cleanup();
      reject(new DOMException("Upload aborted", "AbortError"));
    };

    xhr.onload = () => {
      cleanup();
      const responseHeaders = new Headers();
      xhr.getAllResponseHeaders()
        .trim()
        .split(/[\r\n]+/)
        .forEach((line) => {
          if (!line) return;
          const parts = line.split(": ");
          const header = parts.shift();
          if (!header) return;
          responseHeaders.append(header, parts.join(": "));
        });
      resolve(new Response(xhr.responseText, {
        status: xhr.status,
        statusText: xhr.statusText,
        headers: responseHeaders,
      }));
    };

    const cleanup = () => {
      options.signal?.removeEventListener("abort", abortUpload);
    };

    xhr.send(formData);
  });
}

async function doDownloadFetch(path: string, token?: string): Promise<Response> {
  const headers: Record<string, string> = {};
  if (token && !isSessionMarkerToken(token)) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  return fetch(getApiUrl(path), {
    method: "GET",
    headers,
    credentials: "include",
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
  const filename = parseFilenameFromDisposition(disposition) || "download";
  return { blob: await res.blob(), filename };
}

function parseFilenameFromDisposition(disposition: string): string | null {
  // RFC 5987: filename*=UTF-8''encoded-name
  const encodedMatch = disposition.match(/filename\*\s*=\s*UTF-8''([^;]+)/i);
  if (encodedMatch && encodedMatch[1]) {
    try {
      return decodeURIComponent(encodedMatch[1].trim().replace(/^"|"$/g, ""));
    } catch {
      // fallthrough
    }
  }

  // filename="name.ext"
  const quotedMatch = disposition.match(/filename\s*=\s*"([^"]+)"/i);
  if (quotedMatch && quotedMatch[1]) {
    return quotedMatch[1];
  }

  // filename=name.ext
  const plainMatch = disposition.match(/filename\s*=\s*([^;]+)/i);
  if (plainMatch && plainMatch[1]) {
    return plainMatch[1].trim().replace(/^"|"$/g, "");
  }

  return null;
}

function handleAuthFailure() {
  clearTokens();
  if (typeof window !== "undefined") {
    window.location.href = "/";
  }
}
