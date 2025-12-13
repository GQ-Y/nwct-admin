## 本地 Docker 部署 NPS（服务端）

### 启动

在项目根目录执行：

```bash
cd deploy/nps
docker compose up -d
docker logs -f nwct-nps
```

### 访问管理后台

- Web 管理地址：`http://127.0.0.1:19080`
- 默认账号：`admin`
- 默认密码：`123`

> 账号密码与端口可在 `deploy/nps/conf/nps.conf` 中修改。

### 客户端（本项目 device 端）接入流程

1. 登录 NPS Web → 创建一个客户端（Client），拿到 **vkey**
2. 在本项目的 Go 客户端调用：
   - `POST /api/v1/nps/npc/install`（可选，一键安装 npc）
   - `POST /api/v1/nps/connect`（填 server/vkey/client_id）

示例（server 在本机）：

- server：`127.0.0.1:19024`


