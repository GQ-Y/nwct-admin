import React, { useEffect, useState } from "react";
import { api } from "../lib/api";
import { Card, Button, Input, Badge, ConfirmDialog } from "../components/UI";
import { Toast } from "../components/Toast";
import { Trash2, Plus } from "lucide-react";

export const Invites: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [creating, setCreating] = useState(false);
  const [revoking, setRevoking] = useState<string | null>(null);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });
  const [confirmDialog, setConfirmDialog] = useState<{ open: boolean; inviteId: string | null }>({ open: false, inviteId: null });

  const [invites, setInvites] = useState<any[]>([]);
  const [createForm, setCreateForm] = useState({ ttl_days: 1, max_uses: 50 });
  const [showCreateForm, setShowCreateForm] = useState(false);

  useEffect(() => {
    loadInvites();
  }, []);

  const loadInvites = async () => {
    setLoading(true);
    try {
      const res = await api.listInvites({ limit: 1000, include_revoked: true });
      setInvites(res.invites || []);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "加载邀请码失败" });
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async () => {
    if (createForm.ttl_days < 0 || createForm.max_uses < 0) {
      setToast({ open: true, type: "error", message: "有效期和使用次数不能为负数" });
      return;
    }
    setCreating(true);
    try {
      await api.createInvite({ ...createForm, scope_json: "{}" });
      setToast({ open: true, type: "success", message: "邀请码创建成功" });
      setShowCreateForm(false);
      setCreateForm({ ttl_days: 1, max_uses: 50 });
      await loadInvites();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "创建邀请码失败" });
    } finally {
      setCreating(false);
    }
  };

  const handleRevoke = (inviteId: string) => {
    setConfirmDialog({ open: true, inviteId });
  };

  const confirmRevoke = async () => {
    if (!confirmDialog.inviteId) return;
    setRevoking(confirmDialog.inviteId);
    setConfirmDialog({ open: false, inviteId: null });
    try {
      await api.revokeInvite({ invite_id: confirmDialog.inviteId });
      setToast({ open: true, type: "success", message: "邀请码已删除" });
      await loadInvites();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "删除邀请码失败" });
    } finally {
      setRevoking(null);
    }
  };

  const formatDate = (dateStr: string) => {
    if (!dateStr || dateStr === "0001-01-01T00:00:00Z") return "-";
    try {
      const d = new Date(dateStr);
      return d.toLocaleString("zh-CN");
    } catch {
      return dateStr;
    }
  };

  const isExpired = (expiresAt: string) => {
    if (!expiresAt || expiresAt === "0001-01-01T00:00:00Z") return false;
    try {
      return new Date(expiresAt) < new Date();
    } catch {
      return false;
    }
  };

  return (
    <div>
      <Card
        title="邀请码管理"
        extra={
          <Button variant="primary" onClick={() => setShowCreateForm(!showCreateForm)} style={{ display: "flex", alignItems: "center", gap: 6 }}>
            <Plus size={16} />
            {showCreateForm ? "取消" : "创建邀请码"}
          </Button>
        }
      >
        {showCreateForm && (
          <div
            className="glass"
            style={{
              padding: 18,
              borderRadius: 16,
              marginBottom: 18,
              border: "1px solid rgba(255,255,255,0.65)",
            }}
          >
            <div style={{ fontSize: 14, fontWeight: 800, marginBottom: 12 }}>创建新邀请码</div>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>有效期（天）</div>
                <Input
                  type="number"
                  placeholder="1"
                  value={createForm.ttl_days}
                  onChange={(e) => setCreateForm({ ...createForm, ttl_days: parseInt((e.target as any).value) || 0 })}
                />
              </div>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>最大使用次数</div>
                <Input
                  type="number"
                  placeholder="50"
                  value={createForm.max_uses}
                  onChange={(e) => setCreateForm({ ...createForm, max_uses: parseInt((e.target as any).value) || 0 })}
                />
              </div>
              <div style={{ display: "flex", gap: 10, justifyContent: "flex-end" }}>
                <Button variant="outline" onClick={() => setShowCreateForm(false)} disabled={creating}>
                  取消
                </Button>
                <Button variant="primary" onClick={handleCreate} disabled={creating}>
                  {creating ? "创建中…" : "创建"}
                </Button>
              </div>
            </div>
          </div>
        )}

        <div style={{ display: "flex", gap: 10, marginBottom: 16 }}>
          <Button variant="outline" onClick={loadInvites} disabled={loading}>
            {loading ? "加载中…" : "刷新"}
          </Button>
        </div>

        {loading && invites.length === 0 ? (
          <div style={{ textAlign: "center", padding: 40, color: "var(--text-secondary)" }}>加载中…</div>
        ) : invites.length === 0 ? (
          <div style={{ textAlign: "center", padding: 40, color: "var(--text-secondary)" }}>暂无邀请码</div>
        ) : (
          <div className="table" style={{ width: "100%" }}>
            <table style={{ width: "100%" }}>
              <thead>
                <tr>
                  <th>邀请码</th>
                  <th>状态</th>
                  <th>创建时间</th>
                  <th>过期时间</th>
                  <th>使用情况</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                {invites.map((inv) => {
                  const expired = isExpired(inv.expires_at);
                  const usedUp = inv.max_uses > 0 && inv.used >= inv.max_uses;
                  const status = inv.revoked ? "已删除" : expired ? "已过期" : usedUp ? "已用完" : "有效";
                  return (
                    <tr key={inv.invite_id}>
                      <td>
                        <code style={{ background: "var(--bg-input)", padding: "4px 8px", borderRadius: 6, fontSize: 12, fontWeight: 700 }}>
                          {inv.code || "-"}
                        </code>
                      </td>
                      <td>
                        <Badge
                          status={inv.revoked ? "error" : expired || usedUp ? "warn" : "success"}
                          text={status}
                        />
                      </td>
                      <td>{formatDate(inv.created_at)}</td>
                      <td>{formatDate(inv.expires_at)}</td>
                      <td>
                        {inv.used} / {inv.max_uses || "∞"}
                      </td>
                      <td>
                        {!inv.revoked && (
                          <Button
                            variant="ghost"
                            onClick={() => handleRevoke(inv.invite_id)}
                            disabled={revoking === inv.invite_id}
                            style={{ color: "var(--error)", display: "flex", alignItems: "center", gap: 4 }}
                          >
                            <Trash2 size={14} />
                            {revoking === inv.invite_id ? "删除中…" : "删除"}
                          </Button>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Card>

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />

      <ConfirmDialog
        open={confirmDialog.open}
        title="删除邀请码"
        message="确定要删除此邀请码吗？删除后该邀请码将无法使用。"
        confirmText="删除"
        cancelText="取消"
        variant="danger"
        onConfirm={confirmRevoke}
        onCancel={() => setConfirmDialog({ open: false, inviteId: null })}
      />
    </div>
  );
};

