package usage

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// Signer signs recovery documents (usage checkpoints and recovery records)
// using a recovery-purpose key (§16.2, §17.3).
type Signer interface {
	Sign(purpose envelope.Purpose, signed any) (envelope.Envelope, error)
}

// Ed25519Signer is an in-memory Signer implementation backed by an Ed25519 key.
type Ed25519Signer struct {
	KeyID      string
	PrivateKey ed25519.PrivateKey
	Purpose    envelope.Purpose
}

// NewRecoverySigner creates an Ed25519Signer restricted to envelope.PurposeRecovery
// for signing per-stream usage checkpoints (§16.2, §17.3).
func NewRecoverySigner(keyID string, priv ed25519.PrivateKey) *Ed25519Signer {
	return &Ed25519Signer{
		KeyID:      keyID,
		PrivateKey: priv,
		Purpose:    envelope.PurposeRecovery,
	}
}

// NewLedgerEpochSigner creates an Ed25519Signer restricted to envelope.PurposeLedgerEpoch
// for signing central-only usage_recovery_records when rotating ledger_epoch (§16.2, §17.3).
// Ledger-epoch keys must never be included in node trust bundles.
func NewLedgerEpochSigner(keyID string, priv ed25519.PrivateKey) *Ed25519Signer {
	return &Ed25519Signer{
		KeyID:      keyID,
		PrivateKey: priv,
		Purpose:    envelope.PurposeLedgerEpoch,
	}
}

// VerifyRecoveryRecord verifies a signed usage_recovery_records envelope under
// the central-only envelope.PurposeLedgerEpoch purpose (§16.2, §17.3).
func VerifyRecoveryRecord(raw []byte, ts *envelope.TrustStore, now time.Time) (RecoverySignedPayload, string, error) {
	signedJSON, keyID, err := envelope.Verify(raw, ts, envelope.PurposeLedgerEpoch, now)
	if err != nil {
		return RecoverySignedPayload{}, "", err
	}
	var payload RecoverySignedPayload
	if err := json.Unmarshal(signedJSON, &payload); err != nil {
		return RecoverySignedPayload{}, "", fmt.Errorf("usage: decode recovery record payload: %w", err)
	}
	return payload, keyID, nil
}

// Sign implements Signer.
func (s *Ed25519Signer) Sign(purpose envelope.Purpose, signed any) (envelope.Envelope, error) {
	if s == nil {
		return envelope.Envelope{}, fmt.Errorf("usage: signer is nil")
	}
	if !purpose.Valid() {
		return envelope.Envelope{}, fmt.Errorf("usage: invalid signing purpose %q", purpose)
	}
	if s.Purpose != "" && purpose != s.Purpose {
		return envelope.Envelope{}, fmt.Errorf("usage: signer purpose mismatch: got %q, want %q", purpose, s.Purpose)
	}
	return envelope.Sign(signed, s.PrivateKey, s.KeyID)
}

// ComputeEventHash computes the RFC 8785 canonical JSON of ev (including payload_sha256)
// for storage, and the SHA-256 digest of the canonical JSON of ev without the top-level
// payload_sha256 field (§10.5).
func ComputeEventHash(ev api.UsageEvent) (hashBytes [32]byte, hashHex string, canonicalPayload []byte, err error) {
	norm := normalizeEventForMarshal(ev)
	raw, err := json.Marshal(norm)
	if err != nil {
		return [32]byte{}, "", nil, fmt.Errorf("usage: marshal event: %w", err)
	}
	withoutHash, err := canonjson.CanonicalizeWithout(raw, "payload_sha256")
	if err != nil {
		return [32]byte{}, "", nil, fmt.Errorf("usage: canonicalize event without payload_sha256: %w", err)
	}
	sum := sha256.Sum256(withoutHash)
	hexStr := hex.EncodeToString(sum[:])

	norm.PayloadSha256 = hexStr
	rawWithHash, err := json.Marshal(norm)
	if err != nil {
		return [32]byte{}, "", nil, fmt.Errorf("usage: marshal event with payload_sha256: %w", err)
	}
	canonFull, err := canonjson.Canonicalize(rawWithHash)
	if err != nil {
		return [32]byte{}, "", nil, fmt.Errorf("usage: canonicalize full event: %w", err)
	}
	return sum, hexStr, canonFull, nil
}

// WithComputedHash returns a copy of ev with PayloadSha256 populated from its canonical content.
func WithComputedHash(ev api.UsageEvent) (api.UsageEvent, error) {
	norm := normalizeEventForMarshal(ev)
	_, hexStr, _, err := ComputeEventHash(norm)
	if err != nil {
		return api.UsageEvent{}, err
	}
	norm.PayloadSha256 = hexStr
	return norm, nil
}

func normalizeEventForMarshal(ev api.UsageEvent) api.UsageEvent {
	ev.IntervalStart = ev.IntervalStart.UTC()
	ev.IntervalEnd = ev.IntervalEnd.UTC()
	if ev.Gap != nil {
		g := *ev.Gap
		if t, err := g.LastCheckpointAt.Get(); err == nil {
			g.LastCheckpointAt.Set(t.UTC())
		}
		ev.Gap = &g
	}
	if ev.Records == nil {
		ev.Records = []api.UsageRecord{}
	}
	return ev
}

// computeTombstoneHash produces a deterministic 32-byte SHA-256 hash for a
// replay_unavailable tombstone row in usage_events (§10.8 step 4, §17.3 step 9).
func computeTombstoneHash(nodeID, incarnationID uuid.UUID, seq uint64, from, until time.Time) ([32]byte, error) {
	doc := struct {
		Kind              string     `json:"kind"`
		NodeID            uuid.UUID  `json:"node_id"`
		NodeIncarnationID uuid.UUID  `json:"node_incarnation_id"`
		EventSeq          decstr.U64 `json:"event_seq"`
		IntervalStart     time.Time  `json:"interval_start"`
		IntervalEnd       time.Time  `json:"interval_end"`
	}{
		Kind:              "replay_unavailable",
		NodeID:            nodeID,
		NodeIncarnationID: incarnationID,
		EventSeq:          decstr.U64(seq),
		IntervalStart:     from.UTC(),
		IntervalEnd:       until.UTC(),
	}
	canon, err := canonjson.Marshal(doc)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(canon), nil
}

func toAPISignatures(sigs []envelope.Signature) []api.Signature {
	out := make([]api.Signature, len(sigs))
	for i, s := range sigs {
		out[i] = api.Signature{
			Algorithm:    api.SignatureAlgorithm(s.Algorithm),
			KeyId:        s.KeyID,
			SignatureB64: s.SignatureB64,
		}
	}
	return out
}
