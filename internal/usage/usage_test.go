package usage_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/TED-H-com/Freedom-VPN/internal/usage"
	"github.com/TED-H-com/Freedom-VPN/pkg/api"
	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestMTR05_PayloadSha256GoldenVector(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "vectors", "usage-event-payload-sha256.json"))
	if err != nil {
		t.Fatalf("read payload-sha256 vector: %v", err)
	}
	var vec struct {
		CanonicalWithoutHash string         `json:"canonical_without_hash"`
		Event                api.UsageEvent `json:"event"`
		PayloadSha256        string         `json:"payload_sha256"`
	}
	if err := json.Unmarshal(raw, &vec); err != nil {
		t.Fatalf("unmarshal payload-sha256 vector: %v", err)
	}

	_, gotHex, _, err := usage.ComputeEventHash(vec.Event)
	if err != nil {
		t.Fatalf("ComputeEventHash: %v", err)
	}
	if gotHex != vec.PayloadSha256 {
		t.Fatalf("hash mismatch: got %s, want %s", gotHex, vec.PayloadSha256)
	}
	if vec.Event.Records[1].UploadBytes != decstr.U64(9007199254740993) {
		t.Fatalf("2^53+1 precision lost: got %d", vec.Event.Records[1].UploadBytes)
	}

	// Also verify both events in contracts/examples/valid/UsageBatchRequest.json.
	batchRaw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "valid", "UsageBatchRequest.json"))
	if err != nil {
		t.Fatalf("read UsageBatchRequest.json: %v", err)
	}
	var batch api.UsageBatchRequest
	if err := json.Unmarshal(batchRaw, &batch); err != nil {
		t.Fatalf("unmarshal UsageBatchRequest.json: %v", err)
	}
	for i, ev := range batch.Events {
		_, h, _, err := usage.ComputeEventHash(ev)
		if err != nil {
			t.Fatalf("event[%d] ComputeEventHash: %v", i, err)
		}
		if h != ev.PayloadSha256 {
			t.Fatalf("event[%d] hash mismatch: got %s, want %s", i, h, ev.PayloadSha256)
		}
	}
}

func Test17_3_CheckpointSigningGoldenVector(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "contracts", "vectors", "usage-checkpoint-signing.json"))
	if err != nil {
		t.Fatalf("read checkpoint signing vector: %v", err)
	}
	var vec struct {
		CanonicalSigned       string              `json:"canonical_signed"`
		CanonicalSignedSha256 string              `json:"canonical_signed_sha256"`
		EnvelopeSha256        string              `json:"envelope_sha256"`
		KeyID                 string              `json:"key_id"`
		PublicKeyB64          string              `json:"public_key_b64"`
		Purpose               string              `json:"purpose"`
		SeedHex               string              `json:"seed_hex"`
		SignatureB64          string              `json:"signature_b64"`
		Signed                api.UsageCheckpoint `json:"signed"`
		VerifyAt              time.Time           `json:"verify_at"`
	}
	if err := json.Unmarshal(raw, &vec); err != nil {
		t.Fatalf("unmarshal checkpoint signing vector: %v", err)
	}

	seed, err := hex.DecodeString(vec.SeedHex)
	if err != nil {
		t.Fatalf("decode seed_hex: %v", err)
	}
	priv := ed25519.NewKeyFromSeed(seed)
	pubBytes, err := base64.StdEncoding.DecodeString(vec.PublicKeyB64)
	if err != nil {
		t.Fatalf("decode public_key_b64: %v", err)
	}

	signer := usage.NewRecoverySigner(vec.KeyID, priv)
	env, err := signer.Sign(envelope.PurposeRecovery, vec.Signed)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if string(env.Signed) != vec.CanonicalSigned {
		t.Fatalf("canonical signed mismatch:\ngot:  %s\nwant: %s", string(env.Signed), vec.CanonicalSigned)
	}
	signedSum := sha256.Sum256(env.Signed)
	if hex.EncodeToString(signedSum[:]) != vec.CanonicalSignedSha256 {
		t.Fatalf("canonical_signed_sha256 mismatch: got %x, want %s", signedSum, vec.CanonicalSignedSha256)
	}
	if len(env.Signatures) != 1 || env.Signatures[0].SignatureB64 != vec.SignatureB64 {
		t.Fatalf("signature mismatch: got %+v, want %s", env.Signatures, vec.SignatureB64)
	}
	canonEnv, err := env.MarshalCanonical()
	if err != nil {
		t.Fatalf("MarshalCanonical: %v", err)
	}
	envSum := sha256.Sum256(canonEnv)
	if hex.EncodeToString(envSum[:]) != vec.EnvelopeSha256 {
		t.Fatalf("envelope_sha256 mismatch: got %x, want %s", envSum, vec.EnvelopeSha256)
	}

	// Verify with pkg/envelope TrustStore.
	ts, err := envelope.NewTrustStore(
		envelope.TrustedKey{
			KeyID:     vec.KeyID,
			PublicKey: ed25519.PublicKey(pubBytes),
			Purposes:  []envelope.Purpose{envelope.PurposeRecovery},
			NotBefore: vec.VerifyAt.Add(-time.Hour),
			NotAfter:  vec.VerifyAt.Add(time.Hour),
		},
	)
	if err != nil {
		t.Fatalf("NewTrustStore: %v", err)
	}
	verifiedBytes, keyID, err := envelope.Verify(canonEnv, ts, envelope.PurposeRecovery, vec.VerifyAt)
	if err != nil {
		t.Fatalf("envelope.Verify: %v", err)
	}
	if keyID != vec.KeyID || string(verifiedBytes) != vec.CanonicalSigned {
		t.Fatalf("verified output mismatch: keyID=%s", keyID)
	}
	// Wrong purpose must be rejected fail-closed.
	if _, _, err := envelope.Verify(canonEnv, ts, envelope.PurposeDesiredState, vec.VerifyAt); err == nil {
		t.Fatalf("expected wrong purpose verification to fail")
	}
}

func TestAT17_BoundarySplitConservationProperty(t *testing.T) {
	// 1. Negative and zero-length intervals must be rejected.
	t0 := time.Date(2026, 10, 31, 23, 59, 0, 0, time.UTC)
	if _, err := usage.Split(t0, t0, 100, 200, usage.GranularityMonth); err == nil {
		t.Fatalf("expected zero-length interval to fail")
	}
	if _, err := usage.Split(t0, t0.Add(-time.Second), 100, 200, usage.GranularityMonth); err == nil {
		t.Fatalf("expected negative interval to fail")
	}

	// 2. Exact single-period sample is not marked Estimated.
	segs, err := usage.Split(
		time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 15, 10, 1, 0, 0, time.UTC),
		123456, 789012,
		usage.GranularityMonth,
	)
	if err != nil || len(segs) != 1 || segs[0].Estimated || segs[0].Upload != 123456 || segs[0].Download != 789012 {
		t.Fatalf("single-period split unexpected: %+v err=%v", segs, err)
	}

	// 3. Month boundary split with remainder going to the later segment.
	// 30s in Oct, 30s in Nov, 3 bytes upload -> Oct gets floor(3*30/60)=1, Nov gets remainder 3-1=2.
	bStart := time.Date(2026, 10, 31, 23, 59, 30, 0, time.UTC)
	bEnd := time.Date(2026, 11, 1, 0, 0, 30, 0, time.UTC)
	segs, err = usage.Split(bStart, bEnd, 3, 5, usage.GranularityMonth)
	if err != nil {
		t.Fatalf("boundary split: %v", err)
	}
	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].ID != "2026-10" || segs[0].Upload != 1 || segs[0].Download != 2 || !segs[0].Estimated {
		t.Fatalf("segment 0 unexpected: %+v", segs[0])
	}
	if segs[1].ID != "2026-11" || segs[1].Upload != 2 || segs[1].Download != 3 || !segs[1].Estimated {
		t.Fatalf("segment 1 unexpected: %+v", segs[1])
	}

	// 4. Property test: 2000 random intervals across month/year/day boundaries,
	// including values > 2^53 and MaxUint64, always sum exactly to the original bytes.
	rng := rand.New(rand.NewPCG(20261029, 17))
	specialBytes := []uint64{
		0, 1, 2, 3, 7, 13, 999999999,
		(1 << 53) - 1,
		(1 << 53),
		(1 << 53) + 1,
		(1 << 62) + 17,
		math.MaxUint64 - 1,
		math.MaxUint64,
	}
	baseTime := time.Date(2026, 1, 31, 23, 0, 0, 0, time.UTC)

	for i := 0; i < 2000; i++ {
		offsetNs := rng.Int64N(int64(70 * 24 * time.Hour))
		durNs := rng.Int64N(int64(45*24*time.Hour)) + 1 // at least 1ns
		start := baseTime.Add(time.Duration(offsetNs))
		end := start.Add(time.Duration(durNs))

		up := rng.Uint64()
		down := rng.Uint64()
		if i < len(specialBytes)*len(specialBytes) {
			up = specialBytes[i%len(specialBytes)]
			down = specialBytes[(i/len(specialBytes))%len(specialBytes)]
		}

		for _, gran := range []usage.Granularity{usage.GranularityMonth, usage.GranularityDay} {
			parts, err := usage.Split(start, end, up, down, gran)
			if err != nil {
				t.Fatalf("iter %d (%s) Split failed: %v", i, gran, err)
			}
			if len(parts) == 0 {
				t.Fatalf("iter %d (%s) Split returned 0 segments", i, gran)
			}
			var sumUp, sumDown uint64
			for j, p := range parts {
				if len(parts) > 1 && !p.Estimated {
					t.Fatalf("iter %d (%s) segment %d not marked Estimated", i, gran, j)
				}
				if len(parts) == 1 && p.Estimated {
					t.Fatalf("iter %d (%s) single segment marked Estimated", i, gran)
				}
				if sumUp > math.MaxUint64-p.Upload || sumDown > math.MaxUint64-p.Download {
					t.Fatalf("iter %d (%s) sum overflowed uint64", i, gran)
				}
				sumUp += p.Upload
				sumDown += p.Download
			}
			if sumUp != up || sumDown != down {
				t.Fatalf("iter %d (%s) conservation violated: sum=(%d,%d) != orig=(%d,%d)",
					i, gran, sumUp, sumDown, up, down)
			}
		}
	}
}

func TestAT15_AdvanceContiguousAndMissingRanges(t *testing.T) {
	// Spec example: 100 is ACKed. 102 arrives before 101.
	hcs := usage.AdvanceContiguous(100, []uint64{102})
	if hcs != 100 {
		t.Fatalf("want hcs=100 when 101 is missing, got %d", hcs)
	}
	gaps := usage.MissingRanges(hcs, []uint64{102})
	if len(gaps) != 1 || gaps[0].Start != 101 || gaps[0].End != 101 {
		t.Fatalf("want missing [101, 101], got %+v", gaps)
	}

	// Now 101 arrives; HCS advances to 102 and missing_ranges becomes empty.
	hcs = usage.AdvanceContiguous(hcs, []uint64{101, 102})
	if hcs != 102 {
		t.Fatalf("want hcs=102 after 101 arrives, got %d", hcs)
	}
	gaps = usage.MissingRanges(hcs, []uint64{101, 102})
	if len(gaps) != 0 {
		t.Fatalf("want no missing ranges, got %+v", gaps)
	}

	// Multiple disjoint gaps: hcs=400, committed={410, 411, 415, 417}.
	gaps = usage.MissingRanges(400, []uint64{417, 410, 415, 411})
	want := []usage.SeqRange{
		{Start: 401, End: 409},
		{Start: 412, End: 414},
		{Start: 416, End: 416},
	}
	if len(gaps) != len(want) {
		t.Fatalf("gaps=%+v, want %+v", gaps, want)
	}
	for i := range want {
		if gaps[i] != want[i] {
			t.Fatalf("gap[%d]=%+v, want %+v", i, gaps[i], want[i])
		}
	}
}

func TestService_UnavailableDatabaseReturns503(t *testing.T) {
	var svc usage.Service
	_, err := svc.GetLedgerState(context.Background())
	var ae *apierr.Error
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusServiceUnavailable || !ae.IsRetryable() {
		t.Fatalf("want 503 retryable error, got %v", err)
	}

	_, err = svc.IngestBatch(
		context.Background(),
		usage.NodeIdentity{NodeID: uuid.New(), NodeIncarnationID: uuid.New()},
		api.UsageBatchRequest{},
	)
	if !errors.As(err, &ae) || ae.HTTPStatus() != http.StatusServiceUnavailable || !ae.IsRetryable() {
		t.Fatalf("want 503 retryable error from IngestBatch, got %v", err)
	}
}
