package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type fixture struct {
	dir string
	p   *signing.FileProvider
	ids map[signing.Purpose]string
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func newVault(t *testing.T, release bool) fixture {
	t.Helper()
	// TEST ONLY: keys are generated at runtime in t.TempDir on session disk scratch.
	f := fixture{dir: filepath.Join(t.TempDir(), "vault"), ids: map[signing.Purpose]string{}}
	var err error
	f.p, err = signing.InitializeFileProvider(f.dir)
	must(t, err)
	for _, purpose := range purposes() {
		if purpose == signing.Release {
			continue
		}
		f.ids[purpose], err = f.p.Rotate(context.Background(), purpose, time.Hour, time.Minute)
		must(t, err)
	}
	if release {
		pub, _, err := ed25519.GenerateKey(rand.Reader)
		must(t, err)
		f.ids[signing.Release] = "TEST-ONLY-release"
		must(t, f.p.PinReleaseRoot(context.Background(), envelope.TrustedKey{KeyID: f.ids[signing.Release], PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour)}))
	}
	return f
}

type entry struct {
	Size     int64
	Mode     fs.FileMode
	Modified time.Time
	SHA256   string
}

func manifest(t *testing.T, path string) map[string]entry {
	t.Helper()
	entries := map[string]entry{}
	err := filepath.WalkDir(path, func(name string, _ fs.DirEntry, walkErr error) error {
		if errors.Is(walkErr, os.ErrNotExist) {
			return nil
		}
		if walkErr != nil {
			return walkErr
		}
		info, err := os.Lstat(name)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(path, name)
		if err != nil {
			return err
		}
		e := entry{Size: info.Size(), Mode: info.Mode(), Modified: info.ModTime()}
		if info.Mode().IsRegular() {
			raw, err := os.ReadFile(name)
			if err != nil {
				return err
			}
			e.SHA256 = digest(raw)
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(name)
			if err != nil {
				return err
			}
			e.SHA256 = digest([]byte(target))
		}
		entries[rel] = e
		return nil
	})
	must(t, err)
	return entries
}

type outcome struct {
	code     int
	out, err []byte
}

func invoke(t *testing.T, c command, dir string, args ...string) outcome {
	t.Helper()
	before := manifest(t, dir)
	var out, stderr bytes.Buffer
	started := time.Now().UTC()
	code := c.run(args, &out, &stderr)
	ended := time.Now().UTC()
	after := manifest(t, dir)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("command mutated vault manifest")
	}
	if code == exitOK || code == exitResultFail {
		o, ok := parse(args)
		if !ok {
			t.Fatal("successful command had invalid arguments")
		}
		requireReportInventory(t, dir, out.Bytes(), reportInvocation{options: o, started: started, ended: ended})
	}
	return outcome{code, out.Bytes(), stderr.Bytes()}
}
func requireCode(t *testing.T, got outcome, code int) {
	t.Helper()
	if got.code != code {
		t.Fatalf("exit %d, want %d; stderr: %s", got.code, code, got.err)
	}
	if code != exitOK && code != exitResultFail && len(got.out) != 0 {
		t.Fatal("failure printed result document")
	}
	if code == exitOK || code == exitResultFail {
		if len(got.err) != 0 || !json.Valid(got.out) || bytes.Count(got.out, []byte(`"kind"`)) != 1 {
			t.Fatal("expected one JSON document and empty stderr")
		}
	}
}
func inspectionFrom(t *testing.T, got outcome) inspection {
	t.Helper()
	var r inspection
	must(t, json.Unmarshal(got.out, &r))
	return r
}
func TestSEC01_NormalInspectExportAndNoLeakage(t *testing.T) {
	f := newVault(t, true)
	denied := uuid.NewString()
	must(t, f.p.DenyIdentity(context.Background(), denied))
	secrets := privateMaterial(t, f.dir)
	captured := &probeRecorder{}
	c := command{probeProvider: func(p signing.Provider) signing.Provider { captured.Provider = p; return captured }}
	var outputs [][]byte
	for _, check := range []bool{false, true} {
		args := []string{"inspect", "-vault", f.dir}
		if check {
			args = append(args, "-check-keys")
		}
		got := invoke(t, c, f.dir, args...)
		requireCode(t, got, exitOK)
		outputs = append(outputs, got.out, got.err)
		r := inspectionFrom(t, got)
		if r.Kind != "fvpn-file-vault-inspect-v1" || r.Provider != "file" || r.Result != "pass" || r.KeyUseChecked != check || r.DeniedIdentityCount != 1 || len(r.Keys) != 7 || len(r.Purposes) != 7 || r.LedgerRevision != 8 || r.ReleaseRevision != 0 {
			t.Fatal("missing inspection metadata")
		}
		if r.InspectedAt.Location() != time.UTC {
			t.Fatal("timestamp not UTC")
		}
		var prior string
		for _, k := range r.Keys {
			if k.ID <= prior {
				t.Fatal("keys not sorted")
			}
			prior = k.ID
			if k.Purpose == signing.BootstrapSecret && k.PublicSHA256 != "" {
				t.Fatal("bootstrap fingerprint present")
			}
			want := "not_checked"
			if check {
				want = "passed"
				if k.Purpose == signing.Release {
					want = "not_applicable"
				}
			}
			if k.KeyUseStatus != want || k.WindowStatus != "current" {
				t.Fatal("wrong key status")
			}
		}
		for i, p := range r.Purposes {
			if p.Purpose != purposes()[i] || p.Result != "pass" || p.MetadataStatus != "pass" {
				t.Fatal("wrong purpose summary")
			}
		}
	}
	if len(captured.ciphertexts) != 1 || len(captured.signed) != 4 || captured.caCalls != 1 {
		t.Fatal("six real purpose probes not exercised")
	}
	seenProbes := map[string]int{}
	for _, e := range captured.signed {
		var payload map[string]string
		must(t, json.Unmarshal(e.Signed, &payload))
		seenProbes[payload["purpose"]]++
		if payload["kind"] != "fvpn-file-vault-inspect-probe-v1" || len(payload) != 3 {
			t.Fatal("probe not domain separated")
		}
	}
	for _, p := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		if seenProbes[string(p)] != 1 {
			t.Fatal("signing purpose probe missing or duplicated")
		}
	}
	output := filepath.Join(t.TempDir(), "trust.json")
	got := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", output)
	requireCode(t, got, exitOK)
	raw, err := os.ReadFile(output)
	must(t, err)
	ts, err := envelope.ParseNodeTrustStoreJSON(raw)
	must(t, err)
	if ts.Len() != 4 {
		t.Fatal("wrong node purposes")
	}
	bundle, err := f.p.TrustBundle(context.Background())
	must(t, err)
	want, err := json.MarshalIndent(bundle, "", "  ")
	must(t, err)
	if !bytes.Equal(raw, append(want, '\n')) {
		t.Fatal("serialization differs from initializer")
	}
	info, err := os.Stat(output)
	must(t, err)
	if info.Mode().Perm() != 0600 {
		t.Fatal("wrong output mode")
	}
	e, err := f.p.SignEnvelope(context.Background(), signing.DesiredState, map[string]string{"kind": "TEST ONLY"})
	must(t, err)
	signed, err := e.MarshalCanonical()
	must(t, err)
	_, id, err := envelope.Verify(signed, ts, envelope.PurposeDesiredState, time.Now())
	must(t, err)
	if id != f.ids[signing.DesiredState] {
		t.Fatal("export does not verify provider signature")
	}
	var receipt exportReceipt
	must(t, json.Unmarshal(got.out, &receipt))
	if receipt.Kind != "fvpn-file-vault-export-trust-v1" || receipt.KeyUseChecked || receipt.Output != "trust.json" || receipt.SHA256 != digest(raw) || receipt.LedgerRevision != 8 || len(receipt.Keys) != 4 || len(receipt.Purposes) != 4 {
		t.Fatal("invalid receipt")
	}
	for _, key := range receipt.Keys {
		trusted, ok := ts.Lookup(key.ID)
		if !ok || key.PublicSHA256 != digest(trusted.PublicKey) || key.Revoked {
			t.Fatal("receipt key differs")
		}
	}
	for _, p := range receipt.Purposes {
		if !p.CurrentNonRevokedKey {
			t.Fatal("current key not reported")
		}
	}
	outputs = append(outputs, got.out, got.err, raw)
	secrets = append(secrets, captured.ciphertexts...)
	secrets = append(secrets, []byte(denied))
	assertNoLeak(t, outputs, secrets)
}

type probeRecorder struct {
	signing.Provider
	ciphertexts [][]byte
	signed      []envelope.Envelope
	caCalls     int
}

func (p *probeRecorder) Seal(ctx context.Context, plain, aad []byte) (signing.Sealed, error) {
	s, err := p.Provider.Seal(ctx, plain, aad)
	if err == nil {
		p.ciphertexts = append(p.ciphertexts, bytes.Clone(s.Ciphertext))
	}
	return s, err
}
func (p *probeRecorder) SignEnvelope(ctx context.Context, purpose signing.Purpose, v any) (envelope.Envelope, error) {
	e, err := p.Provider.SignEnvelope(ctx, purpose, v)
	if err == nil {
		p.signed = append(p.signed, e)
	}
	return e, err
}
func (p *probeRecorder) CertificateSigner(ctx context.Context, id string) (crypto.Signer, error) {
	p.caCalls++
	return p.Provider.CertificateSigner(ctx, id)
}
func privateMaterial(t *testing.T, dir string) [][]byte {
	t.Helper()
	var result [][]byte
	files, err := filepath.Glob(filepath.Join(dir, "*.key"))
	must(t, err)
	if len(files) == 0 {
		t.Fatal("no private material to scan")
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		must(t, err)
		result = append(result, raw)
		key, err := x509.ParsePKCS8PrivateKey(raw)
		if err != nil {
			continue
		} // Bootstrap bytes were included above.
		switch key := key.(type) {
		case ed25519.PrivateKey:
			result = append(result, []byte(key), key.Seed())
		case *ecdsa.PrivateKey:
			scalar, err := key.Bytes()
			must(t, err)
			result = append(result, scalar)
		}
	}
	return result
}
func assertNoLeak(t *testing.T, outputs, secrets [][]byte) {
	t.Helper()
	if len(outputs) == 0 || len(secrets) == 0 {
		t.Fatal("empty leakage scan")
	}
	for _, secret := range secrets {
		if len(secret) == 0 {
			t.Fatal("empty leakage sentinel")
		}
	}
	for _, out := range outputs {
		for _, secret := range secrets {
			for _, encoded := range [][]byte{secret, []byte(hex.EncodeToString(secret)), []byte(base64.StdEncoding.EncodeToString(secret))} {
				if len(encoded) != 0 && bytes.Contains(out, encoded) {
					t.Fatal("secret or identity leaked")
				}
			}
		}
	}
}
func commands(dir, out string) [][]string {
	return [][]string{{"inspect", "-vault", dir}, {"inspect", "-vault", dir, "-check-keys"}, {"export-trust", "-vault", dir, "-out", out}}
}
func TestSEC01_UnavailableNeverCreatesOrRepairs(t *testing.T) {
	for _, fault := range []string{"missing", "empty", "wrong_path", "missing_lock", "missing_state", "symlink", "directory_permissions", "state_permissions", "invalid_json", "invalid_ledger", "state_symlink"} {
		t.Run(fault, func(t *testing.T) {
			f := newVault(t, true)
			dir := f.dir
			switch fault {
			case "missing":
				dir = filepath.Join(t.TempDir(), "missing")
			case "empty":
				dir = t.TempDir()
			case "wrong_path":
				dir = filepath.Join(t.TempDir(), "file")
				must(t, os.WriteFile(dir, []byte("TEST ONLY"), 0600))
			case "missing_lock":
				must(t, os.Remove(filepath.Join(dir, "lock")))
			case "missing_state":
				must(t, os.Remove(filepath.Join(dir, "state.json")))
			case "symlink":
				dir = filepath.Join(t.TempDir(), "link")
				must(t, os.Symlink(f.dir, dir))
			case "directory_permissions":
				must(t, os.Chmod(dir, 0750))
			case "state_permissions":
				must(t, os.Chmod(filepath.Join(dir, "state.json"), 0640))
			case "invalid_json":
				must(t, os.WriteFile(filepath.Join(dir, "state.json"), []byte("TEST ONLY invalid"), 0600))
			case "invalid_ledger":
				editLedger(t, dir, func(s *testLedger) { s.Keys[0].State = "TEST ONLY invalid" })
			case "state_symlink":
				state := filepath.Join(dir, "state.json")
				must(t, os.Rename(state, filepath.Join(dir, "saved.json")))
				must(t, os.Symlink("saved.json", state))
			}
			original := manifest(t, f.dir)
			output := filepath.Join(t.TempDir(), "new.json")
			for _, args := range commands(dir, output) {
				got := invoke(t, command{}, dir, args...)
				requireCode(t, got, exitUnavailable)
				if string(got.err) != "fvpn-vault: vault unavailable\n" {
					t.Fatal("non-controlled failure")
				}
				if _, err := os.Lstat(output); !errors.Is(err, os.ErrNotExist) {
					t.Fatal("created export on unavailable vault")
				}
			}
			if !reflect.DeepEqual(original, manifest(t, f.dir)) {
				t.Fatal("real vault changed through alias")
			}
		})
	}
}

// Handwritten TEST ONLY ledgers are confined to window/corruption cases the
// provider cannot produce quickly. Other transitions use the public API.
type testLedger struct {
	Revision         decstr.U64          `json:"revision"`
	ReleaseRevision  decstr.U64          `json:"release_revision"`
	Keys             []signing.KeyRecord `json:"keys"`
	DeniedIdentities []string            `json:"denied_identities,omitempty"`
}

func editLedger(t *testing.T, dir string, edit func(*testLedger)) {
	t.Helper()
	path := filepath.Join(dir, "state.json")
	raw, err := os.ReadFile(path)
	must(t, err)
	var s testLedger
	must(t, json.Unmarshal(raw, &s))
	edit(&s)
	raw, err = json.Marshal(s)
	must(t, err)
	must(t, os.WriteFile(path, raw, 0600))
}
func TestSEC01_KeyStatesAndRevocationExport(t *testing.T) {
	for _, state := range []string{"expired", "not_yet_valid", "revoked", "retired", "no_release", "release_revoked", "release_expired", "release_future"} {
		t.Run(state, func(t *testing.T) {
			f := newVault(t, state != "no_release")
			purpose := signing.Lease
			want, result := state, exitResultFail
			oldID := f.ids[purpose]
			switch state {
			case "expired", "not_yet_valid", "release_expired", "release_future":
				if strings.HasPrefix(state, "release_") {
					purpose = signing.Release
					want = "expired"
					if state == "release_future" {
						want = "not_yet_valid"
					}
				}
				editLedger(t, f.dir, func(s *testLedger) {
					changed := 0
					for i := range s.Keys {
						if s.Keys[i].Purpose != purpose {
							continue
						}
						changed++
						now := time.Now().UTC()
						s.Keys[i].NotBefore, s.Keys[i].NotAfter = now.Add(-2*time.Hour), now.Add(-time.Hour)
						if want == "not_yet_valid" {
							s.Keys[i].NotBefore, s.Keys[i].NotAfter = now.Add(time.Hour), now.Add(2*time.Hour)
						}
					}
					if changed != 1 {
						t.Fatal("window fixture key missing or duplicated")
					}
				})
			case "revoked":
				must(t, f.p.Revoke(context.Background(), oldID))
			case "retired":
				_, err := f.p.Rotate(context.Background(), purpose, time.Hour, time.Minute)
				must(t, err)
				want, result = "pass", exitOK
			case "no_release":
				purpose, want = signing.Release, "no_key"
			case "release_revoked":
				purpose, want = signing.Release, "revoked"
				must(t, f.p.Revoke(context.Background(), f.ids[purpose]))
			}
			snapshot, err := f.p.Snapshot(context.Background())
			must(t, err)
			for _, checked := range []bool{false, true} {
				args := []string{"inspect", "-vault", f.dir}
				if checked {
					args = append(args, "-check-keys")
				}
				got := invoke(t, command{}, f.dir, args...)
				requireCode(t, got, result)
				r := inspectionFrom(t, got)
				summary := requirePurpose(t, r, purpose)
				if summary.MetadataStatus != want {
					t.Fatalf("metadata = %s want %s", summary.MetadataStatus, want)
				}
				if state == "retired" {
					key := requireOne(t, r.Keys, "retired overlap key", func(k keySummary) bool { return k.ID == oldID })
					use := "not_checked"
					if checked {
						use = "not_applicable"
					}
					if key.State != "retired" || key.WindowStatus != "current" || key.KeyUseStatus != use {
						t.Fatal("wrong overlap key status")
					}
				}
				for _, k := range r.Keys {
					if !checked && k.KeyUseStatus != "not_checked" {
						t.Fatal("metadata claimed use")
					}
					if checked && (k.State != "active" || k.WindowStatus != "current" || k.Purpose == signing.Release) && k.KeyUseStatus != "not_applicable" {
						t.Fatal("ineligible key exercised")
					}
				}
			}
			output := filepath.Join(t.TempDir(), "trust.json")
			got := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", output)
			requireCode(t, got, exitOK)
			raw, err := os.ReadFile(output)
			must(t, err)
			bundle, err := envelope.ParseNodeTrustStoreJSON(raw)
			must(t, err)
			if state == "revoked" || state == "release_revoked" {
				k, ok := bundle.Lookup(f.ids[purpose])
				if !ok || !k.Revoked {
					t.Fatal("revocation lost")
				}
			}
			var r exportReceipt
			must(t, json.Unmarshal(got.out, &r))
			if requireExportPurpose(t, r, purpose).CurrentNonRevokedKey != (state == "retired") {
				t.Fatal("wrong export window summary")
			}
			if state == "retired" {
				requireTrustedRecord(t, raw, requireRecord(t, snapshot.Keys, oldID))
			}
		})
	}
}
func TestSEC01_MaterialFaultsStillCheckOtherPurposes(t *testing.T) {
	for _, purpose := range purposes() {
		if purpose == signing.Release {
			continue
		}
		for _, fault := range []string{"missing", "mismatch", "permissions"} {
			t.Run(string(purpose)+"/"+fault, func(t *testing.T) {
				f := newVault(t, true)
				path := filepath.Join(f.dir, f.ids[purpose]+".key")
				secrets := privateMaterial(t, f.dir)
				switch fault {
				case "missing":
					must(t, os.Remove(path))
				case "permissions":
					must(t, os.Chmod(path, 0644))
				case "mismatch":
					other := signing.DesiredState
					if purpose == other {
						other = signing.Lease
					}
					raw, err := os.ReadFile(filepath.Join(f.dir, f.ids[other]+".key"))
					must(t, err)
					must(t, os.WriteFile(path, raw, 0600))
				}
				meta := invoke(t, command{}, f.dir, "inspect", "-vault", f.dir)
				requireCode(t, meta, exitOK)
				for _, k := range inspectionFrom(t, meta).Keys {
					if k.KeyUseStatus != "not_checked" {
						t.Fatal("metadata claimed key use")
					}
				}
				got := invoke(t, command{}, f.dir, "inspect", "-vault", f.dir, "-check-keys")
				requireCode(t, got, exitResultFail)
				r := inspectionFrom(t, got)
				for _, k := range r.Keys {
					want := "passed"
					if k.Purpose == purpose {
						want = "failed"
					}
					if k.Purpose == signing.Release {
						want = "not_applicable"
					}
					if k.KeyUseStatus != want {
						t.Fatalf("purpose %s: %s want %s", k.Purpose, k.KeyUseStatus, want)
					}
				}
				output := filepath.Join(t.TempDir(), "trust.json")
				exported := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", output)
				requireCode(t, exported, exitOK)
				raw, err := os.ReadFile(output)
				must(t, err)
				assertNoLeak(t, [][]byte{meta.out, meta.err, got.out, got.err, exported.out, exported.err, raw}, secrets)
			})
		}
	}
}
func TestSEC01_LockDeadlineAndRecovery(t *testing.T) {
	f := newVault(t, true)
	lock, err := os.OpenFile(filepath.Join(f.dir, "lock"), os.O_RDWR, 0)
	must(t, err)
	defer lock.Close()
	for _, args := range commands(f.dir, filepath.Join(t.TempDir(), "trust.json")) {
		must(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
		start := time.Now()
		got := invoke(t, command{}, f.dir, append(args, "-lock-timeout", "20ms")...)
		must(t, syscall.Flock(int(lock.Fd()), syscall.LOCK_UN))
		requireCode(t, got, exitTimeout)
		if time.Since(start) > 2*time.Second || string(got.err) != "fvpn-vault: vault lock deadline exceeded\n" {
			t.Fatal("lock not bounded or timeout not distinguished")
		}
		requireCode(t, invoke(t, command{}, f.dir, args...), exitOK)
	}
}
func TestSEC01_ChangedDuringInspection(t *testing.T) {
	f := newVault(t, true)
	before := manifest(t, f.dir)
	var afterMutation map[string]entry
	c := command{afterProbes: func() {
		if !reflect.DeepEqual(before, manifest(t, f.dir)) {
			t.Fatal("probes mutated vault")
		}
		_, err := f.p.Rotate(context.Background(), signing.Lease, time.Hour, time.Minute)
		must(t, err)
		afterMutation = manifest(t, f.dir)
	}}
	var out, stderr bytes.Buffer
	code := c.run([]string{"inspect", "-vault", f.dir, "-check-keys"}, &out, &stderr)
	requireCode(t, outcome{code, out.Bytes(), stderr.Bytes()}, exitChanged)
	if stderr.String() != "fvpn-vault: changed during inspection\n" {
		t.Fatal("wrong change failure")
	}
	if !reflect.DeepEqual(afterMutation, manifest(t, f.dir)) {
		t.Fatal("inspection changed externally rotated vault")
	}
}
func TestSEC01_ExportDestinations(t *testing.T) {
	for _, fault := range []string{"file", "directory", "symlink", "dangling_symlink", "missing_parent", "unwritable_parent", "vault", "below_vault", "symlink_to_vault", "symlink_below_vault"} {
		t.Run(fault, func(t *testing.T) {
			f := newVault(t, true)
			parent := t.TempDir()
			out := filepath.Join(parent, "trust.json")
			target := filepath.Join(parent, "target.json")
			must(t, os.WriteFile(target, []byte("TEST ONLY existing target"), 0600))
			targetBefore := manifest(t, target)
			switch fault {
			case "file":
				must(t, os.WriteFile(out, []byte("TEST ONLY existing file"), 0640))
			case "directory":
				must(t, os.Mkdir(out, 0700))
			case "symlink":
				must(t, os.Symlink(target, out))
			case "dangling_symlink":
				must(t, os.Symlink(filepath.Join(parent, "missing"), out))
			case "missing_parent":
				out = filepath.Join(parent, "missing", "trust.json")
			case "unwritable_parent":
				if os.Geteuid() == 0 {
					t.Skip("permission refusal requires unprivileged execution")
				}
				must(t, os.Chmod(parent, 0500))
				t.Cleanup(func() { _ = os.Chmod(parent, 0700) })
			case "vault":
				out = filepath.Join(f.dir, "trust.json")
			case "below_vault":
				must(t, os.Mkdir(filepath.Join(f.dir, "child"), 0700))
				out = filepath.Join(f.dir, "child", "trust.json")
			case "symlink_to_vault", "symlink_below_vault":
				dest := f.dir
				if fault == "symlink_below_vault" {
					dest = filepath.Join(f.dir, "child")
					must(t, os.Mkdir(dest, 0700))
				}
				link := filepath.Join(parent, "alias")
				must(t, os.Symlink(dest, link))
				out = filepath.Join(link, "trust.json")
			}
			outputBefore := manifest(t, out)
			got := invoke(t, command{}, f.dir, "export-trust", "-vault", f.dir, "-out", out)
			requireCode(t, got, exitExport)
			if !reflect.DeepEqual(outputBefore, manifest(t, out)) || !reflect.DeepEqual(targetBefore, manifest(t, target)) {
				t.Fatal("existing target changed or final name created")
			}
		})
	}
}
func TestSEC01_ExportPublicationFaults(t *testing.T) {
	injected := errors.New("TEST ONLY forbidden raw error detail")
	for _, fault := range []string{"write", "short_write", "sync", "close", "link", "read", "read_mismatch"} {
		t.Run(fault, func(t *testing.T) {
			f := newVault(t, true)
			out := filepath.Join(t.TempDir(), "trust.json")
			ops := publicationOps{}
			switch fault {
			case "write":
				ops.write = func(f *os.File, b []byte) (int, error) { _, _ = f.Write(b[:len(b)/2]); return 0, injected }
			case "short_write":
				ops.write = func(f *os.File, b []byte) (int, error) { return f.Write(b[:len(b)/2]) }
			case "sync":
				ops.sync = func(*os.File) error { return injected }
			case "close":
				ops.close = func(f *os.File) error { _ = f.Close(); return injected }
			case "link":
				ops.link = func(*os.Root, string, string) error { return injected }
			case "read":
				ops.read = func(*os.Root, string) ([]byte, error) { return nil, injected }
			case "read_mismatch":
				ops.read = func(*os.Root, string) ([]byte, error) { return []byte("TEST ONLY mismatch"), nil }
			}
			got := invoke(t, command{publication: ops}, f.dir, "export-trust", "-vault", f.dir, "-out", out)
			requireCode(t, got, exitExport)
			if string(got.err) != "fvpn-vault: public trust publication failed\n" {
				t.Fatal("raw fault leaked")
			}
			files, err := os.ReadDir(filepath.Dir(out))
			must(t, err)
			if len(files) != 0 {
				t.Fatal("failure left final or temporary file")
			}
		})
	}
}

type failingWriter struct{ short bool }

func (w failingWriter) Write(b []byte) (int, error) {
	if w.short {
		return len(b) / 2, nil
	}
	return 0, errors.New("TEST ONLY stdout secret")
}
func TestSEC01_StdoutFailureAfterPublication(t *testing.T) {
	f := newVault(t, true)
	for _, short := range []bool{false, true} {
		for _, subcommand := range []string{"inspect", "export-trust"} {
			out := filepath.Join(t.TempDir(), "trust.json")
			args := []string{subcommand, "-vault", f.dir}
			if subcommand == "export-trust" {
				args = append(args, "-out", out)
			}
			before := manifest(t, f.dir)
			var stderr bytes.Buffer
			if code := (command{}).run(args, failingWriter{short}, &stderr); code != exitOutput {
				t.Fatalf("exit %d", code)
			}
			if !reflect.DeepEqual(before, manifest(t, f.dir)) {
				t.Fatal("stdout failure mutated vault")
			}
			if subcommand == "export-trust" {
				raw, err := os.ReadFile(out)
				must(t, err)
				_, err = envelope.ParseNodeTrustStoreJSON(raw)
				must(t, err)
				if stderr.String() != "fvpn-vault: output failed; published file remains\n" {
					t.Fatal("publication status missing")
				}
			} else if stderr.String() != "fvpn-vault: output failed\n" {
				t.Fatal("wrong stdout failure")
			}
		}
	}
}
func TestSEC01_ArgumentsTouchNothing(t *testing.T) {
	f := newVault(t, true)
	out := filepath.Join(t.TempDir(), "trust.json")
	cases := [][]string{
		{}, {"unknown"}, {"inspect"}, {"export-trust"},
		{"inspect", "-vault"}, {"inspect", "-vault", ""},
		{"inspect", "-vault", f.dir, "-vault", f.dir},
		{"inspect", "-vault", f.dir, "--vault=" + f.dir},
		{"inspect", "-vault", f.dir, "extra"},
		{"inspect", "-vault", f.dir, "-out", out},
		{"inspect", "-vault", f.dir, "-check-keys", "-check-keys"},
		{"inspect", "-vault", f.dir, "-check-keys=bad"},
		{"inspect", "-vault", f.dir, "-lock-timeout", "1s", "-lock-timeout", "2s"},
		{"export-trust", "-vault", f.dir}, {"export-trust", "-out", out},
		{"export-trust", "-vault", f.dir, "-out", out, "-out", out},
		{"export-trust", "-vault", f.dir, "-out", out, "-check-keys"},
		{"export-trust", "-vault", f.dir, "-out", out, "extra"},
	}
	for _, duration := range []string{"bad", "0", "0s", "-1s", "999999999999999999999h"} {
		for _, args := range commands(f.dir, out) {
			cases = append(cases, append(args, "-lock-timeout", duration))
		}
	}
	for _, args := range cases {
		requireCode(t, invoke(t, command{}, f.dir, args...), exitUsage)
		if _, err := os.Lstat(out); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("usage error created output")
		}
	}
	if o, ok := parse([]string{"inspect", "-vault", f.dir + "/.", "-check-keys=false"}); !ok || o.vault != f.dir || o.checkKeys || o.timeout != 5*time.Second {
		t.Fatal("defaults or cleaning changed")
	}
}
func TestSEC01_EmptyVaultAndEmptyNodeBundle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "vault")
	_, err := signing.InitializeFileProvider(dir)
	must(t, err)
	for _, args := range commands(dir, filepath.Join(t.TempDir(), "trust.json")) {
		got := invoke(t, command{}, dir, args...)
		code := exitResultFail
		if args[0] == "export-trust" {
			code = exitExport
		}
		requireCode(t, got, code)
		if code == exitResultFail {
			for _, p := range inspectionFrom(t, got).Purposes {
				want := "no_active_key"
				if p.Purpose == signing.Release {
					want = "no_key"
				}
				if p.MetadataStatus != want {
					t.Fatal("empty purpose misreported")
				}
			}
		}
	}
}
func TestSEC01_MetadataAndExportNeverReadPrivateFiles(t *testing.T) {
	f := newVault(t, true)
	// FIFOs would block an attempted private-file read. Provider metadata and
	// export must succeed without opening them; checked inspection rejects them.
	for _, id := range f.ids {
		if id == f.ids[signing.Release] {
			continue
		}
		path := filepath.Join(f.dir, id+".key")
		must(t, os.Remove(path))
		must(t, syscall.Mkfifo(path, 0600))
	}
	for _, args := range commands(f.dir, filepath.Join(t.TempDir(), "trust.json")) {
		want := exitOK
		if len(args) > 3 && args[3] == "-check-keys" {
			want = exitResultFail
		}
		requireCode(t, invoke(t, command{}, f.dir, args...), want)
	}
}
func TestSEC01_ExportExclusiveLinkPreservesConcurrentTarget(t *testing.T) {
	f := newVault(t, true)
	out := filepath.Join(t.TempDir(), "trust.json")
	existing := []byte("TEST ONLY concurrent target")
	c := command{publication: publicationOps{link: func(root *os.Root, temp, name string) error {
		must(t, root.WriteFile(name, existing, 0600))
		return root.Link(temp, name)
	}}}
	requireCode(t, invoke(t, c, f.dir, "export-trust", "-vault", f.dir, "-out", out), exitExport)
	got, err := os.ReadFile(out)
	must(t, err)
	if !bytes.Equal(got, existing) {
		t.Fatal("concurrent target modified")
	}
}

var _ io.Writer = failingWriter{}
