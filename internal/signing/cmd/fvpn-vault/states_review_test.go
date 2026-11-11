package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestSEC01_RetiredPurposeMetadata(t *testing.T) {
	for _, purpose := range []signing.Purpose{signing.Release, signing.Lease} {
		t.Run(string(purpose), func(t *testing.T) {
			f := newVault(t, false)
			public, private, err := ed25519.GenerateKey(rand.Reader)
			must(t, err)
			root := envelope.TrustedKey{KeyID: "TEST-ONLY-release", PublicKey: public, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
			must(t, f.p.PinReleaseRoot(context.Background(), root))
			f.ids[signing.Release] = root.KeyID
			if purpose == signing.Release {
				snapshot, err := f.p.Snapshot(context.Background())
				must(t, err)
				record := requireRecord(t, snapshot.Keys, root.KeyID)
				record.State = "retired"
				signed, err := envelope.Sign(signing.ReleaseUpdate{Kind: "fvpn-release-trust-v1", Revision: 1, Keys: []signing.KeyRecord{record}}, private, root.KeyID)
				must(t, err)
				raw, err := signed.MarshalCanonical()
				must(t, err)
				must(t, f.p.UpdateReleaseTrust(context.Background(), raw))
			} else {
				// TEST ONLY: Rotate always supplies an active replacement, so the API
				// cannot produce a central purpose consisting solely of retired records.
				editLedger(t, f.dir, func(s *testLedger) {
					count := 0
					for i := range s.Keys {
						if s.Keys[i].Purpose == purpose {
							s.Keys[i].State = "retired"
							count++
						}
					}
					if count != 1 {
						t.Fatal("central retired-only fixture missing")
					}
				})
			}
			snapshot, err := f.p.Snapshot(context.Background())
			must(t, err)
			record := requireRecord(t, snapshot.Keys, f.ids[purpose])
			for _, checked := range []bool{false, true} {
				args := []string{"inspect", "-vault", f.dir}
				if checked {
					args = append(args, "-check-keys")
				}
				got := invoke(t, command{}, f.dir, args...)
				expectedExit := exitOK
				metadata := "pass"
				if purpose == signing.Lease {
					expectedExit, metadata = exitResultFail, "no_active_key"
				}
				requireCode(t, got, expectedExit)
				r := inspectionFrom(t, got)
				k := requireOne(t, r.Keys, "retired key", func(k keySummary) bool { return k.ID == record.ID })
				use := "not_checked"
				if checked {
					use = "not_applicable"
				}
				if k.State != "retired" || k.WindowStatus != "current" || k.KeyUseStatus != use {
					t.Fatal("retired key incorrectly classified")
				}
				summary := requirePurpose(t, r, purpose)
				if summary.MetadataStatus != metadata || summary.KeyUseStatus != use {
					t.Fatal("retired purpose metadata/use mismatch")
				}
			}
			output := filepath.Join(t.TempDir(), "trust.json")
			got := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", output)
			requireCode(t, got, exitOK)
			raw, err := os.ReadFile(output)
			must(t, err)
			requireTrustedRecord(t, raw, record)
			var receipt exportReceipt
			must(t, json.Unmarshal(got.out, &receipt))
			if !requireExportPurpose(t, receipt, purpose).CurrentNonRevokedKey {
				t.Fatal("retired in-window verification key lost from export summary")
			}
		})
	}
}
func TestSEC01_ExportReceiptWindowEndpoints(t *testing.T) {
	f := newVault(t, true)
	snapshot, err := f.p.Snapshot(context.Background())
	must(t, err)
	before := manifest(t, f.dir)
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.Release} {
		record := requireRecord(t, snapshot.Keys, f.ids[purpose])
		for _, tc := range []struct {
			name    string
			at      time.Time
			current bool
		}{
			{"before_start", record.NotBefore.Add(-time.Nanosecond), false},
			{"at_start", record.NotBefore, true},
			{"before_end", record.NotAfter.Add(-time.Nanosecond), true},
			{"at_end", record.NotAfter, false},
		} {
			t.Run(string(purpose)+"/"+tc.name, func(t *testing.T) {
				raw, r, err := prepareExport(snapshot, tc.at, "trust.json")
				must(t, err)
				requireExportInventory(t, r, snapshot.Keys)
				requireTrustedRecord(t, raw, record)
				if requireExportPurpose(t, r, purpose).CurrentNonRevokedKey != tc.current {
					t.Fatal("export window endpoint is not half-open")
				}
			})
		}
	}
	if !reflect.DeepEqual(before, manifest(t, f.dir)) {
		t.Fatal("receipt construction changed vault")
	}
}
func TestSEC01_DenialBeforeSecondSnapshot(t *testing.T) {
	f := newVault(t, true)
	initial, err := f.p.Snapshot(context.Background())
	must(t, err)
	before := manifest(t, f.dir)
	var afterMutation map[string]entry
	c := command{afterProbes: func() {
		if !reflect.DeepEqual(before, manifest(t, f.dir)) {
			t.Fatal("probes changed vault")
		}
		must(t, f.p.DenyIdentity(context.Background(), uuid.NewString()))
		afterMutation = manifest(t, f.dir)
	}}
	var stdout, stderr bytes.Buffer
	code := c.run([]string{"inspect", "-vault", f.dir, "-check-keys"}, &stdout, &stderr)
	requireChanged(t, outcome{code, stdout.Bytes(), stderr.Bytes()})
	after, err := f.p.Snapshot(context.Background())
	must(t, err)
	if after.DeniedIdentityCount != initial.DeniedIdentityCount+1 || after.Revision != initial.Revision+1 {
		t.Fatal("denial fixture not committed")
	}
	if !reflect.DeepEqual(afterMutation, manifest(t, f.dir)) {
		t.Fatal("inspection changed external denial")
	}
}
func TestSEC01_DeadlineBeforeSecondSnapshot(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		f := newVault(t, true)
		var inspectionContext context.Context
		c := command{probeProvider: func(p signing.Provider) signing.Provider {
			return probeOverrides{Provider: p, sign: func(ctx context.Context, purpose signing.Purpose, v any) (envelope.Envelope, error) {
				inspectionContext = ctx
				return p.SignEnvelope(ctx, purpose, v)
			}}
		}, afterProbes: func() {
			if inspectionContext == nil {
				t.Fatal("no real probe context captured")
			}
			watchdog := time.NewTimer(defaultLockTimeout + time.Second)
			defer watchdog.Stop()
			select {
			case <-inspectionContext.Done():
			case <-watchdog.C:
				t.Fatal("command deadline did not expire before second snapshot")
			}
		}}
		got := invoke(t, c, f.dir, "inspect", "-vault", f.dir, "-check-keys")
		requireCode(t, got, exitTimeout)
		if string(got.err) != "fvpn-vault: vault lock deadline exceeded\n" {
			t.Fatal("deadline failure not controlled")
		}
	})
}
