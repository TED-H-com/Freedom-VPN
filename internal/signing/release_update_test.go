package signing

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

type releaseKeyPair struct {
	id   string
	pub  ed25519.PublicKey
	priv ed25519.PrivateKey
}

type releaseTestFixture struct {
	t     *testing.T
	p     *FileProvider
	now   time.Time
	start time.Time
	end   time.Time
	keys  map[string]releaseKeyPair
}

func newReleaseTestFixture(t *testing.T) *releaseTestFixture {
	t.Helper()
	p := vault(t)
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	f := &releaseTestFixture{
		t:     t,
		p:     p,
		now:   now,
		start: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		end:   time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
		keys:  map[string]releaseKeyPair{},
	}
	p.now = func() time.Time { return f.now }
	for _, id := range []string{"release-A", "release-B", "release-C"} {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = releaseKeyPair{id: id, pub: pub, priv: priv}
	}
	root := envelope.TrustedKey{
		KeyID:     "release-A",
		PublicKey: f.keys["release-A"].pub,
		Purposes:  []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: f.start,
		NotAfter:  f.end,
	}
	if err := p.PinReleaseRoot(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	return f
}

func (f *releaseTestFixture) record(id, state string) KeyRecord {
	k := f.keys[id]
	rec := KeyRecord{
		ID:        id,
		Purpose:   Release,
		Public:    bytes.Clone(k.pub),
		State:     state,
		NotBefore: f.start,
		NotAfter:  f.end,
	}
	if state == "revoked" {
		at := f.now
		rec.RevokedAt = &at
	}
	return rec
}

func (f *releaseTestFixture) sign(payload any, signerIDs ...string) []byte {
	f.t.Helper()
	first := f.keys[signerIDs[0]]
	env, err := envelope.Sign(payload, first.priv, first.id)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, id := range signerIDs[1:] {
		k := f.keys[id]
		if err := env.AddSignature(k.priv, k.id); err != nil {
			f.t.Fatal(err)
		}
	}
	raw, err := env.MarshalCanonical()
	if err != nil {
		f.t.Fatal(err)
	}
	return raw
}

func (f *releaseTestFixture) signUpdate(rev decstr.U64, signerID string, records ...KeyRecord) []byte {
	return f.sign(ReleaseUpdate{
		Kind:     "fvpn-release-trust-v1",
		Revision: rev,
		Keys:     records,
	}, signerID)
}

func TestUPD02_ReleaseTrustConditionalHandover(t *testing.T) {
	f := newReleaseTestFixture(t)
	ctx := context.Background()

	// Step 1: add release-B signed by release-A.
	addB := f.signUpdate(1, "release-A", f.record("release-B", "active"))
	r1, err := f.p.UpdateReleaseTrustConditional(ctx, addB, 1, false)
	if err != nil || r1.Outcome != ChangeApplied {
		t.Fatalf("add B: outcome=%v err=%v", r1.Outcome, err)
	}
	if r1.OldRevision != 1 || r1.NewRevision != 2 || r1.OldReleaseRevision != 0 || r1.NewReleaseRevision != 1 ||
		len(r1.Keys) != 1 || r1.Keys[0].Before != nil || r1.Keys[0].After.ID != "release-B" ||
		!reflect.DeepEqual(r1.UsableRootIDs, []string{"release-A", "release-B"}) {
		t.Fatalf("unexpected add B result: %+v", r1)
	}

	// Step 2: retire release-A signed by release-B with narrowed window.
	retireRec := f.record("release-A", "retired")
	retireEnd := f.now.Add(time.Hour)
	retireRec.NotAfter = retireEnd
	retireA := f.signUpdate(2, "release-B", retireRec)
	r2, err := f.p.UpdateReleaseTrustConditional(ctx, retireA, 2, false)
	if err != nil || r2.Outcome != ChangeApplied {
		t.Fatalf("retire A: outcome=%v err=%v", r2.Outcome, err)
	}
	if r2.OldRevision != 2 || r2.NewRevision != 3 || r2.OldReleaseRevision != 1 || r2.NewReleaseRevision != 2 ||
		len(r2.Keys) != 1 || r2.Keys[0].Before == nil || r2.Keys[0].Before.State != "active" ||
		r2.Keys[0].After.State != "retired" || !r2.Keys[0].After.NotAfter.Equal(retireEnd) ||
		!reflect.DeepEqual(r2.UsableRootIDs, []string{"release-A", "release-B"}) {
		t.Fatalf("unexpected retire A result: %+v", r2)
	}

	// Retired release-A still verifies within its narrowed window, then expires at NotAfter.
	probeA := f.sign(map[string]string{"release": "probe-A"}, "release-A")
	if _, err := f.p.VerifyRelease(ctx, probeA); err != nil {
		t.Fatalf("retired A within window rejected: %v", err)
	}
	f.now = retireEnd
	if _, err := f.p.VerifyRelease(ctx, probeA); err == nil {
		t.Fatal("retired A accepted at NotAfter")
	}
	f.now = retireEnd.Add(-time.Minute)

	// Step 3: revoke release-A signed by release-B.
	revokeRec := retireRec
	revokeRec.State = "revoked"
	revokedAt := f.now
	revokeRec.RevokedAt = &revokedAt
	revokeA := f.signUpdate(3, "release-B", revokeRec)
	r3, err := f.p.UpdateReleaseTrustConditional(ctx, revokeA, 3, false)
	if err != nil || r3.Outcome != ChangeApplied {
		t.Fatalf("revoke A: outcome=%v err=%v", r3.Outcome, err)
	}
	if r3.OldRevision != 3 || r3.NewRevision != 4 || r3.OldReleaseRevision != 2 || r3.NewReleaseRevision != 3 ||
		len(r3.Keys) != 1 || r3.Keys[0].Before == nil || r3.Keys[0].Before.State != "retired" ||
		r3.Keys[0].After.State != "revoked" || !r3.Keys[0].After.RevokedAt.Equal(revokedAt) ||
		!reflect.DeepEqual(r3.UsableRootIDs, []string{"release-B"}) {
		t.Fatalf("unexpected revoke A result: %+v", r3)
	}
	if _, err := f.p.VerifyRelease(ctx, probeA); err == nil {
		t.Fatal("revoked A accepted")
	}
	probeB := f.sign(map[string]string{"release": "probe-B"}, "release-B")
	if _, err := f.p.VerifyRelease(ctx, probeB); err != nil {
		t.Fatalf("active B rejected: %v", err)
	}
}

func TestUPD02_ReleaseTrustConditionalRepetitionAndPrecondition(t *testing.T) {
	f := newReleaseTestFixture(t)
	ctx := context.Background()
	addB := f.signUpdate(1, "release-A", f.record("release-B", "active"))

	// Wrong expected revision before apply: precondition refused.
	before := manifest(t, f.p.dir)
	for _, wrong := range []decstr.U64{0, 2, 99} {
		r, err := f.p.UpdateReleaseTrustConditional(ctx, addB, wrong, false)
		if err != nil || !reflect.DeepEqual(r, ReleaseUpdateResult{Outcome: ChangeRefused}) {
			t.Fatalf("expected clean ChangeRefused for %d, got %+v err=%v", wrong, r, err)
		}
		unchanged(t, f.p, before)
	}

	// Apply once.
	applied, err := f.p.UpdateReleaseTrustConditional(ctx, addB, 1, false)
	if err != nil || applied.Outcome != ChangeApplied {
		t.Fatalf("apply failed: %+v err=%v", applied, err)
	}

	// Set a distinct prior directory timestamp so any transient file creation is caught.
	prior := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(f.p.dir, prior, prior); err != nil {
		t.Fatal(err)
	}
	before = manifest(t, f.p.dir)

	// Repeating the same update returns ChangeAlreadyApplied regardless of expectedRevision.
	for _, expected := range []decstr.U64{0, 1, 2, 99} {
		r, err := f.p.UpdateReleaseTrustConditional(ctx, addB, expected, false)
		if err != nil || r.Outcome != ChangeAlreadyApplied {
			t.Fatalf("repeat(expected=%d): got %+v err=%v", expected, r, err)
		}
		if r.OldRevision != 2 || r.NewRevision != 2 || r.OldReleaseRevision != 1 || r.NewReleaseRevision != 1 ||
			len(r.Keys) != 1 || r.Keys[0].Before != nil || !equalKeyRecord(r.Keys[0].After, applied.Keys[0].After) ||
			!reflect.DeepEqual(r.UsableRootIDs, []string{"release-A", "release-B"}) {
			t.Fatalf("unexpected already-applied result: %+v", r)
		}
		unchanged(t, f.p, before)
	}

	// An unrelated central rotation advances ledger revision to 3; a new update expecting 2 is refused.
	if _, err := f.p.Rotate(ctx, Lease, time.Hour, 0); err != nil {
		t.Fatal(err)
	}
	before = manifest(t, f.p.dir)
	addC := f.signUpdate(2, "release-A", f.record("release-C", "active"))
	r, err := f.p.UpdateReleaseTrustConditional(ctx, addC, 2, false)
	if err != nil || !reflect.DeepEqual(r, ReleaseUpdateResult{Outcome: ChangeRefused}) {
		t.Fatalf("expected ChangeRefused after unrelated rotation, got %+v err=%v", r, err)
	}
	unchanged(t, f.p, before)
}

func TestUPD02_ReleaseTrustConditionalRejections(t *testing.T) {
	ctx := context.Background()

	type tc struct {
		name   string
		reason ReleaseRejectReason
		build  func(*releaseTestFixture) ([]byte, decstr.U64)
	}
	cases := []tc{
		{
			name:   "unknown_signer",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.signUpdate(1, "release-C", f.record("release-B", "active")), 1
			},
		},
		{
			name:   "expired_signer",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				f.now = f.end
				return f.signUpdate(1, "release-A", f.record("release-B", "active")), 1
			},
		},
		{
			name:   "not_yet_valid_signer",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				f.now = f.start.Add(-time.Second)
				return f.signUpdate(1, "release-A", f.record("release-B", "active")), 1
			},
		},
		{
			name:   "revoked_signer",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if err := f.p.Revoke(ctx, "release-A"); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(1, "release-A", f.record("release-B", "active")), 2
			},
		},
		{
			name:   "other_purpose_signer_only",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.Rotate(ctx, Lease, time.Hour, 0); err != nil {
					t.Fatal(err)
				}
				env, err := f.p.SignEnvelope(ctx, Lease, ReleaseUpdate{
					Kind:     "fvpn-release-trust-v1",
					Revision: 1,
					Keys:     []KeyRecord{f.record("release-B", "active")},
				})
				if err != nil {
					t.Fatal(err)
				}
				raw, err := env.MarshalCanonical()
				if err != nil {
					t.Fatal(err)
				}
				return raw, 2
			},
		},
		{
			name:   "bad_signature",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				raw := f.signUpdate(1, "release-A", f.record("release-B", "active"))
				return bytes.Replace(raw, []byte(`"revision":"1"`), []byte(`"revision":"2"`), 1), 1
			},
		},
		{
			name:   "revoked_second_signature",
			reason: RejectSignature,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				revokedB := f.record("release-B", "revoked")
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(1, "release-A", revokedB), 1, false); err != nil {
					t.Fatal(err)
				}
				raw := f.sign(ReleaseUpdate{
					Kind:     "fvpn-release-trust-v1",
					Revision: 2,
					Keys:     []KeyRecord{f.record("release-C", "active")},
				}, "release-A", "release-B")
				return raw, 2
			},
		},
		{
			name:   "wrong_kind",
			reason: RejectPayload,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.sign(ReleaseUpdate{
					Kind:     "fvpn-unknown-trust-v1",
					Revision: 1,
					Keys:     []KeyRecord{f.record("release-B", "active")},
				}, "release-A"), 1
			},
		},
		{
			name:   "empty_keys",
			reason: RejectPayload,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.sign(ReleaseUpdate{
					Kind:     "fvpn-release-trust-v1",
					Revision: 1,
					Keys:     []KeyRecord{},
				}, "release-A"), 1
			},
		},
		{
			name:   "unknown_payload_field",
			reason: RejectPayload,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.sign(map[string]any{
					"kind":     "fvpn-release-trust-v1",
					"revision": "1",
					"keys":     []KeyRecord{f.record("release-B", "active")},
					"extra":    "rejected",
				}, "release-A"), 1
			},
		},
		{
			name:   "overflow_revision",
			reason: RejectPayload,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.sign(map[string]any{
					"kind":     "fvpn-release-trust-v1",
					"revision": "184467440737099551616"[:20], // 18446744073709551616
					"keys":     []KeyRecord{f.record("release-B", "active")},
				}, "release-A"), 1
			},
		},
		{
			name:   "equal_revision_other_content",
			reason: RejectRevision,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(2, "release-A", f.record("release-B", "active")), 1, false); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(2, "release-A", f.record("release-C", "active")), 2
			},
		},
		{
			name:   "lower_revision",
			reason: RejectRevision,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(2, "release-A", f.record("release-B", "active")), 1, false); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(1, "release-A", f.record("release-C", "active")), 2
			},
		},
		{
			name:   "zero_revision_matching_pinned_root",
			reason: RejectRevision,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.signUpdate(0, "release-A", f.record("release-A", "active")), 1
			},
		},
		{
			name:   "changed_public_key",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-A", "active")
				rec.Public = bytes.Clone(f.keys["release-B"].pub)
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "reused_public_key",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "active")
				rec.Public = bytes.Clone(f.keys["release-A"].pub)
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "duplicate_id_in_update",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				return f.signUpdate(1, "release-A", f.record("release-B", "active"), f.record("release-B", "active")), 1
			},
		},
		{
			name:   "earlier_not_before",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-A", "active")
				rec.NotBefore = f.start.Add(-time.Second)
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "later_not_after",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-A", "active")
				rec.NotAfter = f.end.Add(time.Second)
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "revoked_made_active",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(1, "release-A", f.record("release-B", "revoked")), 1, false); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(2, "release-A", f.record("release-B", "active")), 2
			},
		},
		{
			name:   "revoked_made_retired",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(1, "release-A", f.record("release-B", "revoked")), 1, false); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(2, "release-A", f.record("release-B", "retired")), 2
			},
		},
		{
			name:   "changed_revoked_at",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(1, "release-A", f.record("release-B", "revoked")), 1, false); err != nil {
					t.Fatal(err)
				}
				rec := f.record("release-B", "revoked")
				later := f.now.Add(time.Second)
				rec.RevokedAt = &later
				return f.signUpdate(2, "release-A", rec), 2
			},
		},
		{
			name:   "retired_made_active",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				if _, err := f.p.UpdateReleaseTrustConditional(ctx, f.signUpdate(1, "release-A", f.record("release-B", "retired")), 1, false); err != nil {
					t.Fatal(err)
				}
				return f.signUpdate(2, "release-A", f.record("release-B", "active")), 2
			},
		},
		{
			name:   "wrong_record_purpose",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "active")
				rec.Purpose = Lease
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "existing_other_purpose_id",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				leaseID, err := f.p.Rotate(ctx, Lease, time.Hour, 0)
				if err != nil {
					t.Fatal(err)
				}
				rec := f.record("release-B", "active")
				rec.ID = leaseID
				return f.signUpdate(1, "release-A", rec), 2
			},
		},
		{
			name:   "empty_window",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "active")
				rec.NotAfter = rec.NotBefore
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "unicode_id_over_128_bytes",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "active")
				rec.ID = strings.Repeat("é", 65)
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "revoked_without_revoked_at",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "revoked")
				rec.RevokedAt = nil
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
		{
			name:   "bad_public_key_size",
			reason: RejectRecord,
			build: func(f *releaseTestFixture) ([]byte, decstr.U64) {
				rec := f.record("release-B", "active")
				rec.Public = rec.Public[:16]
				return f.signUpdate(1, "release-A", rec), 1
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newReleaseTestFixture(t)
			raw, expectedRev := tc.build(f)
			before := manifest(t, f.p.dir)
			r, err := f.p.UpdateReleaseTrustConditional(ctx, raw, expectedRev, false)
			want := ReleaseUpdateResult{Outcome: ChangeRejected, RejectReason: tc.reason}
			if err != nil || !reflect.DeepEqual(r, want) {
				t.Fatalf("got %+v err=%v; want %+v", r, err, want)
			}
			unchanged(t, f.p, before)
		})
	}
}

func TestUPD02_ReleaseTrustConditionalUsableRootGuard(t *testing.T) {
	ctx := context.Background()
	for _, op := range []string{"revoke", "retire_expired"} {
		t.Run(op, func(t *testing.T) {
			f := newReleaseTestFixture(t)
			rec := f.record("release-A", "revoked")
			if op == "retire_expired" {
				rec = f.record("release-A", "retired")
				rec.NotAfter = f.now
			}
			raw := f.signUpdate(1, "release-A", rec)
			before := manifest(t, f.p.dir)

			r, err := f.p.UpdateReleaseTrustConditional(ctx, raw, 1, false)
			want := ReleaseUpdateResult{Outcome: ChangeRejected, RejectReason: RejectNoUsableRoot}
			if err != nil || !reflect.DeepEqual(r, want) {
				t.Fatalf("without flag: got %+v err=%v; want %+v", r, err, want)
			}
			unchanged(t, f.p, before)

			r, err = f.p.UpdateReleaseTrustConditional(ctx, raw, 1, true)
			if err != nil || r.Outcome != ChangeApplied || len(r.UsableRootIDs) != 0 || r.UsableRootIDs == nil {
				t.Fatalf("with flag: got %+v err=%v", r, err)
			}
		})
	}
}

func TestUPD02_ReleaseTrustConditionalMultipleRecords(t *testing.T) {
	f := newReleaseTestFixture(t)
	ctx := context.Background()

	retireA := f.record("release-A", "retired")
	retireA.NotAfter = f.now.Add(time.Hour)
	addB := f.record("release-B", "active")
	raw := f.signUpdate(1, "release-A", retireA, addB)

	r, err := f.p.UpdateReleaseTrustConditional(ctx, raw, 1, false)
	if err != nil || r.Outcome != ChangeApplied || len(r.Keys) != 2 {
		t.Fatalf("two-record apply: got %+v err=%v", r, err)
	}
	if r.Keys[0].Before == nil || r.Keys[0].Before.State != "active" || r.Keys[0].After.ID != "release-A" || r.Keys[0].After.State != "retired" ||
		r.Keys[1].Before != nil || r.Keys[1].After.ID != "release-B" || r.Keys[1].After.State != "active" {
		t.Fatalf("unexpected key changes: %+v", r.Keys)
	}
	before := manifest(t, f.p.dir)
	repeat, err := f.p.UpdateReleaseTrustConditional(ctx, raw, 1, false)
	if err != nil || repeat.Outcome != ChangeAlreadyApplied || len(repeat.Keys) != 2 || repeat.Keys[0].Before != nil || repeat.Keys[1].Before != nil {
		t.Fatalf("two-record repeat: got %+v err=%v", repeat, err)
	}
	unchanged(t, f.p, before)
}

func TestUPD02_ReleaseTrustConditionalConcurrency(t *testing.T) {
	f := newReleaseTestFixture(t)
	ctx := context.Background()
	u1 := f.signUpdate(1, "release-A", f.record("release-B", "active"))
	u2 := f.signUpdate(2, "release-A", f.record("release-C", "active"))

	start := make(chan struct{})
	var wg sync.WaitGroup
	results := make(chan ReleaseUpdateResult, 8)
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		h, err := OpenFileProvider(f.p.dir)
		if err != nil {
			t.Fatal(err)
		}
		h.now = f.p.now
		raw := u1
		if i%2 == 1 {
			raw = u2
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			r, err := h.UpdateReleaseTrustConditional(ctx, raw, 1, false)
			results <- r
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	applied := 0
	for r := range results {
		switch r.Outcome {
		case ChangeApplied:
			applied++
		case ChangeRefused, ChangeAlreadyApplied:
		default:
			t.Fatalf("unexpected concurrent outcome: %+v", r)
		}
	}
	s := snap(t, f.p)
	if applied != 1 || s.Revision != 2 || (s.ReleaseRevision != 1 && s.ReleaseRevision != 2) {
		t.Fatalf("applied=%d snap=%+v", applied, s)
	}
}

func TestUPD02_ReleaseTrustConditionalPartialFailure(t *testing.T) {
	for _, stage := range []string{"before rename", "revision max", "after rename"} {
		t.Run(stage, func(t *testing.T) {
			f := newReleaseTestFixture(t)
			ctx := context.Background()
			expectedRev := decstr.U64(1)
			if stage == "revision max" {
				rawState, err := os.ReadFile(filepath.Join(f.p.dir, "state.json"))
				if err != nil {
					t.Fatal(err)
				}
				var st ledger
				if err := json.Unmarshal(rawState, &st); err != nil {
					t.Fatal(err)
				}
				st.Revision = decstr.Max
				if err := f.p.save(st); err != nil {
					t.Fatal(err)
				}
				expectedRev = decstr.Max
			}
			before := manifest(t, f.p.dir)
			injected := errors.New("TEST ONLY injected failure")
			switch stage {
			case "before rename":
				f.p.conditionalHooks.beforeRename = func() error { return injected }
			case "after rename":
				f.p.conditionalHooks.afterRename = func() error { return injected }
			}
			raw := f.signUpdate(1, "release-A", f.record("release-B", "active"))
			r, err := f.p.UpdateReleaseTrustConditional(ctx, raw, expectedRev, false)
			if err != nil {
				t.Fatal(err)
			}
			if stage == "after rename" {
				if r.Outcome != ChangeUncertain || r.NewRevision != 2 || r.NewReleaseRevision != 1 {
					t.Fatalf("expected ChangeUncertain, got %+v", r)
				}
				s := snap(t, f.p)
				if s.Revision != 2 || s.ReleaseRevision != 1 {
					t.Fatalf("replacement missing in snapshot: %+v", s)
				}
				f.p.conditionalHooks = conditionalHooks{}
				again, err := f.p.UpdateReleaseTrustConditional(ctx, raw, expectedRev, false)
				if err != nil || again.Outcome != ChangeAlreadyApplied {
					t.Fatalf("expected ChangeAlreadyApplied after uncertain replacement, got %+v err=%v", again, err)
				}
			} else {
				if r.Outcome != ChangeNotApplied {
					t.Fatalf("expected ChangeNotApplied, got %+v", r)
				}
				if manifest(t, f.p.dir)["state.json"] != before["state.json"] {
					t.Fatal("ledger changed on ChangeNotApplied")
				}
				if stage == "before rename" {
					f.p.conditionalHooks = conditionalHooks{}
					again, err := f.p.UpdateReleaseTrustConditional(ctx, raw, expectedRev, false)
					if err != nil || again.Outcome != ChangeApplied {
						t.Fatalf("retry after beforeRename failure: got %+v err=%v", again, err)
					}
				}
			}
		})
	}
}

func TestUPD02_ReleaseTrustConditionalLockAndDeadline(t *testing.T) {
	f := newReleaseTestFixture(t)
	ctx := context.Background()
	raw := f.signUpdate(1, "release-A", f.record("release-B", "active"))

	// Held lock produces DeadlineExceeded and leaves vault unchanged, then succeeds after unlock.
	lockFile, err := os.OpenFile(filepath.Join(f.p.dir, "lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer lockFile.Close()
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	before := manifest(t, f.p.dir)
	shortCtx, cancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer cancel()
	r, err := f.p.UpdateReleaseTrustConditional(shortCtx, raw, 1, false)
	if !errors.Is(err, context.DeadlineExceeded) || r.Outcome != 0 {
		t.Fatalf("expected DeadlineExceeded, got %+v err=%v", r, err)
	}
	unchanged(t, f.p, before)
	if err := syscall.Flock(int(lockFile.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}

	// Deadline expiring after lock acquisition does not cancel the save.
	saveCtx, saveCancel := context.WithTimeout(ctx, 20*time.Millisecond)
	defer saveCancel()
	f.p.conditionalHooks.locked = func() { <-saveCtx.Done() }
	r, err = f.p.UpdateReleaseTrustConditional(saveCtx, raw, 1, false)
	if err != nil || r.Outcome != ChangeApplied || snap(t, f.p).Revision != 2 {
		t.Fatalf("deadline canceled locked save: %+v err=%v", r, err)
	}

	// Lock validation failure when lock file is replaced while held.
	f2 := newReleaseTestFixture(t)
	f2.p.conditionalHooks.locked = func() {
		p := filepath.Join(f2.p.dir, "lock")
		_ = os.Remove(p)
		_ = os.WriteFile(p, nil, 0600)
	}
	r, err = f2.p.UpdateReleaseTrustConditional(ctx, f2.signUpdate(1, "release-A", f2.record("release-B", "active")), 1, false)
	var lockErr *LockValidationError
	if !errors.As(err, &lockErr) || r.Outcome != 0 {
		t.Fatalf("expected LockValidationError, got %+v err=%v", r, err)
	}
}

func TestUPD02_ReleaseTrustConditionalUmask(t *testing.T) {
	f := newReleaseTestFixture(t)
	raw := f.signUpdate(1, "release-A", f.record("release-B", "active"))
	func() {
		old := syscall.Umask(0777)
		defer syscall.Umask(old)
		r, err := f.p.UpdateReleaseTrustConditional(context.Background(), raw, 1, false)
		if err != nil || r.Outcome != ChangeApplied {
			t.Fatalf("apply under 0777 umask: %+v err=%v", r, err)
		}
	}()
	info, err := os.Stat(filepath.Join(f.p.dir, "state.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("state.json mode = %v, err = %v; want 0600", info.Mode().Perm(), err)
	}
	reopened, err := OpenFileProvider(f.p.dir)
	if err != nil {
		t.Fatalf("cannot reopen vault: %v", err)
	}
	reopened.now = f.p.now
	probe := f.sign(map[string]string{"release": "probe-B"}, "release-B")
	if _, err := reopened.VerifyRelease(context.Background(), probe); err != nil {
		t.Fatalf("reopened vault failed to verify new release root: %v", err)
	}
}
