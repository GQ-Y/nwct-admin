import { API_BASE, getToken } from "./api";

export type RealtimeMessage =
  | { type: "hello"; data?: any; ts: string }
  | { type: "event"; event: string; data?: any; ts: string }
  | { type: string; [k: string]: any };

function toWsBase(apiBase: string): string {
  if (apiBase.startsWith("https://")) return apiBase.replace(/^https:\/\//, "wss://");
  if (apiBase.startsWith("http://")) return apiBase.replace(/^http:\/\//, "ws://");
  // fallback
  return `ws://${apiBase}`;
}

export function openWebSocket(onMessage: (msg: RealtimeMessage) => void): WebSocket {
  const token = getToken();
  const wsBase = toWsBase(API_BASE);
  const url = `${wsBase}/ws${token ? `?token=${encodeURIComponent(token)}` : ""}`;
  const ws = new WebSocket(url);

  ws.onmessage = (e) => {
    try {
      const msg = JSON.parse(e.data);
      onMessage(msg);
    } catch {
      // ignore
    }
  };
  return ws;
}


