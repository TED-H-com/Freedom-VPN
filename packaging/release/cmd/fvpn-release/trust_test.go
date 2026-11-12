package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/canonjson"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// TEST ONLY - NOT A SECRET. Keys are generated per test, except the public
// golden vector. The provider has no clock injection: fixture windows surround
// each test's start, and the CLI uses that same reference time explicitly.
func trustDate(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}

type trustFixture struct {
	start, end, at, narrowEnd string
	t                         *testing.T
	dir                       string
	keys                      map[string]release.SigningKey
	provider                  *signing.FileProvider
}

func newTrustFixture(t *testing.T) *trustFixture {
	t.Helper()
	return newTrustFixtureAt(t, time.Now().UTC().Truncate(time.Second))
}

// newTrustFixtureAt is for literal-format tests whose CLI always supplies -at
// and never asks the provider to verify an update against its wall clock.
func newTrustFixtureAt(t *testing.T, now time.Time) *trustFixture {
	t.Helper()
	f := &trustFixture{
		t: t, dir: t.TempDir(), keys: map[string]release.SigningKey{},
		start:     now.Add(-48 * time.Hour).Format(time.RFC3339),
		end:       now.Add(48 * time.Hour).Format(time.RFC3339),
		at:        now.Format(time.RFC3339),
		narrowEnd: now.Add(24 * time.Hour).Format(time.RFC3339),
	}
	for _, id := range []string{"A", "B", "C"} {
		f.success([]string{"keygen", "-key-id", id, "-out", f.path(id + ".key"), "-trust-out", f.path(id + ".trust"), "-not-before", f.start, "-not-after", f.end})
		k, err := release.LoadKey(f.path(id + ".key"))
		if err != nil {
			t.Fatal(err)
		}
		f.keys[id] = k
	}
	var err error
	f.provider, err = signing.InitializeFileProvider(f.path("vault"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.provider.PinReleaseRoot(context.Background(), f.trusted("A")); err != nil {
		t.Fatal(err)
	}
	f.export()
	return f
}

func (f *trustFixture) path(name string) string { return filepath.Join(f.dir, name) }
func (f *trustFixture) trusted(id string) envelope.TrustedKey {
	return envelope.TrustedKey{KeyID: id, PublicKey: f.keys[id].Public(), Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: trustDate(f.start), NotAfter: trustDate(f.end)}
}
func (f *trustFixture) record(id string) release.ReleaseTrustRecord {
	k := f.trusted(id)
	return release.ReleaseTrustRecord{ID: id, Purpose: envelope.PurposeRelease, Public: k.PublicKey, State: "active", NotBefore: k.NotBefore, NotAfter: k.NotAfter}
}
func (f *trustFixture) update(records ...release.ReleaseTrustRecord) release.ReleaseTrustUpdate {
	return release.ReleaseTrustUpdate{Kind: release.ReleaseTrustUpdateKind, Revision: 10, Keys: records}
}
func (f *trustFixture) signed(value any, signer string) []byte {
	f.t.Helper()
	e, err := envelope.Sign(value, f.keys[signer].Private(), f.keys[signer].KeyID)
	if err != nil {
		f.t.Fatal(err)
	}
	b, err := e.MarshalCanonical()
	if err != nil {
		f.t.Fatal(err)
	}
	return b
}
func writeTrustTest(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
}
func jsonTrustTest(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func (f *trustFixture) export() *envelope.TrustStore {
	f.t.Helper()
	ts, err := f.provider.TrustBundle(context.Background())
	if err != nil {
		f.t.Fatal(err)
	}
	writeTrustTest(f.t, f.path("current.trust"), jsonTrustTest(f.t, ts))
	return ts
}
func (f *trustFixture) apply(raw []byte) {
	f.t.Helper()
	if err := f.provider.UpdateReleaseTrust(context.Background(), raw); err != nil {
		f.t.Fatalf("centre apply: %v", err)
	}
}
func (f *trustFixture) setRecords(records ...release.ReleaseTrustRecord) {
	u := f.update(records...)
	u.Revision = 1
	f.apply(f.signed(u, "A"))
	f.export()
}
func (f *trustFixture) rejectCentre(raw []byte) {
	f.t.Helper()
	before := readTestFile(f.t, f.path("vault/state.json"))
	if err := f.provider.UpdateReleaseTrust(context.Background(), raw); err == nil {
		f.t.Fatal("centre accepted refused update")
	}
	if !bytes.Equal(before, readTestFile(f.t, f.path("vault/state.json"))) {
		f.t.Fatal("centre refusal modified ledger")
	}
}
func (f *trustFixture) success(args []string) string {
	f.t.Helper()
	var out, diagnostic bytes.Buffer
	if code := run(args, &out, &diagnostic); code != 0 {
		f.t.Fatalf("command %s exit %d: %s", args[0], code, &diagnostic)
	}
	f.noSeed(out.Bytes(), diagnostic.Bytes())
	return out.String()
}
func (f *trustFixture) noSeed(values ...[]byte) {
	f.t.Helper()
	for _, k := range f.keys {
		for _, v := range values {
			if bytes.Contains(v, []byte(base64.StdEncoding.EncodeToString(k.Seed))) || bytes.Contains(v, []byte(hex.EncodeToString(k.Seed))) {
				f.t.Fatal("private seed leaked")
			}
		}
	}
}
func (f *trustFixture) produceFlags() map[string]string {
	return map[string]string{"key": f.path("A.key"), "trust": f.path("current.trust"), "revision": "10", "out": f.path("update.json"), "add": f.path("B.trust"), "at": f.at}
}
func trustArgs(command string, flags map[string]string) []string {
	names := make([]string, 0, len(flags))
	for k := range flags {
		names = append(names, k)
	}
	sort.Strings(names)
	args := []string{command}
	for _, k := range names {
		args = append(args, "-"+k+"="+flags[k])
	}
	return args
}
func (f *trustFixture) verifyArgs(extra ...string) []string {
	return append([]string{"trust-verify", "-update", f.path("update.json"), "-trust", f.path("current.trust"), "-at", f.at}, extra...)
}

// Snapshot files, permissions and symlink destinations, without following links.
// Every refusal must leave inputs, existing outputs and the ledger unchanged.
func trustSnapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		value := info.Mode().String()
		if info.Mode().IsRegular() {
			value += string(readTestFile(t, path))
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			value += target
		}
		out[path] = value
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func (f *trustFixture) refuse(args []string, want int) {
	f.t.Helper()
	before := trustSnapshot(f.t, f.dir)
	var out, diagnostic bytes.Buffer
	code := run(args, &out, &diagnostic)
	if code != want || out.Len() != 0 || diagnostic.Len() == 0 {
		f.t.Fatalf("refusal: exit %d want %d; stdout=%q stderr=%s", code, want, &out, &diagnostic)
	}
	f.noSeed(out.Bytes(), diagnostic.Bytes())
	f.noPaths(out.String(), diagnostic.String())
	if !reflect.DeepEqual(before, trustSnapshot(f.t, f.dir)) {
		f.t.Fatal("refusal changed files or created an output")
	}
}

func TestUPD02_TrustGoldenVector(t *testing.T) {
	f := newTrustFixture(t)
	var vector struct {
		SeedHex        string                     `json:"seed_hex"`
		KeyID          string                     `json:"key_id"`
		VerifyAt       string                     `json:"verify_at"`
		Envelope       json.RawMessage            `json:"envelope"`
		EnvelopeSHA256 string                     `json:"envelope_sha256"`
		Signed         release.ReleaseTrustUpdate `json:"signed"`
	}
	raw := readTestFile(t, "../../../../contracts/vectors/release-trust-update-signing.json")
	if err := json.Unmarshal(raw, &vector); err != nil {
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
	f.keys["golden"] = key
	if err := key.Save(f.path("golden.key")); err != nil {
		t.Fatal(err)
	}
	root := f.trusted("golden")
	root.KeyID = vector.KeyID
	// The golden bytes and verification instant are immutable. This root must
	// also authorize the provider at the run's clock; only its window widens.
	vectorAt := trustDate(vector.VerifyAt)
	if vectorAt.Before(root.NotBefore) {
		root.NotBefore = vectorAt.Add(-time.Hour)
	}
	if !vectorAt.Before(root.NotAfter) {
		root.NotAfter = vectorAt.Add(time.Hour)
	}
	f.provider, err = signing.InitializeFileProvider(f.path("golden-vault"))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.provider.PinReleaseRoot(context.Background(), root); err != nil {
		t.Fatal(err)
	}
	f.export()
	k := vector.Signed.Keys[0]
	added, err := envelope.NewNodeTrustStore(envelope.TrustedKey{KeyID: k.ID, PublicKey: k.Public, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: k.NotBefore, NotAfter: k.NotAfter})
	if err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("new.trust"), jsonTrustTest(t, added))
	flags := f.produceFlags()
	flags["key"], flags["add"], flags["revision"], flags["at"] = f.path("golden.key"), f.path("new.trust"), vector.Signed.Revision.String(), vector.VerifyAt
	out := f.success(trustArgs("trust-update", flags))
	want, err := canonjson.Canonicalize(vector.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	actual := readTestFile(t, f.path("update.json"))
	if !bytes.Equal(actual, append(want, '\n')) {
		t.Fatal("golden canonical envelope mismatch")
	}
	if !strings.Contains(out, "update_sha256="+vector.EnvelopeSHA256+"\n") {
		t.Fatal("golden hash mismatch")
	}
	f.success([]string{"trust-verify", "-update", f.path("update.json"), "-trust", f.path("current.trust"), "-at", vector.VerifyAt})
	if err := release.ValidateTrustUpdateEnvelope(actual); err != nil {
		t.Fatal(err)
	}
	f.noSeed(actual)
	f.apply(actual)
}

func TestUPD02_TrustOperationsAndHandover(t *testing.T) {
	f := newTrustFixture(t)
	flags := f.produceFlags()
	flags["key"] = f.path("B.key")
	f.refuse(trustArgs("trust-update", flags), 1) // B is not trusted yet.
	f.rejectCentre(f.signed(f.update(f.record("C")), "B"))
	flags["key"], flags["revision"] = f.path("A.key"), "1"
	f.success(trustArgs("trust-update", flags))
	step1 := readTestFile(t, f.path("update.json"))
	if err := release.ValidateTrustUpdateEnvelope(step1); err != nil {
		t.Fatal(err)
	}
	verified := f.success(f.verifyArgs("-current-revision", "0"))
	if !strings.Contains(verified, `"new":true`) || !strings.Contains(verified, "usable_root_after=true\n") {
		t.Fatal(verified)
	}
	f.apply(step1)
	ts := f.export()
	for _, id := range []string{"A", "B"} {
		if k, ok := ts.Lookup(id); !ok || k.Revoked {
			t.Fatal("handover add not applied")
		}
	}
	if ts.Len() != 2 {
		t.Fatal("unexpected trust bundle size")
	}
	delete(flags, "add")
	flags["key"], flags["retire"], flags["not-after"], flags["revision"], flags["out"] = f.path("B.key"), "A", f.narrowEnd, "2", f.path("retire.json")
	f.success(trustArgs("trust-update", flags))
	retired := readTestFile(t, flags["out"])
	writeTrustTest(t, f.path("update.json"), retired)
	if err := release.ValidateTrustUpdateEnvelope(retired); err != nil {
		t.Fatal(err)
	}
	verified = f.success(f.verifyArgs("-current-revision", "1"))
	if !strings.Contains(verified, `"window_narrowed":true`) || !strings.Contains(verified, `"state":"retired"`) {
		t.Fatal(verified)
	}
	f.apply(retired)
	ts = f.export()
	a, _ := ts.Lookup("A")
	b, _ := ts.Lookup("B")
	if a.NotAfter != trustDate(f.narrowEnd) || a.Revoked || b.Revoked || b.NotAfter != trustDate(f.end) {
		t.Fatal("retirement bundle mismatch")
	}
	f.rejectCentre(step1)
	// Retired release roots still verify within their narrowed window.
	if _, _, err := envelope.Verify(f.signed(map[string]string{"test": "release-purpose"}, "A"), ts, envelope.PurposeRelease, trustDate(f.at)); err != nil {
		t.Fatal(err)
	}
	delete(flags, "retire")
	delete(flags, "not-after")
	flags["revoke"], flags["revision"], flags["out"] = "A", "3", f.path("revoke.json")
	f.success(trustArgs("trust-update", flags))
	revoked := readTestFile(t, flags["out"])
	writeTrustTest(t, f.path("update.json"), revoked)
	if err := release.ValidateTrustUpdateEnvelope(revoked); err != nil {
		t.Fatal(err)
	}
	verified = f.success(f.verifyArgs("-current-revision", "2"))
	if !strings.Contains(verified, `"newly_revoked":true`) {
		t.Fatal(verified)
	}
	f.apply(revoked)
	ts = f.export()
	a, _ = ts.Lookup("A")
	b, _ = ts.Lookup("B")
	if !a.Revoked || b.Revoked || a.NotAfter != trustDate(f.narrowEnd) || ts.Len() != 2 {
		t.Fatal("revocation bundle mismatch")
	}
	for _, id := range []string{"A", "B"} {
		_, _, err := envelope.Verify(f.signed(map[string]string{"test": "release-purpose"}, id), ts, envelope.PurposeRelease, trustDate(f.at))
		if (err != nil) != (id == "A") {
			t.Fatalf("final signer %s: %v", id, err)
		}
	}
	flags = f.produceFlags()
	flags["add"], flags["out"] = f.path("C.trust"), f.path("after-revocation.json")
	f.refuse(trustArgs("trust-update", flags), 1)
	f.rejectCentre(f.signed(f.update(f.record("C")), "A"))
}

func TestSEC01_TrustOutputProperties(t *testing.T) {
	f := newTrustFixtureAt(t, trustDate("2026-11-01T12:00:00Z"))
	flags := f.produceFlags()
	var first []byte
	for _, mask := range []int{0, 0777} {
		flags["out"] = f.path(fmt.Sprintf("mode-%o.json", mask))
		func() {
			previous := syscall.Umask(mask)
			defer syscall.Umask(previous)
			f.success(trustArgs("trust-update", flags))
		}()
		raw := readTestFile(t, flags["out"])
		info, err := os.Stat(flags["out"])
		if err != nil || info.Mode().Perm() != 0644 {
			t.Fatal("output mode is not 0644")
		}
		f.noSeed(raw)
		if first != nil && !bytes.Equal(first, raw) {
			t.Fatal("identical inputs changed output")
		}
		first = raw
	}
	// Offset inputs normalize to UTC, preserving explicitly supplied precision.
	flags["out"], flags["at"] = f.path("utc.json"), "2026-11-01T07:00:00.123456789-05:00"
	delete(flags, "add")
	flags["revoke"], flags["allow-no-usable-root"] = "A", "true"
	f.success(trustArgs("trust-update", flags))
	if !bytes.Contains(readTestFile(t, flags["out"]), []byte(`"revoked_at":"2026-11-01T12:00:00.123456789Z"`)) {
		t.Fatal("revocation time was not UTC or lost precision")
	}
}

func (f *trustFixture) noPaths(values ...string) {
	f.t.Helper()
	for _, value := range values {
		if strings.Contains(value, f.dir) {
			f.t.Fatal("command output disclosed a test directory path")
		}
	}
}
