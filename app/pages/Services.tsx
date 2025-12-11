import React, { useState } from 'react';
import { Card, Button, Input, Badge } from '../components/UI';
import { mockMqttMessages, mockTunnels } from '../data';
import { Pause, Play, Save, Activity, Users, ArrowUp, ArrowDown, RefreshCw } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const NPSPage: React.FC = () => {
  const { t } = useLanguage();
  const [isConnected, setIsConnected] = useState(true);

  const toggleConnection = () => {
    setIsConnected(!isConnected);
  };

  return (
    <div>
      <div className="grid-2" style={{ marginBottom: 24 }}>
        <Card title={t('services.nps_conn')}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 }}>
             <div style={{ display: 'flex', gap: 16 }}>
               <div style={{ 
                 width: 12, 
                 height: 12, 
                 borderRadius: '50%', 
                 background: isConnected ? '#52c41a' : '#ff4d4f', 
                 marginTop: 6,
                 boxShadow: isConnected ? '0 0 10px rgba(82, 196, 26, 0.4)' : '0 0 10px rgba(255, 77, 79, 0.4)'
               }} />
               <div>
                 <div style={{ fontSize: 18, fontWeight: 600 }}>{isConnected ? t('common.connected') : t('common.disconnected')}</div>
                 <div style={{ color: '#666' }}>server.nps.host:8024</div>
               </div>
             </div>
             <Button 
                variant={isConnected ? "outline" : "primary"} 
                onClick={toggleConnection}
                style={isConnected ? { color: '#ff4d4f', borderColor: '#ff4d4f' } : {}}
             >
                {isConnected ? <Pause size={16} /> : <Play size={16} />} 
                {isConnected ? t('common.disconnect') : t('common.connect')}
             </Button>
          </div>
          <div style={{ background: '#f5f5f5', padding: 20, borderRadius: 12 }}>
            <div style={{ fontSize: 13, color: '#666', marginBottom: 12, fontWeight: 500 }}>{t('services.config')}</div>
            <div className="grid-2">
              <Input defaultValue="server.nps.host:8024" />
              <Input defaultValue="my-vkey-token" type="password" />
            </div>
            <Button style={{ marginTop: 16 }} variant="primary"><Save size={16} /> {t('services.save_config')}</Button>
          </div>
        </Card>
        
        <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
             <Card title={t('services.tunnel_stats')}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <div style={{ fontSize: 36, fontWeight: 700, color: 'var(--primary)' }}>3</div>
                        <div style={{ color: 'var(--text-secondary)' }}>{t('services.active_tunnels')}</div>
                    </div>
                    <div style={{ width: 48, height: 48, borderRadius: '50%', background: 'rgba(10, 89, 247, 0.1)', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <Activity size={24} color="var(--primary)" />
                    </div>
                </div>
             </Card>

             <div className="grid-2" style={{ gap: 24 }}>
                 <Card style={{ marginBottom: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                            <div style={{ fontSize: 24, fontWeight: 700, color: '#52c41a' }}>12</div>
                            <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{t('services.clients_online') || 'Clients Online'}</div>
                        </div>
                         <Users size={20} color="#52c41a" />
                    </div>
                 </Card>
                 <Card style={{ marginBottom: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                            <div style={{ fontSize: 24, fontWeight: 700, color: '#faad14' }}>2.4G</div>
                            <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{t('services.total_traffic') || 'Total Traffic'}</div>
                        </div>
                        <div style={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                            <ArrowUp size={12} color="#faad14" />
                            <ArrowDown size={12} color="#faad14" />
                        </div>
                    </div>
                 </Card>
             </div>
        </div>
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
  const [connectionState, setConnectionState] = useState<'connected' | 'disconnected' | 'connecting'>('connected');

  const handleReconnect = () => {
    if (connectionState === 'connected') {
        setConnectionState('disconnected');
    } else {
        setConnectionState('connecting');
        setTimeout(() => {
            setConnectionState('connected');
        }, 2000);
    }
  };

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
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
               <Badge 
                  status={connectionState === 'connected' ? 'online' : connectionState === 'connecting' ? 'warn' : 'offline'} 
                  text={connectionState === 'connected' ? t('common.connected') : connectionState === 'connecting' ? t('common.loading') : t('common.disconnected')} 
               />
               <Button onClick={handleReconnect} disabled={connectionState === 'connecting'}>
                  {connectionState === 'connecting' ? (
                      <>
                        <RefreshCw size={16} className="animate-spin" /> {t('common.loading')}
                      </>
                  ) : connectionState === 'connected' ? (
                      t('services.reconnect')
                  ) : (
                      t('common.connect')
                  )}
               </Button>
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
