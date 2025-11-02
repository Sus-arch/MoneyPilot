const API_URL = import.meta.env.VITE_API_URL;

type Headers = Record<string, string>;

function getToken(): string | null {
  return localStorage.getItem("token");
}

function buildHeaders(extraHeaders: Headers = {}): Headers {
  const token = getToken();
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...extraHeaders,
  };
}

export async function get<T = any>(url: string, headers: Headers = {}): Promise<T> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "GET",
    headers: buildHeaders(headers),
  });
  if (!res.ok) throw new Error(`GET ${url} failed: ${res.status}`);
  return res.json();
}

export async function post<T = any>(url: string, body?: any, headers: Headers = {}): Promise<T> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "POST",
    headers: buildHeaders(headers),
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`POST ${url} failed: ${res.status}`);
  return res.json();
}

export async function del<T = any>(url: string, body?: any, headers: Headers = {}): Promise<T> {
  const res = await fetch(`${API_URL}/api${url}`, {
    method: "DELETE",
    headers: buildHeaders(headers),
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`DELETE ${url} failed: ${res.status}`);
  return res.json();
}
