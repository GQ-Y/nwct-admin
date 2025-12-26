import React, { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Check, Wifi, Globe, Lock, ArrowLeft, ArrowRight } from 'lucide-react';
import { Button, Input, Card, Alert, Select } from '../components/UI';
import { useLanguage } from '../contexts/LanguageContext';
import { api, getToken } from '../lib/api';
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

  // Step 1: Network
  const [ipMode, setIpMode] = useState<'dhcp' | 'static'>('dhcp');
  const [staticIP, setStaticIP] = useState('');
  const [staticNetmask, setStaticNetmask] = useState('');
  const [staticGateway, setStaticGateway] = useState('');
  const [staticDNS, setStaticDNS] = useState('');
  const [wifiSSID, setWifiSSID] = useState('');
  const [wifiPass, setWifiPass] = useState('');
  const [wifiNetworks, setWifiNetworks] = useState<Array<{ ssid: string; signal?: number; security?: string; in_use?: boolean }>>([]);
  const [wifiScanning, setWifiScanning] = useState(false);
  const [wifiConnecting, setWifiConnecting] = useState(false);
  const [netStatus, setNetStatus] = useState<{
    current_interface?: string;
    ip?: string;
    gateway?: string;
    status?: string;
    latency?: number;
  } | null>(null);
  const [netLoading, setNetLoading] = useState(false);
  const [netError, setNetError] = useState('');

  const steps = [
    { title: t('wizard.step_welcome'), icon: <Globe size={20} /> },
    { title: t('wizard.step_network'), icon: <Wifi size={20} /> },
    { title: t('wizard.step_security'), icon: <Lock size={20} /> },
    { title: t('wizard.step_finish'), icon: <Check size={20} /> },
  ];

  const next = async () => {
    // 网络步骤：先实际下发配置
    if (currentStep === 1) {
      setNetError('');
      try {
        const skipAuth = !getToken();
        await api.networkApply(
          {
            interface: netStatus?.current_interface || undefined,
            ip_mode: ipMode,
            ip: ipMode === 'static' ? staticIP : undefined,
            netmask: ipMode === 'static' ? staticNetmask : undefined,
            gateway: ipMode === 'static' ? staticGateway : undefined,
            dns: ipMode === 'static' ? staticDNS : undefined,
          },
          { skipAuth }
        );
        await refreshNetworkStatus();
      } catch (e: any) {
        setNetError(e?.message || '应用网络配置失败');
        return;
      }
    }
    setCurrentStep((c) => Math.min(c + 1, steps.length - 1));
  };
  const prev = () => setCurrentStep(c => Math.max(c - 1, 0));
  
  const refreshNetworkStatus = async () => {
    setNetError('');
    setNetLoading(true);
    try {
      const skipAuth = !getToken();
      const st = await api.networkStatus({ skipAuth });
      setNetStatus(st || null);
    } catch (e: any) {
      setNetError(e?.message || '获取网络状态失败');
    } finally {
      setNetLoading(false);
    }
  };

  const scanWiFi = async () => {
    setNetError('');
    setWifiScanning(true);
    try {
      const skipAuth = !getToken();
      const res = await api.wifiScan({ allow_redacted: true, skipAuth });
      const list = (res?.networks || []) as any[];
      const normalized = list
        .map((n) => ({
          ssid: String(n?.ssid || '').trim(),
          signal: typeof n?.signal === 'number' ? n.signal : undefined,
          security: n?.security ? String(n.security) : undefined,
          in_use: Boolean(n?.in_use),
        }))
        .filter((n) => n.ssid);
      setWifiNetworks(normalized);
      // 如果当前已连接某个 WiFi，优先选中
      const inUse = normalized.find((n) => n.in_use);
      if (inUse && !wifiSSID) setWifiSSID(inUse.ssid);
    } catch (e: any) {
      setNetError(e?.message || '扫描WiFi失败');
    } finally {
      setWifiScanning(false);
    }
  };

  const connectWiFi = async () => {
    setNetError('');
    const ssid = wifiSSID.trim();
    if (!ssid) {
      setNetError('请选择或输入 WiFi 名称');
      return;
    }
    setWifiConnecting(true);
    try {
      const selected = wifiNetworks.find((n) => n.ssid === ssid);
      const skipAuth = !getToken();
      await api.wifiConnect(
        {
          ssid,
          password: wifiPass,
          security: selected?.security,
          save: true,
        },
        { skipAuth }
      );
      await refreshNetworkStatus();
      // 连接成功后，刷新一次扫描列表（标记 in_use）
      await scanWiFi();
    } catch (e: any) {
      setNetError(e?.message || '连接WiFi失败');
    } finally {
      setWifiConnecting(false);
    }
  };

  // 进入网络步骤时拉取一次状态，并每 5 秒刷新（用户配置时能看到变化）
  useEffect(() => {
    if (currentStep !== 1) return;
    refreshNetworkStatus();
    const tmr = window.setInterval(() => refreshNetworkStatus(), 5000);
    return () => window.clearInterval(tmr);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [currentStep]);

  const wifiOptions = useMemo(() => {
    const opts = wifiNetworks.map((n) => {
      const suffix = [
        n.in_use ? '已连接' : '',
        typeof n.signal === 'number' ? `${n.signal}%` : '',
        n.security ? n.security : '',
      ]
        .filter(Boolean)
        .join(' · ');
      return { value: n.ssid, label: suffix ? `${n.ssid}（${suffix}）` : n.ssid };
    });
    // 允许用户手动输入 SSID：如果不在列表里，也加进 options 以便 Select 显示
    const ssid = wifiSSID.trim();
    if (ssid && !opts.some((o) => o.value === ssid)) {
      opts.unshift({ value: ssid, label: ssid });
    }
    return opts;
  }, [wifiNetworks, wifiSSID]);

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
      const selectedWiFi = wifiNetworks.find((n) => n.ssid === wifiSSID.trim());
      const networkPartial: any = {
        interface: netStatus?.current_interface || '',
        ip_mode: ipMode,
        ip: ipMode === 'static' ? staticIP : '',
        netmask: ipMode === 'static' ? staticNetmask : '',
        gateway: ipMode === 'static' ? staticGateway : '',
        dns: ipMode === 'static' ? staticDNS : '',
        wifi: {
          ssid: wifiSSID.trim(),
          password: wifiPass,
          security: selectedWiFi?.security || '',
        },
        wifi_profiles: [],
      };
      await api.configInit(adminPass, { network: networkPartial });
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
              {netError && <Alert type="error">{netError}</Alert>}
              <div style={{ marginBottom: 24 }}>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.conn_type')}</label>
                <Select
                  value={ipMode}
                  onChange={(v) => setIpMode((v as any) === 'static' ? 'static' : 'dhcp')}
                  options={[
                    { value: 'dhcp', label: 'DHCP（自动获取）' },
                    { value: 'static', label: '静态 IP（仅保存配置）' },
                  ]}
                />
              </div>
              {ipMode === 'static' && (
                <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12, marginBottom: 24 }}>
                  <div>
                    <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>IP</label>
                    <Input value={staticIP} onChange={(e) => setStaticIP((e.target as HTMLInputElement).value)} placeholder="192.168.1.10" />
                  </div>
                  <div>
                    <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>Netmask</label>
                    <Input value={staticNetmask} onChange={(e) => setStaticNetmask((e.target as HTMLInputElement).value)} placeholder="255.255.255.0" />
                  </div>
                  <div>
                    <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>Gateway</label>
                    <Input value={staticGateway} onChange={(e) => setStaticGateway((e.target as HTMLInputElement).value)} placeholder="192.168.1.1" />
                  </div>
                  <div>
                    <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>DNS</label>
                    <Input value={staticDNS} onChange={(e) => setStaticDNS((e.target as HTMLInputElement).value)} placeholder="8.8.8.8" />
                  </div>
                </div>
              )}
              <div style={{ marginBottom: 24 }}>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.wifi_ssid')}</label>
                <div style={{ display: 'flex', gap: 12 }}>
                  <div style={{ flex: 1 }}>
                    <Select
                      value={wifiSSID}
                      onChange={(v) => setWifiSSID(String(v || ''))}
                      options={wifiOptions}
                      placeholder="选择 WiFi"
                    />
                  </div>
                  <Button variant="outline" onClick={scanWiFi} disabled={wifiScanning}>
                    {wifiScanning ? t('common.loading') : t('common.scan')}
                  </Button>
                </div>
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 10, fontWeight: 500 }}>{t('wizard.wifi_pass')}</label>
                <Input
                  type="password"
                  value={wifiPass}
                  onChange={(e) => setWifiPass((e.target as HTMLInputElement).value)}
                  placeholder="如无密码可留空"
                />
                <div style={{ marginTop: 12 }}>
                  <Button onClick={connectWiFi} disabled={wifiConnecting || !wifiSSID.trim()} style={{ width: '100%' }}>
                    {wifiConnecting ? '连接中...' : '连接 WiFi'}
                  </Button>
                </div>
              </div>
            </div>
            <div style={{ background: 'var(--bg-input)', padding: 32, borderRadius: 24 }}>
               <h4 style={{ margin: '0 0 24px 0', fontSize: '18px' }}>{t('wizard.current_status')}</h4>
               <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ color: 'var(--text-secondary)' }}>{t('wizard.status_interface')}</span>
                   <span style={{ fontWeight: 500 }}>{netStatus?.current_interface || '-'}</span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                    <span style={{ color: 'var(--text-secondary)' }}>{t('wizard.status_ip')}</span>
                   <span style={{ fontWeight: 500 }}>{netStatus?.ip || '-'}</span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                   <span style={{ color: 'var(--text-secondary)' }}>{t('wizard.status_gateway')}</span>
                   <span style={{ fontWeight: 500 }}>{netStatus?.gateway || '-'}</span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between' }}>
                   <span style={{ color: 'var(--text-secondary)' }}>{t('wizard.status_latency')}</span>
                   <span style={{ fontWeight: 500 }}>
                     {typeof netStatus?.latency === 'number' && netStatus.latency > 0 ? `${netStatus.latency} ms` : '-'}
                   </span>
                 </div>
                 <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <span style={{ color: 'var(--text-secondary)' }}>{t('wizard.status_status')}</span>
                   {netLoading ? (
                     <span className="badge">刷新中...</span>
                   ) : (
                     <span className={`badge ${netStatus?.status === 'connected' ? 'badge-success' : 'badge-warning'}`}>
                       {netStatus?.status === 'connected' ? t('common.connected') : t('common.disconnected')}
                     </span>
                   )}
                 </div>
               </div>
            </div>
          </div>
        );
      case 2:
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
      case 3:
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
    <div style={{ minHeight: '100vh', background: 'var(--bg-body)', padding: '24px 16px', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
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
                  background: i < currentStep ? 'var(--primary)' : '#E8E8E8' 
                }} />
              )}
            </div>
          ))}
        </div>
        
        <Card style={{ minHeight: 0, display: 'flex', flexDirection: 'column' }}>
          <div style={{ flex: 1, padding: '20px' }}>
            {renderContent()}
          </div>

          {currentStep > 0 && currentStep < steps.length - 1 && (
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
