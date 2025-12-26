# Totoro 服务端 FSD（无注册版：邀请码 + 节点密钥）

## 1. 范围说明

本 FSD 覆盖：`totoro-control-plane`（控制面）、`totoro-frps`（数据面小改造）、`totoro-node-agent`（节点侧，可选但建议）的 **功能规格 + API 契约**。  
核心约束：**终端用户无需注册/登录**；连接授权只依赖：

- **节点密钥（node_key）**：节点与控制面/数据面之间的可信身份
- **邀请码（invite_code）**：用于分享私有节点的解析入口
- **短期连接票据（connection_ticket）**：由控制面签发的短期签名票据（可用 JWT 结构，但不用于“用户登录鉴权”）

---

## 2. 术语与对象

- **Node（节点）**：运行 `totoro-frps` 的服务实例，通常配套 `totoro-node-agent`。
- **PublicNode（公开节点）**：会出现在官方节点列表的节点。
- **PrivateNode（私有节点）**：不出现在公开列表，仅邀请码可解析。
- **Invite（邀请码）**：绑定节点的分享凭证，具备 TTL/次数/范围限制。
- **ConnectionTicket（连接票据）**：客户端连接 frps 使用的短期签名票据，绑定 `node_id + invite_id + scope`。
- **Session（会话）**：一次 frpc 与 frps 的连接生命周期（可用于统计与审计）。
- **Proxy（代理/隧道）**：frpc 注册到 frps 的 proxy（tcp/udp/http/https…）。

---

## 3. 身份与鉴权（仅 3 类）

### 3.1 Admin（平台管理员）

用途：运维控制面（创建节点、查看全局统计、维护域名/证书策略等）。  
形式：单个或多个静态密钥（不做账号系统）。

- Header：`X-Admin-Key: <admin_key>`

### 3.2 Node（节点身份）

用途：节点自注册/心跳/拉取策略/生成邀请码（节点范围内）。  
形式：`node_id + node_key`（或等价签名机制）。

建议：
- Header：`X-Node-Id: <node_id>`
- Header：`X-Node-Key: <node_key>`

### 3.3 Client（匿名客户端）

用途：解析邀请码、获取连接票据、连接 frps。  
形式：不登录；只在 `invites/resolve` 返回的 `connection_ticket` 中携带 scope。

> 注意：`connection_ticket` 可以用 JWT 格式（便于解析/签名/过期），但语义是“连接票据”，不是“用户登录 JWT”。  

---

## 4. 控制面 API（对外）

统一前缀：`/api/v1`

### 4.1 公开节点列表（客户端）

#### GET `/public/nodes`

用途：客户端启动或手动刷新官方公开节点列表。

Query（可选）：
- `region`: string（例如 `cn-east`）
- `isp`: string（例如 `cmcc/ctcc/cu`）
- `tag`: string（例如 `free/paid/high_bw`）

Response 200：

```json
{
  "code": 0,
  "data": {
    "nodes": [
      {
        "node_id": "node_xxx",
        "name": "Tokyo-1",
        "public": true,
        "status": "online",
        "region": "jp",
        "isp": "ntt",
        "tags": ["free"],
        "endpoints": [
          { "addr": "frps.example.com", "port": 7000, "proto": "tcp" }
        ],
        "domain_suffix": "node.example.com",
        "tcp_port_pool": { "min": 20000, "max": 29999 },
        "udp_port_pool": { "min": 30000, "max": 39999 },
        "updated_at": "2025-12-26T00:00:00Z"
      }
    ]
  }
}
```

说明：
- `status` 由 node-agent 心跳决定（`online/offline/degraded`）。

---

### 4.2 邀请码解析（客户端）

#### POST `/invites/resolve`

用途：客户端输入邀请码，换取节点连接信息与短期连接票据。

Request：

```json
{
  "code": "ABCD-EFGH-IJKL",
  "client_meta": {
    "app": "nwct-client",
    "version": "0.1.0",
    "platform": "linux"
  }
}
```

Response 200（成功）：

```json
{
  "code": 0,
  "data": {
    "node": {
      "node_id": "node_xxx",
      "name": "MyPrivateNode",
      "endpoints": [
        { "addr": "1.2.3.4", "port": 7000, "proto": "tcp" }
      ],
      "domain_suffix": "node.example.com"
    },
    "connection_ticket": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9....",
    "scope": {
      "allow_types": ["tcp","udp","http","https"],
      "tcp_port_pool": { "min": 20000, "max": 20999 },
      "udp_port_pool": { "min": 30000, "max": 30999 },
      "max_proxies": 10,
      "max_conns": 2,
      "max_bandwidth_kbps": 0,
      "max_traffic_mb": 0,
      "domain_policy": {
        "mode": "allocate_subdomain",
        "suffix": "node.example.com"
      }
    },
    "expires_at": "2025-12-26T01:00:00Z"
  }
}
```

失败响应（示例）：
- 400：`code` 格式非法
- 404：邀请码不存在
- 410：邀请码过期
- 429：次数用尽/触发限频

备注：
- 这里返回 `scope` 是为了让客户端 UI 做提示；真正的强约束在 frps 的鉴权回调里执行。

---

## 5. 控制面 API（节点/管理员）

### 5.1 节点自注册（可选）

如果你倾向“节点完全由管理员创建”，可跳过该接口；node-agent 仅做心跳。

#### POST `/nodes/register`（Node）

Headers：
- `X-Node-Id`
- `X-Node-Key`

Request：

```json
{
  "name": "Tokyo-1",
  "region": "jp",
  "isp": "ntt",
  "endpoints": [{ "addr": "frps.example.com", "port": 7000, "proto": "tcp" }],
  "domain_suffix": "node.example.com",
  "public": false
}
```

Response 200：返回 node 的当前配置与策略版本。

---

### 5.2 节点心跳（Node）

#### POST `/nodes/heartbeat`

Headers：
- `X-Node-Id`
- `X-Node-Key`

Request：

```json
{
  "ts": "2025-12-26T00:00:00Z",
  "version": {
    "node_agent": "0.1.0",
    "frps": "0.52.3-totoro1"
  },
  "metrics": {
    "cpu": 12.3,
    "mem": 43.1,
    "disk": 66.0,
    "net_rx_kbps": 1234,
    "net_tx_kbps": 2345,
    "frps_conns": 12,
    "frps_proxies": 30,
    "frps_auth_fail": 2
  }
}
```

Response 200：

```json
{
  "code": 0,
  "data": {
    "status": "ok",
    "policy_rev": 12
  }
}
```

---

### 5.3 节点生成邀请码（Node）

#### POST `/nodes/invites`

Headers：
- `X-Node-Id`
- `X-Node-Key`

Request：

```json
{
  "public_hint": false,
  "ttl_seconds": 86400,
  "max_uses": 50,
  "scope": {
    "allow_types": ["tcp","udp","http","https"],
    "tcp_port_pool": { "min": 20000, "max": 20999 },
    "udp_port_pool": { "min": 30000, "max": 30999 },
    "max_proxies": 10,
    "max_conns": 2,
    "domain_policy": { "mode": "allocate_subdomain" }
  }
}
```

Response 200：

```json
{
  "code": 0,
  "data": {
    "invite_id": "inv_xxx",
    "code": "ABCD-EFGH-IJKL",
    "expires_at": "2025-12-27T00:00:00Z",
    "max_uses": 50,
    "used": 0
  }
}
```

---

### 5.4 节点撤销邀请码（Node）

#### POST `/nodes/invites/revoke`

Headers：
- `X-Node-Id`
- `X-Node-Key`

Request：

```json
{ "invite_id": "inv_xxx" }
```

---

### 5.5 管理员节点管理（Admin）

如果你想把节点创建/公开开关/域名后缀等完全放到控制面运维侧，可提供以下接口（用 `X-Admin-Key` 保护）：\n
- `POST /admin/nodes`（创建）\n
- `PUT /admin/nodes/{node_id}`（更新）\n
- `GET /admin/nodes`（列表）\n
- `POST /admin/nodes/{node_id}/rotate-key`（轮换 node_key）\n

> 这些接口不引入“用户注册”，只是运维能力。\n

---

## 6. frps -> 控制面鉴权回调（内部 API）

统一前缀：`/api/v1/frps/auth/*`  
调用方：`totoro-frps`  
鉴权：建议使用 `X-FRPS-Key: <shared_secret>` 或 mTLS（二选一，优先 shared_secret + 内网隔离）。

### 6.1 握手鉴权

#### POST `/frps/auth/handshake`

Request：

```json
{
  "node_id": "node_xxx",
  "remote_addr": "8.8.8.8:52333",
  "token": "connection_ticket_or_raw_token",
  "client_info": {
    "frp_version": "0.52.3",
    "os": "linux",
    "arch": "arm64"
  }
}
```

Response 200：

```json
{
  "allow": true,
  "reason": "",
  "session": {
    "session_id": "sess_xxx",
    "invite_id": "inv_xxx",
    "expires_at": "2025-12-26T01:00:00Z"
  },
  "limits": {
    "max_conns": 2,
    "max_proxies": 10,
    "max_bandwidth_kbps": 0,
    "max_traffic_mb": 0
  }
}
```

失败：

```json
{ "allow": false, "reason": "invite_expired" }
```

说明：
- `token` 优先放 `connection_ticket`（签名票据）。如未来支持“长期 token”，也可在这里兼容。
- 控制面需要对 `node_id` 做一致性校验：票据中的 `node_id` 必须等于请求 `node_id`。

---

### 6.2 代理注册鉴权（proxy）

#### POST `/frps/auth/proxy`

Request：

```json
{
  "session_id": "sess_xxx",
  "node_id": "node_xxx",
  "proxy": {
    "name": "p1",
    "type": "tcp",
    "remote_port": 20001,
    "custom_domain": "",
    "subdomain": ""
  }
}
```

Response 200（允许）：

```json
{
  "allow": true,
  "reason": "",
  "assigned": {
    "remote_port": 20001,
    "custom_domain": "",
    "subdomain": "a1b2c3"
  },
  "limits": {
    "bandwidth_kbps": 0,
    "traffic_mb": 0
  }
}
```

拒绝示例：

```json
{ "allow": false, "reason": "port_out_of_pool" }
```

说明：
- **TCP/UDP**：校验端口池；`remote_port=0` 时可由控制面分配端口后返回 `assigned.remote_port`。
- **HTTP/HTTPS**：建议优先 `allocate_subdomain`，由控制面分配 `subdomain`；若用户提供 `custom_domain`，需先在控制面完成域名所有权校验（Beta 再做）。

---

### 6.3 会话关闭回调（可选）

#### POST `/frps/auth/session_close`

用途：frps 在连接断开时通知控制面做会话统计落盘。

---

## 7. 数据模型（最小表）

> 这里定义“必须落库的对象”，具体字段可按实现调整。

- `nodes`：`node_id, node_key_hash, name, public, endpoints, domain_suffix, region, isp, tags, status, updated_at`
- `invites`：`invite_id, node_id, code_hash, ttl, expires_at, max_uses, used, scope_json, revoked_at`
- `sessions`：`session_id, node_id, invite_id, remote_addr, opened_at, closed_at, close_reason`
- `proxies`：`proxy_id, session_id, node_id, name, type, remote_port, domain, created_at, closed_at, status`
- `usage_samples`（可先聚合）：`node_id, invite_id, proxy_id, rx_bytes, tx_bytes, ts_bucket`

---

## 8. 业务规则（强约束）

### 8.1 邀请码规则

- `code` 只保存 hash（数据库不落明文）
- 可配置：TTL、max_uses、scope（端口池/类型/数量/并发）
- `resolve` 成功会增加 `used`（或记录一次使用事件）\n

### 8.2 连接票据规则

- `connection_ticket` 必须包含：`node_id, invite_id, exp, scope_hash, nonce`
- 票据签名密钥：只在控制面保存
- frps 必须通过回调校验票据（不在 frps 侧本地验签，保持一致的策略中心）

### 8.3 配额规则（MVP）

- `max_conns`：同一 `invite_id` 同时在线会话数上限
- `max_proxies`：同一 `invite_id` 当前生效代理数上限
- 端口池：TCP/UDP 的 `remote_port` 必须落在允许范围

---

## 9. 与客户端对接建议（兼容你现状）

你现有 `client-nps` 使用 `frpc` 的 `token` 字段连接：\n
- 把 `invites/resolve` 返回的 `connection_ticket` 直接写入客户端的 `frpc` 配置 token（或在 `/api/v1/frp/connect` 请求里传入 token）。\n
- `server_addr/server_port` 使用 resolve 返回的 `endpoints[0]`。\n
\n
这样可以实现：客户端无需注册、仅输入邀请码即可“一键连接私有节点”。\n


