package main

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"strings"
)

func signRS256(key *rsa.PrivateKey, kid string, claims map[string]any) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signing := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func sha256Sum(value string) string {
	sum := sha256.Sum256([]byte(value))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func verifyRS256(token string, lookup func(kid string) (*rsa.PublicKey, error)) (map[string]json.RawMessage, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	header, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var h struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if json.Unmarshal(header, &h) != nil || h.Alg != "RS256" || h.Kid == "" {
		return nil, errors.New("unsupported token")
	}
	pub, err := lookup(h.Kid)
	if err != nil {
		return nil, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig) != nil {
		return nil, errors.New("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]json.RawMessage
	if json.Unmarshal(payload, &claims) != nil {
		return nil, errors.New("bad claims")
	}
	return claims, nil
}

func (s *server) publicJWK(kid string) map[string]string {
	pub := s.key.PublicKey
	return map[string]string{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": kid,
		"n": base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
		"e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
	}
}

func rsaFromModExp(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := decodeB64(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := decodeB64(eB64)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e < 2 || len(nBytes) == 0 {
		return nil, errors.New("bad jwk")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func decodeB64(s string) ([]byte, error) {
	if b, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func claimString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

func claimInt(raw json.RawMessage) int64 {
	var n int64
	if json.Unmarshal(raw, &n) != nil {
		return 0
	}
	return n
}

func audienceMatches(raw json.RawMessage, clientID string) bool {
	if claimString(raw) == clientID {
		return true
	}
	var list []string
	if json.Unmarshal(raw, &list) != nil {
		return false
	}
	for _, item := range list {
		if item == clientID {
			return true
		}
	}
	return false
}
