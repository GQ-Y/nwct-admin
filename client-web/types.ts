
export interface Device {
  ip: string;
  mac: string;
  name: string;
  type: 'server' | 'camera' | 'iot' | 'pc' | 'mobile';
  vendor: string;
  model?: string;
  status: 'online' | 'offline';
  lastSeen: string;
  ports?: number[];
}

export interface SystemStats {
  cpu: number;
  memory: number;
  storage: number;
  uptime: string;
  netIn: number;
  netOut: number;
}

export interface LogEntry {
  id: number;
  timestamp: string;
  level: 'info' | 'warn' | 'error';
  module: string;
  message: string;
}

export interface NPSTunnel {
  id: number;
  type: string;
  local: string;
  remote: number;
  status: 'online' | 'offline';
}

export interface MQTTMessage {
  id: string;
  topic: string;
  payload: string;
  qos: 0 | 1 | 2;
  direction: 'in' | 'out';
  timestamp: string;
}
