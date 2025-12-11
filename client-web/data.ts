
import { Device, LogEntry, MQTTMessage, NPSTunnel, SystemStats } from './types';

export const mockDevices: Device[] = [
  { ip: '192.168.1.101', mac: 'AA:BB:CC:DD:EE:01', name: 'Main-Server', type: 'server', vendor: 'Dell', status: 'online', lastSeen: 'Just now', ports: [80, 443, 22] },
  { ip: '192.168.1.105', mac: 'AA:BB:CC:DD:EE:02', name: 'Living Room Cam', type: 'camera', vendor: 'Hikvision', status: 'online', lastSeen: '1 min ago', ports: [554, 80] },
  { ip: '192.168.1.108', mac: 'AA:BB:CC:DD:EE:03', name: 'Smart Light', type: 'iot', vendor: 'Xiaomi', status: 'offline', lastSeen: '2 hours ago' },
  { ip: '192.168.1.112', mac: 'AA:BB:CC:DD:EE:04', name: 'Admin PC', type: 'pc', vendor: 'Apple', status: 'online', lastSeen: 'Just now' },
  { ip: '192.168.1.120', mac: 'AA:BB:CC:DD:EE:05', name: 'Guest Mobile', type: 'mobile', vendor: 'Samsung', status: 'online', lastSeen: '5 mins ago' },
];

export const mockStats: SystemStats = {
  cpu: 45,
  memory: 62,
  storage: 28,
  uptime: '15d 4h 32m',
  netIn: 1.2, // MB/s
  netOut: 0.8 // MB/s
};

export const mockLogs: LogEntry[] = [
  { id: 1, timestamp: '2023-10-27 10:00:01', level: 'info', module: 'SYSTEM', message: 'System startup complete' },
  { id: 2, timestamp: '2023-10-27 10:05:23', level: 'warn', module: 'NETWORK', message: 'High latency detected on eth0' },
  { id: 3, timestamp: '2023-10-27 10:15:00', level: 'error', module: 'MQTT', message: 'Connection lost to broker' },
  { id: 4, timestamp: '2023-10-27 10:15:05', level: 'info', module: 'MQTT', message: 'Reconnected to broker' },
];

export const mockTunnels: NPSTunnel[] = [
  { id: 101, type: 'tcp', local: '127.0.0.1:8080', remote: 28080, status: 'online' },
  { id: 102, type: 'udp', local: '127.0.0.1:53', remote: 20053, status: 'online' },
  { id: 103, type: 'p2p', local: '192.168.1.105:80', remote: 0, status: 'offline' },
];

export const mockMqttMessages: MQTTMessage[] = [
  { id: '1', topic: 'sensors/temp', payload: '{"val": 24.5}', qos: 1, direction: 'in', timestamp: '10:00:01' },
  { id: '2', topic: 'controls/light', payload: 'ON', qos: 1, direction: 'out', timestamp: '10:00:05' },
];
