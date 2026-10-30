package main

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestNetBirdPublicKeyHashesTheProfileScalar(t *testing.T) {
	priv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(map[string]string{"PrivateKey": base64.StdEncoding.EncodeToString(priv.Bytes())})
	if err != nil {
		t.Fatal(err)
	}
	pub, err := netbirdPublicKey(raw)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(pub) != sha256.Sum256(priv.PublicKey().Bytes()) {
		t.Fatal("public key hash mismatch")
	}
	if _, err = netbirdPublicKey([]byte(`{"ManagementURL":"https://netbird.example.invalid"}`)); err == nil {
		t.Fatal("profile without a key accepted")
	}
	if _, err = netbirdPublicKey([]byte(`{"PrivateKey":"not-a-key"}`)); err == nil {
		t.Fatal("malformed profile key accepted")
	}
}
