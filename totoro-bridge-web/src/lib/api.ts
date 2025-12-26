export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
  timestamp: string;
};

function getApiBase(): string {
  // 开发环境：必须走同源（由 Vite proxy 转发到 18090），否则会触发浏览器 CORS
  const isDev = Boolean((import.meta as any).env?.DEV);
  const env = (import.meta as any).env?.VITE_API_BASE as string | undefined;
  if (isDev) {
    // 允许用相对路径覆盖（例如 "/" 或 ""），但禁止用绝对地址（http://...）绕过 proxy
    const v = String(env || "").trim();
    if (!v || v === "/") return "";
    if (v.startsWith("/")) return v.replace(/\/+$/, "");
    return "";
  }
  if (env && env.trim()) return env.trim().replace(/\/+$/, "");
  if (typeof window !== "undefined") {
    const p = window.location.protocol || "http:";
    const h = window.location.hostname || "localhost";
    // 默认强制打到桥梁服务端口（避免前端 dev server 端口导致 API 404）
    return `${p}//${h}:18090`;
  }
  return "http://localhost:18090";
}

export const API_BASE = getApiBase();

const ADMIN_TOKEN_STORAGE = "bridge_admin_token";

export function getAdminToken(): string | null {
  return localStorage.getItem(ADMIN_TOKEN_STORAGE);
}

export function setAdminToken(t: string) {
  localStorage.setItem(ADMIN_TOKEN_STORAGE, String(t || "").trim());
}

export function clearAdminToken() {
  localStorage.removeItem(ADMIN_TOKEN_STORAGE);
}

export function sanitizeErrorMessage(input: string): string {
  let s = String(input || "").trim();
  if (!s) return "请求失败";

  // 不暴露桥梁地址 / IP / URL
  s = s.replace(/https?:\/\/[^\s"')]+/g, "");
  s = s.replace(/\b\d{1,3}(?:\.\d{1,3}){3}:\d+\b/g, "");

  const isNetDown =
    /connection refused|i\/o timeout|no such host|context deadline exceeded|network is unreachable/i.test(s);
  if (isNetDown) return "桥梁服务不可达";

  s = s.replace(/\s{2,}/g, " ").trim();
  s = s.replace(/^[,:;\-\s]+/, "").replace(/[,:;\-\s]+$/, "");
  return s || "请求失败";
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = `${API_BASE}${path.startsWith("/") ? "" : "/"}${path}`;
  const isForm = typeof FormData !== "undefined" && options.body instanceof FormData;
  const headers: Record<string, string> = {
    ...(isForm ? {} : { "Content-Type": "application/json" }),
    ...(options.headers as any),
  };

  const tok = getAdminToken();
  if (tok && !headers["Authorization"]) headers["Authorization"] = `Bearer ${tok}`;

  const res = await fetch(url, { ...options, headers });
  const text = await res.text();

  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(sanitizeErrorMessage(text || `HTTP ${res.status}`));
  }

  // bridge 约定：code=0 表示成功（兼容历史：code=200）
  const okCode =
    json && typeof json.code === "number" ? json.code === 0 || json.code === 200 : true;
  if (!res.ok || !okCode) {
    const msg = json?.message || `HTTP ${res.status}`;
    throw new Error(sanitizeErrorMessage(msg));
  }
  return (json as ApiEnvelope<T>).data;
}

export type NodeEndpoint = { addr: string; port: number; proto: string };
export type PublicNode = {
  node_id: string;
  name: string;
  description?: string;
  public: boolean;
  status: string;
  region: string;
  isp: string;
  tags: string[];
  endpoints: NodeEndpoint[];
  node_api?: string;
  domain_suffix: string;
  http_enabled: boolean;
  https_enabled: boolean;
  updated_at: string;
  heartbeat_age_s: number;
};

export type OfficialNode = {
  node_id: string;
  name: string;
  server: string;
  token: string;
  admin_addr: string;
  admin_user: string;
  admin_pwd: string;
  node_api: string;
  domain_suffix: string;
  http_enabled: boolean;
  https_enabled: boolean;
  updated_at: string;
};

export type DeviceWhitelistRow = {
  device_id: string;
  mac: string;
  enabled: boolean;
  note: string;
  updated_at: string;
};

export const api = {
  publicNodes: () => request<{ nodes: PublicNode[] }>("/api/v1/public/nodes", { method: "GET" }),

  adminLogin: (password: string) =>
    request<{ token: string; expires_at: string }>("/api/v1/admin/login", {
      method: "POST",
      body: JSON.stringify({ password: String(password || "").trim() }),
    }),

  adminChangePassword: (req: { old_password: string; new_password: string }) =>
    request<{ token: string; expires_at: string }>("/api/v1/admin/password/change", {
      method: "POST",
      body: JSON.stringify(req || {}),
    }),

  officialNodesList: () => request<{ nodes: OfficialNode[] }>("/api/v1/admin/official_nodes", { method: "GET" }),
  officialNodesUpsert: (req: {
    node_id: string;
    name?: string;
    server: string;
    token?: string;
    admin_addr?: string;
    admin_user?: string;
    admin_pwd?: string;
    node_api?: string;
    domain_suffix?: string;
    http_enabled?: boolean;
    https_enabled?: boolean;
  }) => request<any>("/api/v1/admin/official_nodes/upsert", { method: "POST", body: JSON.stringify(req || {}) }),
  officialNodesDelete: (req: { node_id: string }) =>
    request<any>("/api/v1/admin/official_nodes/delete", { method: "POST", body: JSON.stringify(req || {}) }),

  whitelistList: (params: { limit: number; offset: number }) => {
    const q = new URLSearchParams();
    q.set("limit", String(params.limit));
    q.set("offset", String(params.offset));
    return request<{ devices: DeviceWhitelistRow[]; total: number }>(`/api/v1/admin/devices/whitelist?${q.toString()}`, {
      method: "GET",
    });
  },
  whitelistUpsert: (req: { device_id: string; mac?: string; enabled?: boolean; note?: string }) =>
    request<any>("/api/v1/admin/devices/whitelist/upsert", { method: "POST", body: JSON.stringify(req || {}) }),
  whitelistDelete: (req: { device_id: string }) =>
    request<any>("/api/v1/admin/devices/whitelist/delete", { method: "POST", body: JSON.stringify(req || {}) }),
  whitelistImport: (req: { csv: string }) =>
    request<{ imported: number; skipped: number }>("/api/v1/admin/devices/whitelist/import", {
      method: "POST",
      body: JSON.stringify(req || {}),
    }),
  whitelistImportFile: (file: File) => {
    const fd = new FormData();
    fd.append("file", file);
    return request<{ imported: number; skipped: number }>("/api/v1/admin/devices/whitelist/import", {
      method: "POST",
      body: fd as any,
    });
  },
};


