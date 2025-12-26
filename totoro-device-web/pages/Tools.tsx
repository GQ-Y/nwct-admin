import React, { useMemo, useState } from 'react';
import { Card, Button, Input } from '../components/UI';
import { Activity, Globe, Zap, Radio, Terminal } from 'lucide-react';
import { useLanguage } from '../contexts/LanguageContext';
import { api } from '../lib/api';

const ToolTab: React.FC<{ icon: any, label: string, active: boolean, onClick: () => void }> = ({ icon, label, active, onClick }) => (
  <div 
    onClick={onClick}
    style={{ 
      padding: '12px 24px', 
      cursor: 'pointer', 
      display: 'flex', 
      alignItems: 'center', 
      gap: 8,
      color: active ? 'var(--primary)' : 'var(--text-secondary)',
      fontWeight: active ? 600 : 500,
      position: 'relative',
      transition: 'all 0.3s ease',
      borderRadius: '12px',
      background: active ? 'rgba(10, 89, 247, 0.08)' : 'transparent'
    }}
  >
    {icon} 
    <span>{label}</span>
    {active && (
       <div style={{
          position: 'absolute',
          bottom: 0,
          left: '50%',
          transform: 'translateX(-50%)',
          width: '20px',
          height: '3px',
          background: 'var(--primary)',
          borderRadius: '2px'
       }} />
    )}
  </div>
);

export const Tools: React.FC = () => {
  const { t } = useLanguage();
  const [activeTab, setActiveTab] = useState('ping');
  const [output, setOutput] = useState<string[]>([]);
  const [target, setTarget] = useState('');
  const [running, setRunning] = useState(false);

  const title = useMemo(() => {
    return activeTab === 'ping'
      ? t('tools.ping')
      : activeTab === 'trace'
        ? t('tools.traceroute')
        : activeTab === 'speed'
          ? t('tools.speedtest')
          : t('tools.portscan');
  }, [activeTab, t]);

  const runTool = async () => {
    const tg = target.trim();
    setOutput([]);
    setRunning(true);
    try {
      if (activeTab === 'ping') {
        if (!tg) throw new Error('请输入目标 IP 或域名');
        // 先解析域名到 IP（如果输入本身是 IP，则跳过）
        let resolved = '';
        if (!/^(?:\\d{1,3}\\.){3}\\d{1,3}$/.test(tg)) {
          try {
            const dns = await api.toolsDNS({ query: tg, type: 'A' });
            const recs = Array.isArray(dns?.records) ? dns.records : [];
            // 兼容后端可能返回 string 或对象
            for (const r of recs) {
              if (typeof r === 'string' && r.trim()) {
                resolved = r.trim();
                break;
              }
              if (r && typeof r === 'object') {
                const v = String((r.value ?? r.data ?? r.ip ?? '') || '').trim();
                if (v) {
                  resolved = v;
                  break;
                }
              }
            }
          } catch {
            // 忽略 DNS 失败：仍然可以直接 ping
          }
        }
        setOutput(['Starting...', resolved ? `PING ${tg} (${resolved})` : `PING ${tg}`]);
        const r = await api.toolsPing({ target: tg, count: 4, timeout: 5 });
        const lines: string[] = [];
        if (Array.isArray(r?.results)) {
          for (const p of r.results) {
            const seq = p?.sequence ?? '-';
            if (p?.status === 'success') {
              lines.push(`icmp_seq=${seq} time=${Number(p?.latency || 0).toFixed(1)} ms`);
            } else {
              lines.push(`icmp_seq=${seq} timeout`);
            }
          }
        }
        lines.push(
          `--- ${r?.target || tg} ping statistics ---`,
          `${r?.packets_sent ?? 0} packets transmitted, ${r?.packets_received ?? 0} received, ${Number(r?.packet_loss ?? 0).toFixed(1)}% packet loss`,
          `round-trip min/avg/max = ${Number(r?.min_latency ?? 0).toFixed(1)}/${Number(r?.avg_latency ?? 0).toFixed(1)}/${Number(r?.max_latency ?? 0).toFixed(1)} ms`,
          'Done.'
        );
        setOutput((prev) => [...prev, ...lines]);
        return;
      }

      if (activeTab === 'trace') {
        setOutput(['Starting traceroute...']);
        const r = await api.toolsTraceroute({ target: tg || undefined, max_hops: 30, timeout: 5 });
        const lines: string[] = [];
        const hops = Array.isArray(r?.hops) ? r.hops : [];
        lines.push(`traceroute to ${r?.target || tg || '(gateway)'}`);
        for (const h of hops) {
          const hop = h?.hop ?? '';
          const ip = h?.ip || '*';
          const lat = typeof h?.latency === 'number' && h.latency > 0 ? `${h.latency.toFixed(1)} ms` : '';
          lines.push(`${hop}\t${ip}\t${lat}`.trim());
        }
        lines.push('Done.');
        setOutput(lines);
        return;
      }

      if (activeTab === 'speed') {
        setOutput(['Starting speed test...']);
        // 用后端 web 模式（更轻量，能测 DNS/TCP/TLS/TTFB）
        const r = await api.toolsSpeedtest({ mode: 'web', url: tg || 'default', method: 'GET', count: 3, timeout: 8 });
        const lines: string[] = [];
        lines.push(`URL: ${r?.url || tg || 'default'}`);
        if (Array.isArray(r?.attempts)) {
          r.attempts.forEach((a: any, idx: number) => {
            if (a?.ok) {
              lines.push(
                `#${idx + 1} OK status=${a.status_code} dns=${a.dns_ms}ms conn=${a.connect_ms}ms tls=${a.tls_ms}ms ttfb=${a.ttfb_ms}ms total=${a.total_ms}ms bytes=${a.bytes_read}`
              );
            } else {
              lines.push(`#${idx + 1} FAIL ${a?.error || 'unknown error'}`);
            }
          });
        }
        if (r?.summary) {
          lines.push(`summary: ${JSON.stringify(r.summary)}`);
        }
        lines.push('Done.');
        setOutput(lines);
        return;
      }

      // port scan
      if (!tg) throw new Error('请输入目标 IP 或域名');
      setOutput(['Starting port scan...']);
      const r = await api.toolsPortscan({ target: tg, ports: '1-1024', timeout: 1, scan_type: 'tcp' });
      const lines: string[] = [];
      lines.push(`target: ${r?.target || tg}`);
      lines.push(`scanned_ports: ${r?.scanned_ports ?? 0}`);
      const open = Array.isArray(r?.open_ports) ? r.open_ports : [];
      if (open.length === 0) {
        lines.push('open_ports: (none)');
      } else {
        lines.push('open_ports:');
        for (const p of open) {
          lines.push(`- ${p?.port}/${p?.protocol} ${p?.service || ''}`.trim());
        }
      }
      lines.push('Done.');
      setOutput(lines);
    } catch (e: any) {
      setOutput([`Error: ${e?.message || String(e)}`]);
    } finally {
      setRunning(false);
    }
  };

  return (
    <div>
      <div style={{ 
        display: 'flex', 
        marginBottom: 32, 
        background: 'rgba(255,255,255,0.6)', 
        backdropFilter: 'blur(20px)',
        padding: '6px',
        borderRadius: '16px',
        justifyContent: 'space-between',
        gap: 8,
        flexWrap: 'wrap'
      }}>
        <div style={{ display: 'flex', gap: 8 }}>
            <ToolTab icon={<Activity size={18} />} label={t('tools.ping')} active={activeTab === 'ping'} onClick={() => setActiveTab('ping')} />
            <ToolTab icon={<Globe size={18} />} label={t('tools.traceroute')} active={activeTab === 'trace'} onClick={() => setActiveTab('trace')} />
            <ToolTab icon={<Zap size={18} />} label={t('tools.speedtest')} active={activeTab === 'speed'} onClick={() => setActiveTab('speed')} />
            <ToolTab icon={<Radio size={18} />} label={t('tools.portscan')} active={activeTab === 'port'} onClick={() => setActiveTab('port')} />
        </div>
      </div>

      <div style={{ maxWidth: 900, margin: '0 auto' }}>
        <Card 
            title={
                <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
                    <Terminal size={20} color="var(--primary)" />
                    {title}
                </div>
            }
            glass
        >
           <div style={{ display: 'flex', gap: 16, marginBottom: 24 }}>
             <Input 
                placeholder={activeTab === 'ping' ? t('tools.enter_ip') : t('tools.target')} 
                value={target}
                onChange={e => setTarget(e.target.value)}
                style={{ flex: 1 }}
             />
             <Button onClick={runTool} disabled={running} style={{ minWidth: 120 }}>
               {running ? t('common.loading') : t('common.start')}
             </Button>
           </div>
           
           <div style={{ 
             background: '#1E1E1E', 
             color: '#E0E0E0', 
             padding: '20px', 
             borderRadius: '16px', 
             minHeight: 360, 
             fontFamily: 'SF Mono, Consolas, Monaco, monospace', 
             fontSize: '14px',
             lineHeight: '1.6',
             boxShadow: 'inset 0 2px 10px rgba(0,0,0,0.2)'
           }}>
             <div style={{ color: '#666', marginBottom: 12 }}>// {t('tools.console_output')}</div>
             {output.map((line, i) => (
               <div key={i} style={{ animation: 'fadeIn 0.2s ease-in' }}>{line}</div>
             ))}
             {output.length === 0 && <div style={{ color: '#555', fontStyle: 'italic' }}>{t('tools.ready')}</div>}
           </div>
        </Card>
      </div>
    </div>
  );
};
