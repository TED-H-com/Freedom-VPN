package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/contracts"
	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

// TEST ONLY writers: include private paths/raw text in errors to prove they
// cannot escape through result-output diagnostics. A short writer returns nil.
type trustResultFailure struct {
	short    bool
	rawError string
	written  bytes.Buffer
}

func (w *trustResultFailure) Write(p []byte) (int, error) {
	if w.short {
		n := len(p) / 2
		_, _ = w.written.Write(p[:n])
		return n, nil
	}
	return 0, errors.New(w.rawError)
}

func TestUPD02_TrustR1ResultWriteFailures(t *testing.T) {
	for _, command := range []string{"trust-update", "trust-verify"} {
		for _, short := range []bool{false, true} {
			name := command + "/failed"
			if short {
				name = command + "/short"
			}
			t.Run(name, func(t *testing.T) {
				f := newTrustFixture(t)
				flags := f.produceFlags()
				want, _, err := release.BuildTrustUpdate(f.keys["A"], mustCurrentTrust(t, f), release.TrustUpdateOptions{
					Revision: 10, Operation: "add", AddTrust: readTestFile(t, f.path("B.trust")), At: trustDate(trustAt),
				})
				if err != nil {
					t.Fatal(err)
				}
				args := trustArgs(command, flags)
				message := "trust-update: result output failed; published update file remains"
				if command == "trust-verify" {
					f.success(trustArgs("trust-update", flags))
					args, message = f.verifyArgs(), "trust-verify: result output failed"
				}
				writer := &trustResultFailure{short: short, rawError: f.path("raw-system-error")}
				var diagnostic bytes.Buffer
				if code := run(args, writer, &diagnostic); code != 1 || diagnostic.String() != "fvpn-release: "+message+"\n" {
					t.Fatalf("exit=%d diagnostic=%q", code, &diagnostic)
				}
				f.noPaths(writer.written.String(), diagnostic.String())
				actual := readTestFile(t, flags["out"])
				if !bytes.Equal(actual, append(want, '\n')) {
					t.Fatal("failed result output changed published file")
				}
				f.success(f.verifyArgs())
				f.noSeed(actual, writer.written.Bytes(), diagnostic.Bytes())
			})
		}
	}
}

func mustCurrentTrust(t *testing.T, f *trustFixture) *release.ReleaseTrust {
	t.Helper()
	trust, err := release.ParseReleaseTrust(readTestFile(t, f.path("current.trust")))
	if err != nil {
		t.Fatal(err)
	}
	return trust
}

func assertTrustFailure(t *testing.T, f *trustFixture, args []string, code int, message string) {
	t.Helper()
	var output, diagnostic bytes.Buffer
	got := run(args, &output, &diagnostic)
	if got != code || output.Len() != 0 || diagnostic.String() != "fvpn-release: "+message+"\n" {
		t.Fatalf("exit=%d want %d, stdout=%q stderr=%q", got, code, &output, &diagnostic)
	}
	f.noPaths(output.String(), diagnostic.String())
	f.noSeed(output.Bytes(), diagnostic.Bytes())
}

func TestSEC01_TrustR2FilesystemDiagnostics(t *testing.T) {
	for _, command := range []string{"trust-update", "trust-verify"} {
		inputs := []string{"key", "add", "trust"}
		if command == "trust-verify" {
			inputs = []string{"update", "trust"}
		}
		for _, input := range inputs {
			for _, failure := range []string{"missing", "unreadable"} {
				t.Run(command+"/"+input+"/"+failure, func(t *testing.T) {
					if failure == "unreadable" && os.Geteuid() == 0 {
						t.Skip("permission test requires an unprivileged user")
					}
					f := newTrustFixture(t)
					flags := f.produceFlags()
					if command == "trust-verify" {
						f.success(trustArgs("trust-update", flags))
						flags = map[string]string{"update": f.path("update.json"), "trust": f.path("current.trust"), "at": trustAt}
					}
					before := trustSnapshot(t, f.dir)
					if failure == "missing" {
						flags[input] = f.path("missing-" + input)
					} else {
						info, err := os.Stat(flags[input])
						if err != nil {
							t.Fatal(err)
						}
						if err := os.Chmod(flags[input], 0); err != nil {
							t.Fatal(err)
						}
						defer func() {
							if err := os.Chmod(flags[input], info.Mode().Perm()); err != nil {
								t.Error(err)
							}
						}()
						assertTrustFailure(t, f, trustArgs(command, flags), 1, "-"+input+" cannot be read")
						if err := os.Chmod(flags[input], info.Mode().Perm()); err != nil {
							t.Fatal(err)
						}
					}
					if failure == "missing" {
						assertTrustFailure(t, f, trustArgs(command, flags), 1, "-"+input+" cannot be read")
					}
					if !reflect.DeepEqual(before, trustSnapshot(t, f.dir)) {
						t.Fatal("input refusal modified files")
					}
				})
			}
		}
	}
	for _, kind := range []string{"file", "directory", "symlink", "dangling-symlink", "missing-parent", "unwritable-parent", "publication"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "unwritable-parent" && os.Geteuid() == 0 {
				t.Skip("permission test requires an unprivileged user")
			}
			f := newTrustFixture(t)
			flags := f.produceFlags()
			message := "-out already exists"
			switch kind {
			case "file":
				writeTrustTest(t, flags["out"], []byte("preserve\n"))
			case "directory":
				if err := os.Mkdir(flags["out"], 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink", "dangling-symlink":
				if kind == "symlink" {
					writeTrustTest(t, f.path("target"), []byte("preserve\n"))
				}
				if err := os.Symlink("target", flags["out"]); err != nil {
					t.Fatal(err)
				}
			case "missing-parent":
				flags["out"], message = f.path("absent/update.json"), "-out parent directory unavailable"
			case "unwritable-parent":
				if err := os.Mkdir(f.path("readonly"), 0500); err != nil {
					t.Fatal(err)
				}
				defer func() {
					if err := os.Chmod(f.path("readonly"), 0700); err != nil {
						t.Error(err)
					}
				}()
				flags["out"], message = f.path("readonly/update.json"), "-out parent directory unavailable"
			case "publication":
				// Linux NAME_MAX is 255: staging succeeds, then the actual link
				// publication fails. No mocked syscall or global hook is needed.
				flags["out"], message = f.path(strings.Repeat("x", 256)), "-out cannot be published"
			}
			before := trustSnapshot(t, f.dir)
			assertTrustFailure(t, f, trustArgs("trust-update", flags), 1, message)
			if !reflect.DeepEqual(before, trustSnapshot(t, f.dir)) {
				t.Fatal("publication refusal changed files or left staging")
			}
		})
	}
}

func TestUPD02_TrustR3FractionPrecision(t *testing.T) {
	for _, scenario := range []struct{ command, flag string }{{"trust-update", "at"}, {"trust-update", "not-after"}, {"trust-verify", "at"}} {
		t.Run(scenario.command+"/"+scenario.flag, func(t *testing.T) {
			f := newTrustFixture(t)
			flags := f.produceFlags()
			delete(flags, "add")
			flags["revoke"], flags["allow-no-usable-root"] = "A", "true"
			if scenario.flag == "not-after" {
				delete(flags, "revoke")
				flags["retire"] = "A"
			}
			flags[scenario.flag] = "2026-11-01T07:00:00.123456789-05:00"
			if scenario.command == "trust-verify" {
				f.success(trustArgs("trust-update", flags))
				flags = map[string]string{"update": f.path("update.json"), "trust": f.path("current.trust"), "at": flags["at"]}
			}
			f.success(trustArgs(scenario.command, flags))
			raw := readTestFile(t, f.path("update.json"))
			field := "revoked_at"
			if scenario.flag == "not-after" {
				field = "not_after"
			}
			if !bytes.Contains(raw, []byte(`"`+field+`":"2026-11-01T12:00:00.123456789Z"`)) {
				t.Fatal("nine-digit fraction not preserved in signed record")
			}
			// Use absent inputs: the time error must win before any file read.
			for _, name := range []string{"key", "trust", "update"} {
				if _, ok := flags[name]; ok {
					flags[name] = f.path("missing-" + name)
				}
			}
			if scenario.command == "trust-update" {
				if err := os.Remove(f.path("update.json")); err != nil {
					t.Fatal(err)
				}
			}
			for _, separator := range []string{".", ","} {
				flags[scenario.flag] = "2026-11-01T12:00:00" + separator + "1234567899Z"
				before := trustSnapshot(t, f.dir)
				assertTrustFailure(t, f, trustArgs(scenario.command, flags), 2, "-"+scenario.flag+" allows at most nine fractional digits")
				if !reflect.DeepEqual(before, trustSnapshot(t, f.dir)) {
					t.Fatal("time refusal wrote a file")
				}
			}
		})
	}
}

func TestUPD02_TrustR4SignatureIDSyntax(t *testing.T) {
	raw, err := fs.ReadFile(contracts.Schemas(), "trust/release-update-envelope.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatal(err)
	}
	pattern := schema["$defs"].(map[string]any)["Signature"].(map[string]any)["properties"].(map[string]any)["key_id"].(map[string]any)["pattern"]
	if pattern != trustSignerIDPattern {
		t.Fatalf("signature key ID syntax drift: schema=%v CLI=%s", pattern, trustSignerIDPattern)
	}
	for _, id := range []string{"release-B", "1._:-", strings.Repeat("a", 128)} {
		if !trustSignerID.MatchString(id) {
			t.Fatalf("valid ID rejected: %q", id)
		}
	}
	for _, id := range []string{"release B", "", "-B", "é", "a\n", strings.Repeat("a", 129)} {
		if trustSignerID.MatchString(id) {
			t.Fatalf("invalid ID accepted: %q", id)
		}
	}
}

func TestUPD02_TrustR4OpaqueIDHandoverAndRemoval(t *testing.T) {
	f := newTrustFixture(t)
	f.success([]string{"keygen", "-key-id", "release B", "-out", f.path("opaque.key"), "-trust-out", f.path("opaque.trust"), "-not-before", trustStart, "-not-after", trustEnd})
	opaque, err := release.LoadKey(f.path("opaque.key"))
	if err != nil {
		t.Fatal(err)
	}
	f.keys["opaque"] = opaque
	flags := f.produceFlags()
	flags["add"] = f.path("opaque.trust")
	assertTrustFailure(t, f, trustArgs("trust-update", flags), 1, "-add key_id cannot sign: must match "+trustSignerIDPattern)
	if _, err := os.Lstat(flags["out"]); !os.IsNotExist(err) {
		t.Fatal("opaque add wrote output")
	}
	// A malformed signer ID must be explained even before other input reads.
	flags["key"], flags["trust"] = f.path("opaque.key"), f.path("missing-trust")
	assertTrustFailure(t, f, trustArgs("trust-update", flags), 1, "-key key_id cannot sign: must match "+trustSignerIDPattern)
	// Observe the unchanged legacy sign command with the same key and a valid
	// manifest payload: it still rejects its signature key_id at schema checking.
	var manifest struct {
		Signed json.RawMessage `json:"signed"`
	}
	if err := json.Unmarshal(readTestFile(t, "../../../../contracts/vectors/release-manifest-signing.json"), &manifest); err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("manifest-signed.json"), manifest.Signed)
	var output, diagnostic bytes.Buffer
	if code := run([]string{"sign", "-signed", f.path("manifest-signed.json"), "-key", f.path("opaque.key"), "-out", f.path("manifest.json")}, &output, &diagnostic); code != 1 || output.Len() != 0 || !strings.Contains(diagnostic.String(), "manifest schema") {
		t.Fatalf("legacy sign observation: exit=%d stderr=%s", code, &diagnostic)
	}
	if _, err := os.Lstat(f.path("manifest.json")); !os.IsNotExist(err) {
		t.Fatal("legacy sign unexpectedly wrote output")
	}
	// Same public/private bytes, now under a signable ID, complete the handover.
	opaque.KeyID = "release-B"
	if err := opaque.Save(f.path("renamed.key")); err != nil {
		t.Fatal(err)
	}
	ts, err := opaque.TrustStore(trustDate(trustStart), trustDate(trustEnd))
	if err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("renamed.trust"), jsonTrustTest(t, ts))
	flags = f.produceFlags()
	flags["add"], flags["revision"] = f.path("renamed.trust"), "1"
	f.success(trustArgs("trust-update", flags))
	f.success(f.verifyArgs())
	f.apply(readTestFile(t, flags["out"]))
	f.export()
	delete(flags, "add")
	flags["key"], flags["retire"], flags["not-after"], flags["revision"], flags["out"] = f.path("renamed.key"), "A", trustNarrowEnd, "2", f.path("retire.json")
	f.success(trustArgs("trust-update", flags))
	f.apply(readTestFile(t, flags["out"]))
	f.export()
	delete(flags, "retire")
	delete(flags, "not-after")
	flags["revoke"], flags["revision"], flags["out"] = "A", "3", f.path("revoke.json")
	f.success(trustArgs("trust-update", flags))
	f.apply(readTestFile(t, flags["out"]))
	final := f.export()
	if old, _ := final.Lookup("A"); !old.Revoked {
		t.Fatal("handover did not revoke A")
	}
	if root, ok := final.Lookup("release-B"); !ok || root.Revoked || !bytes.Equal(root.PublicKey, opaque.Public()) {
		t.Fatal("renamed key did not become root")
	}

	// Existing opaque IDs remain valid record IDs for the centre and verifier,
	// and removable by a valid signer. The producer-only add rule is not applied.
	f = newTrustFixture(t)
	k := f.record("B")
	k.ID = "release B"
	raw := f.signed(f.update(k), "A")
	writeTrustTest(t, f.path("update.json"), raw)
	if out := f.success(f.verifyArgs()); !strings.Contains(out, `"id":"release B"`) {
		t.Fatal(out)
	}
	f.apply(raw)
	f.export()
	for i, operation := range []string{"retire", "revoke"} {
		flags = f.produceFlags()
		delete(flags, "add")
		flags[operation], flags["revision"], flags["out"] = "release B", []string{"11", "12"}[i], f.path(operation+"-opaque.json")
		if operation == "retire" {
			flags["not-after"] = trustNarrowEnd
		}
		f.success(trustArgs("trust-update", flags))
		f.success([]string{"trust-verify", "-update", flags["out"], "-trust", flags["trust"], "-at", trustAt})
		f.apply(readTestFile(t, flags["out"]))
		bundle := f.export()
		got, ok := bundle.Lookup("release B")
		if !ok || got.NotAfter != trustDate(trustNarrowEnd) || got.Revoked != (operation == "revoke") {
			t.Fatal("opaque target transition not applied")
		}
	}
}

func TestUPD02_TrustR5MixedPurposeDifference(t *testing.T) {
	f := newTrustFixture(t)
	// The provider generates and holds the lease private key; tests do not
	// inspect its files. Use one instant in both lease window and CLI checks.
	leaseID, err := f.provider.Rotate(context.Background(), signing.Lease, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC()
	f.export()
	u := f.update(f.record("B"))
	e, err := envelope.Sign(u, f.keys["A"].Private(), "A")
	if err != nil {
		t.Fatal(err)
	}
	leaseEnvelope, err := f.provider.SignEnvelope(context.Background(), signing.Lease, u)
	if err != nil {
		t.Fatal(err)
	}
	if leaseEnvelope.Signatures[0].KeyID != leaseID {
		t.Fatal("wrong lease signer")
	}
	e.Signatures = append(e.Signatures, leaseEnvelope.Signatures...)
	raw, err := e.MarshalCanonical()
	if err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("update.json"), raw)
	f.refuse([]string{"trust-verify", "-update", f.path("update.json"), "-trust", f.path("current.trust"), "-at", at.Format(time.RFC3339Nano)}, 1)
	// Documented difference: offline sees a known wrong-purpose key and rejects;
	// the centre's release-only bundle ignores that signature as unknown.
	f.apply(raw)
	if k, ok := f.export().Lookup("B"); !ok || k.Revoked {
		t.Fatal("centre did not apply mixed-purpose envelope")
	}
}

var _ io.Writer = (*trustResultFailure)(nil)

func TestUPD02_TrustR6RevisionMaySkip(t *testing.T) {
	f := newTrustFixture(t)
	flags := f.produceFlags() // revision 10, although the pinned centre is at 0.
	f.success(trustArgs("trust-update", flags))
	f.success(f.verifyArgs("-current-revision", "0"))
	raw := readTestFile(t, flags["out"])
	f.apply(raw)
	f.refuse(f.verifyArgs("-current-revision", "10"), 1)
	f.rejectCentre(raw)
}

func TestSEC01_TrustR2FlagDiagnostics(t *testing.T) {
	f := newTrustFixture(t)
	args := trustArgs("trust-update", f.produceFlags())
	assertTrustFailure(t, f, append(args, "-key", f.path("duplicate-private-key")), 2, "invalid flags: unknown, repeated or malformed flag")
	assertTrustFailure(t, f, append(args, "-"+f.path("unknown-flag")), 2, "invalid flags: unknown, repeated or malformed flag")
	var output, diagnostic bytes.Buffer
	if code := run([]string{"trust-verify", "-h"}, &output, &diagnostic); code != 0 || output.Len() != 0 || strings.Count(diagnostic.String(), "usage: fvpn-release") != 1 {
		t.Fatalf("help exit=%d stderr=%s", code, &diagnostic)
	}
}
