import React, { useEffect, useMemo, useState } from 'react';
import { Card, Button, Input, Badge, Select } from '../components/UI';
import { Pause, Play, Save, Activity, Users, ArrowUp, ArrowDown, RefreshCw, Plus, Edit, Trash2, X } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';

export const FRPPage: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<any>(null);
  const [tunnels, setTunnels] = useState<any[]>([]);
  const [server, setServer] = useState('');
  const [token, setToken] = useState('');
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [domainSuffix, setDomainSuffix] = useState('frpc.zyckj.club');
  
  // 隧道管理相关状态
  const [showTunnelModal, setShowTunnelModal] = useState(false);
  const [editingTunnel, setEditingTunnel] = useState<any>(null);
  const [tunnelForm, setTunnelForm] = useState({
    name: '',
    type: 'tcp',
    local_ip: '',
    local_port: '',
    remote_port: '',
    domain: '',
  });

  const connected = useMemo(() => {
    const s = rt.frpStatus ?? status;
    return !!s?.connected;
  }, [rt.frpStatus, status]);

  const serverLabel = useMemo(() => {
    const s = rt.frpStatus ?? status;
    return s?.server || server || '-';
  }, [rt.frpStatus, status, server]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (rt.frpStatus) setStatus(rt.frpStatus);
  }, [rt.frpStatus]);

  const copyText = async (key: string, text: string) => {
    try {
      await navigator.clipboard.writeText(text);
      setCopiedKey(key);
      window.setTimeout(() => setCopiedKey((k) => (k === key ? null : k)), 1200);
    } catch {
      // clipboard 不可用时静默失败
    }
  };

  const refresh = async () => {
    try {
      // 读取后端配置：获取默认域名后缀
      const cfg = await api.configGet();
      const ds = (cfg?.frp_server?.domain_suffix || '').trim();
      if (ds) setDomainSuffix(ds.replace(/^\./, ''));

      const s = await api.frpStatus();
      setStatus(s);
      if (!server) setServer(s?.server || '');
      const tt = await api.frpTunnels();
      setTunnels(tt?.tunnels || []);
    } catch {
      // ignore
    }
  };

  const onToggle = async () => {
    setLoading(true);
    try {
      if (connected) {
        await api.frpDisconnect();
      } else {
        const req = {
          server: server.trim() || undefined,
          token: token.trim() || undefined,
        };
        await api.frpConnect(req);
      }
      const s = await api.frpStatus();
      setStatus(s);
      const tt = await api.frpTunnels();
      setTunnels(tt?.tunnels || []);
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  // 打开创建隧道对话框
  const openCreateModal = () => {
    setEditingTunnel(null);
    setTunnelForm({
      name: '',
      type: 'tcp',
      local_ip: '',
      local_port: '',
      remote_port: '',
      domain: '',
    });
    setShowTunnelModal(true);
  };

  // 打开编辑隧道对话框
  const openEditModal = (tunnel: any) => {
    setEditingTunnel(tunnel);
    const rawDomain = String(tunnel.domain || '').trim();
    const ds = domainSuffix.replace(/^\./, '');
    let domainInput = rawDomain;
    if (rawDomain && ds && rawDomain.toLowerCase().endsWith(`.${ds.toLowerCase()}`)) {
      domainInput = rawDomain.slice(0, rawDomain.length - (ds.length + 1)); // 去掉 ".suffix"
    }
    setTunnelForm({
      name: tunnel.name || '',
      type: tunnel.type || 'tcp',
      local_ip: tunnel.local_ip || '',
      local_port: String(tunnel.local_port || ''),
      remote_port: tunnel.remote_port ? String(tunnel.remote_port) : '',
      domain: domainInput,
    });
    setShowTunnelModal(true);
  };

  // 关闭对话框
  const closeModal = () => {
    setShowTunnelModal(false);
    setEditingTunnel(null);
  };

  // 生成隧道名称（如果未提供）
  const generateTunnelName = (localIP: string, localPort: string) => {
    if (localIP && localPort) {
      return localIP.replace(/\./g, '_') + '_' + localPort;
    }
    return '';
  };

  // 创建或更新隧道
  const handleSaveTunnel = async () => {
    // 验证表单
    if (!tunnelForm.local_ip.trim()) {
      alert('请输入本地 IP');
      return;
    }
    if (!tunnelForm.local_port.trim() || Number(tunnelForm.local_port) < 1 || Number(tunnelForm.local_port) > 65535) {
      alert('请输入有效的本地端口（1-65535）');
      return;
    }
    if (tunnelForm.remote_port && (Number(tunnelForm.remote_port) < 0 || Number(tunnelForm.remote_port) > 65535)) {
      alert('远程端口必须在 0-65535 之间（0 表示自动分配）');
      return;
    }

    const name = tunnelForm.name.trim() || generateTunnelName(tunnelForm.local_ip, tunnelForm.local_port);
    if (!name) {
      alert('无法生成隧道名称，请手动输入');
      return;
    }

    // 处理 HTTP/HTTPS 域名：默认只填前缀
    let domain: string | undefined = undefined;
    if (tunnelForm.type === 'http' || tunnelForm.type === 'https') {
      const v = tunnelForm.domain.trim();
      if (v) {
        // 若包含点号，视为完整域名；否则拼接默认后缀
        if (v.includes('.')) {
          domain = v;
        } else {
          const ds = domainSuffix.replace(/^\./, '');
          domain = ds ? `${v}.${ds}` : v;
        }
      }
    }

    setLoading(true);
    try {
      const tunnelData = {
        name,
        type: tunnelForm.type,
        local_ip: tunnelForm.local_ip.trim(),
        local_port: Number(tunnelForm.local_port),
        remote_port: tunnelForm.remote_port ? Number(tunnelForm.remote_port) : 0,
        domain,
      };

      if (editingTunnel) {
        // 更新隧道
        await api.frpUpdateTunnel(editingTunnel.name, tunnelData);
      } else {
        // 创建隧道
        await api.frpAddTunnel(tunnelData);
      }

      // 刷新列表
      await refresh();
      closeModal();
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  // 删除隧道
  const handleDeleteTunnel = async (tunnel: any) => {
    if (!confirm(`确定要删除隧道 "${tunnel.name}" 吗？`)) {
      return;
    }

    setLoading(true);
    try {
      await api.frpRemoveTunnel(tunnel.name);
      // 刷新列表
      await refresh();
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="grid-2" style={{ marginBottom: 24 }}>
        <Card title="FRP 连接">
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
              <Input value={server} onChange={(e) => setServer((e.target as any).value)} placeholder="117.172.29.237:7000" />
              <Input value={token} onChange={(e) => setToken((e.target as any).value)} type="password" placeholder="token" />
            </div>
            <div style={{ marginTop: 12, display: "flex", gap: 8 }}>
              <Button variant="ghost" onClick={refresh} disabled={loading}>
                刷新
              </Button>
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
              <div
                style={{
                  marginTop: 12,
                  background: 'rgba(0,0,0,0.03)',
                  borderRadius: 12,
                  padding: 12,
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 10,
                }}
              >
                {status?.pid ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div style={{ width: 72, fontSize: 12, color: 'var(--text-secondary)', fontWeight: 600 }}>进程</div>
                    <div
                      style={{
                        flex: 1,
                        fontFamily: '"SF Mono", Consolas, Monaco, monospace',
                        fontSize: 12,
                        color: 'var(--text-primary)',
                      }}
                    >
                      pid {status.pid}
                    </div>
                    <button
                      className="btn btn-ghost"
                      style={{ height: 32, padding: '0 14px' }}
                      onClick={() => copyText('frp_pid', String(status.pid))}
                      title="复制 PID"
                    >
                      {copiedKey === 'frp_pid' ? '已复制' : '复制'}
                    </button>
                  </div>
                ) : null}

                {status?.log_path ? (
                  <div style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
                    <div style={{ width: 72, fontSize: 12, color: 'var(--text-secondary)', fontWeight: 600 }}>日志</div>
                    <div
                      title={status.log_path}
                      style={{
                        flex: 1,
                        fontFamily: '"SF Mono", Consolas, Monaco, monospace',
                        fontSize: 12,
                        color: 'var(--text-primary)',
                        overflow: 'hidden',
                        textOverflow: 'ellipsis',
                        whiteSpace: 'nowrap',
                        padding: '6px 10px',
                        borderRadius: 10,
                        background: 'rgba(255,255,255,0.7)',
                        border: '1px solid rgba(255,255,255,0.6)',
                      }}
                    >
                      {status.log_path}
                    </div>
                    <button
                      className="btn btn-ghost"
                      style={{ height: 32, padding: '0 14px' }}
                      onClick={() => copyText('frp_log', status.log_path)}
                      title="复制日志路径"
                    >
                      {copiedKey === 'frp_log' ? '已复制' : '复制'}
                    </button>
                  </div>
                ) : null}
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
             </div>
        </div>
      </div>

      <Card 
        title={t('services.tunnels_list')}
        extra={
          <Button 
            variant="primary" 
            onClick={openCreateModal}
            disabled={!connected || loading}
            style={{ display: 'flex', alignItems: 'center', gap: 8 }}
          >
            <Plus size={16} /> 创建隧道
          </Button>
        }
      >
          <table className="table">
          <thead>
            <tr>
              <th>名称</th>
              <th>类型</th>
              <th>本地地址</th>
              <th>远程端口</th>
              <th>域名</th>
              <th>创建时间</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            {tunnels.map((tunnel: any, idx: number) => (
              <tr key={tunnel.name || idx}>
                <td>{tunnel.name || '-'}</td>
                <td>{String(tunnel.type || 'tcp').toUpperCase()}</td>
                <td>{tunnel.local_ip && tunnel.local_port ? `${tunnel.local_ip}:${tunnel.local_port}` : '-'}</td>
                <td>{tunnel.remote_port != null && tunnel.remote_port > 0 ? tunnel.remote_port : '自动分配'}</td>
                <td>
                  {(tunnel.type === 'http' || tunnel.type === 'https') && tunnel.domain ? (
                    <a
                      href={`${tunnel.type === 'https' ? 'https' : 'http'}://${tunnel.domain}`}
                      target="_blank"
                      rel="noreferrer"
                      style={{ color: 'var(--primary)' }}
                      title="新标签打开"
                    >
                      {tunnel.domain}
                    </a>
                  ) : (
                    '-'
                  )}
                </td>
                <td>{tunnel.created_at || '-'}</td>
                <td>
                  <div style={{ display: 'flex', gap: 8 }}>
                    <Button
                      variant="ghost"
                      onClick={() => openEditModal(tunnel)}
                      disabled={loading}
                      style={{ padding: '4px 8px', height: 'auto' }}
                      title="编辑"
                    >
                      <Edit size={14} />
                    </Button>
                    <Button
                      variant="ghost"
                      onClick={() => handleDeleteTunnel(tunnel)}
                      disabled={loading}
                      style={{ padding: '4px 8px', height: 'auto', color: '#ff4d4f' }}
                      title="删除"
                    >
                      <Trash2 size={14} />
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
            {tunnels.length === 0 ? (
              <tr>
                <td colSpan={7} style={{ color: '#888', padding: 12 }}>
                  {connected ? '暂无隧道，点击"创建隧道"按钮添加' : '未连接'}
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>

      {/* 创建/编辑隧道对话框 */}
      {showTunnelModal && (
        <div
          style={{
            position: 'fixed',
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            background: 'rgba(0, 0, 0, 0.5)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            zIndex: 1000,
          }}
          onClick={(e) => {
            if (e.target === e.currentTarget) closeModal();
          }}
        >
          <Card
            title={editingTunnel ? '编辑隧道' : '创建隧道'}
            extra={
              <button
                onClick={closeModal}
                style={{
                  background: 'none',
                  border: 'none',
                  cursor: 'pointer',
                  padding: 4,
                  display: 'flex',
                  alignItems: 'center',
                }}
              >
                <X size={20} />
              </button>
            }
            style={{
              width: '90%',
              maxWidth: 600,
              maxHeight: '90vh',
              overflow: 'auto',
            }}
          >
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
              <div>
                <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                  隧道名称 <span style={{ color: '#999' }}>(可选，不填则自动生成)</span>
                </label>
                <Input
                  value={tunnelForm.name}
                  onChange={(e) => setTunnelForm({ ...tunnelForm, name: (e.target as any).value })}
                  placeholder="自动生成"
                />
              </div>

              <div>
                <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                  类型 <span style={{ color: '#f5222d' }}>*</span>
                </label>
                <Select
                  value={tunnelForm.type}
                  onChange={(value) => setTunnelForm({ ...tunnelForm, type: value })}
                  options={[
                    { label: 'TCP', value: 'tcp' },
                    { label: 'UDP', value: 'udp' },
                    { label: 'HTTP', value: 'http' },
                    { label: 'HTTPS', value: 'https' },
                    { label: 'STCP', value: 'stcp' },
                  ]}
                />
              </div>

              <div>
                <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                  本地 IP <span style={{ color: '#f5222d' }}>*</span>
                </label>
                <Input
                  value={tunnelForm.local_ip}
                  onChange={(e) => setTunnelForm({ ...tunnelForm, local_ip: (e.target as any).value })}
                  placeholder="192.168.1.100"
                />
              </div>

              <div>
                <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                  本地端口 <span style={{ color: '#f5222d' }}>*</span>
                </label>
                <Input
                  type="number"
                  value={tunnelForm.local_port}
                  onChange={(e) => setTunnelForm({ ...tunnelForm, local_port: (e.target as any).value })}
                  placeholder="80"
                  min="1"
                  max="65535"
                />
              </div>

              <div>
                <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                  远程端口 <span style={{ color: '#999' }}>(可选，0 表示自动分配)</span>
                </label>
                <Input
                  type="number"
                  value={tunnelForm.remote_port}
                  onChange={(e) => setTunnelForm({ ...tunnelForm, remote_port: (e.target as any).value })}
                  placeholder="0 (自动分配)"
                  min="0"
                  max="65535"
                />
              </div>

              {(tunnelForm.type === 'http' || tunnelForm.type === 'https') && (
                <div>
                  <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                    域名前缀 <span style={{ color: '#999' }}>(可选，仅 HTTP/HTTPS 使用)</span>
                  </label>
                  <Input
                    value={tunnelForm.domain}
                    onChange={(e) => setTunnelForm({ ...tunnelForm, domain: (e.target as any).value })}
                    placeholder={`例如：e6666666（默认后缀 .${domainSuffix}）或完整域名`}
                  />
                </div>
              )}

              <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 8 }}>
                <Button variant="outline" onClick={closeModal} disabled={loading}>
                  取消
                </Button>
                <Button variant="primary" onClick={handleSaveTunnel} disabled={loading}>
                  {loading ? '保存中...' : editingTunnel ? '更新' : '创建'}
                </Button>
              </div>
            </div>
          </Card>
        </div>
      )}
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
