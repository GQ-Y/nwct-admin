package cryptobox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
)

// EncryptedPayload 应用层加密载荷（桥梁 -> 设备）。
// 算法：X25519(ephemeral) + HKDF-SHA256 + AES-256-GCM
type EncryptedPayload struct {
	Alg          string `json:"alg"`
	EphemeralPub string `json:"ephemeral_pub"` // base64(raw 32)
	Nonce        string `json:"nonce"`         // base64(12)
	Ciphertext   string `json:"ciphertext"`    // base64
}

const algV1 = "x25519-hkdf-sha256-aes256gcm-v1"

func EncryptForDevice(devicePubKeyB64 string, deviceID string, plaintext []byte) (*EncryptedPayload, error) {
	devicePubKeyB64 = strings.TrimSpace(devicePubKeyB64)
	deviceID = strings.TrimSpace(deviceID)
	if devicePubKeyB64 == "" {
		return nil, fmt.Errorf("device pub_key missing")
	}
	if deviceID == "" {
		return nil, fmt.Errorf("device_id required")
	}
	pubRaw, err := base64.StdEncoding.DecodeString(devicePubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid pub_key")
	}
	curve := ecdh.X25519()
	pub, err := curve.NewPublicKey(pubRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid pub_key")
	}
	ephemeralPriv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	shared, err := ephemeralPriv.ECDH(pub)
	if err != nil {
		return nil, err
	}
	key, err := hkdf.Key(sha256.New, shared, []byte("totoro-bridge:"+deviceID), "resp:"+algV1, 32)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, plaintext, []byte(deviceID))
	return &EncryptedPayload{
		Alg:          algV1,
		EphemeralPub: base64.StdEncoding.EncodeToString(ephemeralPriv.PublicKey().Bytes()),
		Nonce:        base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:   base64.StdEncoding.EncodeToString(ct),
	}, nil
}


