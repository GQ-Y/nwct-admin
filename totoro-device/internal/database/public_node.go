package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"totoro-device/internal/cryptobox"
)

// public_node_selected: 存储用户选择的“公开节点直连 node_id”（密文），用于重启后自动续票/重连

func EnsurePublicNodeTable(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS public_node_selected (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  node_id_enc TEXT NOT NULL DEFAULT '',
  node_id_nonce TEXT NOT NULL DEFAULT '',
  updated_at INTEGER NOT NULL DEFAULT 0
);`)
	return err
}

func SetPublicNodeID(db *sql.DB, nodeID string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node_id required")
	}
	_ = EnsurePublicNodeTable(db)
	crypto, err := GetOrCreateDeviceCrypto(db)
	if err != nil || crypto == nil || strings.TrimSpace(crypto.PrivKeyB64) == "" {
		return fmt.Errorf("设备密钥不可用")
	}
	nonce, ct, err := cryptobox.EncryptLocal(crypto.PrivKeyB64, nodeID)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = db.Exec(`
INSERT INTO public_node_selected(id, node_id_enc, node_id_nonce, updated_at)
VALUES(1,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  node_id_enc=excluded.node_id_enc,
  node_id_nonce=excluded.node_id_nonce,
  updated_at=excluded.updated_at
`, strings.TrimSpace(ct), strings.TrimSpace(nonce), now)
	return err
}

func GetPublicNodeID(db *sql.DB) (string, error) {
	if db == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	_ = EnsurePublicNodeTable(db)
	row := db.QueryRow(`SELECT node_id_enc, node_id_nonce FROM public_node_selected WHERE id=1`)
	var ct, nonce string
	if err := row.Scan(&ct, &nonce); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	if strings.TrimSpace(ct) == "" || strings.TrimSpace(nonce) == "" {
		return "", nil
	}
	crypto, err := GetOrCreateDeviceCrypto(db)
	if err != nil || crypto == nil || strings.TrimSpace(crypto.PrivKeyB64) == "" {
		return "", fmt.Errorf("设备密钥不可用")
	}
	return cryptobox.DecryptLocal(crypto.PrivKeyB64, nonce, ct)
}

func ClearPublicNodeID(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_ = EnsurePublicNodeTable(db)
	_, err := db.Exec(`DELETE FROM public_node_selected WHERE id=1`)
	return err
}


