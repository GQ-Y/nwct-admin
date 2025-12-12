import React, { useEffect, useMemo, useState } from 'react';
import { Card, Button, Input, Badge } from '../components/UI';
import { Pause, Play, Save, Activity, Users, ArrowUp, ArrowDown, RefreshCw } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';

export const NPSPage: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<any>(null);
  const [tunnels, setTunnels] = useState<any[]>([]);
  const [server, setServer] = useState('');
  const [vkey, setVKey] = useState('');
  const [clientId, setClientId] = useState('');

  const connected = useMemo(() => {
    const s = rt.npsStatus ?? status;
    return !!s?.connected;
  }, [rt.npsStatus, status]);

  const serverLabel = useMemo(() => {
    const s = rt.npsStatus ?? status;
    return s?.server || server || '-';
  }, [rt.npsStatus, status, server]);

  useEffect(() => {
    api.npsStatus()
      .then((s) => {
        setStatus(s);
        if (!server) setServer(s?.server || '');
        if (!clientId) setClientId(s?.client_id || '');
      })
      .catch(() => {});
    api.npsTunnels()
      .then((d) => setTunnels(d?.tunnels || []))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (rt.npsStatus) setStatus(rt.npsStatus);
  }, [rt.npsStatus]);

  const onToggle = async () => {
    setLoading(true);
    try {
      if (connected) {
        await api.npsDisconnect();
      } else {
        await api.npsConnect({ server: server.trim(), vkey: vkey.trim(), client_id: clientId.trim() });
      }
      const s = await api.npsStatus();
      setStatus(s);
      const tt = await api.npsTunnels();
      setTunnels(tt?.tunnels || []);
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
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
                 background: connected ? '#52c41a' : '#ff4d4f', 
                 marginTop: 6,
                 boxShadow: connected ? '0 0 10px rgba(82, 196, 26, 0.4)' : '0 0 10px rgba(255, 77, 79, 0.4)'
               }} />
               <div>
                 <div style={{ fontSize: 18, fontWeight: 600 }}>{connected ? t('common.connected') : t('common.disconnected')}</div>
                 <div style={{ color: '#666' }}>{serverLabel}</div>
               </div>
             </div>
             <Button 
                variant={connected ? "outline" : "primary"} 
                onClick={onToggle}
                disabled={loading}
                style={connected ? { color: '#ff4d4f', borderColor: '#ff4d4f' } : {}}
             >
                {connected ? <Pause size={16} /> : <Play size={16} />} 
                {connected ? t('common.disconnect') : t('common.connect')}
             </Button>
          </div>
          <div style={{ background: '#f5f5f5', padding: 20, borderRadius: 12 }}>
            <div style={{ fontSize: 13, color: '#666', marginBottom: 12, fontWeight: 500 }}>{t('services.config')}</div>
            <div className="grid-2">
              <Input value={server} onChange={(e) => setServer((e.target as any).value)} placeholder="server:8024" />
              <Input value={vkey} onChange={(e) => setVKey((e.target as any).value)} type="password" placeholder="vkey" />
            </div>
            <div style={{ marginTop: 12 }}>
              <div style={{ fontSize: 13, color: '#666', marginBottom: 8 }}>Client ID</div>
              <Input value={clientId} onChange={(e) => setClientId((e.target as any).value)} placeholder="client_id" />
            </div>
            <Button style={{ marginTop: 16 }} variant="primary" onClick={onToggle} disabled={loading}>
              <Save size={16} /> {connected ? t('common.disconnect') : t('services.save_config')}
            </Button>
          </div>
        </Card>
        
        <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
             <Card title={t('services.tunnel_stats')}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <div style={{ fontSize: 36, fontWeight: 700, color: 'var(--primary)' }}>{tunnels.length}</div>
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
            {tunnels.map((tunnel: any, idx: number) => (
              <tr key={tunnel.id || idx}>
                <td>{tunnel.id || '-'}</td>
                <td>{String(tunnel.type || '').toUpperCase()}</td>
                <td>{tunnel.local_port != null ? `:${tunnel.local_port}` : '-'}</td>
                <td>{tunnel.remote_port != null ? tunnel.remote_port : '-'}</td>
                <td><Badge status={tunnel.status || 'offline'} text={t(`common.${tunnel.status || 'offline'}`)} /></td>
              </tr>
            ))}
            {tunnels.length === 0 ? (
              <tr>
                <td colSpan={5} style={{ color: '#888', padding: 12 }}>
                  {connected ? '暂无隧道（当前实现中 npc 隧道列表可能为空）' : '未连接'}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>
    </div>
  );
};

export const MQTTPage: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<any>(null);
  const [logs, setLogs] = useState<any[]>([]);

  const [host, setHost] = useState("");
  const [port, setPort] = useState(1883);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [clientId, setClientId] = useState("");

  const [pubTopic, setPubTopic] = useState("");
  const [pubPayload, setPubPayload] = useState("");

  const connected = !!status?.connected;

  useEffect(() => {
    api.mqttStatus()
      .then((s) => {
        setStatus(s);
        // 形如 "host:port"
        const sp = String(s?.server || "");
        if (!host && sp.includes(":")) {
          const [h, p] = sp.split(":");
          if (h) setHost(h);
          const n = Number(p);
          if (!Number.isNaN(n) && n > 0) setPort(n);
        }
        if (!clientId) setClientId(s?.client_id || "");
      })
      .catch(() => {});
    api.mqttLogs({ page: 1, page_size: 50 })
      .then((d) => setLogs(d?.logs || []))
      .catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!rt.mqttLogNew) return;
    setLogs((prev) => {
      const next = [rt.mqttLogNew, ...prev];
      return next.slice(0, 200);
    });
  }, [rt.mqttLogNew]);

  const onToggle = async () => {
    setLoading(true);
    try {
      if (connected) {
        await api.mqttDisconnect();
      } else {
        await api.mqttConnect({
          server: host.trim(),
          port: Number(port) || 1883,
          username: username.trim() || undefined,
          password: password || undefined,
          client_id: clientId.trim(),
          tls: false,
        });
      }
      const s = await api.mqttStatus();
      setStatus(s);
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  const onPublish = async () => {
    setLoading(true);
    try {
      await api.mqttPublish({ topic: pubTopic.trim(), payload: pubPayload });
      const d = await api.mqttLogs({ page: 1, page_size: 50 });
      setLogs(d?.logs || []);
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
       <div className="grid-2" style={{ marginBottom: 24 }}>
         <Card title={t('services.broker_conn')}>
            <div style={{ marginBottom: 16 }}>
              <label style={{ display: 'block', marginBottom: 8 }}>{t('services.host')}</label>
              <Input value={host} onChange={(e) => setHost((e.target as any).value)} placeholder="mqtt.example.com" />
            </div>
            <div className="grid-2" style={{ marginBottom: 16 }}>
               <div>
                 <label style={{ display: 'block', marginBottom: 8 }}>{t('services.user')}</label>
                 <Input value={username} onChange={(e) => setUsername((e.target as any).value)} />
               </div>
               <div>
                 <label style={{ display: 'block', marginBottom: 8 }}>{t('services.pass')}</label>
                 <Input value={password} onChange={(e) => setPassword((e.target as any).value)} type="password" />
               </div>
            </div>
            <div className="grid-2" style={{ marginBottom: 16 }}>
              <div>
                <label style={{ display: 'block', marginBottom: 8 }}>Client ID</label>
                <Input value={clientId} onChange={(e) => setClientId((e.target as any).value)} placeholder="device_001" />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 8 }}>Port</label>
                <Input value={String(port)} onChange={(e) => setPort(Number((e.target as any).value) || 1883)} />
              </div>
            </div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
               <Badge 
                  status={connected ? 'online' : 'offline'} 
                  text={connected ? t('common.connected') : t('common.disconnected')} 
               />
               <Button onClick={onToggle} disabled={loading}>
                  {loading ? (
                    <>
                      <RefreshCw size={16} className="animate-spin" /> {t('common.loading')}
                    </>
                  ) : connected ? (
                    t('common.disconnect')
                  ) : (
                    t('common.connect')
                  )}
               </Button>
            </div>
         </Card>
         <Card title={t('services.publish')}>
            <div style={{ marginBottom: 16 }}>
              <label>{t('services.topic')}</label>
              <Input value={pubTopic} onChange={(e) => setPubTopic((e.target as any).value)} placeholder="test/topic" />
            </div>
            <div style={{ marginBottom: 16 }}>
              <label>{t('services.payload')}</label>
              <Input value={pubPayload} onChange={(e) => setPubPayload((e.target as any).value)} placeholder='{"msg":"hello"} 或纯文本' />
            </div>
            <Button onClick={onPublish} disabled={loading || !pubTopic.trim()}>
              {t('services.publish_btn')}
            </Button>
         </Card>
       </div>

       <Card title={t('services.live_msgs')}>
          <table className="table">
            <thead><tr><th>{t('services.time')}</th><th>{t('services.dir')}</th><th>{t('services.topic')}</th><th>{t('services.payload')}</th><th>{t('services.qos')}</th></tr></thead>
            <tbody>
              {logs.map((m: any, idx: number) => (
                <tr key={`${m.timestamp || idx}-${idx}`}>
                  <td style={{ color: '#666', fontSize: 13 }}>{m.timestamp || '-'}</td>
                  <td><Badge status={m.direction === 'subscribe' ? 'success' : 'warn'} text={String(m.direction || '').toUpperCase()} /></td>
                  <td>{m.topic}</td>
                  <td style={{ fontFamily: 'monospace' }}>{m.payload}</td>
                  <td>{m.qos}</td>
                </tr>
              ))}
              {logs.length === 0 ? (
                <tr>
                  <td colSpan={5} style={{ color: '#888', padding: 12 }}>暂无日志</td>
                </tr>
              ) : null}
            </tbody>
          </table>
       </Card>
    </div>
  );
};
