// @ts-ignore - Vite types are available but TypeScript may not recognize them
const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080";

type Headers = Record<string, string>;

// Кэш для GET запросов
interface CacheEntry {
  data: any;
  timestamp: number;
  expiresAt: number;
}

const cache = new Map<string, CacheEntry>();
const CACHE_TTL = 30000; // 30 секунд для большинства запросов
const CACHE_TTL_LONG = 60000; // 60 секунд для счетов и продуктов

// Активные запросы для предотвращения дублирования
const activeRequests = new Map<string, Promise<any>>();

// Debounce таймеры
const debounceTimers = new Map<string, ReturnType<typeof setTimeout>>();

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

function getCacheKey(url: string, headers: Headers): string {
  const authHeader = headers.Authorization || "";
  return `${url}_${authHeader}`;
}

function getCacheTTL(url: string): number {
  // Длинный кэш для счетов и продуктов
  if (url.includes("/accounts") || url.includes("/products")) {
    return CACHE_TTL_LONG;
  }
  return CACHE_TTL;
}

async function fetchWithRetry(
  url: string,
  options: RequestInit,
  retries = 3,
  delay = 1000
): Promise<Response> {
  try {
    const res = await fetch(url, options);
    
    // Если 429, ждем и повторяем
    if (res.status === 429 && retries > 0) {
      const retryAfter = res.headers.get("Retry-After");
      const waitTime = retryAfter ? parseInt(retryAfter) * 1000 : delay;
      
      await new Promise((resolve) => setTimeout(resolve, waitTime));
      return fetchWithRetry(url, options, retries - 1, delay * 2);
    }
    
    return res;
  } catch (error) {
    if (retries > 0) {
      await new Promise((resolve) => setTimeout(resolve, delay));
      return fetchWithRetry(url, options, retries - 1, delay * 2);
    }
    throw error;
  }
}

export async function get<T = any>(url: string, headers: Headers = {}): Promise<T> {
  const cacheKey = getCacheKey(url, headers);
  const cached = cache.get(cacheKey);
  
  // Проверяем кэш
  if (cached && Date.now() < cached.expiresAt) {
    return cached.data as T;
  }
  
  // Проверяем активные запросы
  const activeRequest = activeRequests.get(cacheKey);
  if (activeRequest) {
    return activeRequest as Promise<T>;
  }
  
  // Создаем новый запрос
  const requestPromise = (async () => {
    try {
      const res = await fetchWithRetry(`${API_URL}/api${url}`, {
        method: "GET",
        headers: buildHeaders(headers),
      });
      
      if (!res.ok) {
        if (res.status === 429) {
          throw new Error(`Rate limit exceeded. Please try again later.`);
        }
        throw new Error(`GET ${url} failed: ${res.status}`);
      }
      
      const data = await res.json();
      
      // Сохраняем в кэш
      const ttl = getCacheTTL(url);
      cache.set(cacheKey, {
        data,
        timestamp: Date.now(),
        expiresAt: Date.now() + ttl,
      });
      
      return data as T;
    } finally {
      activeRequests.delete(cacheKey);
    }
  })();
  
  activeRequests.set(cacheKey, requestPromise);
  return requestPromise;
}

export async function post<T = any>(url: string, body?: any, headers: Headers = {}): Promise<T> {
  const res = await fetchWithRetry(`${API_URL}/api${url}`, {
    method: "POST",
    headers: buildHeaders(headers),
    body: body ? JSON.stringify(body) : undefined,
  });
  
  if (!res.ok) {
    if (res.status === 429) {
      throw new Error(`Rate limit exceeded. Please try again later.`);
    }
    throw new Error(`POST ${url} failed: ${res.status}`);
  }
  
  // Инвалидируем кэш для связанных GET запросов
  if (url.includes("/account-consent") || url.includes("/product-consents")) {
    clearCache();
  }
  
  return res.json();
}

export async function del<T = any>(url: string, body?: any, headers: Headers = {}): Promise<T> {
  const res = await fetchWithRetry(`${API_URL}/api${url}`, {
    method: "DELETE",
    headers: buildHeaders(headers),
    body: body ? JSON.stringify(body) : undefined,
  });
  
  if (!res.ok) {
    if (res.status === 429) {
      throw new Error(`Rate limit exceeded. Please try again later.`);
    }
    throw new Error(`DELETE ${url} failed: ${res.status}`);
  }
  
  // Инвалидируем кэш
  clearCache();
  
  return res.json();
}

// Функция для очистки кэша
export function clearCache(pattern?: string) {
  if (pattern) {
    for (const key of cache.keys()) {
      if (key.includes(pattern)) {
        cache.delete(key);
      }
    }
  } else {
    cache.clear();
  }
}

// Debounce функция для запросов
export function debouncedGet<T = any>(
  url: string,
  headers: Headers = {},
  delay = 500
): Promise<T> {
  const cacheKey = getCacheKey(url, headers);
  
  return new Promise((resolve, reject) => {
    const existingTimer = debounceTimers.get(cacheKey);
    if (existingTimer) {
      clearTimeout(existingTimer);
    }
    
    const timer = setTimeout(async () => {
      debounceTimers.delete(cacheKey);
      try {
        const result = await get<T>(url, headers);
        resolve(result);
      } catch (error) {
        reject(error);
      }
    }, delay);
    
    debounceTimers.set(cacheKey, timer);
  });
}
