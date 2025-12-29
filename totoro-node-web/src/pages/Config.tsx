import React, { useEffect, useState } from "react";
import { api, setAdminToken } from "../lib/api";
import { Card, Button, Input, Checkbox } from "../components/UI";
import { Toast } from "../components/Toast";

export const Config: React.FC = () => {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [changingPwd, setChangingPwd] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  const [pwdForm, setPwdForm] = useState({ old_password: "", new_password: "", confirm_password: "" });
  const [showPwdForm, setShowPwdForm] = useState(false);

  const [config, setConfig] = useState({
    public: false,
    name: "",
    description: "",
    region: "",
    isp: "",
    domain_suffix: "",
    http_enabled: false,
    https_enabled: false,
  });

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    setLoading(true);
    try {
      const cfg = await api.getNodeConfig();
      setConfig({
        public: cfg.public || false,
        name: cfg.name || "",
        description: cfg.description || "",
        region: cfg.region || "",
        isp: cfg.isp || "",
        domain_suffix: cfg.domain_suffix || "",
        http_enabled: cfg.http_enabled || false,
        https_enabled: cfg.https_enabled || false,
      });
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "加载配置失败" });
    } finally {
      setLoading(false);
    }
  };

  const saveConfig = async () => {
    setSaving(true);
    try {
      // 不传递 bridge_url（后端不允许修改）
      await api.updateNodeConfig({
        public: config.public,
        name: config.name,
        description: config.description,
        region: config.region,
        isp: config.isp,
        domain_suffix: config.domain_suffix,
        http_enabled: config.http_enabled,
        https_enabled: config.https_enabled,
      });
      setToast({ open: true, type: "success", message: "配置保存成功" });
      // 重新加载配置
      await loadConfig();
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "保存配置失败" });
    } finally {
      setSaving(false);
    }
  };

  const handleChangePassword = async () => {
    const oldPwd = pwdForm.old_password.trim();
    const newPwd = pwdForm.new_password.trim();
    const confirmPwd = pwdForm.confirm_password.trim();
    if (!oldPwd || !newPwd || !confirmPwd) {
      setToast({ open: true, type: "error", message: "请完整填写所有字段" });
      return;
    }
    if (newPwd !== confirmPwd) {
      setToast({ open: true, type: "error", message: "新密码与确认密码不一致" });
      return;
    }
    setChangingPwd(true);
    try {
      const res = await api.adminChangePassword({ old_password: oldPwd, new_password: newPwd });
      setAdminToken(res.token);
      setToast({ open: true, type: "success", message: "密码修改成功" });
      setShowPwdForm(false);
      setPwdForm({ old_password: "", new_password: "", confirm_password: "" });
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "修改失败" });
    } finally {
      setChangingPwd(false);
    }
  };

  return (
    <div>
      <Card
        title="密码管理"
        extra={
          <Button variant="outline" onClick={() => setShowPwdForm(!showPwdForm)}>
            {showPwdForm ? "取消" : "修改密码"}
          </Button>
        }
      >
        {showPwdForm && (
          <div
            className="glass"
            style={{
              padding: 18,
              borderRadius: 16,
              marginBottom: 18,
              border: "1px solid rgba(255,255,255,0.65)",
            }}
          >
            <div style={{ fontSize: 14, fontWeight: 800, marginBottom: 12 }}>修改密码</div>
            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>当前密码</div>
                <Input
                  type="password"
                  placeholder="请输入当前密码"
                  value={pwdForm.old_password}
                  onChange={(e) => setPwdForm({ ...pwdForm, old_password: (e.target as any).value })}
                />
              </div>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>新密码</div>
                <Input
                  type="password"
                  placeholder="请输入新密码（至少 8 位）"
                  value={pwdForm.new_password}
                  onChange={(e) => setPwdForm({ ...pwdForm, new_password: (e.target as any).value })}
                />
              </div>
              <div>
                <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>确认新密码</div>
                <Input
                  type="password"
                  placeholder="请再次输入新密码"
                  value={pwdForm.confirm_password}
                  onChange={(e) => setPwdForm({ ...pwdForm, confirm_password: (e.target as any).value })}
                />
              </div>
              <div style={{ display: "flex", gap: 10, justifyContent: "flex-end" }}>
                <Button variant="outline" onClick={() => setShowPwdForm(false)} disabled={changingPwd}>
                  取消
                </Button>
                <Button variant="primary" onClick={handleChangePassword} disabled={changingPwd}>
                  {changingPwd ? "修改中…" : "修改"}
                </Button>
              </div>
            </div>
          </div>
        )}
      </Card>

      <Card title="节点配置">
        <div style={{ display: "flex", flexDirection: "column", gap: 16 }}>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>节点名称</div>
            <Input
              placeholder="节点名称"
              value={config.name}
              onChange={(e) => setConfig({ ...config, name: (e.target as any).value })}
            />
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>描述</div>
            <Input
              placeholder="节点描述（纯文本）"
              value={config.description}
              onChange={(e) => setConfig({ ...config, description: (e.target as any).value })}
            />
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>区域</div>
            <Input
              placeholder="区域"
              value={config.region}
              onChange={(e) => setConfig({ ...config, region: (e.target as any).value })}
            />
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>ISP</div>
            <Input placeholder="ISP" value={config.isp} onChange={(e) => setConfig({ ...config, isp: (e.target as any).value })} />
          </div>
          <div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", fontWeight: 800, marginBottom: 6 }}>域名后缀</div>
            <Input
              placeholder="例如 frpc.example.com"
              value={config.domain_suffix}
              onChange={(e) => setConfig({ ...config, domain_suffix: (e.target as any).value })}
            />
          </div>
          <div style={{ display: "flex", gap: 24, alignItems: "center", flexWrap: "wrap" }}>
            <Checkbox
              checked={config.public}
              onChange={(checked) => setConfig({ ...config, public: checked })}
              label="公开节点"
            />
            <Checkbox
              checked={config.http_enabled}
              onChange={(checked) => setConfig({ ...config, http_enabled: checked })}
              label="启用 HTTP"
            />
            <Checkbox
              checked={config.https_enabled}
              onChange={(checked) => setConfig({ ...config, https_enabled: checked })}
              label="启用 HTTPS"
            />
          </div>
          <div style={{ display: "flex", gap: 10, justifyContent: "flex-end" }}>
            <Button variant="outline" onClick={loadConfig} disabled={loading || saving}>
              {loading ? "加载中…" : "刷新"}
            </Button>
            <Button variant="primary" onClick={saveConfig} disabled={loading || saving}>
              {saving ? "保存中…" : "保存配置"}
            </Button>
          </div>
        </div>
      </Card>

      <Toast open={toast.open} type={toast.type} message={toast.message} onClose={() => setToast((t) => ({ ...t, open: false }))} />
    </div>
  );
};

