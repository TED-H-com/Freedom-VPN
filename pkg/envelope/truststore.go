package envelope

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"
)

// TrustedKey is one public key a verifier trusts.
type TrustedKey struct {
	KeyID     string
	PublicKey ed25519.PublicKey
	// Purposes the key may sign. Production keys have exactly one purpose.
	Purposes []Purpose
	// NotBefore and NotAfter bound the validity window [NotBefore, NotAfter).
	// Both are required: a trusted key never lasts forever (§15.1, §16.2).
	NotBefore time.Time
	NotAfter  time.Time
	// Revoked keys are never trusted again, whatever their window (§16.2,
	// §17.2: restoring an old snapshot must not revive a revoked key).
	Revoked bool
}

func (k TrustedKey) usable(p Purpose, now time.Time) error {
	if k.Revoked {
		return untrusted(k.KeyID, "key is revoked")
	}
	if now.Before(k.NotBefore) {
		return untrusted(k.KeyID, "key is not yet valid")
	}
	if !now.Before(k.NotAfter) {
		return untrusted(k.KeyID, "key has expired")
	}
	for _, kp := range k.Purposes {
		if kp == p {
			return nil
		}
	}
	return untrusted(k.KeyID, fmt.Sprintf("key is not allowed for purpose %q", p))
}

// TrustStore maps key_id to a trusted public key. It is immutable after
// construction and safe for concurrent use.
type TrustStore struct {
	keys map[string]TrustedKey
}

// NewTrustStore validates keys and builds a trust store. Key ids must be
// unique and non-empty, public keys 32 bytes, purposes valid and windows
// non-empty.
func NewTrustStore(keys ...TrustedKey) (*TrustStore, error) {
	ts := &TrustStore{keys: make(map[string]TrustedKey, len(keys))}
	for _, k := range keys {
		if k.KeyID == "" {
			return nil, errors.New("envelope: trusted key with empty key_id")
		}
		if _, dup := ts.keys[k.KeyID]; dup {
			return nil, fmt.Errorf("envelope: duplicate trusted key_id %q", k.KeyID)
		}
		if len(k.PublicKey) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("envelope: key %q: public key must be %d bytes", k.KeyID, ed25519.PublicKeySize)
		}
		if len(k.Purposes) == 0 {
			return nil, fmt.Errorf("envelope: key %q: no purposes", k.KeyID)
		}
		for _, p := range k.Purposes {
			if !p.Valid() {
				return nil, fmt.Errorf("envelope: key %q: unknown purpose %q", k.KeyID, p)
			}
		}
		if k.NotBefore.IsZero() || k.NotAfter.IsZero() || !k.NotBefore.Before(k.NotAfter) {
			return nil, fmt.Errorf("envelope: key %q: validity window must satisfy not_before < not_after", k.KeyID)
		}
		k.PublicKey = append(ed25519.PublicKey(nil), k.PublicKey...)
		k.Purposes = append([]Purpose(nil), k.Purposes...)
		ts.keys[k.KeyID] = k
	}
	return ts, nil
}

// Lookup returns a copy of the trusted key with the given id.
func (ts *TrustStore) Lookup(keyID string) (TrustedKey, bool) {
	k, ok := ts.keys[keyID]
	if ok {
		k.PublicKey = append(ed25519.PublicKey(nil), k.PublicKey...)
		k.Purposes = append([]Purpose(nil), k.Purposes...)
	}
	return k, ok
}

// Len returns the number of keys.
func (ts *TrustStore) Len() int { return len(ts.keys) }

// trustStoreFile is the JSON form of a trust store, for example a node's
// /etc/vpn-node/trust.json. It contains public keys only.
type trustStoreFile struct {
	Keys []trustedKeyJSON `json:"keys"`
}

type trustedKeyJSON struct {
	KeyID        string    `json:"key_id"`
	Algorithm    string    `json:"algorithm"`
	PublicKeyB64 string    `json:"public_key_b64"`
	Purposes     []Purpose `json:"purposes"`
	NotBefore    time.Time `json:"not_before"`
	NotAfter     time.Time `json:"not_after"`
	Revoked      bool      `json:"revoked"`
}

// ParseTrustStoreJSON parses the JSON trust store format:
//
//	{"keys": [{"key_id": "...", "algorithm": "ed25519", "public_key_b64": "...",
//	           "purposes": ["release"], "not_before": "...Z", "not_after": "...Z",
//	           "revoked": false}]}
//
// Unknown fields are rejected.
func ParseTrustStoreJSON(raw []byte) (*TrustStore, error) {
	var f trustStoreFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return nil, fmt.Errorf("envelope: trust store: %w", err)
	}
	keys := make([]TrustedKey, 0, len(f.Keys))
	for _, k := range f.Keys {
		if k.Algorithm != AlgorithmEd25519 {
			return nil, fmt.Errorf("envelope: trust store key %q: unsupported algorithm %q", k.KeyID, k.Algorithm)
		}
		pub, err := base64.StdEncoding.Strict().DecodeString(k.PublicKeyB64)
		if err != nil {
			return nil, fmt.Errorf("envelope: trust store key %q: bad public_key_b64", k.KeyID)
		}
		keys = append(keys, TrustedKey{
			KeyID: k.KeyID, PublicKey: pub, Purposes: k.Purposes,
			NotBefore: k.NotBefore, NotAfter: k.NotAfter, Revoked: k.Revoked,
		})
	}
	return NewTrustStore(keys...)
}

// MarshalJSON encodes the trust store in the format ParseTrustStoreJSON reads,
// with keys sorted by key_id.
func (ts *TrustStore) MarshalJSON() ([]byte, error) {
	f := trustStoreFile{Keys: []trustedKeyJSON{}}
	ids := make([]string, 0, len(ts.keys))
	for id := range ts.keys {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		k := ts.keys[id]
		f.Keys = append(f.Keys, trustedKeyJSON{
			KeyID: k.KeyID, Algorithm: AlgorithmEd25519,
			PublicKeyB64: base64.StdEncoding.EncodeToString(k.PublicKey),
			Purposes:     k.Purposes, NotBefore: k.NotBefore.UTC(), NotAfter: k.NotAfter.UTC(), Revoked: k.Revoked,
		})
	}
	return json.Marshal(f)
}
