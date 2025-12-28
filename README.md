## Totoro 项目说明

本仓库是 Totoro 内网穿透体系的单仓库实现，包含三类核心组件：
- **设备端**：`totoro-device/`（Go，运行在 Luckfox 等设备上：API + WebSocket +（可选）屏幕 UI + FRP 客户端）
- **节点端**：`totoro-node/`（Go，运行在 Linux 服务器/节点上：FRPS + 节点 API + WebUI）
- **桥梁端**：`totoro-bridge/`（Go，提供设备注册/公开节点/邀请码兑换/票据签发等桥梁 API，并内嵌管理 WebUI）

另外还有两个 Web 管理面板工程：
- `totoro-device-web/`：设备端 WebUI（由 `totoro-device` embed 或独立部署）
- `totoro-bridge-web/`：桥梁端 WebUI（由 `totoro-bridge` embed 或独立部署）

桌面端仅用于“本机一键拉起 node”的便捷控制：
- `totoro-node-desktop/`：Flutter 桌面端（保持现有 GitHub Actions 构建；Linux 节点部署不依赖 desktop）

---

## 架构与访问入口（默认）

- **totoro-device（设备端）**：
  - 面板：`http://设备IP:18080/`（Luckfox 固件上建议用 18080 避免占用系统 80）
  - API：`http://设备IP:18080/api/v1/*`
  - WebSocket：`ws://设备IP:18080/ws`
- **totoro-node（节点端）**：
  - 节点 WebUI：`http://节点IP:18080/`
  - 节点 API：`http://节点IP:18080/api/v1/*`
- **totoro-bridge（桥梁端）**：
  - 管理 WebUI：`http://桥梁IP:18090/`
  - 桥梁 API：`http://桥梁IP:18090/api/v1/*`

后端路由约定：
- `/api/v1/*`：REST API（未命中返回 JSON 404，不会被前端 SPA “吞掉”）
- `/ws`：实时推送（JWT 鉴权）
- 其它路径：静态资源优先，否则 SPA 回退到首页（支持前端路由刷新不 404）

---

## 设备端（totoro-device）构建与部署

### 两种编译产物（按硬件区分）
- **Luckfox Pico Ultra（带屏）**：编译 **带屏版本**（build tag：`device_display`）
- **Luckfox Pico Plus/Pro（无屏，仅网口）**：编译 **无屏版本**（默认，不带 tag）

### 一键部署到 Luckfox（脚本）

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

### 手动交叉编译（示例）

如果需要手动编译（不推荐，建议使用部署脚本）：

- **带屏版本（Ultra）**：

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

- **无屏版本（Plus/Pro）**：

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

### 运行（设备上）

- **带屏版本（Ultra）**：建议 `-display=true`
- **无屏版本（Plus/Pro）**：不需要/也不会启用 UI

## 节点端（totoro-node）构建与部署（Linux）

Linux 节点不需要 `totoro-node-desktop`，只需要运行 `totoro-node` 并通过其 WebUI 管理。

### 编译

```bash
cd totoro-node
./build.sh linux amd64
```

产物在 `totoro-node/bin/`。

### 运行（最小）
首次运行会在可执行文件同目录生成 `.env`，按需修改即可：

```bash
./totoro-node
```

### 桥梁地址配置（节点侧）
编辑同目录 `.env`：
- `TOTOTO_BRIDGE_URL=http://192.168.2.32:18090`

## 桥梁端（totoro-bridge）配置（仅 .env）

桥梁端会在首次运行自动生成 `.env`（同目录），只需编辑 `.env` 即可完成配置：
- `TOTOTO_BRIDGE_ADDR`（默认 `:18090`）
- `TOTOTO_BRIDGE_DB`（默认 `./bridge.db`）
- `TOTOTO_BRIDGE_ADMIN_KEY`（生产必填）
- `TOTOTO_TICKET_TTL_DAYS`（默认 `30`）

## 配置文件与关键参数（设备端）

### 配置文件路径

`totoro-device` 启动时会加载配置（若不存在会写入默认配置）：
- **Linux/开发板**：默认 `/etc/nwct/config.json`
- **macOS 本地开发**：默认写到仓库根目录 `config.json`

可通过环境变量覆盖：
- **`NWCT_CONFIG_PATH`**：指定配置文件路径

### 端口
- 设备端（Luckfox 固件）：推荐 `18080`（避免占用系统服务）
- 配置项：`/etc/nwct/config.json` → `server.port`

### 日志目录

- **`NWCT_LOG_DIR`**：日志目录（默认 `/var/log/nwct`，不可用时会回退到临时目录）

### 数据库路径

- 配置项：`config.json` → `database.path`
- 环境变量覆盖：**`NWCT_DB_PATH`**

### NPS（npc）与“一键获取 vkey”

NPS 相关配置：
- `config.json` → `nps_server.server`（形如 `host:port`）
- `config.json` → `nps_server.vkey`
- `config.json` → `nps_server.client_id`
- `config.json` → `nps_server.npc_path` / `npc_config_path` / `npc_args`

一键获取 vkey（后端会在 vkey 为空且未提供 `npc_config_path` 时尝试）：
- **`NWCT_NPS_WEB_URL`**：NPS Web 地址（例如 `http://127.0.0.1:19080`）
- **`NWCT_NPS_WEB_USER`**：NPS Web 用户名
- **`NWCT_NPS_WEB_PASS`**：NPS Web 密码

npc 安装目录：
- **`NWCT_NPC_DIR`**：指定 npc 下载/安装目录（不指定则使用缓存目录）

### MQTT

MQTT 配置：
- `config.json` → `mqtt.server` / `mqtt.port` / `mqtt.username` / `mqtt.password` / `mqtt.client_id`
- `config.json` → `mqtt.auto_connect`：是否允许启动时自动连接（设备侧常用）

### OUI（厂商库）相关（可选）

用于 MAC 厂商识别：
- **`NWCT_OUI_PATH`**：本地 oui 文件路径
- **`NWCT_OUI_AUTO_UPDATE`**：缺失时是否自动更新（`1/true` 启用）
- **`NWCT_OUI_URL`**：下载源地址
- **`NWCT_OUI_CACHE_DIR`**：缓存目录

---

## 常见操作（快速自检）
- 设备端（本机/设备上）：`curl http://127.0.0.1:18080/api/v1/network/status`
- 节点端：浏览器打开 `http://127.0.0.1:18080/`
- 桥梁端：浏览器打开 `http://127.0.0.1:18090/`


