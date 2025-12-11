
import React, { useState } from 'react';
import { Card, Button, Input, Badge } from '../components/UI';
import { mockDevices } from '../data';
import { Device } from '../types';
import { Search, RefreshCw, Smartphone, Monitor, Server, Camera } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const Devices: React.FC = () => {
  const { t } = useLanguage();
  const [searchTerm, setSearchTerm] = useState('');
  const [view, setView] = useState<'list' | 'detail'>('list');
  const [selectedDevice, setSelectedDevice] = useState<Device | null>(null);

  const filtered = mockDevices.filter(d => d.name.toLowerCase().includes(searchTerm.toLowerCase()) || d.ip.includes(searchTerm));

  const getIcon = (type: string) => {
    switch(type) {
      case 'server': return <Server size={18} />;
      case 'camera': return <Camera size={18} />;
      case 'mobile': return <Smartphone size={18} />;
      default: return <Monitor size={18} />;
    }
  };

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
                      <div style={{ color: '#666' }}>{selectedDevice.vendor}</div>
                    </div>
                 </div>
                 <div style={{ height: 1, background: '#f0f0f0' }} />
                 <div className="grid-2">
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.ip')}</label><div>{selectedDevice.ip}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.mac')}</label><div>{selectedDevice.mac}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.last_seen')}</label><div>{selectedDevice.lastSeen}</div></div>
                    <div><label style={{ fontSize: 12, color: '#999' }}>{t('devices.type')}</label><div style={{ textTransform: 'capitalize' }}>{selectedDevice.type}</div></div>
                 </div>
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
                <Button variant="outline" style={{ fontSize: 13 }}>{t('devices.scan_ports')}</Button>
              </div>
           </Card>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 24 }}>
         <div style={{ display: 'flex', gap: 12 }}>
            <div style={{ position: 'relative' }}>
              <Search size={16} style={{ position: 'absolute', left: 10, top: 10, color: '#999' }} />
              <Input 
                placeholder={t('common.search')}
                style={{ paddingLeft: 36, width: 300 }} 
                value={searchTerm}
                onChange={e => setSearchTerm(e.target.value)}
              />
            </div>
            <select className="input" style={{ width: 120 }}>
               <option>{t('devices.all_types')}</option>
               <option>{t('common.online')}</option>
               <option>{t('common.offline')}</option>
            </select>
         </div>
         <Button>
           <RefreshCw size={16} /> {t('devices.scan_network')}
         </Button>
      </div>

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
            {filtered.map(d => (
              <tr key={d.ip}>
                <td style={{ color: '#666' }}>{getIcon(d.type)}</td>
                <td style={{ fontWeight: 500 }}>{d.name}</td>
                <td>{d.ip}</td>
                <td style={{ fontFamily: 'monospace', color: '#666' }}>{d.mac}</td>
                <td>{d.vendor}</td>
                <td><Badge status={d.status} text={t(`common.${d.status}`)} /></td>
                <td>
                  <Button variant="ghost" style={{ padding: '4px 8px', fontSize: 13 }} onClick={() => { setSelectedDevice(d); setView('detail'); }}>
                    {t('common.detail')}
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </Card>
    </div>
  );
};
