// API 辅助函数：所有后端接口都是 POST + JSON，错误响应为 apierror.ErrorResponse

export class ApiError extends Error {
  status: number;
  code?: string;
  requestId?: string;

  constructor(message: string, status: number, code?: string, requestId?: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }
}

interface ErrorResponseBody {
  errors?: { code?: string; message?: string }[];
  requestID?: string;
  message?: string;
  error?: string;
}

function parseError(status: number, statusText: string, text: string): ApiError {
  try {
    const body = JSON.parse(text) as ErrorResponseBody;
    const first = body.errors?.[0];
    const message = first?.message || body.message || body.error;
    if (message) {
      return new ApiError(message, status, first?.code, body.requestID);
    }
  } catch {
    // 非 JSON 响应，使用默认错误信息
  }
  return new ApiError(text.trim() || `Request failed: ${status} ${statusText}`, status);
}

export async function api<T = unknown>(path: string, body: unknown = {}): Promise<T> {
  let response: Response;
  try {
    response = await fetch(path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    throw new ApiError("Unable to reach the JVP server. Check that the backend is running.", 0);
  }

  const text = await response.text();
  if (!response.ok) {
    throw parseError(response.status, response.statusText, text);
  }
  if (!text.trim()) {
    return {} as T;
  }
  try {
    return JSON.parse(text) as T;
  } catch {
    throw new ApiError("Invalid JSON response from server", response.status);
  }
}

export function errorMessage(error: unknown, fallback = "Something went wrong"): string {
  if (error instanceof Error && error.message) return error.message;
  if (typeof error === "string" && error) return error;
  return fallback;
}
