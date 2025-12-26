import React from "react";
import { BrowserRouter, Navigate, Outlet, Route, Routes } from "react-router-dom";
import { MainLayout } from "./components/Layout";
import { Dashboard } from "./pages/Dashboard";
import { PublicNodesPage } from "./pages/PublicNodes";
import { OfficialNodesPage } from "./pages/OfficialNodes";
import { DeviceWhitelistPage } from "./pages/DeviceWhitelist";
import { LoginPage } from "./pages/Login";
import { getAdminToken } from "./lib/api";

const ProtectedRoute = () => {
  const hasTok = Boolean((getAdminToken() || "").trim());
  return hasTok ? (
    <MainLayout>
      <Outlet />
    </MainLayout>
  ) : (
    <Navigate to="/login" replace />
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
          <Route path="/public-nodes" element={<PublicNodesPage />} />
          <Route path="/official-nodes" element={<OfficialNodesPage />} />
          <Route path="/device-whitelist" element={<DeviceWhitelistPage />} />
        </Route>
        <Route path="*" element={<Navigate to="/dashboard" replace />} />
      </Routes>
    </BrowserRouter>
  );
};


