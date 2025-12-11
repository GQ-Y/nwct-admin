
import React from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet } from 'react-router-dom';
import { MainLayout } from './components/Layout';
import { Login } from './pages/Login';
import { InitWizard } from './pages/InitWizard';
import { Dashboard } from './pages/Dashboard';
import { Devices } from './pages/Devices';
import { Tools } from './pages/Tools';
import { System } from './pages/System';
import { NPSPage, MQTTPage } from './pages/Services';
import { LanguageProvider } from './contexts/LanguageContext';

// Protected Route Wrapper
const ProtectedRoute = () => {
  const isAuthenticated = localStorage.getItem('isAuthenticated') === 'true';
  return isAuthenticated ? (
    <MainLayout>
      <Outlet />
    </MainLayout>
  ) : (
    <Navigate to="/login" replace />
  );
};

export const App: React.FC = () => {
  return (
    <LanguageProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/init" element={<InitWizard />} />
          
          {/* Protected Routes */}
          <Route element={<ProtectedRoute />}>
            <Route path="/" element={<Navigate to="/dashboard" replace />} />
            <Route path="/dashboard" element={<Dashboard />} />
            <Route path="/devices" element={<Devices />} />
            <Route path="/tools" element={<Tools />} />
            <Route path="/system" element={<System />} />
            <Route path="/nps" element={<NPSPage />} />
            <Route path="/mqtt" element={<MQTTPage />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </LanguageProvider>
  );
};
