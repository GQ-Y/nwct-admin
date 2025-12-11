
import React, { useState } from 'react';
import { Card, Button, Input } from '../components/UI';
import { Activity, Globe, Zap, Radio } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

const ToolTab: React.FC<{ icon: any, label: string, active: boolean, onClick: () => void }> = ({ icon, label, active, onClick }) => (
  <div 
    onClick={onClick}
    style={{ 
      padding: '12px 24px', 
      cursor: 'pointer', 
      display: 'flex', 
      alignItems: 'center', 
      gap: 8,
      borderBottom: active ? '2px solid var(--primary)' : '2px solid transparent',
      color: active ? 'var(--primary)' : '#666',
      fontWeight: active ? 500 : 400
    }}
  >
    {icon} {label}
  </div>
);

export const Tools: React.FC = () => {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState('ping');
  const [output, setOutput] = useState<string[]>([]);
  const [target, setTarget] = useState('');

  const runTool = () => {
    setOutput(['Starting...', 'Sending packets...', 'Reply from 8.8.8.8: bytes=32 time=24ms TTL=118', 'Reply from 8.8.8.8: bytes=32 time=23ms TTL=118', 'Done.']);
  };

  return (
    <div>
      <div style={{ display: 'flex', borderBottom: '1px solid #f0f0f0', marginBottom: 24, background: 'white', padding: '0 24px' }}>
        <ToolTab icon={<Activity size={18} />} label={t('tools.ping')} active={activeTab === 'ping'} onClick={() => setActiveTab('ping')} />
        <ToolTab icon={<Globe size={18} />} label={t('tools.traceroute')} active={activeTab === 'trace'} onClick={() => setActiveTab('trace')} />
        <ToolTab icon={<Zap size={18} />} label={t('tools.speedtest')} active={activeTab === 'speed'} onClick={() => setActiveTab('speed')} />
        <ToolTab icon={<Radio size={18} />} label={t('tools.portscan')} active={activeTab === 'port'} onClick={() => setActiveTab('port')} />
      </div>

      <div style={{ maxWidth: 800, margin: '0 auto' }}>
        <Card title={`${activeTab === 'ping' ? t('tools.ping') : activeTab === 'trace' ? t('tools.traceroute') : activeTab === 'speed' ? t('tools.speedtest') : t('tools.portscan')}`}>
           <div style={{ display: 'flex', gap: 16, marginBottom: 24 }}>
             <Input 
                placeholder={activeTab === 'ping' ? t('tools.enter_ip') : t('tools.target')} 
                value={target}
                onChange={e => setTarget(e.target.value)}
                style={{ flex: 1 }}
             />
             <Button onClick={runTool}>{t('common.start')}</Button>
           </div>
           
           <div style={{ background: '#1e1e1e', color: '#00ff00', padding: 16, borderRadius: 6, minHeight: 300, fontFamily: 'monospace', fontSize: 13 }}>
             <div style={{ color: '#666', marginBottom: 8 }}>// {t('tools.console_output')}</div>
             {output.map((line, i) => (
               <div key={i}>{line}</div>
             ))}
             {output.length === 0 && <div style={{ color: '#555' }}>{t('tools.ready')}</div>}
           </div>
        </Card>
      </div>
    </div>
  );
};
