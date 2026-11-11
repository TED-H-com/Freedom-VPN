package release

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// ReleaseTrustUpdateKind identifies the offline authorization in §13.4.
const ReleaseTrustUpdateKind = "fvpn-release-trust-v1"

// ReleaseTrustRecord is public metadata, compatible with the centre's ledger.
// The offline tool deliberately does not link the vault provider (§16.2).
type ReleaseTrustRecord struct {
	ID        string           `json:"id"`
	Purpose   envelope.Purpose `json:"purpose"`
	Public    []byte           `json:"public"`
	State     string           `json:"state"`
	NotBefore time.Time        `json:"not_before"`
	NotAfter  time.Time        `json:"not_after"`
	RevokedAt *time.Time       `json:"revoked_at,omitempty"`
}

// ReleaseTrustUpdate changes only the named records; omitted keys remain.
type ReleaseTrustUpdate struct {
	Kind     string               `json:"kind"`
	Revision decstr.U64           `json:"revision"`
	Keys     []ReleaseTrustRecord `json:"keys"`
}

// ReleaseTrust is a validated node-format bundle. It can contain other node
// purposes, but only release keys can authorize or be changed by an update.
type ReleaseTrust struct {
	store *envelope.TrustStore
	keys  []envelope.TrustedKey
}

// ParseReleaseTrust retains all keys to check public-key reuse and the usable
// root guard. Lookup alone cannot detect reuse under an omitted ID.
func ParseReleaseTrust(raw []byte) (*ReleaseTrust, error) {
	ts, err := envelope.ParseNodeTrustStoreJSON(raw)
	if err != nil {
		return nil, errors.New("release: invalid current trust store")
	}
	// The authoritative parser above has already validated the whole wire
	// format. This projection only enumerates IDs; it never rebuilds trust.
	var ids struct {
		Keys []struct {
			ID string `json:"key_id"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, errors.New("release: invalid current trust store")
	}
	t := &ReleaseTrust{store: ts}
	for _, id := range ids.Keys {
		k, ok := ts.Lookup(id.ID)
		if !ok {
			return nil, errors.New("release: invalid current trust store")
		}
		t.keys = append(t.keys, k)
	}
	return t, nil
}

// TrustUpdateOptions specifies exactly one operation and one decision instant.
// Revision is chosen by the operator, never inferred from a public bundle.
type TrustUpdateOptions struct {
	Revision          decstr.U64
	Operation         string // add, retire or revoke
	TargetKeyID       string
	AddTrust          []byte
	NotAfter          time.Time
	At                time.Time
	AllowNoUsableRoot bool
}

// TrustRecordResult describes a record relative to the supplied public bundle.
type TrustRecordResult struct {
	ID             string    `json:"id"`
	New            bool      `json:"new"`
	State          string    `json:"state"`
	NotBefore      time.Time `json:"not_before"`
	NotAfter       time.Time `json:"not_after"`
	WindowNarrowed bool      `json:"window_narrowed"`
	NewlyRevoked   bool      `json:"newly_revoked"`
}

// TrustVerification is an offline observation, not an apply receipt. Retirement,
// revocation times and the stored revision are absent from node trust bundles.
type TrustVerification struct {
	Update          ReleaseTrustUpdate
	SignerKeyIDs    []string
	UpdateSHA256    string
	Records         []TrustRecordResult
	UsableRootAfter bool
}

// BuildTrustUpdate constructs and verifies the complete authorization in memory
// against OLD trust (§13.4). No private material is included in the result.
func BuildTrustUpdate(key SigningKey, trust *ReleaseTrust, opt TrustUpdateOptions) ([]byte, TrustVerification, error) {
	fail := func(message string) ([]byte, TrustVerification, error) {
		return nil, TrustVerification{}, errors.New("release: " + message)
	}
	if trust == nil || trust.store == nil || opt.At.IsZero() || opt.Revision == 0 {
		return fail("trust update requires trust, time and positive revision")
	}
	if len(key.Seed) != ed25519.SeedSize {
		return fail("invalid signing key")
	}
	signer, ok := trust.store.Lookup(key.KeyID)
	if !ok || !bytes.Equal(signer.PublicKey, key.Public()) {
		return fail("signing key does not match current trust")
	}
	at := opt.At.UTC()
	var record ReleaseTrustRecord
	switch opt.Operation {
	case "add":
		if opt.TargetKeyID != "" || !opt.NotAfter.IsZero() {
			return fail("add cannot include a target ID or not-after")
		}
		added, err := ParseReleaseTrust(opt.AddTrust)
		if err != nil || len(added.keys) != 1 {
			return fail("add requires a one-key node trust store")
		}
		k := added.keys[0]
		if k.Purposes[0] != envelope.PurposeRelease || k.Revoked || !at.Before(k.NotAfter) {
			return fail("added key must be release, not revoked and not expired")
		}
		for _, old := range trust.keys {
			if old.KeyID == k.KeyID || bytes.Equal(old.PublicKey, k.PublicKey) {
				return fail("added key ID and public key must be new")
			}
		}
		record = trustRecord(k, "active")
	case "retire", "revoke":
		if len(opt.AddTrust) != 0 || (opt.Operation == "revoke" && !opt.NotAfter.IsZero()) {
			return fail("unexpected operation options")
		}
		k, ok := trust.store.Lookup(opt.TargetKeyID)
		if !ok || k.Purposes[0] != envelope.PurposeRelease || k.Revoked {
			return fail("target must be a current non-revoked release key")
		}
		record = trustRecord(k, "retired")
		if opt.Operation == "retire" {
			if !k.NotBefore.Before(opt.NotAfter) || opt.NotAfter.After(k.NotAfter) {
				return fail("retirement end must be after not_before and no later than current not_after")
			}
			record.NotAfter = opt.NotAfter.UTC()
		} else {
			record.State, record.RevokedAt = "revoked", &at
		}
	default:
		return fail("exactly one add, retire or revoke operation is required")
	}
	u := ReleaseTrustUpdate{Kind: ReleaseTrustUpdateKind, Revision: opt.Revision, Keys: []ReleaseTrustRecord{record}}
	e, err := envelope.Sign(u, key.Private(), key.KeyID)
	if err != nil {
		return fail("cannot sign trust update")
	}
	raw, err := e.MarshalCanonical()
	if err != nil {
		return fail("cannot canonicalize trust update")
	}
	result, err := VerifyTrustUpdate(raw, trust, at, nil)
	if err != nil {
		return nil, TrustVerification{}, err
	}
	if !result.UsableRootAfter && !opt.AllowNoUsableRoot {
		return fail("no usable release root would remain; -allow-no-usable-root is required for deliberate emergency removal")
	}
	return raw, result, nil
}

func trustRecord(k envelope.TrustedKey, state string) ReleaseTrustRecord {
	return ReleaseTrustRecord{ID: k.KeyID, Public: k.PublicKey, Purpose: envelope.PurposeRelease,
		State: state, NotBefore: k.NotBefore.UTC(), NotAfter: k.NotAfter.UTC()}
}

// VerifyTrustUpdate checks schema, signatures and every transition visible in
// the public trust store. The centre must independently enforce ledger history
// at apply time (§13.4, §17.2). A false UsableRootAfter is a report, not a failure.
func VerifyTrustUpdate(raw []byte, trust *ReleaseTrust, at time.Time, currentRevision *decstr.U64) (TrustVerification, error) {
	var result TrustVerification
	if trust == nil || trust.store == nil || at.IsZero() {
		return result, errors.New("release: trust verification requires trust and time")
	}
	if err := ValidateTrustUpdateEnvelope(raw); err != nil {
		return result, err
	}
	signed, _, err := envelope.Verify(raw, trust.store, envelope.PurposeRelease, at)
	if err != nil {
		return result, err
	}
	var u ReleaseTrustUpdate
	d := json.NewDecoder(bytes.NewReader(signed))
	d.DisallowUnknownFields()
	if err := d.Decode(&u); err != nil || u.Kind != ReleaseTrustUpdateKind || u.Revision == 0 || len(u.Keys) == 0 {
		return result, errors.New("release: invalid trust update payload")
	}
	if currentRevision != nil && u.Revision <= *currentRevision {
		return result, errors.New("release: update revision must exceed current revision")
	}
	ids, pubs := map[string]bool{}, map[string]bool{}
	after := make(map[string]envelope.TrustedKey, len(trust.keys)+len(u.Keys))
	owners := make(map[string]string, len(trust.keys))
	for _, k := range trust.keys {
		after[k.KeyID], owners[string(k.PublicKey)] = k, k.KeyID
	}
	for _, k := range u.Keys {
		// The ledger bounds IDs by bytes, whereas JSON Schema counts characters.
		if k.ID == "" || len(k.ID) > 128 || ids[k.ID] || pubs[string(k.Public)] || k.Purpose != envelope.PurposeRelease ||
			len(k.Public) != ed25519.PublicKeySize || k.NotBefore.IsZero() || !k.NotBefore.Before(k.NotAfter) ||
			(k.State != "active" && k.State != "retired" && k.State != "revoked") || (k.State == "revoked") != (k.RevokedAt != nil) {
			return TrustVerification{}, errors.New("release: invalid or duplicate trust key record")
		}
		ids[k.ID], pubs[string(k.Public)] = true, true
		old, exists := trust.store.Lookup(k.ID)
		if exists && (old.Purposes[0] != envelope.PurposeRelease || !bytes.Equal(old.PublicKey, k.Public) ||
			k.NotBefore.Before(old.NotBefore) || k.NotAfter.After(old.NotAfter) || (old.Revoked && k.State != "revoked")) {
			return TrustVerification{}, errors.New("release: existing key substitution, expanded window or reversed revocation")
		}
		if owner, used := owners[string(k.Public)]; used && owner != k.ID {
			return TrustVerification{}, errors.New("release: public key is already recorded under another ID")
		}
		result.Records = append(result.Records, TrustRecordResult{ID: k.ID, New: !exists, State: k.State,
			NotBefore: k.NotBefore, NotAfter: k.NotAfter,
			WindowNarrowed: exists && (k.NotBefore.After(old.NotBefore) || k.NotAfter.Before(old.NotAfter)),
			NewlyRevoked:   k.State == "revoked" && (!exists || !old.Revoked)})
		after[k.ID] = envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{envelope.PurposeRelease},
			NotBefore: k.NotBefore, NotAfter: k.NotAfter, Revoked: k.State == "revoked"}
	}
	for _, k := range after {
		if k.Purposes[0] == envelope.PurposeRelease && !k.Revoked && !at.Before(k.NotBefore) && at.Before(k.NotAfter) {
			result.UsableRootAfter = true
		}
	}
	var e envelope.Envelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return TrustVerification{}, err
	}
	// Verify already checked EVERY known signature, not just the first one.
	for _, sig := range e.Signatures {
		if _, known := trust.store.Lookup(sig.KeyID); known {
			result.SignerKeyIDs = append(result.SignerKeyIDs, sig.KeyID)
		}
	}
	result.UpdateSHA256, err = EnvelopeSHA256(raw)
	if err != nil {
		return TrustVerification{}, err
	}
	result.Update = u
	return result, nil
}

// TrustRecordJSON provides a single canonical JSON value for a key_record result
// line, so opaque IDs cannot inject additional output lines.
func TrustRecordJSON(record TrustRecordResult) ([]byte, error) {
	return canonjson.Marshal(record)
}
