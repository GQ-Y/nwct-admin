import React, { useState } from 'react';
import { Card, Button, Badge, Alert } from '../components/UI';
import { mockLogs } from '../data';
import { RefreshCw, Download, Trash2, Power, RotateCcw } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';

export const System: React.FC = () => {
  const { t } = useLanguage();
  const [showResetConfirm, setShowResetConfirm] = useState(false);

  const handleFactoryReset = () => {
    // In a real app, this would trigger the reset process
    alert('Factory reset triggered');
    setShowResetConfirm(false);
  };

  return (
    <div className="grid-2">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        <Card title={t('system.info')}>
            <div className="table" style={{ display: 'table' }}>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.hostname')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>netadmin-gw-01</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.firmware')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>v2.4.1-stable</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.uptime')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>15 days, 4 hours</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666' }}>{t('devices.ip')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, textAlign: 'right' }}>192.168.1.100</div>
            </div>
            </div>
        </Card>

        <Card title={t('system.actions') || "System Actions"}>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <div style={{ fontWeight: 500 }}>{t('system.reboot')}</div>
                        <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{t('system.reboot_desc')}</div>
                    </div>
                    <Button variant="outline" style={{ color: 'var(--text-primary)' }}>
                        <Power size={16} /> {t('system.reboot')}
                    </Button>
                </div>
                <div style={{ height: 1, background: '#f0f0f0' }} />
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div>
                        <div style={{ fontWeight: 500, color: 'var(--error)' }}>{t('system.factory_reset')}</div>
                        <div style={{ fontSize: 13, color: 'var(--text-secondary)' }}>{t('system.factory_reset_desc')}</div>
                    </div>
                    {!showResetConfirm ? (
                        <Button variant="outline" style={{ color: 'var(--error)', borderColor: 'rgba(232, 64, 38, 0.3)', background: 'rgba(232, 64, 38, 0.05)' }} onClick={() => setShowResetConfirm(true)}>
                            <RotateCcw size={16} /> {t('system.factory_reset')}
                        </Button>
                    ) : (
                        <div style={{ display: 'flex', gap: 8 }}>
                            <Button variant="ghost" onClick={() => setShowResetConfirm(false)} style={{ padding: '8px 12px' }}>{t('common.cancel')}</Button>
                            <Button variant="primary" style={{ background: 'var(--error)', color: 'white' }} onClick={handleFactoryReset}>{t('common.confirm')}</Button>
                        </div>
                    )}
                </div>
            </div>
        </Card>
      </div>

      <Card title={t('system.logs')} extra={<Button variant="ghost" style={{ padding: 4 }}><RefreshCw size={16}/></Button>}>
        <div style={{ maxHeight: 500, overflowY: 'auto' }}>
          {mockLogs.map(log => (
            <div key={log.id} style={{ padding: '16px 0', borderBottom: '1px solid #f0f0f0', fontSize: 13 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
                <span style={{ color: '#999', fontFamily: 'monospace' }}>{log.timestamp}</span>
                <Badge status={log.level === 'error' ? 'error' : log.level === 'warn' ? 'warn' : 'success'} text={log.level.toUpperCase()} />
              </div>
              <div style={{ fontWeight: 600, marginBottom: 4, color: 'var(--text-primary)' }}>[{log.module}]</div>
              <div style={{ color: 'var(--text-secondary)', lineHeight: 1.5 }}>{log.message}</div>
            </div>
          ))}
        </div>
        <div style={{ marginTop: 20, display: 'flex', gap: 12, paddingTop: 16, borderTop: '1px solid #f0f0f0' }}>
          <Button variant="outline" style={{ flex: 1 }}><Download size={16} /> {t('system.export')}</Button>
          <Button variant="outline" style={{ flex: 1 }}><Trash2 size={16} /> {t('system.clear')}</Button>
        </div>
      </Card>
    </div>
  );
};
