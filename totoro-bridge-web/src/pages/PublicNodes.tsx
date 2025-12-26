import React, { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronUp, RefreshCw } from "lucide-react";
import { Toast } from "../components/Toast";
import { Badge, Button, Card, SearchInput, Select } from "../components/UI";
import { api, PublicNode } from "../lib/api";

function nodeOnline(n: PublicNode) {
  return (n.status || "").toLowerCase() !== "offline";
}

function statusText(n: PublicNode) {
  const s = String(n.status || "").toLowerCase();
  if (s === "online") return "在线";
  if (s === "offline") return "离线";
  return n.status || (nodeOnline(n) ? "在线" : "离线");
}

export const PublicNodesPage: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  const [nodes, setNodes] = useState<PublicNode[]>([]);
  const [q, setQ] = useState("");
  const [status, setStatus] = useState<"all" | "online" | "offline">("all");
  const [pub, setPub] = useState<"all" | "public" | "private">("all");

  const [expanded, setExpanded] = useState<Record<string, boolean>>({});

  const refresh = async () => {
    setLoading(true);
    try {
      const data = await api.publicNodes();
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
    return nodes
      .filter((n) => {
        if (status === "online" && !nodeOnline(n)) return false;
        if (status === "offline" && nodeOnline(n)) return false;
        if (pub === "public" && !n.public) return false;
        if (pub === "private" && n.public) return false;
        if (!qq) return true;
        const hay = [
          n.node_id,
          n.name,
          n.description || "",
          n.region || "",
          n.isp || "",
          n.domain_suffix || "",
          (n.tags || []).join(","),
        ]
          .join(" ")
          .toLowerCase();
        return hay.includes(qq);
      })
      .sort((a, b) => String(b.updated_at || "").localeCompare(String(a.updated_at || "")));
  }, [nodes, pub, q, status]);

  return (
    <div>
      <Card
        title="公共节点"
        extra={
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <Button variant="outline" onClick={refresh} disabled={loading} style={{ height: 40 }}>
              <RefreshCw size={16} /> 刷新
            </Button>
          </div>
        }
      >
        <div style={{ display: "flex", gap: 12, flexWrap: "wrap", alignItems: "center", marginBottom: 14 }}>
          <SearchInput placeholder="搜索 节点ID / 名称 / 地区 / 标签…" value={q} onChange={(e) => setQ((e.target as any).value)} />
          <Select
            value={status}
            onChange={(v) => setStatus(v as any)}
            options={[
              { label: "全部状态", value: "all" },
              { label: "在线", value: "online" },
              { label: "离线", value: "offline" },
            ]}
          />
          <Select
            value={pub}
            onChange={(v) => setPub(v as any)}
            options={[
              { label: "全部类型", value: "all" },
              { label: "公开节点", value: "public" },
              { label: "私有节点", value: "private" },
            ]}
          />
          <div style={{ marginLeft: "auto", fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>
            {loading ? "加载中…" : `共 ${filtered.length} 条`}
          </div>
        </div>

        <table className="table">
          <thead>
            <tr>
              <th>节点</th>
              <th>状态</th>
              <th>公开</th>
              <th>域名后缀</th>
              <th>HTTP/HTTPS</th>
              <th>心跳</th>
              <th style={{ width: 120 }}>操作</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((n) => {
              const isOpen = Boolean(expanded[n.node_id]);
              return (
                <React.Fragment key={n.node_id}>
                  <tr>
                    <td>
                      <div style={{ fontWeight: 900 }}>{n.name || n.node_id}</div>
                      <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>{n.node_id}</div>
                      {n.description && (
                        <div style={{ fontSize: 12, color: "var(--text-secondary)", marginTop: 6, maxWidth: 520, fontWeight: 700 }}>
                          {n.description}
                        </div>
                      )}
                    </td>
                    <td>{nodeOnline(n) ? <Badge status="online" text={statusText(n)} /> : <Badge status="offline" text={statusText(n)} />}</td>
                    <td style={{ fontWeight: 800, color: n.public ? "var(--success)" : "var(--text-secondary)" }}>{n.public ? "公开" : "私有"}</td>
                    <td style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace" }}>
                      {n.domain_suffix || "-"}
                    </td>
                    <td style={{ fontWeight: 800 }}>
                      <span style={{ color: n.http_enabled ? "var(--success)" : "var(--text-secondary)" }}>HTTP</span>
                      <span style={{ margin: "0 6px", color: "var(--text-tertiary)" }}>/</span>
                      <span style={{ color: n.https_enabled ? "var(--success)" : "var(--text-secondary)" }}>HTTPS</span>
                    </td>
                    <td style={{ fontWeight: 800, color: "var(--text-secondary)" }}>{typeof n.heartbeat_age_s === "number" ? `${n.heartbeat_age_s}s` : "-"}</td>
                    <td>
                      <button
                        className="btn btn-outline"
                        style={{ height: 36, padding: "0 12px" }}
                        onClick={() => setExpanded((m) => ({ ...m, [n.node_id]: !isOpen }))}
                      >
                        {isOpen ? (
                          <>
                            收起 <ChevronUp size={16} />
                          </>
                        ) : (
                          <>
                            详情 <ChevronDown size={16} />
                          </>
                        )}
                      </button>
                    </td>
                  </tr>
                  {isOpen && (
                    <tr>
                      <td colSpan={7} style={{ background: "rgba(0,0,0,0.02)" }}>
                        <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 16 }}>
                          <div className="glass" style={{ padding: 14, borderRadius: 16 }}>
                            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 8 }}>连接地址</div>
                            {n.endpoints && n.endpoints.length ? (
                              <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
                                {n.endpoints.map((ep, idx) => (
                                  <div
                                    key={idx}
                                    style={{
                                      fontFamily:
                                        "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
                                      fontSize: 12,
                                      fontWeight: 800,
                                      color: "var(--text-primary)",
                                    }}
                                  >
                                    {ep.proto}://{ep.addr}:{ep.port}
                                  </div>
                                ))}
                              </div>
                            ) : (
                              <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>无</div>
                            )}
                          </div>
                          <div className="glass" style={{ padding: 14, borderRadius: 16 }}>
                            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 8 }}>节点 API</div>
                            <div style={{ fontSize: 12, color: "var(--text-primary)", fontWeight: 800, wordBreak: "break-all" }}>
                              {n.node_api || "-"}
                            </div>
                            <div style={{ marginTop: 10, fontSize: 12, color: "var(--text-secondary)", fontWeight: 800 }}>
                              地区：{n.region || "-"} / ISP：{n.isp || "-"}
                            </div>
                            {n.tags && n.tags.length ? (
                              <div style={{ marginTop: 10, display: "flex", flexWrap: "wrap", gap: 8 }}>
                                {n.tags.slice(0, 12).map((t) => (
                                  <span key={t} className="badge badge-warning">
                                    {t}
                                  </span>
                                ))}
                              </div>
                            ) : null}
                          </div>
                        </div>
                      </td>
                    </tr>
                  )}
                </React.Fragment>
              );
            })}
            {!filtered.length && (
              <tr>
                <td colSpan={7} style={{ color: "var(--text-secondary)", fontWeight: 800 }}>
                  暂无数据
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </Card>

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};


