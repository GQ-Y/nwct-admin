import React, { useEffect, useMemo, useState } from 'react';
import { Card, Button, Badge, Alert, Input } from '../components/UI';
import { RefreshCw, Download, Trash2, Power, RotateCcw } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';
import { useRealtime } from '../contexts/RealtimeContext';

export const System: React.FC = () => {
  const { t } = useLanguage();
  const rt = useRealtime();
  const [showResetConfirm, setShowResetConfirm] = useState(false);
  const [loading, setLoading] = useState(false);
  const [lines, setLines] = useState(200);
  const [logs, setLogs] = useState<string[]>([]);
  const [source, setSource] = useState<string>("");
  const [fallbackInfo, setFallbackInfo] = useState<any>(null);

  const sys = rt.systemStatus || fallbackInfo;
  const ip = sys?.network?.ip ?? "-";
  const uptimeSec = Number(sys?.uptime ?? 0);
  const hostname = String(sys?.hostname || "").trim() || "-";
  const firmware = String(sys?.firmware_version || "").trim() || "-";
  const uptimeText = useMemo(() => {
    if (!uptimeSec) return "-";
    const d = Math.floor(uptimeSec / 86400);
    const h = Math.floor((uptimeSec % 86400) / 3600);
    const m = Math.floor((uptimeSec % 3600) / 60);
    if (d > 0) return `${d} days, ${h} hours`;
    if (h > 0) return `${h} hours, ${m} min`;
    return `${m} min`;
  }, [uptimeSec]);

  const handleFactoryReset = () => {
    // In a real app, this would trigger the reset process
    alert('Factory reset triggered');
    setShowResetConfirm(false);
  };

  const refreshLogs = async () => {
    setLoading(true);
    try {
      const d = await api.systemLogs(lines);
      setLogs((d?.logs || []).map((x: any) => x?.line).filter(Boolean));
      setSource(d?.source || "");
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  const exportLogs = () => {
    const ts = new Date();
    const pad = (n: number) => String(n).padStart(2, "0");
    const name = `system-${ts.getFullYear()}${pad(ts.getMonth() + 1)}${pad(ts.getDate())}-${pad(ts.getHours())}${pad(ts.getMinutes())}${pad(ts.getSeconds())}.log`;
    const header = source ? `# source: ${source}\n` : "";
    const content = header + logs.join("\n") + "\n";
    const blob = new Blob([content], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = name;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };

  const clearLogs = async () => {
    if (!confirm("确认清空系统日志？")) return;
    setLoading(true);
    try {
      await api.systemLogsClear();
      setLogs([]);
      await refreshLogs();
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshLogs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // System Info 兜底：WS 未连接时也能显示真实数据
  useEffect(() => {
    api.systemInfo()
      .then((d) => setFallbackInfo(d))
      .catch(() => {});
  }, []);

  const reboot = async () => {
    setLoading(true);
    try {
      await api.systemRestart("soft");
      alert("已发送重启请求（soft）");
    } catch (e: any) {
      alert(e?.message || String(e));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="grid-2">
      <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
        <Card title={t('system.info')}>
            <div className="table" style={{ display: 'table' }}>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.hostname')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>{hostname}</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.firmware')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>{firmware}</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666', borderBottom: '1px solid #f0f0f0' }}>{t('system.uptime')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, borderBottom: '1px solid #f0f0f0', textAlign: 'right' }}>{uptimeText}</div>
            </div>
            <div style={{ display: 'table-row' }}>
                <div style={{ display: 'table-cell', padding: '12px 8px', color: '#666' }}>{t('devices.ip')}</div>
                <div style={{ display: 'table-cell', padding: '12px 8px', fontWeight: 500, textAlign: 'right' }}>{ip}</div>
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
                    <Button variant="outline" style={{ color: 'var(--text-primary)' }} onClick={reboot} disabled={loading}>
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

      <Card
        title={t('system.logs')}
        extra={
          <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
            <div style={{ width: 120 }}>
              <Input value={String(lines)} onChange={(e) => setLines(Number((e.target as any).value) || 200)} />
            </div>
            <Button variant="ghost" style={{ padding: 4 }} onClick={refreshLogs} disabled={loading}>
              <RefreshCw size={16} />
            </Button>
          </div>
        }
      >
        {source ? <div style={{ fontSize: 12, color: "#888", marginBottom: 8 }}>source: {source}</div> : null}
        <div style={{ maxHeight: 500, overflowY: 'auto' }}>
          {logs.map((line, idx) => (
            <div key={idx} style={{ padding: "8px 0", borderBottom: "1px solid #f0f0f0", fontSize: 12 }}>
              <div style={{ fontFamily: "monospace", whiteSpace: "pre-wrap", wordBreak: "break-word" }}>{line}</div>
            </div>
          ))}
          {logs.length === 0 ? <div style={{ color: "#888", padding: 12 }}>暂无日志</div> : null}
        </div>
        <div style={{ marginTop: 20, display: 'flex', gap: 12, paddingTop: 16, borderTop: '1px solid #f0f0f0' }}>
          <Button variant="outline" style={{ flex: 1 }} onClick={exportLogs} disabled={loading || logs.length === 0}>
            <Download size={16} /> {t('system.export')}
          </Button>
          <Button variant="outline" style={{ flex: 1 }} onClick={clearLogs} disabled={loading}>
            <Trash2 size={16} /> {t('system.clear')}
          </Button>
        </div>
      </Card>
    </div>
  );
};
