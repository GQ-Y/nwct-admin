import React, { useEffect, useMemo, useState } from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Server, Activity, Settings, Globe, Shield, Languages, LogOut, KeyRound, X, Radar, Menu } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { useAuth } from '../contexts/AuthContext';
import { api } from '../lib/api';
import { useIsMobile } from '../lib/useIsMobile';

interface LayoutProps {
  children: React.ReactNode;
}

export const MainLayout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { t, language, setLanguage } = useLanguage();
  const { logout } = useAuth();
  const isMobile = useIsMobile();
  const [drawerOpen, setDrawerOpen] = useState(false);
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
    { path: '/frp', label: t('nav.frp'), icon: <Globe size={20} /> },
    { path: '/public-nodes', label: t('nav.public_nodes'), icon: <Radar size={20} /> },
    { path: '/system', label: t('nav.system'), icon: <Settings size={20} /> },
  ];

  // Helper to check if active (including sub-routes)
  const isActive = (path: string) => location.pathname.startsWith(path);

  const toggleLanguage = () => {
    setLanguage(language === 'en' ? 'zh' : 'en');
  };

  // 手机端：路由变更后自动关闭抽屉
  useEffect(() => {
    if (isMobile) setDrawerOpen(false);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.pathname, isMobile]);

  // 手机端：抽屉打开时锁定背景滚动
  useEffect(() => {
    if (!isMobile) return;
    const prev = document.body.style.overflow;
    if (drawerOpen) document.body.style.overflow = 'hidden';
    return () => {
      document.body.style.overflow = prev;
    };
  }, [drawerOpen, isMobile]);

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
      {isMobile && drawerOpen ? (
        <div
          className="sidebar-overlay"
          onClick={() => setDrawerOpen(false)}
          onMouseDown={() => setDrawerOpen(false)}
        />
      ) : null}

      <aside className={`sidebar ${isMobile ? 'sidebar-drawer' : ''} ${isMobile && drawerOpen ? 'is-open' : ''}`}>
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
          <span>Totoro 智能网关</span>
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
              <button
                className="btn btn-ghost"
                style={{ width: 40, height: 40, padding: 0, borderRadius: '50%', color: 'var(--error)' }}
                title={t('header.logout')}
                onClick={handleLogout}
              >
                <LogOut size={18} />
              </button>
           </div>
        </div>
      </aside>
      
      <main className="main-content">
        <header className="header">
          <div className="header-left">
            {isMobile ? (
              <button
                className="btn btn-ghost mobile-only"
                style={{ width: 40, height: 40, padding: 0, borderRadius: '50%' }}
                onClick={() => setDrawerOpen((v) => !v)}
                aria-label="打开菜单"
                title="菜单"
              >
                <Menu size={20} />
              </button>
            ) : null}
            <h1 style={{ margin: 0, fontSize: '24px', fontWeight: 700 }}>
               {navItems.find(i => isActive(i.path))?.label || t('nav.dashboard')}
            </h1>
          </div>
          <div className="header-right">
            <button
              onClick={toggleLanguage}
              className="btn btn-ghost"
              style={{
                width: isMobile ? 40 : undefined,
                height: isMobile ? 40 : undefined,
                padding: isMobile ? 0 : '8px 16px',
                borderRadius: isMobile ? '50%' : undefined,
                display: 'inline-flex',
                alignItems: 'center',
                justifyContent: 'center',
                gap: 6,
              }}
              title={language === 'en' ? '切换中文' : 'Switch to English'}
              aria-label="切换语言"
            >
              <Languages size={20} />
              {!isMobile ? <span>{language === 'en' ? '中文' : 'English'}</span> : null}
            </button>

            {/* 移动端：在线状态用指示点；桌面端保留文字 pill */}
            {isMobile ? (
              <div
                title={t('header.system_online')}
                aria-label={t('header.system_online')}
                style={{
                  width: 10,
                  height: 10,
                  borderRadius: '50%',
                  background: '#41BA41',
                  boxShadow: '0 0 0 4px rgba(65, 186, 65, 0.14)',
                }}
              />
            ) : (
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', background: 'rgba(65, 186, 65, 0.1)', padding: '6px 12px', borderRadius: '99px' }}>
                 <div style={{ width: 8, height: 8, background: '#41BA41', borderRadius: '50%' }}></div>
                 <span style={{ fontSize: 13, fontWeight: 600, color: '#41BA41' }}>{t('header.system_online')}</span>
              </div>
            )}

            {/* 退出在抽屉里提供；移动端 header 不再占位 */}
            {!isMobile ? (
              <button onClick={handleLogout} className="btn btn-ghost" style={{ color: 'var(--error)', width: 40, height: 40, padding: 0, borderRadius: '50%' }} title={t('header.logout')}>
                <LogOut size={20} />
              </button>
            ) : null}
          </div>
        </header>
        <div className="page-content">
          {children}
        </div>

        {/* 手机端底部导航：6 个核心入口（抽屉通过顶部汉堡打开） */}
        <nav className="bottom-nav mobile-only" role="navigation" aria-label="底部导航">
          <div className="bottom-nav-inner">
            {navItems.map((item) => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) => `bottom-nav-item ${isActive ? 'active' : ''}`}
              >
                <span className="bottom-nav-icon">{item.icon}</span>
                <span className="bottom-nav-label">{item.label}</span>
              </NavLink>
            ))}
          </div>
        </nav>
      </main>

      {pwdOpen && (
        <div className="modal-overlay" onClick={() => setPwdOpen(false)} onMouseDown={() => setPwdOpen(false)}>
          <div
            className="card glass modal-panel"
            onMouseDown={(e) => e.stopPropagation()}
            onClick={(e) => e.stopPropagation()}
            onTouchStart={(e) => e.stopPropagation()}
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
