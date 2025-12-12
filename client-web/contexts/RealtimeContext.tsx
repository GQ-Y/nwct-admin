import React, { createContext, useContext, useEffect, useMemo, useRef, useState } from "react";
import { openWebSocket, RealtimeMessage } from "../lib/ws";

type RealtimeState = {
  connected: boolean;
  lastHello?: any;
  systemStatus?: any;
  npsStatus?: any;
  mqttLogNew?: any;
  devicesByIp: Record<string, any>;
};

const Ctx = createContext<RealtimeState | null>(null);

export const RealtimeProvider: React.FC<{ children: React.ReactNode; enabled: boolean }> = ({
  children,
  enabled,
}) => {
  const [connected, setConnected] = useState(false);
  const [lastHello, setLastHello] = useState<any>(null);
  const [systemStatus, setSystemStatus] = useState<any>(null);
  const [npsStatus, setNpsStatus] = useState<any>(null);
  const [mqttLogNew, setMqttLogNew] = useState<any>(null);
  const [devicesByIp, setDevicesByIp] = useState<Record<string, any>>({});

  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    if (!enabled) return;
    const ws = openWebSocket((msg: RealtimeMessage) => {
      if (msg.type === "hello") {
        setLastHello(msg.data);
        return;
      }
      if (msg.type === "event") {
        switch (msg.event) {
          case "system_status":
            setSystemStatus(msg.data);
            return;
          case "nps_status_changed":
            setNpsStatus(msg.data);
            return;
          case "mqtt_log_new":
            setMqttLogNew(msg.data);
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
    ws.onopen = () => setConnected(true);
    ws.onclose = () => setConnected(false);
    ws.onerror = () => setConnected(false);

    return () => {
      try {
        ws.close();
      } catch {}
      wsRef.current = null;
    };
  }, [enabled]);

  const value = useMemo<RealtimeState>(
    () => ({ connected, lastHello, systemStatus, npsStatus, mqttLogNew, devicesByIp }),
    [connected, lastHello, systemStatus, npsStatus, mqttLogNew, devicesByIp]
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
};

export function useRealtime() {
  const v = useContext(Ctx);
  if (!v) throw new Error("RealtimeProvider missing");
  return v;
}


