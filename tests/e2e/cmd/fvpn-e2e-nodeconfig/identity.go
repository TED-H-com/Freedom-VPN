package main

import (
	"crypto/ecdh"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
)

func identityFingerprints() error {
	digest := func(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
	machine, err := os.ReadFile("/etc/machine-id")
	if err != nil || len(strings.TrimSpace(string(machine))) != 32 {
		return fmt.Errorf("machine id unavailable")
	}
	raw, err := os.ReadFile("/var/lib/vpn-node/identity/cert.pem")
	if err != nil {
		return err
	}
	b, _ := pem.Decode(raw)
	if b == nil {
		return fmt.Errorf("certificate unavailable")
	}
	cert, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return err
	}
	raw, err = os.ReadFile("/etc/netbird/config.json")
	if err != nil {
		return err
	}
	var cfg struct{ PrivateKey string }
	if json.Unmarshal(raw, &cfg) != nil {
		return fmt.Errorf("native key unavailable")
	}
	key, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		return err
	}
	priv, err := ecdh.X25519().NewPrivateKey(key)
	if err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]string{"machine_id_sha256": digest([]byte(strings.TrimSpace(string(machine)))), "agent_spki_sha256": digest(cert.RawSubjectPublicKeyInfo), "netbird_public_key_sha256": digest(priv.PublicKey().Bytes())})
}
