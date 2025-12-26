package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
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
	row := db.QueryRow(`SELECT bridge_url, device_id, mac, device_token, expires_at, updated_at FROM bridge_session WHERE id=1`)
	var s BridgeSession
	if err := row.Scan(&s.BridgeURL, &s.DeviceID, &s.MAC, &s.DeviceToken, &s.ExpiresAt, &s.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
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
	now := time.Now().Unix()
	_, err := db.Exec(`
INSERT INTO bridge_session(id, bridge_url, device_id, mac, device_token, expires_at, updated_at)
VALUES(1,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  bridge_url=excluded.bridge_url,
  device_id=excluded.device_id,
  mac=excluded.mac,
  device_token=excluded.device_token,
  expires_at=excluded.expires_at,
  updated_at=excluded.updated_at
`, strings.TrimSpace(s.BridgeURL), strings.TrimSpace(s.DeviceID), strings.TrimSpace(s.MAC), strings.TrimSpace(s.DeviceToken), s.ExpiresAt, now)
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


