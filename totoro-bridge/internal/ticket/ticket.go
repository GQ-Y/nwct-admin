package ticket

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"encoding/base64"
	"fmt"
	"strings"
	"time"
)

type Claims struct {
	NodeID   string          `json:"node_id"`
	InviteID string          `json:"invite_id"`
	Scope    json.RawMessage `json:"scope"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
	Iss string `json:"iss"`
}

func IssueHMAC(nodeID string, inviteID string, scopeJSON string, key []byte, ttl time.Duration) (string, time.Time, error) {
	if nodeID == "" || inviteID == "" {
		return "", time.Time{}, fmt.Errorf("node_id/invite_id required")
	}
	if len(key) == 0 {
		return "", time.Time{}, fmt.Errorf("empty signing key")
	}
	exp := time.Now().Add(ttl).UTC()
	scope := json.RawMessage("null")
	if scopeJSON != "" {
		scope = json.RawMessage([]byte(scopeJSON))
	}
	claims := Claims{
		NodeID:   nodeID,
		InviteID: inviteID,
		Scope:    scope,
		Exp: exp.Unix(),
		Iat: time.Now().UTC().Unix(),
		Iss: "totoro-bridge",
	}
	headerJSON := `{"alg":"HS256","typ":"JWT"}`
	hb64 := b64url([]byte(headerJSON))
	pb, _ := json.Marshal(claims)
	pb64 := b64url(pb)
	signingInput := hb64 + "." + pb64
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(signingInput))
	sig := mac.Sum(nil)
	sb64 := b64url(sig)
	return signingInput + "." + sb64, exp, nil
}

func b64url(b []byte) string {
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}


