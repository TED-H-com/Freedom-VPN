// Package envelope implements the signed document envelope used for release
// manifests (§13.3), signed desired state (§14.7), serving leases (§16.3) and
// usage checkpoints after a central restore (§17.3):
//
//	{"signed": {...}, "signatures": [{"key_id": "...", "algorithm": "ed25519", "signature_b64": "..."}]}
//
// The signature input is exactly the RFC 8785 canonical form (package
// canonjson) of the "signed" object. The only algorithm is Ed25519, and
// signature_b64 is standard base64 with padding of the 64-byte signature.
//
// Verification is fail closed (§16.2): unknown algorithms, an empty signature
// list, duplicate signature key_ids, malformed base64, and any signature by a known key that is revoked,
// outside its validity window, not allowed for the requested purpose or that
// does not verify, all reject the whole envelope. Signatures by unknown
// key_ids are ignored, but at least one valid signature by a trusted key is
// required. Each purpose uses its own keys; nodes hold public keys only.
package envelope

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
)

// AlgorithmEd25519 is the only accepted signature algorithm.
const AlgorithmEd25519 = "ed25519"

// Purpose is the use a signing key is authorized for (§16.2).
type Purpose string

// The four signing purposes. Keys are never shared between purposes in
// production; the trust store enforces that a key signs only documents of
// purposes it is allowed for.
const (
	PurposeRelease      Purpose = "release"       // release manifests (§13.3)
	PurposeDesiredState Purpose = "desired-state" // desired state and policy (§14.7)
	PurposeLease        Purpose = "lease"         // serving leases (§16.3)
	PurposeRecovery     Purpose = "recovery"      // usage checkpoints and recovery records (§17.3)
)

// Valid reports whether p is one of the defined purposes.
func (p Purpose) Valid() bool {
	switch p {
	case PurposeRelease, PurposeDesiredState, PurposeLease, PurposeRecovery:
		return true
	}
	return false
}

// Signature is one entry of the signatures array.
type Signature struct {
	KeyID        string `json:"key_id"`
	Algorithm    string `json:"algorithm"`
	SignatureB64 string `json:"signature_b64"`
}

// Envelope is a signed document. Signed holds the canonical JSON of the
// signed object.
type Envelope struct {
	Signed     json.RawMessage `json:"signed"`
	Signatures []Signature     `json:"signatures"`
}

// Error codes, equal to the API error codes of §13.11.
const (
	CodeKeyUntrusted     = "KEY_UNTRUSTED"
	CodeSignatureInvalid = "SIGNATURE_INVALID"
)

// Sentinel errors for errors.Is.
var (
	// ErrKeyUntrusted: no trusted key signed the document, or a known key
	// is revoked, expired, not yet valid or lacks the requested purpose.
	ErrKeyUntrusted = errors.New(CodeKeyUntrusted)
	// ErrSignatureInvalid: the envelope is malformed, a signature uses an
	// unknown algorithm or bad encoding, or a known key's signature fails.
	ErrSignatureInvalid = errors.New(CodeSignatureInvalid)
)

// Error is a typed verification error. Code is CodeKeyUntrusted or
// CodeSignatureInvalid. Messages never include key material or document
// contents.
type Error struct {
	Code   string
	KeyID  string // the key involved, if any
	Reason string
}

func (e *Error) Error() string {
	if e.KeyID != "" {
		return fmt.Sprintf("%s: key %q: %s", e.Code, e.KeyID, e.Reason)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Reason)
}

// Is matches the sentinel for the error's code.
func (e *Error) Is(target error) bool {
	switch target {
	case ErrKeyUntrusted:
		return e.Code == CodeKeyUntrusted
	case ErrSignatureInvalid:
		return e.Code == CodeSignatureInvalid
	}
	return false
}

func untrusted(keyID, reason string) error {
	return &Error{Code: CodeKeyUntrusted, KeyID: keyID, Reason: reason}
}

func invalid(keyID, reason string) error {
	return &Error{Code: CodeSignatureInvalid, KeyID: keyID, Reason: reason}
}

// Sign canonicalizes signed and returns an envelope with one Ed25519
// signature. signed may be a []byte or json.RawMessage holding JSON, or any
// value encoding/json can encode; it must be a JSON object within the
// canonjson subset.
func Sign(signed any, priv ed25519.PrivateKey, keyID string) (Envelope, error) {
	c, err := canonicalObject(signed)
	if err != nil {
		return Envelope{}, err
	}
	env := Envelope{Signed: c}
	if err := env.AddSignature(priv, keyID); err != nil {
		return Envelope{}, err
	}
	return env, nil
}

// AddSignature appends another signature over the same signed object, for
// example during a key rotation overlap window (§16.2).
func (e *Envelope) AddSignature(priv ed25519.PrivateKey, keyID string) error {
	if len(priv) != ed25519.PrivateKeySize {
		return errors.New("envelope: invalid Ed25519 private key size")
	}
	if keyID == "" {
		return errors.New("envelope: empty key_id")
	}
	for _, s := range e.Signatures {
		if s.KeyID == keyID {
			return invalid(keyID, "duplicate signature key_id")
		}
	}
	c, err := canonicalObject([]byte(e.Signed))
	if err != nil {
		return err
	}
	e.Signed = c
	sig := ed25519.Sign(priv, c)
	e.Signatures = append(e.Signatures, Signature{
		KeyID:        keyID,
		Algorithm:    AlgorithmEd25519,
		SignatureB64: base64.StdEncoding.EncodeToString(sig),
	})
	return nil
}

// MarshalCanonical returns the RFC 8785 canonical JSON of the envelope.
func (e Envelope) MarshalCanonical() ([]byte, error) {
	return canonjson.Marshal(e)
}

func canonicalObject(v any) ([]byte, error) {
	var c []byte
	var err error
	switch t := v.(type) {
	case []byte:
		c, err = canonjson.Canonicalize(t)
	case json.RawMessage:
		c, err = canonjson.Canonicalize(t)
	default:
		c, err = canonjson.Marshal(v)
	}
	if err != nil {
		return nil, fmt.Errorf("envelope: signed document: %w", err)
	}
	if len(c) == 0 || c[0] != '{' {
		return nil, fmt.Errorf("envelope: signed document: %w", canonjson.ErrNotObject)
	}
	return c, nil
}

// Verify checks raw, a JSON envelope, against ts for the given purpose at
// time now. On success it returns the canonical JSON of the signed object
// and the key_id of the first valid trusted signature.
//
// The caller must still validate the signed document itself (schema, node
// and incarnation binding, generation, validity times) as §16.2 requires.
func Verify(raw []byte, ts *TrustStore, purpose Purpose, now time.Time) (signedJSON []byte, keyID string, err error) {
	if ts == nil {
		return nil, "", untrusted("", "no trust store")
	}
	if !purpose.Valid() {
		return nil, "", untrusted("", fmt.Sprintf("unknown purpose %q", purpose))
	}
	// Validate the whole document with the strict parser first, so that
	// duplicate members or invalid UTF-8 cannot be interpreted differently
	// by different JSON parsers.
	if _, err := canonjson.Canonicalize(raw); err != nil {
		return nil, "", invalid("", "malformed envelope: "+err.Error())
	}
	var env Envelope
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&env); err != nil {
		return nil, "", invalid("", "malformed envelope structure")
	}
	if len(env.Signed) == 0 || bytes.Equal(bytes.TrimSpace(env.Signed), []byte("null")) {
		return nil, "", invalid("", "missing signed object")
	}
	signed, err := canonjson.Canonicalize(env.Signed)
	if err != nil || signed[0] != '{' {
		return nil, "", invalid("", "signed member is not a JSON object")
	}
	if len(env.Signatures) == 0 {
		return nil, "", invalid("", "no signatures")
	}

	// Structural checks apply to every signature, known key or not.
	sigs := make([][]byte, len(env.Signatures))
	seen := make(map[string]bool, len(env.Signatures))
	for i, s := range env.Signatures {
		if s.KeyID == "" {
			return nil, "", invalid("", fmt.Sprintf("signature %d has empty key_id", i))
		}
		if seen[s.KeyID] {
			return nil, "", invalid(s.KeyID, "duplicate signature key_id")
		}
		seen[s.KeyID] = true
		if s.Algorithm != AlgorithmEd25519 {
			return nil, "", invalid(s.KeyID, fmt.Sprintf("unsupported algorithm %q", short(s.Algorithm)))
		}
		b, err := base64.StdEncoding.Strict().DecodeString(s.SignatureB64)
		if err != nil {
			return nil, "", invalid(s.KeyID, "signature_b64 is not valid padded standard base64")
		}
		if len(b) != ed25519.SignatureSize {
			return nil, "", invalid(s.KeyID, fmt.Sprintf("signature is %d bytes, want %d", len(b), ed25519.SignatureSize))
		}
		sigs[i] = b
	}

	var first string
	for i, s := range env.Signatures {
		k, ok := ts.keys[s.KeyID]
		if !ok {
			continue // unknown key ids are ignored
		}
		if err := k.usable(purpose, now); err != nil {
			return nil, "", err
		}
		if !ed25519.Verify(k.PublicKey, signed, sigs[i]) {
			return nil, "", invalid(s.KeyID, "signature does not verify")
		}
		if first == "" {
			first = s.KeyID
		}
	}
	if first == "" {
		return nil, "", untrusted("", "no signature by a trusted key")
	}
	return signed, first, nil
}

func short(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}
