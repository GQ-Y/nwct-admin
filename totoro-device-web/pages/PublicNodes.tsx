import React, { useEffect, useMemo, useState } from "react";
import { api } from "../lib/api";

type NodeEndpoint = { addr: string; port: number; proto: string };
type PublicNode = {
  node_id: string;
  name: string;
  public: boolean;
  status: string;
  region?: string;
  isp?: string;
  tags?: string[];
  endpoints: NodeEndpoint[];
  domain_suffix?: string;
  heartbeat_age_s?: number;
};

export const PublicNodesPage: React.FC = () => {
  const [nodes, setNodes] = useState<PublicNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [inviteNodeApi, setInviteNodeApi] = useState("http://127.0.0.1:18080");
  const [inviteCode, setInviteCode] = useState("");

  const onlineCount = useMemo(() => nodes.filter((n) => n.status === "online").length, [nodes]);

  const load = async () => {
    setLoading(true);
    setErr(null);
    try {
      const data = await api.publicNodes();
      setNodes((data?.nodes || []) as any);
    } catch (e: any) {
      setErr(e?.message || "加载失败");
    } finally {
      setLoading(false);
    }
  };

  const connect = async (n: PublicNode) => {
    const ep = n.endpoints?.[0];
    if (!ep) {
      setErr("该节点缺少 endpoints");
      return;
    }
    try {
      await api.frpConnect({ server: `${ep.addr}:${ep.port}` });
      await api.frpReload();
      setErr(null);
      alert("连接请求已发送");
    } catch (e: any) {
      setErr(e?.message || "连接失败");
    }
  };

  const connectByInvite = async () => {
    const nodeApi = inviteNodeApi.trim();
    const code = inviteCode.trim();
    if (!nodeApi || !code) {
      setErr("请填写 node_api 与 邀请码");
      return;
    }
    setLoading(true);
    setErr(null);
    try {
      await api.inviteConnect({ node_api: nodeApi, code });
      alert("已获取票据并发起连接");
    } catch (e: any) {
      setErr(e?.message || "邀请码连接失败");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="page">
      <div className="card glass" style={{ padding: 20, marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center" }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 800 }}>公开节点</div>
            <div style={{ fontSize: 13, color: "var(--text-secondary)" }}>
              共 {nodes.length} 个，在线 {onlineCount} 个
            </div>
          </div>
          <button className="btn btn-primary" onClick={load} disabled={loading}>
            {loading ? "刷新中…" : "刷新"}
          </button>
        </div>
        <div style={{ display: "flex", gap: 10, marginTop: 14, flexWrap: "wrap" }}>
          <input
            className="input"
            style={{ minWidth: 260 }}
            value={inviteNodeApi}
            onChange={(e) => setInviteNodeApi(e.target.value)}
            placeholder="节点 Node API，例如 http://1.2.3.4:18080"
          />
          <input
            className="input"
            style={{ minWidth: 220 }}
            value={inviteCode}
            onChange={(e) => setInviteCode(e.target.value)}
            placeholder="邀请码，例如 ABCD-EFGH-IJKL"
          />
          <button className="btn btn-outline" onClick={connectByInvite} disabled={loading}>
            邀请码连接
          </button>
        </div>
        {err && (
          <div style={{ marginTop: 12, color: "var(--error)", fontWeight: 600, fontSize: 13 }}>{err}</div>
        )}
      </div>

      <div className="grid" style={{ display: "grid", gap: 12 }}>
        {nodes.map((n) => {
          const ep = n.endpoints?.[0];
          const badgeColor =
            n.status === "online" ? "rgba(65,186,65,0.12)" : n.status === "degraded" ? "rgba(245,158,11,0.12)" : "rgba(239,68,68,0.12)";
          const badgeText =
            n.status === "online" ? "#41BA41" : n.status === "degraded" ? "#F59E0B" : "#EF4444";
          return (
            <div key={n.node_id} className="card glass" style={{ padding: 16 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12 }}>
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontWeight: 800, fontSize: 15, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {n.name || n.node_id}
                  </div>
                  <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                    {n.region || "-"} · {n.isp || "-"} · {ep ? `${ep.addr}:${ep.port}` : "no-endpoint"}
                  </div>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <div style={{ padding: "6px 10px", borderRadius: 999, background: badgeColor, color: badgeText, fontWeight: 800, fontSize: 12 }}>
                    {n.status}
                  </div>
                  <button className="btn btn-outline" onClick={() => connect(n)} disabled={!ep || loading}>
                    连接
                  </button>
                </div>
              </div>
              {n.tags?.length ? (
                <div style={{ marginTop: 10, display: "flex", gap: 8, flexWrap: "wrap" }}>
                  {n.tags.slice(0, 6).map((t) => (
                    <span key={t} style={{ fontSize: 12, padding: "4px 8px", borderRadius: 999, background: "rgba(10,89,247,0.08)" }}>
                      {t}
                    </span>
                  ))}
                </div>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
};


