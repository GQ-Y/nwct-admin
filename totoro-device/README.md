# Totoro Device（设备端）

Totoro 内网穿透体系的设备端实现，运行在 Luckfox Pico 等开发板上。

## 功能特性

- 网络管理（有线/WiFi）
- 设备扫描和识别
- 网络工具箱（Ping、Traceroute、网速测试等）
- FRP 客户端集成（支持 builtin/public/manual 三种模式）
- HTTP API 服务
- WebSocket 实时推送
- Web 控制面板支持
- 可选屏幕 UI（Ultra 版本）

## 默认端口与访问入口

- 默认监听：`:18080`（Luckfox 固件上建议用 18080 避免占用系统 80）
- Web 管理面板：`http://设备IP:18080/`
- API Base：`http://设备IP:18080/api/v1`
- WebSocket：`ws://设备IP:18080/ws?token=...`

## 开发板部署（推荐）

### 一键部署脚本

在 `totoro-device/` 目录下使用 `deploy_luckfox.sh` 脚本进行一键部署。

#### 部署方式

脚本支持两种模式：

1. **非交互式模式**（推荐用于自动化/CI/CD）：
   - 通过环境变量提供所有参数
   - 适合脚本调用和批量部署

2. **交互式模式**：
   - 脚本会提示输入缺失的参数
   - 适合手动部署和测试

#### 环境变量说明

| 变量名 | 说明 | 默认值 | 是否必需 |
|--------|------|--------|----------|
| `TARGET_HOST` | 开发板 IP 地址 | `192.168.2.221` | 是 |
| `TARGET_USER` | SSH 登录用户名 | `root` | 是 |
| `TARGET_PASS` | SSH 登录密码 | - | 是（或使用 SSH key） |
| `TARGET_PORT` | SSH 端口 | `22` | 否 |
| `DEVICE_MODEL` | 设备型号 | `ultra` | 是（ultra/plus/pro） |
| `BRIDGE_API_URL` | 桥梁 API 地址 | - | 否（留空使用默认值） |
| `DEVICE_ID` | 设备号 | - | 否（留空自动生成随机设备号） |
| `NWCT_HTTP_PORT` | 设备端 HTTP 端口 | `18080` | 否 |
| `TARGET_PATH` | 上传路径 | `/root/totoro-device` | 否（脚本会智能选择） |
| `INTERACTIVE` | 是否交互式 | `0` | 否 |

#### 完整部署命令示例

**Ultra 开发板（带屏版本）：**

```bash
cd totoro-device
BRIDGE_API_URL="http://192.168.2.32:18090" \
TARGET_HOST=192.168.2.127 \
TARGET_USER=root \
TARGET_PASS=luckfox \
DEVICE_MODEL=ultra \
INTERACTIVE=0 \
./deploy_luckfox.sh
```

**Pro 开发板（无屏版本）：**

```bash
cd totoro-device
BRIDGE_API_URL="http://192.168.2.32:18090" \
TARGET_HOST=192.168.2.182 \
TARGET_USER=root \
TARGET_PASS=luckfox \
DEVICE_MODEL=pro \
INTERACTIVE=0 \
./deploy_luckfox.sh
```

**Plus 开发板（无屏版本，需使用 SD 卡运行系统）：**

Plus 开发板由于内存和存储限制，需要将系统运行在 SD 卡上。

**步骤一：烧录 MicroSD 镜像到 SD 卡（推荐：通过 SSH 在开发板上烧录）**

```bash
# 1. 确保 SD 卡已插入 Plus 开发板，开发板可以 SSH 连接
cd totoro-device/scripts
TARGET_HOST=192.168.2.226 \
TARGET_USER=root \
TARGET_PASS=luckfox \
FORCE=1 \
./flash_plus_sdcard_remote.sh
```

脚本会自动：
- 检测开发板上的 SD 卡设备
- 解压镜像文件
- 按分区烧录（解析 `sd_update.txt` 获取分区偏移）
- 流式传输各个分区镜像到开发板并烧录

**步骤二：重启开发板，确认从 SD 卡启动**

重启后，检查是否从 SD 卡启动：
```bash
ssh root@<开发板IP> 'cat /proc/cmdline | grep storagemedia'
```

如果显示 `storagemedia=sd` 或 `root=/dev/mmcblk1p7`，说明已从 SD 卡启动。

**步骤三：部署 totoro-device**

```bash
cd totoro-device
BRIDGE_API_URL="http://192.168.2.32:18090" \
TARGET_HOST=<开发板IP> \
TARGET_USER=root \
TARGET_PASS=luckfox \
DEVICE_MODEL=plus \
INTERACTIVE=0 \
./deploy_luckfox.sh
```

> **注意**：
> - Plus 版本会自动使用 `device_minimal` build tag，移除了 Ping、Traceroute、SpeedTest 等非必要工具，但保留了设备扫描和端口扫描等核心功能
> - 如果开发板重启后仍从内置存储启动，请尝试按住 BOOT 键的同时重启开发板
> - 从 SD 卡启动后，根文件系统会有更大的存储空间（约 5.8GB），可以正常运行 totoro-device

> **注意**：Plus 版本会自动使用 `device_minimal` build tag，移除了 Ping、Traceroute、SpeedTest 等非必要工具，但保留了设备扫描和端口扫描等核心功能。

**交互式部署：**

```bash
cd totoro-device
INTERACTIVE=1 ./deploy_luckfox.sh
```

脚本会依次提示输入：
- 设备 IP（TARGET_HOST）
- 登录账号（TARGET_USER）
- SSH 端口（TARGET_PORT）
- SSH 密码（TARGET_PASS）
- 设备型号（DEVICE_MODEL：ultra/plus/pro）
- 设备端测试端口（NWCT_HTTP_PORT）
- 桥梁 API 地址（BRIDGE_API_URL，留空则使用默认值）
- 设备号（DEVICE_ID，留空则自动生成随机设备号）
- 上传路径（TARGET_PATH）

#### 脚本功能

部署脚本会自动完成以下操作：

1. **检测开发板架构**：自动识别 ARMv7/ARM64
2. **构建前端**：自动构建 `totoro-device-web` 并嵌入到二进制
3. **交叉编译后端**：
   - 根据 `DEVICE_MODEL` 自动选择编译标签（ultra 带 `device_display`）
   - 自动生成随机设备号（如果未指定）
   - 注入设备名称、设备号、设备型号、桥梁 API 地址到二进制
4. **智能路径选择**：根据可用空间自动选择最佳上传路径
5. **上传并设置权限**：上传二进制文件并设置可执行权限
6. **安装开机自启**：自动安装 `/etc/init.d/S99totoro-device` 脚本
7. **启动服务**：后台启动服务并验证运行状态

#### 设备信息注入

编译时会自动注入以下信息到二进制文件：

- **设备号**：每次编译自动生成随机设备号（格式：`DEV` + 6位随机数字），或通过 `DEVICE_ID` 手动指定
- **设备型号**：根据 `DEVICE_MODEL` 自动设置（ultra/pro/plus）
- **设备名称**：根据设备型号自动设置（Totoro S1 Ultra / Totoro S1 Pro / Totoro S1 Plus）
- **桥梁 API 地址**：通过 `BRIDGE_API_URL` 环境变量注入（用户不可修改，优先级最高）

这些信息会在应用启动时写入数据库，WebUI 会从数据库读取并显示。

## 手动编译

如果需要手动编译（不推荐，建议使用部署脚本）：

### 带屏版本（Ultra）

```bash
cd totoro-device
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
go build -tags device_display \
  -ldflags "-s -w \
    -X 'totoro-device/config.DefaultDeviceName=Totoro S1 Ultra' \
    -X 'totoro-device/config.DefaultDeviceID=DEV123456' \
    -X 'totoro-device/config.DefaultDeviceModel=ultra' \
    -X 'totoro-device/config.EmbeddedBridgeURL=http://192.168.2.32:18090'" \
  -o bin/totoro-device_linux_armv7_ultra_display .
```

### 无屏版本（Plus/Pro）

```bash
cd totoro-device
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 \
go build \
  -ldflags "-s -w \
    -X 'totoro-device/config.DefaultDeviceName=Totoro S1 Pro' \
    -X 'totoro-device/config.DefaultDeviceID=DEV123456' \
    -X 'totoro-device/config.DefaultDeviceModel=pro' \
    -X 'totoro-device/config.EmbeddedBridgeURL=http://192.168.2.32:18090'" \
  -o bin/totoro-device_linux_armv7_pro .
```

> 注意：Web 管理面板会被 embed 到后端二进制中。若你有更新前端，请先在 `totoro-device-web/` 执行 `pnpm build`，再编译本项目。

## 运行

### 带屏版本（Ultra）

```bash
./totoro-device -display=true
```

### 无屏版本（Plus/Pro）

```bash
./totoro-device
```

程序会在 `:18080` 端口启动 HTTP API 服务，并同端口托管 Web 管理面板。

> 18080 端口通常需要 root 权限，请在开发板侧使用 root 启动。

## 服务管理

### 开机自启动

部署脚本会自动安装开机自启动脚本 `/etc/init.d/S99totoro-device`，设备重启后会自动启动服务。

### 手动管理服务

```bash
# 启动
/etc/init.d/S99totoro-device start

# 停止
/etc/init.d/S99totoro-device stop

# 重启
/etc/init.d/S99totoro-device restart

# 查看状态
/etc/init.d/S99totoro-device status
```

### 查看日志

```bash
# 系统日志
tail -f /var/log/totoro-device.log

# 应用日志
tail -f /var/log/nwct/system.log
```

### 快捷查找/杀进程

```bash
# 查找
pgrep -af totoro-device

# 结束（优雅）
pkill -TERM -f totoro-device

# 强杀（必要时）
pkill -KILL -f totoro-device
```

## 配置

### 配置文件路径

- **Linux/开发板**：默认 `/etc/nwct/config.json`
- **macOS 本地开发**：默认写到仓库根目录 `config.json`

可通过环境变量覆盖：
- **`NWCT_CONFIG_PATH`**：指定配置文件路径

### 数据库路径

- 配置项：`config.json` → `database.path`
- 环境变量覆盖：**`NWCT_DB_PATH`**（默认 `/var/nwct/devices.db`）

### 日志目录

- **`NWCT_LOG_DIR`**：日志目录（默认 `/var/log/nwct`，不可用时会回退到临时目录）

首次运行会自动创建默认配置文件。

## API文档

API基础路径：`http://设备IP:18080/api/v1`

主要 API 端点：
- `GET /api/v1/system/info` - 获取系统信息（包含设备号、设备型号）
- `GET /api/v1/cloud/status` - 获取云服务连接状态
- `GET /api/v1/config` - 获取配置（包含设备信息）
- `POST /api/v1/frp/connect` - 连接 FRP
- `GET /api/v1/frp/status` - 获取 FRP 连接状态
- 更多 API 请参考代码中的 `internal/api/server.go`

## 开发

### 项目结构

```
totoro-device/
├── main.go                 # 程序入口（根据 build tag 选择 main_display.go 或 main_headless.go）
├── main_display.go        # 带屏版本入口
├── main_headless.go        # 无屏版本入口
├── config/                 # 配置管理
├── internal/               # 内部模块
│   ├── api/               # HTTP API服务
│   ├── database/          # 数据库模块（包含设备信息表）
│   ├── logger/            # 日志模块
│   ├── network/           # 网络管理
│   ├── frp/               # FRP 客户端
│   ├── scanner/           # 设备扫描
│   ├── display/           # 屏幕 UI（仅带屏版本）
│   └── toolkit/           # 网络工具箱
├── models/                # 数据模型
├── utils/                 # 工具函数
└── deploy_luckfox.sh      # 一键部署脚本
```

### Build Tags

- `device_display`：编译带屏版本（Ultra），包含屏幕 UI 功能
- 无 tag：编译无屏版本（Plus/Pro），不包含屏幕 UI

## 依赖

主要依赖库：
- Gin - Web框架
- modernc.org/sqlite - 纯 Go SQLite 驱动（避免 CGO）
- gopsutil - 系统信息
- gorilla/websocket - WebSocket支持

## 许可证

MIT

