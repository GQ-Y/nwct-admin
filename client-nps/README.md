# 内网穿透盒子客户端

内网穿透盒子设备的Go语言客户端实现。

## 功能特性

- 网络管理（有线/WiFi）
- 设备扫描和识别
- 网络工具箱（Ping、Traceroute、网速测试等）
- NPS客户端集成
- HTTP API服务
- WebSocket实时推送
- Web控制面板支持

## 编译

```bash
go build -o nwct-client .
```

## 运行

```bash
./nwct-client
```

程序会在 `:8080` 端口启动HTTP API服务。

## 配置

配置文件默认路径：`/etc/nwct/config.json`

首次运行会自动创建默认配置文件。

## API文档

API基础路径：`http://localhost:8080/api/v1`

详细API文档请参考 `docs/API接口设计.md`

## 开发

### 项目结构

```
client-nps/
├── main.go                 # 程序入口
├── config/                 # 配置管理
├── internal/               # 内部模块
│   ├── api/               # HTTP API服务
│   ├── database/          # 数据库模块
│   ├── logger/            # 日志模块
│   ├── network/           # 网络管理
│   ├── nps/               # NPS客户端
│   ├── scanner/           # 设备扫描
│   └── toolkit/           # 网络工具箱
├── models/                # 数据模型
└── utils/                 # 工具函数
```

## 依赖

主要依赖库：
- Gin - Web框架
- SQLite - 数据库
- gopsutil - 系统信息
- gorilla/websocket - WebSocket支持

## 许可证

MIT

