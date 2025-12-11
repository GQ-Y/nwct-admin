import React, { useState } from 'react';
import { Card, Button, Input } from '../components/UI';
import { Activity, Globe, Zap, Radio, Terminal } from 'lucide-react';
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
      color: active ? 'var(--primary)' : 'var(--text-secondary)',
      fontWeight: active ? 600 : 500,
      position: 'relative',
      transition: 'all 0.3s ease',
      borderRadius: '12px',
      background: active ? 'rgba(10, 89, 247, 0.08)' : 'transparent'
    }}
  >
    {icon} 
    <span>{label}</span>
    {active && (
       <div style={{
          position: 'absolute',
          bottom: 0,
          left: '50%',
          transform: 'translateX(-50%)',
          width: '20px',
          height: '3px',
          background: 'var(--primary)',
          borderRadius: '2px'
       }} />
    )}
  </div>
);

export const Tools: React.FC = () => {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState('ping');
  const [output, setOutput] = useState<string[]>([]);
  const [target, setTarget] = useState('');

  const runTool = () => {
    setOutput([]);
    setTimeout(() => setOutput(prev => [...prev, 'Starting...']), 100);
    setTimeout(() => setOutput(prev => [...prev, 'Sending packets...']), 500);
    setTimeout(() => setOutput(prev => [...prev, 'Reply from 8.8.8.8: bytes=32 time=24ms TTL=118']), 800);
    setTimeout(() => setOutput(prev => [...prev, 'Reply from 8.8.8.8: bytes=32 time=23ms TTL=118']), 1200);
    setTimeout(() => setOutput(prev => [...prev, 'Done.']), 1500);
  };

  return (
    <div>
      <div style={{ 
        display: 'flex', 
        marginBottom: 32, 
        background: 'rgba(255,255,255,0.6)', 
        backdropFilter: 'blur(20px)',
        padding: '6px',
        borderRadius: '16px',
        justifyContent: 'space-between',
        gap: 8,
        flexWrap: 'wrap'
      }}>
        <div style={{ display: 'flex', gap: 8 }}>
            <ToolTab icon={<Activity size={18} />} label={t('tools.ping')} active={activeTab === 'ping'} onClick={() => setActiveTab('ping')} />
            <ToolTab icon={<Globe size={18} />} label={t('tools.traceroute')} active={activeTab === 'trace'} onClick={() => setActiveTab('trace')} />
            <ToolTab icon={<Zap size={18} />} label={t('tools.speedtest')} active={activeTab === 'speed'} onClick={() => setActiveTab('speed')} />
            <ToolTab icon={<Radio size={18} />} label={t('tools.portscan')} active={activeTab === 'port'} onClick={() => setActiveTab('port')} />
        </div>
      </div>

      <div style={{ maxWidth: 900, margin: '0 auto' }}>
        <Card 
            title={
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <Terminal size={20} color="var(--primary)" />
                    {activeTab === 'ping' ? t('tools.ping') : activeTab === 'trace' ? t('tools.traceroute') : activeTab === 'speed' ? t('tools.speedtest') : t('tools.portscan')}
                </div>
            }
            glass
        >
           <div style={{ display: 'flex', gap: 16, marginBottom: 24 }}>
             <Input 
                placeholder={activeTab === 'ping' ? t('tools.enter_ip') : t('tools.target')} 
                value={target}
                onChange={e => setTarget(e.target.value)}
                style={{ flex: 1 }}
             />
             <Button onClick={runTool} style={{ minWidth: 120 }}>{t('common.start')}</Button>
           </div>
           
           <div style={{ 
             background: '#1E1E1E', 
             color: '#E0E0E0', 
             padding: '20px', 
             borderRadius: '16px', 
             minHeight: 360, 
             fontFamily: 'SF Mono, Consolas, Monaco, monospace', 
             fontSize: '14px',
             lineHeight: '1.6',
             boxShadow: 'inset 0 2px 10px rgba(0,0,0,0.2)'
           }}>
             <div style={{ color: '#666', marginBottom: 12 }}>// {t('tools.console_output')}</div>
             {output.map((line, i) => (
               <div key={i} style={{ animation: 'fadeIn 0.2s ease-in' }}>{line}</div>
             ))}
             {output.length === 0 && <div style={{ color: '#555', fontStyle: 'italic' }}>{t('tools.ready')}</div>}
           </div>
        </Card>
      </div>
    </div>
  );
};
