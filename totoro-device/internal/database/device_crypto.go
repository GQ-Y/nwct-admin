package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"totoro-device/internal/cryptobox"
)

type DeviceCrypto struct {
	PrivKeyB64 string
	PubKeyB64  string
	UpdatedAt  int64
}

func GetOrCreateDeviceCrypto(db *sql.DB) (*DeviceCrypto, error) {
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	row := db.QueryRow(`SELECT priv_key_b64, pub_key_b64, updated_at FROM device_crypto WHERE id=1`)
	var s DeviceCrypto
	if err := row.Scan(&s.PrivKeyB64, &s.PubKeyB64, &s.UpdatedAt); err == nil {
		if strings.TrimSpace(s.PrivKeyB64) != "" && strings.TrimSpace(s.PubKeyB64) != "" {
			return &s, nil
		}
	}
	priv, pub, err := cryptobox.GenerateDeviceKeyPair()
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	_, err = db.Exec(`
INSERT INTO device_crypto(id, priv_key_b64, pub_key_b64, updated_at)
VALUES(1,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  priv_key_b64=excluded.priv_key_b64,
  pub_key_b64=excluded.pub_key_b64,
  updated_at=excluded.updated_at
`, strings.TrimSpace(priv), strings.TrimSpace(pub), now)
	if err != nil {
		return nil, err
	}
	return &DeviceCrypto{PrivKeyB64: priv, PubKeyB64: pub, UpdatedAt: now}, nil
}


