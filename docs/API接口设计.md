# API接口设计文档

## 1. API概述

### 1.1 基础信息
- **Base URL**: `http://device-ip:8080/api/v1`
- **协议**: HTTP/HTTPS
- **数据格式**: JSON
- **字符编码**: UTF-8

### 1.2 认证方式
- **Web界面**: Session/Cookie认证
- **API调用**: JWT Token认证
- **Token获取**: 通过 `/api/v1/auth/login` 接口

### 1.3 响应格式
```json
{
  "code": 200,
  "message": "success",
  "data": {},
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 1.4 错误码
- `200`: 成功
- `400`: 请求参数错误
- `401`: 未授权
- `403`: 禁止访问
- `404`: 资源不存在
- `500`: 服务器内部错误

## 2. 认证相关接口

### 2.1 用户登录
```
POST /api/v1/auth/login
```

**请求体**:
```json
{
  "username": "admin",
  "password": "password"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "登录成功",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600
  }
}
```

### 2.2 用户登出
```
POST /api/v1/auth/logout
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "message": "登出成功"
}
```

### 2.3 修改密码
```
POST /api/v1/auth/change-password
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "old_password": "old_password",
  "new_password": "new_password",
  "confirm_password": "new_password"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "密码修改成功"
}
```

## 3. 系统管理接口

### 3.1 获取系统信息
```
GET /api/v1/system/info
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "device_id": "device_001",
    "firmware_version": "1.0.0",
    "uptime": 86400,
    "start_time": "2024-01-01T00:00:00Z",
    "cpu_usage": 25.5,
    "memory_usage": 45.2,
    "disk_usage": 30.1,
    "network": {
      "interface": "eth0",
      "ip": "192.168.1.100",
      "status": "connected"
    }
  }
}
```

### 3.2 设备重启
```
POST /api/v1/system/restart
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "type": "soft"  // soft: 软重启, hard: 硬重启
}
```

**响应**:
```json
{
  "code": 200,
  "message": "重启命令已发送"
}
```

### 3.3 获取系统日志
```
GET /api/v1/system/logs
```

**请求头**:
```
Authorization: Bearer {token}
```

**查询参数**:
- `level`: 日志级别 (debug/info/warn/error)
- `module`: 模块名称
- `start_time`: 开始时间 (ISO 8601)
- `end_time`: 结束时间 (ISO 8601)
- `page`: 页码 (默认1)
- `page_size`: 每页数量 (默认50)

**响应**:
```json
{
  "code": 200,
  "data": {
    "logs": [
      {
        "timestamp": "2024-01-01T12:00:00Z",
        "level": "info",
        "module": "network",
        "message": "Network connected",
        "data": {}
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 50
  }
}
```

## 4. 网络管理接口

### 4.1 获取网络接口列表
```
GET /api/v1/network/interfaces
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "interfaces": [
      {
        "name": "eth0",
        "type": "ethernet",
        "status": "up",
        "ip": "192.168.1.100",
        "netmask": "255.255.255.0",
        "gateway": "192.168.1.1",
        "mac": "00:11:22:33:44:55"
      },
      {
        "name": "wlan0",
        "type": "wifi",
        "status": "down",
        "ip": "",
        "netmask": "",
        "gateway": "",
        "mac": "00:11:22:33:44:56"
      }
    ]
  }
}
```

### 4.2 连接WiFi
```
POST /api/v1/network/wifi/connect
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "ssid": "WiFi_SSID",
  "password": "WiFi_password",
  "security": "WPA2"  // WPA2, WPA, WEP, Open
}
```

**响应**:
```json
{
  "code": 200,
  "message": "WiFi连接成功",
  "data": {
    "ip": "192.168.1.100",
    "signal_strength": -50
  }
}
```

### 4.3 扫描WiFi热点
```
GET /api/v1/network/wifi/scan
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "networks": [
      {
        "ssid": "WiFi_SSID",
        "signal_strength": -50,
        "security": "WPA2",
        "frequency": 2450,
        "channel": 6
      }
    ]
  }
}
```

### 4.4 获取网络状态
```
GET /api/v1/network/status
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "current_interface": "eth0",
    "ip": "192.168.1.100",
    "status": "connected",
    "upload_speed": 10.5,
    "download_speed": 50.2,
    "latency": 20
  }
}
```

## 5. 设备扫描接口

### 5.1 获取设备列表
```
GET /api/v1/devices
```

**请求头**:
```
Authorization: Bearer {token}
```

**查询参数**:
- `status`: 设备状态 (online/offline/all)
- `type`: 设备类型
- `keyword`: 搜索关键词 (IP/名称/MAC)
- `page`: 页码
- `page_size`: 每页数量

**响应**:
```json
{
  "code": 200,
  "data": {
    "devices": [
      {
        "ip": "192.168.1.101",
        "mac": "00:11:22:33:44:55",
        "name": "PC-001",
        "vendor": "Intel",
        "type": "computer",
        "os": "Windows 10",
        "status": "online",
        "open_ports": [80, 443, 3389],
        "last_seen": "2024-01-01T12:00:00Z",
        "first_seen": "2024-01-01T00:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

### 5.2 获取设备详情
```
GET /api/v1/devices/{ip}
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "ip": "192.168.1.101",
    "mac": "00:11:22:33:44:55",
    "name": "PC-001",
    "vendor": "Intel",
    "type": "computer",
    "os": "Windows 10",
    "status": "online",
    "open_ports": [
      {
        "port": 80,
        "protocol": "tcp",
        "service": "http",
        "version": "Apache/2.4"
      }
    ],
    "last_seen": "2024-01-01T12:00:00Z",
    "first_seen": "2024-01-01T00:00:00Z",
    "history": [
      {
        "timestamp": "2024-01-01T12:00:00Z",
        "status": "online"
      }
    ]
  }
}
```

### 5.3 启动设备扫描
```
POST /api/v1/devices/scan/start
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "subnet": "192.168.1.0/24",  // 可选，不指定则自动检测
  "timeout": 30  // 扫描超时时间（秒）
}
```

**响应**:
```json
{
  "code": 200,
  "message": "扫描已启动",
  "data": {
    "scan_id": "scan_001",
    "status": "running"
  }
}
```

### 5.4 停止设备扫描
```
POST /api/v1/devices/scan/stop
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "message": "扫描已停止"
}
```

### 5.5 获取扫描状态
```
GET /api/v1/devices/scan/status
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "status": "running",  // running, stopped, completed
    "progress": 50,  // 扫描进度百分比
    "scanned_count": 25,
    "found_count": 10,
    "start_time": "2024-01-01T12:00:00Z"
  }
}
```

## 6. 网络工具箱接口

### 6.1 Ping测试
```
POST /api/v1/tools/ping
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "target": "192.168.1.1",
  "count": 4,
  "timeout": 5
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "target": "192.168.1.1",
    "packets_sent": 4,
    "packets_received": 4,
    "packet_loss": 0,
    "min_latency": 1.2,
    "max_latency": 2.5,
    "avg_latency": 1.8,
    "results": [
      {
        "sequence": 1,
        "latency": 1.2,
        "status": "success"
      }
    ]
  }
}
```

### 6.2 Traceroute
```
POST /api/v1/tools/traceroute
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "target": "8.8.8.8",
  "max_hops": 30,
  "timeout": 5
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "target": "8.8.8.8",
    "hops": [
      {
        "hop": 1,
        "ip": "192.168.1.1",
        "hostname": "router.local",
        "latency": 1.2
      },
      {
        "hop": 2,
        "ip": "10.0.0.1",
        "hostname": "",
        "latency": 5.5
      }
    ]
  }
}
```

### 6.3 网速测试
```
POST /api/v1/tools/speedtest
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "server": "default",  // default 或指定服务器地址
  "test_type": "all"  // upload, download, all
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "server": "speedtest.example.com",
    "upload_speed": 10.5,  // Mbps
    "download_speed": 50.2,  // Mbps
    "latency": 20,  // ms
    "test_time": "2024-01-01T12:00:00Z",
    "duration": 30  // 测试耗时（秒）
  }
}
```

### 6.4 端口扫描
```
POST /api/v1/tools/portscan
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "target": "192.168.1.101",
  "ports": [80, 443, 8080],  // 或 "1-1000" 端口范围
  "timeout": 5,
  "scan_type": "tcp"  // tcp, udp, both
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "target": "192.168.1.101",
    "scanned_ports": 3,
    "open_ports": [
      {
        "port": 80,
        "protocol": "tcp",
        "service": "http",
        "version": "Apache/2.4",
        "status": "open"
      }
    ],
    "closed_ports": [443, 8080],
    "scan_time": "2024-01-01T12:00:00Z"
  }
}
```

### 6.5 DNS查询
```
POST /api/v1/tools/dns
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "query": "example.com",
  "type": "A",  // A, AAAA, PTR, MX, CNAME, TXT
  "server": "8.8.8.8"  // 可选，默认使用系统DNS
}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "query": "example.com",
    "type": "A",
    "records": [
      {
        "name": "example.com",
        "type": "A",
        "value": "93.184.216.34",
        "ttl": 3600
      }
    ]
  }
}
```

## 7. NPS管理接口

### 7.1 获取NPS状态
```
GET /api/v1/nps/status
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "connected": true,
    "server": "nps.example.com:8024",
    "client_id": "client_001",
    "connected_at": "2024-01-01T00:00:00Z",
    "tunnels": [
      {
        "id": "tunnel_001",
        "type": "tcp",
        "local_port": 8080,
        "remote_port": 18080,
        "status": "active"
      }
    ]
  }
}
```

### 7.2 连接NPS服务端
```
POST /api/v1/nps/connect
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "server": "nps.example.com:8024",
  "vkey": "verification_key",
  "client_id": "client_001"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "NPS连接成功"
}
```

### 7.3 断开NPS连接
```
POST /api/v1/nps/disconnect
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "message": "NPS连接已断开"
}
```

### 7.4 获取隧道列表
```
GET /api/v1/nps/tunnels
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "tunnels": [
      {
        "id": "tunnel_001",
        "type": "tcp",
        "local_address": "127.0.0.1:8080",
        "remote_port": 18080,
        "status": "active",
        "created_at": "2024-01-01T00:00:00Z"
      }
    ]
  }
}
```

## 8. MQTT管理接口

### 8.1 获取MQTT状态
```
GET /api/v1/mqtt/status
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "connected": true,
    "server": "mqtt.example.com:1883",
    "client_id": "device_001",
    "connected_at": "2024-01-01T00:00:00Z",
    "subscribed_topics": [
      "nwct/device_001/command"
    ],
    "published_topics": [
      "nwct/device_001/status",
      "nwct/device_001/heartbeat"
    ]
  }
}
```

### 8.2 连接MQTT服务器
```
POST /api/v1/mqtt/connect
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "server": "mqtt.example.com",
  "port": 1883,
  "username": "user",
  "password": "password",
  "client_id": "device_001",
  "tls": false
}
```

**响应**:
```json
{
  "code": 200,
  "message": "MQTT连接成功"
}
```

### 8.3 断开MQTT连接
```
POST /api/v1/mqtt/disconnect
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "message": "MQTT连接已断开"
}
```

### 8.4 获取MQTT日志
```
GET /api/v1/mqtt/logs
```

**请求头**:
```
Authorization: Bearer {token}
```

**查询参数**:
- `topic`: 主题过滤
- `direction`: 方向 (publish/subscribe/all)
- `start_time`: 开始时间
- `end_time`: 结束时间
- `page`: 页码
- `page_size`: 每页数量

**响应**:
```json
{
  "code": 200,
  "data": {
    "logs": [
      {
        "timestamp": "2024-01-01T12:00:00Z",
        "direction": "publish",
        "topic": "nwct/device_001/status",
        "qos": 1,
        "payload": {
          "status": "online"
        },
        "status": "success"
      }
    ],
    "total": 100,
    "page": 1,
    "page_size": 50
  }
}
```

## 9. 配置管理接口

### 9.1 获取当前配置
```
GET /api/v1/config
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "code": 200,
  "data": {
    "device": {
      "device_id": "device_001",
      "name": "内网穿透盒子"
    },
    "network": {
      "interface": "eth0",
      "ip_mode": "dhcp"
    },
    "nps_server": {
      "server": "nps.example.com:8024",
      "vkey": "***",
      "client_id": "client_001"
    },
    "mqtt": {
      "server": "mqtt.example.com",
      "port": 1883,
      "username": "user",
      "client_id": "device_001"
    },
    "scanner": {
      "auto_scan": true,
      "scan_interval": 300,
      "timeout": 30
    }
  }
}
```

### 9.2 更新配置
```
POST /api/v1/config
```

**请求头**:
```
Authorization: Bearer {token}
```

**请求体**:
```json
{
  "network": {
    "interface": "wlan0",
    "ip_mode": "static",
    "ip": "192.168.1.100",
    "netmask": "255.255.255.0",
    "gateway": "192.168.1.1"
  }
}
```

**响应**:
```json
{
  "code": 200,
  "message": "配置更新成功"
}
```

### 9.3 设备初始化
```
POST /api/v1/config/init
```

**请求体**:
```json
{
  "network": {
    "type": "wifi",
    "ssid": "WiFi_SSID",
    "password": "password"
  },
  "nps_server": {
    "server": "nps.example.com:8024",
    "vkey": "vkey",
    "client_id": "client_001"
  },
  "mqtt": {
    "server": "mqtt.example.com",
    "port": 1883,
    "username": "user",
    "password": "password",
    "client_id": "device_001"
  },
  "admin_password": "admin_password"
}
```

**响应**:
```json
{
  "code": 200,
  "message": "初始化完成"
}
```

### 9.4 导出配置
```
GET /api/v1/config/export
```

**请求头**:
```
Authorization: Bearer {token}
```

**响应**: 配置文件下载

### 9.5 导入配置
```
POST /api/v1/config/import
```

**请求头**:
```
Authorization: Bearer {token}
Content-Type: multipart/form-data
```

**请求体**: 配置文件文件

**响应**:
```json
{
  "code": 200,
  "message": "配置导入成功"
}
```

## 10. WebSocket接口

### 10.1 连接
```
WS /ws
```

**认证**: 通过Query参数传递Token
```
ws://device-ip:8080/ws?token={jwt_token}
```

### 10.2 消息类型

#### 10.2.1 实时日志
```json
{
  "type": "log",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z",
    "level": "info",
    "module": "network",
    "message": "Network connected"
  }
}
```

#### 10.2.2 设备状态更新
```json
{
  "type": "device_status",
  "data": {
    "ip": "192.168.1.101",
    "status": "online",
    "last_seen": "2024-01-01T12:00:00Z"
  }
}
```

#### 10.2.3 扫描进度
```json
{
  "type": "scan_progress",
  "data": {
    "progress": 50,
    "scanned_count": 25,
    "found_count": 10
  }
}
```

#### 10.2.4 测试结果
```json
{
  "type": "test_result",
  "data": {
    "test_id": "test_001",
    "type": "ping",
    "result": {
      "target": "192.168.1.1",
      "avg_latency": 1.8
    }
  }
}
```

#### 10.2.5 MQTT消息
```json
{
  "type": "mqtt_message",
  "data": {
    "timestamp": "2024-01-01T12:00:00Z",
    "direction": "publish",
    "topic": "nwct/device_001/status",
    "payload": {
      "status": "online"
    }
  }
}
```

