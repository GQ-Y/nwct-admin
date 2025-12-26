package ticket

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	NodeID   string          `json:"node_id"`
	InviteID string          `json:"invite_id"`
	Scope    json.RawMessage `json:"scope"`
	jwt.RegisteredClaims
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
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(exp),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
			Issuer:    "totoro-node",
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := t.SignedString(key)
	if err != nil {
		return "", time.Time{}, err
	}
	return s, exp, nil
}

func VerifyHMAC(token string, key []byte) (*Claims, error) {
	if token == "" {
		return nil, fmt.Errorf("empty token")
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return key, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}


