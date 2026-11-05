import { clearAuth } from "@/lib/auth";

/** 后端统一响应结构 */
export interface R<T = unknown> {
  code: string;
  message: string;
  data: T;
  success: boolean;
  requestId: string;
}

export class ApiError extends Error {
  code: string;
  constructor(code: string, message: string) {
    super(message);
    this.code = code;
  }
}

const UNAUTHORIZED_CODES = new Set(["A0231"]);

export async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string> | undefined),
  };

  const resp = await fetch(path, {
    ...options,
    headers,
    credentials: "same-origin", // 登录凭证在 HttpOnly Cookie 中，随同源请求自动携带
  });
  const body = (await resp.json().catch(() => null)) as R<T> | null;

  if (!body) {
    throw new ApiError("B0001", "响应解析失败");
  }
  if (!body.success) {
    if (UNAUTHORIZED_CODES.has(body.code)) {
      clearAuth();
      window.location.href = "/login";
    }
    throw new ApiError(body.code, body.message);
  }
  return body.data;
}

export const http = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: "POST", body: JSON.stringify(data ?? {}) }),
  put: <T>(path: string, data?: unknown) =>
    request<T>(path, { method: "PUT", body: JSON.stringify(data ?? {}) }),
  delete: <T>(path: string) => request<T>(path, { method: "DELETE" }),
};
