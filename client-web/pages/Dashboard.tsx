
import React, { useEffect, useMemo, useState } from 'react';
import { Card, ProgressBar, Badge } from '../components/UI';
import { Activity, HardDrive, Cpu, Network } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';

export const Dashboard: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [fallbackInfo, setFallbackInfo] = useState<any>(null);
  const sys = rt.systemStatus || fallbackInfo;
  const [activities, setActivities] = useState<any[]>([]);

  useEffect(() => {
    api.systemInfo()
      .then((d) => setFallbackInfo(d))
      .catch(() => {});
  }, []);

  useEffect(() => {
    let stop = false;
    const load = () =>
      api.devicesActivity(5)
        .then((d) => {
          if (stop) return;
          setActivities(Array.isArray(d?.activities) ? d.activities : []);
        })
        .catch(() => {});
    load();
    const timer = window.setInterval(load, 8000);
    return () => {
      stop = true;
      window.clearInterval(timer);
    };
  }, []);

  const cpu = useMemo(() => Number(sys?.cpu_usage ?? 0), [sys]);
  const mem = useMemo(() => Number(sys?.memory_usage ?? 0), [sys]);
  const disk = useMemo(() => Number(sys?.disk_usage ?? 0), [sys]);
  const netIp = sys?.network?.ip ?? '-';

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
               <div style={{ fontSize: 24, fontWeight: 600 }}>{cpu.toFixed(1)}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={cpu} color={cpu > 80 ? '#f5222d' : '#1890ff'} /></div>
        </Card>
        
        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(114, 46, 209, 0.1)', borderRadius: 8, color: '#722ed1' }}><Activity /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.memory')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{mem.toFixed(1)}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={mem} color="#722ed1" /></div>
        </Card>

        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(82, 196, 26, 0.1)', borderRadius: 8, color: '#52c41a' }}><HardDrive /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.storage')}</div>
               <div style={{ fontSize: 24, fontWeight: 600 }}>{disk.toFixed(1)}%</div>
            </div>
          </div>
          <div style={{ marginTop: 16 }}><ProgressBar value={disk} color="#52c41a" /></div>
        </Card>

        <Card className="dashboard-stat">
          <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
             <div style={{ padding: 12, background: 'rgba(250, 173, 20, 0.1)', borderRadius: 8, color: '#faad14' }}><Network /></div>
            <div>
               <div style={{ color: '#8c8c8c', fontSize: 12 }}>{t('dashboard.net_io')}</div>
               <div style={{ fontSize: 18, fontWeight: 600 }}>{netIp}</div>
            </div>
          </div>
          <div style={{ marginTop: 16, fontSize: 12, color: '#8c8c8c' }}>
            WS: {rt.connected ? t('common.online') : t('common.offline')}
          </div>
        </Card>
      </div>

      <div className="grid-2">
        <Card title={t('dashboard.service_status')}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.nps_client')}</span>
              <Badge status={rt.npsStatus?.connected ? 'online' : 'offline'} text={rt.npsStatus?.connected ? t('common.connected') : t('common.disconnected')} />
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', padding: '12px', background: '#f9f9f9', borderRadius: 4 }}>
              <span>{t('dashboard.mqtt_broker')}</span>
              <Badge status={rt.mqttLogNew ? 'online' : 'warn'} text={t('common.online')} />
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
             {activities.map((a, idx) => {
               const ts = a?.timestamp ? new Date(a.timestamp) : null;
               const timeLabel = ts && !Number.isNaN(ts.getTime()) ? ts.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '--:--';
               const ip = a?.ip || '-';
               const st = String(a?.status || '').toLowerCase();
               const msg = st === 'online' ? t('dashboard.device_connected', { ip }) : t('dashboard.device_disconnected', { ip });
               const name = a?.name ? String(a.name) : '';
               const vendorRaw = a?.vendor ? String(a.vendor) : '';
               const vendor = vendorRaw && vendorRaw.toLowerCase() !== 'unknown' ? vendorRaw : '';
               const model = a?.model ? String(a.model) : '';
               const meta = [name, vendor, model].filter(Boolean).join(' · ');
               return (
                 <li key={`${a?.timestamp || idx}-${idx}`} style={{ padding: '12px 0', borderBottom: '1px solid #f0f0f0', display: 'flex', gap: 12 }}>
                   <div style={{ color: '#8c8c8c', fontSize: 12, minWidth: 70 }}>{timeLabel}</div>
                   <div style={{ flex: 1 }}>
                     <div>{msg}</div>
                     {meta ? (
                       <div style={{ color: '#8c8c8c', fontSize: 12, marginTop: 4 }}>
                         {meta}
                       </div>
                     ) : null}
                   </div>
                 </li>
               );
             })}
             {activities.length === 0 ? (
               <li style={{ padding: '12px 0', color: '#8c8c8c', fontSize: 13 }}>暂无活动</li>
             ) : null}
           </ul>
        </Card>
      </div>
    </div>
  );
};
