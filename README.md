## NWCT（内网穿透盒子）项目说明

本仓库包含两部分：
- **后端/设备端**：`client-nps/`（Go，提供 API + WebSocket + 静态 Web 管理面板托管）
- **前端 Web**：`client-web/`（React + Vite，构建产物会被 embed 进后端二进制）

当前已实现 **单端口（默认 80）一体化部署**：只需要在开发板固件中放入一个 `client-nps` 可执行文件，即可通过 `http://设备IP/` 进入管理面板并完成初始化，同时同域访问 `/api/v1/*` 与 `/ws`。

---

## 架构与访问入口

- **Web 管理面板**：`http://设备IP/`
- **API Base**：`http://设备IP/api/v1`
- **WebSocket**：`ws://设备IP/ws?token=...`

后端路由约定：
- `/api/v1/*`：REST API（未命中返回 JSON 404，不会被前端 SPA “吞掉”）
- `/ws`：实时推送（JWT 鉴权）
- 其它路径：静态资源优先，否则 SPA 回退到首页（支持前端路由刷新不 404）

---

## 前端构建（会被 embed 到后端）

### 依赖安装

在 `client-web/`：

```bash
pnpm install
```

### 生产构建（必须执行，才能 embed 最新前端）

```bash
cd client-web
pnpm build
```

说明：
- `pnpm build` 会把产物输出到 `client-nps/internal/webui/dist/`
- 后端使用 `go:embed` 将该目录内的文件打进二进制

---

## 后端构建（Go）

在 `client-nps/`：

```bash
go build -o nwct-client .
```

提示：
- 默认监听 **80 端口**，需要具备绑定 80 的权限（例如 root / capabilities）
- 如果你先 `go build`、后 `pnpm build`，前端变更不会进二进制；应先构建前端再构建后端

---

## 运行/启动方式

### 本机运行（开发调试）

```bash
cd client-nps
./nwct-client
```

访问：
- 管理面板：`http://127.0.0.1/`
- 初始化状态：`http://127.0.0.1/api/v1/config/init/status`

### 开发板运行（固件内启动）

将后端二进制写入固件并启动（示例）：

```bash
./nwct-client
```

然后在局域网浏览器打开：`http://设备IP/`

---

## 配置文件与关键参数

### 配置文件路径

后端启动时会加载配置（若不存在会写入默认配置）：
- **Linux/开发板**：默认 `/etc/nwct/config.json`
- **macOS/Windows 本地开发**：默认写到仓库根目录 `config.json`

可通过环境变量覆盖：
- **`NWCT_CONFIG_PATH`**：指定配置文件路径

### 端口（统一前后端同端口）

- 配置项：`config.json` → `server.port`
- 当前仓库默认值：`80`

> 若你修改了端口，请确保前端访问同源：生产环境会使用 `window.location.origin` 自动同源；开发环境可用 `VITE_API_BASE` 覆盖。

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

- 检查服务是否启动：
  - `curl http://127.0.0.1/api/v1/config/init/status`
- 浏览器打开面板：
  - `http://127.0.0.1/`


