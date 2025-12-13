export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
  timestamp: string;
};

function getApiBase(): string {
  const env = (import.meta as any).env?.VITE_API_BASE as string | undefined;
  if (env && env.trim()) return env.trim().replace(/\/+$/, "");

  // Vite dev 环境：无论端口是多少，都默认把 API 指向后端（避免误打到前端 origin 导致 404）
  const isDev = Boolean((import.meta as any).env?.DEV);
  if (isDev) return "http://localhost:8080";

  // 生产环境同域部署（例如 Nginx 反代到 /api/v1），则直接用当前 origin
  if (typeof window !== "undefined") return window.location.origin;
  return "http://localhost:8080";
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
  systemLogs: (lines: number = 200) =>
    request<any>(`/api/v1/system/logs?lines=${encodeURIComponent(String(lines))}`),
  systemRestart: (type: "soft" | "hard" = "soft") =>
    request<any>("/api/v1/system/restart", { method: "POST", body: JSON.stringify({ type }) }),

  devices: () => request<{ devices: any[]; total: number }>("/api/v1/devices"),
  deviceDetail: (ip: string) => request<any>(`/api/v1/devices/${encodeURIComponent(ip)}`),
  scanStart: () =>
    request<any>("/api/v1/devices/scan/start", { method: "POST", body: "{}" }),

  npsStatus: () => request<any>("/api/v1/nps/status"),
  npsNpcInstall: (req?: { version?: string; install_dir?: string }) =>
    request<any>("/api/v1/nps/npc/install", { method: "POST", body: JSON.stringify(req || {}) }),
  npsConnect: (req: {
    server: string;
    vkey: string;
    client_id: string;
    npc_path?: string;
    npc_config_path?: string;
    npc_args?: string[];
  }) => request<any>("/api/v1/nps/connect", { method: "POST", body: JSON.stringify(req) }),
  npsDisconnect: () => request<any>("/api/v1/nps/disconnect", { method: "POST", body: "{}" }),
  npsTunnels: () => request<{ tunnels: any[] }>("/api/v1/nps/tunnels"),

  mqttStatus: () => request<any>("/api/v1/mqtt/status"),
  mqttConnect: (req: {
    server: string;
    port: number;
    username?: string;
    password?: string;
    client_id: string;
    tls?: boolean;
  }) => request<any>("/api/v1/mqtt/connect", { method: "POST", body: JSON.stringify(req) }),
  mqttDisconnect: () => request<any>("/api/v1/mqtt/disconnect", { method: "POST", body: "{}" }),
  mqttPublish: (req: { topic: string; payload: string }) =>
    request<any>("/api/v1/mqtt/publish", { method: "POST", body: JSON.stringify(req) }),
  mqttLogs: (params?: { topic?: string; direction?: string; page?: number; page_size?: number }) => {
    const q = new URLSearchParams();
    if (params?.topic) q.set("topic", params.topic);
    if (params?.direction) q.set("direction", params.direction);
    if (params?.page) q.set("page", String(params.page));
    if (params?.page_size) q.set("page_size", String(params.page_size));
    const qs = q.toString();
    return request<any>(`/api/v1/mqtt/logs${qs ? `?${qs}` : ""}`);
  },
};


