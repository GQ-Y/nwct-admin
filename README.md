## NWCT（Totoro）项目说明

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

在 `totoro-device/`：

```bash
# 非交互式（推荐用于自动化/复制粘贴）：通过环境变量提供连接信息
TARGET_HOST=192.168.2.174 TARGET_USER=root TARGET_PASS=luckfox DEVICE_MODEL=ultra bash ./deploy_luckfox.sh
```

或使用 **交互式输入**（脚本会提示输入 IP/账号/密码等；如果你已提前通过环境变量传入，则不会重复提问）：

```bash
INTERACTIVE=1 bash ./deploy_luckfox.sh
```

- **Ultra**：`DEVICE_MODEL=ultra`（默认会带 `device_display` tag，并默认 `-display=true`）
- **Plus/Pro**：`DEVICE_MODEL=plus` 或 `DEVICE_MODEL=pro`（默认无 tag、无 `-display` 参数）

脚本的两种形式通过 `INTERACTIVE` 控制：
- `INTERACTIVE=0`（默认）：非交互式，仅使用环境变量/脚本默认值
- `INTERACTIVE=1`：交互式输入（stdin 不是 TTY 时会自动降级为非交互，避免卡住）

### 手动交叉编译（示例）

- **带屏版本（Ultra）**：

```bash
cd totoro-device
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -tags device_display -o bin/totoro-device_linux_armv7_display .
```

- **无屏版本（Plus/Pro）**：

```bash
cd totoro-device
GOOS=linux GOARCH=arm GOARM=7 CGO_ENABLED=0 go build -o bin/totoro-device_linux_armv7 .
```

### 运行（设备上）
- 带屏版本：建议 `-display=true`（无屏版本不需要/也不会启用 UI）

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


