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
            {/* 按需：进程 PID/日志路径属于调试信息，这里不在 UI 中展示 */}
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
                      style={{ color: 'var(--primary)', textDecoration: 'none' }}
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
// MQTT 功能已移除
