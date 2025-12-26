package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store interface {
	UpsertNodeHeartbeat(hb NodeHeartbeat) error
	ListPublicNodes() ([]PublicNode, error)
	Close() error
}

type SQLiteStore struct {
	db *sql.DB
}

type PublicNode struct {
	NodeID        string            `json:"node_id"`
	Name          string            `json:"name"`
	Public        bool              `json:"public"`
	Status        string            `json:"status"`
	Region        string            `json:"region"`
	ISP           string            `json:"isp"`
	Tags          []string          `json:"tags"`
	Endpoints     []NodeEndpoint    `json:"endpoints"`
	DomainSuffix  string            `json:"domain_suffix"`
	TCPPortPool   *PortPool         `json:"tcp_port_pool,omitempty"`
	UDPPortPool   *PortPool         `json:"udp_port_pool,omitempty"`
	UpdatedAt     string            `json:"updated_at"`
	Extra         map[string]string `json:"extra,omitempty"`
	HeartbeatAgeS int64             `json:"heartbeat_age_s"`
}

type NodeEndpoint struct {
	Addr  string `json:"addr"`
	Port  int    `json:"port"`
	Proto string `json:"proto"`
}

type PortPool struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type NodeHeartbeat struct {
	NodeID       string
	NodeKey      string // 已验证通过的 node_key（bridge 不存明文，只用于鉴权阶段；此处保留字段，便于未来扩展）
	Public       bool
	Name         string
	Region       string
	ISP          string
	Tags         []string
	Endpoints    []NodeEndpoint
	DomainSuffix string
	TCPPortPool  *PortPool
	UDPPortPool  *PortPool
	MetricsJSON  json.RawMessage
	VersionJSON  json.RawMessage
	ExtraJSON    json.RawMessage
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	// WAL 提升并发读写
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	st := &SQLiteStore{db: db}
	if err := st.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return st, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS nodes (
  node_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  public INTEGER NOT NULL DEFAULT 0,
  region TEXT NOT NULL DEFAULT '',
  isp TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  endpoints_json TEXT NOT NULL DEFAULT '[]',
  domain_suffix TEXT NOT NULL DEFAULT '',
  tcp_pool_json TEXT NOT NULL DEFAULT 'null',
  udp_pool_json TEXT NOT NULL DEFAULT 'null',
  version_json TEXT NOT NULL DEFAULT 'null',
  metrics_json TEXT NOT NULL DEFAULT 'null',
  extra_json TEXT NOT NULL DEFAULT 'null',
  last_seen_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_nodes_public_lastseen ON nodes(public, last_seen_at);
`
	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStore) UpsertNodeHeartbeat(hb NodeHeartbeat) error {
	if hb.NodeID == "" {
		return fmt.Errorf("node_id required")
	}
	tags, _ := json.Marshal(hb.Tags)
	eps, _ := json.Marshal(hb.Endpoints)
	tcp, _ := json.Marshal(hb.TCPPortPool)
	udp, _ := json.Marshal(hb.UDPPortPool)

	now := time.Now().Unix()
	_, err := s.db.Exec(`
INSERT INTO nodes(node_id,name,public,region,isp,tags_json,endpoints_json,domain_suffix,tcp_pool_json,udp_pool_json,version_json,metrics_json,extra_json,last_seen_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  name=excluded.name,
  public=excluded.public,
  region=excluded.region,
  isp=excluded.isp,
  tags_json=excluded.tags_json,
  endpoints_json=excluded.endpoints_json,
  domain_suffix=excluded.domain_suffix,
  tcp_pool_json=excluded.tcp_pool_json,
  udp_pool_json=excluded.udp_pool_json,
  version_json=excluded.version_json,
  metrics_json=excluded.metrics_json,
  extra_json=excluded.extra_json,
  last_seen_at=excluded.last_seen_at
`, hb.NodeID, hb.Name, boolToInt(hb.Public), hb.Region, hb.ISP, string(tags), string(eps), hb.DomainSuffix, string(tcp), string(udp), string(hb.VersionJSON), string(hb.MetricsJSON), string(hb.ExtraJSON), now)
	return err
}

func (s *SQLiteStore) ListPublicNodes() ([]PublicNode, error) {
	rows, err := s.db.Query(`
SELECT node_id,name,public,region,isp,tags_json,endpoints_json,domain_suffix,tcp_pool_json,udp_pool_json,last_seen_at
FROM nodes
WHERE public=1
ORDER BY last_seen_at DESC
LIMIT 200
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]PublicNode, 0, 64)
	now := time.Now().Unix()
	for rows.Next() {
		var (
			nodeID, name, region, isp, tagsJSON, epsJSON, domainSuffix, tcpJSON, udpJSON string
			publicInt                                                     int
			lastSeen                                                      int64
		)
		if err := rows.Scan(&nodeID, &name, &publicInt, &region, &isp, &tagsJSON, &epsJSON, &domainSuffix, &tcpJSON, &udpJSON, &lastSeen); err != nil {
			return nil, err
		}
		var tags []string
		var eps []NodeEndpoint
		var tcp *PortPool
		var udp *PortPool
		_ = json.Unmarshal([]byte(tagsJSON), &tags)
		_ = json.Unmarshal([]byte(epsJSON), &eps)
		_ = json.Unmarshal([]byte(tcpJSON), &tcp)
		_ = json.Unmarshal([]byte(udpJSON), &udp)

		age := now - lastSeen
		status := "online"
		if age > 90 {
			status = "offline"
		} else if age > 30 {
			status = "degraded"
		}

		out = append(out, PublicNode{
			NodeID:        nodeID,
			Name:          name,
			Public:        publicInt == 1,
			Status:        status,
			Region:        region,
			ISP:           isp,
			Tags:          tags,
			Endpoints:     eps,
			DomainSuffix:  domainSuffix,
			TCPPortPool:   tcp,
			UDPPortPool:   udp,
			UpdatedAt:     time.Unix(lastSeen, 0).UTC().Format(time.RFC3339),
			HeartbeatAgeS: age,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}


