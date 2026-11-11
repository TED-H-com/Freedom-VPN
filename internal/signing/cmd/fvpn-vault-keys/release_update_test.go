package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

// TEST ONLY - NOT A SECRET. Keys in this file are generated per test or taken
// from the public golden vector in contracts/vectors/.
var (
	releaseTestStart     = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	releaseTestEnd       = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
	releaseTestNarrowEnd = time.Date(2090, 1, 1, 0, 0, 0, 0, time.UTC)
	releaseTestAt        = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
)

type releaseKey struct {
	id         string
	signingKey release.SigningKey
	pub        ed25519.PublicKey
	priv       ed25519.PrivateKey
}

type releaseFixture struct {
	t       *testing.T
	root    string
	dir     string
	p       *signing.FileProvider
	a, b, c releaseKey
	update  string
	sha256  string
}

func genReleaseKey(t *testing.T) releaseKey {
	t.Helper()
	id := uuid.NewString()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := release.KeyFromSeed(id, priv.Seed())
	if err != nil {
		t.Fatal(err)
	}
	return releaseKey{id: id, signingKey: sk, pub: pub, priv: priv}
}

func (k releaseKey) trusted(notBefore, notAfter time.Time) envelope.TrustedKey {
	return envelope.TrustedKey{
		KeyID:     k.id,
		PublicKey: append(ed25519.PublicKey(nil), k.pub...),
		Purposes:  []envelope.Purpose{envelope.PurposeRelease},
		NotBefore: notBefore,
		NotAfter:  notAfter,
	}
}

func (k releaseKey) oneKeyTrustJSON(t *testing.T, notBefore, notAfter time.Time) []byte {
	t.Helper()
	ts, err := envelope.NewNodeTrustStore(k.trusted(notBefore, notAfter))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func (k releaseKey) record(state string, notBefore, notAfter time.Time, revokedAt *time.Time) signing.KeyRecord {
	var rev *time.Time
	if revokedAt != nil {
		at := revokedAt.UTC()
		rev = &at
	}
	return signing.KeyRecord{
		ID:        k.id,
		Purpose:   signing.Release,
		Public:    append([]byte(nil), k.pub...),
		State:     state,
		NotBefore: notBefore.UTC(),
		NotAfter:  notAfter.UTC(),
		RevokedAt: rev,
	}
}

func signReleaseEnvelope(t *testing.T, signer releaseKey, payload any) []byte {
	t.Helper()
	e, err := envelope.Sign(payload, signer.priv, signer.id)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := e.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	return append(raw, '\n')
}

func setupRelease(t *testing.T) *releaseFixture {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "vault")
	p, err := signing.InitializeFileProvider(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := genReleaseKey(t)
	b := genReleaseKey(t)
	c := genReleaseKey(t)
	if err := p.PinReleaseRoot(context.Background(), a.trusted(releaseTestStart, releaseTestEnd)); err != nil {
		t.Fatal(err)
	}
	ts, err := p.TrustBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	trustBytes, err := json.Marshal(ts)
	if err != nil {
		t.Fatal(err)
	}
	trust, err := release.ParseReleaseTrust(trustBytes)
	if err != nil {
		t.Fatal(err)
	}
	raw, res, err := release.BuildTrustUpdate(a.signingKey, trust, release.TrustUpdateOptions{
		Revision:  1,
		Operation: "add",
		AddTrust:  b.oneKeyTrustJSON(t, releaseTestStart, releaseTestEnd),
		At:        releaseTestAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	updatePath := filepath.Join(root, "update.json")
	if err := os.WriteFile(updatePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	return &releaseFixture{
		t:      t,
		root:   root,
		dir:    dir,
		p:      p,
		a:      a,
		b:      b,
		c:      c,
		update: updatePath,
		sha256: res.UpdateSHA256,
	}
}

func (f *releaseFixture) args() []string {
	return []string{"apply-release-update", "-vault", f.dir, "-update", f.update, "-expect-revision", "1"}
}

func (f *releaseFixture) writeUpdate(name string, raw []byte) (string, string) {
	f.t.Helper()
	path := filepath.Join(f.root, name)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		f.t.Fatal(err)
	}
	sum, _ := release.EnvelopeSHA256(raw)
	return path, sum
}

func assertReleaseOutput(t *testing.T, code int, stdout, stderr string, wantCode int, wantOutcome, wantStderr string) document {
	t.Helper()
	if code != wantCode || stderr != wantStderr {
		t.Fatalf("code=%d stderr=%q; want %d %q; stdout=%s", code, stderr, wantCode, wantStderr, stdout)
	}
	var d document
	if wantOutcome == "" {
		if stdout != "" {
			t.Fatalf("unexpected document: %s", stdout)
		}
		return d
	}
	decoder := json.NewDecoder(strings.NewReader(stdout))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		t.Fatalf("decode document: %v\nstdout=%s", err, stdout)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		t.Fatal("more than one document")
	}
	if d.Kind != "fvpn-file-vault-apply-release-update-v1" || d.Outcome != wantOutcome || d.Message == "" {
		t.Fatalf("unexpected document: %+v", d)
	}
	return d
}

func invokeRelease(t *testing.T, c command, args []string, code int, outcome string) document {
	t.Helper()
	var out, err bytes.Buffer
	got := c.run(args, &out, &err)
	stderr := ""
	switch code {
	case exitUsage:
		stderr = "fvpn-vault-keys: " + usage + "\n"
	case exitUnavailable:
		stderr = "fvpn-vault-keys: vault unavailable\n"
	case exitTimeout:
		stderr = "fvpn-vault-keys: vault lock deadline exceeded\n"
	}
	return assertReleaseOutput(t, got, out.String(), err.String(), code, outcome, stderr)
}

func buildVaultCommand(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "fvpn-vault")
	cmd := exec.Command("go", "build", "-o", binary, "../fvpn-vault")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build fvpn-vault: %v\n%s", err, out)
	}
	return binary
}

type inspectOutput struct {
	Kind            string     `json:"kind"`
	LedgerRevision  decstr.U64 `json:"ledger_revision"`
	ReleaseRevision decstr.U64 `json:"release_revision"`
	Result          string     `json:"result"`
}

func runVaultInspect(t *testing.T, vaultBinary, vaultDir string) inspectOutput {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(vaultBinary, "inspect", "-vault", vaultDir)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := processCode(t, err)
	if code != 0 && code != 1 {
		t.Fatalf("fvpn-vault inspect exit %d: %s", code, stderr.String())
	}
	var out inspectOutput
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("decode inspect: %v\n%s", err, stdout.String())
	}
	return out
}

func runVaultExportTrust(t *testing.T, vaultBinary, vaultDir, outPath string) (*envelope.TrustStore, *release.ReleaseTrust) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(vaultBinary, "export-trust", "-vault", vaultDir, "-out", outPath)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("fvpn-vault export-trust failed: %v\n%s", err, stderr.String())
	}
	raw, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	ts, err := envelope.ParseNodeTrustStoreJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	rt, err := release.ParseReleaseTrust(raw)
	if err != nil {
		t.Fatal(err)
	}
	return ts, rt
}

func TestUPD02_CommandReleaseTrustHandover(t *testing.T) {
	vaultBinary := buildVaultCommand(t)

	t.Run("golden vector", func(t *testing.T) {
		var vector struct {
			SeedHex        string                     `json:"seed_hex"`
			KeyID          string                     `json:"key_id"`
			Envelope       json.RawMessage            `json:"envelope"`
			EnvelopeSHA256 string                     `json:"envelope_sha256"`
			Signed         release.ReleaseTrustUpdate `json:"signed"`
		}
		rawVec, err := os.ReadFile("../../../../contracts/vectors/release-trust-update-signing.json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(rawVec, &vector); err != nil {
			t.Fatal(err)
		}
		seed, err := hex.DecodeString(vector.SeedHex)
		if err != nil {
			t.Fatal(err)
		}
		key, err := release.KeyFromSeed(vector.KeyID, seed)
		if err != nil {
			t.Fatal(err)
		}
		root := t.TempDir()
		vaultDir := filepath.Join(root, "vault")
		p, err := signing.InitializeFileProvider(vaultDir)
		if err != nil {
			t.Fatal(err)
		}
		if err := p.PinReleaseRoot(context.Background(), envelope.TrustedKey{
			KeyID:     vector.KeyID,
			PublicKey: key.Public(),
			Purposes:  []envelope.Purpose{envelope.PurposeRelease},
			NotBefore: releaseTestStart,
			NotAfter:  releaseTestEnd,
		}); err != nil {
			t.Fatal(err)
		}
		canonical, err := canonjson.Canonicalize(vector.Envelope)
		if err != nil {
			t.Fatal(err)
		}
		updatePath := filepath.Join(root, "golden-update.json")
		if err := os.WriteFile(updatePath, append(canonical, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
		insp := runVaultInspect(t, vaultBinary, vaultDir)
		if insp.LedgerRevision != 1 || insp.ReleaseRevision != 0 {
			t.Fatalf("initial inspect = %+v", insp)
		}
		d := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", vaultDir, "-update", updatePath, "-expect-revision", "1"}, exitOK, "applied")
		if d.UpdateSHA256 != vector.EnvelopeSHA256 || d.Change == nil || d.Change.OldRevision != 1 || d.Change.NewRevision != 2 || d.Change.OldReleaseRevision != 0 || d.Change.NewReleaseRevision != vector.Signed.Revision || len(d.Change.Keys) != 1 || d.Change.Keys[0].Previous != nil || d.Change.Keys[0].Key.ID != vector.Signed.Keys[0].ID || d.Change.NodeTrust != "nodes_require_new_bundle" {
			t.Fatalf("unexpected golden apply document: %+v", d)
		}
		insp = runVaultInspect(t, vaultBinary, vaultDir)
		if insp.LedgerRevision != 2 || insp.ReleaseRevision != vector.Signed.Revision {
			t.Fatalf("after golden inspect = %+v", insp)
		}
		ts, _ := runVaultExportTrust(t, vaultBinary, vaultDir, filepath.Join(root, "golden.trust"))
		if ts.Len() != 2 {
			t.Fatalf("exported keys = %d; want 2", ts.Len())
		}
	})

	t.Run("add retire revoke handover", func(t *testing.T) {
		f := setupRelease(t)
		insp := runVaultInspect(t, vaultBinary, f.dir)
		if insp.LedgerRevision != 1 || insp.ReleaseRevision != 0 {
			t.Fatalf("step 0 inspect = %+v", insp)
		}
		_, rt0 := runVaultExportTrust(t, vaultBinary, f.dir, filepath.Join(f.root, "step0.trust"))
		rev0 := decstr.U64(0)
		step1Bytes, err := os.ReadFile(f.update)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := release.VerifyTrustUpdate(step1Bytes, rt0, releaseTestAt, &rev0); err != nil {
			t.Fatalf("step 1 offline verify: %v", err)
		}

		// Step 1: Add root B signed by A.
		d1 := invokeRelease(t, command{}, f.args(), exitOK, "applied")
		wantRoots := []string{f.a.id, f.b.id}
		sort.Strings(wantRoots)
		if d1.UpdateSHA256 != f.sha256 || d1.Change == nil || d1.Change.OldRevision != 1 || d1.Change.NewRevision != 2 || d1.Change.OldReleaseRevision != 0 || d1.Change.NewReleaseRevision != 1 || len(d1.Change.Keys) != 1 || d1.Change.Keys[0].Previous != nil || d1.Change.Keys[0].Key.ID != f.b.id || d1.Change.Keys[0].Key.State != "active" || !reflect.DeepEqual(d1.Change.UsableRootKeyIDs, wantRoots) || d1.Change.NodeTrust != "nodes_require_new_bundle" {
			t.Fatalf("step 1 document = %+v", d1)
		}
		insp = runVaultInspect(t, vaultBinary, f.dir)
		if insp.LedgerRevision != 2 || insp.ReleaseRevision != 1 {
			t.Fatalf("step 1 inspect = %+v", insp)
		}
		ts1, rt1 := runVaultExportTrust(t, vaultBinary, f.dir, filepath.Join(f.root, "step1.trust"))
		for _, id := range []string{f.a.id, f.b.id} {
			if k, ok := ts1.Lookup(id); !ok || k.Revoked {
				t.Fatalf("step 1 exported bundle missing or revoked %s", id)
			}
		}

		// Step 2: Retire root A signed by B.
		step2Raw, step2Res, err := release.BuildTrustUpdate(f.b.signingKey, rt1, release.TrustUpdateOptions{
			Revision:    2,
			Operation:   "retire",
			TargetKeyID: f.a.id,
			NotAfter:    releaseTestNarrowEnd,
			At:          releaseTestAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		rev1 := decstr.U64(1)
		if _, err := release.VerifyTrustUpdate(step2Raw, rt1, releaseTestAt, &rev1); err != nil {
			t.Fatalf("step 2 offline verify: %v", err)
		}
		step2Path, _ := f.writeUpdate("step2.json", step2Raw)
		d2 := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", step2Path, "-expect-revision", "2"}, exitOK, "applied")
		if d2.UpdateSHA256 != step2Res.UpdateSHA256 || d2.Change == nil || d2.Change.OldRevision != 2 || d2.Change.NewRevision != 3 || d2.Change.OldReleaseRevision != 1 || d2.Change.NewReleaseRevision != 2 || len(d2.Change.Keys) != 1 || d2.Change.Keys[0].Previous == nil || d2.Change.Keys[0].Previous.State != "active" || d2.Change.Keys[0].Key.ID != f.a.id || d2.Change.Keys[0].Key.State != "retired" || !d2.Change.Keys[0].Key.NotAfter.Equal(releaseTestNarrowEnd) || !reflect.DeepEqual(d2.Change.UsableRootKeyIDs, wantRoots) || d2.Change.NodeTrust != "nodes_require_new_bundle" {
			t.Fatalf("step 2 document = %+v", d2)
		}
		insp = runVaultInspect(t, vaultBinary, f.dir)
		if insp.LedgerRevision != 3 || insp.ReleaseRevision != 2 {
			t.Fatalf("step 2 inspect = %+v", insp)
		}
		ts2, rt2 := runVaultExportTrust(t, vaultBinary, f.dir, filepath.Join(f.root, "step2.trust"))
		if kA, ok := ts2.Lookup(f.a.id); !ok || kA.Revoked || !kA.NotAfter.Equal(releaseTestNarrowEnd) {
			t.Fatalf("step 2 bundle A = %+v", kA)
		}

		// Step 3: Revoke root A signed by B.
		step3Raw, step3Res, err := release.BuildTrustUpdate(f.b.signingKey, rt2, release.TrustUpdateOptions{
			Revision:    3,
			Operation:   "revoke",
			TargetKeyID: f.a.id,
			At:          releaseTestAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		rev2 := decstr.U64(2)
		if _, err := release.VerifyTrustUpdate(step3Raw, rt2, releaseTestAt, &rev2); err != nil {
			t.Fatalf("step 3 offline verify: %v", err)
		}
		step3Path, _ := f.writeUpdate("step3.json", step3Raw)
		d3 := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", step3Path, "-expect-revision", "3"}, exitOK, "applied")
		if d3.UpdateSHA256 != step3Res.UpdateSHA256 || d3.Change == nil || d3.Change.OldRevision != 3 || d3.Change.NewRevision != 4 || d3.Change.OldReleaseRevision != 2 || d3.Change.NewReleaseRevision != 3 || len(d3.Change.Keys) != 1 || d3.Change.Keys[0].Previous == nil || d3.Change.Keys[0].Previous.State != "retired" || d3.Change.Keys[0].Key.ID != f.a.id || d3.Change.Keys[0].Key.State != "revoked" || d3.Change.Keys[0].Key.RevokedAt == nil || !d3.Change.Keys[0].Key.RevokedAt.Equal(releaseTestAt) || !reflect.DeepEqual(d3.Change.UsableRootKeyIDs, []string{f.b.id}) || d3.Change.NodeTrust != "nodes_require_new_bundle" {
			t.Fatalf("step 3 document = %+v", d3)
		}
		insp = runVaultInspect(t, vaultBinary, f.dir)
		if insp.LedgerRevision != 4 || insp.ReleaseRevision != 3 {
			t.Fatalf("step 3 inspect = %+v", insp)
		}
		ts3, _ := runVaultExportTrust(t, vaultBinary, f.dir, filepath.Join(f.root, "step3.trust"))
		if kA, ok := ts3.Lookup(f.a.id); !ok || !kA.Revoked {
			t.Fatalf("step 3 bundle A = %+v", kA)
		}
		if kB, ok := ts3.Lookup(f.b.id); !ok || kB.Revoked {
			t.Fatalf("step 3 bundle B = %+v", kB)
		}
		if _, _, err := envelope.Verify(signReleaseEnvelope(t, f.a, map[string]string{"test": "revoked"}), ts3, envelope.PurposeRelease, releaseTestAt); err == nil {
			t.Fatal("revoked root A verified against exported bundle")
		}
		if _, _, err := envelope.Verify(signReleaseEnvelope(t, f.b, map[string]string{"test": "active"}), ts3, envelope.PurposeRelease, releaseTestAt); err != nil {
			t.Fatalf("active root B failed verification: %v", err)
		}
	})
}

func TestUPD02_CommandReleaseTrustRepetition(t *testing.T) {
	f := setupRelease(t)
	first := invokeRelease(t, command{}, f.args(), exitOK, "applied")
	before := manifest(t, f.dir)
	for _, expected := range []string{"1", "2", "99"} {
		repeat := invokeRelease(t, command{}, flagValue(f.args(), "expect-revision", expected), exitAlreadyApplied, "already_applied")
		unchanged(t, f.dir, before)
		if repeat.UpdateSHA256 != f.sha256 || repeat.Change == nil || repeat.Change.OldRevision != 2 || repeat.Change.NewRevision != 2 || repeat.Change.OldReleaseRevision != 1 || repeat.Change.NewReleaseRevision != 1 || len(repeat.Change.Keys) != 1 || repeat.Change.Keys[0].Previous != nil || !reflect.DeepEqual(repeat.Change.Keys[0].Key, first.Change.Keys[0].Key) || repeat.Change.NodeTrust != "nodes_require_new_bundle" {
			t.Fatalf("unexpected repeat document for expect-revision=%s: %+v", expected, repeat)
		}
	}
}

func TestUPD02_CommandReleaseTrustPrecondition(t *testing.T) {
	f := setupRelease(t)
	before := manifest(t, f.dir)
	for _, rev := range []string{"0", "2", "99"} {
		d := invokeRelease(t, command{}, flagValue(f.args(), "expect-revision", rev), exitRefused, "precondition_refused")
		unchanged(t, f.dir, before)
		if d.UpdateSHA256 != f.sha256 || d.Change != nil || d.Reason != "" {
			t.Fatalf("refusal document leaked or missing hash: %+v", d)
		}
	}
	// An unrelated central rotation advances the ledger revision; the previously
	// read expected revision must be refused without changing the vault.
	if _, err := f.p.Rotate(context.Background(), signing.Lease, time.Hour, time.Minute); err != nil {
		t.Fatal(err)
	}
	before = manifest(t, f.dir)
	d := invokeRelease(t, command{}, f.args(), exitRefused, "precondition_refused")
	unchanged(t, f.dir, before)
	if d.UpdateSHA256 != f.sha256 || d.Change != nil || d.Reason != "" {
		t.Fatalf("refusal after rotation: %+v", d)
	}
}

func TestUPD02_CommandReleaseTrustRejected(t *testing.T) {
	t.Run("schema fixtures do not open vault", func(t *testing.T) {
		matches, err := filepath.Glob("../../../../contracts/examples/trust/invalid/*.json")
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 6 {
			t.Fatalf("schema fixtures = %d; want 6", len(matches))
		}
		f := setupRelease(t)
		before := manifest(t, f.dir)
		neverOpen := command{open: func(context.Context, string) (changeProvider, error) {
			t.Fatal("vault opened on schema rejection")
			return nil, nil
		}}
		for _, fixturePath := range matches {
			raw, err := os.ReadFile(fixturePath)
			if err != nil {
				t.Fatal(err)
			}
			localPath, sum := f.writeUpdate(filepath.Base(fixturePath), raw)
			d := invokeRelease(t, neverOpen, []string{"apply-release-update", "-vault", filepath.Join(f.root, "nonexistent-vault"), "-update", localPath, "-expect-revision", "1"}, exitRejected, "rejected")
			unchanged(t, f.dir, before)
			if d.Reason != "schema" || d.UpdateSHA256 != sum || d.Change != nil {
				t.Fatalf("schema fixture %s document = %+v", filepath.Base(fixturePath), d)
			}
		}
		// Wrong kind in a freshly signed envelope is also rejected by the schema check without opening the vault.
		wrongKindRaw := signReleaseEnvelope(t, f.a, map[string]any{
			"kind":     "fvpn-release-trust-v0",
			"revision": "1",
			"keys":     []signing.KeyRecord{f.b.record("active", releaseTestStart, releaseTestEnd, nil)},
		})
		wrongKindPath, wrongKindSum := f.writeUpdate("wrong-kind-signed.json", wrongKindRaw)
		d := invokeRelease(t, neverOpen, []string{"apply-release-update", "-vault", filepath.Join(f.root, "nonexistent-vault"), "-update", wrongKindPath, "-expect-revision", "1"}, exitRejected, "rejected")
		if d.Reason != "schema" || d.UpdateSHA256 != wrongKindSum || d.Change != nil {
			t.Fatalf("wrong kind document = %+v", d)
		}
	})

	t.Run("provider rejections", func(t *testing.T) {
		f := setupRelease(t)
		// Apply initial update (adds B at release_revision 1, ledger_revision 2).
		invokeRelease(t, command{}, f.args(), exitOK, "applied")
		// Retire B at release_revision 2 (ledger_revision 3), and add C and an expired root E and a revoked root R.
		expired := genReleaseKey(t)
		revoked := genReleaseKey(t)
		revAt := releaseTestAt
		setupEnv := signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
			Kind:     release.ReleaseTrustUpdateKind,
			Revision: 2,
			Keys: []signing.KeyRecord{
				f.b.record("retired", releaseTestStart, releaseTestNarrowEnd, nil),
				f.c.record("active", releaseTestStart, releaseTestEnd, nil),
				expired.record("active", releaseTestStart, releaseTestStart.Add(time.Hour), nil),
				revoked.record("revoked", releaseTestStart, releaseTestEnd, &revAt),
			},
		})
		setupPath, _ := f.writeUpdate("setup-rev2.json", setupEnv)
		invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", setupPath, "-expect-revision", "2"}, exitOK, "applied")

		// Also create a central lease key to prove a signer of another purpose is rejected.
		leaseID, err := f.p.Rotate(context.Background(), signing.Lease, time.Hour, time.Minute)
		if err != nil {
			t.Fatal(err)
		}
		leaseSigned, err := f.p.SignEnvelope(context.Background(), signing.Lease, signing.ReleaseUpdate{
			Kind:     release.ReleaseTrustUpdateKind,
			Revision: 3,
			Keys:     []signing.KeyRecord{genReleaseKey(t).record("active", releaseTestStart, releaseTestEnd, nil)},
		})
		if err != nil || leaseSigned.Signatures[0].KeyID != leaseID {
			t.Fatal(err)
		}
		leaseRaw, err := leaseSigned.MarshalCanonical()
		if err != nil {
			t.Fatal(err)
		}

		// Build a corrupted-signature envelope.
		var badSigEnv envelope.Envelope
		validRaw := signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
			Kind:     release.ReleaseTrustUpdateKind,
			Revision: 3,
			Keys:     []signing.KeyRecord{genReleaseKey(t).record("active", releaseTestStart, releaseTestEnd, nil)},
		})
		if err := json.Unmarshal(validRaw, &badSigEnv); err != nil {
			t.Fatal(err)
		}
		sigBytes, err := base64.StdEncoding.DecodeString(badSigEnv.Signatures[0].SignatureB64)
		if err != nil {
			t.Fatal(err)
		}
		sigBytes[0] ^= 0xff
		badSigEnv.Signatures[0].SignatureB64 = base64.StdEncoding.EncodeToString(sigBytes)
		badSigRaw, err := badSigEnv.MarshalCanonical()
		if err != nil {
			t.Fatal(err)
		}

		other := genReleaseKey(t)
		laterRevAt := revAt.Add(time.Minute)
		cases := []struct {
			name   string
			raw    []byte
			reason string
		}{
			{
				name: "unknown signer",
				raw: signReleaseEnvelope(t, other, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "signature",
			},
			{
				name: "expired signer",
				raw: signReleaseEnvelope(t, expired, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "signature",
			},
			{
				name: "revoked signer",
				raw: signReleaseEnvelope(t, revoked, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "signature",
			},
			{
				name:   "signer of another purpose only",
				raw:    append(leaseRaw, '\n'),
				reason: "signature",
			},
			{
				name:   "bad signature",
				raw:    append(badSigRaw, '\n'),
				reason: "signature",
			},
			{
				name: "overflow uint64 revision",
				raw: signReleaseEnvelope(t, f.a, map[string]any{
					"kind":     release.ReleaseTrustUpdateKind,
					"revision": "18446744073709551616",
					"keys":     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "payload",
			},
			{
				name: "revision equal to stored with other content",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 2,
					Keys:     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "revision",
			},
			{
				name: "lower revision",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 1,
					Keys:     []signing.KeyRecord{other.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "revision",
			},
			{
				name: "changed public key",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys: []signing.KeyRecord{{
						ID:        f.c.id,
						Purpose:   signing.Release,
						Public:    append([]byte(nil), other.pub...),
						State:     "active",
						NotBefore: releaseTestStart,
						NotAfter:  releaseTestEnd,
					}},
				}),
				reason: "record",
			},
			{
				name: "earlier not_before",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{f.c.record("active", releaseTestStart.Add(-time.Hour), releaseTestEnd, nil)},
				}),
				reason: "record",
			},
			{
				name: "later not_after",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{f.b.record("retired", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "record",
			},
			{
				name: "revoked key made active",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{revoked.record("active", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "record",
			},
			{
				name: "revoked key made retired",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{revoked.record("retired", releaseTestStart, releaseTestEnd, nil)},
				}),
				reason: "record",
			},
			{
				name: "changed revoked_at",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{revoked.record("revoked", releaseTestStart, releaseTestEnd, &laterRevAt)},
				}),
				reason: "record",
			},
			{
				name: "retired key made active",
				raw: signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
					Kind:     release.ReleaseTrustUpdateKind,
					Revision: 3,
					Keys:     []signing.KeyRecord{f.b.record("active", releaseTestStart, releaseTestNarrowEnd, nil)},
				}),
				reason: "record",
			},
		}

		before := manifest(t, f.dir)
		for i, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				path, sum := f.writeUpdate("reject-"+string(rune('a'+i))+".json", tc.raw)
				d := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", path, "-expect-revision", "4"}, exitRejected, "rejected")
				unchanged(t, f.dir, before)
				if d.Reason != tc.reason || d.UpdateSHA256 != sum || d.Change != nil {
					t.Fatalf("%s: got reason=%q sha=%q change=%+v; want reason=%q sha=%q", tc.name, d.Reason, d.UpdateSHA256, d.Change, tc.reason, sum)
				}
			})
		}
	})
}

func TestUPD02_CommandReleaseTrustUsableRootGuard(t *testing.T) {
	f := setupRelease(t)
	revAt := releaseTestAt
	revokeLastRaw := signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
		Kind:     release.ReleaseTrustUpdateKind,
		Revision: 1,
		Keys:     []signing.KeyRecord{f.a.record("revoked", releaseTestStart, releaseTestEnd, &revAt)},
	})
	path, sum := f.writeUpdate("revoke-last.json", revokeLastRaw)
	before := manifest(t, f.dir)

	// Without -allow-no-usable-root: rejected with reason no_usable_root.
	d := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", path, "-expect-revision", "1"}, exitRejected, "rejected")
	unchanged(t, f.dir, before)
	if d.Reason != "no_usable_root" || d.UpdateSHA256 != sum || d.Change != nil {
		t.Fatalf("expected no_usable_root rejection, got %+v", d)
	}

	// With -allow-no-usable-root: applied with empty usable_root_key_ids.
	d = invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", path, "-expect-revision", "1", "-allow-no-usable-root"}, exitOK, "applied")
	if d.UpdateSHA256 != sum || d.Change == nil || len(d.Change.UsableRootKeyIDs) != 0 || d.Change.UsableRootKeyIDs == nil || d.Change.Keys[0].Key.State != "revoked" {
		t.Fatalf("expected applied with empty usable roots, got %+v", d)
	}
	// Now that the last usable root is revoked, any further update signed by it is rejected.
	before = manifest(t, f.dir)
	afterRevoke := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", path, "-expect-revision", "2"}, exitRejected, "rejected")
	unchanged(t, f.dir, before)
	if afterRevoke.Reason != "signature" || afterRevoke.Change != nil {
		t.Fatalf("after revoking last root: %+v", afterRevoke)
	}
}

func TestUPD02_CommandReleaseTrustMultipleRecords(t *testing.T) {
	f := setupRelease(t)
	multiRaw := signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
		Kind:     release.ReleaseTrustUpdateKind,
		Revision: 1,
		Keys: []signing.KeyRecord{
			f.a.record("retired", releaseTestStart, releaseTestNarrowEnd, nil),
			f.b.record("active", releaseTestStart, releaseTestEnd, nil),
		},
	})
	path, sum := f.writeUpdate("multi.json", multiRaw)
	d := invokeRelease(t, command{}, []string{"apply-release-update", "-vault", f.dir, "-update", path, "-expect-revision", "1"}, exitOK, "applied")
	if d.UpdateSHA256 != sum || d.Change == nil || len(d.Change.Keys) != 2 {
		t.Fatalf("expected 2 changed records, got %+v", d)
	}
	if d.Change.Keys[0].Previous == nil || d.Change.Keys[0].Previous.State != "active" || d.Change.Keys[0].Key.ID != f.a.id || d.Change.Keys[0].Key.State != "retired" {
		t.Fatalf("first changed record = %+v", d.Change.Keys[0])
	}
	if d.Change.Keys[1].Previous != nil || d.Change.Keys[1].Key.ID != f.b.id || d.Change.Keys[1].Key.State != "active" {
		t.Fatalf("second changed record = %+v", d.Change.Keys[1])
	}
}

func TestUPD02_CommandReleaseTrustProcessConcurrency(t *testing.T) {
	binary := buildCommand(t)
	f := setupRelease(t)
	secondRaw := signReleaseEnvelope(t, f.a, signing.ReleaseUpdate{
		Kind:     release.ReleaseTrustUpdateKind,
		Revision: 2,
		Keys:     []signing.KeyRecord{f.c.record("active", releaseTestStart, releaseTestEnd, nil)},
	})
	secondPath, _ := f.writeUpdate("update-c.json", secondRaw)
	updates := []string{f.update, secondPath}

	lock, err := os.OpenFile(filepath.Join(f.dir, "lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	const n = 2
	commands := make([]*exec.Cmd, n)
	stdout := make([]bytes.Buffer, n)
	stderr := make([]bytes.Buffer, n)
	errs := make([]error, n)
	for i := range n {
		commands[i] = exec.CommandContext(ctx, binary, "apply-release-update", "-vault", f.dir, "-update", updates[i], "-expect-revision", "1")
		commands[i].Stdout = &stdout[i]
		commands[i].Stderr = &stderr[i]
		if err := commands[i].Start(); err != nil {
			t.Fatal(err)
		}
	}
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = commands[i].Wait()
		}(i)
	}
	wg.Wait()

	applied := 0
	other := 0
	for i := range n {
		code := processCode(t, errs[i])
		switch code {
		case exitOK:
			applied++
			assertReleaseOutput(t, code, stdout[i].String(), stderr[i].String(), exitOK, "applied", "")
		case exitRefused:
			other++
			assertReleaseOutput(t, code, stdout[i].String(), stderr[i].String(), exitRefused, "precondition_refused", "")
		case exitRejected:
			other++
			assertReleaseOutput(t, code, stdout[i].String(), stderr[i].String(), exitRejected, "rejected", "")
		default:
			t.Fatalf("unexpected child exit code %d: stdout=%s stderr=%s", code, stdout[i].String(), stderr[i].String())
		}
	}
	if applied != 1 || other != 1 {
		t.Fatalf("applied=%d other=%d; want 1 and 1", applied, other)
	}
	s := getSnapshot(t, f.p)
	if s.Revision != 2 || (s.ReleaseRevision != 1 && s.ReleaseRevision != 2) || len(s.Keys) != 2 {
		t.Fatalf("invalid final snapshot: %+v", s)
	}
}

func TestUPD02_CommandReleaseTrustHeldLock(t *testing.T) {
	f := setupRelease(t)
	lock, err := os.OpenFile(filepath.Join(f.dir, "lock"), os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		t.Fatal(err)
	}
	args := append(f.args(), "-lock-timeout", "20ms")
	before := manifest(t, f.dir)
	invokeRelease(t, command{}, args, exitTimeout, "")
	unchanged(t, f.dir, before)
	if err := lock.Close(); err != nil {
		t.Fatal(err)
	}
	invokeRelease(t, command{}, args, exitOK, "applied")
}

type releaseFaultProvider struct {
	changeProvider
	mutationError                                error
	result                                       *signing.ReleaseUpdateResult
	uncertain, readFail, different, waitDeadline bool
	reads, calls                                 int
	alterSnapshot                                func(*signing.FileSnapshot)
}

func (p *releaseFaultProvider) UpdateReleaseTrustConditional(ctx context.Context, raw []byte, expectedRevision decstr.U64, allowNoUsableRoot bool) (signing.ReleaseUpdateResult, error) {
	if p.mutationError != nil {
		return signing.ReleaseUpdateResult{}, p.mutationError
	}
	if p.result != nil {
		p.calls++
		return *p.result, nil
	}
	r, err := p.changeProvider.UpdateReleaseTrustConditional(ctx, raw, expectedRevision, allowNoUsableRoot)
	p.calls++
	if p.waitDeadline {
		<-ctx.Done()
	}
	if err == nil && p.uncertain {
		r.Outcome = signing.ChangeUncertain
	}
	return r, err
}

func (p *releaseFaultProvider) Snapshot(ctx context.Context) (signing.FileSnapshot, error) {
	p.reads++
	if p.readFail {
		return signing.FileSnapshot{}, signing.ErrUnavailable
	}
	s, err := p.changeProvider.Snapshot(ctx)
	if p.different {
		s.Revision++
	}
	if err == nil && p.alterSnapshot != nil {
		p.alterSnapshot(&s)
	}
	return s, err
}

// TestUPD02_ReleaseTrustConditionalPartialFailure and
// TestUPD02_ReleaseTrustConditionalDeadline in internal/signing/release_update_test.go
// run the real save-boundary and post-lock deadline sequences through provider seams.
// This test exercises the command's handling of each typed result and read-back.
func TestUPD02_CommandReleaseTrustPartialFailure(t *testing.T) {
	for _, stage := range []string{"before rename", "after rename", "read failure", "different ledger revision", "different release revision", "different key record"} {
		t.Run(stage, func(t *testing.T) {
			f := setupRelease(t)
			p := &releaseFaultProvider{changeProvider: f.p}
			want, outcome := exitNotApplied, "not_applied"
			switch stage {
			case "before rename":
				p.result = &signing.ReleaseUpdateResult{Outcome: signing.ChangeNotApplied}
			case "after rename":
				p.uncertain = true
				want, outcome = exitDurability, "durability_not_confirmed"
			case "read failure":
				p.uncertain, p.readFail = true, true
				want, outcome = exitUnknown, "unknown"
			case "different ledger revision":
				p.uncertain, p.different = true, true
				want, outcome = exitUnknown, "unknown"
			case "different release revision":
				p.uncertain = true
				p.alterSnapshot = func(s *signing.FileSnapshot) {
					s.ReleaseRevision++
				}
				want, outcome = exitUnknown, "unknown"
			case "different key record":
				p.uncertain = true
				p.alterSnapshot = func(s *signing.FileSnapshot) {
					for i := range s.Keys {
						if s.Keys[i].ID == f.b.id {
							s.Keys[i].State = "retired"
							return
						}
					}
					t.Fatal("target key not found in snapshot")
				}
				want, outcome = exitUnknown, "unknown"
			}
			before := manifest(t, f.dir)
			d := invokeRelease(t, withProvider(p), f.args(), want, outcome)
			if d.UpdateSHA256 != f.sha256 {
				t.Fatalf("missing update_sha256: %+v", d)
			}
			if p.calls != 1 {
				t.Fatal("retried mutation")
			}
			if p.result != nil {
				unchanged(t, f.dir, before)
				if p.reads != 0 {
					t.Fatal("unnecessary read-back")
				}
				invokeRelease(t, command{}, f.args(), exitOK, "applied")
			} else {
				if p.reads != 1 {
					t.Fatal("read-back count")
				}
				if stage == "after rename" {
					if d.ReadBack != "matched" || d.Change == nil || d.Change.NewRevision != 2 || d.Change.NewReleaseRevision != 1 {
						t.Fatalf("missing confirmation: %+v", d)
					}
				} else {
					wantReadBack := "different"
					if stage == "read failure" {
						wantReadBack = "failed"
					}
					if d.ReadBack != wantReadBack || d.Change != nil {
						t.Fatalf("wrong uncertain document: %+v", d)
					}
				}
				before = manifest(t, f.dir)
				invokeRelease(t, command{}, f.args(), exitAlreadyApplied, "already_applied")
				unchanged(t, f.dir, before)
			}
			if s := getSnapshot(t, f.p); s.Revision != 2 || s.ReleaseRevision != 1 {
				t.Fatalf("wrong final state: %+v", s)
			}
		})
	}

	t.Run("expired context read-back uses fresh deadline", func(t *testing.T) {
		for _, uncertain := range []bool{false, true} {
			f := setupRelease(t)
			p := &releaseFaultProvider{changeProvider: f.p, waitDeadline: true, uncertain: uncertain}
			code, outcome := exitOK, "applied"
			if uncertain {
				code, outcome = exitDurability, "durability_not_confirmed"
			}
			invokeRelease(t, withProvider(p), append(f.args(), "-lock-timeout", "30ms"), code, outcome)
			if s := getSnapshot(t, f.p); s.Revision != 2 || s.ReleaseRevision != 1 {
				t.Fatal("change canceled after lock acquisition")
			}
			if uncertain && p.reads != 1 {
				t.Fatal("no fresh read-back")
			}
		}
	})
}

func TestUPD02_CommandReleaseTrustOutputFailure(t *testing.T) {
	for _, short := range []bool{false, true} {
		f := setupRelease(t)
		var stderr bytes.Buffer
		code := run(f.args(), failingWriter{short: short}, &stderr)
		assertReleaseOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: applied; run fvpn-vault inspect; do not repeat blindly\n")
		if s := getSnapshot(t, f.p); s.Revision != 2 || s.ReleaseRevision != 1 {
			t.Fatal("output failure undid change")
		}
		before := manifest(t, f.dir)
		invokeRelease(t, command{}, f.args(), exitAlreadyApplied, "already_applied")
		unchanged(t, f.dir, before)
	}
	for _, outcome := range []string{"already_applied", "rejected"} {
		var stderr bytes.Buffer
		code := writeResult(failingWriter{}, &stderr, document{Outcome: outcome}, 0)
		assertReleaseOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: "+outcome+"; run fvpn-vault inspect; do not repeat blindly\n")
	}
}

func TestUPD02_CommandReleaseTrustProcessClosedPipe(t *testing.T) {
	binary := buildCommand(t)
	f := setupRelease(t)
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, binary, f.args()...)
	cmd.Stdout = writer
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err = cmd.Run()
	if closeErr := writer.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	code := processCode(t, err)
	assertReleaseOutput(t, code, "", stderr.String(), exitOutput, "", "fvpn-vault-keys: result output failed: applied; run fvpn-vault inspect; do not repeat blindly\n")
	if s := getSnapshot(t, f.p); s.Revision != 2 || s.ReleaseRevision != 1 {
		t.Fatal("lost or duplicated mutation")
	}
	before := manifest(t, f.dir)
	repeat := exec.CommandContext(ctx, binary, f.args()...)
	var stdout bytes.Buffer
	stderr.Reset()
	repeat.Stdout = &stdout
	repeat.Stderr = &stderr
	code = processCode(t, repeat.Run())
	assertReleaseOutput(t, code, stdout.String(), stderr.String(), exitAlreadyApplied, "already_applied", "")
	unchanged(t, f.dir, before)
}

func TestUPD02_CommandReleaseTrustUmask(t *testing.T) {
	f := setupRelease(t)
	var d document
	func() {
		previous := syscall.Umask(0777)
		defer syscall.Umask(previous)
		d = invokeRelease(t, command{}, f.args(), exitOK, "applied")
	}()
	info, err := os.Stat(filepath.Join(f.dir, "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("ledger mode = %04o; want 0600", info.Mode().Perm())
	}
	reopened, err := signing.OpenFileProvider(f.dir)
	if err != nil {
		t.Fatalf("cannot reopen saved vault: %v", err)
	}
	s := getSnapshot(t, reopened)
	if s.Revision != d.Change.NewRevision || s.ReleaseRevision != d.Change.NewReleaseRevision {
		t.Fatalf("snapshot disagrees with command result: %+v vs %+v", s, d.Change)
	}
	ts, err := reopened.TrustBundle(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ts.Lookup(f.b.id); !ok {
		t.Fatal("reopened provider missing added release root")
	}
}

func TestUPD02_CommandReleaseTrustArguments(t *testing.T) {
	f := setupRelease(t)
	base := f.args()

	oversized := filepath.Join(f.root, "oversized.json")
	if err := os.WriteFile(oversized, bytes.Repeat([]byte("a"), canonjson.MaxBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(f.root, "dir-input")
	if err := os.Mkdir(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	fifoPath := filepath.Join(f.root, "fifo-input")
	if err := syscall.Mkfifo(fifoPath, 0600); err != nil {
		t.Fatal(err)
	}

	cases := [][]string{
		{"apply-release-update"},
		append(append([]string{}, base...), "extra"),
		append(append([]string{}, base...), "-unknown"),
		append(append([]string{}, base...), "-purpose", "lease"),
		append(append([]string{}, base...), "-key-id", f.a.id),
		append(append([]string{}, base...), "-revoke-active"),
		append(append([]string{}, base...), "-lock-timeout", "0"),
		append(append([]string{}, base...), "-lock-timeout", "-1s"),
		append(append([]string{}, base...), "-lock-timeout", "bad"),
		append(append([]string{}, base...), "-lock-timeout", "1s", "-lock-timeout", "1s"),
		append(append([]string{}, base...), "-allow-no-usable-root", "-allow-no-usable-root"),
		append(append([]string{}, base...), "-allow-no-usable-root=bad"),
		flagValue(base, "vault", ""),
		flagValue(base, "vault", ".."),
		flagValue(base, "vault", f.dir+"/.."),
		flagValue(base, "update", ""),
		flagValue(base, "update", filepath.Join(f.root, "missing.json")),
		flagValue(base, "update", subDir),
		flagValue(base, "update", fifoPath),
		flagValue(base, "update", oversized),
	}
	for i := 1; i < len(base); i += 2 {
		without := append([]string{}, base[:i]...)
		without = append(without, base[i+2:]...)
		cases = append(cases, without)
		repeat := append([]string{}, base...)
		repeat = append(repeat, base[i], base[i+1])
		cases = append(cases, repeat)
		cases = append(cases, append(append([]string{}, base...), base[i]))
	}
	for _, revision := range []string{"", "-1", "+1", "01", " 1", "1.0", "1e0", "18446744073709551616", "999999999999999999999"} {
		cases = append(cases, flagValue(base, "expect-revision", revision))
	}

	before := manifest(t, f.dir)
	neverOpen := command{open: func(context.Context, string) (changeProvider, error) {
		t.Fatal("vault opened on usage failure")
		return nil, nil
	}}
	for i, a := range cases {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			invokeRelease(t, neverOpen, a, exitUsage, "")
			unchanged(t, f.dir, before)
		})
	}
}

func TestUPD02_CommandReleaseTrustUnavailableAndPaths(t *testing.T) {
	for _, kind := range []string{"missing path", "final symlink", "missing lock", "missing state", "lock group", "lock other", "lock symlink inside", "lock directory", "directory mode", "state mode", "invalid ledger"} {
		t.Run(kind, func(t *testing.T) {
			f := setupRelease(t)
			path := f.dir
			lock := filepath.Join(path, "lock")
			state := filepath.Join(path, "state.json")
			switch kind {
			case "missing path":
				path = filepath.Join(f.root, "missing")
			case "final symlink":
				path = filepath.Join(f.root, "alias")
				if err := os.Symlink(f.dir, path); err != nil {
					t.Fatal(err)
				}
			case "missing lock":
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
			case "missing state":
				if err := os.Remove(state); err != nil {
					t.Fatal(err)
				}
			case "lock group":
				if err := os.Chmod(lock, 0640); err != nil {
					t.Fatal(err)
				}
			case "lock other":
				if err := os.Chmod(lock, 0604); err != nil {
					t.Fatal(err)
				}
			case "lock symlink inside":
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("state.json", lock); err != nil {
					t.Fatal(err)
				}
			case "lock directory":
				if err := os.Remove(lock); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(lock, 0700); err != nil {
					t.Fatal(err)
				}
			case "directory mode":
				if err := os.Chmod(path, 0750); err != nil {
					t.Fatal(err)
				}
			case "state mode":
				if err := os.Chmod(state, 0604); err != nil {
					t.Fatal(err)
				}
			case "invalid ledger":
				if err := os.WriteFile(state, []byte(`{"revision":"invalid"}`), 0600); err != nil {
					t.Fatal(err)
				}
			}
			before := manifest(t, f.root)
			invokeRelease(t, command{}, flagValue(f.args(), "vault", path), exitUnavailable, "")
			unchanged(t, f.root, before)
		})
	}

	t.Run("FIFO lock in child process", func(t *testing.T) {
		binary := buildCommand(t)
		f := setupRelease(t)
		lockPath := filepath.Join(f.dir, "lock")
		if err := os.Remove(lockPath); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(lockPath, 0600); err != nil {
			t.Fatal(err)
		}
		before := manifest(t, f.dir)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, f.args()...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		if ctx.Err() != nil {
			t.Fatal("FIFO command exceeded 2s process deadline")
		}
		code := processCode(t, err)
		assertReleaseOutput(t, code, stdout.String(), stderr.String(), exitUnavailable, "", "fvpn-vault-keys: vault unavailable\n")
		unchanged(t, f.dir, before)
	})
}
