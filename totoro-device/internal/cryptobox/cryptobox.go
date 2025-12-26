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

// EncryptedPayload 与 bridge 返回一致
type EncryptedPayload struct {
	Alg          string `json:"alg"`
	EphemeralPub string `json:"ephemeral_pub"`
	Nonce        string `json:"nonce"`
	Ciphertext   string `json:"ciphertext"`
}

const algV1 = "x25519-hkdf-sha256-aes256gcm-v1"

func GenerateDeviceKeyPair() (privB64 string, pubB64 string, err error) {
	curve := ecdh.X25519()
	priv, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(priv.Bytes()), base64.StdEncoding.EncodeToString(priv.PublicKey().Bytes()), nil
}

func DecryptFromBridge(devicePrivKeyB64 string, deviceID string, enc EncryptedPayload) ([]byte, error) {
	devicePrivKeyB64 = strings.TrimSpace(devicePrivKeyB64)
	deviceID = strings.TrimSpace(deviceID)
	if devicePrivKeyB64 == "" || deviceID == "" {
		return nil, fmt.Errorf("missing device key/id")
	}
	if strings.TrimSpace(enc.Alg) != algV1 {
		return nil, fmt.Errorf("unsupported alg")
	}
	privRaw, err := base64.StdEncoding.DecodeString(devicePrivKeyB64)
	if err != nil {
		return nil, fmt.Errorf("invalid priv_key")
	}
	epubRaw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc.EphemeralPub))
	if err != nil {
		return nil, fmt.Errorf("invalid ephemeral_pub")
	}
	nonce, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc.Nonce))
	if err != nil {
		return nil, fmt.Errorf("invalid nonce")
	}
	ct, err := base64.StdEncoding.DecodeString(strings.TrimSpace(enc.Ciphertext))
	if err != nil {
		return nil, fmt.Errorf("invalid ciphertext")
	}

	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(privRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid priv_key")
	}
	epub, err := curve.NewPublicKey(epubRaw)
	if err != nil {
		return nil, fmt.Errorf("invalid ephemeral_pub")
	}
	shared, err := priv.ECDH(epub)
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
	pt, err := gcm.Open(nil, nonce, ct, []byte(deviceID))
	if err != nil {
		return nil, fmt.Errorf("decrypt failed")
	}
	return pt, nil
}

func localKeyFromPrivB64(devicePrivKeyB64 string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(devicePrivKeyB64))
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(append(raw, []byte("totoro-device-local-v1")...))
	k := make([]byte, 32)
	copy(k, sum[:])
	return k, nil
}

func EncryptLocal(devicePrivKeyB64 string, plaintext string) (nonceB64 string, ctB64 string, err error) {
	key, err := localKeyFromPrivB64(devicePrivKeyB64)
	if err != nil {
		return "", "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", "", err
	}
	ct := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(nonce), base64.StdEncoding.EncodeToString(ct), nil
}

func DecryptLocal(devicePrivKeyB64 string, nonceB64 string, ctB64 string) (string, error) {
	key, err := localKeyFromPrivB64(devicePrivKeyB64)
	if err != nil {
		return "", err
	}
	nonce, err := base64.StdEncoding.DecodeString(strings.TrimSpace(nonceB64))
	if err != nil {
		return "", err
	}
	ct, err := base64.StdEncoding.DecodeString(strings.TrimSpace(ctB64))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed")
	}
	return string(pt), nil
}


