import React, { useState, useEffect } from 'react';
import { Card, Button, SearchInput, Select, Badge, Pagination, ProgressBar, Alert, Input } from '../components/UI';
import { Device } from '../types';
import { RefreshCw, Smartphone, Monitor, Server, Camera, Radar } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';

export const Devices: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [devices, setDevices] = useState<Device[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  // 默认只显示在线设备
  const [filterType, setFilterType] = useState('online');
  const [view, setView] = useState<'list' | 'detail'>('list');
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [selectedExtra, setSelectedExtra] = useState<any>(null);
  const [selectedPorts, setSelectedPorts] = useState<any[]>([]);
  const [scanPortsInput, setScanPortsInput] = useState<string>('');
  const [domainSuffix, setDomainSuffix] = useState('frpc.zyckj.club');
  const [showTunnelModal, setShowTunnelModal] = useState(false);
  const [tunnelForm, setTunnelForm] = useState({
    name: '',
    type: 'tcp',
    local_ip: '',
    local_port: '',
    remote_port: '',
    domain: '',
  });
  const [isScanning, setIsScanning] = useState(false);
  const [isScanningPorts, setIsScanningPorts] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [scanStatus, setScanStatus] = useState<any>(null);
  const [scanError, setScanError] = useState<string>('');
  const itemsPerPage = 10;

  const refreshDevices = async () => {
    const status = filterType === 'all' ? 'all' : filterType;
    // 该页面使用前端分页，这里拉一个较大的 page_size 避免只拿到后端默认 20 条
    const res = await api.devices({ status, page: 1, page_size: 1000 });
    const mapped: Device[] = (res.devices || []).map((d: any) => ({
      ip: d.ip,
      mac: d.mac,
      name: d.name || d.ip,
      vendor: d.vendor || 'Unknown',
      model: d.model || '',
      type: (d.type || 'pc') as any,
      status: (d.status || 'offline') as any,
      lastSeen: d.last_seen || '',
      ports: d.open_ports || [],
    }));
    setDevices(mapped);
  };

  useEffect(() => {
    refreshDevices().catch(() => {});
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filterType]);

  // 初次进入：拉一次扫描状态（避免刷新后 UI 不知道是否正在扫描）
  useEffect(() => {
    api.scanStatus()
      .then((s) => setScanStatus(s))
      .catch(() => {});
  }, []);

  // domain_suffix：用于从端口一键创建 HTTP/HTTPS 隧道时拼接域名
  useEffect(() => {
    api.configGet()
      .then((cfg) => {
        const ds = (cfg?.frp_server?.domain_suffix || '').trim();
        if (ds) setDomainSuffix(ds.replace(/^\./, ''));
      })
      .catch(() => {});
  }, []);

  // realtime scan status
  useEffect(() => {
    if (!rt.scanStatus) return;
    setScanStatus(rt.scanStatus);
  }, [rt.scanStatus]);

  // 兜底：扫描中轮询状态（某些环境 ws 断连时仍能看到进度）
  useEffect(() => {
    const st = scanStatus?.status;
    const running = st === 'running';
    if (!running) return;
    const timer = window.setInterval(() => {
      api.scanStatus()
        .then((s) => setScanStatus(s))
        .catch(() => {});
    }, 2000);
    return () => window.clearInterval(timer);
  }, [scanStatus?.status]);

  // 扫描结束：刷新列表（兜底保证 UI 能看到最终设备）
  useEffect(() => {
    const st = scanStatus?.status;
    if (st === 'completed') {
      refreshDevices().catch(() => {});
    }
  }, [scanStatus?.status]);

  // 扫描开始：清空历史列表（后端已清空 DB，这里同步清空前端状态）
  useEffect(() => {
    const st = scanStatus?.status;
    if (st === 'running') {
      setDevices([]);
      setCurrentPage(1);
      refreshDevices().catch(() => {});
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [scanStatus?.status]);

  // realtime upsert/status update: merge into list
  useEffect(() => {
    const map = rt.devicesByIp;
    if (!map) return;
    setDevices((prev) => {
      const byIp = new Map(prev.map((d) => [d.ip, d]));
      Object.keys(map).forEach((ip) => {
        const cur = byIp.get(ip) || ({} as any);
        const patch = map[ip] || {};
        byIp.set(ip, {
          ip,
          mac: patch.mac ?? cur.mac ?? '',
          name: patch.name ?? cur.name ?? ip,
          vendor: patch.vendor ?? cur.vendor ?? 'Unknown',
          type: (patch.type ?? cur.type ?? 'pc') as any,
          status: (patch.status ?? cur.status ?? 'offline') as any,
          lastSeen: patch.last_seen ?? cur.lastSeen ?? '',
          ports: cur.ports,
        });
      });
      return Array.from(byIp.values());
    });
  }, [rt.devicesByIp]);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchTerm, filterType]);

  const filtered = devices.filter(d => {
    const matchesSearch = d.name.toLowerCase().includes(searchTerm.toLowerCase()) || d.ip.includes(searchTerm);
    const matchesType = filterType === 'all' ||
                        (filterType === 'online' && d.status === 'online') ||
                        (filterType === 'offline' && d.status === 'offline');
    return matchesSearch && matchesType;
  });

  const totalPages = Math.ceil(filtered.length / itemsPerPage);
  const currentDevices = filtered.slice(
    (currentPage - 1) * itemsPerPage,
    currentPage * itemsPerPage
  );

  const getIcon = (type: string) => {
    switch(type) {
      case 'server': return <Server size={18} />;
      case 'camera': return <Camera size={18} />;
      case 'mobile': return <Smartphone size={18} />;
      default: return <Monitor size={18} />;
    }
  };

  const handleScan = () => {
    if (isScanning || scanStatus?.status === 'running') return;
    setScanError('');
    setIsScanning(true);
    api.scanStart()
      .then(() => api.scanStatus().then((s) => setScanStatus(s)).catch(() => {}))
      .catch(() => {
        // 扫描重复触发时后端可能返回“扫描已在进行中”，这里无需打断 UI
      })
      .finally(() => setIsScanning(false));
  };

  const handleScanPorts = async () => {
    if (isScanningPorts) return;
    if (!selectedDevice?.ip) return;
    setIsScanningPorts(true);
    try {
      await api.deviceScanPorts(selectedDevice.ip, scanPortsInput);
      // 轮询刷新端口（最多 15 秒）
      for (let i = 0; i < 15; i++) {
        // eslint-disable-next-line no-await-in-loop
        await new Promise((r) => setTimeout(r, 1000));
        // eslint-disable-next-line no-await-in-loop
        const detail = await api.deviceDetail(selectedDevice.ip);
        const ports = Array.isArray(detail?.open_ports) ? detail.open_ports : [];
        setSelectedPorts(ports);
        // 同步更新 selectedDevice 的 ports (number[]) 兜底
        setSelectedDevice((prev) => (prev ? { ...prev, ports: ports.map((p: any) => Number(p?.port ?? p)).filter((n: any) => Number.isFinite(n)) } : prev));
      }
    } catch (e: any) {
      // ignore
    } finally {
      setIsScanningPorts(false);
    }
  };

  const openTunnelModalForPort = (port: number) => {
    if (!selectedDevice?.ip) return;
    const ip = selectedDevice.ip;
    const p = String(port);
    setTunnelForm({
      name: `${ip.replace(/\./g, '_')}_${p}`,
      type: 'tcp',
      local_ip: ip,
      local_port: p,
      remote_port: p,
      domain: '',
    });
    setShowTunnelModal(true);
  };

  const closeTunnelModal = () => setShowTunnelModal(false);

  const saveTunnel = async () => {
    const name = tunnelForm.name.trim();
    if (!name) {
      alert('请输入隧道名称');
      return;
    }
    const localIP = tunnelForm.local_ip.trim();
    const localPort = Number(tunnelForm.local_port);
    const remotePort = Number(tunnelForm.remote_port);
    if (!localIP) {
      alert('请输入本地 IP');
      return;
    }
    if (!Number.isFinite(localPort) || localPort < 1 || localPort > 65535) {
      alert('请输入有效的本地端口');
      return;
    }
    if (!Number.isFinite(remotePort) || remotePort < 0 || remotePort > 65535) {
      alert('请输入有效的远程端口（0-65535）');
      return;
    }

    let domain: string | undefined = undefined;
    if (tunnelForm.type === 'http' || tunnelForm.type === 'https') {
      const v = tunnelForm.domain.trim();
      if (v) {
        if (v.includes('.')) {
          domain = v;
        } else {
          const ds = domainSuffix.replace(/^\./, '');
          domain = ds ? `${v}.${ds}` : v;
        }
      }
    }

    try {
      await api.frpAddTunnel({
        name,
        type: tunnelForm.type,
        local_ip: localIP,
        local_port: localPort,
        remote_port: remotePort,
        domain,
      });
      closeTunnelModal();
      alert('隧道已创建');
    } catch (e: any) {
      alert(e?.message || String(e));
    }
  };

  const filterOptions = [
    { label: t('devices.all_types'), value: 'all' },
    { label: t('common.online'), value: 'online' },
    { label: t('common.offline'), value: 'offline' }
  ];

  if (view === 'detail' && selectedDevice) {
    return (
      <div>
        <Button variant="outline" onClick={() => setView('list')} style={{ marginBottom: 16 }}>← {t('common.back')}</Button>
        <div className="grid-2">
           <Card title={t('devices.device_info')} extra={<Badge status={selectedDevice.status} text={t(`common.${selectedDevice.status}`)} />}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                 <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
                    <div style={{ padding: 16, background: '#f0f2f5', borderRadius: '50%' }}>{getIcon(selectedDevice.type)}</div>
                    <div>
                      <h3 style={{ margin: 0 }}>{selectedDevice.name}</h3>
                      <div style={{ color: '#666' }}>
                        {selectedDevice.vendor}
                        {selectedDevice.model ? ` · ${selectedDevice.model}` : ''}
                      </div>
                    </div>
                 </div>
                 <div style={{ height: 1, background: '#f0f0f0' }} />
                 <div className="grid-2">
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.ip')}</label><div>{selectedDevice.ip}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.mac')}</label><div>{selectedDevice.mac}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.last_seen')}</label><div>{selectedDevice.lastSeen}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.type')}</label><div style={{ textTransform: 'capitalize' }}>{selectedDevice.type}</div></div>
                 </div>
                  {selectedExtra ? (
                    <>
                      <div style={{ height: 1, background: '#f0f0f0' }} />
                      <div>
                        <label style={{ fontSize: 12, color: '#999' }}>识别信息（调试）</label>
                        <pre style={{ margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-word', fontSize: 12, color: '#333' }}>
                          {JSON.stringify(selectedExtra, null, 2)}
                        </pre>
                      </div>
                    </>
                  ) : null}
              </div>
           </Card>
           
           <Card title={t('devices.open_ports')}>
              <div style={{ display: 'flex', gap: 12, marginBottom: 12 }}>
                <Input
                  style={{ flex: 1 } as any}
                  value={scanPortsInput}
                  onChange={(e) => setScanPortsInput(e.target.value)}
                  placeholder="端口范围/列表（可选）：例如 80,443,3000-3010"
                />
              </div>
              {(selectedPorts && selectedPorts.length > 0) || (selectedDevice.ports && selectedDevice.ports.length > 0) ? (
                <table className="table">
                  <thead><tr><th>{t('devices.port')}</th><th>{t('devices.protocol')}</th><th>服务</th><th>{t('devices.state')}</th><th>{t('common.action')}</th></tr></thead>
                  <tbody>
                    {(selectedPorts && selectedPorts.length > 0
                      ? selectedPorts
                      : (selectedDevice.ports || []).map((p: number) => ({ port: p, protocol: 'tcp', service: '', status: 'open' }))
                    ).map((p: any) => {
                      const port = Number(p?.port ?? 0);
                      const protocol = String(p?.protocol || 'tcp').toUpperCase();
                      const service = String(p?.service || '-');
                      const st = String(p?.status || 'open').toLowerCase();
                      return (
                        <tr key={`${port}-${protocol}`}>
                          <td>{port}</td>
                          <td>{protocol}</td>
                          <td>{service}</td>
                          <td><Badge status={st === 'open' ? 'online' : 'offline'} text={st === 'open' ? t('devices.open') : t('common.offline')} /></td>
                          <td>
                            <Button
                              variant="outline"
                              style={{ fontSize: 13, padding: '4px 10px' }}
                              onClick={() => openTunnelModalForPort(port)}
                            >
                              添加隧道
                            </Button>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              ) : (
                <div style={{ textAlign: 'center', color: '#999', padding: 20 }}>{t('devices.no_ports')}</div>
              )}
              <div style={{ marginTop: 16, textAlign: 'right' }}>
                <Button 
                    variant="outline" 
                    style={{ fontSize: 13 }} 
                    onClick={handleScanPorts}
                    disabled={isScanningPorts}
                >
                    <Radar size={16} className={isScanningPorts ? 'animate-spin' : ''} />
                    {isScanningPorts ? t('devices.scanning') : t('devices.scan_ports')}
                </Button>
              </div>
           </Card>
        </div>

        {showTunnelModal ? (
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
              if (e.target === e.currentTarget) closeTunnelModal();
            }}
          >
            <Card title="添加隧道" style={{ width: '90%', maxWidth: 560 }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
                <div>
                  <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>名称</label>
                  <Input value={tunnelForm.name} onChange={(e) => setTunnelForm({ ...tunnelForm, name: (e.target as any).value })} />
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>类型</label>
                  <Select
                    value={tunnelForm.type}
                    onChange={(value) => setTunnelForm({ ...tunnelForm, type: value })}
                    options={[
                      { label: 'TCP', value: 'tcp' },
                      { label: 'HTTP', value: 'http' },
                      { label: 'HTTPS', value: 'https' },
                    ]}
                  />
                </div>

                <div className="grid-2" style={{ gap: 12 }}>
                  <div>
                    <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>本地 IP</label>
                    <Input value={tunnelForm.local_ip} onChange={(e) => setTunnelForm({ ...tunnelForm, local_ip: (e.target as any).value })} />
                  </div>
                  <div>
                    <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>本地端口</label>
                    <Input type="number" value={tunnelForm.local_port} onChange={(e) => setTunnelForm({ ...tunnelForm, local_port: (e.target as any).value })} />
                  </div>
                </div>

                <div>
                  <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>远程端口</label>
                  <Input type="number" value={tunnelForm.remote_port} onChange={(e) => setTunnelForm({ ...tunnelForm, remote_port: (e.target as any).value })} />
                </div>

                {(tunnelForm.type === 'http' || tunnelForm.type === 'https') ? (
                  <div>
                    <label style={{ display: 'block', marginBottom: 8, fontSize: 14, fontWeight: 500 }}>
                      域名前缀 <span style={{ color: '#999' }}>(默认后缀 .{domainSuffix})</span>
                    </label>
                    <Input value={tunnelForm.domain} onChange={(e) => setTunnelForm({ ...tunnelForm, domain: (e.target as any).value })} placeholder="例如：e6666666" />
                  </div>
                ) : null}

                <div style={{ display: 'flex', gap: 12, justifyContent: 'flex-end', marginTop: 6 }}>
                  <Button variant="outline" onClick={closeTunnelModal}>取消</Button>
                  <Button variant="primary" onClick={saveTunnel}>创建</Button>
                </div>
              </div>
            </Card>
          </div>
        ) : null}
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 24, alignItems: 'center' }}>
         <div style={{ display: 'flex', gap: 16, flex: 1 }}>
            <SearchInput 
              placeholder={t('common.search')}
              width={320} 
              value={searchTerm}
              onChange={e => setSearchTerm(e.target.value)}
            />
            <Select 
              width={160} 
              options={filterOptions}
              value={filterType}
              onChange={setFilterType}
            />
         </div>
         <Button onClick={handleScan} disabled={isScanning || scanStatus?.status === 'running'}>
           <RefreshCw size={18} className={isScanning ? 'animate-spin' : ''} /> 
           {(isScanning || scanStatus?.status === 'running') ? t('devices.scanning') : t('devices.scan_network')}
         </Button>
      </div>

      {scanError ? <Alert type="error">{scanError}</Alert> : null}
      {scanStatus?.status === 'running' ? (
        <Card style={{ marginBottom: 16 }}>
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 12 }}>
            <div style={{ fontWeight: 600 }}>扫描中</div>
            <div style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
              进度 {Number(scanStatus?.progress || 0)}% · 已扫描 {Number(scanStatus?.scanned_count || 0)} · 发现 {Number(scanStatus?.found_count || 0)}
            </div>
          </div>
          <ProgressBar value={Math.max(0, Math.min(100, Number(scanStatus?.progress || 0)))} />
          <div style={{ marginTop: 12, display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              variant="outline"
              onClick={() => {
                setScanError('');
                api.scanStop()
                  .then(() => api.scanStatus().then((s) => setScanStatus(s)).catch(() => {}))
                  .catch((e: any) => setScanError(e?.message || '停止扫描失败'));
              }}
            >
              停止扫描
            </Button>
          </div>
        </Card>
      ) : null}

      <Card>
        <table className="table">
          <thead>
            <tr>
              <th style={{ width: 50 }}></th>
              <th>{t('devices.name')}</th>
              <th>{t('devices.ip')}</th>
              <th>{t('devices.mac')}</th>
              <th>{t('devices.vendor')}</th>
              <th>{t('devices.status')}</th>
              <th>{t('common.action')}</th>
            </tr>
          </thead>
          <tbody>
            {currentDevices.map(d => (
              <tr key={d.ip}>
                <td style={{ color: '#666' }}>{getIcon(d.type)}</td>
                <td style={{ fontWeight: 500 }}>{d.name}</td>
                <td>{d.ip}</td>
                <td style={{ fontFamily: 'monospace', color: '#666' }}>{d.mac}</td>
                <td>{d.vendor}</td>
                <td><Badge status={d.status} text={t(`common.${d.status}`)} /></td>
                <td>
                  <Button
                    variant="ghost"
                    style={{ padding: '4px 8px', fontSize: 13 }}
                    onClick={() => {
                      setSelectedDevice(d);
                      setSelectedExtra(null);
                      setSelectedPorts([]);
                      setView('detail');
                      // 拉取详情（拿 model/extra/端口信息等）
                      api.deviceDetail(d.ip)
                        .then((detail) => {
                          setSelectedDevice((prev) => (prev ? { ...prev, model: detail?.model || prev.model || '' } : prev));
                          setSelectedPorts(Array.isArray(detail?.open_ports) ? detail.open_ports : []);
                          try {
                            setSelectedExtra(detail?.extra ? JSON.parse(detail.extra) : null);
                          } catch {
                            setSelectedExtra(detail?.extra || null);
                          }
                        })
                        .catch(() => {});
                    }}
                  >
                    {t('common.detail')}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {totalPages > 1 && (
          <Pagination
            currentPage={currentPage}
            totalPages={totalPages}
            onPageChange={setCurrentPage}
            texts={{
              prev: t('common.prev_page'),
              next: t('common.next_page'),
              info: `${t('common.page')} ${currentPage} ${t('common.of')} ${totalPages}`
            }}
          />
        )}
      </Card>
    </div>
  );
};
