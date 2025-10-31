const API_URL = import.meta.env.VITE_API_URL;

// Функция для получения токена из localStorage
function getToken(): string | null {
  return localStorage.getItem("token");
}

// Тип для заголовков
type Headers = Record<string, string>;

// GET запрос с JWT
export async function get<T = any>(url: string, headers: Headers = {}): Promise<T> {
  const token = getToken();
  const res = await fetch(`${API_URL}${url}`, {
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
  });

  if (!res.ok) {
    throw new Error(`GET ${url} failed with status ${res.status}`);
  }

  return res.json();
}

// POST запрос с JWT
export async function post<T = any>(url: string, body?: any, headers: Headers = {}): Promise<T> {
  const token = getToken();
  const res = await fetch(`${API_URL}${url}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    body: JSON.stringify(body),
  });

  if (!res.ok) {
    throw new Error(`POST ${url} failed with status ${res.status}`);
  }

  return res.json();
}
