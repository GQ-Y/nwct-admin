import React, { useEffect, useMemo, useState } from 'react';
import { Card, Button, Input, Badge, Select, SuffixInput } from '../components/UI';
import { Pause, Play, Save, Activity, Users, ArrowUp, ArrowDown, RefreshCw, Plus, Edit, Trash2, X } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api, sanitizeErrorMessage } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';
import { Toast } from '../components/Toast';
import { useIsMobile } from '../lib/useIsMobile';

export const FRPPage: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [status, setStatus] = useState<any>(null);
  const [tunnels, setTunnels] = useState<any[]>([]);
  const [server, setServer] = useState('');
  const [token, setToken] = useState('');
  const [frpMode, setFrpMode] = useState<string>("");
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [domainSuffix, setDomainSuffix] = useState('frpc.zyckj.club');
  const [httpEnabled, setHttpEnabled] = useState(false);
  const [httpsEnabled, setHttpsEnabled] = useState(false);
  const [toastOpen, setToastOpen] = useState(false);
  const [toastType, setToastType] = useState<"success" | "error" | "info">("info");
  const [toastMsg, setToastMsg] = useState("");

  const [cloud, setCloud] = useState<any>(null);
  
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
    return (s?.display_server || s?.server || server || '-') as any;
  }, [rt.frpStatus, status, server]);

  useEffect(() => {
    refresh();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    let stopped = false;
    const tick = async () => {
      try {
        const s = await api.cloudStatus();
        if (!stopped) setCloud(s);
      } catch (e: any) {
        if (!stopped) setCloud({ ok: false, error: sanitizeErrorMessage(e?.message || String(e)) });
      }
    };
    tick();
    const t = window.setInterval(tick, 5000);
    return () => {
      stopped = true;
      window.clearInterval(t);
    };
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
      setFrpMode(String(cfg?.frp_server?.mode || ""));
      const ds = (cfg?.frp_server?.domain_suffix || '').trim();
      if (ds) setDomainSuffix(ds.replace(/^\./, ''));
      setHttpEnabled(!!cfg?.frp_server?.http_enabled);
      setHttpsEnabled(!!cfg?.frp_server?.https_enabled);

      const s = await api.frpStatus();
      setStatus(s);
      if (!server) setServer(s?.server || '');
      const tt = await api.frpTunnels();
      setTunnels(tt?.tunnels || []);
    } catch {
      // ignore
    }
  };

  const refreshWithBridgeSync = async () => {
    setLoading(true);
    try {
      // 仅在“已连接且非手动模式”时才从桥梁同步节点能力配置
      if (connected && frpMode && frpMode !== "manual") {
        await api.frpSync();
      }
      await refresh();
      setToastType("success");
      setToastMsg("已刷新。");
      setToastOpen(true);
    } catch (e: any) {
      setToastType("error");
      setToastMsg(e?.message || String(e));
      setToastOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const tunnelTypeOptions = useMemo(() => {
    const base = [
      { label: 'TCP', value: 'tcp' },
      { label: 'UDP', value: 'udp' },
      { label: 'STCP', value: 'stcp' },
    ];
    // 手动模式：默认全协议开放，不限制 HTTP/HTTPS
    if (frpMode === 'manual') {
      base.splice(2, 0, { label: 'HTTP', value: 'http' });
      base.splice(3, 0, { label: 'HTTPS', value: 'https' });
    } else {
      // builtin/public 模式：根据配置决定是否显示 HTTP/HTTPS
      const hasDomain = !!domainSuffix.trim();
      if (httpEnabled && hasDomain) base.splice(2, 0, { label: 'HTTP', value: 'http' });
      if (httpsEnabled && hasDomain) base.splice(3, 0, { label: 'HTTPS', value: 'https' });
    }
    return base;
  }, [frpMode, httpEnabled, httpsEnabled, domainSuffix]);

  const useBuiltin = async () => {
    setLoading(true);
    try {
      await api.frpUseBuiltin();
      await api.frpConnect({});
      await refresh();
      setToastType("success");
      setToastMsg("已使用官方内置并完成连接。");
      setToastOpen(true);
    } catch (e: any) {
      setToastType("error");
      setToastMsg(e?.message || String(e));
      setToastOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const saveAndConnectManual = async () => {
    setLoading(true);
    try {
      await api.frpConfigSave({
        server: server.trim(),
        token: token.trim(),
        domain_suffix: domainSuffix.trim(),
        // 手动模式：不传递 http_enabled/https_enabled，让后端默认设置为 true（全协议开放）
      });
      // 保存即切换到手动配置，并立即使用该配置连接
      await api.frpConnect({});
      await refresh();
      setToastType("success");
      setToastMsg("已保存并连接。");
      setToastOpen(true);
    } catch (e: any) {
      setToastType("error");
      setToastMsg(e?.message || String(e));
      setToastOpen(true);
    } finally {
      setLoading(false);
    }
  };

  const onToggle = async () => {
    setLoading(true);
    try {
      if (connected) {
        await api.frpDisconnect();
      } else {
        // 连接：使用后端当前选择的模式（builtin/manual/public）的已保存配置
        await api.frpConnect({});
      }
      const s = await api.frpStatus();
      setStatus(s);
      const tt = await api.frpTunnels();
      setTunnels(tt?.tunnels || []);
      setToastType("success");
      setToastMsg(connected ? "已断开。" : "已连接。");
      setToastOpen(true);
    } catch (e: any) {
      setToastType("error");
      setToastMsg(e?.message || String(e));
      setToastOpen(true);
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
    let domainInput = rawDomain;
    // 手动模式：直接使用完整域名，不做前缀提取
    if (frpMode !== 'manual') {
      // builtin/public 模式：尝试提取前缀
      const ds = domainSuffix.replace(/^\./, '');
      if (rawDomain && ds && rawDomain.toLowerCase().endsWith(`.${ds.toLowerCase()}`)) {
        domainInput = rawDomain.slice(0, rawDomain.length - (ds.length + 1)); // 去掉 ".suffix"
      }
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

    // 处理 HTTP/HTTPS 域名
    let domain: string | undefined = undefined;
    if (tunnelForm.type === 'http' || tunnelForm.type === 'https') {
      const v = tunnelForm.domain.trim();
      if (v) {
        // 手动模式：直接使用用户填写的完整域名，不做拼接
        if (frpMode === 'manual') {
          domain = v;
        } else {
          // builtin/public 模式：若包含点号，视为完整域名；否则拼接默认后缀
          if (v.includes('.')) {
            domain = v;
          } else {
            const ds = domainSuffix.replace(/^\./, '');
            domain = ds ? `${v}.${ds}` : v;
          }
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
      <Toast open={toastOpen} type={toastType} message={toastMsg} onClose={() => setToastOpen(false)} />
      <style>{`
        @keyframes totoroPing { 0%{ transform: scale(0.85); opacity: 0.75; } 70%{ transform: scale(1.8); opacity: 0; } 100%{ transform: scale(1.8); opacity: 0; } }
      `}</style>
      <div className="grid-2" style={{ marginBottom: 24 }}>
        <Card title="隧道穿透">
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
              <Input value={server} onChange={(e) => setServer((e.target as any).value)} placeholder="ip" />
              <Input value={token} onChange={(e) => setToken((e.target as any).value)} type="password" placeholder="token" />
            </div>
            <div style={{ marginTop: 12, display: "flex", gap: 8, flexWrap: "wrap" }}>
              <Button variant="ghost" onClick={refreshWithBridgeSync} disabled={loading}>{t('frp.refresh')}</Button>
              <Button variant="outline" onClick={useBuiltin} disabled={loading}>
                {t('frp.use_builtin')}
              </Button>
              <Button variant="primary" onClick={saveAndConnectManual} disabled={loading || !server.trim()}>
                <Save size={16} /> {t('frp.save_config')}
              </Button>
            </div>
            {status?.last_error ? (
              <div style={{ marginTop: 12, color: "#b91c1c", fontSize: 12 }}>
                {sanitizeErrorMessage(String(status.last_error || ""))}
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

             <Card title="Totoro 云服务状态">
               <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: 12 }}>
                 <div style={{ display: "flex", alignItems: "center", gap: 10 }}>
                   <div style={{ position: "relative", width: 14, height: 14 }}>
                     <div
                       style={{
                         position: "absolute",
                         inset: 0,
                         borderRadius: 999,
                         background: cloud?.ok ? "#41BA41" : "#EF4444",
                         boxShadow: cloud?.ok ? "0 0 0 4px rgba(65,186,65,0.14)" : "0 0 0 4px rgba(239,68,68,0.14)",
                       }}
                     />
                     <div
                       style={{
                         position: "absolute",
                         inset: 0,
                         borderRadius: 999,
                         background: cloud?.ok ? "rgba(65,186,65,0.40)" : "rgba(239,68,68,0.40)",
                         animation: "totoroPing 1.6s ease-out infinite",
                       }}
                     />
                   </div>
                   <div>
                     <div style={{ fontWeight: 700, fontSize: 14 }}>
                       {cloud?.ok ? "在线" : "离线"}
                     </div>
                     <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>
                       {cloud?.ok ? "正常" : "离线"}
                     </div>
                   </div>
                 </div>
                 <div style={{ display: "flex", gap: 8, flexWrap: "wrap", justifyContent: "flex-end" }}>
                   <div
                     style={{
                       padding: "6px 10px",
                       borderRadius: 999,
                       background: "rgba(0,0,0,0.04)",
                       color: "var(--text-secondary)",
                       fontSize: 12,
                       lineHeight: 1.2,
                       whiteSpace: "nowrap",
                     }}
                   >
                     设备号 <span style={{ color: "var(--text-primary)", fontWeight: 800, marginLeft: 6 }}>{cloud?.device_id || "-"}</span>
                   </div>
                   <div
                     style={{
                       padding: "6px 10px",
                       borderRadius: 999,
                       background: "rgba(0,0,0,0.04)",
                       color: "var(--text-secondary)",
                       fontSize: 12,
                       lineHeight: 1.2,
                       whiteSpace: "nowrap",
                     }}
                   >
                     固件 <span style={{ color: "var(--text-primary)", fontWeight: 800, marginLeft: 6 }}>{cloud?.firmware_version || "-"}</span>
                   </div>
                   {cloud?.ok && typeof cloud?.official_nodes === "number" ? (
                     <div
                       style={{
                         padding: "6px 10px",
                         borderRadius: 999,
                         background: "rgba(10,89,247,0.10)",
                         color: "var(--primary)",
                         fontSize: 12,
                         lineHeight: 1.2,
                         whiteSpace: "nowrap",
                         fontWeight: 800,
                       }}
                     >
                       {cloud.official_nodes}个云节点
                     </div>
                   ) : null}
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
        {isMobile ? (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
            {tunnels.map((tunnel: any, idx: number) => {
              const name = tunnel?.name || '-';
              const type = String(tunnel?.type || 'tcp').toUpperCase();
              const local = tunnel?.local_ip && tunnel?.local_port ? `${tunnel.local_ip}:${tunnel.local_port}` : '-';
              const remote = tunnel?.remote_port != null && tunnel.remote_port > 0 ? String(tunnel.remote_port) : '自动分配';
              const domain = (tunnel?.type === 'http' || tunnel?.type === 'https') && tunnel?.domain ? String(tunnel.domain) : '-';
              const created = tunnel?.created_at || '-';
              const domainHref = (tunnel?.type === 'http' || tunnel?.type === 'https') && tunnel?.domain ? `${tunnel.type === 'https' ? 'https' : 'http'}://${tunnel.domain}` : '';

              return (
                <div
                  key={tunnel?.name || idx}
                  style={{
                    border: '1px solid rgba(0,0,0,0.06)',
                    borderRadius: 18,
                    padding: 14,
                    background: 'rgba(255,255,255,0.7)',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: 12 }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontWeight: 800, fontSize: 14, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>{name}</div>
                      <div style={{ marginTop: 6, fontSize: 12, color: 'var(--text-secondary)', display: 'flex', gap: 10, flexWrap: 'wrap' }}>
                        <span>类型：<span style={{ color: 'var(--text-primary)', fontWeight: 700 }}>{type}</span></span>
                        <span>本地：<span style={{ color: 'var(--text-primary)', fontWeight: 700 }}>{local}</span></span>
                        <span>远程：<span style={{ color: 'var(--text-primary)', fontWeight: 700 }}>{remote}</span></span>
                      </div>
                      <div style={{ marginTop: 6, fontSize: 12, color: 'var(--text-secondary)' }}>
                        域名：
                        {domainHref ? (
                          <a href={domainHref} target="_blank" rel="noreferrer" style={{ color: 'var(--primary)', textDecoration: 'none', marginLeft: 6, fontWeight: 800 }}>
                            {domain}
                          </a>
                        ) : (
                          <span style={{ color: 'var(--text-primary)', fontWeight: 700, marginLeft: 6 }}>{domain}</span>
                        )}
                      </div>
                      <div style={{ marginTop: 6, fontSize: 12, color: 'var(--text-secondary)' }}>
                        创建：<span style={{ color: 'var(--text-primary)', fontWeight: 700, marginLeft: 6 }}>{created}</span>
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: 8, flex: '0 0 auto' }}>
                      <Button
                        variant="ghost"
                        onClick={() => openEditModal(tunnel)}
                        disabled={loading}
                        style={{ padding: '6px 10px', height: 'auto' }}
                        title="编辑"
                      >
                        <Edit size={14} />
                      </Button>
                      <Button
                        variant="ghost"
                        onClick={() => handleDeleteTunnel(tunnel)}
                        disabled={loading}
                        style={{ padding: '6px 10px', height: 'auto', color: '#ff4d4f' }}
                        title="删除"
                      >
                        <Trash2 size={14} />
                      </Button>
                    </div>
                  </div>
                </div>
              );
            })}
            {tunnels.length === 0 ? (
              <div style={{ color: '#888', padding: 12, textAlign: 'center' }}>
                {connected ? '暂无隧道，点击“创建隧道”按钮添加' : '未连接'}
              </div>
            ) : null}
          </div>
        ) : (
          <div className="table-wrap">
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
          </div>
        )}
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
                  options={tunnelTypeOptions as any}
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
                    {frpMode === 'manual' ? '自定义域名' : '域名前缀'}
                  </label>
                  {frpMode === 'manual' ? (
                    <Input
                      value={tunnelForm.domain}
                      onChange={(e) => setTunnelForm({ ...tunnelForm, domain: (e.target as any).value })}
                      placeholder="例如：example.com 或 subdomain.example.com"
                    />
                  ) : (
                    <SuffixInput
                      value={tunnelForm.domain}
                      onChange={(v) => setTunnelForm({ ...tunnelForm, domain: v })}
                      suffixText={
                        domainSuffix.trim()
                          ? `.${domainSuffix.replace(/^\./, '')}`
                          : "（未配置根域名）"
                      }
                      placeholder="请输入域名前缀"
                    />
                  )}
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
