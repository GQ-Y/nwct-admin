package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"totoro-device/internal/cryptobox"
)

type BridgeSession struct {
	BridgeURL   string
	DeviceID    string
	MAC         string
	DeviceToken string
	ExpiresAt   int64
	UpdatedAt   int64
}

func GetBridgeSession(db *sql.DB) (*BridgeSession, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	row := db.QueryRow(`SELECT bridge_url, device_id, mac, device_token, device_token_enc, device_token_nonce, expires_at, updated_at FROM bridge_session WHERE id=1`)
	var s BridgeSession
	var legacyToken string
	var tokEnc, tokNonce string
	if err := row.Scan(&s.BridgeURL, &s.DeviceID, &s.MAC, &legacyToken, &tokEnc, &tokNonce, &s.ExpiresAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	// 优先新列（密文存储）
	if strings.TrimSpace(tokEnc) != "" && strings.TrimSpace(tokNonce) != "" {
		crypto, err := GetOrCreateDeviceCrypto(db)
		if err == nil && crypto != nil && strings.TrimSpace(crypto.PrivKeyB64) != "" {
			if pt, derr := cryptobox.DecryptLocal(crypto.PrivKeyB64, tokNonce, tokEnc); derr == nil {
				s.DeviceToken = strings.TrimSpace(pt)
			}
		}
	} else {
		s.DeviceToken = strings.TrimSpace(legacyToken)
	}
	if strings.TrimSpace(s.DeviceToken) == "" {
		return nil, nil
	}
	return &s, nil
}

func UpsertBridgeSession(db *sql.DB, s BridgeSession) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	crypto, err := GetOrCreateDeviceCrypto(db)
	if err != nil || crypto == nil || strings.TrimSpace(crypto.PrivKeyB64) == "" {
		return fmt.Errorf("设备密钥不可用")
	}
	nonce, ct, err := cryptobox.EncryptLocal(crypto.PrivKeyB64, strings.TrimSpace(s.DeviceToken))
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = db.Exec(`
INSERT INTO bridge_session(id, bridge_url, device_id, mac, device_token, device_token_enc, device_token_nonce, expires_at, updated_at)
VALUES(1,?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  bridge_url=excluded.bridge_url,
  device_id=excluded.device_id,
  mac=excluded.mac,
  device_token=excluded.device_token,
  device_token_enc=excluded.device_token_enc,
  device_token_nonce=excluded.device_token_nonce,
  expires_at=excluded.expires_at,
  updated_at=excluded.updated_at
`, strings.TrimSpace(s.BridgeURL), strings.TrimSpace(s.DeviceID), strings.TrimSpace(s.MAC),
		"", strings.TrimSpace(ct), strings.TrimSpace(nonce),
		s.ExpiresAt, now)
	return err
}

func ClearBridgeSession(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := db.Exec(`DELETE FROM bridge_session WHERE id=1`)
	return err
}

func BridgeSessionExpired(s *BridgeSession, skew time.Duration) bool {
	if s == nil || strings.TrimSpace(s.DeviceToken) == "" {
		return true
	}
	if s.ExpiresAt <= 0 {
		return true
	}
	return time.Now().Add(skew).Unix() >= s.ExpiresAt
}


