package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// KeyFile is the release signing key as stored on the build machine.
// seed_b64 is the 32-byte Ed25519 seed. This file never goes to a node
// (§16.2). purpose is always "release"; a key for another purpose is refused.
type KeyFile struct {
	Purpose string `json:"purpose"`
	KeyID   string `json:"key_id"`
	SeedB64 string `json:"seed_b64"`
}

// SigningKey is a parsed release key.
type SigningKey struct {
	KeyID string
	Seed  []byte
}

// GenerateKey creates a new Ed25519 release key. keyID is the public
// identifier stored in manifests.
func GenerateKey(keyID string) (SigningKey, error) {
	if keyID == "" {
		return SigningKey{}, errors.New("release: empty key_id")
	}
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return SigningKey{}, err
	}
	return SigningKey{KeyID: keyID, Seed: priv.Seed()}, nil
}

// KeyFromSeed builds a signing key from a 32-byte seed.
func KeyFromSeed(keyID string, seed []byte) (SigningKey, error) {
	if keyID == "" {
		return SigningKey{}, errors.New("release: empty key_id")
	}
	if len(seed) != ed25519.SeedSize {
		return SigningKey{}, fmt.Errorf("release: seed must be %d bytes", ed25519.SeedSize)
	}
	return SigningKey{KeyID: keyID, Seed: append([]byte(nil), seed...)}, nil
}

// Private returns the expanded Ed25519 private key.
func (k SigningKey) Private() ed25519.PrivateKey {
	return ed25519.NewKeyFromSeed(k.Seed)
}

// Public returns the Ed25519 public key.
func (k SigningKey) Public() ed25519.PublicKey {
	return k.Private().Public().(ed25519.PublicKey)
}

// Save writes the key with mode 0600. The directory is created at 0700.
func (k SigningKey) Save(path string) error {
	f := KeyFile{
		Purpose: string(envelope.PurposeRelease),
		KeyID:   k.KeyID,
		SeedB64: base64.StdEncoding.EncodeToString(k.Seed),
	}
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return writeSecret(path, b)
}

// LoadKey reads a release signing key. Any purpose other than release, or a
// seed that is not 32 bytes, is refused.
func LoadKey(path string) (SigningKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return SigningKey{}, err
	}
	var f KeyFile
	dec := json.NewDecoder(bytesReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return SigningKey{}, fmt.Errorf("release: key file: %w", err)
	}
	if f.Purpose != string(envelope.PurposeRelease) {
		return SigningKey{}, fmt.Errorf("release: key %q purpose %q, want release", f.KeyID, f.Purpose)
	}
	seed, err := base64.StdEncoding.Strict().DecodeString(f.SeedB64)
	if err != nil {
		return SigningKey{}, fmt.Errorf("release: key %q: seed_b64: %w", f.KeyID, err)
	}
	return KeyFromSeed(f.KeyID, seed)
}

// TrustStore builds a one-key release trust store for the validity window
// [notBefore, notAfter). Nodes receive this public document, not the seed.
func (k SigningKey) TrustStore(notBefore, notAfter time.Time) (*envelope.TrustStore, error) {
	return envelope.NewTrustStore(envelope.TrustedKey{
		KeyID:     k.KeyID,
		PublicKey: k.Public(),
		Purposes:  []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	})
}

func writeSecret(path string, b []byte) error {
	dir := dirOf(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".key-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = tmp.Close()
			_ = os.Remove(name)
		}
	}()
	if err := tmp.Chmod(0o600); err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	ok = true
	return nil
}
