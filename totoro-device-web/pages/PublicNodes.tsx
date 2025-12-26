import React, { useEffect, useMemo, useState } from "react";
import { api } from "../lib/api";
import { useLanguage } from "../contexts/LanguageContext";
import { Toast } from "../components/Toast";

type PublicNode = {
  node_id: string;
  name: string;
  public: boolean;
  status: string;
  region?: string;
  isp?: string;
  tags?: string[];
  domain_suffix?: string;
  heartbeat_age_s?: number;
  description?: string;
};

export const PublicNodesPage: React.FC = () => {
  const { t } = useLanguage();
  const [nodes, setNodes] = useState<PublicNode[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [inviteCode, setInviteCode] = useState("");
  const [toastOpen, setToastOpen] = useState(false);
  const [toastType, setToastType] = useState<"success" | "error" | "info">("info");
  const [toastMsg, setToastMsg] = useState("");

  // 自定义弹窗：邀请码连接（确认=解析+连接）
  const [connectOpen, setConnectOpen] = useState(false);
  const [connectNode, setConnectNode] = useState<PublicNode | null>(null);
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
    // 公开节点：直接一键连接（无需邀请码）
    setErr(null);
    try {
      await api.publicNodeConnect({ node_id: n.node_id });
      setToastType("success");
      setToastMsg("连接成功，已自动启动 FRPC。");
      setToastOpen(true);
      load();
    } catch (e: any) {
      setToastType("error");
      setToastMsg(e?.message || "连接失败");
      setToastOpen(true);
    }
  };

  const mapInviteError = (raw: string): string => {
    const s = String(raw || "");
    // 后端常见：msg=invalid_code / invalid / expired / revoked / exhausted / not_public / node_offline ...
    if (/invalid_code/i.test(s) || /invalid/i.test(s)) return "邀请码错误";
    if (/expired/i.test(s)) return "邀请码已过期";
    if (/revoked/i.test(s)) return "邀请码已失效";
    if (/exhausted/i.test(s)) return "邀请码次数已用尽";
    return "邀请码不可用，请检查后重试";
  };

  const confirmConnect = async () => {
    const code = inviteCode.trim();
    if (!code) {
      setErr("请输入正确有效的邀请码");
      return;
    }
    setConnecting(true);
    setErr(null);
    try {
      // 由后端完成“解析邀请码 -> 换票据 -> 连接并启动”
      await api.inviteConnect({ code });
      setConnectOpen(false);
      setConnectNode(null);
      setToastType("success");
      setToastMsg("连接成功，已自动启动 FRPC。");
      setToastOpen(true);
      load();
    } catch (e: any) {
      // 弹窗内提示，避免直接把后端原始错误暴露给用户
      setErr(mapInviteError(e?.message || ""));
    } finally {
      setConnecting(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  return (
    <div className="page">
      <Toast open={toastOpen} type={toastType} message={toastMsg} onClose={() => setToastOpen(false)} />
      <div className="card glass" style={{ padding: 20, marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", gap: 12, alignItems: "center" }}>
          <div>
            <div style={{ fontSize: 18, fontWeight: 800 }}>{t("public_nodes.title")}</div>
            <div style={{ fontSize: 13, color: "var(--text-secondary)" }}>
              {t("public_nodes.total")} {nodes.length} {t("public_nodes.online")} {onlineCount}
            </div>
          </div>
          <button className="btn btn-primary" onClick={load} disabled={loading}>
            {loading ? t("common.loading") : t("public_nodes.refresh")}
          </button>
        </div>
        <div style={{ display: "flex", gap: 10, marginTop: 14, flexWrap: "wrap" }}>
          <button
            className="btn btn-outline"
            onClick={() => {
              setConnectNode(null);
              setErr(null);
              setConnectOpen(true);
            }}
            disabled={loading}
          >
            {t("public_nodes.invite_connect")}
          </button>
        </div>
        {err && (
          <div style={{ marginTop: 12, color: "var(--error)", fontWeight: 600, fontSize: 13 }}>{err}</div>
        )}
      </div>

      <div className="grid" style={{ display: "grid", gap: 12 }}>
        {nodes.map((n) => {
          const badgeColor =
            n.status === "online" ? "rgba(65,186,65,0.12)" : n.status === "degraded" ? "rgba(245,158,11,0.12)" : "rgba(239,68,68,0.12)";
          const badgeText =
            n.status === "online" ? "#41BA41" : n.status === "degraded" ? "#F59E0B" : "#EF4444";
          const statusLabel =
            n.status === "online"
              ? t("common.online")
              : n.status === "degraded"
              ? t("common.degraded")
              : t("common.offline");
          return (
            <div key={n.node_id} className="card glass" style={{ padding: 16 }}>
              <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", gap: 12 }}>
                <div style={{ minWidth: 0 }}>
                  <div style={{ fontWeight: 800, fontSize: 15, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
                    {n.name || n.node_id}
                  </div>
                  {String(n.description || "").trim() ? (
                    <div style={{ fontSize: 12, color: "var(--text-secondary)", marginTop: 4, lineHeight: 1.35 }}>
                      {String(n.description || "").trim()}
                    </div>
                  ) : null}
                  <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                    {n.region || "-"} · {n.isp || "-"}
                  </div>
                </div>
                <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                  <div style={{ padding: "6px 10px", borderRadius: 999, background: badgeColor, color: badgeText, fontWeight: 800, fontSize: 12 }}>
                    {statusLabel}
                  </div>
                  <button className="btn btn-outline" onClick={() => connect(n)} disabled={loading}>
                    {t("public_nodes.connect")}
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
            if (!connecting) {
              setConnectOpen(false);
            }
          }}
        >
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 14 }}>
              <div style={{ fontSize: 18, fontWeight: 800 }}>{t("public_nodes.modal_title")}</div>
              <button
                className="btn btn-ghost"
                style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }}
                onClick={() => setConnectOpen(false)}
                disabled={connecting}
                title="关闭"
              >
                ×
              </button>
            </div>

            <div style={{ fontSize: 13, color: "var(--text-secondary)", marginBottom: 10 }}>
              {connectNode ? `目标节点：${connectNode.name || connectNode.node_id}（${connectNode.node_id}）` : t("public_nodes.enter_invite")}
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", marginBottom: 6 }}>{t("public_nodes.invite_code")}</div>
                <input className="input" value={inviteCode} onChange={(e) => setInviteCode(e.target.value)} placeholder="例如 ABCD-EFGH-IJKL" />
              </div>

              <div style={{ display: "flex", gap: 10, justifyContent: "flex-end", marginTop: 6 }}>
                <button className="btn btn-outline" onClick={() => setConnectOpen(false)} disabled={connecting}>
                  {t("common.cancel")}
                </button>
                <button className="btn btn-primary" onClick={confirmConnect} disabled={connecting || !inviteCode.trim()}>
                  {connecting ? t("public_nodes.connecting") : t("public_nodes.confirm")}
                </button>
              </div>

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


