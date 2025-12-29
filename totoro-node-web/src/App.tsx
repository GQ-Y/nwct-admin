import React, { useEffect } from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes, useNavigate } from "react-router-dom";
import { MainLayout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { Config } from "./pages/Config";
import { Invites } from "./pages/Invites";
import { LoginPage } from "./pages/Login";
import { getAdminToken } from "./lib/api";
import { api } from "./lib/api";

const ProtectedRoute = () => {
  const navigate = useNavigate();

  useEffect(() => {
    // 检查是否需要登录（仅在首次加载时检查）
    const checkAuth = async () => {
      const hasToken = Boolean((getAdminToken() || "").trim());
      try {
        await api.getNodeConfig();
        // 成功，说明 token 有效或不需要认证
      } catch (e: any) {
        // 如果返回 401，说明需要登录
        if (e?.message?.includes("unauthorized") || e?.message?.includes("401")) {
          if (!hasToken) {
            navigate("/login", { replace: true });
          }
        }
      }
    };
    checkAuth();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <MainLayout>
      <Outlet />
    </MainLayout>
  );
};

export const App: React.FC = () => {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/config" element={<Config />} />
          <Route path="/invites" element={<Invites />} />
        </Route>
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  );
};

