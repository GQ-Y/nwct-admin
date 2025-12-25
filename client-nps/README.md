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

## 默认端口与访问入口

- 默认监听：`:80`
- Web 管理面板：`http://device-ip/`
- API Base：`http://device-ip/api/v1`
- WebSocket：`ws://device-ip/ws?token=...`

## 编译

```bash
go build -o nwct-client .
```

> 注意：Web 管理面板会被 embed 到后端二进制中。若你有更新前端，请先在 `client-web/` 执行 `pnpm build`，再编译本项目。

## 运行

```bash
./nwct-client
```

程序会在 `:80` 端口启动 HTTP API 服务，并同端口托管 Web 管理面板。

> 80 端口通常需要 root/capabilities，请在开发板侧使用 root 启动或配置相应权限。

## 后台启动/停止（推荐脚本）

仓库已提供控制脚本：`client-nps/nwctctl.sh`，支持后台启动、停止、重启、查看状态与日志。

### 快速使用

在 `client-nps/` 目录：

```bash
chmod +x ./nwctctl.sh
sudo ./nwctctl.sh start
./nwctctl.sh status
./nwctctl.sh logs
sudo ./nwctctl.sh restart
sudo ./nwctctl.sh stop
```

### 快捷查找/杀进程（不使用脚本时）

```bash
# 查找
pgrep -af nwct-client

# 结束（优雅）
pkill -TERM -f nwct-client

# 强杀（必要时）
pkill -KILL -f nwct-client
```

> 更推荐用 `nwctctl.sh`，它会使用 pidfile 避免误杀其它同名进程。

## 配置

配置文件默认路径：`/etc/nwct/config.json`

首次运行会自动创建默认配置文件。

## API文档

API基础路径：`http://localhost/api/v1`

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

