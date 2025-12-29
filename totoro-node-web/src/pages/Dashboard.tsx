import React, { useEffect, useState } from "react";
import { api, getAdminToken } from "../lib/api";
import { Card } from "../components/UI";
import { Toast } from "../components/Toast";

export const Dashboard: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });
  const [config, setConfig] = useState<any>(null);
  const [invitesCount, setInvitesCount] = useState(0);

  const hasToken = Boolean((getAdminToken() || "").trim());

  const refresh = async () => {
    setLoading(true);
    try {
      const [cfg, inv] = await Promise.all([api.getNodeConfig(), api.listInvites({ limit: 1000 })]);
      setConfig(cfg);
      setInvitesCount((inv.invites || []).length);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "刷新失败" });
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refresh();
    const t = window.setInterval(() => refresh(), 10000);
    return () => window.clearInterval(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(3, minmax(0, 1fr))", gap: 16, marginBottom: 16 }}>
        <Card glass title="节点状态">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1 }}>{config?.node_id || "-"}</div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>
            {config?.name || "未配置"}
          </div>
        </Card>
        <Card glass title="公开状态">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1, color: config?.public ? "var(--success)" : "var(--text-secondary)" }}>
            {config?.public ? "公开" : "私有"}
          </div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>
            {config?.public ? "节点已公开" : "节点未公开"}
          </div>
        </Card>
        <Card glass title="邀请码">
          <div style={{ fontSize: 28, fontWeight: 900, lineHeight: 1.1 }}>{invitesCount}</div>
          <div style={{ marginTop: 10, fontSize: 13, fontWeight: 800, color: "var(--text-secondary)" }}>已创建邀请码数量</div>
        </Card>
      </div>

      <Card title="节点信息">
        <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 16 }}>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>节点ID</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.node_id || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>节点名称</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.name || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>描述</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.description || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>区域</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.region || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>ISP</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.isp || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>域名后缀</div>
            <div style={{ fontSize: 14, fontWeight: 700 }}>{config?.domain_suffix || "-"}</div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>HTTP</div>
            <div style={{ fontSize: 14, fontWeight: 700, color: config?.http_enabled ? "var(--success)" : "var(--text-secondary)" }}>
              {config?.http_enabled ? "启用" : "禁用"}
            </div>
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 900, marginBottom: 6 }}>HTTPS</div>
            <div style={{ fontSize: 14, fontWeight: 700, color: config?.https_enabled ? "var(--success)" : "var(--text-secondary)" }}>
              {config?.https_enabled ? "启用" : "禁用"}
            </div>
          </div>
        </div>
      </Card>


      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};

