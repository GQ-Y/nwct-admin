package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type NodeEndpoint struct {
	Addr  string `json:"addr"`
	Port  int    `json:"port"`
	Proto string `json:"proto"`
}

type NodeConfig struct {
	NodeID       string         `json:"node_id"`
	NodeKey      string         `json:"node_key"` // 仅用于运行期；落库只存 hash
	Public       bool           `json:"public"`
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Region       string         `json:"region"`
	ISP          string         `json:"isp"`
	Tags         []string       `json:"tags"`
	BridgeURL    string         `json:"bridge_url"`
	DomainSuffix string         `json:"domain_suffix"`
	HTTPEnabled  bool           `json:"http_enabled"`
	HTTPSEnabled bool           `json:"https_enabled"`
	Endpoints    []NodeEndpoint `json:"endpoints"`
}

type Invite struct {
	InviteID  string `json:"invite_id"`
	Code      string `json:"code"`
	Revoked   bool   `json:"revoked"`
	CreatedAt string `json:"created_at"`
	ExpiresAt string `json:"expires_at"`
	MaxUses   int    `json:"max_uses"`
	Used      int    `json:"used"`
	ScopeJSON string `json:"scope_json"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	_, _ = db.Exec("PRAGMA synchronous=NORMAL;")

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS node_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  node_id TEXT NOT NULL,
  node_key_hash TEXT NOT NULL,
  node_key_plain TEXT NOT NULL DEFAULT '',
  public INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  isp TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  bridge_url TEXT NOT NULL DEFAULT '',
  domain_suffix TEXT NOT NULL DEFAULT '',
  http_enabled INTEGER NOT NULL DEFAULT 0,
  https_enabled INTEGER NOT NULL DEFAULT 0,
  endpoints_json TEXT NOT NULL DEFAULT '[]',
  admin_key TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS invites (
  invite_id TEXT PRIMARY KEY,
  code TEXT NOT NULL DEFAULT '',
  code_hash TEXT NOT NULL,
  revoked INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  expires_at INTEGER NOT NULL DEFAULT 0,
  max_uses INTEGER NOT NULL DEFAULT 0,
  used INTEGER NOT NULL DEFAULT 0,
  scope_json TEXT NOT NULL DEFAULT '{}'
);
CREATE INDEX IF NOT EXISTS idx_invites_revoked ON invites(revoked);
`
	_, err := s.db.Exec(schema)
	// 兼容旧库：补列
	_, _ = s.db.Exec(`ALTER TABLE node_config ADD COLUMN description TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE node_config ADD COLUMN node_key_plain TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE node_config ADD COLUMN http_enabled INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE node_config ADD COLUMN https_enabled INTEGER NOT NULL DEFAULT 0`)
	_, _ = s.db.Exec(`ALTER TABLE node_config ADD COLUMN admin_key TEXT NOT NULL DEFAULT ''`)
	_, _ = s.db.Exec(`ALTER TABLE invites ADD COLUMN code TEXT NOT NULL DEFAULT ''`)
	return err
}

func hashKey(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (s *Store) InitNodeIfEmpty(cfg NodeConfig, adminKey string) error {
	if cfg.NodeID == "" || cfg.NodeKey == "" {
		return fmt.Errorf("node_id/node_key required")
	}
	now := time.Now().Unix()
	tags, _ := json.Marshal(cfg.Tags)
	eps, _ := json.Marshal(cfg.Endpoints)

	// 仅当为空时插入
	_, err := s.db.Exec(`
INSERT INTO node_config(id,node_id,node_key_hash,node_key_plain,public,name,description,region,isp,tags_json,bridge_url,domain_suffix,http_enabled,https_enabled,endpoints_json,admin_key,updated_at)
VALUES(1,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING
`, cfg.NodeID, hashKey(cfg.NodeKey), cfg.NodeKey, boolToInt(cfg.Public), cfg.Name, cfg.Description, cfg.Region, cfg.ISP, string(tags), cfg.BridgeURL, cfg.DomainSuffix, boolToInt(cfg.HTTPEnabled), boolToInt(cfg.HTTPSEnabled), string(eps), strings.TrimSpace(adminKey), now)
	return err
}

func (s *Store) GetNodeConfig() (NodeConfig, string, error) {
	row := s.db.QueryRow(`
SELECT node_id,node_key_hash,node_key_plain,public,name,description,region,isp,tags_json,bridge_url,domain_suffix,http_enabled,https_enabled,endpoints_json,admin_key
FROM node_config WHERE id=1
`)
	var (
		cfg           NodeConfig
		keyHash       string
		keyPlain      string
		publicInt     int
		httpInt       int
		httpsInt      int
		tagsJSON      string
		endpointsJSON string
		adminKey      string
	)
	if err := row.Scan(&cfg.NodeID, &keyHash, &keyPlain, &publicInt, &cfg.Name, &cfg.Description, &cfg.Region, &cfg.ISP, &tagsJSON, &cfg.BridgeURL, &cfg.DomainSuffix, &httpInt, &httpsInt, &endpointsJSON, &adminKey); err != nil {
		return NodeConfig{}, "", err
	}
	cfg.NodeKey = strings.TrimSpace(keyPlain)
	cfg.Public = publicInt == 1
	cfg.HTTPEnabled = httpInt == 1
	cfg.HTTPSEnabled = httpsInt == 1
	_ = json.Unmarshal([]byte(tagsJSON), &cfg.Tags)
	_ = json.Unmarshal([]byte(endpointsJSON), &cfg.Endpoints)
	return cfg, keyHash, nil
}

func (s *Store) GetAdminKey() (string, error) {
	row := s.db.QueryRow(`SELECT admin_key FROM node_config WHERE id=1`)
	var adminKey string
	if err := row.Scan(&adminKey); err != nil {
		return "", err
	}
	return strings.TrimSpace(adminKey), nil
}

func (s *Store) UpdateAdminKey(newKey string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`UPDATE node_config SET admin_key=?, updated_at=? WHERE id=1`, strings.TrimSpace(newKey), now)
	return err
}

func (s *Store) UpdateBridgeURL(bridgeURL string) error {
	now := time.Now().Unix()
	_, err := s.db.Exec(`UPDATE node_config SET bridge_url=?, updated_at=? WHERE id=1`, strings.TrimSpace(bridgeURL), now)
	return err
}

func (s *Store) UpdateNodeConfig(adminNodeKey string, patch NodeConfig) error {
	cfg, keyHash, err := s.GetNodeConfig()
	if err != nil {
		return err
	}
	if hashKey(adminNodeKey) != keyHash {
		return fmt.Errorf("node_key invalid")
	}

	return s.updateNodeConfigUnlocked(cfg, patch)
}

// UpdateNodeConfigAsAdmin 在“已通过 X-Admin-Key 鉴权”的前提下更新配置（不要求 X-Node-Key）。
// 注意：不会修改 node_id / node_key_hash / node_key_plain。
func (s *Store) UpdateNodeConfigAsAdmin(patch NodeConfig) error {
	cfg, _, err := s.GetNodeConfig()
	if err != nil {
		return err
	}
	return s.updateNodeConfigUnlocked(cfg, patch)
}

func (s *Store) updateNodeConfigUnlocked(cfg NodeConfig, patch NodeConfig) error {
	// merge
	if patch.Name != "" {
		cfg.Name = patch.Name
	}
	if patch.Description != "" || patch.Description == "" {
		// 允许显式清空（传空字符串）
		cfg.Description = patch.Description
	}
	if patch.Region != "" {
		cfg.Region = patch.Region
	}
	if patch.ISP != "" {
		cfg.ISP = patch.ISP
	}
	if patch.Tags != nil {
		cfg.Tags = patch.Tags
	}
	// BridgeURL 不允许通过 API 修改（只读）
	// if patch.BridgeURL != "" {
	// 	cfg.BridgeURL = patch.BridgeURL
	// }
	if patch.DomainSuffix != "" {
		cfg.DomainSuffix = patch.DomainSuffix
	}
	// bool 需要显式 patch：这里要求调用方始终传（节点面板会传）
	cfg.HTTPEnabled = patch.HTTPEnabled
	cfg.HTTPSEnabled = patch.HTTPSEnabled
	if patch.Endpoints != nil {
		cfg.Endpoints = patch.Endpoints
	}
	// bool 需要显式 patch（用 Public 字段是否变化无法判断），这里要求调用者始终传
	cfg.Public = patch.Public

	now := time.Now().Unix()
	tags, _ := json.Marshal(cfg.Tags)
	eps, _ := json.Marshal(cfg.Endpoints)
	_, err := s.db.Exec(`
UPDATE node_config
SET public=?,name=?,description=?,region=?,isp=?,tags_json=?,domain_suffix=?,http_enabled=?,https_enabled=?,endpoints_json=?,updated_at=?
WHERE id=1
`, boolToInt(cfg.Public), cfg.Name, cfg.Description, cfg.Region, cfg.ISP, string(tags), cfg.DomainSuffix, boolToInt(cfg.HTTPEnabled), boolToInt(cfg.HTTPSEnabled), string(eps), now)
	return err
}

func (s *Store) CreateInvite(codeHash string, ttlSeconds int, maxUses int, scopeJSON string) (Invite, error) {
	now := time.Now().Unix()
	expiresAt := int64(0)
	if ttlSeconds > 0 {
		expiresAt = now + int64(ttlSeconds)
	}
	invID := fmt.Sprintf("inv_%d", time.Now().UnixNano())
	if scopeJSON == "" {
		scopeJSON = "{}"
	}
	_, err := s.db.Exec(`
INSERT INTO invites(invite_id,code,code_hash,revoked,created_at,expires_at,max_uses,used,scope_json)
VALUES(?,?,?,?,?,?,?,?,?)
`, invID, "", codeHash, 0, now, expiresAt, maxUses, 0, scopeJSON)
	if err != nil {
		return Invite{}, err
	}
	return Invite{
		InviteID:  invID,
		Revoked:   false,
		CreatedAt: time.Unix(now, 0).UTC().Format(time.RFC3339),
		ExpiresAt: time.Unix(expiresAt, 0).UTC().Format(time.RFC3339),
		MaxUses:   maxUses,
		Used:      0,
		ScopeJSON: scopeJSON,
	}, nil
}

// UpsertInviteFromBridge 将 bridge 返回的邀请码信息落库，便于节点侧列表管理（撤销/查看）。
func (s *Store) UpsertInviteFromBridge(inviteID string, code string, expiresAtRFC3339 string, maxUses int, scopeJSON string) error {
	inviteID = strings.TrimSpace(inviteID)
	if inviteID == "" {
		return fmt.Errorf("invite_id required")
	}
	code = strings.TrimSpace(code)
	if scopeJSON == "" {
		scopeJSON = "{}"
	}

	now := time.Now().Unix()
	expiresAt := int64(0)
	if strings.TrimSpace(expiresAtRFC3339) != "" {
		// 尝试 RFC3339；失败则尝试 unix 秒字符串
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(expiresAtRFC3339)); err == nil {
			expiresAt = t.UTC().Unix()
		} else if n, nerr := strconv.ParseInt(strings.TrimSpace(expiresAtRFC3339), 10, 64); nerr == nil {
			expiresAt = n
		}
	}

	_, err := s.db.Exec(`
INSERT INTO invites(invite_id,code,code_hash,revoked,created_at,expires_at,max_uses,used,scope_json)
VALUES(?,?,?,?,?,?,?,?,?)
ON CONFLICT(invite_id) DO UPDATE SET
  code=excluded.code,
  code_hash=excluded.code_hash,
  expires_at=excluded.expires_at,
  max_uses=excluded.max_uses,
  scope_json=excluded.scope_json,
  revoked=0
`, inviteID, code, hashKey(code), 0, now, expiresAt, maxUses, 0, scopeJSON)
	return err
}

func (s *Store) ListInvites(limit int, includeRevoked bool) ([]Invite, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	var (
		rows *sql.Rows
		err  error
	)
	if includeRevoked {
		rows, err = s.db.Query(`
SELECT invite_id, code, revoked, created_at, expires_at, max_uses, used, scope_json
FROM invites
WHERE code <> ''
ORDER BY created_at DESC
LIMIT ?
`, limit)
	} else {
		rows, err = s.db.Query(`
SELECT invite_id, code, revoked, created_at, expires_at, max_uses, used, scope_json
FROM invites
WHERE revoked=0 AND code <> ''
ORDER BY created_at DESC
LIMIT ?
`, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Invite
	for rows.Next() {
		var (
			inv       Invite
			revoked   int
			createdAt int64
			expiresAt int64
		)
		if err := rows.Scan(
			&inv.InviteID,
			&inv.Code,
			&revoked,
			&createdAt,
			&expiresAt,
			&inv.MaxUses,
			&inv.Used,
			&inv.ScopeJSON,
		); err != nil {
			return nil, err
		}
		inv.Revoked = revoked == 1
		if createdAt > 0 {
			inv.CreatedAt = time.Unix(createdAt, 0).UTC().Format(time.RFC3339)
		}
		if expiresAt > 0 {
			inv.ExpiresAt = time.Unix(expiresAt, 0).UTC().Format(time.RFC3339)
		} else {
			inv.ExpiresAt = ""
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

type InviteResolveResult struct {
	InviteID  string
	ScopeJSON string
}

// ResolveInviteByCode 校验邀请码并消耗一次使用次数（MVP/Beta 通用）。
func (s *Store) ResolveInviteByCode(code string) (InviteResolveResult, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return InviteResolveResult{}, fmt.Errorf("code required")
	}
	ch := hashKey(code)

	// 事务：校验有效性 + used+1
	tx, err := s.db.Begin()
	if err != nil {
		return InviteResolveResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	row := tx.QueryRow(`
SELECT invite_id, revoked, created_at, expires_at, max_uses, used, scope_json
FROM invites WHERE code_hash=?
`, ch)
	var (
		inviteID  string
		revoked   int
		createdAt int64
		expiresAt int64
		maxUses   int
		used      int
		scopeJSON string
	)
	if err := row.Scan(&inviteID, &revoked, &createdAt, &expiresAt, &maxUses, &used, &scopeJSON); err != nil {
		if err == sql.ErrNoRows {
			return InviteResolveResult{}, fmt.Errorf("invite_not_found")
		}
		return InviteResolveResult{}, err
	}
	if revoked == 1 {
		return InviteResolveResult{}, fmt.Errorf("invite_revoked")
	}
	now := time.Now().Unix()
	if expiresAt > 0 && now > expiresAt {
		return InviteResolveResult{}, fmt.Errorf("invite_expired")
	}
	if maxUses > 0 && used >= maxUses {
		return InviteResolveResult{}, fmt.Errorf("invite_exhausted")
	}

	_, err = tx.Exec(`UPDATE invites SET used=used+1 WHERE invite_id=?`, inviteID)
	if err != nil {
		return InviteResolveResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return InviteResolveResult{}, err
	}
	return InviteResolveResult{InviteID: inviteID, ScopeJSON: scopeJSON}, nil
}

func (s *Store) RevokeInvite(inviteID string) error {
	_, err := s.db.Exec(`UPDATE invites SET revoked=1 WHERE invite_id=?`, inviteID)
	return err
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
