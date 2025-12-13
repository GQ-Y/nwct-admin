import React, { useEffect, useMemo, useState } from 'react';
import { Card, Button, Input, Badge, Select } from '../components/UI';
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
  const [npcPath, setNpcPath] = useState('');

  const connected = useMemo(() => {
    const s = rt.npsStatus ?? status;
    return !!s?.connected;
  }, [rt.npsStatus, status]);

  const serverLabel = useMemo(() => {
    const s = rt.npsStatus ?? status;
    return s?.server || server || '-';
  }, [rt.npsStatus, status, server]);

  const clientsOnline = useMemo(() => {
    const s = rt.npsStatus ?? status;
    const n = Number(s?.clients_online);
    return Number.isFinite(n) ? n : null;
  }, [rt.npsStatus, status]);

  const totalTrafficHuman = useMemo(() => {
    const s = rt.npsStatus ?? status;
    const human = String(s?.total_traffic_human || '').trim();
    if (human) return human;
    const bytes = Number(s?.total_traffic_bytes);
    if (!Number.isFinite(bytes) || bytes <= 0) return '';
    // 简单格式化（IEC）
    const units = ['B', 'K', 'M', 'G', 'T', 'P'];
    let v = bytes;
    let i = 0;
    while (v >= 1024 && i < units.length - 1) {
      v = v / 1024;
      i++;
    }
    return `${v.toFixed(1)}${units[i]}`;
  }, [rt.npsStatus, status]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (rt.npsStatus) setStatus(rt.npsStatus);
  }, [rt.npsStatus]);

  const refresh = async () => {
    try {
      const s = await api.npsStatus();
      setStatus(s);
      if (!server) setServer(s?.server || '');
      if (!clientId) setClientId(s?.client_id || '');
      if (!npcPath) setNpcPath(s?.npc_path || '');
      const tt = await api.npsTunnels();
      setTunnels(tt?.tunnels || []);
    } catch {
      // ignore
    }
  };

  const onToggle = async () => {
    setLoading(true);
    try {
      if (connected) {
        await api.npsDisconnect();
      } else {
        const req = {
          server: server.trim() || undefined,
          vkey: vkey.trim() || undefined,
          client_id: clientId.trim() || undefined,
          npc_path: npcPath.trim() || undefined,
        };
        await api.npsConnect(req);
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

  const installNpc = async () => {
    setLoading(true);
    try {
      const res = await api.npsNpcInstall({});
      setNpcPath(res?.path || '');
      const s = await api.npsStatus();
      setStatus(s);
      alert(`npc 已安装: ${res?.path || ''}`);
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
            <div style={{ marginTop: 12 }}>
              <div style={{ fontSize: 13, color: '#666', marginBottom: 8 }}>npc 路径</div>
              <Input value={npcPath} onChange={(e) => setNpcPath((e.target as any).value)} placeholder="npc 可执行文件路径" />
              <div style={{ marginTop: 8, display: "flex", gap: 8 }}>
                <Button variant="outline" onClick={installNpc} disabled={loading}>
                  一键安装 npc
                </Button>
                <Button variant="ghost" onClick={refresh} disabled={loading}>
                  刷新
                </Button>
              </div>
            </div>
            <Button style={{ marginTop: 16 }} variant="primary" onClick={onToggle} disabled={loading}>
              <Save size={16} /> {connected ? t('common.disconnect') : t('services.save_config')}
            </Button>
            {status?.last_error ? (
              <div style={{ marginTop: 12, color: "#b91c1c", fontSize: 12 }}>
                last_error: {status.last_error}
              </div>
            ) : null}
            {(status?.pid || status?.log_path) ? (
              <div style={{ marginTop: 10, color: "#666", fontSize: 12 }}>
                {status?.pid ? <div>pid: {status.pid}</div> : null}
                {status?.log_path ? <div>npc.log: {status.log_path}</div> : null}
              </div>
            ) : null}
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
                            <div style={{ fontSize: 24, fontWeight: 700, color: '#52c41a' }}>
                              {connected ? (clientsOnline ?? '-') : '-'}
                            </div>
                            <div style={{ color: 'var(--text-secondary)', fontSize: 13 }}>{t('services.clients_online') || 'Clients Online'}</div>
                        </div>
                         <Users size={20} color="#52c41a" />
                    </div>
                 </Card>
                 <Card style={{ marginBottom: 0 }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                            <div style={{ fontSize: 24, fontWeight: 700, color: '#faad14' }}>
                              {connected ? (totalTrafficHuman || '-') : '-'}
                            </div>
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
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(50);
  const [total, setTotal] = useState(0);
  const [hasNew, setHasNew] = useState(false);
  const [lastError, setLastError] = useState<string>("");

  const [host, setHost] = useState("");
  const [port, setPort] = useState(1883);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [clientId, setClientId] = useState("");

  const [pubTopic, setPubTopic] = useState("");
  const [pubPayload, setPubPayload] = useState("");

  const connected = !!status?.connected;

  const refreshLogs = async (p: number = page, ps: number = pageSize) => {
    const d = await api.mqttLogs({ page: p, page_size: ps });
    const list = Array.isArray(d?.logs) ? d.logs : [];
    // 兜底：即使后端返回条数不受 page_size 影响，也保证前端按 pageSize 显示
    setLogs(list.slice(0, ps));
    setTotal(Number(d?.total || 0));
    setPage(Number(d?.page || p) || p);
    setPageSize(Number(d?.page_size || ps) || ps);
    setHasNew(false);
  };

  useEffect(() => {
    api.mqttStatus()
      .then((s) => {
        setStatus(s);
        if (!username && s?.username) setUsername(String(s.username || ""));
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
    refreshLogs(1, pageSize).catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!rt.mqttLogNew) return;
    // 只有第一页才实时 prepend（分页下避免破坏用户翻页阅读）
    if (page === 1) {
      setLogs((prev) => {
        const next = [rt.mqttLogNew, ...prev];
        return next.slice(0, pageSize);
      });
      setTotal((x) => x + 1);
    } else {
      setHasNew(true);
    }
  }, [rt.mqttLogNew]);

  // pageSize 变更时立即截断当前列表，避免“切换条数但列表不变”的感知
  useEffect(() => {
    if (page !== 1) return;
    setLogs((prev) => prev.slice(0, pageSize));
  }, [page, pageSize]);

  const onToggle = async () => {
    setLoading(true);
    setLastError("");
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
      // 连接成功后刷新第一页日志
      if (!connected) {
        await refreshLogs(1, pageSize);
      }
    } catch (e: any) {
      setLastError(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  const onPublish = async () => {
    setLoading(true);
    setLastError("");
    try {
      await api.mqttPublish({ topic: pubTopic.trim(), payload: pubPayload });
      // 发布后刷新当前页（一般在第一页看到最新）
      await refreshLogs(page, pageSize);
    } catch (e: any) {
      setLastError(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  const totalPages = Math.max(1, Math.ceil((total || 0) / (pageSize || 50)));
  const canPrev = page > 1;
  const canNext = page < totalPages;

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
            {lastError ? (
              <div style={{ marginTop: 10, color: "#b91c1c", fontSize: 12 }}>
                {lastError}
              </div>
            ) : null}
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
            <Button onClick={onPublish} disabled={loading || !connected || !pubTopic.trim()}>
              {t('services.publish_btn')}
            </Button>
            {!connected ? (
              <div style={{ marginTop: 10, color: "#666", fontSize: 12 }}>
                请先连接 MQTT 后再发布
              </div>
            ) : null}
         </Card>
       </div>

       <Card title={t('services.live_msgs')}>
          {hasNew ? (
            <div style={{ marginBottom: 10, color: "#0A59F7", fontSize: 12 }}>
              有新消息到达（当前在第 {page} 页）。点击“刷新”查看。
            </div>
          ) : null}
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

          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginTop: 12, gap: 12, flexWrap: "wrap" }}>
            <div style={{ display: "flex", alignItems: "center", gap: 10, color: "#666", fontSize: 12 }}>
              <span>共 {total} 条</span>
              <span style={{ fontVariantNumeric: "tabular-nums" }}>页码 {page}/{totalPages}</span>
              <span>每页</span>
              <Select
                width={120}
                value={String(pageSize)}
                options={[
                  { label: "20 条", value: "20" },
                  { label: "50 条", value: "50" },
                  { label: "100 条", value: "100" },
                ]}
                onChange={(v) => {
                  const ps = Number(v) || 50;
                  setPageSize(ps);
                  setPage(1);
                  // 立刻更新 UI（即使网络慢也能看到条数变化）
                  setLogs((prev) => prev.slice(0, ps));
                  refreshLogs(1, ps).catch(() => {});
                }}
              />
            </div>
            <div style={{ display: "flex", gap: 8 }}>
              <Button variant="ghost" disabled={loading} onClick={() => refreshLogs(page, pageSize).catch(() => {})}>
                刷新
              </Button>
              <Button variant="outline" disabled={loading || !canPrev} onClick={() => refreshLogs(page - 1, pageSize).catch(() => {})}>
                上一页
              </Button>
              <Button variant="outline" disabled={loading || !canNext} onClick={() => refreshLogs(page + 1, pageSize).catch(() => {})}>
                下一页
              </Button>
            </div>
          </div>
       </Card>
    </div>
  );
};
