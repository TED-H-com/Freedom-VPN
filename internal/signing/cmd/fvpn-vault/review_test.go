package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"testing/synctest"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

func TestSEC01_LockValidation(t *testing.T) {
	// The FIFO cases run this same test binary as a bounded child process. A
	// regression that opens a FIFO in a blocking mode cannot hang the parent.
	if len(os.Args) > 3 && os.Args[2] == "--" {
		os.Exit(run(os.Args[3:], os.Stdout, os.Stderr))
	}
	for _, fault := range []string{"group_bits", "other_bits", "symlink_inside", "symlink_outside", "fifo", "directory"} {
		t.Run(fault, func(t *testing.T) {
			f := newVault(t, true)
			lock := filepath.Join(f.dir, "lock")
			outside := filepath.Join(t.TempDir(), "outside-lock")
			must(t, os.WriteFile(outside, []byte("TEST ONLY external lock target"), 0600))
			switch fault {
			case "group_bits":
				must(t, os.Chmod(lock, 0660))
			case "other_bits":
				must(t, os.Chmod(lock, 0606))
			case "symlink_inside":
				must(t, os.Rename(lock, filepath.Join(f.dir, "original-lock")))
				must(t, os.Symlink("original-lock", lock))
			case "symlink_outside":
				must(t, os.Remove(lock))
				must(t, os.Symlink(outside, lock))
			case "fifo":
				must(t, os.Remove(lock))
				must(t, syscall.Mkfifo(lock, 0600))
			case "directory":
				must(t, os.Remove(lock))
				must(t, os.Mkdir(lock, 0700))
			}
			externalBefore := manifest(t, outside)
			output := filepath.Join(t.TempDir(), "trust.json")
			for _, args := range commands(f.dir, output) {
				var got outcome
				if fault == "fifo" {
					before := manifest(t, f.dir)
					got = lockChild(t, args)
					if !reflect.DeepEqual(before, manifest(t, f.dir)) {
						t.Fatal("FIFO inspection changed vault")
					}
				} else {
					got = invoke(t, command{}, f.dir, args...)
				}
				requireCode(t, got, exitUnavailable)
				if len(got.out) != 0 || string(got.err) != "fvpn-vault: vault unavailable\n" {
					t.Fatal("lock failure not controlled")
				}
				if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("lock failure created output")
				}
				if !reflect.DeepEqual(externalBefore, manifest(t, outside)) {
					t.Fatal("external lock target changed")
				}
			}
		})
	}
}
func lockChild(t *testing.T, args []string) outcome {
	t.Helper()
	binary, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	child := exec.CommandContext(ctx, binary, append([]string{"-test.run=^TestSEC01_LockValidation$", "--"}, args...)...)
	child.WaitDelay = time.Second
	var stdout, stderr bytes.Buffer
	child.Stdout, child.Stderr = &stdout, &stderr
	err = child.Run()
	if ctx.Err() != nil {
		t.Fatal("FIFO command exceeded bounded subprocess deadline")
	}
	code := 0
	if err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatal(err)
		}
		code = exit.ExitCode()
	}
	return outcome{code, stdout.Bytes(), stderr.Bytes()}
}

func TestSEC01_VaultSymlinkedAncestor(t *testing.T) {
	f := newVault(t, true)
	aliasParent := filepath.Join(t.TempDir(), "storage")
	must(t, os.Symlink(filepath.Dir(f.dir), aliasParent))
	aliasVault := filepath.Join(aliasParent, filepath.Base(f.dir))
	for _, checked := range []bool{false, true} {
		directArgs := []string{"inspect", "-vault", f.dir}
		aliasArgs := []string{"inspect", "-vault", aliasVault}
		if checked {
			directArgs = append(directArgs, "-check-keys")
			aliasArgs = append(aliasArgs, "-check-keys")
		}
		direct := invoke(t, command{}, f.dir, directArgs...)
		aliased := invoke(t, command{}, f.dir, aliasArgs...)
		requireCode(t, direct, exitOK)
		requireCode(t, aliased, exitOK)
		a, b := inspectionFrom(t, direct), inspectionFrom(t, aliased)
		if a.LedgerRevision != b.LedgerRevision || a.ReleaseRevision != b.ReleaseRevision || !reflect.DeepEqual(a.Keys, b.Keys) || !reflect.DeepEqual(a.Purposes, b.Purposes) {
			t.Fatal("ancestor alias changed inspection metadata or fingerprints")
		}
	}
	var direct exportReceipt
	var directBytes []byte
	for i, dir := range []string{f.dir, aliasVault} {
		output := filepath.Join(t.TempDir(), "trust.json")
		got := invoke(t, command{}, f.dir, "export-trust", "-vault", dir, "-out", output)
		requireCode(t, got, exitOK)
		raw, err := os.ReadFile(output)
		must(t, err)
		var receipt exportReceipt
		must(t, json.Unmarshal(got.out, &receipt))
		if i == 0 {
			direct, directBytes = receipt, raw
			continue
		}
		if direct.LedgerRevision != receipt.LedgerRevision || direct.ReleaseRevision != receipt.ReleaseRevision || !reflect.DeepEqual(direct.Keys, receipt.Keys) || !reflect.DeepEqual(direct.Purposes, receipt.Purposes) || !bytes.Equal(directBytes, raw) {
			t.Fatal("ancestor alias changed export metadata, fingerprints or bytes")
		}
	}
	for _, dir := range []string{f.dir, aliasVault} {
		output := filepath.Join(aliasVault, "forbidden.json")
		got := invoke(t, command{}, f.dir, "export-trust", "-vault", dir, "-out", output)
		requireCode(t, got, exitExport)
		if string(got.err) != "fvpn-vault: public trust publication failed\n" {
			t.Fatal("wrong alias destination failure")
		}
		if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("alias placed output in vault")
		}
	}
}

func TestSEC01_WindowExactEndpoints(t *testing.T) {
	f := newVault(t, true)
	snapshot, err := f.p.Snapshot(context.Background())
	must(t, err)
	target := requireRecord(t, snapshot.Keys, f.ids[signing.Lease])
	before := manifest(t, f.dir)
	for _, tc := range []struct {
		name             string
		instant          time.Time
		window, metadata string
	}{
		{"before_start", target.NotBefore.Add(-time.Nanosecond), "not_yet_valid", "not_yet_valid"},
		{"at_start", target.NotBefore, "current", "pass"},
		{"before_end", target.NotAfter.Add(-time.Nanosecond), "current", "pass"},
		{"at_end", target.NotAfter, "expired", "expired"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := inspect(context.Background(), f.p, snapshot, tc.instant, false)
			requireInspectionInventory(t, r, snapshot.Keys)
			key := requireOne(t, r.Keys, "window key", func(k keySummary) bool { return k.ID == target.ID })
			if key.WindowStatus != tc.window || key.KeyUseStatus != "not_checked" {
				t.Fatal("wrong endpoint key status")
			}
			if requirePurpose(t, r, target.Purpose).MetadataStatus != tc.metadata {
				t.Fatal("wrong endpoint purpose status")
			}
		})
	}
	if !reflect.DeepEqual(before, manifest(t, f.dir)) {
		t.Fatal("window classification changed vault")
	}
}

// Override just the provider response under test; all other operations still
// use the real local provider. These are command seams, not provider hooks.
type probeOverrides struct {
	signing.Provider
	sign func(context.Context, signing.Purpose, any) (envelope.Envelope, error)
	ca   func(context.Context, string) (crypto.Signer, error)
	seal func(context.Context, []byte, []byte) (signing.Sealed, error)
	open func(context.Context, signing.Sealed, []byte) ([]byte, error)
}

func (p probeOverrides) SignEnvelope(ctx context.Context, purpose signing.Purpose, v any) (envelope.Envelope, error) {
	if p.sign != nil {
		return p.sign(ctx, purpose, v)
	}
	return p.Provider.SignEnvelope(ctx, purpose, v)
}
func (p probeOverrides) CertificateSigner(ctx context.Context, id string) (crypto.Signer, error) {
	if p.ca != nil {
		return p.ca(ctx, id)
	}
	return p.Provider.CertificateSigner(ctx, id)
}
func (p probeOverrides) Seal(ctx context.Context, plain, aad []byte) (signing.Sealed, error) {
	if p.seal != nil {
		return p.seal(ctx, plain, aad)
	}
	return p.Provider.Seal(ctx, plain, aad)
}
func (p probeOverrides) Open(ctx context.Context, sealed signing.Sealed, aad []byte) ([]byte, error) {
	if p.open != nil {
		return p.open(ctx, sealed, aad)
	}
	return p.Provider.Open(ctx, sealed, aad)
}
func requireUseFailure(t *testing.T, r inspection, purpose signing.Purpose) {
	t.Helper()
	if r.Result != "fail" || !r.KeyUseChecked {
		t.Fatal("inspection failure missing")
	}
	if len(r.Purposes) != len(purposes()) {
		t.Fatal("purpose summaries missing")
	}
	for _, expectedPurpose := range purposes() {
		p := requirePurpose(t, r, expectedPurpose)
		want, result := "passed", "pass"
		if p.Purpose == purpose {
			want, result = "failed", "fail"
		}
		if p.Purpose == signing.Release {
			want = "not_applicable"
		}
		if p.MetadataStatus != "pass" || p.KeyUseStatus != want || p.Result != result {
			t.Fatalf("unexpected %s summary: %+v", p.Purpose, p)
		}
	}
	k := requireOne(t, r.Keys, "failed active key", func(k keySummary) bool { return k.Purpose == purpose && k.State == "active" })
	if k.WindowStatus != "current" || k.KeyUseStatus != "failed" {
		t.Fatal("active key's metadata/use distinction lost")
	}
}
func TestSEC01_ExpiresBetweenSnapshotAndProbe(t *testing.T) {
	// Isolated Go test time removes scheduler/host-clock dependence. All vault
	// IO is still real disk IO; no provider clock setter or host clock change.
	synctest.Test(t, func(t *testing.T) {
		f := newVault(t, true)
		id, err := f.p.Rotate(context.Background(), signing.Lease, time.Second, time.Minute)
		must(t, err)
		snapshot, err := f.p.Snapshot(context.Background())
		must(t, err)
		expires := requireRecord(t, snapshot.Keys, id).NotAfter
		reached, expired := false, false
		c := command{probeProvider: func(p signing.Provider) signing.Provider {
			return probeOverrides{Provider: p, sign: func(ctx context.Context, purpose signing.Purpose, v any) (envelope.Envelope, error) {
				if purpose == signing.Lease {
					reached = true
					if !time.Now().Before(expires) {
						t.Fatal("probe was not synchronized before expiry")
					}
					timer := time.NewTimer(time.Until(expires))
					defer timer.Stop()
					select {
					case <-timer.C:
						expired = !time.Now().Before(expires)
					case <-ctx.Done():
						t.Fatal("bounded expiry synchronization timed out")
					}
				}
				return p.SignEnvelope(ctx, purpose, v)
			}}
		}}
		got := invoke(t, c, f.dir, "inspect", "-vault", f.dir, "-check-keys")
		requireCode(t, got, exitResultFail)
		r := inspectionFrom(t, got)
		requireUseFailure(t, r, signing.Lease)
		if !reached || !expired || !r.InspectedAt.Before(expires) || r.LedgerRevision != snapshot.Revision {
			t.Fatal("expiry did not occur between snapshot and real probe")
		}
	})
}

func TestSEC01_RejectsProbeResponses(t *testing.T) {
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		t.Run(string(purpose)+"/foreign_signature_same_id", func(t *testing.T) {
			f := newVault(t, true)
			// TEST ONLY: a valid signature by an unrelated runtime-generated key,
			// deliberately labelled with the snapshot's ID, must still be rejected.
			wrongPublic, wrongPrivate, err := ed25519.GenerateKey(rand.Reader)
			must(t, err)
			calls := 0
			c := command{probeProvider: func(p signing.Provider) signing.Provider {
				return probeOverrides{Provider: p, sign: func(ctx context.Context, gotPurpose signing.Purpose, v any) (envelope.Envelope, error) {
					if gotPurpose != purpose {
						return p.SignEnvelope(ctx, gotPurpose, v)
					}
					calls++
					e, err := envelope.Sign(v, wrongPrivate, f.ids[purpose])
					must(t, err)
					raw, err := e.MarshalCanonical()
					must(t, err)
					trust, err := envelope.NewTrustStore(envelope.TrustedKey{KeyID: f.ids[purpose], PublicKey: wrongPublic, Purposes: []envelope.Purpose{envelope.Purpose(purpose)}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)})
					must(t, err)
					_, verifiedID, err := envelope.Verify(raw, trust, envelope.Purpose(purpose), time.Now())
					must(t, err)
					if verifiedID != f.ids[purpose] {
						t.Fatal("foreign signature fixture has wrong ID")
					}
					return e, nil
				}}
			}}
			got := invoke(t, c, f.dir, "inspect", "-vault", f.dir, "-check-keys")
			requireCode(t, got, exitResultFail)
			requireUseFailure(t, inspectionFrom(t, got), purpose)
			if calls != 1 {
				t.Fatal("foreign signature seam not exercised exactly once")
			}
			assertNoLeak(t, [][]byte{got.out, got.err}, [][]byte{wrongPrivate, wrongPrivate.Seed()})
		})
	}
	for _, malformed := range []bool{true, false} {
		name := "node-ca/foreign_signature"
		if malformed {
			name = "node-ca/malformed_signature"
		}
		t.Run(name, func(t *testing.T) {
			f := newVault(t, true)
			wrong, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			must(t, err)
			calls := 0
			c := command{probeProvider: func(p signing.Provider) signing.Provider {
				return probeOverrides{Provider: p, ca: func(ctx context.Context, id string) (crypto.Signer, error) {
					real, err := p.CertificateSigner(ctx, id)
					if err != nil {
						return nil, err
					}
					return overriddenSigner{Signer: real, sign: func(_ io.Reader, digest []byte, _ crypto.SignerOpts) ([]byte, error) {
						calls++
						if malformed {
							return []byte("TEST ONLY malformed ASN.1"), nil
						}
						sig, err := ecdsa.SignASN1(rand.Reader, wrong, digest)
						if err == nil && !ecdsa.VerifyASN1(&wrong.PublicKey, digest, sig) {
							t.Fatal("foreign CA fixture is not a valid signature")
						}
						return sig, err
					}}, nil
				}}
			}}
			got := invoke(t, c, f.dir, "inspect", "-vault", f.dir, "-check-keys")
			requireCode(t, got, exitResultFail)
			requireUseFailure(t, inspectionFrom(t, got), signing.NodeCA)
			if calls != 1 {
				t.Fatal("invalid CA response did not reach signature verification")
			}
		})
	}
	t.Run("bootstrap-secret/wrong_key_id", func(t *testing.T) {
		f := newVault(t, true)
		seals, opens := 0, 0
		var original signing.Sealed
		c := command{probeProvider: func(p signing.Provider) signing.Provider {
			return probeOverrides{Provider: p,
				seal: func(ctx context.Context, plain, aad []byte) (signing.Sealed, error) {
					sealed, err := p.Seal(ctx, plain, aad)
					if err != nil {
						return sealed, err
					}
					seals++
					original = sealed
					sealed.KeyID = uuid.NewString()
					return sealed, nil
				},
				// If the command skipped its own ID check, a permissive provider could
				// still return the right plaintext. Do not let provider.Open mask that bug.
				open: func(ctx context.Context, _ signing.Sealed, aad []byte) ([]byte, error) {
					opens++
					return p.Open(ctx, original, aad)
				},
			}
		}}
		got := invoke(t, c, f.dir, "inspect", "-vault", f.dir, "-check-keys")
		requireCode(t, got, exitResultFail)
		requireUseFailure(t, inspectionFrom(t, got), signing.BootstrapSecret)
		if seals != 1 || opens != 0 {
			t.Fatal("command did not reject sealed key ID before Open")
		}
		assertNoLeak(t, [][]byte{got.out, got.err}, [][]byte{original.Ciphertext})
	})
}

type overriddenSigner struct {
	crypto.Signer
	sign func(io.Reader, []byte, crypto.SignerOpts) ([]byte, error)
}

func (s overriddenSigner) Sign(r io.Reader, digest []byte, opts crypto.SignerOpts) ([]byte, error) {
	return s.sign(r, digest, opts)
}

func TestSEC01_ChangedByRevocationOrReleaseUpdate(t *testing.T) {
	for _, change := range []string{"revocation", "signed_release_update"} {
		t.Run(change, func(t *testing.T) {
			f := newVault(t, false)
			// TEST ONLY: retain this runtime-generated root solely to authorize the
			// fixture's real release trust update through the existing provider API.
			public, private, err := ed25519.GenerateKey(rand.Reader)
			must(t, err)
			root := envelope.TrustedKey{KeyID: "TEST-ONLY-release", PublicKey: public, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}
			must(t, f.p.PinReleaseRoot(context.Background(), root))
			initial, err := f.p.Snapshot(context.Background())
			must(t, err)
			before := manifest(t, f.dir)
			var afterMutation map[string]entry
			c := command{afterProbes: func() {
				if !reflect.DeepEqual(before, manifest(t, f.dir)) {
					t.Fatal("probes mutated vault before external change")
				}
				if change == "revocation" {
					must(t, f.p.Revoke(context.Background(), f.ids[signing.Recovery]))
				} else {
					record := requireRecord(t, initial.Keys, root.KeyID)
					record.NotAfter = record.NotAfter.Add(-time.Minute)
					update := signing.ReleaseUpdate{Kind: "fvpn-release-trust-v1", Revision: initial.ReleaseRevision + 1, Keys: []signing.KeyRecord{record}}
					signed, err := envelope.Sign(update, private, root.KeyID)
					must(t, err)
					raw, err := signed.MarshalCanonical()
					must(t, err)
					must(t, f.p.UpdateReleaseTrust(context.Background(), raw))
				}
				afterMutation = manifest(t, f.dir)
			}}
			var stdout, stderr bytes.Buffer
			code := c.run([]string{"inspect", "-vault", f.dir, "-check-keys"}, &stdout, &stderr)
			got := outcome{code, stdout.Bytes(), stderr.Bytes()}
			requireCode(t, got, exitChanged)
			if string(got.err) != "fvpn-vault: changed during inspection\n" {
				t.Fatal("wrong changed outcome")
			}
			if !reflect.DeepEqual(afterMutation, manifest(t, f.dir)) {
				t.Fatal("inspection changed externally mutated vault")
			}
			after, err := f.p.Snapshot(context.Background())
			must(t, err)
			if after.Revision != initial.Revision+1 {
				t.Fatal("external mutation did not advance revision")
			}
			if change == "signed_release_update" && after.ReleaseRevision != initial.ReleaseRevision+1 {
				t.Fatal("signed release update did not advance release revision")
			}
		})
	}
}

func TestSEC01_ExportRestrictiveUmask(t *testing.T) {
	f := newVault(t, true)
	output := filepath.Join(t.TempDir(), "trust.json")
	// Process-global umask: this test is intentionally serial and restores the
	// original value even when an assertion exits the test early.
	previous := syscall.Umask(0777)
	defer syscall.Umask(previous)
	reads := 0
	c := command{publication: publicationOps{read: func(root *os.Root, name string) ([]byte, error) {
		reads++
		return root.ReadFile(name)
	}}}
	got := invoke(t, c, f.dir, "export-trust", "-vault", f.dir, "-out", output)
	requireCode(t, got, exitOK)
	info, err := os.Stat(output)
	must(t, err)
	if info.Mode().Perm() != 0600 || reads != 1 {
		t.Fatal("mode0600 or final-name read-back lost under restrictive umask")
	}
	raw, err := os.ReadFile(output)
	must(t, err)
	_, err = envelope.ParseNodeTrustStoreJSON(raw)
	must(t, err)
	var receipt exportReceipt
	must(t, json.Unmarshal(got.out, &receipt))
	if receipt.SHA256 != digest(raw) {
		t.Fatal("receipt does not match read-back bytes")
	}
}
func TestSEC01_ExportExistingVaultHardLink(t *testing.T) {
	f := newVault(t, true)
	source := filepath.Join(f.dir, f.ids[signing.Lease]+".key")
	output := filepath.Join(t.TempDir(), "trust.json")
	must(t, os.Link(source, output))
	before, err := os.ReadFile(source)
	must(t, err)
	infoBefore, err := os.Stat(source)
	must(t, err)
	linksBefore := infoBefore.Sys().(*syscall.Stat_t).Nlink
	outputBefore := manifest(t, output)
	got := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", output)
	requireCode(t, got, exitExport)
	if string(got.err) != "fvpn-vault: public trust publication failed\n" {
		t.Fatal("hard link refusal not controlled")
	}
	after, err := os.ReadFile(source)
	must(t, err)
	infoAfter, err := os.Stat(source)
	must(t, err)
	if !bytes.Equal(before, after) || infoAfter.Sys().(*syscall.Stat_t).Nlink != linksBefore || !reflect.DeepEqual(outputBefore, manifest(t, output)) {
		t.Fatal("existing vault hard link changed bytes, metadata or link count")
	}
	targetInfo, err := os.Stat(output)
	must(t, err)
	if !os.SameFile(infoAfter, targetInfo) {
		t.Fatal("existing hard link replaced")
	}
}
