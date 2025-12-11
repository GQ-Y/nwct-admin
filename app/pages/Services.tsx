
import React from 'react';
import { Card, Button, Input, Badge } from '../components/UI';
import { mockMqttMessages, mockTunnels } from '../data';
import { Pause, Save } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const NPSPage: React.FC = () => {
  const { t } = useLanguage();

  return (
    <div>
      <div className="grid-2" style={{ marginBottom: 24 }}>
        <Card title={t('services.nps_conn')}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 }}>
             <div style={{ display: 'flex', gap: 16 }}>
               <div style={{ width: 12, height: 12, borderRadius: '50%', background: '#52c41a', marginTop: 6 }} />
               <div>
                 <div style={{ fontSize: 18, fontWeight: 600 }}>{t('common.connected')}</div>
                 <div style={{ color: '#666' }}>server.nps.host:8024</div>
               </div>
             </div>
             <Button variant="outline"><Pause size={16} /> {t('common.disconnected')}</Button>
          </div>
          <div style={{ background: '#f5f5f5', padding: 16, borderRadius: 6 }}>
            <div style={{ fontSize: 12, color: '#666', marginBottom: 8 }}>{t('services.config')}</div>
            <div className="grid-2">
              <Input defaultValue="server.nps.host:8024" />
              <Input defaultValue="my-vkey-token" type="password" />
            </div>
            <Button style={{ marginTop: 12 }} variant="primary"><Save size={16} /> {t('services.save_config')}</Button>
          </div>
        </Card>
        
        <Card title={t('services.tunnel_stats')}>
           <div style={{ fontSize: 32, fontWeight: 600, color: '#1890ff' }}>3</div>
           <div style={{ color: '#666' }}>{t('services.active_tunnels')}</div>
        </Card>
      </div>

      <Card title={t('services.tunnels_list')}>
        <table className="table">
          <thead><tr><th>{t('services.id')}</th><th>{t('devices.type')}</th><th>{t('services.local')}</th><th>{t('services.remote_port')}</th><th>{t('devices.status')}</th></tr></thead>
          <tbody>
            {mockTunnels.map(tunnel => (
              <tr key={tunnel.id}>
                <td>{tunnel.id}</td>
                <td>{tunnel.type.toUpperCase()}</td>
                <td>{tunnel.local}</td>
                <td>{tunnel.remote}</td>
                <td><Badge status={tunnel.status} text={t(`common.${tunnel.status}`)} /></td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
};

export const MQTTPage: React.FC = () => {
  const { t } = useLanguage();

  return (
    <div>
       <div className="grid-2" style={{ marginBottom: 24 }}>
         <Card title={t('services.broker_conn')}>
            <div style={{ marginBottom: 16 }}>
              <label style={{ display: 'block', marginBottom: 8 }}>{t('services.host')}</label>
              <Input defaultValue="mqtt.server.io" />
            </div>
            <div className="grid-2" style={{ marginBottom: 16 }}>
               <div><label style={{ display: 'block', marginBottom: 8 }}>{t('services.user')}</label><Input defaultValue="admin" /></div>
               <div><label style={{ display: 'block', marginBottom: 8 }}>{t('services.pass')}</label><Input type="password" /></div>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
               <Badge status="online" text={t('common.connected')} />
               <Button>{t('services.reconnect')}</Button>
            </div>
         </Card>
         <Card title={t('services.publish')}>
            <div style={{ marginBottom: 16 }}>
              <label>{t('services.topic')}</label>
              <Input placeholder="test/topic" />
            </div>
            <div style={{ marginBottom: 16 }}>
              <label>{t('services.payload')}</label>
              <Input placeholder='{"msg": "hello"}' />
            </div>
            <Button>{t('services.publish_btn')}</Button>
         </Card>
       </div>

       <Card title={t('services.live_msgs')}>
          <table className="table">
            <thead><tr><th>{t('services.time')}</th><th>{t('services.dir')}</th><th>{t('services.topic')}</th><th>{t('services.payload')}</th><th>{t('services.qos')}</th></tr></thead>
            <tbody>
              {mockMqttMessages.map(m => (
                <tr key={m.id}>
                  <td style={{ color: '#666', fontSize: 13 }}>{m.timestamp}</td>
                  <td><Badge status={m.direction === 'in' ? 'success' : 'warn'} text={m.direction.toUpperCase()} /></td>
                  <td>{m.topic}</td>
                  <td style={{ fontFamily: 'monospace' }}>{m.payload}</td>
                  <td>{m.qos}</td>
                </tr>
              ))}
            </tbody>
          </table>
       </Card>
    </div>
  );
};
