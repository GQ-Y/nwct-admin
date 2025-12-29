export type ApiEnvelope<T> = {
  code: number;
  message: string;
  data: T;
};

function getApiBase(): string {
  // 开发环境：必须走同源（由 Vite proxy 转发到 18080），否则会触发浏览器 CORS
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
    // 默认强制打到节点服务端口
    return `${p}//${h}:18080`;
  }
  return "http://localhost:18080";
}

export const API_BASE = getApiBase();

const ADMIN_TOKEN_STORAGE = "node_admin_token";

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

  // 不暴露节点地址 / IP / URL
  s = s.replace(/https?:\/\/[^\s"')]+/g, "");
  s = s.replace(/\b\d{1,3}(?:\.\d{1,3}){3}:\d+\b/g, "");

  const isNetDown =
    /connection refused|i\/o timeout|no such host|context deadline exceeded|network is unreachable/i.test(s);
  if (isNetDown) return "节点服务不可达";

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

  // 添加认证头（使用 token）
  const token = getAdminToken();
  if (token) headers["Authorization"] = `Bearer ${token}`;

  const res = await fetch(url, { ...options, headers });
  const text = await res.text();

  let json: any = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    throw new Error(sanitizeErrorMessage(text || `HTTP ${res.status}`));
  }

  // node 约定：code=0 表示成功
  const okCode = json && typeof json.code === "number" ? json.code === 0 : true;
  if (!res.ok || !okCode) {
    const msg = json?.message || `HTTP ${res.status}`;
    throw new Error(sanitizeErrorMessage(msg));
  }
  return (json as ApiEnvelope<T>).data;
}

export type NodeEndpoint = { addr: string; port: number; proto: string };

export type NodeConfig = {
  node_id: string;
  public: boolean;
  name: string;
  description: string;
  region: string;
  isp: string;
  tags: string[];
  bridge_url: string;
  domain_suffix: string;
  http_enabled: boolean;
  https_enabled: boolean;
  endpoints: NodeEndpoint[];
};

export type Invite = {
  invite_id: string;
  code: string;
  revoked: boolean;
  created_at: string;
  expires_at: string;
  max_uses: number;
  used: number;
  scope_json: string;
};

export const api = {
  adminLogin: (adminKey: string) =>
    request<{ token: string; expires_at: string }>("/api/v1/admin/login", {
      method: "POST",
      body: JSON.stringify({ admin_key: String(adminKey || "").trim() }),
    }),

  adminChangePassword: (req: { old_password: string; new_password: string }) =>
    request<{ token: string; expires_at: string }>("/api/v1/admin/password/change", {
      method: "POST",
      body: JSON.stringify(req || {}),
    }),

  getNodeConfig: () => request<NodeConfig>("/api/v1/node/config", { method: "GET" }),

  updateNodeConfig: (req: {
    public?: boolean;
    name?: string;
    description?: string;
    region?: string;
    isp?: string;
    tags?: string[];
    domain_suffix?: string;
    http_enabled?: boolean;
    https_enabled?: boolean;
    endpoints?: NodeEndpoint[];
    // bridge_url 不允许修改，不在此处定义
  }) => request<{ updated: boolean }>("/api/v1/node/config", { method: "POST", body: JSON.stringify(req || {}) }),

  listInvites: (params?: { limit?: number; include_revoked?: boolean }) => {
    const q = new URLSearchParams();
    if (params?.limit) q.set("limit", String(params.limit));
    if (params?.include_revoked) q.set("include_revoked", "1");
    const query = q.toString();
    return request<{ invites: Invite[] }>(`/api/v1/node/invites${query ? `?${query}` : ""}`, { method: "GET" });
  },

  createInvite: (req: { ttl_days: number; max_uses: number; scope_json?: string }) =>
    request<{ invite_id: string; code: string; expires_at: string }>("/api/v1/node/invites", {
      method: "POST",
      body: JSON.stringify({ ...req, scope_json: req.scope_json || "{}" }),
    }),

  revokeInvite: (req: { invite_id: string }) =>
    request<{ deleted: boolean; kicked: boolean }>("/api/v1/node/invites/revoke", {
      method: "POST",
      body: JSON.stringify(req || {}),
    }),
};

