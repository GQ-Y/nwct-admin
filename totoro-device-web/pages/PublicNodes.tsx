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
  node_api?: string;
  domain_suffix?: string;
  heartbeat_age_s?: number;
};

export const PublicNodesPage: React.FC = () => {
  const [nodes, setNodes] = useState<PublicNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [inviteNodeApi, setInviteNodeApi] = useState("http://127.0.0.1:18080");
  const [inviteCode, setInviteCode] = useState("");
  const [notice, setNotice] = useState<string | null>(null);

  // 自定义弹窗：公开节点一键连接（输入邀请码 -> 解析预览 -> 确认连接）
  const [connectOpen, setConnectOpen] = useState(false);
  const [connectNode, setConnectNode] = useState<PublicNode | null>(null);
  const [resolving, setResolving] = useState(false);
  const [resolved, setResolved] = useState<any | null>(null);
  const [connecting, setConnecting] = useState(false);

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
    // 打开自定义弹窗：输入邀请码 -> 解析预览 -> 确认连接
    setNotice(null);
    setErr(null);
    setConnectNode(n);
    // 由桥梁下发真实 node_api，避免猜端口
    setInviteNodeApi((n.node_api || "").trim() || `http://${ep.addr}:18080`);
    setInviteCode("");
    setResolved(null);
    setConnectOpen(true);
  };

  const resolveInvite = async () => {
    const nodeApi = inviteNodeApi.trim();
    const code = inviteCode.trim();
    if (!nodeApi || !code) {
      setErr("请填写 node_api 与 邀请码");
      return;
    }
    setResolving(true);
    setErr(null);
    try {
      const data = await api.inviteResolve({ node_api: nodeApi, code });
      setResolved(data);
    } catch (e: any) {
      setResolved(null);
      setErr(e?.message || "解析失败");
    } finally {
      setResolving(false);
    }
  };

  const confirmConnect = async () => {
    const nodeApi = inviteNodeApi.trim();
    const code = inviteCode.trim();
    if (!nodeApi || !code) {
      setErr("请填写 node_api 与 邀请码");
      return;
    }
    setConnecting(true);
    setErr(null);
    try {
      await api.inviteConnect({ node_api: nodeApi, code });
      setConnectOpen(false);
      setConnectNode(null);
      setResolved(null);
      setNotice("已连接：已保存公开节点配置，并自动启动 FRPC。");
      load();
    } catch (e: any) {
      setErr(e?.message || "连接失败");
    } finally {
      setConnecting(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="page">
      {notice && (
        <div className="card glass" style={{ padding: 14, marginBottom: 12, border: "1px solid rgba(65, 186, 65, 0.25)" }}>
          <div style={{ color: "var(--success)", fontWeight: 700, fontSize: 13 }}>{notice}</div>
        </div>
      )}
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
          <button
            className="btn btn-outline"
            onClick={() => {
              setConnectNode(null);
              setResolved(null);
              setErr(null);
              setNotice(null);
              setConnectOpen(true);
            }}
            disabled={loading}
          >
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

      {connectOpen && (
        <div
          className="modal-overlay"
          onMouseDown={() => {
            if (!connecting && !resolving) {
              setConnectOpen(false);
              setResolved(null);
            }
          }}
        >
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
              <div style={{ fontSize: 18, fontWeight: 800 }}>一键连接（公开节点）</div>
              <button
                className="btn btn-ghost"
                style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }}
                onClick={() => setConnectOpen(false)}
                disabled={connecting || resolving}
                title="关闭"
              >
                ×
              </button>
            </div>

            <div style={{ fontSize: 13, color: "var(--text-secondary)", marginBottom: 10 }}>
              {connectNode ? `目标节点：${connectNode.name || connectNode.node_id}（${connectNode.node_id}）` : "请输入邀请码以解析节点信息"}
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", marginBottom: 6 }}>node_api（可编辑）</div>
                <input className="input" value={inviteNodeApi} onChange={(e) => setInviteNodeApi(e.target.value)} placeholder="例如 http://1.2.3.4:18080" />
              </div>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", marginBottom: 6 }}>邀请码</div>
                <input className="input" value={inviteCode} onChange={(e) => setInviteCode(e.target.value)} placeholder="例如 ABCD-EFGH-IJKL" />
              </div>

              <div style={{ display: "flex", gap: 10, justifyContent: "flex-end", marginTop: 6 }}>
                <button className="btn btn-outline" onClick={() => setConnectOpen(false)} disabled={connecting || resolving}>
                  取消
                </button>
                <button className="btn btn-outline" onClick={resolveInvite} disabled={connecting || resolving || !inviteNodeApi.trim() || !inviteCode.trim()}>
                  {resolving ? "解析中…" : "解析"}
                </button>
                <button className="btn btn-primary" onClick={confirmConnect} disabled={connecting || resolving || !resolved}>
                  {connecting ? "连接中…" : "确认连接"}
                </button>
              </div>

              {resolved && (
                <div style={{ marginTop: 10 }}>
                  <div style={{ fontSize: 13, fontWeight: 800, marginBottom: 8 }}>解析结果（确认后才会连接）</div>
                  <div className="card" style={{ padding: 12, background: "rgba(0,0,0,0.02)" }}>
                    <div style={{ fontSize: 13, color: "var(--text-primary)", marginBottom: 6 }}>
                      <strong>server：</strong> {resolved.server}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--text-primary)", marginBottom: 6 }}>
                      <strong>expires_at：</strong> {resolved.expires_at}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--text-primary)", marginBottom: 6 }}>
                      <strong>node_id：</strong> {resolved?.node?.node_id || "-"}
                    </div>
                    <div style={{ fontSize: 13, color: "var(--text-primary)" }}>
                      <strong>endpoints：</strong>{" "}
                      {Array.isArray(resolved?.node?.endpoints) ? JSON.stringify(resolved.node.endpoints) : "-"}
                    </div>
                  </div>
                </div>
              )}

              {err && (
                <div style={{ marginTop: 10, color: "var(--error)", fontWeight: 700, fontSize: 13 }}>{err}</div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};


