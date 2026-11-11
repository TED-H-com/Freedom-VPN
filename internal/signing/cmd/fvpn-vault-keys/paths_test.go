package main

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
)

func TestSEC01_CommandParentPaths(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		f := setup(t, signing.Lease)
		before := manifest(t, f.dir)
		for _, path := range []string{"..", "../", "../vault", f.dir + "/..", f.dir + "/../", f.dir + "/a/../b", f.dir + "//..//vault", f.dir + "/./../vault", f.dir + "/../vault/."} {
			invoke(t, command{}, flagValue(f.args(op), "vault", path), exitUsage, "")
			unchanged(t, f.dir, before)
		}
		// Lexical cleaning selects local/vault; OS traversal through local/link
		// selects remote/vault. Both are real vaults and neither may be touched.
		root := t.TempDir()
		local := filepath.Join(root, "local")
		remote := filepath.Join(root, "remote")
		for _, dir := range []string{local, remote, filepath.Join(remote, "child")} {
			if err := os.Mkdir(dir, 0700); err != nil {
				t.Fatal(err)
			}
		}
		for _, parent := range []string{local, remote} {
			p, err := signing.InitializeFileProvider(filepath.Join(parent, "vault"))
			if err != nil {
				t.Fatal(err)
			}
			if _, err = p.Rotate(context.Background(), signing.Lease, time.Hour, 0); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Symlink(filepath.Join(remote, "child"), filepath.Join(local, "link")); err != nil {
			t.Fatal(err)
		}
		both := manifest(t, root)
		invoke(t, command{}, flagValue(f.args(op), "vault", local+"/link/../vault"), exitUsage, "")
		unchanged(t, root, both)
		name := filepath.Join(filepath.Dir(f.dir), "..data")
		if err := os.Rename(f.dir, name); err != nil {
			t.Fatal(err)
		}
		outcome := "applied"
		if op == "revoke" {
			outcome = "revoked"
		}
		invoke(t, command{}, flagValue(f.args(op), "vault", name), exitOK, outcome)
	}
}
func TestSEC01_CommandUnavailableVault(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		for _, kind := range []string{"missing path", "final symlink", "final symlink slash", "final symlink dot", "missing lock", "missing state", "lock group", "lock other", "lock symlink inside", "lock symlink outside", "lock fifo", "lock directory", "directory mode", "state mode", "invalid ledger"} {
			t.Run(op+"/"+kind, func(t *testing.T) {
				f := setup(t, signing.Lease)
				path := f.dir
				lock := filepath.Join(path, "lock")
				state := filepath.Join(path, "state.json")
				root := filepath.Dir(path)
				removeLock := func() {
					t.Helper()
					if err := os.Remove(lock); err != nil {
						t.Fatal(err)
					}
				}
				must := func(err error) {
					t.Helper()
					if err != nil {
						t.Fatal(err)
					}
				}
				switch kind {
				case "missing path":
					path = filepath.Join(root, "missing")
				case "final symlink", "final symlink slash", "final symlink dot":
					path = filepath.Join(root, "alias")
					must(os.Symlink(f.dir, path))
					if kind == "final symlink slash" {
						path += "/"
					}
					if kind == "final symlink dot" {
						path += "/."
					}
				case "missing lock":
					removeLock()
				case "missing state":
					must(os.Remove(state))
				case "lock group":
					must(os.Chmod(lock, 0640))
				case "lock other":
					must(os.Chmod(lock, 0604))
				case "lock symlink inside":
					removeLock()
					must(os.Symlink("state.json", lock))
				case "lock symlink outside":
					removeLock()
					target := filepath.Join(root, "outside")
					must(os.WriteFile(target, []byte("TEST ONLY"), 0600))
					must(os.Symlink(target, lock))
				case "lock fifo":
					removeLock()
					must(syscall.Mkfifo(lock, 0600))
				case "lock directory":
					removeLock()
					must(os.Mkdir(lock, 0700))
				case "directory mode":
					must(os.Chmod(path, 0750))
				case "state mode":
					must(os.Chmod(state, 0604))
				case "invalid ledger":
					must(os.WriteFile(state, []byte(`{"revision":"invalid"}`), 0600))
				}
				before := manifest(t, root)
				// This call is synchronous: O_PATH cannot wait for a FIFO writer. The
				// child-process version below additionally enforces an external deadline.
				invoke(t, command{}, flagValue(f.args(op), "vault", path), exitUnavailable, "")
				unchanged(t, root, before)
			})
		}
	}
}
func TestSEC01_CommandSymlinkedParent(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		f := setup(t, signing.Lease)
		alias := filepath.Join(t.TempDir(), "parent")
		if err := os.Symlink(filepath.Dir(f.dir), alias); err != nil {
			t.Fatal(err)
		}
		outcome := "applied"
		if op == "revoke" {
			outcome = "revoked"
		}
		invoke(t, command{}, flagValue(f.args(op), "vault", alias+"//./vault/"), exitOK, outcome)
	}
}
func TestSEC01_CommandHeldLock(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		f := setup(t, signing.Lease)
		lock, err := os.OpenFile(filepath.Join(f.dir, "lock"), os.O_RDWR, 0)
		if err != nil {
			t.Fatal(err)
		}
		if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
			t.Fatal(err)
		}
		args := append(f.args(op), "-lock-timeout", "20ms")
		before := manifest(t, f.dir)
		invoke(t, command{}, args, exitTimeout, "")
		unchanged(t, f.dir, before)
		if err := lock.Close(); err != nil {
			t.Fatal(err)
		}
		outcome := "applied"
		if op == "revoke" {
			outcome = "revoked"
		}
		invoke(t, command{}, args, exitOK, outcome)
	}
}
func TestSEC01_CommandLockInUse(t *testing.T) {
	for _, op := range []string{"rotate", "revoke"} {
		for _, kind := range []string{"group", "symlink", "typed replacement rejection", "acceptable replacement"} {
			t.Run(op+"/"+kind, func(t *testing.T) {
				f := setup(t, signing.Lease)
				var before map[string]fileFact
				c := command{open: func(ctx context.Context, dir string) (changeProvider, error) {
					p, err := signing.OpenFileProviderContext(ctx, dir)
					if err != nil {
						return nil, err
					}
					path := filepath.Join(dir, "lock")
					switch kind {
					case "group":
						err = os.Chmod(path, 0640)
					case "symlink":
						if err = os.Remove(path); err == nil {
							err = os.Symlink("state.json", path)
						}
					case "acceptable replacement":
						if err = os.Remove(path); err == nil {
							err = os.WriteFile(path, nil, 0600)
						}
					}
					if err != nil {
						t.Fatal(err)
					}
					before = manifest(t, dir)
					if kind == "typed replacement rejection" {
						return &faultProvider{changeProvider: p, mutationError: &signing.LockValidationError{}}, nil
					}
					return p, nil
				}}
				if kind == "acceptable replacement" {
					outcome := "applied"
					if op == "revoke" {
						outcome = "revoked"
					}
					invoke(t, c, f.args(op), exitOK, outcome)
				} else {
					invoke(t, c, f.args(op), exitUnavailable, "")
					unchanged(t, f.dir, before)
				}
			})
		}
	}
}
