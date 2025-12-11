
import React from 'react';
import { Card, ProgressBar, Badge } from '../components/UI';
import { mockStats } from '../data';
import { Activity, HardDrive, Cpu, Network } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const Dashboard: React.FC = () => {
  const { t } = useLanguage();

  return (
    <div>
      <h2 style={{ marginTop: 0, marginBottom: 24 }}>{t('dashboard.overview')}</h2>
      
      {/* Status Cards */}
      <div className="grid-4" style={{ marginBottom: 24 }}>
        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
            <div style={{ padding: 12, background: 'rgba(24, 144, 255, 0.1)', borderRadius: 8, color: 'var(--primary)' }}><Cpu /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.cpu')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{mockStats.cpu}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={mockStats.cpu} color={mockStats.cpu > 80 ? '#f5222d' : '#1890ff'} /></div>
        </Card>
        
        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(114, 46, 209, 0.1)', borderRadius: 8, color: '#722ed1' }}><Activity /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.memory')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{mockStats.memory}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={mockStats.memory} color="#722ed1" /></div>
        </Card>

        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(82, 196, 26, 0.1)', borderRadius: 8, color: '#52c41a' }}><HardDrive /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.storage')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{mockStats.storage}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={mockStats.storage} color="#52c41a" /></div>
        </Card>

        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(250, 173, 20, 0.1)', borderRadius: 8, color: '#faad14' }}><Network /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.net_io')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{mockStats.netIn}M</div>
            </div>
          </div>
          <div style={{ marginTop: 16, fontSize: 12, color: '#8c8c8c' }}>
            Up: {mockStats.netOut} MB/s
          </div>
        </Card>
      </div>

      <div className="grid-2">
        <Card title={t('dashboard.service_status')}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.nps_client')}</span>
              <Badge status="online" text={t('common.connected')} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.mqtt_broker')}</span>
              <Badge status="online" text={t('common.connected')} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.web_server')}</span>
              <Badge status="online" text={t('common.online')} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.ssh_service')}</span>
              <Badge status="online" text={t('common.online')} />
            </div>
          </div>
        </Card>

        <Card title={t('dashboard.recent_activity')}>
           <ul style={{ padding: 0, margin: 0, listStyle: 'none' }}>
             {[1,2,3,4].map(i => (
               <li key={i} style={{ padding: '12px 0', borderBottom: '1px solid #f0f0f0', display: 'flex', gap: 12 }}>
                 <div style={{ color: '#8c8c8c', fontSize: 12, minWidth: 60 }}>10:3{i} AM</div>
                 <div>{t('dashboard.device_connected', { ip: `192.168.1.10${i}` })}</div>
               </li>
             ))}
           </ul>
        </Card>
      </div>
    </div>
  );
};
