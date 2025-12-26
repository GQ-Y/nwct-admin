package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"totoro-node/internal/bridgeclient"
	"totoro-node/internal/frpswrap"
	"totoro-node/internal/nodeapi"
	"totoro-node/internal/store"
)

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	// 基础身份（节点）
	nodeID := strings.TrimSpace(getenv("TOTOTO_NODE_ID", "node_local"))
	nodeKey := strings.TrimSpace(getenv("TOTOTO_NODE_KEY", "change_me"))

	// frps 配置文件（直接复用 frp 的 TOML/YAML/JSON 配置格式）
	frpsCfgFile := getenv("TOTOTO_FRPS_CONFIG", "./frps.toml")

	// 节点管理 API
	apiAddr := getenv("TOTOTO_NODE_API_ADDR", ":18080")
	adminKey := getenv("TOTOTO_NODE_ADMIN_KEY", "") // 建议生产设置；为空则不校验

	// 本地节点状态库
	dbPath := getenv("TOTOTO_NODE_DB", "./node.db")
	st, err := store.Open(dbPath)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()
	_ = st.InitNodeIfEmpty(store.NodeConfig{
		NodeID:  nodeID,
		NodeKey: nodeKey,
		Public:  false,
		Name:    nodeID,
	})

	// 启动 frps（在同进程）
	svrCfg, err := frpswrap.LoadAndValidate(frpsCfgFile)
	if err != nil {
		log.Fatalf("load frps config: %v", err)
	}
	runner, err := frpswrap.NewRunner(svrCfg)
	if err != nil {
		log.Fatalf("init frps runner: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runner.Run(ctx)
	log.Printf("frps started (config=%s)", frpsCfgFile)

	// 启动节点管理 API
	go func() {
		r := nodeapi.NewRouter(st, nodeapi.Options{AdminKey: adminKey})
		log.Printf("totoro-node api listening on %s (db=%s)", apiAddr, dbPath)
		if err := r.Run(apiAddr); err != nil {
			log.Fatalf("node api run: %v", err)
		}
	}()

	// 节点上报（仅当 public=true 且 bridge_url 已配置）
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		httpClient := &bridgeclient.Client{
			NodeID:  nodeID,
			NodeKey: nodeKey,
		}
		for range ticker.C {
			cfg, _, err := st.GetNodeConfig()
			if err != nil {
				continue
			}
			if !cfg.Public {
				continue
			}
			if strings.TrimSpace(cfg.BridgeURL) == "" {
				continue
			}
			httpClient.BaseURL = cfg.BridgeURL

			hb := bridgeclient.Heartbeat{
				Ts:           time.Now().UTC().Format(time.RFC3339),
				NodeID:       cfg.NodeID,
				Public:       cfg.Public,
				Name:         cfg.Name,
				Region:       cfg.Region,
				ISP:          cfg.ISP,
				Tags:         cfg.Tags,
				Endpoints:    cfg.Endpoints,
				DomainSuffix: cfg.DomainSuffix,
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


