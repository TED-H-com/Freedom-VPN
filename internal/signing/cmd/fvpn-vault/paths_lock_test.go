package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
)

func vaultAt(t *testing.T, path string) fixture {
	t.Helper()
	f := newVault(t, true)
	must(t, os.MkdirAll(filepath.Dir(path), 0700))
	must(t, os.Rename(f.dir, path))
	f.dir = path
	var err error
	f.p, err = signing.OpenFileProviderContext(context.Background(), path)
	must(t, err)
	return f
}
func startCommandChild(t *testing.T, testName, cwd string, args []string) (*exec.Cmd, *bytes.Buffer, *bytes.Buffer, context.Context, context.CancelFunc) {
	t.Helper()
	binary, err := os.Executable()
	must(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	child := exec.CommandContext(ctx, binary, append([]string{"-test.run=^" + testName + "$", "--"}, args...)...)
	child.Dir, child.WaitDelay = cwd, time.Second
	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	child.Stdout, child.Stderr = stdout, stderr
	if err := child.Start(); err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cancel()
		if child.ProcessState == nil {
			_ = child.Wait()
		}
	})
	return child, stdout, stderr, ctx, cancel
}
func waitCommandChild(t *testing.T, child *exec.Cmd, stdout, stderr *bytes.Buffer, ctx context.Context) outcome {
	t.Helper()
	err := child.Wait()
	if ctx.Err() != nil {
		t.Fatal("command child exceeded bounded deadline")
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
func childCommand(t *testing.T, cwd string, args ...string) outcome {
	t.Helper()
	child, stdout, stderr, ctx, cancel := startCommandChild(t, "TestSEC01_LockValidation", cwd, args)
	defer cancel()
	return waitCommandChild(t, child, stdout, stderr, ctx)
}
func requireUsage(t *testing.T, got outcome) {
	t.Helper()
	requireCode(t, got, exitUsage)
	const message = "fvpn-vault: usage: fvpn-vault inspect -vault EXISTING_VAULT [-check-keys] [-lock-timeout DURATION] | fvpn-vault export-trust -vault EXISTING_VAULT -out NEW_FILE [-lock-timeout DURATION]\n"
	if string(got.err) != message {
		t.Fatal("usage message changed")
	}
}
func TestSEC01_RejectsParentPathElements(t *testing.T) {
	root := t.TempDir()
	lexical := vaultAt(t, filepath.Join(root, "lexical", "vault"))
	physical := vaultAt(t, filepath.Join(root, "physical", "vault"))
	must(t, physical.p.DenyIdentity(context.Background(), uuid.NewString()))
	must(t, os.Mkdir(filepath.Join(root, "physical", "child"), 0700))
	must(t, os.Symlink(filepath.Join(root, "physical", "child"), filepath.Join(root, "lexical", "jump")))
	rawVault := root + "/lexical/jump/../vault"
	named, err := os.Stat(rawVault + "/state.json")
	must(t, err)
	expected, err := os.Stat(physical.dir + "/state.json")
	must(t, err)
	if !os.SameFile(named, expected) {
		t.Fatal("fixture does not reproduce OS resolution")
	}
	a, err := lexical.p.Snapshot(context.Background())
	must(t, err)
	b, err := physical.p.Snapshot(context.Background())
	must(t, err)
	if a.Revision == b.Revision || bytes.Equal(a.Keys[0].Public, b.Keys[0].Public) {
		t.Fatal("vault fixtures are not distinguishable")
	}
	before := manifest(t, root)
	for _, args := range commands(rawVault, physical.dir+"/forbidden.json") {
		var stdout, stderr bytes.Buffer
		code := run(args, &stdout, &stderr)
		requireUsage(t, outcome{code, stdout.Bytes(), stderr.Bytes()})
		if !reflect.DeepEqual(before, manifest(t, root)) {
			t.Fatal("parent-element refusal touched a vault or created output")
		}
	}
	// Also reject an ambiguous output, even with an unambiguous vault value.
	for _, out := range []string{root + "/lexical/jump/../vault/forbidden.json", root + "/lexical/jump/../new.json"} {
		requireUsage(t, invoke(t, command{}, lexical.dir, "export-trust", "-vault", lexical.dir, "-out", out))
		if !reflect.DeepEqual(before, manifest(t, root)) {
			t.Fatal("ambiguous output touched test tree")
		}
	}
}
func TestSEC01_ParentElementSpellings(t *testing.T) {
	root := t.TempDir()
	f := vaultAt(t, filepath.Join(root, "vault"))
	cwd := filepath.Join(root, "work")
	must(t, os.Mkdir(cwd, 0700))
	must(t, os.Mkdir(filepath.Join(f.dir, "child"), 0700))
	for _, path := range []string{"../vault", f.dir + "/..", "..", f.dir + "/../vault"} {
		for _, suffix := range []string{"", "/"} {
			before := manifest(t, root)
			for _, args := range commands(path+suffix, filepath.Join(cwd, "out.json")) {
				requireUsage(t, childCommand(t, cwd, args...))
				if !reflect.DeepEqual(before, manifest(t, root)) {
					t.Fatal("vault path refusal changed test tree")
				}
			}
		}
	}
	for _, path := range []string{"../vault", f.dir + "/..", "..", f.dir + "/../vault", "../out.json", cwd + "/..", cwd + "/../out.json", f.dir + "/child/../out.json"} {
		for _, suffix := range []string{"", "/"} {
			before := manifest(t, root)
			requireUsage(t, childCommand(t, cwd, "export-trust", "-vault", f.dir, "-out", path+suffix))
			if !reflect.DeepEqual(before, manifest(t, root)) {
				t.Fatal("output path refusal changed test tree")
			}
		}
	}
}
func comparePublicReport(t *testing.T, f fixture, raw []byte, checked bool) {
	t.Helper()
	requireReportInventory(t, f.dir, raw)
	snapshot, err := f.p.Snapshot(context.Background())
	must(t, err)
	// Compare typed revisions for either document; both use the same JSON names.
	var meta struct {
		LedgerRevision  decstr.U64 `json:"ledger_revision"`
		ReleaseRevision decstr.U64 `json:"release_revision"`
		KeyUseChecked   bool       `json:"key_use_checked"`
	}
	must(t, json.Unmarshal(raw, &meta))
	if meta.LedgerRevision != snapshot.Revision || meta.ReleaseRevision != snapshot.ReleaseRevision || meta.KeyUseChecked != checked {
		t.Fatal("path spelling selected different revision/check result")
	}
}
func TestSEC01_AcceptedPathSpellings(t *testing.T) {
	root := t.TempDir()
	f := vaultAt(t, filepath.Join(root, "a", "vault"))
	for _, tc := range []struct{ cwd, path string }{
		{root + "/a", "vault"}, {root + "/a", "./vault"}, {root, "a//vault"}, {root, "a/./vault"}, {root, "a/vault/"},
	} {
		for i, args := range commands(tc.path, filepath.Join(t.TempDir(), "trust.json")) {
			before := manifest(t, f.dir)
			got := childCommand(t, tc.cwd, args...)
			requireCode(t, got, exitOK)
			comparePublicReport(t, f, got.out, i == 1)
			if !reflect.DeepEqual(before, manifest(t, f.dir)) {
				t.Fatal("accepted spelling mutated vault")
			}
		}
	}
	for _, name := range []string{"..data", "a..b"} {
		f := vaultAt(t, filepath.Join(root, name))
		output := filepath.Join(root, name+".json")
		for i, args := range commands(name, output) {
			got := childCommand(t, root, args...)
			requireCode(t, got, exitOK)
			comparePublicReport(t, f, got.out, i == 1)
		}
		if _, err := os.Stat(output); err != nil {
			t.Fatal("two-dot filename was not published")
		}
	}
	alias := filepath.Join(root, "alias")
	must(t, os.Symlink(f.dir, alias))
	for _, suffix := range []string{"/", "/."} {
		for _, args := range commands(alias+suffix, filepath.Join(t.TempDir(), "out.json")) {
			got := invoke(t, command{}, f.dir, args...)
			requireCode(t, got, exitUnavailable)
			if string(got.err) != "fvpn-vault: vault unavailable\n" {
				t.Fatal("final symlink refusal changed")
			}
		}
	}
}
func TestSEC01_RelativePathsFromLinkedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	f := vaultAt(t, filepath.Join(root, "physical", "vault"))
	alias := filepath.Join(root, "linked-working-directory")
	must(t, os.Symlink(filepath.Dir(f.dir), alias))
	before := manifest(t, f.dir)
	got := childCommand(t, alias, "export-trust", "-vault", "./vault", "-out", "./trust.json")
	requireCode(t, got, exitOK)
	comparePublicReport(t, f, got.out, false)
	physicalOutput := filepath.Join(root, "physical", "trust.json")
	a, err := os.Stat(physicalOutput)
	must(t, err)
	b, err := os.Stat(alias + "/trust.json")
	must(t, err)
	if !os.SameFile(a, b) {
		t.Fatal("relative output differs from OS resolution")
	}
	raw, err := os.ReadFile(physicalOutput)
	must(t, err)
	var receipt exportReceipt
	must(t, json.Unmarshal(got.out, &receipt))
	if receipt.SHA256 != digest(raw) {
		t.Fatal("relative output receipt mismatch")
	}
	if !reflect.DeepEqual(before, manifest(t, f.dir)) {
		t.Fatal("relative invocation changed vault")
	}
	if _, err := os.Lstat(filepath.Join(root, "trust.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("relative output appeared in wrong directory")
	}
}

func replaceLock(t *testing.T, dir, fault, outside string) {
	t.Helper()
	lock := filepath.Join(dir, "lock")
	if fault == "owner_bits" {
		must(t, os.Chmod(lock, 0400))
		return
	}
	if fault == "recreate_same" || fault == "recreate_different" {
		must(t, os.Remove(lock))
		var content []byte // The initializer's original lock content is empty.
		if fault == "recreate_different" {
			content = []byte("TEST ONLY replacement content")
		}
		must(t, os.WriteFile(lock, content, 0600))
		return
	}
	must(t, os.Rename(lock, filepath.Join(dir, "original-lock")))
	switch fault {
	case "symlink":
		must(t, os.Symlink(outside, lock))
	case "fifo":
		must(t, syscall.Mkfifo(lock, 0600))
	case "group_bits":
		must(t, os.WriteFile(lock, nil, 0600))
		must(t, os.Chmod(lock, 0660))
	case "regular":
		must(t, os.WriteFile(lock, nil, 0600))
	case "missing":
	default:
		t.Fatal("unknown replacement fixture")
	}
}
func TestSEC01_LockReplacementAtCheckpoints(t *testing.T) {
	// Keep every injected case, including FIFO regressions, killable by its
	// parent test without relying on the provider to return from an open.
	if len(os.Args) > 5 && os.Args[2] == "--" {
		stage, fault, outside := os.Args[3], os.Args[4], os.Args[5]
		args := os.Args[6:]
		o, ok := parse(args)
		if !ok {
			t.Fatal("invalid child arguments")
		}
		c := command{}
		var after map[string]entry
		mutation := func() { replaceLock(t, o.vault, fault, outside); after = manifest(t, o.vault) }
		if stage == "between_snapshots" {
			c.afterProbes = mutation
		} else {
			c.beforeResult = mutation
		}
		var stdout, stderr bytes.Buffer
		code := c.run(args, &stdout, &stderr)
		if after == nil || !reflect.DeepEqual(after, manifest(t, o.vault)) {
			t.Fatal("command mutated externally replaced vault")
		}
		_, _ = os.Stdout.Write(stdout.Bytes())
		_, _ = os.Stderr.Write(stderr.Bytes())
		os.Exit(code)
	}
	for _, stage := range []string{"between_snapshots", "before_result"} {
		for _, fault := range []string{"symlink", "fifo", "group_bits", "regular", "missing", "owner_bits", "recreate_same", "recreate_different"} {
			t.Run(stage+"/"+fault, func(t *testing.T) {
				for i := 0; i < 3; i++ {
					if stage == "between_snapshots" && i != 1 {
						continue
					}
					f := newVault(t, true)
					outside := filepath.Join(t.TempDir(), "external-lock")
					must(t, os.WriteFile(outside, []byte("TEST ONLY external lock"), 0600))
					outsideBefore := manifest(t, outside)
					outputDir := t.TempDir()
					args := commands(f.dir, filepath.Join(outputDir, "trust.json"))[i]
					child, stdout, stderr, ctx, cancel := startCommandChild(t, "TestSEC01_LockReplacementAtCheckpoints", "", append([]string{stage, fault, outside}, args...))
					got := waitCommandChild(t, child, stdout, stderr, ctx)
					cancel()
					requireChanged(t, got)
					entries, err := os.ReadDir(outputDir)
					must(t, err)
					if len(entries) != 0 {
						t.Fatal("lock change left an output name")
					}
					if !reflect.DeepEqual(outsideBefore, manifest(t, outside)) {
						t.Fatal("external lock target changed")
					}
				}
			})
		}
	}
}
func requireChanged(t *testing.T, got outcome) {
	t.Helper()
	requireCode(t, got, exitChanged)
	if string(got.err) != "fvpn-vault: changed during inspection\n" {
		t.Fatal("lock change outcome not controlled")
	}
}
func TestSEC01_LockReplacementWhileWaiting(t *testing.T) {
	for _, fault := range []string{"symlink", "fifo", "group_bits", "regular", "missing", "recreate_same", "recreate_different"} {
		t.Run(fault, func(t *testing.T) {
			for i := 0; i < 3; i++ {
				f := newVault(t, true)
				lock := filepath.Join(f.dir, "lock")
				held, err := os.OpenFile(lock, os.O_RDWR, 0)
				must(t, err)
				t.Cleanup(func() { _ = held.Close() })
				must(t, syscall.Flock(int(held.Fd()), syscall.LOCK_EX|syscall.LOCK_NB))
				outside := filepath.Join(t.TempDir(), "external-lock")
				must(t, os.WriteFile(outside, []byte("TEST ONLY external lock"), 0600))
				outputDir := t.TempDir()
				// IN_OPEN establishes that the unmodified command passed its initial
				// validation and opened the held inode. The event is bounded by a read deadline.
				fd, err := syscall.InotifyInit1(syscall.IN_CLOEXEC | syscall.IN_NONBLOCK)
				must(t, err)
				events := os.NewFile(uintptr(fd), "lock-open-events")
				t.Cleanup(func() { _ = events.Close() })
				_, err = syscall.InotifyAddWatch(fd, lock, syscall.IN_OPEN)
				must(t, err)
				must(t, events.SetReadDeadline(time.Now().Add(5*time.Second)))
				child, stdout, stderr, ctx, cancel := startCommandChild(t, "TestSEC01_LockValidation", "", commands(f.dir, filepath.Join(outputDir, "trust.json"))[i])
				var event [4096]byte
				n, err := events.Read(event[:])
				must(t, err)
				if n < syscall.SizeofInotifyEvent {
					t.Fatal("no lock open event")
				}
				must(t, events.Close())
				replaceLock(t, f.dir, fault, outside)
				afterMutation := manifest(t, f.dir)
				must(t, syscall.Flock(int(held.Fd()), syscall.LOCK_UN))
				must(t, held.Close())
				got := waitCommandChild(t, child, stdout, stderr, ctx)
				cancel()
				requireChanged(t, got)
				if !reflect.DeepEqual(afterMutation, manifest(t, f.dir)) {
					t.Fatal("waiting command mutated replaced vault")
				}
				entries, err := os.ReadDir(outputDir)
				must(t, err)
				if len(entries) != 0 {
					t.Fatal("waiting command created export output")
				}
			}
		})
	}
}
