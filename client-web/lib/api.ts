export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
  timestamp: string;
};

function getApiBase(): string {
  const env = (import.meta as any).env?.VITE_API_BASE as string | undefined;
  if (env && env.trim()) return env.trim().replace(/\/+$/, "");

  // dev 默认后端端口（与你目前的运行方式一致）
  if (typeof window !== "undefined") {
    const isVite = window.location.port === "5173" || window.location.port === "5174";
    if (isVite) return "http://localhost:18080";
    return window.location.origin;
  }
  return "http://localhost:18080";
}

export const API_BASE = getApiBase();

export function getToken(): string | null {
  return localStorage.getItem("token");
}

export function setToken(token: string) {
  localStorage.setItem("token", token);
}

export function clearToken() {
  localStorage.removeItem("token");
}

async function request<T>(
  path: string,
  options: RequestInit & { skipAuth?: boolean } = {}
): Promise<T> {
  const url = `${API_BASE}${path.startsWith("/") ? "" : "/"}${path}`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as any),
  };

  if (!options.skipAuth) {
    const token = getToken();
    if (token) headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(url, { ...options, headers });
  const text = await res.text();
  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    // 非 JSON：直接抛
    throw new Error(text || `HTTP ${res.status}`);
  }

  if (!res.ok || (json && typeof json.code === "number" && json.code !== 200)) {
    const msg = json?.message || `HTTP ${res.status}`;
    throw new Error(msg);
  }
  return (json as ApiEnvelope<T>).data;
}

export const api = {
  login: (username: string, password: string) =>
    request<{ token: string; expires_in: number }>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
      skipAuth: true,
    }),

  initStatus: () => request<{ initialized: boolean }>("/api/v1/config/init/status"),
  configInit: (adminPassword: string, partial?: any) =>
    request<any>("/api/v1/config/init", {
      method: "POST",
      body: JSON.stringify({ ...(partial || {}), admin_password: adminPassword }),
      skipAuth: true,
    }),

  systemInfo: () => request<any>("/api/v1/system/info"),

  devices: () => request<{ devices: any[]; total: number }>("/api/v1/devices"),
  deviceDetail: (ip: string) => request<any>(`/api/v1/devices/${encodeURIComponent(ip)}`),
  scanStart: () =>
    request<any>("/api/v1/devices/scan/start", { method: "POST", body: "{}" }),

  npsStatus: () => request<any>("/api/v1/nps/status"),
  mqttStatus: () => request<any>("/api/v1/mqtt/status"),
  mqttLogs: () => request<any>("/api/v1/mqtt/logs"),
};


