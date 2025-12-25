import React, { useMemo, useState } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Server, Activity, Settings, Globe, Shield, Languages, LogOut, KeyRound, X } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../lib/api';

interface LayoutProps {
  children: React.ReactNode;
}

export const MainLayout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { t, language, setLanguage } = useLanguage();
  const { logout } = useAuth();
  const [pwdOpen, setPwdOpen] = useState(false);
  const [oldPwd, setOldPwd] = useState('');
  const [newPwd, setNewPwd] = useState('');
  const [confirmPwd, setConfirmPwd] = useState('');
  const [pwdLoading, setPwdLoading] = useState(false);
  const [pwdError, setPwdError] = useState<string | null>(null);
  const [pwdSuccess, setPwdSuccess] = useState<string | null>(null);

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  const navItems = [
    { path: '/dashboard', label: t('nav.dashboard'), icon: <LayoutDashboard size={20} /> },
    { path: '/devices', label: t('nav.devices'), icon: <Server size={20} /> },
    { path: '/tools', label: t('nav.tools'), icon: <Activity size={20} /> },
    { path: '/frp', label: 'FRP 服务', icon: <Globe size={20} /> },
    { path: '/system', label: t('nav.system'), icon: <Settings size={20} /> },
  ];

  // Helper to check if active (including sub-routes)
  const isActive = (path: string) => location.pathname.startsWith(path);

  const toggleLanguage = () => {
    setLanguage(language === 'en' ? 'zh' : 'en');
  };

  const canSubmitPwd = useMemo(() => {
    return Boolean(oldPwd.trim() && newPwd.trim() && confirmPwd.trim() && !pwdLoading);
  }, [oldPwd, newPwd, confirmPwd, pwdLoading]);

  const openPwdModal = () => {
    setPwdError(null);
    setPwdSuccess(null);
    setOldPwd('');
    setNewPwd('');
    setConfirmPwd('');
    setPwdOpen(true);
  };

  const submitPwd = async () => {
    setPwdError(null);
    setPwdSuccess(null);
    if (!oldPwd.trim() || !newPwd.trim() || !confirmPwd.trim()) {
      setPwdError('请完整填写旧密码、新密码与确认密码');
      return;
    }
    if (newPwd !== confirmPwd) {
      setPwdError('新密码与确认密码不一致');
      return;
    }
    if (newPwd.length < 6) {
      setPwdError('新密码长度至少 6 位');
      return;
    }
    setPwdLoading(true);
    try {
      await api.changePassword({
        old_password: oldPwd,
        new_password: newPwd,
        confirm_password: confirmPwd,
      });
      setPwdSuccess('密码修改成功');
      // 给用户短暂反馈后自动关闭
      window.setTimeout(() => {
        setPwdOpen(false);
      }, 800);
    } catch (e: any) {
      setPwdError(e?.message || '密码修改失败');
    } finally {
      setPwdLoading(false);
    }
  };

  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-logo">
          <div style={{ 
            width: 32, 
            height: 32, 
            background: 'linear-gradient(135deg, #0A59F7 0%, #3275F9 100%)', 
            borderRadius: '10px', 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            marginRight: 12,
            boxShadow: '0 4px 10px rgba(10, 89, 247, 0.3)'
          }}>
            <Shield size={18} color="white" />
          </div>
          <span>NetAdmin</span>
        </div>
        <nav className="nav-menu">
          {navItems.map((item) => (
            <li key={item.path} className="nav-item">
              <NavLink 
                to={item.path} 
                className={({ isActive }) => `nav-link ${isActive ? 'active' : ''}`}
              >
                {item.icon}
                <span>{item.label}</span>
              </NavLink>
            </li>
          ))}
        </nav>
        
        <div style={{ padding: '24px', marginTop: 'auto' }}>
           <div className="glass" style={{ padding: '16px', borderRadius: '16px', display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{ width: 40, height: 40, borderRadius: '50%', background: '#E6E6E6', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                 <span style={{ fontWeight: 600 }}>A</span>
              </div>
              <div style={{ flex: 1 }}>
                 <div style={{ fontWeight: 600, fontSize: '14px' }}>Admin</div>
                 <div style={{ fontSize: '12px', color: 'var(--text-secondary)' }}>System Admin</div>
              </div>
              <button
                className="btn btn-ghost"
                style={{ width: 40, height: 40, padding: 0, borderRadius: '50%' }}
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
          <h1 style={{ margin: 0, fontSize: '24px', fontWeight: 700 }}>
             {navItems.find(i => isActive(i.path))?.label || t('nav.dashboard')}
          </h1>
          <div style={{ display: 'flex', alignItems: 'center', gap: '24px' }}>
            <button onClick={toggleLanguage} className="btn btn-ghost" style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '8px 16px' }}>
              <Languages size={20} />
              <span>{language === 'en' ? '中文' : 'English'}</span>
            </button>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'rgba(65, 186, 65, 0.1)', padding: '6px 12px', borderRadius: '99px' }}>
               <div style={{ width: 8, height: 8, background: '#41BA41', borderRadius: '50%' }}></div>
               <span style={{ fontSize: 13, fontWeight: 600, color: '#41BA41' }}>{t('header.system_online')}</span>
            </div>
            <button onClick={handleLogout} className="btn btn-ghost" style={{ color: 'var(--error)', width: 40, height: 40, padding: 0, borderRadius: '50%' }} title={t('header.logout')}>
              <LogOut size={20} />
            </button>
          </div>
        </header>
        <div className="page-content">
          {children}
        </div>
      </main>

      {pwdOpen && (
        <div className="modal-overlay" onMouseDown={() => setPwdOpen(false)}>
          <div
            className="card glass modal-panel"
            onMouseDown={(e) => e.stopPropagation()}
            style={{ padding: 24 }}
          >
            <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
              <div style={{ fontSize: 18, fontWeight: 700 }}>修改密码</div>
              <button className="btn btn-ghost" style={{ width: 40, height: 40, padding: 0, borderRadius: '50%' }} onClick={() => setPwdOpen(false)} title="关闭">
                <X size={18} />
              </button>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
              <input
                className="input"
                type="password"
                placeholder="旧密码"
                value={oldPwd}
                onChange={(e) => setOldPwd(e.target.value)}
                autoFocus
              />
              <input
                className="input"
                type="password"
                placeholder="新密码（至少 6 位）"
                value={newPwd}
                onChange={(e) => setNewPwd(e.target.value)}
              />
              <input
                className="input"
                type="password"
                placeholder="确认新密码"
                value={confirmPwd}
                onChange={(e) => setConfirmPwd(e.target.value)}
              />

              {pwdError && (
                <div style={{ color: 'var(--error)', fontSize: 13, fontWeight: 600 }}>
                  {pwdError}
                </div>
              )}
              {pwdSuccess && (
                <div style={{ color: 'var(--success)', fontSize: 13, fontWeight: 600 }}>
                  {pwdSuccess}
                </div>
              )}

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 12, marginTop: 8 }}>
                <button className="btn btn-outline" onClick={() => setPwdOpen(false)} disabled={pwdLoading}>
                  取消
                </button>
                <button className="btn btn-primary" onClick={submitPwd} disabled={!canSubmitPwd}>
                  {pwdLoading ? '保存中…' : '保存'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
