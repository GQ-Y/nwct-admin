import React, { useEffect, useMemo, useState } from "react";
import { api, PublicNode } from "../lib/api";
import { Card } from "../components/UI";
import { Toast } from "../components/Toast";
import { getAdminToken } from "../lib/api";

export const Dashboard: React.FC = () => {
  const hasTok = Boolean((getAdminToken() || "").trim());
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  const [publicNodes, setPublicNodes] = useState<PublicNode[]>([]);
  const [officialCount, setOfficialCount] = useState(0);
  const [whitelistTotal, setWhitelistTotal] = useState(0);

  const refresh = async () => {
    setLoading(true);
    try {
      const [pub, off, wl] = await Promise.all([
        api.publicNodes(),
        api.officialNodesList(),
        api.whitelistList({ limit: 1, offset: 0 }),
      ]);
      setPublicNodes(pub.nodes || []);
      setOfficialCount((off.nodes || []).length);
      setWhitelistTotal(wl.total || 0);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "刷新失败" });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    const t = window.setInterval(() => refresh(), 5000);
    return () => window.clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const stats = useMemo(() => {
    const total = publicNodes.length;
    const online = publicNodes.filter((n) => (n.status || "").toLowerCase() !== "offline").length;
    const offline = total - online;
    const pubCount = publicNodes.filter((n) => Boolean(n.public)).length;
    const privCount = total - pubCount;
    return { total, online, offline, pubCount, privCount };
  }, [publicNodes]);

  const pct = (num: number, den: number) => Math.round((num / Math.max(1, den)) * 100);

  const MiniBar = (props: { label: string; leftText: string; rightText: string; value: number; color: string }) => {
    return (
      <div>
        <div style={{ display: "flex", alignItems: "baseline", justifyContent: "space-between", gap: 12, marginBottom: 8 }}>
          <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)" }}>{props.label}</div>
          <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)" }}>
            <span style={{ color: "var(--text-primary)" }}>{props.leftText}</span>
            <span style={{ margin: "0 6px", color: "var(--text-tertiary)" }}>·</span>
            <span>{props.rightText}</span>
          </div>
        </div>
        <div style={{ height: 10, borderRadius: 999, background: "rgba(0,0,0,0.06)", overflow: "hidden" }}>
          <div style={{ width: `${Math.max(0, Math.min(100, props.value))}%`, height: "100%", background: props.color }} />
        </div>
      </div>
    );
  };

  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, minmax(0, 1fr))", gap: 16, marginBottom: 16 }}>
        <Card glass title="公共节点">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1 }}>{stats.total}</div>
          <div style={{ marginTop: 10, display: "flex", gap: 10, fontSize: 13, fontWeight: 800 }}>
            <span style={{ color: "var(--success)" }}>在线 {stats.online}</span>
            <span style={{ color: "var(--error)" }}>离线 {stats.offline}</span>
          </div>
        </Card>
        <Card glass title="官方节点">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1 }}>{officialCount}</div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>内置节点配置数量</div>
        </Card>
        <Card glass title="设备白名单">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1 }}>{whitelistTotal}</div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>可注册设备总数</div>
        </Card>
        <Card glass title="鉴权状态">
          <div style={{ fontSize: 18, fontWeight: 900, color: hasTok ? "var(--success)" : "var(--error)" }}>
            {hasTok ? "已登录" : "未登录"}
          </div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>
            {loading ? "刷新中…" : "每 5 秒自动刷新"}
          </div>
        </Card>
      </div>

      <Card title="公共节点概览">
        <div
          className="glass"
          style={{
            padding: 18,
            borderRadius: 20,
            border: "1px solid rgba(255,255,255,0.65)",
            background:
              "linear-gradient(180deg, rgba(255,255,255,0.70) 0%, rgba(255,255,255,0.55) 100%)",
          }}
        >
          <div style={{ display: "flex", alignItems: "stretch", justifyContent: "space-between", gap: 18, flexWrap: "wrap" }}>
            <div style={{ minWidth: 240, flex: "1 1 240px" }}>
              <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900 }}>总节点</div>
              <div style={{ marginTop: 6, display: "flex", alignItems: "baseline", gap: 10 }}>
                <div style={{ fontSize: 34, fontWeight: 1000 as any, letterSpacing: -0.6 }}>{stats.total}</div>
                <div style={{ fontSize: 12, fontWeight: 900, color: "var(--text-secondary)" }}>个</div>
              </div>

              <div style={{ marginTop: 10, display: "flex", gap: 10, flexWrap: "wrap" }}>
                <span className="badge badge-success" style={{ fontWeight: 900 }}>
                  在线 {stats.online}
                </span>
                <span className="badge badge-error" style={{ fontWeight: 900 }}>
                  离线 {stats.offline}
                </span>
                <span className="badge badge-warning" style={{ fontWeight: 900 }}>
                  在线率 {pct(stats.online, stats.total)}%
                </span>
              </div>
            </div>

            <div style={{ minWidth: 320, flex: "1 1 320px", display: "flex", flexDirection: "column", gap: 14 }}>
              <MiniBar
                label="公开占比"
                leftText={`公开 ${stats.pubCount}`}
                rightText={`私有 ${stats.privCount}`}
                value={pct(stats.pubCount, stats.total)}
                color="linear-gradient(90deg, rgba(10,89,247,0.95), rgba(50,117,249,0.85))"
              />
              <MiniBar
                label="在线占比"
                leftText={`在线 ${stats.online}`}
                rightText={`离线 ${stats.offline}`}
                value={pct(stats.online, stats.total)}
                color={stats.offline > 0 ? "linear-gradient(90deg, rgba(232,166,0,0.95), rgba(232,166,0,0.75))" : "linear-gradient(90deg, rgba(65,186,65,0.95), rgba(65,186,65,0.75))"}
              />
            </div>
          </div>
        </div>
      </Card>

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};


