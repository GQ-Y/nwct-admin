import React, { useState, useEffect } from 'react';
import { Card, Button, SearchInput, Select, Badge, Pagination, ProgressBar, Alert } from '../components/UI';
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
  const [filterType, setFilterType] = useState('all');
  const [view, setView] = useState<'list' | 'detail'>('list');
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);
  const [selectedExtra, setSelectedExtra] = useState<any>(null);
  const [isScanning, setIsScanning] = useState(false);
  const [isScanningPorts, setIsScanningPorts] = useState(false);
  const [currentPage, setCurrentPage] = useState(1);
  const [scanStatus, setScanStatus] = useState<any>(null);
  const [scanError, setScanError] = useState<string>('');
  const itemsPerPage = 10;

  const refreshDevices = async () => {
    const res = await api.devices();
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
  }, []);

  // 初次进入：拉一次扫描状态（避免刷新后 UI 不知道是否正在扫描）
  useEffect(() => {
    api.scanStatus()
      .then((s) => setScanStatus(s))
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

  const handleScanPorts = () => {
    if (isScanningPorts) return;
    setIsScanningPorts(true);
    setTimeout(() => {
        setIsScanningPorts(false);
        // In a real app, this would update the ports list
    }, 3000);
  }

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
              {selectedDevice.ports ? (
                <table className="table">
                  <thead><tr><th>{t('devices.port')}</th><th>{t('devices.protocol')}</th><th>{t('devices.state')}</th></tr></thead>
                  <tbody>
                    {selectedDevice.ports.map((p: number) => (
                      <tr key={p}>
                        <td>{p}</td>
                        <td>TCP</td>
                        <td><Badge status="online" text={t('devices.open')} /></td>
                      </tr>
                    ))}
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
                      setView('detail');
                      // 拉取详情（拿 model/extra/端口信息等）
                      api.deviceDetail(d.ip)
                        .then((detail) => {
                          setSelectedDevice((prev) => (prev ? { ...prev, model: detail?.model || prev.model || '' } : prev));
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
