import React, { useEffect, useMemo, useState } from "react";
import { Plus, RefreshCw, Trash2, Pencil } from "lucide-react";
import { Toast } from "../components/Toast";
import { Badge, Button, Card, Input, SearchInput } from "../components/UI";
import { api, OfficialNode } from "../lib/api";

type EditState = {
  open: boolean;
  mode: "create" | "edit";
  node: Partial<OfficialNode>;
};

function formatTimeShort(s: string): { label: string; title: string } {
  const raw = String(s || "").trim();
  if (!raw) return { label: "-", title: "" };
  const d = new Date(raw);
  if (Number.isNaN(d.getTime())) return { label: raw, title: raw };
  const pad = (n: number) => String(n).padStart(2, "0");
  const label = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
  return { label, title: raw };
}

export const OfficialNodesPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  const [nodes, setNodes] = useState<OfficialNode[]>([]);
  const [q, setQ] = useState("");
  const [edit, setEdit] = useState<EditState>({ open: false, mode: "create", node: {} });
  const [saving, setSaving] = useState(false);

  const refresh = async () => {
    setLoading(true);
    try {
      const data = await api.officialNodesList();
      setNodes(data.nodes || []);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "加载失败" });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const filtered = useMemo(() => {
    const qq = q.trim().toLowerCase();
    return (nodes || []).filter((n) => {
      if (!qq) return true;
      const hay = [n.node_id, n.name, n.server, n.domain_suffix, n.node_api].join(" ").toLowerCase();
      return hay.includes(qq);
    });
  }, [nodes, q]);

  const openCreate = () => setEdit({ open: true, mode: "create", node: { http_enabled: false, https_enabled: false } as any });
  const openEdit = (n: OfficialNode) => setEdit({ open: true, mode: "edit", node: { ...n } });

  const canSave = useMemo(() => {
    const n = edit.node || {};
    return Boolean(String(n.node_id || "").trim() && String(n.server || "").trim() && !saving);
  }, [edit.node, saving]);

  const save = async () => {
    const n = edit.node || {};
    const req = {
      node_id: String(n.node_id || "").trim(),
      name: String(n.name || "").trim(),
      server: String(n.server || "").trim(),
      token: String(n.token || "").trim(),
      admin_addr: String(n.admin_addr || "").trim(),
      admin_user: String(n.admin_user || "").trim(),
      admin_pwd: String(n.admin_pwd || "").trim(),
      node_api: String(n.node_api || "").trim(),
      domain_suffix: String(n.domain_suffix || "").trim(),
      http_enabled: Boolean((n as any).http_enabled),
      https_enabled: Boolean((n as any).https_enabled),
    };
    if (!req.node_id || !req.server) {
      setToast({ open: true, type: "error", message: "node_id 与 server 为必填" });
      return;
    }
    setSaving(true);
    try {
      await api.officialNodesUpsert(req);
      setToast({ open: true, type: "success", message: "保存成功" });
      setEdit((s) => ({ ...s, open: false }));
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "保存失败" });
    } finally {
      setSaving(false);
    }
  };

  const del = async (nodeID: string) => {
    const ok = window.confirm(`确认删除官方节点：${nodeID} ？`);
    if (!ok) return;
    try {
      await api.officialNodesDelete({ node_id: nodeID });
      setToast({ open: true, type: "success", message: "已删除" });
      await refresh();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "删除失败" });
    }
  };

  return (
    <div>
      <Card
        title="官方节点"
        extra={
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <Button variant="outline" onClick={refresh} disabled={loading} style={{ height: 40 }}>
              <RefreshCw size={16} /> 刷新
            </Button>
            <Button variant="primary" onClick={openCreate} style={{ height: 40 }}>
              <Plus size={16} /> 新增
            </Button>
          </div>
        }
      >
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center", marginBottom: 14 }}>
          <SearchInput placeholder="搜索 节点ID / 名称 / 服务地址 / 域名后缀…" value={q} onChange={(e) => setQ((e.target as any).value)} />
          <div style={{ marginLeft: "auto", fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>
            {loading ? "加载中…" : `共 ${filtered.length} 条`}
          </div>
        </div>

        <table className="table">
          <thead>
            <tr>
              <th>节点</th>
              <th>服务地址</th>
              <th>域名后缀</th>
              <th>HTTP/HTTPS</th>
              <th>更新时间</th>
              <th style={{ width: 160 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((n) => (
              <tr key={n.node_id}>
                <td>
                  <div style={{ fontWeight: 900 }}>{n.name || n.node_id}</div>
                  <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>{n.node_id}</div>
                </td>
                <td style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace" }}>
                  {n.server}
                </td>
                <td style={{ fontWeight: 800 }}>{n.domain_suffix || "-"}</td>
                <td>
                  <span style={{ display: "inline-flex", gap: 8 }}>
                    {n.http_enabled ? <Badge status="success" text="HTTP" /> : <Badge status="warn" text="HTTP 关" />}
                    {n.https_enabled ? <Badge status="success" text="HTTPS" /> : <Badge status="warn" text="HTTPS 关" />}
                  </span>
                </td>
                <td style={{ fontWeight: 800, color: "var(--text-secondary)" }} title={formatTimeShort(n.updated_at).title}>
                  {formatTimeShort(n.updated_at).label}
                </td>
                <td>
                  <div style={{ display: "flex", gap: 10 }}>
                    <button className="btn btn-outline" style={{ height: 36, padding: "0 12px" }} onClick={() => openEdit(n)}>
                      <Pencil size={16} /> 编辑
                    </button>
                    <button
                      className="btn btn-outline"
                      style={{ height: 36, padding: "0 12px", color: "var(--error)" }}
                      onClick={() => del(n.node_id)}
                      title="删除"
                    >
                      <Trash2 size={16} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
            {!filtered.length && (
              <tr>
                <td colSpan={6} style={{ color: "var(--text-secondary)", fontWeight: 800 }}>
                  暂无数据
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      {edit.open && (
        <div className="modal-overlay" onMouseDown={() => setEdit((s) => ({ ...s, open: false }))}>
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
              <div style={{ fontSize: 18, fontWeight: 900 }}>{edit.mode === "create" ? "新增官方节点" : "编辑官方节点"}</div>
              <button className="btn btn-ghost" style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }} onClick={() => setEdit((s) => ({ ...s, open: false }))}>
                ×
              </button>
            </div>

            <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 12 }}>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>节点ID *</div>
                <Input
                  value={String(edit.node.node_id || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, node_id: (e.target as any).value } }))}
                  placeholder="例如 node_001（仅字母/数字/下划线）"
                  disabled={edit.mode === "edit"}
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>节点名称</div>
                <Input
                  value={String(edit.node.name || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, name: (e.target as any).value } }))}
                  placeholder="例如 上海-联通"
                />
              </div>
              <div style={{ gridColumn: "1 / -1" }}>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>服务地址（FRPS）*</div>
                <Input
                  value={String(edit.node.server || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, server: (e.target as any).value } }))}
                  placeholder="例如 1.2.3.4:7000"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>访问令牌（Token）</div>
                <Input
                  value={String(edit.node.token || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, token: (e.target as any).value } }))}
                  placeholder="可选：FRPS token"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>域名后缀</div>
                <Input
                  value={String(edit.node.domain_suffix || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, domain_suffix: (e.target as any).value } }))}
                  placeholder="例如 frpc.zyckj.club（不需要填前导点）"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>管理地址</div>
                <Input
                  value={String(edit.node.admin_addr || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, admin_addr: (e.target as any).value } }))}
                  placeholder="可选：例如 1.2.3.4:7500"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>管理账号</div>
                <Input
                  value={String(edit.node.admin_user || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, admin_user: (e.target as any).value } }))}
                  placeholder="可选"
                />
              </div>
              <div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>管理密码</div>
                <Input
                  type="password"
                  value={String(edit.node.admin_pwd || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, admin_pwd: (e.target as any).value } }))}
                  placeholder="可选"
                />
              </div>
              <div style={{ gridColumn: "1 / -1" }}>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)", marginBottom: 6 }}>节点 API 地址</div>
                <Input
                  value={String(edit.node.node_api || "")}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, node_api: (e.target as any).value } }))}
                  placeholder="可选：例如 http://1.2.3.4:18091"
                />
              </div>
            </div>

            <div style={{ display: "flex", gap: 12, marginTop: 14, alignItems: "center" }}>
              <label style={{ display: "flex", alignItems: "center", gap: 8, fontWeight: 900, fontSize: 13 }}>
                <input
                  type="checkbox"
                  checked={Boolean((edit.node as any).http_enabled)}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, http_enabled: e.target.checked } as any }))}
                />
                启用 HTTP
              </label>
              <label style={{ display: "flex", alignItems: "center", gap: 8, fontWeight: 900, fontSize: 13 }}>
                <input
                  type="checkbox"
                  checked={Boolean((edit.node as any).https_enabled)}
                  onChange={(e) => setEdit((s) => ({ ...s, node: { ...s.node, https_enabled: e.target.checked } as any }))}
                />
                启用 HTTPS
              </label>
            </div>

            <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 16 }}>
              <Button variant="outline" onClick={() => setEdit((s) => ({ ...s, open: false }))} disabled={saving}>
                取消
              </Button>
              <Button variant="primary" onClick={save} disabled={!canSave}>
                {saving ? "保存中…" : "保存"}
              </Button>
            </div>
          </div>
        </div>
      )}

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};


