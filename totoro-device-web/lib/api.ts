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
  if (isDev) return "http://localhost:80";

  // 生产环境同域部署（例如 Nginx 反代到 /api/v1），则直接用当前 origin
  if (typeof window !== "undefined") return window.location.origin;
  return "http://localhost:80";
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
  changePassword: (req: { old_password: string; new_password: string; confirm_password: string }) =>
    request<any>("/api/v1/auth/change-password", { method: "POST", body: JSON.stringify(req) }),

  initStatus: () => request<{ initialized: boolean }>("/api/v1/config/init/status"),
  configInit: (adminPassword: string, partial?: any) =>
    request<any>("/api/v1/config/init", {
      method: "POST",
      body: JSON.stringify({ ...(partial || {}), admin_password: adminPassword }),
      skipAuth: true,
    }),
  configGet: () => request<any>("/api/v1/config"),

  systemInfo: () => request<any>("/api/v1/system/info"),
  systemLogs: (lines: number = 200) =>
    request<any>(`/api/v1/system/logs?lines=${encodeURIComponent(String(lines))}`),
  systemLogsClear: () => request<any>("/api/v1/system/logs/clear", { method: "POST", body: "{}" }),
  systemRestart: (type: "soft" | "hard" = "soft") =>
    request<any>("/api/v1/system/restart", { method: "POST", body: JSON.stringify({ type }) }),

  devices: (params?: { status?: string; type?: string; page?: number; page_size?: number }) => {
    const q = new URLSearchParams();
    if (params?.status) q.set("status", params.status);
    if (params?.type) q.set("type", params.type);
    if (params?.page) q.set("page", String(params.page));
    if (params?.page_size) q.set("page_size", String(params.page_size));
    const qs = q.toString();
    return request<{ devices: any[]; total: number }>(`/api/v1/devices${qs ? `?${qs}` : ""}`);
  },
  devicesActivity: (limit: number = 20) =>
    request<{ activities: any[] }>(`/api/v1/devices/activity?limit=${encodeURIComponent(String(limit))}`),
  deviceDetail: (ip: string) => request<any>(`/api/v1/devices/${encodeURIComponent(ip)}`),
  deviceScanPorts: (ip: string, ports?: string) =>
    request<any>(`/api/v1/devices/${encodeURIComponent(ip)}/ports/scan`, {
      method: "POST",
      body: JSON.stringify({ ports: ports?.trim() || undefined }),
    }),
  scanStart: () =>
    request<any>("/api/v1/devices/scan/start", { method: "POST", body: "{}" }),
  scanStop: () =>
    request<any>("/api/v1/devices/scan/stop", { method: "POST", body: "{}" }),
  scanStatus: () => request<any>("/api/v1/devices/scan/status"),

  frpStatus: () => request<any>("/api/v1/frp/status"),
  frpModeSet: (req: { mode: "builtin" | "manual" | "public" }) =>
    request<any>("/api/v1/frp/mode", { method: "POST", body: JSON.stringify(req) }),
  frpConfigSave: (req: {
    server?: string;
    token?: string;
    admin_addr?: string;
    admin_user?: string;
    admin_pwd?: string;
    domain_suffix?: string;
  }) => request<any>("/api/v1/frp/config", { method: "POST", body: JSON.stringify(req || {}) }),
  frpUseBuiltin: () => request<any>("/api/v1/frp/builtin/use", { method: "POST", body: "{}" }),
  frpConnect: (req?: {
    server?: string;
    token?: string;
    admin_addr?: string;
    admin_user?: string;
    admin_pwd?: string;
  }) => request<any>("/api/v1/frp/connect", { method: "POST", body: JSON.stringify(req || {}) }),
  frpDisconnect: () => request<any>("/api/v1/frp/disconnect", { method: "POST", body: "{}" }),
  frpTunnels: () => request<{ tunnels: any[] }>("/api/v1/frp/tunnels"),
  frpAddTunnel: (tunnel: {
    name: string;
    type: string;
    local_ip: string;
    local_port: number;
    remote_port?: number;
    domain?: string;
  }) => request<any>("/api/v1/frp/tunnels", { method: "POST", body: JSON.stringify(tunnel) }),
  frpRemoveTunnel: (name: string) =>
    request<any>(`/api/v1/frp/tunnels/${encodeURIComponent(name)}`, { method: "DELETE" }),
  frpUpdateTunnel: (name: string, tunnel: {
    name: string;
    type: string;
    local_ip: string;
    local_port: number;
    remote_port?: number;
    domain?: string;
  }) =>
    request<any>(`/api/v1/frp/tunnels/${encodeURIComponent(name)}`, {
      method: "PUT",
      body: JSON.stringify(tunnel),
    }),
  frpReload: () => request<any>("/api/v1/frp/reload", { method: "POST", body: "{}" }),

  publicNodes: () => request<{ nodes: any[] }>("/api/v1/public/nodes"),
  publicNodeConnect: (req: { node_id: string }) =>
    request<any>("/api/v1/public/nodes/connect", { method: "POST", body: JSON.stringify(req) }),
  inviteResolve: (req: { code: string }) =>
    request<any>("/api/v1/public/invites/resolve", { method: "POST", body: JSON.stringify(req) }),
  inviteConnect: (req: { code: string }) =>
    request<any>("/api/v1/public/invites/connect", { method: "POST", body: JSON.stringify(req) }),

  networkStatus: (opts?: { skipAuth?: boolean }) =>
    request<any>("/api/v1/network/status", { skipAuth: Boolean(opts?.skipAuth) }),
  networkInterfaces: (opts?: { skipAuth?: boolean }) =>
    request<{ interfaces: any[] }>("/api/v1/network/interfaces", { skipAuth: Boolean(opts?.skipAuth) }),
  wifiScan: (opts?: { allow_redacted?: boolean; skipAuth?: boolean }) => {
    const q = new URLSearchParams();
    if (opts?.allow_redacted) q.set("allow_redacted", "1");
    const qs = q.toString();
    return request<{ networks: any[] }>(`/api/v1/network/wifi/scan${qs ? `?${qs}` : ""}`, {
      skipAuth: Boolean(opts?.skipAuth),
    });
  },
  wifiConnect: (
    req: {
      ssid: string;
      password?: string;
      security?: string;
      save?: boolean;
      auto_connect?: boolean;
      priority?: number;
    },
    opts?: { skipAuth?: boolean }
  ) =>
    request<any>("/api/v1/network/wifi/connect", {
      method: "POST",
      body: JSON.stringify(req),
      skipAuth: Boolean(opts?.skipAuth),
    }),

  networkApply: (
    req: { interface?: string; ip_mode: string; ip?: string; netmask?: string; gateway?: string; dns?: string },
    opts?: { skipAuth?: boolean }
  ) =>
    request<any>("/api/v1/network/apply", {
      method: "POST",
      body: JSON.stringify(req),
      skipAuth: Boolean(opts?.skipAuth),
    }),

  toolsPing: (req: { target: string; count?: number; timeout?: number }) =>
    request<any>("/api/v1/tools/ping", { method: "POST", body: JSON.stringify(req) }),
  toolsTraceroute: (req: { target?: string; max_hops?: number; timeout?: number }) =>
    request<any>("/api/v1/tools/traceroute", { method: "POST", body: JSON.stringify(req || {}) }),
  toolsSpeedtest: (req?: {
    mode?: "web" | "download";
    url?: string;
    method?: "GET" | "HEAD";
    count?: number;
    timeout?: number;
    download_bytes?: number;
    server?: string;
    test_type?: string;
  }) => request<any>("/api/v1/tools/speedtest", { method: "POST", body: JSON.stringify(req || {}) }),
  toolsPortscan: (req: { target: string; ports?: any; timeout?: number; scan_type?: string }) =>
    request<any>("/api/v1/tools/portscan", { method: "POST", body: JSON.stringify(req) }),
  toolsDNS: (req: { query: string; type?: string; server?: string }) =>
    request<any>("/api/v1/tools/dns", { method: "POST", body: JSON.stringify(req) }),
};


