const API_URL = import.meta.env.VITE_API_URL;

// Функция для получения токена из localStorage
function getToken(): string | null {
  return localStorage.getItem("token");
}

// Тип для заголовков
type Headers = Record<string, string>;

// Универсальная функция для сборки заголовков
function buildHeaders(extraHeaders: Headers = {}): Headers {
  const token = getToken();
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...extraHeaders,
  };
}

// ✅ GET запрос с JWT и опциональным X-Bank-Code
export async function get<T = any>(url: string, headers: Headers = {}): Promise<T> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "GET",
    headers: buildHeaders(headers),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`GET ${url} failed: ${res.status} ${text}`);
  }

  return res.json();
}

// ✅ POST запрос с JWT и опциональным X-Bank-Code
export async function post<T = any>(
  url: string,
  body?: any,
  headers: Headers = {}
): Promise<T> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "POST",
    headers: buildHeaders(headers),
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`POST ${url} failed: ${res.status} ${text}`);
  }

  return res.json();
}

// ✅ DELETE запрос с JWT и опциональным X-Bank-Code
export async function del(
  url: string,
  body?: any,
  headers: Headers = {}
): Promise<any> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "DELETE",
    headers: buildHeaders(headers),
    body: body ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    const text = await res.text();
    throw new Error(`DELETE ${url} failed: ${res.status} ${text}`);
  }

  return res.json();
}
