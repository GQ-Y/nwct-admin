import React, { useMemo } from "react";
import { NavLink, useLocation, useNavigate } from "react-router-dom";
import { LayoutDashboard, Server, Settings, KeyRound, LogOut } from "lucide-react";
import { getAdminToken, clearAdminToken } from "../lib/api";
import { Button } from "./UI";

export const MainLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const hasToken = Boolean((getAdminToken() || "").trim());

  const navItems = [
    { path: "/dashboard", label: "概览", icon: <LayoutDashboard size={20} /> },
    { path: "/config", label: "节点配置", icon: <Settings size={20} /> },
    { path: "/invites", label: "邀请码管理", icon: <KeyRound size={20} /> },
  ];

  const title = useMemo(() => {
    const m = navItems.find((i) => location.pathname.startsWith(i.path));
    return m?.label || "节点管理";
  }, [location.pathname]);

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
            <Server size={18} color="white" />
          </div>
          <span>Totoro Node</span>
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
              N
            </div>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ fontWeight: 800, fontSize: 13 }}>Node Admin</div>
              <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                {hasToken ? "已登录" : "未登录"}
              </div>
            </div>
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
                background: hasToken ? "rgba(65, 186, 65, 0.10)" : "rgba(232, 64, 38, 0.10)",
                padding: "6px 12px",
                borderRadius: 99,
              }}
              title={hasToken ? "已登录" : "未登录"}
            >
              <div
                style={{
                  width: 8,
                  height: 8,
                  background: hasToken ? "#41BA41" : "var(--error)",
                  borderRadius: "50%",
                }}
              />
              <span style={{ fontSize: 12, fontWeight: 800, color: hasToken ? "#41BA41" : "var(--error)" }}>
                {hasToken ? "已登录" : "未登录"}
              </span>
            </div>
            {hasToken && (
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
            )}
          </div>
        </header>

        <div className="page-content">{children}</div>
      </main>
    </div>
  );
};

