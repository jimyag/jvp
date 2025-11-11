// API 辅助函数

export async function apiPost<T = any>(
  path: string,
  body: any = {}
): Promise<T> {
  try {
    const response = await fetch(path, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(body),
    });

    // 检查响应状态
    if (!response.ok) {
      const text = await response.text();
      console.error(`API Error [${response.status}]:`, text);
      throw new Error(`API request failed: ${response.status} ${response.statusText}`);
    }

    // 检查是否有内容
    const contentLength = response.headers.get("content-length");
    if (contentLength === "0") {
      return {} as T;
    }

    // 尝试解析 JSON
    const text = await response.text();
    if (!text || text.trim() === "") {
      console.warn("API returned empty response");
      return {} as T;
    }

    try {
      return JSON.parse(text);
    } catch (e) {
      console.error("Failed to parse JSON:", text);
      throw new Error("Invalid JSON response from server");
    }
  } catch (error) {
    if (error instanceof Error) {
      console.error("API request failed:", error.message);
    }
    throw error;
  }
}

export async function handleApiError(error: unknown): Promise<string> {
  if (error instanceof Error) {
    return error.message;
  }
  return "An unknown error occurred";
}
