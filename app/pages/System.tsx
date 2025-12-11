
import React from 'react';
import { Card, Button, Badge } from '../components/UI';
import { mockLogs } from '../data';
import { RefreshCw, Download, Trash2, Power } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const System: React.FC = () => {
  const { t } = useLanguage();

  return (
    <div className="grid-2">
      <Card title={t('system.info')}>
        <div className="table" style={{ display: 'table' }}>
           <div style={{ display: 'table-row' }}>
             <div style={{ display: 'table-cell', padding: 8, color: '#666' }}>{t('system.hostname')}</div>
             <div style={{ display: 'table-cell', padding: 8, fontWeight: 500 }}>netadmin-gw-01</div>
           </div>
           <div style={{ display: 'table-row' }}>
             <div style={{ display: 'table-cell', padding: 8, color: '#666' }}>{t('system.firmware')}</div>
             <div style={{ display: 'table-cell', padding: 8, fontWeight: 500 }}>v2.4.1-stable</div>
           </div>
           <div style={{ display: 'table-row' }}>
             <div style={{ display: 'table-cell', padding: 8, color: '#666' }}>{t('system.uptime')}</div>
             <div style={{ display: 'table-cell', padding: 8, fontWeight: 500 }}>15 days, 4 hours</div>
           </div>
           <div style={{ display: 'table-row' }}>
             <div style={{ display: 'table-cell', padding: 8, color: '#666' }}>{t('devices.ip')}</div>
             <div style={{ display: 'table-cell', padding: 8, fontWeight: 500 }}>192.168.1.100</div>
           </div>
        </div>
        <div style={{ marginTop: 24, paddingTop: 24, borderTop: '1px solid #f0f0f0' }}>
           <Button variant="outline" style={{ color: '#f5222d', borderColor: '#f5222d' }}>
             <Power size={16} /> {t('system.reboot')}
           </Button>
        </div>
      </Card>

      <Card title={t('system.logs')} extra={<Button variant="ghost" style={{ padding: 4 }}><RefreshCw size={16}/></Button>}>
        <div style={{ maxHeight: 400, overflowY: 'auto' }}>
          {mockLogs.map(log => (
            <div key={log.id} style={{ padding: '12px 0', borderBottom: '1px solid #f0f0f0', fontSize: 13 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span style={{ color: '#999' }}>{log.timestamp}</span>
                <Badge status={log.level === 'error' ? 'error' : log.level === 'warn' ? 'warn' : 'success'} text={log.level.toUpperCase()} />
              </div>
              <div style={{ fontWeight: 500, marginBottom: 2 }}>[{log.module}]</div>
              <div style={{ color: '#333' }}>{log.message}</div>
            </div>
          ))}
        </div>
        <div style={{ marginTop: 16, display: 'flex', gap: 12 }}>
          <Button variant="outline"><Download size={16} /> {t('system.export')}</Button>
          <Button variant="outline"><Trash2 size={16} /> {t('system.clear')}</Button>
        </div>
      </Card>
    </div>
  );
};
