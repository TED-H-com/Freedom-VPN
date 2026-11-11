package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/envelope"
	"github.com/google/uuid"
)

type fixture struct {
	p       *signing.FileProvider
	dir, id string
}

func setup(t *testing.T, purpose signing.Purpose) fixture {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "vault")
	p, err := signing.InitializeFileProvider(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, err := p.Rotate(context.Background(), purpose, time.Hour, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	return fixture{p, dir, id}
}
func (f fixture) args(op string) []string {
	if op == "rotate" {
		return []string{"rotate", "-vault", f.dir, "-purpose", "lease", "-expect-revision", "1", "-expect-active-key", f.id, "-validity", "1h", "-overlap", "1m"}
	}
	return []string{"revoke", "-vault", f.dir, "-key-id", f.id, "-expect-purpose", "lease", "-expect-revision", "1", "-revoke-active"}
}
func flagValue(args []string, name, value string) []string {
	args = append([]string(nil), args...)
	for i := range args {
		if args[i] == "-"+name {
			args[i+1] = value
			return args
		}
	}
	panic("flag missing")
}
func getSnapshot(t *testing.T, p *signing.FileProvider) signing.FileSnapshot {
	t.Helper()
	s, err := p.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func assertOutput(t *testing.T, code int, stdout, stderr string, wantCode int, wantOutcome, wantStderr string) document {
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
		t.Fatal("no result document", err)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		t.Fatal("more than one document")
	}
	if d.Outcome != wantOutcome || d.Message == "" || (d.Kind != "fvpn-file-vault-rotate-v1" && d.Kind != "fvpn-file-vault-revoke-v1") {
		t.Fatalf("unexpected document: %+v", d)
	}
	return d
}
func invoke(t *testing.T, c command, args []string, code int, outcome string) document {
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
	return assertOutput(t, got, out.String(), err.String(), code, outcome, stderr)
}

type fileFact struct {
	Size     int64
	Mode     os.FileMode
	Modified time.Time
	Hash     [32]byte
}

func manifest(t *testing.T, root string) map[string]fileFact {
	t.Helper()
	out := map[string]fileFact{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		var data []byte
		if info.Mode().IsRegular() {
			data, err = os.ReadFile(path)
			if err != nil {
				return err
			}
		} else if info.Mode()&os.ModeSymlink != 0 {
			target, e := os.Readlink(path)
			if e != nil {
				return e
			}
			data = []byte(target)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[rel] = fileFact{info.Size(), info.Mode(), info.ModTime(), sha256.Sum256(data)}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
func unchanged(t *testing.T, root string, before map[string]fileFact) {
	t.Helper()
	if !reflect.DeepEqual(before, manifest(t, root)) {
		t.Fatal("vault manifest changed")
	}
}
func TestSEC01_CommandRotationPurposes(t *testing.T) {
	for _, p := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		t.Run(string(p), func(t *testing.T) {
			f := setup(t, p)
			before := getSnapshot(t, f.p)
			args := flagValue(f.args("rotate"), "purpose", string(p))
			start := time.Now().UTC()
			d := invoke(t, command{}, args, exitOK, "applied")
			end := time.Now().UTC()
			if d.Change == nil {
				t.Fatal("missing change")
			}
			change := d.Change
			k := change.Key
			old := change.Previous
			if change.OldRevision != 1 || change.NewRevision != 2 || k.ID == f.id || k.State != "active" || k.Purpose != p || k.NotBefore.Before(start) || k.NotBefore.After(end) || k.NotAfter.Sub(k.NotBefore) != time.Hour || old.ID != f.id || old.State != "retired" || !old.NotAfter.Equal(k.NotBefore.Add(time.Minute)) {
				t.Fatal("wrong public rotation facts")
			}
			wantTrust := "nodes_require_new_bundle"
			if p == signing.LedgerEpoch {
				wantTrust = "node_bundle_unchanged"
			}
			if change.NodeTrust != wantTrust || change.NoSigner {
				t.Fatal("wrong trust/signer advice")
			}
			after := getSnapshot(t, f.p)
			sum := sha256.Sum256(after.Keys[1].Public)
			if k.PublicSHA256 != hex.EncodeToString(sum[:]) {
				t.Fatal("wrong fingerprint")
			}
			if p == signing.LedgerEpoch {
				a, _ := json.Marshal(before.NodeTrust)
				b, _ := json.Marshal(after.NodeTrust)
				if !bytes.Equal(a, b) {
					t.Fatal("node bundle changed")
				}
			}
			unchangedBefore := manifest(t, f.dir)
			d = invoke(t, command{}, args, exitRefused, "precondition_refused")
			unchanged(t, f.dir, unchangedBefore)
			if d.Change != nil || d.AttemptedKeyID != "" || d.UnreferencedKeyID != "" {
				t.Fatal("refusal echoes ledger")
			}
		})
	}
}
func TestSEC01_CommandRevocation(t *testing.T) {
	for _, purpose := range []signing.Purpose{signing.DesiredState, signing.Lease, signing.Recovery, signing.LedgerEpoch} {
		t.Run(string(purpose), func(t *testing.T) {
			f := setup(t, purpose)
			args := flagValue(f.args("revoke"), "expect-purpose", string(purpose))
			before := manifest(t, f.dir)
			invoke(t, command{}, args[:len(args)-1], exitRefused, "precondition_refused")
			unchanged(t, f.dir, before)
			d := invoke(t, command{}, args, exitOK, "revoked")
			if d.Change.Key.ID != f.id || d.Change.Key.Purpose != purpose || d.Change.Key.RevokedAt == nil || d.Change.Previous.State != "active" || !d.Change.NoSigner || d.Change.OldRevision != 1 || d.Change.NewRevision != 2 {
				t.Fatal("revocation details")
			}
			if (d.Change.NodeTrust == "node_bundle_unchanged") != (purpose == signing.LedgerEpoch) {
				t.Fatal("wrong trust advice")
			}
			at := *d.Change.Key.RevokedAt
			before = manifest(t, f.dir)
			repeat := invoke(t, command{}, args, 0, "already_revoked")
			unchanged(t, f.dir, before)
			if !repeat.Change.Key.RevokedAt.Equal(at) {
				t.Fatal("changed revocation time")
			}
			if _, err := f.p.SignEnvelope(context.Background(), purpose, map[string]bool{"test_only": true}); err == nil {
				t.Fatal("revoked signer usable")
			}
			rotate := flagValue(flagValue(flagValue(f.args("rotate"), "purpose", string(purpose)), "expect-revision", "2"), "expect-active-key", "none")
			next := invoke(t, command{}, rotate, exitOK, "applied")
			if next.Change.Previous != nil {
				t.Fatal("revoked key retired")
			}
			before = manifest(t, f.dir)
			repeat = invoke(t, command{}, args, exitAlreadyRevoked, "already_revoked")
			unchanged(t, f.dir, before)
			if repeat.Change.NoSigner || !repeat.Change.Key.RevokedAt.Equal(at) {
				t.Fatal("wrong repeated revocation facts")
			}
			if s := getSnapshot(t, f.p); s.Keys[0].State != "revoked" || !s.Keys[0].RevokedAt.Equal(at) {
				t.Fatal("resurrected key")
			}
		})
	}
	f := setup(t, signing.Lease)
	invoke(t, command{}, f.args("rotate"), exitOK, "applied")
	a := flagValue(f.args("revoke"), "expect-revision", "2")
	d := invoke(t, command{}, a[:len(a)-1], exitOK, "revoked")
	if d.Change.Previous.State != "retired" || d.Change.NoSigner {
		t.Fatal("retired revocation")
	}
}
func TestSEC01_CommandRefusals(t *testing.T) {
	f := setup(t, signing.Lease)
	before := manifest(t, f.dir)
	for _, args := range [][]string{
		flagValue(f.args("rotate"), "expect-revision", "0"), flagValue(f.args("rotate"), "expect-active-key", uuid.NewString()), flagValue(f.args("rotate"), "expect-active-key", "none"), flagValue(f.args("rotate"), "purpose", "recovery"),
		flagValue(f.args("revoke"), "expect-revision", "0"), flagValue(f.args("revoke"), "expect-purpose", "recovery"), flagValue(f.args("revoke"), "key-id", uuid.NewString()),
	} {
		invoke(t, command{}, args, exitRefused, "precondition_refused")
		unchanged(t, f.dir, before)
	}
	for _, purpose := range []signing.Purpose{signing.NodeCA, signing.BootstrapSecret, signing.Release} {
		var id string
		if purpose == signing.Release {
			pub, _, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			id = uuid.NewString()
			now := time.Now().UTC()
			if err := f.p.PinReleaseRoot(context.Background(), envelope.TrustedKey{KeyID: id, PublicKey: pub, Purposes: []envelope.Purpose{envelope.PurposeRelease}, NotBefore: now, NotAfter: now.Add(time.Hour)}); err != nil {
				t.Fatal(err)
			}
		} else {
			var err error
			id, err = f.p.Rotate(context.Background(), purpose, time.Hour, 0)
			if err != nil {
				t.Fatal(err)
			}
		}
		before = manifest(t, f.dir)
		a := flagValue(f.args("revoke"), "key-id", id)
		invoke(t, command{}, a, exitRefused, "precondition_refused")
		unchanged(t, f.dir, before)
		invoke(t, command{}, flagValue(a, "expect-purpose", string(purpose)), exitUsage, "")
		unchanged(t, f.dir, before)
	}
}
func TestSEC01_CommandArguments(t *testing.T) {
	f := setup(t, signing.Lease)
	before := manifest(t, f.dir)
	for _, op := range []string{"rotate", "revoke"} {
		base := f.args(op)
		cases := [][]string{nil, {"unknown"}, {op}, append(append([]string{}, base...), "extra"), append(append([]string{}, base...), "-unknown"), append(append([]string{}, base...), "-lock-timeout", "0"), append(append([]string{}, base...), "-lock-timeout", "-1s"), append(append([]string{}, base...), "-lock-timeout", "bad"), append(append([]string{}, base...), "-lock-timeout", "1s", "-lock-timeout", "1s")}
		for i := 1; i < len(base); i++ {
			if !strings.HasPrefix(base[i], "-") {
				continue
			}
			n := 2
			if base[i] == "-revoke-active" {
				n = 1
			}
			if n == 2 {
				without := append([]string{}, base[:i]...)
				without = append(without, base[i+n:]...)
				cases = append(cases, without)
			}
			repeat := append([]string{}, base...)
			repeat = append(repeat, base[i:i+n]...)
			cases = append(cases, repeat)
			if n == 2 {
				cases = append(cases, append(append([]string{}, base...), base[i]))
			}
		}
		for _, revision := range []string{"", "-1", "+1", "01", " 1", "1.0", "1e0", "18446744073709551616", "999999999999999999999"} {
			cases = append(cases, flagValue(base, "expect-revision", revision))
		}
		purposeFlag, idFlag := "purpose", "expect-active-key"
		if op == "revoke" {
			purposeFlag, idFlag = "expect-purpose", "key-id"
		}
		for _, p := range []string{"node-ca", "bootstrap-secret", "release", "unknown", ""} {
			cases = append(cases, flagValue(base, purposeFlag, p))
		}
		for _, id := range []string{"", "bad", strings.ReplaceAll(f.id, "-", ""), "urn:uuid:" + f.id, "{" + f.id + "}"} {
			cases = append(cases, flagValue(base, idFlag, id))
		}
		cases = append(cases, flagValue(base, "vault", ""))
		if op == "rotate" {
			for _, v := range []string{"0", "-1s", "bad", "999999999999999999h"} {
				cases = append(cases, flagValue(base, "validity", v))
			}
			for _, v := range []string{"-1ns", "bad", "999999999999999999h"} {
				cases = append(cases, flagValue(base, "overlap", v))
			}
		}
		for i, a := range cases {
			t.Run(op+"/"+string(rune('A'+i)), func(t *testing.T) { invoke(t, command{}, a, exitUsage, ""); unchanged(t, f.dir, before) })
		}
	}
}
func TestSEC01_CommandNoLeakageAndPermissions(t *testing.T) {
	for _, mask := range []int{0, 0077} {
		for _, op := range []string{"rotate", "revoke"} {
			f := setup(t, signing.Lease)
			var out, errOut bytes.Buffer
			old := syscall.Umask(mask)
			code := run(f.args(op), &out, &errOut)
			syscall.Umask(old)
			outcome := "applied"
			if op == "revoke" {
				outcome = "revoked"
			}
			assertOutput(t, code, out.String(), errOut.String(), exitOK, outcome, "")
			entries, err := os.ReadDir(f.dir)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if !strings.HasSuffix(entry.Name(), ".key") {
					continue
				}
				info, err := entry.Info()
				if err != nil {
					t.Fatal(err)
				}
				if info.Mode().Perm() != 0600 {
					t.Fatal("key mode")
				}
				raw, err := os.ReadFile(filepath.Join(f.dir, entry.Name()))
				if err != nil {
					t.Fatal(err)
				}
				key, err := x509.ParsePKCS8PrivateKey(raw)
				if err != nil {
					t.Fatal(err)
				}
				private := key.(ed25519.PrivateKey)
				for _, secret := range [][]byte{raw, private, private.Seed()} {
					for _, value := range []string{string(secret), hex.EncodeToString(secret), base64.StdEncoding.EncodeToString(secret)} {
						if strings.Contains(out.String()+errOut.String(), value) {
							t.Fatal("private material leaked")
						}
					}
				}
			}
			if strings.Contains(out.String()+errOut.String(), f.dir) {
				t.Fatal("path leaked")
			}
		}
	}
}

func TestSEC01_CommandExitCodeTable(t *testing.T) {
	got := map[string]int{
		"applied": exitOK, "revoked": exitOK, "already_revoked": exitAlreadyRevoked,
		"usage": exitUsage, "unavailable": exitUnavailable, "deadline": exitTimeout,
		"refused": exitRefused, "not_applied": exitNotApplied, "durability": exitDurability,
		"unknown": exitUnknown, "output": exitOutput,
	}
	want := map[string]int{
		"applied": 0, "revoked": 0, "already_revoked": 0,
		"usage": 2, "unavailable": 3, "deadline": 4,
		"refused": 5, "not_applied": 6, "durability": 7,
		"unknown": 8, "output": 9,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("exit codes = %v; want %v", got, want)
	}
}

func TestSEC01_CommandLedgerOwnerPermissions(t *testing.T) {
	for _, operation := range []string{"rotate", "revoke"} {
		t.Run(operation, func(t *testing.T) {
			f := setup(t, signing.Lease)
			args := f.args(operation)
			wantSigner := f.id
			outcome := "applied"
			if operation == "revoke" {
				var err error
				wantSigner, err = f.p.Rotate(context.Background(), signing.Lease, time.Hour, time.Minute)
				if err != nil {
					t.Fatal(err)
				}
				args = flagValue(args[:len(args)-1], "expect-revision", "2")
				outcome = "revoked"
			}
			var d document
			func() {
				previous := syscall.Umask(0777)
				defer syscall.Umask(previous)
				d = invoke(t, command{}, args, 0, outcome)
			}()
			if operation == "rotate" {
				wantSigner = d.Change.Key.ID
			}
			info, err := os.Stat(filepath.Join(f.dir, "state.json"))
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0600 {
				t.Errorf("ledger mode = %04o; want 0600", info.Mode().Perm())
			}
			p, err := signing.OpenFileProvider(f.dir)
			if err != nil {
				t.Fatal("cannot reopen saved vault", err)
			}
			s := getSnapshot(t, p)
			if s.Revision != d.Change.NewRevision {
				t.Fatal("snapshot disagrees with command result")
			}
			signed, err := p.SignEnvelope(context.Background(), signing.Lease, map[string]bool{"test_only": true})
			if err != nil || signed.Signatures[0].KeyID != wantSigner {
				t.Fatal("expected signer unavailable after command", err)
			}
		})
	}
}
