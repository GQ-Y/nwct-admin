import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Shield, Languages } from 'lucide-react';
import { Button, Input, Alert } from '../components/UI';
import { useLanguage } from '../contexts/LanguageContext';
import { useAuth } from '../contexts/AuthContext';

export const Login: React.FC = () => {
  const navigate = useNavigate();
  const { t, language, setLanguage } = useLanguage();
  const { login } = useAuth();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [username, setUsername] = useState('admin');
  const [password, setPassword] = useState('admin');

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const isInit = await login(username, password);
      navigate(isInit ? '/dashboard' : '/init');
    } catch (e: any) {
      setError(e?.message || '登录失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ 
      height: '100vh', 
      display: 'flex', 
      alignItems: 'center', 
      justifyContent: 'center', 
      background: 'radial-gradient(circle at 10% 20%, rgb(242, 246, 252) 0%, rgb(220, 230, 245) 90%)',
      position: 'relative',
      overflow: 'hidden'
    }}>
      {/* Decorative background blobs */}
      <div style={{
        position: 'absolute',
        top: '-10%',
        left: '-10%',
        width: '50%',
        height: '50%',
        background: 'rgba(10, 89, 247, 0.1)',
        filter: 'blur(80px)',
        borderRadius: '50%'
      }} />
      <div style={{
        position: 'absolute',
        bottom: '-10%',
        right: '-10%',
        width: '50%',
        height: '50%',
        background: 'rgba(10, 89, 247, 0.1)',
        filter: 'blur(80px)',
        borderRadius: '50%'
      }} />

       <div style={{ position: 'absolute', top: 32, right: 32, zIndex: 10 }}>
         <Button variant="ghost" onClick={() => setLanguage(language === 'en' ? 'zh' : 'en')}>
            <Languages size={20} style={{ marginRight: 8 }}/> {language === 'en' ? '中文' : 'English'}
         </Button>
       </div>
       
      <div className="glass" style={{ 
        width: '440px', 
        padding: '48px', 
        borderRadius: '32px',
        boxShadow: '0 8px 32px rgba(0, 0, 0, 0.08)',
        zIndex: 1
      }}>
        <div style={{ textAlign: 'center', marginBottom: '40px' }}>
          <div style={{ 
            width: 80, 
            height: 80, 
            background: 'linear-gradient(135deg, #0A59F7 0%, #3275F9 100%)', 
            borderRadius: '24px', 
            display: 'flex', 
            alignItems: 'center', 
            justifyContent: 'center',
            margin: '0 auto 24px',
            boxShadow: '0 8px 20px rgba(10, 89, 247, 0.3)'
          }}>
            <Shield size={40} color="white" />
          </div>
          <h2 style={{ fontSize: '28px', fontWeight: 700, margin: '0 0 12px 0', color: 'var(--text-primary)' }}>{t('login.title')}</h2>
          <p style={{ color: 'var(--text-secondary)', margin: 0, fontSize: '16px' }}>{t('login.subtitle')}</p>
        </div>

        {error && <Alert type="error">{error}</Alert>}

        <form onSubmit={handleLogin}>
          <div style={{ marginBottom: '24px' }}>
            <label style={{ display: 'block', marginBottom: '10px', fontWeight: 600, color: 'var(--text-primary)' }}>{t('login.username')}</label>
            <Input
              type="text"
              placeholder="admin"
              required
              style={{ padding: '16px 20px' }}
              value={username}
              onChange={(e) => setUsername((e.target as HTMLInputElement).value)}
            />
          </div>
          <div style={{ marginBottom: '32px' }}>
            <label style={{ display: 'block', marginBottom: '10px', fontWeight: 600, color: 'var(--text-primary)' }}>{t('login.password')}</label>
            <Input
              type="password"
              placeholder="••••••"
              required
              style={{ padding: '16px 20px' }}
              value={password}
              onChange={(e) => setPassword((e.target as HTMLInputElement).value)}
            />
          </div>
          
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: '32px' }}>
             <input type="checkbox" id="remember" style={{ 
               width: 18, 
               height: 18, 
               marginRight: 10, 
               accentColor: 'var(--primary)',
               borderRadius: 4
             }} />
             <label htmlFor="remember" style={{ color: 'var(--text-secondary)', userSelect: 'none' }}>{t('login.remember')}</label>
          </div>
          
          <Button type="submit" style={{ width: '100%', height: '56px', fontSize: '16px' }} disabled={loading}>
            {loading ? t('login.logging_in') : t('login.button')}
          </Button>
        </form>
      </div>
    </div>
  );
};
