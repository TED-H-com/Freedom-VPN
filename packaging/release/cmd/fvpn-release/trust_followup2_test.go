package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
)

func TestUPD02_TrustR8ClosedStdoutProcess(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "fvpn-release")
	build := exec.CommandContext(t.Context(), "go", "build", "-o", bin, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build command: %v\n%s", err, output)
	}
	for _, command := range []string{"trust-update", "trust-verify", "keygen"} {
		t.Run(command, func(t *testing.T) {
			f := newTrustFixture(t)
			flags := f.produceFlags()
			args := trustArgs("trust-update", flags)
			wantMessage := "fvpn-release: trust-update: result output failed; published update file remains\n"
			if command == "trust-verify" {
				f.success(args)
				args = f.verifyArgs()
				wantMessage = "fvpn-release: trust-verify: result output failed\n"
			}
			if command == "keygen" {
				args = []string{"keygen", "-key-id", "TEST-ONLY-legacy", "-out", f.path("legacy.key")}
			}
			readEnd, writeEnd, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer writeEnd.Close()
			// Close the reader BEFORE Start, so failure is deterministic and the
			// child really writes descriptor 1 (no Go writer stub or shell pipe).
			if err := readEnd.Close(); err != nil {
				t.Fatal(err)
			}
			child := exec.CommandContext(t.Context(), bin, args...)
			child.Stdout = writeEnd
			var diagnostic bytes.Buffer
			child.Stderr = &diagnostic
			err = child.Run()
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("expected failing child, got %v", err)
			}
			status, ok := child.ProcessState.Sys().(syscall.WaitStatus)
			if !ok {
				t.Fatal("expected Linux wait status")
			}
			shellStatus, signalNumber := status.ExitStatus(), 0
			if status.Signaled() {
				signalNumber = int(status.Signal())
				shellStatus = 128 + signalNumber
			}
			t.Logf("%s: exit=%d signal=%d shell_status=%d stderr=%q", command, status.ExitStatus(), signalNumber, shellStatus, diagnostic.String())
			if command == "keygen" {
				// Baseline legacy commands do not check result writes. Preserve
				// SIGPIPE rather than turning a broken pipe into silent success.
				if !status.Signaled() || status.Signal() != syscall.SIGPIPE || diagnostic.Len() != 0 {
					t.Errorf("legacy closed-pipe behavior changed")
				}
				return
			}
			if status.Signaled() || status.ExitStatus() != 1 || diagnostic.String() != wantMessage {
				t.Errorf("closed pipe did not produce the fixed exit-1 diagnostic")
			}
			f.noPaths(diagnostic.String())
			f.noSeed(diagnostic.Bytes())
			before := readTestFile(t, flags["out"])
			// Verify the complete published file through another real process,
			// this time with a functioning stdout pipe owned by os/exec.
			verify := exec.CommandContext(t.Context(), bin, f.verifyArgs()...)
			output, err := verify.CombinedOutput()
			if err != nil {
				t.Fatalf("published file failed verification: %v\n%s", err, output)
			}
			if !bytes.Equal(before, readTestFile(t, flags["out"])) {
				t.Fatal("published file changed")
			}
			want, _, err := release.BuildTrustUpdate(f.keys["A"], mustCurrentTrust(t, f), release.TrustUpdateOptions{
				Revision: 10, Operation: "add", AddTrust: readTestFile(t, f.path("B.trust")), At: trustDate(trustAt),
			})
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, append(want, '\n')) {
				t.Fatal("published file was incomplete")
			}
			t.Log("published update remains complete and passes trust-verify")
		})
	}
}

func TestSEC01_TrustR9EmptyKeyIDLoadFailure(t *testing.T) {
	f := newTrustFixture(t)
	key := release.KeyFile{Purpose: "release", KeyID: "", SeedB64: base64.StdEncoding.EncodeToString(f.keys["A"].Seed)}
	raw := jsonTrustTest(t, key)
	writeTrustTest(t, f.path("empty-id.key"), raw)
	flags := f.produceFlags()
	flags["key"] = f.path("empty-id.key")
	before := trustSnapshot(t, f.dir)
	assertTrustFailure(t, f, trustArgs("trust-update", flags), 1, "trust-update: cannot load release signing key")
	f.refuse(trustArgs("trust-update", flags), 1)
	if _, err := os.Lstat(flags["out"]); !os.IsNotExist(err) {
		t.Fatal("empty-ID key produced output")
	}
	if !bytes.Equal(raw, readTestFile(t, flags["key"])) || len(before) != len(trustSnapshot(t, f.dir)) {
		t.Fatal("load failure changed files")
	}
}

func TestUPD02_TrustR10VerifierFractionalWindow(t *testing.T) {
	f := newTrustFixture(t)
	key := f.trusted("A")
	key.NotBefore = trustDate("2026-11-01T12:00:00.123456788Z")
	key.NotAfter = trustDate("2026-11-01T12:00:00.123456789Z")
	trust, err := envelope.NewNodeTrustStore(key)
	if err != nil {
		t.Fatal(err)
	}
	writeTrustTest(t, f.path("current.trust"), jsonTrustTest(t, trust))
	writeTrustTest(t, f.path("update.json"), f.signed(f.update(f.record("B")), "A"))
	for _, zone := range []struct {
		name   string
		offset int
	}{{"UTC", 0}, {"positive_offset", 5*3600 + 30*60}, {"negative_offset", -4 * 3600}} {
		t.Run(zone.name, func(t *testing.T) {
			for _, boundary := range []struct {
				name string
				at   time.Time
				code int
			}{
				{"at_not_before", key.NotBefore, 0},
				{"one_nanosecond_before", key.NotBefore.Add(-time.Nanosecond), 1},
				{"at_not_after", key.NotAfter, 1},
			} {
				t.Run(boundary.name, func(t *testing.T) {
					// Give helpers this subtest's T so each offset remains an
					// independent regression assertion when a boundary fails.
					fixture := *f
					fixture.t = t
					at := boundary.at.In(time.FixedZone(zone.name, zone.offset)).Format(time.RFC3339Nano)
					args := []string{"trust-verify", "-update", f.path("update.json"), "-trust", f.path("current.trust"), "-at", at}
					if boundary.code == 0 {
						fixture.success(args)
					} else {
						fixture.refuse(args, boundary.code)
					}
				})
			}
		})
	}
}
