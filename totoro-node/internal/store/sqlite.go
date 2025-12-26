package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

type NodeConfig struct {
	NodeID       string   `json:"node_id"`
	NodeKey      string   `json:"node_key"` // 仅用于运行期；落库只存 hash
	Public       bool     `json:"public"`
	Name         string   `json:"name"`
	Region       string   `json:"region"`
	ISP          string   `json:"isp"`
	Tags         []string `json:"tags"`
	BridgeURL    string   `json:"bridge_url"`
	DomainSuffix string   `json:"domain_suffix"`
	Endpoints    []any    `json:"endpoints"` // 节点侧保持透明（addr/port/proto）
}

type Invite struct {
	InviteID   string `json:"invite_id"`
	Code       string `json:"code"`
	Revoked    bool   `json:"revoked"`
	CreatedAt  string `json:"created_at"`
	ExpiresAt  string `json:"expires_at"`
	MaxUses    int    `json:"max_uses"`
	Used       int    `json:"used"`
	ScopeJSON  string `json:"scope_json"`
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
  public INTEGER NOT NULL DEFAULT 0,
  name TEXT NOT NULL DEFAULT '',
  region TEXT NOT NULL DEFAULT '',
  isp TEXT NOT NULL DEFAULT '',
  tags_json TEXT NOT NULL DEFAULT '[]',
  bridge_url TEXT NOT NULL DEFAULT '',
  domain_suffix TEXT NOT NULL DEFAULT '',
  endpoints_json TEXT NOT NULL DEFAULT '[]',
  updated_at INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS invites (
  invite_id TEXT PRIMARY KEY,
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
	return err
}

func hashKey(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func (s *Store) InitNodeIfEmpty(cfg NodeConfig) error {
	if cfg.NodeID == "" || cfg.NodeKey == "" {
		return fmt.Errorf("node_id/node_key required")
	}
	now := time.Now().Unix()
	tags, _ := json.Marshal(cfg.Tags)
	eps, _ := json.Marshal(cfg.Endpoints)

	// 仅当为空时插入
	_, err := s.db.Exec(`
INSERT INTO node_config(id,node_id,node_key_hash,public,name,region,isp,tags_json,bridge_url,domain_suffix,endpoints_json,updated_at)
VALUES(1,?,?,?,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO NOTHING
`, cfg.NodeID, hashKey(cfg.NodeKey), boolToInt(cfg.Public), cfg.Name, cfg.Region, cfg.ISP, string(tags), cfg.BridgeURL, cfg.DomainSuffix, string(eps), now)
	return err
}

func (s *Store) GetNodeConfig() (NodeConfig, string, error) {
	row := s.db.QueryRow(`
SELECT node_id,node_key_hash,public,name,region,isp,tags_json,bridge_url,domain_suffix,endpoints_json
FROM node_config WHERE id=1
`)
	var (
		cfg          NodeConfig
		keyHash      string
		publicInt    int
		tagsJSON     string
		endpointsJSON string
	)
	if err := row.Scan(&cfg.NodeID, &keyHash, &publicInt, &cfg.Name, &cfg.Region, &cfg.ISP, &tagsJSON, &cfg.BridgeURL, &cfg.DomainSuffix, &endpointsJSON); err != nil {
		return NodeConfig{}, "", err
	}
	cfg.Public = publicInt == 1
	_ = json.Unmarshal([]byte(tagsJSON), &cfg.Tags)
	_ = json.Unmarshal([]byte(endpointsJSON), &cfg.Endpoints)
	return cfg, keyHash, nil
}

func (s *Store) UpdateNodeConfig(adminNodeKey string, patch NodeConfig) error {
	cfg, keyHash, err := s.GetNodeConfig()
	if err != nil {
		return err
	}
	if hashKey(adminNodeKey) != keyHash {
		return fmt.Errorf("node_key invalid")
	}

	// merge
	if patch.Name != "" {
		cfg.Name = patch.Name
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
	if patch.BridgeURL != "" {
		cfg.BridgeURL = patch.BridgeURL
	}
	if patch.DomainSuffix != "" {
		cfg.DomainSuffix = patch.DomainSuffix
	}
	if patch.Endpoints != nil {
		cfg.Endpoints = patch.Endpoints
	}
	// bool 需要显式 patch（用 Public 字段是否变化无法判断），这里要求调用者始终传
	cfg.Public = patch.Public

	now := time.Now().Unix()
	tags, _ := json.Marshal(cfg.Tags)
	eps, _ := json.Marshal(cfg.Endpoints)
	_, err = s.db.Exec(`
UPDATE node_config
SET public=?,name=?,region=?,isp=?,tags_json=?,bridge_url=?,domain_suffix=?,endpoints_json=?,updated_at=?
WHERE id=1
`, boolToInt(cfg.Public), cfg.Name, cfg.Region, cfg.ISP, string(tags), cfg.BridgeURL, cfg.DomainSuffix, string(eps), now)
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
INSERT INTO invites(invite_id,code_hash,revoked,created_at,expires_at,max_uses,used,scope_json)
VALUES(?,?,?,?,?,?,?,?)
`, invID, codeHash, 0, now, expiresAt, maxUses, 0, scopeJSON)
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


