package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store interface {
	UpsertNodeAuth(nodeID string, nodeKey string) error
	VerifyNodeAuth(nodeID string, nodeKey string) (bool, error)
	UpsertNodeHeartbeat(hb NodeHeartbeat) error
	ListPublicNodes() ([]PublicNode, error)
	// device auth
	UpsertDeviceWhitelist(deviceID string, mac string, enabled bool, note string) error
	DeleteDeviceWhitelist(deviceID string) error
	ListDeviceWhitelist(limit int, offset int) ([]DeviceWhitelistRow, int, error)
	VerifyDeviceWhitelist(deviceID string, mac string) (bool, error)
	CreateDeviceSession(deviceID string, mac string, ttlSeconds int) (token string, expiresAt int64, err error)
	VerifyDeviceSession(token string) (bool, string, error)
	// official nodes
	UpsertOfficialNode(n OfficialNode) error
	DeleteOfficialNode(nodeID string) error
	ListOfficialNodes() ([]OfficialNode, error)
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
	NodeAPI       string            `json:"node_api,omitempty"`
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
	NodeAPI      string
	DomainSuffix string
	TCPPortPool  *PortPool
	UDPPortPool  *PortPool
	MetricsJSON  json.RawMessage
	VersionJSON  json.RawMessage
	ExtraJSON    json.RawMessage
}

// OfficialNode 官方内置节点（由桥梁平台管理）
type OfficialNode struct {
	NodeID       string `json:"node_id"`
	Name         string `json:"name"`
	Server       string `json:"server"`      // frps addr，例如 117.xx.xx.xx:7000
	Token        string `json:"token"`       // frps token（builtin 用）
	AdminAddr    string `json:"admin_addr"`  // 可选
	AdminUser    string `json:"admin_user"`  // 可选
	AdminPwd     string `json:"admin_pwd"`   // 可选
	NodeAPI      string `json:"node_api"`    // 可选，节点管理 API 地址（公开节点可用）
	DomainSuffix string `json:"domain_suffix"`
	UpdatedAt    string `json:"updated_at"`
}

type DeviceWhitelistRow struct {
	DeviceID  string `json:"device_id"`
	MAC       string `json:"mac"`
	Enabled   bool   `json:"enabled"`
	Note      string `json:"note"`
	UpdatedAt string `json:"updated_at"`
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
CREATE TABLE IF NOT EXISTS node_auth (
  node_id TEXT PRIMARY KEY,
  node_key_hash TEXT NOT NULL,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nodes (
  node_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  public INTEGER NOT NULL DEFAULT 0,
  region TEXT NOT NULL DEFAULT '',
  isp TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  endpoints_json TEXT NOT NULL DEFAULT '[]',
  node_api TEXT NOT NULL DEFAULT '',
  domain_suffix TEXT NOT NULL DEFAULT '',
  tcp_pool_json TEXT NOT NULL DEFAULT 'null',
  udp_pool_json TEXT NOT NULL DEFAULT 'null',
  version_json TEXT NOT NULL DEFAULT 'null',
  metrics_json TEXT NOT NULL DEFAULT 'null',
  extra_json TEXT NOT NULL DEFAULT 'null',
  last_seen_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_nodes_public_lastseen ON nodes(public, last_seen_at);

CREATE TABLE IF NOT EXISTS device_whitelist (
  device_id TEXT PRIMARY KEY,
  mac TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1,
  note TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS device_sessions (
  token TEXT PRIMARY KEY,
  device_id TEXT NOT NULL,
  mac TEXT NOT NULL DEFAULT '',
  expires_at INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_device_sessions_expires ON device_sessions(expires_at);

CREATE TABLE IF NOT EXISTS official_nodes (
  node_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  server TEXT NOT NULL DEFAULT '',
  token TEXT NOT NULL DEFAULT '',
  admin_addr TEXT NOT NULL DEFAULT '',
  admin_user TEXT NOT NULL DEFAULT '',
  admin_pwd TEXT NOT NULL DEFAULT '',
  node_api TEXT NOT NULL DEFAULT '',
  domain_suffix TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0
);
`
	_, err := s.db.Exec(schema)
	return err
}

func hashKey(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func (s *SQLiteStore) UpsertNodeAuth(nodeID string, nodeKey string) error {
	if nodeID == "" || nodeKey == "" {
		return fmt.Errorf("node_id/node_key required")
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`
INSERT INTO node_auth(node_id,node_key_hash,updated_at)
VALUES(?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  node_key_hash=excluded.node_key_hash,
  updated_at=excluded.updated_at
`, nodeID, hashKey(nodeKey), now)
	return err
}

func (s *SQLiteStore) VerifyNodeAuth(nodeID string, nodeKey string) (bool, error) {
	if nodeID == "" || nodeKey == "" {
		return false, nil
	}
	row := s.db.QueryRow(`SELECT node_key_hash FROM node_auth WHERE node_id=?`, nodeID)
	var want string
	if err := row.Scan(&want); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return hashKey(nodeKey) == want, nil
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
INSERT INTO nodes(node_id,name,public,region,isp,tags_json,endpoints_json,node_api,domain_suffix,tcp_pool_json,udp_pool_json,version_json,metrics_json,extra_json,last_seen_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  name=excluded.name,
  public=excluded.public,
  region=excluded.region,
  isp=excluded.isp,
  tags_json=excluded.tags_json,
  endpoints_json=excluded.endpoints_json,
  node_api=excluded.node_api,
  domain_suffix=excluded.domain_suffix,
  tcp_pool_json=excluded.tcp_pool_json,
  udp_pool_json=excluded.udp_pool_json,
  version_json=excluded.version_json,
  metrics_json=excluded.metrics_json,
  extra_json=excluded.extra_json,
  last_seen_at=excluded.last_seen_at
`, hb.NodeID, hb.Name, boolToInt(hb.Public), hb.Region, hb.ISP, string(tags), string(eps), hb.NodeAPI, hb.DomainSuffix, string(tcp), string(udp), string(hb.VersionJSON), string(hb.MetricsJSON), string(hb.ExtraJSON), now)
	return err
}

func (s *SQLiteStore) ListPublicNodes() ([]PublicNode, error) {
	rows, err := s.db.Query(`
SELECT node_id,name,public,region,isp,tags_json,endpoints_json,node_api,domain_suffix,tcp_pool_json,udp_pool_json,last_seen_at
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
			nodeID, name, region, isp, tagsJSON, epsJSON, nodeAPI, domainSuffix, tcpJSON, udpJSON string
			publicInt                                                     int
			lastSeen                                                      int64
		)
		if err := rows.Scan(&nodeID, &name, &publicInt, &region, &isp, &tagsJSON, &epsJSON, &nodeAPI, &domainSuffix, &tcpJSON, &udpJSON, &lastSeen); err != nil {
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
			NodeAPI:       nodeAPI,
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

func (s *SQLiteStore) UpsertDeviceWhitelist(deviceID string, mac string, enabled bool, note string) error {
	if deviceID == "" {
		return fmt.Errorf("device_id required")
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`
INSERT INTO device_whitelist(device_id,mac,enabled,note,updated_at)
VALUES(?,?,?,?,?)
ON CONFLICT(device_id) DO UPDATE SET
  mac=excluded.mac,
  enabled=excluded.enabled,
  note=excluded.note,
  updated_at=excluded.updated_at
`, deviceID, mac, boolToInt(enabled), note, now)
	return err
}

func (s *SQLiteStore) VerifyDeviceWhitelist(deviceID string, mac string) (bool, error) {
	if deviceID == "" {
		return false, nil
	}
	row := s.db.QueryRow(`SELECT mac, enabled FROM device_whitelist WHERE device_id=?`, deviceID)
	var wantMAC string
	var enabled int
	if err := row.Scan(&wantMAC, &enabled); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	if enabled != 1 {
		return false, nil
	}
	// 只校验 device_id 是否在白名单且启用；mac 仅用于记录
	_ = wantMAC
	_ = mac
	return true, nil
}

func (s *SQLiteStore) DeleteDeviceWhitelist(deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device_id required")
	}
	_, err := s.db.Exec(`DELETE FROM device_whitelist WHERE device_id=?`, deviceID)
	return err
}

func (s *SQLiteStore) ListDeviceWhitelist(limit int, offset int) ([]DeviceWhitelistRow, int, error) {
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	if offset < 0 {
		offset = 0
	}

	row := s.db.QueryRow(`SELECT COUNT(1) FROM device_whitelist`)
	var total int
	if err := row.Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := s.db.Query(`
SELECT device_id, mac, enabled, note, updated_at
FROM device_whitelist
ORDER BY updated_at DESC
LIMIT ? OFFSET ?
`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]DeviceWhitelistRow, 0, limit)
	for rows.Next() {
		var r DeviceWhitelistRow
		var enabledInt int
		var ts int64
		if err := rows.Scan(&r.DeviceID, &r.MAC, &enabledInt, &r.Note, &ts); err != nil {
			return nil, 0, err
		}
		r.Enabled = enabledInt == 1
		r.UpdatedAt = time.Unix(ts, 0).UTC().Format(time.RFC3339)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (s *SQLiteStore) CreateDeviceSession(deviceID string, mac string, ttlSeconds int) (string, int64, error) {
	if deviceID == "" {
		return "", 0, fmt.Errorf("device_id required")
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 6 * 3600
	}
	now := time.Now().Unix()
	expires := now + int64(ttlSeconds)
	// token：sha256(deviceID|mac|ts|rand)
	seed := fmt.Sprintf("%s|%s|%d|%d", deviceID, mac, now, time.Now().UnixNano())
	tok := hashKey(seed)
	_, err := s.db.Exec(`INSERT INTO device_sessions(token,device_id,mac,expires_at,created_at) VALUES(?,?,?,?,?)`, tok, deviceID, mac, expires, now)
	if err != nil {
		return "", 0, err
	}
	return tok, expires, nil
}

func (s *SQLiteStore) VerifyDeviceSession(token string) (bool, string, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return false, "", nil
	}
	row := s.db.QueryRow(`SELECT device_id, expires_at FROM device_sessions WHERE token=?`, token)
	var deviceID string
	var exp int64
	if err := row.Scan(&deviceID, &exp); err != nil {
		if err == sql.ErrNoRows {
			return false, "", nil
		}
		return false, "", err
	}
	if time.Now().Unix() > exp {
		return false, "", nil
	}
	return true, deviceID, nil
}

func (s *SQLiteStore) UpsertOfficialNode(n OfficialNode) error {
	if n.NodeID == "" {
		return fmt.Errorf("node_id required")
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`
INSERT INTO official_nodes(node_id,name,server,token,admin_addr,admin_user,admin_pwd,node_api,domain_suffix,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  name=excluded.name,
  server=excluded.server,
  token=excluded.token,
  admin_addr=excluded.admin_addr,
  admin_user=excluded.admin_user,
  admin_pwd=excluded.admin_pwd,
  node_api=excluded.node_api,
  domain_suffix=excluded.domain_suffix,
  updated_at=excluded.updated_at
`, n.NodeID, n.Name, n.Server, n.Token, n.AdminAddr, n.AdminUser, n.AdminPwd, n.NodeAPI, n.DomainSuffix, now)
	return err
}

func (s *SQLiteStore) DeleteOfficialNode(nodeID string) error {
	if nodeID == "" {
		return fmt.Errorf("node_id required")
	}
	_, err := s.db.Exec(`DELETE FROM official_nodes WHERE node_id=?`, nodeID)
	return err
}

func (s *SQLiteStore) ListOfficialNodes() ([]OfficialNode, error) {
	rows, err := s.db.Query(`SELECT node_id,name,server,token,admin_addr,admin_user,admin_pwd,node_api,domain_suffix,updated_at FROM official_nodes ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OfficialNode, 0, 32)
	for rows.Next() {
		var n OfficialNode
		var ts int64
		if err := rows.Scan(&n.NodeID, &n.Name, &n.Server, &n.Token, &n.AdminAddr, &n.AdminUser, &n.AdminPwd, &n.NodeAPI, &n.DomainSuffix, &ts); err != nil {
			return nil, err
		}
		n.UpdatedAt = time.Unix(ts, 0).UTC().Format(time.RFC3339)
		out = append(out, n)
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


