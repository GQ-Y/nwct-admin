
import React, { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, Outlet, useNavigate, useLocation } from 'react-router-dom';
import { MainLayout } from './components/Layout';
import { Login } from './pages/Login';
import { InitWizard } from './pages/InitWizard';
import { Dashboard } from './pages/Dashboard';
import { Devices } from './pages/Devices';
import { Tools } from './pages/Tools';
import { System } from './pages/System';
import { FRPPage } from './pages/Services';
import { PublicNodesPage } from './pages/PublicNodes';
import { LanguageProvider } from './contexts/LanguageContext';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { RealtimeProvider } from './contexts/RealtimeContext';

// Protected Route Wrapper
const ProtectedRoute = () => {
  const { token } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();

  // 监听 token 变化，当 token 失效时自动跳转到登录页
  useEffect(() => {
    if (!token && location.pathname !== '/login' && location.pathname !== '/init') {
      navigate('/login', { replace: true });
    }
  }, [token, navigate, location.pathname]);

  // 监听 token 清除事件
  useEffect(() => {
    const handleTokenCleared = () => {
      if (location.pathname !== '/login' && location.pathname !== '/init') {
        navigate('/login', { replace: true });
      }
    };

    window.addEventListener('token-cleared', handleTokenCleared);
    return () => {
      window.removeEventListener('token-cleared', handleTokenCleared);
    };
  }, [navigate, location.pathname]);

  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return (
    <RealtimeProvider enabled={true}>
      <MainLayout>
        <Outlet />
      </MainLayout>
    </RealtimeProvider>
  );
};

export const App: React.FC = () => {
  return (
    <LanguageProvider>
      <AuthProvider>
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
              <Route path="/frp" element={<FRPPage />} />
              <Route path="/public-nodes" element={<PublicNodesPage />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </AuthProvider>
    </LanguageProvider>
  );
};
