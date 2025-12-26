package store

import (
	"crypto/rand"
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
	GetNodeKeyPlain(nodeID string) (string, error)
	UpsertNodeHeartbeat(hb NodeHeartbeat) error
	ListPublicNodes() ([]PublicNode, error)
	GetPublicNodeByID(nodeID string) (PublicNode, error)
	// invites
	CreateInvite(nodeID string, scopeJSON string, ttlSeconds int, maxUses int) (code string, inviteID string, expiresAt int64, err error)
	RevokeInvite(nodeID string, code string) error
	RevokeInviteByID(nodeID string, inviteID string) error
	PreviewInvite(deviceID string, code string) (node PublicNode, inviteID string, expiresAt int64, err error)
	RedeemInvite(deviceID string, code string) (node PublicNode, inviteID string, scopeJSON string, ticketTTLSeconds int, err error)
	// device auth
	UpsertDeviceWhitelist(deviceID string, mac string, enabled bool, note string) error
	DeleteDeviceWhitelist(deviceID string) error
	ListDeviceWhitelist(limit int, offset int) ([]DeviceWhitelistRow, int, error)
	VerifyDeviceWhitelist(deviceID string, mac string) (bool, error)
	UpsertDevicePubKey(deviceID string, pubKeyB64 string) error
	GetDevicePubKey(deviceID string) (string, error)
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

type InviteRow struct {
	InviteID  string
	NodeID    string
	CodeHash  string
	CodeTail  string
	ScopeJSON string
	ExpiresAt int64
	MaxUses   int
	Used      int
	Revoked   int
	CreatedAt int64
	UpdatedAt int64
}

type PublicNode struct {
	NodeID        string            `json:"node_id"`
	Name          string            `json:"name"`
	Description   string            `json:"description,omitempty"`
	Public        bool              `json:"public"`
	Status        string            `json:"status"`
	Region        string            `json:"region"`
	ISP           string            `json:"isp"`
	Tags          []string          `json:"tags"`
	Endpoints     []NodeEndpoint    `json:"endpoints"`
	NodeAPI       string            `json:"node_api,omitempty"`
	DomainSuffix  string            `json:"domain_suffix"`
	HTTPEnabled   bool              `json:"http_enabled"`
	HTTPSEnabled  bool              `json:"https_enabled"`
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
	Description  string
	Region       string
	ISP          string
	Tags         []string
	Endpoints    []NodeEndpoint
	NodeAPI      string
	DomainSuffix string
	HTTPEnabled  bool
	HTTPSEnabled bool
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
	HTTPEnabled  bool   `json:"http_enabled"`
	HTTPSEnabled bool   `json:"https_enabled"`
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
  node_key_plain TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS nodes (
  node_id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  public INTEGER NOT NULL DEFAULT 0,
  region TEXT NOT NULL DEFAULT '',
  isp TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  endpoints_json TEXT NOT NULL DEFAULT '[]',
  node_api TEXT NOT NULL DEFAULT '',
  domain_suffix TEXT NOT NULL DEFAULT '',
  http_enabled INTEGER NOT NULL DEFAULT 0,
  https_enabled INTEGER NOT NULL DEFAULT 0,
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
  pub_key TEXT NOT NULL DEFAULT '',
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
  http_enabled INTEGER NOT NULL DEFAULT 0,
  https_enabled INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS invites (
  invite_id TEXT PRIMARY KEY,
  node_id TEXT NOT NULL,
  code_hash TEXT NOT NULL UNIQUE,
  code_tail TEXT NOT NULL DEFAULT '',
  scope_json TEXT NOT NULL DEFAULT 'null',
  expires_at INTEGER NOT NULL DEFAULT 0,
  max_uses INTEGER NOT NULL DEFAULT 1,
  used INTEGER NOT NULL DEFAULT 0,
  revoked INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_invites_node ON invites(node_id, updated_at);
`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	// 兼容旧库：补列（忽略已存在错误）
	_ = s.tryAddColumn("node_auth", "node_key_plain", "TEXT NOT NULL DEFAULT ''")
	_ = s.tryAddColumn("device_whitelist", "pub_key", "TEXT NOT NULL DEFAULT ''")
	// nodes 表是早期版本，字段演进较多；这里做一次“宽松补列”
	_ = s.tryAddColumn("nodes", "description", "TEXT NOT NULL DEFAULT ''")
	_ = s.tryAddColumn("nodes", "node_api", "TEXT NOT NULL DEFAULT ''")
	_ = s.tryAddColumn("nodes", "http_enabled", "INTEGER NOT NULL DEFAULT 0")
	_ = s.tryAddColumn("nodes", "https_enabled", "INTEGER NOT NULL DEFAULT 0")
	_ = s.tryAddColumn("nodes", "tcp_pool_json", "TEXT NOT NULL DEFAULT 'null'")
	_ = s.tryAddColumn("nodes", "udp_pool_json", "TEXT NOT NULL DEFAULT 'null'")
	_ = s.tryAddColumn("nodes", "version_json", "TEXT NOT NULL DEFAULT 'null'")
	_ = s.tryAddColumn("nodes", "metrics_json", "TEXT NOT NULL DEFAULT 'null'")
	_ = s.tryAddColumn("nodes", "extra_json", "TEXT NOT NULL DEFAULT 'null'")
	_ = s.tryAddColumn("official_nodes", "http_enabled", "INTEGER NOT NULL DEFAULT 0")
	_ = s.tryAddColumn("official_nodes", "https_enabled", "INTEGER NOT NULL DEFAULT 0")
	return nil
}

func (s *SQLiteStore) tryAddColumn(table, col, colType string) error {
	_, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, col, colType))
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return nil
	}
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
INSERT INTO node_auth(node_id,node_key_hash,node_key_plain,updated_at)
VALUES(?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  node_key_hash=excluded.node_key_hash,
  node_key_plain=excluded.node_key_plain,
  updated_at=excluded.updated_at
`, nodeID, hashKey(nodeKey), nodeKey, now)
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

func (s *SQLiteStore) GetNodeKeyPlain(nodeID string) (string, error) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return "", fmt.Errorf("node_id required")
	}
	row := s.db.QueryRow(`SELECT node_key_plain FROM node_auth WHERE node_id=?`, nodeID)
	var v string
	if err := row.Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("node not found")
		}
		return "", err
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("node_key_plain empty (re-upsert node auth)")
	}
	return v, nil
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
INSERT INTO nodes(node_id,name,description,public,region,isp,tags_json,endpoints_json,node_api,domain_suffix,http_enabled,https_enabled,tcp_pool_json,udp_pool_json,version_json,metrics_json,extra_json,last_seen_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  name=excluded.name,
  description=excluded.description,
  public=excluded.public,
  region=excluded.region,
  isp=excluded.isp,
  tags_json=excluded.tags_json,
  endpoints_json=excluded.endpoints_json,
  node_api=excluded.node_api,
  domain_suffix=excluded.domain_suffix,
  http_enabled=excluded.http_enabled,
  https_enabled=excluded.https_enabled,
  tcp_pool_json=excluded.tcp_pool_json,
  udp_pool_json=excluded.udp_pool_json,
  version_json=excluded.version_json,
  metrics_json=excluded.metrics_json,
  extra_json=excluded.extra_json,
  last_seen_at=excluded.last_seen_at
`, hb.NodeID, hb.Name, hb.Description, boolToInt(hb.Public), hb.Region, hb.ISP, string(tags), string(eps), hb.NodeAPI, hb.DomainSuffix, boolToInt(hb.HTTPEnabled), boolToInt(hb.HTTPSEnabled), string(tcp), string(udp), string(hb.VersionJSON), string(hb.MetricsJSON), string(hb.ExtraJSON), now)
	return err
}

func (s *SQLiteStore) ListPublicNodes() ([]PublicNode, error) {
	rows, err := s.db.Query(`
SELECT node_id,name,description,public,region,isp,tags_json,endpoints_json,node_api,domain_suffix,http_enabled,https_enabled,tcp_pool_json,udp_pool_json,last_seen_at
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
			nodeID, name, desc, region, isp, tagsJSON, epsJSON, nodeAPI, domainSuffix, tcpJSON, udpJSON string
			publicInt, httpInt, httpsInt                                   int
			lastSeen                                                      int64
		)
		if err := rows.Scan(&nodeID, &name, &desc, &publicInt, &region, &isp, &tagsJSON, &epsJSON, &nodeAPI, &domainSuffix, &httpInt, &httpsInt, &tcpJSON, &udpJSON, &lastSeen); err != nil {
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
			Description:   desc,
			Public:        publicInt == 1,
			Status:        status,
			Region:        region,
			ISP:           isp,
			Tags:          tags,
			Endpoints:     eps,
			NodeAPI:       nodeAPI,
			DomainSuffix:  domainSuffix,
			HTTPEnabled:   httpInt == 1,
			HTTPSEnabled:  httpsInt == 1,
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

func (s *SQLiteStore) GetPublicNodeByID(nodeID string) (PublicNode, error) {
	n, err := s.getNodeByID(nodeID)
	if err != nil {
		return PublicNode{}, err
	}
	if !n.Public {
		return PublicNode{}, fmt.Errorf("not_public")
	}
	return n, nil
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

func (s *SQLiteStore) UpsertDevicePubKey(deviceID string, pubKeyB64 string) error {
	deviceID = strings.TrimSpace(deviceID)
	pubKeyB64 = strings.TrimSpace(pubKeyB64)
	if deviceID == "" {
		return fmt.Errorf("device_id required")
	}
	now := time.Now().Unix()
	_, err := s.db.Exec(`
UPDATE device_whitelist
SET pub_key=?, updated_at=?
WHERE device_id=?
`, pubKeyB64, now, deviceID)
	return err
}

func (s *SQLiteStore) GetDevicePubKey(deviceID string) (string, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", fmt.Errorf("device_id required")
	}
	row := s.db.QueryRow(`SELECT pub_key FROM device_whitelist WHERE device_id=?`, deviceID)
	var v string
	if err := row.Scan(&v); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("device not found")
		}
		return "", err
	}
	return strings.TrimSpace(v), nil
}

func (s *SQLiteStore) CreateInvite(nodeID string, scopeJSON string, ttlSeconds int, maxUses int) (string, string, int64, error) {
	nodeID = strings.TrimSpace(nodeID)
	scopeJSON = strings.TrimSpace(scopeJSON)
	if nodeID == "" {
		return "", "", 0, fmt.Errorf("node_id required")
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}
	if maxUses <= 0 {
		maxUses = 1
	}
	code := randomCode(10)
	inviteID := randomCode(16)
	codeHash := hashKey(code)
	codeTail := ""
	if len(code) >= 4 {
		codeTail = code[len(code)-4:]
	}
	now := time.Now().Unix()
	exp := now + int64(ttlSeconds)
	if scopeJSON == "" {
		scopeJSON = "null"
	}
	_, err := s.db.Exec(`
INSERT INTO invites(invite_id,node_id,code_hash,code_tail,scope_json,expires_at,max_uses,used,revoked,created_at,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?)
`, inviteID, nodeID, codeHash, codeTail, scopeJSON, exp, maxUses, 0, 0, now, now)
	if err != nil {
		return "", "", 0, err
	}
	return code, inviteID, exp, nil
}

func (s *SQLiteStore) RevokeInvite(nodeID string, code string) error {
	nodeID = strings.TrimSpace(nodeID)
	code = strings.TrimSpace(code)
	if nodeID == "" || code == "" {
		return fmt.Errorf("node_id/code required")
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(`
UPDATE invites
SET revoked=1, updated_at=?
WHERE node_id=? AND code_hash=?
`, now, nodeID, hashKey(code))
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (s *SQLiteStore) RevokeInviteByID(nodeID string, inviteID string) error {
	nodeID = strings.TrimSpace(nodeID)
	inviteID = strings.TrimSpace(inviteID)
	if nodeID == "" || inviteID == "" {
		return fmt.Errorf("node_id/invite_id required")
	}
	now := time.Now().Unix()
	res, err := s.db.Exec(`
UPDATE invites
SET revoked=1, updated_at=?
WHERE node_id=? AND invite_id=?
`, now, nodeID, inviteID)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

func (s *SQLiteStore) RedeemInvite(deviceID string, code string) (PublicNode, string, string, int, error) {
	_ = strings.TrimSpace(deviceID) // currently not used for policy; reserved for future
	code = strings.TrimSpace(code)
	if code == "" {
		return PublicNode{}, "", "", 0, fmt.Errorf("code required")
	}
	now := time.Now().Unix()
	// transactional: check + increment used
	tx, err := s.db.Begin()
	if err != nil {
		return PublicNode{}, "", "", 0, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRow(`
SELECT invite_id,node_id,scope_json,expires_at,max_uses,used,revoked
FROM invites
WHERE code_hash=?
`, hashKey(code))
	var inviteID, nodeID, scopeJSON string
	var expiresAt int64
	var maxUses, used, revoked int
	if err := row.Scan(&inviteID, &nodeID, &scopeJSON, &expiresAt, &maxUses, &used, &revoked); err != nil {
		if err == sql.ErrNoRows {
			return PublicNode{}, "", "", 0, fmt.Errorf("invalid_code")
		}
		return PublicNode{}, "", "", 0, err
	}
	if revoked == 1 {
		return PublicNode{}, "", "", 0, fmt.Errorf("revoked")
	}
	if expiresAt > 0 && now >= expiresAt {
		return PublicNode{}, "", "", 0, fmt.Errorf("expired")
	}
	if maxUses > 0 && used >= maxUses {
		return PublicNode{}, "", "", 0, fmt.Errorf("exhausted")
	}
	_, err = tx.Exec(`UPDATE invites SET used=used+1, updated_at=? WHERE invite_id=?`, now, inviteID)
	if err != nil {
		return PublicNode{}, "", "", 0, err
	}
	if err := tx.Commit(); err != nil {
		return PublicNode{}, "", "", 0, err
	}

	// load node info from nodes table (can be non-public; invite implies sharing)
	n, err := s.getNodeByID(nodeID)
	if err != nil {
		return PublicNode{}, "", "", 0, err
	}
	// ticket ttl: min(30m, invite剩余)
	ttl := 30 * 60
	if expiresAt > 0 {
		remain := int(expiresAt - now)
		if remain < ttl {
			ttl = remain
		}
	}
	if ttl <= 10 {
		ttl = 10
	}
	return n, inviteID, scopeJSON, ttl, nil
}

func (s *SQLiteStore) PreviewInvite(deviceID string, code string) (PublicNode, string, int64, error) {
	_ = strings.TrimSpace(deviceID) // reserved
	code = strings.TrimSpace(code)
	if code == "" {
		return PublicNode{}, "", 0, fmt.Errorf("code required")
	}
	now := time.Now().Unix()
	row := s.db.QueryRow(`
SELECT invite_id,node_id,expires_at,max_uses,used,revoked
FROM invites
WHERE code_hash=?
`, hashKey(code))
	var inviteID, nodeID string
	var expiresAt int64
	var maxUses, used, revoked int
	if err := row.Scan(&inviteID, &nodeID, &expiresAt, &maxUses, &used, &revoked); err != nil {
		if err == sql.ErrNoRows {
			return PublicNode{}, "", 0, fmt.Errorf("invalid_code")
		}
		return PublicNode{}, "", 0, err
	}
	if revoked == 1 {
		return PublicNode{}, "", 0, fmt.Errorf("revoked")
	}
	if expiresAt > 0 && now >= expiresAt {
		return PublicNode{}, "", 0, fmt.Errorf("expired")
	}
	if maxUses > 0 && used >= maxUses {
		return PublicNode{}, "", 0, fmt.Errorf("exhausted")
	}
	n, err := s.getNodeByID(nodeID)
	if err != nil {
		return PublicNode{}, "", 0, err
	}
	return n, inviteID, expiresAt, nil
}

func (s *SQLiteStore) getNodeByID(nodeID string) (PublicNode, error) {
	nodeID = strings.TrimSpace(nodeID)
	row := s.db.QueryRow(`
SELECT node_id,name,public,region,isp,tags_json,endpoints_json,node_api,domain_suffix,http_enabled,https_enabled,tcp_pool_json,udp_pool_json,last_seen_at
FROM nodes WHERE node_id=?
`, nodeID)
	var (
		id, name, region, isp, tagsJSON, epsJSON, nodeAPI, domainSuffix, tcpJSON, udpJSON string
		publicInt, httpInt, httpsInt                                  int
		lastSeen                                                      int64
	)
	if err := row.Scan(&id, &name, &publicInt, &region, &isp, &tagsJSON, &epsJSON, &nodeAPI, &domainSuffix, &httpInt, &httpsInt, &tcpJSON, &udpJSON, &lastSeen); err != nil {
		if err == sql.ErrNoRows {
			return PublicNode{}, fmt.Errorf("node_offline_or_unknown")
		}
		return PublicNode{}, err
	}
	var tags []string
	var eps []NodeEndpoint
	var tcp *PortPool
	var udp *PortPool
	_ = json.Unmarshal([]byte(tagsJSON), &tags)
	_ = json.Unmarshal([]byte(epsJSON), &eps)
	_ = json.Unmarshal([]byte(tcpJSON), &tcp)
	_ = json.Unmarshal([]byte(udpJSON), &udp)
	now := time.Now().Unix()
	age := now - lastSeen
	status := "online"
	if age > 90 {
		status = "offline"
	} else if age > 30 {
		status = "degraded"
	}
	return PublicNode{
		NodeID:        id,
		Name:          name,
		Public:        publicInt == 1,
		Status:        status,
		Region:        region,
		ISP:           isp,
		Tags:          tags,
		Endpoints:     eps,
		NodeAPI:       nodeAPI,
		DomainSuffix:  domainSuffix,
		HTTPEnabled:   httpInt == 1,
		HTTPSEnabled:  httpsInt == 1,
		TCPPortPool:   tcp,
		UDPPortPool:   udp,
		UpdatedAt:     time.Unix(lastSeen, 0).UTC().Format(time.RFC3339),
		HeartbeatAgeS: age,
	}, nil
}

func randomCode(n int) string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	if n <= 0 {
		n = 8
	}
	b := make([]byte, n)
	// best-effort: fallback to time-based if rand fails
	if _, err := rand.Read(b); err == nil {
		for i := 0; i < n; i++ {
			b[i] = alphabet[int(b[i])%len(alphabet)]
		}
		return string(b)
	}
	seed := fmt.Sprintf("%d", time.Now().UnixNano())
	out := make([]byte, 0, n)
	for len(out) < n {
		sum := sha256.Sum256([]byte(seed))
		for _, x := range sum[:] {
			out = append(out, alphabet[int(x)%len(alphabet)])
			if len(out) >= n {
				break
			}
		}
		seed = string(out)
	}
	return string(out[:n])
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
INSERT INTO official_nodes(node_id,name,server,token,admin_addr,admin_user,admin_pwd,node_api,domain_suffix,http_enabled,https_enabled,updated_at)
VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(node_id) DO UPDATE SET
  name=excluded.name,
  server=excluded.server,
  token=excluded.token,
  admin_addr=excluded.admin_addr,
  admin_user=excluded.admin_user,
  admin_pwd=excluded.admin_pwd,
  node_api=excluded.node_api,
  domain_suffix=excluded.domain_suffix,
  http_enabled=excluded.http_enabled,
  https_enabled=excluded.https_enabled,
  updated_at=excluded.updated_at
`, n.NodeID, n.Name, n.Server, n.Token, n.AdminAddr, n.AdminUser, n.AdminPwd, n.NodeAPI, n.DomainSuffix, boolToInt(n.HTTPEnabled), boolToInt(n.HTTPSEnabled), now)
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
	rows, err := s.db.Query(`SELECT node_id,name,server,token,admin_addr,admin_user,admin_pwd,node_api,domain_suffix,http_enabled,https_enabled,updated_at FROM official_nodes ORDER BY updated_at DESC LIMIT 200`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]OfficialNode, 0, 32)
	for rows.Next() {
		var n OfficialNode
		var ts int64
		var httpInt, httpsInt int
		if err := rows.Scan(&n.NodeID, &n.Name, &n.Server, &n.Token, &n.AdminAddr, &n.AdminUser, &n.AdminPwd, &n.NodeAPI, &n.DomainSuffix, &httpInt, &httpsInt, &ts); err != nil {
			return nil, err
		}
		n.HTTPEnabled = httpInt == 1
		n.HTTPSEnabled = httpsInt == 1
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


