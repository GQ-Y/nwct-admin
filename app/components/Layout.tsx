import React from 'react';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { LayoutDashboard, Server, Activity, Settings, Box, Globe, Shield, Languages, LogOut } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

interface LayoutProps {
  children: React.ReactNode;
}

export const MainLayout: React.FC<LayoutProps> = ({ children }) => {
  const navigate = useNavigate();
  const location = useLocation();
  const { t, language, setLanguage } = useLanguage();

  const handleLogout = () => {
    localStorage.removeItem('isAuthenticated');
    navigate('/login');
  };

  const navItems = [
    { path: '/dashboard', label: t('nav.dashboard'), icon: <LayoutDashboard size={20} /> },
    { path: '/devices', label: t('nav.devices'), icon: <Server size={20} /> },
    { path: '/tools', label: t('nav.tools'), icon: <Activity size={20} /> },
    { path: '/nps', label: t('nav.nps'), icon: <Globe size={20} /> },
    { path: '/mqtt', label: t('nav.mqtt'), icon: <Box size={20} /> },
    { path: '/system', label: t('nav.system'), icon: <Settings size={20} /> },
  ];

  // Helper to check if active (including sub-routes)
  const isActive = (path: string) => location.pathname.startsWith(path);

  const toggleLanguage = () => {
    setLanguage(language === 'en' ? 'zh' : 'en');
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
    </div>
  );
};
