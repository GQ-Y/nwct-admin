import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Check, Wifi, Globe, Lock, Server, ArrowLeft, ArrowRight } from 'lucide-react';
import { Button, Input, Card, Alert } from '../components/UI';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useAuth } from '../contexts/AuthContext';

export const InitWizard: React.FC = () => {
  const navigate = useNavigate();
  const { t } = useLanguage();
  const { refreshInitStatus } = useAuth();
  const [currentStep, setCurrentStep] = useState(0);
  const [adminPass, setAdminPass] = useState('');
  const [adminPass2, setAdminPass2] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');

  const steps = [
    { title: t('wizard.step_welcome'), icon: <Globe size={20} /> },
    { title: t('wizard.step_network'), icon: <Wifi size={20} /> },
    { title: t('wizard.step_services'), icon: <Server size={20} /> },
    { title: t('wizard.step_security'), icon: <Lock size={20} /> },
    { title: t('wizard.step_finish'), icon: <Check size={20} /> },
  ];

  const next = () => setCurrentStep(c => Math.min(c + 1, steps.length - 1));
  const prev = () => setCurrentStep(c => Math.max(c - 1, 0));
  
  const finish = async () => {
    setError('');
    if (!adminPass || adminPass.length < 4) {
      setError('管理员密码至少 4 位');
      return;
    }
    if (adminPass !== adminPass2) {
      setError('两次输入的密码不一致');
      return;
    }
    setSubmitting(true);
    try {
      await api.configInit(adminPass);
      await refreshInitStatus();
      navigate('/dashboard');
    } catch (e: any) {
      setError(e?.message || '初始化失败');
    } finally {
      setSubmitting(false);
    }
  };

  const renderContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <div style={{ textAlign: 'center', padding: '60px 0' }}>
            <h2 style={{ fontSize: '32px', fontWeight: 700, marginBottom: '20px', color: 'var(--text-primary)' }}>{t('wizard.welcome_title')}</h2>
            <p style={{ color: 'var(--text-secondary)', maxWidth: '480px', margin: '0 auto 40px', fontSize: '16px', lineHeight: 1.6 }}>
              {t('wizard.welcome_desc')}
            </p>
            <Button onClick={next} style={{ padding: '0 40px', height: '56px' }}>{t('wizard.get_started')}</Button>
          </div>
        );
      case 1:
        return (
          <div className="grid-2">
            <div>
              <h3 style={{ marginBottom: 24, fontSize: '20px', fontWeight: 600 }}>{t('wizard.network_settings')}</h3>
              <div style={{ marginBottom: 24 }}>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.conn_type')}</label>
                <select className="input" style={{ width: '100%', height: '52px' }}>
                   <option>DHCP (Auto)</option>
                   <option>Static IP</option>
                </select>
              </div>
              <div style={{ marginBottom: 24 }}>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.wifi_ssid')}</label>
                <div style={{ display: 'flex', gap: 12 }}>
                  <Input placeholder="Select Network" />
                  <Button variant="outline">{t('common.scan')}</Button>
                </div>
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.wifi_pass')}</label>
                <Input type="password" />
              </div>
            </div>
            <div style={{ background: 'var(--bg-input)', padding: 32, borderRadius: 24 }}>
               <h4 style={{ margin: '0 0 24px 0', fontSize: '18px' }}>{t('wizard.current_status')}</h4>
               <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                   <span style={{ color: 'var(--text-secondary)' }}>Interface</span>
                   <span style={{ fontWeight: 500 }}>eth0</span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                   <span style={{ color: 'var(--text-secondary)' }}>IP Address</span>
                   <span style={{ fontWeight: 500 }}>192.168.1.100</span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                   <span style={{ color: 'var(--text-secondary)' }}>Status</span>
                   <span className="badge badge-success">{t('common.connected')}</span>
                 </div>
               </div>
            </div>
          </div>
        );
      case 2:
        return (
          <div className="grid-2">
             <Card title={t('wizard.nps_config')} className="glass" style={{ margin: 0 }}>
                <div style={{ marginBottom: 24 }}><label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.server_addr')}</label><Input placeholder="nps.example.com" /></div>
                <div><label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.vkey')}</label><Input placeholder="Secret Key" /></div>
             </Card>
             <Card title={t('wizard.mqtt_config')} className="glass" style={{ margin: 0 }}>
                <div style={{ marginBottom: 24 }}><label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('services.host')}</label><Input placeholder="mqtt.example.com" /></div>
                <div><label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.client_id')}</label><Input placeholder="device-001" /></div>
             </Card>
          </div>
        );
      case 3:
        return (
          <div style={{ maxWidth: 480, margin: '0 auto' }}>
            <h3 style={{ marginBottom: 32, textAlign: 'center', fontSize: '24px' }}>{t('wizard.admin_security')}</h3>
            {error && <Alert type="error">{error}</Alert>}
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.new_pass')}</label>
              <Input
                type="password"
                value={adminPass}
                onChange={(e) => setAdminPass((e.target as HTMLInputElement).value)}
              />
            </div>
            <div style={{ marginBottom: 24 }}>
              <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.confirm_pass')}</label>
              <Input
                type="password"
                value={adminPass2}
                onChange={(e) => setAdminPass2((e.target as HTMLInputElement).value)}
              />
            </div>
          </div>
        );
      case 4:
         return (
           <div style={{ textAlign: 'center', padding: '60px 0' }}>
             <div style={{ width: 88, height: 88, background: 'rgba(65, 186, 65, 0.1)', borderRadius: '50%', color: 'var(--success)', display: 'flex', alignItems: 'center', justifyContent: 'center', margin: '0 auto 32px' }}>
                <Check size={40} />
             </div>
             <h2 style={{ fontSize: '28px', marginBottom: '16px' }}>{t('wizard.setup_complete')}</h2>
             <p style={{ margin: '0 auto 40px', color: 'var(--text-secondary)', maxWidth: 400 }}>{t('wizard.setup_desc')}</p>
             {error && <Alert type="error">{error}</Alert>}
             <Button onClick={finish} disabled={submitting} style={{ padding: '0 48px', height: '56px' }}>
               {submitting ? t('common.loading') : t('wizard.go_dashboard')}
             </Button>
           </div>
         );
      default: return null;
    }
  };

  return (
    <div style={{ minHeight: '100vh', background: 'var(--bg-body)', padding: '40px 20px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <div style={{ width: '100%', maxWidth: '1000px' }}>
        <div className="wizard-steps">
          {steps.map((s, i) => (
            <div key={i} className={`step ${i <= currentStep ? 'active' : ''}`} style={{ flex: 1 }}>
              <div className="step-number">
                {i < currentStep ? <Check size={20} /> : (s.icon || i + 1)}
              </div>
              <span>{s.title}</span>
              {i < steps.length - 1 && (
                <div className="step-line" style={{ 
                  left: '50%', 
                  width: '100%', 
                  background: i < currentStep ? 'var(--primary)' : '#E8E8E8' 
                }} />
              )}
            </div>
          ))}
        </div>
        
        <Card style={{ minHeight: 600, display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1, padding: '20px' }}>
            {renderContent()}
          </div>

          {currentStep > 0 && currentStep < 4 && (
            <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: 40, paddingTop: 32, borderTop: '1px solid #F0F0F0' }}>
              <Button variant="ghost" onClick={prev}><ArrowLeft size={20} /> {t('wizard.prev')}</Button>
              <Button onClick={next}>{t('wizard.next')} <ArrowRight size={20} /></Button>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
};
