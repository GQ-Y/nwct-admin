package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	frpmsg "github.com/fatedier/frp/pkg/msg"
	frpsserver "github.com/fatedier/frp/server"

	"totoro-node/internal/bridgeclient"
	"totoro-node/internal/envfile"
	"totoro-node/internal/frpswrap"
	"totoro-node/internal/kicker"
	"totoro-node/internal/limits"
	"totoro-node/internal/nodeapi"
	"totoro-node/internal/store"
	"totoro-node/internal/ticket"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// defaultBridgeURL 可在编译时通过 -ldflags "-X main.defaultBridgeURL=http://..." 覆盖
var defaultBridgeURL = "http://192.168.2.32:18090"

func firstMAC() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, it := range ifaces {
		// 过滤掉 loopback / down
		if it.Flags&net.FlagLoopback != 0 {
			continue
		}
		if it.Flags&net.FlagUp == 0 {
			continue
		}
		hw := strings.TrimSpace(it.HardwareAddr.String())
		if hw == "" {
			continue
		}
		return hw
	}
	return ""
}

func deriveNodeIDFromMAC(mac string) string {
	mac = strings.TrimSpace(strings.ToLower(mac))
	if mac == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(mac))
	// 取 8 字节，足够短且冲突概率极低
	return "node_" + hex.EncodeToString(sum[:8])
}

func genRandomKey(n int) string {
	if n <= 0 {
		n = 32
	}
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func main() {
	// 自动加载/生成 .env（无需用命令行手动注入环境变量）
	envfile.Bootstrap()

	// frps 配置文件（直接复用 frp 的 TOML/YAML/JSON 配置格式）
	frpsCfgFile := getenv("TOTOTO_FRPS_CONFIG", "./frps.toml")

	// 节点管理 API
	apiAddr := getenv("TOTOTO_NODE_API_ADDR", ":18080")

	// 本地节点状态库
	dbPath := getenv("TOTOTO_NODE_DB", "./node.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	// 让节点"开箱即用"：
	// - node_id：默认由 MAC 推导（稳定标识），fallback 随机
	// - node_key：首次启动随机生成并落库（用于票据校验/桥梁签发 ticket）
	// - bridge_url：编译时从环境变量 TOTOTO_BRIDGE_URL 读取，如果没有则使用程序内置 defaultBridgeURL，写入数据库后不再修改
	// - admin_key：默认 "admin"，可通过环境变量 TOTOTO_NODE_ADMIN_KEY 覆盖
	firstBoot := false
	if cfg, _, gerr := st.GetNodeConfig(); gerr != nil || strings.TrimSpace(cfg.NodeID) == "" {
		firstBoot = true
		mac := firstMAC()
		nodeID := deriveNodeIDFromMAC(mac)
		if nodeID == "" {
			nodeID = "node_" + genRandomKey(6)
		}
		nodeKey := genRandomKey(32)
		// 编译时从环境变量读取，如果没有则使用程序内置默认值
		bridgeURL := strings.TrimSpace(getenv("TOTOTO_BRIDGE_URL", strings.TrimSpace(defaultBridgeURL)))
		// 默认密码为 "admin"，可通过环境变量覆盖
		defaultAdminKey := getenv("TOTOTO_NODE_ADMIN_KEY", "admin")
		_ = st.InitNodeIfEmpty(store.NodeConfig{
			NodeID:    nodeID,
			NodeKey:   nodeKey,
			Public:    false,
			Name:      nodeID,
			BridgeURL: bridgeURL,
		}, defaultAdminKey)
	} else {
		// 如果数据库中没有 admin_key，从环境变量读取并写入（兼容旧库）
		dbAdminKey, _ := st.GetAdminKey()
		if strings.TrimSpace(dbAdminKey) == "" {
			envAdminKey := getenv("TOTOTO_NODE_ADMIN_KEY", "admin")
			_ = st.UpdateAdminKey(envAdminKey)
		}
		// 如果数据库中没有 bridge_url，从环境变量或默认值读取并写入（兼容旧库）
		if strings.TrimSpace(cfg.BridgeURL) == "" {
			bridgeURL := strings.TrimSpace(getenv("TOTOTO_BRIDGE_URL", strings.TrimSpace(defaultBridgeURL)))
			_ = st.UpdateBridgeURL(bridgeURL)
		}
	}

	cfg0, _, _ := st.GetNodeConfig()
	nodeID := strings.TrimSpace(cfg0.NodeID)
	nodeKey := strings.TrimSpace(cfg0.NodeKey)
	if nodeID == "" || nodeKey == "" {
		log.Fatalf("node config invalid (node_id/node_key empty), db=%s", dbPath)
	}
	if firstBoot {
		// 仅首启打印一次，便于节点所有者保存（后续可通过节点 UI/API 再设置自己的管理鉴权）
		log.Printf("node initialized: node_id=%s node_key=%s", nodeID, nodeKey)
	}

	// Beta：frps hook（票据 + 配额）
	lim := limits.New()
	type scopePool struct {
		Min int `json:"min"`
		Max int `json:"max"`
	}
	type scope struct {
		MaxConns   int        `json:"max_conns"`
		MaxProxies int        `json:"max_proxies"`
		TCP        *scopePool `json:"tcp_port_pool"`
		UDP        *scopePool `json:"udp_port_pool"`
	}

	frpsserver.TotoroLoginHook = func(remoteAddr string, login *frpmsg.Login) (*frpsserver.TotoroLoginResult, error) {
		if login == nil {
			return nil, fmt.Errorf("nil login")
		}
		if login.Metas == nil {
			return nil, fmt.Errorf("missing metas")
		}
		tok := strings.TrimSpace(login.Metas["totoro_ticket"])
		if tok == "" {
			return nil, fmt.Errorf("missing totoro_ticket")
		}
		claims, err := ticket.VerifyHMAC(tok, []byte(nodeKey))
		if err != nil {
			return nil, fmt.Errorf("ticket_invalid")
		}
		if claims.NodeID != nodeID {
			return nil, fmt.Errorf("ticket_node_mismatch")
		}
		// 把 exp 写入 metas，供后续“到期主动断开”使用
		if claims.ExpiresAt != nil && !claims.ExpiresAt.Time.IsZero() {
			login.Metas["totoro_ticket_exp_unix"] = fmt.Sprintf("%d", claims.ExpiresAt.Time.Unix())
		}
		// scope
		var sc scope
		if len(claims.Scope) > 0 && string(claims.Scope) != "null" {
			_ = json.Unmarshal(claims.Scope, &sc)
		}
		if !lim.CanOpenConn(claims.InviteID, sc.MaxConns) {
			return nil, fmt.Errorf("exceed_max_conns")
		}
		// 保存 scope 供 proxy hook 使用
		login.Metas["totoro_invite_id"] = claims.InviteID
		login.Metas["totoro_scope"] = string(claims.Scope)
		return &frpsserver.TotoroLoginResult{InviteID: claims.InviteID}, nil
	}

	// 注册连接，用于“撤销邀请码 -> 主动踢下线”和“ticket 到期 -> 主动踢下线”
	frpsserver.TotoroControlCreatedHook = func(ctl *frpsserver.Control) {
		kicker.RegisterControl(ctl)
	}

	frpsserver.TotoroNewProxyHook = func(ctl *frpsserver.Control, pxyMsg *frpmsg.NewProxy) error {
		if ctl == nil || pxyMsg == nil || ctl.LoginMsg() == nil {
			return fmt.Errorf("invalid context")
		}
		metas := ctl.LoginMsg().Metas
		inviteID := strings.TrimSpace(metas["totoro_invite_id"])
		if inviteID == "" {
			return fmt.Errorf("missing_invite")
		}
		var sc scope
		if s := strings.TrimSpace(metas["totoro_scope"]); s != "" && s != "null" {
			_ = json.Unmarshal([]byte(s), &sc)
		}
		if !lim.CanAddProxy(inviteID, sc.MaxProxies) {
			return fmt.Errorf("exceed_max_proxies")
		}
		// 端口池校验（tcp/udp）
		if pxyMsg.ProxyType == "tcp" && sc.TCP != nil {
			if pxyMsg.RemotePort == 0 {
				pxyMsg.RemotePort = lim.NextPort(inviteID, sc.TCP.Min, sc.TCP.Max)
			}
			if pxyMsg.RemotePort < sc.TCP.Min || pxyMsg.RemotePort > sc.TCP.Max {
				return fmt.Errorf("port_out_of_pool")
			}
		}
		if pxyMsg.ProxyType == "udp" && sc.UDP != nil {
			if pxyMsg.RemotePort == 0 {
				pxyMsg.RemotePort = lim.NextPort(inviteID, sc.UDP.Min, sc.UDP.Max)
			}
			if pxyMsg.RemotePort < sc.UDP.Min || pxyMsg.RemotePort > sc.UDP.Max {
				return fmt.Errorf("port_out_of_pool")
			}
		}
		return nil
	}

	frpsserver.TotoroProxyClosedHook = func(ctl *frpsserver.Control, proxyName string) {
		if ctl == nil || ctl.LoginMsg() == nil {
			return
		}
		inviteID := strings.TrimSpace(ctl.LoginMsg().Metas["totoro_invite_id"])
		if inviteID == "" {
			return
		}
		lim.RemoveProxy(inviteID)
	}
	frpsserver.TotoroControlClosedHook = func(ctl *frpsserver.Control) {
		if ctl == nil || ctl.LoginMsg() == nil {
			return
		}
		inviteID := strings.TrimSpace(ctl.LoginMsg().Metas["totoro_invite_id"])
		if inviteID == "" {
			return
		}
		lim.CloseConn(inviteID)
		kicker.UnregisterControl(ctl)
	}

	// 启动 frps（在同进程）
	svrCfg, err := frpswrap.LoadAndValidate(frpsCfgFile)
	if err != nil {
		log.Fatalf("load frps config: %v", err)
	}

	// endpoints 同步：\n
	// - 未配置则自动补一个（便于 MVP 直接跑通）\n
	// - 已配置但端口与当前 bindPort 不一致时，自动更新（避免改端口后残留旧值）\n
	// 生产建议显式配置公网域名/IP。\n
	if cfg, _, err := st.GetNodeConfig(); err == nil {
		publicAddr := strings.TrimSpace(getenv("TOTOTO_NODE_PUBLIC_ADDR", "127.0.0.1"))
		changed := false
		if len(cfg.Endpoints) == 0 {
			cfg.Endpoints = []store.NodeEndpoint{{Addr: publicAddr, Port: svrCfg.BindPort, Proto: "tcp"}}
			changed = true
		} else {
			for i := range cfg.Endpoints {
				ep := cfg.Endpoints[i]
				if strings.TrimSpace(ep.Proto) == "" {
					ep.Proto = "tcp"
				}
				// 只同步“默认 endpoint”（addr=publicAddr 且 proto=tcp），避免覆盖用户自定义多 endpoint
				if ep.Proto == "tcp" && ep.Addr == publicAddr && ep.Port != svrCfg.BindPort {
					ep.Port = svrCfg.BindPort
					cfg.Endpoints[i] = ep
					changed = true
				}
			}
		}
		if changed {
			_ = st.UpdateNodeConfig(nodeKey, store.NodeConfig{Public: cfg.Public, Endpoints: cfg.Endpoints})
		}
	}

	runner, err := frpswrap.NewRunner(svrCfg)
	if err != nil {
		log.Fatalf("init frps runner: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)
	log.Printf("frps started (config=%s)", frpsCfgFile)

	// 启动节点管理 API（AdminKey 从数据库读取，不再从环境变量）
	go func() {
		r := nodeapi.NewRouter(st, nodeapi.Options{TicketKey: []byte(nodeKey)})
		log.Printf("totoro-node api listening on %s (db=%s)", apiAddr, dbPath)
		if err := r.Run(apiAddr); err != nil {
			log.Fatalf("node api run: %v", err)
		}
	}()

	// 节点上报（只要 bridge_url 已配置就上报；public=false 也要上报以便桥梁及时更新状态/能力字段，并避免 public 列表残留旧数据）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		httpClient := &bridgeclient.Client{
			NodeID:  nodeID,
			NodeKey: nodeKey,
		}
		// node_api：默认使用 public_addr + api 端口（可通过环境变量覆盖）
		apiPort := strings.TrimLeft(strings.TrimSpace(apiAddr), ":")
		defaultNodeAPI := "http://" + strings.TrimSpace(getenv("TOTOTO_NODE_PUBLIC_ADDR", "127.0.0.1")) + ":" + apiPort
		publicNodeAPI := strings.TrimSpace(getenv("TOTOTO_NODE_PUBLIC_API", defaultNodeAPI))
		for range ticker.C {
			cfg, _, err := st.GetNodeConfig()
			if err != nil {
				continue
			}
			if strings.TrimSpace(cfg.BridgeURL) == "" {
				continue
			}
			httpClient.BaseURL = cfg.BridgeURL

			hb := bridgeclient.Heartbeat{
				Ts:          time.Now().UTC().Format(time.RFC3339),
				NodeID:      cfg.NodeID,
				Public:      cfg.Public,
				Name:        cfg.Name,
				Description: cfg.Description,
				Region:      cfg.Region,
				ISP:         cfg.ISP,
				Tags:        cfg.Tags,
				NodeAPI:     publicNodeAPI,
				Endpoints: func() []any {
					out := make([]any, 0, len(cfg.Endpoints))
					for _, ep := range cfg.Endpoints {
						out = append(out, ep)
					}
					return out
				}(),
				DomainSuffix: cfg.DomainSuffix,
				HTTPEnabled:  cfg.HTTPEnabled,
				HTTPSEnabled: cfg.HTTPSEnabled,
				Version: map[string]any{
					"totoro_node": "0.0.1",
				},
				// Metrics：后续可接入 frps 内部 metrics，这里先留空
			}
			_ = httpClient.SendHeartbeat(hb)
		}
	}()

	// 阻塞
	select {}
}
