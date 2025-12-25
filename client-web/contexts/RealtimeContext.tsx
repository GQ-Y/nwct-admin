import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { openWebSocket, RealtimeMessage } from "../lib/ws";

type RealtimeState = {
  connected: boolean;
  lastHello?: any;
  systemStatus?: any;
  frpStatus?: any;
  devicesByIp: Record<string, any>;
  scanStatus?: any;
};

const Ctx = createContext<RealtimeState | null>(null);

export const RealtimeProvider: React.FC<{ children: React.ReactNode; enabled: boolean }> = ({
  children,
  enabled,
}) => {
  const [connected, setConnected] = useState(false);
  const [lastHello, setLastHello] = useState<any>(null);
  const [systemStatus, setSystemStatus] = useState<any>(null);
  const [frpStatus, setFrpStatus] = useState<any>(null);
  const [devicesByIp, setDevicesByIp] = useState<Record<string, any>>({});
  const [scanStatus, setScanStatus] = useState<any>(null);

  const wsRef = useRef<WebSocket | null>(null);
  const retryRef = useRef<number | null>(null);
  const attemptRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;
    let stopped = false;

    const clearRetry = () => {
      if (retryRef.current != null) {
        window.clearTimeout(retryRef.current);
        retryRef.current = null;
      }
    };

    const scheduleReconnect = () => {
      if (stopped) return;
      clearRetry();
      const attempt = Math.min(attemptRef.current, 6);
      const delay = [500, 1000, 2000, 5000, 8000, 15000, 30000][attempt] || 30000;
      attemptRef.current = attemptRef.current + 1;
      retryRef.current = window.setTimeout(() => {
        connect();
      }, delay);
    };

    const connect = () => {
      if (stopped) return;
      try {
        wsRef.current?.close();
      } catch {}

      const ws = openWebSocket((msg: RealtimeMessage) => {
      if (msg.type === "hello") {
        setLastHello(msg.data);
        if (msg.data?.scan_status) setScanStatus(msg.data.scan_status);
        if (msg.data?.frp_status) setFrpStatus(msg.data.frp_status);
        return;
      }
      if (msg.type === "event") {
        switch (msg.event) {
          case "system_status":
            setSystemStatus(msg.data);
            return;
          case "frp_status_changed":
            setFrpStatus(msg.data);
            return;
          case "scan_started":
            setScanStatus((prev: any) => ({ ...(prev || {}), status: "running", progress: 0, ...(msg.data || {}) }));
            return;
          case "scan_progress":
            setScanStatus((msg.data || null) as any);
            return;
          case "scan_done":
            setScanStatus((msg.data || null) as any);
            return;
          case "device_upsert": {
            const ip = msg.data?.ip;
            if (!ip) return;
            setDevicesByIp((prev) => ({ ...prev, [ip]: { ...(prev[ip] || {}), ...msg.data } }));
            return;
          }
          case "device_status_changed": {
            const ip = msg.data?.ip;
            if (!ip) return;
            setDevicesByIp((prev) => ({
              ...prev,
              [ip]: { ...(prev[ip] || {}), status: msg.data.status, last_seen: msg.data.ts },
            }));
            return;
          }
          default:
            return;
        }
      }
      });

      wsRef.current = ws;
      ws.onopen = () => {
        attemptRef.current = 0;
        setConnected(true);
      };
      ws.onclose = () => {
        setConnected(false);
        scheduleReconnect();
      };
      ws.onerror = () => {
        setConnected(false);
        // 有些浏览器会同时触发 onerror+onclose，避免在这里额外 close
      };
    };

    connect();

    return () => {
      stopped = true;
      clearRetry();
      try {
        wsRef.current?.close();
      } catch {}
      wsRef.current = null;
    };
  }, [enabled]);

  const value = useMemo<RealtimeState>(
    () => ({ connected, lastHello, systemStatus, frpStatus, devicesByIp, scanStatus }),
    [connected, lastHello, systemStatus, frpStatus, devicesByIp, scanStatus]
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
};

export function useRealtime() {
  const v = useContext(Ctx);
  if (!v) throw new Error("RealtimeProvider missing");
  return v;
}


