import React, { useMemo, useState } from "react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { Globe, KeyRound, LayoutDashboard, ListChecks, LogOut, Server, Shield, X } from "lucide-react";
import { Button, Card, Input } from "./UI";
import { api, clearAdminToken, getAdminToken, setAdminToken } from "../lib/api";
import { Toast } from "./Toast";

export const MainLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const hasTok = Boolean((getAdminToken() || "").trim());

  const [pwdOpen, setPwdOpen] = useState(false);
  const [oldPwd, setOldPwd] = useState("");
  const [newPwd, setNewPwd] = useState("");
  const [confirmPwd, setConfirmPwd] = useState("");
  const [saving, setSaving] = useState(false);
  const [toast, setToast] = useState<{ open: boolean; type: any; message: string }>({ open: false, type: "info", message: "" });

  const navItems = [
    { path: "/dashboard", label: "概览", icon: <LayoutDashboard size={20} /> },
    { path: "/public-nodes", label: "公共节点", icon: <Globe size={20} /> },
    { path: "/official-nodes", label: "官方节点", icon: <Server size={20} /> },
    { path: "/device-whitelist", label: "设备白名单", icon: <ListChecks size={20} /> },
  ];

  const title = useMemo(() => {
    const m = navItems.find((i) => location.pathname.startsWith(i.path));
    return m?.label || "桥梁运维";
  }, [location.pathname]);

  const openPwdModal = () => {
    setOldPwd("");
    setNewPwd("");
    setConfirmPwd("");
    setPwdOpen(true);
  };

  const submitChangePassword = async () => {
    const o = oldPwd.trim();
    const n = newPwd.trim();
    const c = confirmPwd.trim();
    if (!o || !n || !c) {
      setToast({ open: true, type: "error", message: "请完整填写旧密码、新密码与确认密码" });
      return;
    }
    if (n !== c) {
      setToast({ open: true, type: "error", message: "新密码与确认密码不一致" });
      return;
    }
    setSaving(true);
    try {
      const res = await api.adminChangePassword({ old_password: o, new_password: n });
      setAdminToken(res.token);
      setToast({ open: true, type: "success", message: "密码修改成功" });
      setPwdOpen(false);
    } catch (e: any) {
      setToast({ open: true, type: "error", message: e?.message || "修改失败" });
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-logo">
          <div
            style={{
              width: 32,
              height: 32,
              background: "linear-gradient(135deg, #0A59F7 0%, #3275F9 100%)",
              borderRadius: 10,
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              marginRight: 12,
              boxShadow: "0 4px 10px rgba(10, 89, 247, 0.3)",
              flex: "0 0 auto",
            }}
          >
            <Shield size={18} color="white" />
          </div>
          <span>Totoro Bridge</span>
        </div>

        <nav className="nav-menu">
          {navItems.map((item) => (
            <li key={item.path} className="nav-item">
              <NavLink to={item.path} className={({ isActive }) => `nav-link ${isActive ? "active" : ""}`}>
                {item.icon}
                <span>{item.label}</span>
              </NavLink>
            </li>
          ))}
        </nav>

        <div style={{ padding: 18, marginTop: "auto" }}>
          <div className="glass" style={{ padding: 14, borderRadius: 16, display: "flex", alignItems: "center", gap: 12 }}>
            <div
              style={{
                width: 40,
                height: 40,
                borderRadius: "50%",
                background: "#E6E6E6",
                display: "flex",
                alignItems: "center",
                justifyContent: "center",
                fontWeight: 800,
              }}
            >
              B
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontWeight: 800, fontSize: 13 }}>Bridge Admin</div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>{hasTok ? "已登录" : "未登录"}</div>
            </div>
            <button
              className="btn btn-ghost"
              style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }}
              title="修改密码"
              onClick={openPwdModal}
            >
              <KeyRound size={18} />
            </button>
          </div>
        </div>
      </aside>

      <main className="main-content">
        <header className="header">
          <h1 style={{ margin: 0, fontSize: 22, fontWeight: 900 }}>{title}</h1>
          <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
            <div
              style={{
                display: "flex",
                alignItems: "center",
                gap: 8,
                background: hasTok ? "rgba(65, 186, 65, 0.10)" : "rgba(232, 64, 38, 0.10)",
                padding: "6px 12px",
                borderRadius: 99,
              }}
              title={hasTok ? "已登录" : "未登录"}
            >
              <div
                style={{
                  width: 8,
                  height: 8,
                  background: hasTok ? "#41BA41" : "var(--error)",
                  borderRadius: "50%",
                }}
              />
              <span style={{ fontSize: 12, fontWeight: 800, color: hasTok ? "#41BA41" : "var(--error)" }}>
                {hasTok ? "已登录" : "未登录"}
              </span>
            </div>
            <Button variant="outline" onClick={openPwdModal} style={{ height: 40 }}>
              修改密码
            </Button>
            <button
              className="btn btn-ghost"
              style={{ width: 40, height: 40, padding: 0, borderRadius: "50%", color: "var(--error)" }}
              title="退出登录"
              onClick={() => {
                clearAdminToken();
                navigate("/login", { replace: true });
              }}
            >
              <LogOut size={18} />
            </button>
          </div>
        </header>

        <div className="page-content">{children}</div>
      </main>

      {pwdOpen && (
        <div className="modal-overlay" onMouseDown={() => setPwdOpen(false)}>
          <div className="card glass modal-panel" onMouseDown={(e) => e.stopPropagation()} style={{ padding: 22 }}>
            <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
              <div style={{ fontSize: 18, fontWeight: 900 }}>修改密码</div>
              <button className="btn btn-ghost" style={{ width: 40, height: 40, padding: 0, borderRadius: "50%" }} onClick={() => setPwdOpen(false)}>
                <X size={18} />
              </button>
            </div>

            <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
              <Input type="password" placeholder="旧密码" value={oldPwd} onChange={(e) => setOldPwd((e.target as any).value)} autoFocus />
              <Input type="password" placeholder="新密码（至少 8 位）" value={newPwd} onChange={(e) => setNewPwd((e.target as any).value)} />
              <Input type="password" placeholder="确认新密码" value={confirmPwd} onChange={(e) => setConfirmPwd((e.target as any).value)} />
            </div>

            <div style={{ display: "flex", justifyContent: "flex-end", gap: 10, marginTop: 16 }}>
              <Button variant="outline" onClick={() => setPwdOpen(false)} disabled={saving}>
                取消
              </Button>
              <Button variant="primary" onClick={submitChangePassword} disabled={saving}>
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



