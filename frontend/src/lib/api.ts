function requestBase() {
  if (process.env.NEXT_PUBLIC_API_URL) return process.env.NEXT_PUBLIC_API_URL;
  if (typeof window !== "undefined") return "";
  return process.env.API_URL || "http://localhost:8080";
}

let accessToken: string | null = null;
let refreshToken: string | null = null;

export function setTokens(access: string, refresh: string) {
  accessToken = access;
  refreshToken = refresh;
  if (typeof window !== "undefined") {
    localStorage.setItem("access_token", access);
    localStorage.setItem("refresh_token", refresh);
  }
}

export function loadTokens() {
  if (typeof window !== "undefined") {
    accessToken = localStorage.getItem("access_token");
    refreshToken = localStorage.getItem("refresh_token");
  }
}

export function clearTokens() {
  accessToken = null;
  refreshToken = null;
  if (typeof window !== "undefined") {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
  }
}

export function getAccessToken() {
  if (!accessToken && typeof window !== "undefined") {
    accessToken = localStorage.getItem("access_token");
  }
  return accessToken;
}

async function refreshAccessToken(): Promise<boolean> {
  loadTokens();
  if (!refreshToken) return false;
  const res = await fetch(`${requestBase()}/api/v1/auth/refresh`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refresh_token: refreshToken }),
  });
  if (!res.ok) {
    clearTokens();
    return false;
  }
  const data = await res.json();
  setTokens(data.access_token, data.refresh_token);
  return true;
}

export async function api<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  loadTokens();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };
  if (accessToken) {
    headers.Authorization = `Bearer ${accessToken}`;
  }

  const base = requestBase();
  let res = await fetch(`${base}/api/v1${path}`, { ...options, headers });

  if (res.status === 401 && refreshToken) {
    const refreshed = await refreshAccessToken();
    if (refreshed) {
      headers.Authorization = `Bearer ${getAccessToken()}`;
      res = await fetch(`${base}/api/v1${path}`, { ...options, headers });
    }
  }

  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || "Request failed");
  }

  if (res.status === 204) return undefined as T;
  return res.json();
}

export function getApiUrl() {
  const base = requestBase();
  return base || (typeof window !== "undefined" ? window.location.origin : "http://localhost:8080");
}

export function getWsUrl() {
  if (process.env.NEXT_PUBLIC_WS_URL) return process.env.NEXT_PUBLIC_WS_URL;
  if (typeof window !== "undefined") {
    if (process.env.NEXT_PUBLIC_API_URL) {
      return process.env.NEXT_PUBLIC_API_URL.replace(/^http/, "ws");
    }
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    return `${proto}//${window.location.host}`;
  }
  return "ws://localhost:8080";
}
