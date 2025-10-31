const API_URL = import.meta.env.VITE_API_URL;

// Получаем токен JWT из localStorage (или другого хранилища)
function getToken() {
  return localStorage.getItem("token") || "";
}

// Общий метод fetch
async function request(path, options = {}) {
  const headers = {
    "Content-Type": "application/json",
    Authorization: `Bearer ${getToken()}`,
    ...options.headers,
  };

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || "API request failed");
  }

  return res.json();
}

// GET
export async function get(path) {
  return request(path, { method: "GET" });
}

// POST
export async function post(path, data) {
  return request(path, { method: "POST", body: JSON.stringify(data) });
}
