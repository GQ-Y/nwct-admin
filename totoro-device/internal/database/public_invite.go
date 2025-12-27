package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"totoro-device/internal/cryptobox"
)

func SetPublicInviteCode(db *sql.DB, inviteCode string) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	crypto, err := GetOrCreateDeviceCrypto(db)
	if err != nil || crypto == nil || strings.TrimSpace(crypto.PrivKeyB64) == "" {
		return fmt.Errorf("设备密钥不可用")
	}
	nonce, ct, err := cryptobox.EncryptLocal(crypto.PrivKeyB64, strings.TrimSpace(inviteCode))
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	_, err = db.Exec(`
INSERT INTO public_invite(id, invite_code_enc, invite_code_nonce, updated_at)
VALUES(1,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  invite_code_enc=excluded.invite_code_enc,
  invite_code_nonce=excluded.invite_code_nonce,
  updated_at=excluded.updated_at
`, strings.TrimSpace(ct), strings.TrimSpace(nonce), now)
	return err
}

func GetPublicInviteCode(db *sql.DB) (string, error) {
	if db == nil {
		return "", fmt.Errorf("数据库未初始化")
	}
	row := db.QueryRow(`SELECT invite_code_enc, invite_code_nonce FROM public_invite WHERE id=1`)
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

func ClearPublicInviteCode(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}
	_, err := db.Exec(`DELETE FROM public_invite WHERE id=1`)
	return err
}


