// fvpn-vault reads an existing development FileProvider without changing it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
)

const defaultLockTimeout = 5 * time.Second
const (
	exitOK = iota
	exitResultFail
	exitUsage
	exitUnavailable
	exitTimeout
	exitChanged
	exitExport
	exitOutput
)

// Dependencies are per invocation; tests can observe probes, change the ledger
// between snapshots, and inject publication faults without provider test hooks.
type command struct {
	afterProbes   func()
	beforeResult  func()
	probeProvider func(signing.Provider) signing.Provider
	publication   publicationOps
}

type options struct {
	subcommand, vault, out string
	checkKeys              bool
	timeout                time.Duration
}

type onceFlag struct {
	seen    bool
	boolean bool
	set     func(string) error
}

func (v *onceFlag) String() string   { return "" }
func (v *onceFlag) IsBoolFlag() bool { return v.boolean }
func (v *onceFlag) Set(s string) error {
	if v.seen {
		return errors.New("repeated flag")
	}
	v.seen = true
	return v.set(s)
}
func parse(args []string) (o options, ok bool) {
	if len(args) == 0 || (args[0] != "inspect" && args[0] != "export-trust") {
		return o, false
	}
	o.subcommand, o.timeout = args[0], defaultLockTimeout
	fs := flag.NewFlagSet("fvpn-vault", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Var(&onceFlag{set: func(s string) error { o.vault = s; return nil }}, "vault", "existing vault")
	fs.Var(&onceFlag{set: func(s string) (err error) { o.timeout, err = time.ParseDuration(s); return }}, "lock-timeout", "positive provider deadline")
	if o.subcommand == "inspect" {
		fs.Var(&onceFlag{boolean: true, set: func(s string) (err error) { o.checkKeys, err = strconv.ParseBool(s); return }}, "check-keys", "exercise current central keys")
	} else {
		fs.Var(&onceFlag{set: func(s string) error { o.out = s; return nil }}, "out", "new public trust file")
	}
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 || o.vault == "" || o.timeout <= 0 || (o.subcommand == "export-trust" && o.out == "") {
		return o, false
	}
	// Reject parent traversal on raw values: cleaning symlink/.. lexically can
	// select a different vault than the operating system would resolve.
	if hasParentElement(o.vault) || hasParentElement(o.out) {
		return o, false
	}
	o.vault = filepath.Clean(o.vault)
	if o.out != "" {
		o.out = filepath.Clean(o.out)
	}
	return o, true
}

func hasParentElement(path string) bool {
	for _, element := range strings.Split(path, string(filepath.Separator)) {
		if element == ".." {
			return true
		}
	}
	return false
}

func validLock(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
}

// This observes the pathname at a point in time, without binding the provider's
// separate opens to this inode or defending against a writable vault owner.
func sameLock(path string, original os.FileInfo) bool {
	current, err := os.Lstat(path)
	return err == nil && validLock(current) && os.SameFile(original, current) && original.Mode().Perm() == current.Mode().Perm()
}

func main() {
	// Report broken stdout pipes through the same controlled output failure path.
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
func run(args []string, stdout, stderr io.Writer) int {
	return (command{}).run(args, stdout, stderr)
}
func fail(stderr io.Writer, code int, message string) int {
	fmt.Fprintln(stderr, "fvpn-vault: "+message)
	return code
}
func providerFailure(stderr io.Writer, err error) int {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fail(stderr, exitTimeout, "vault lock deadline exceeded")
	}
	return fail(stderr, exitUnavailable, "vault unavailable")
}
func (c command) run(args []string, stdout, stderr io.Writer) int {
	o, ok := parse(args)
	if !ok {
		return fail(stderr, exitUsage, "usage: fvpn-vault inspect -vault EXISTING_VAULT [-check-keys] [-lock-timeout DURATION] | fvpn-vault export-trust -vault EXISTING_VAULT -out NEW_FILE [-lock-timeout DURATION]")
	}
	// One deadline bounds all provider work, including the initial open. No
	// filesystem operation here creates anything in the vault (§17.2, §17.4).
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()
	// Point-in-time command check: the provider itself opens the lock without
	// validating its type or permissions. Never open a symlink or special file
	// here, and leave provider locking/validation behavior unchanged (§16.2).
	lockPath := filepath.Join(o.vault, "lock")
	lock, err := os.Lstat(lockPath)
	if err != nil || !validLock(lock) {
		return fail(stderr, exitUnavailable, "vault unavailable")
	}
	p, err := signing.OpenFileProviderContext(ctx, o.vault)
	if err != nil {
		if !sameLock(lockPath, lock) {
			return fail(stderr, exitChanged, "changed during inspection")
		}
		return providerFailure(stderr, err)
	}
	before, err := p.Snapshot(ctx)
	if !sameLock(lockPath, lock) {
		return fail(stderr, exitChanged, "changed during inspection")
	}
	if err != nil {
		return providerFailure(stderr, err)
	}
	instant := time.Now().UTC()
	var raw, bundle []byte
	result := exitOK
	if o.subcommand == "inspect" {
		checker := signing.Provider(p)
		if c.probeProvider != nil {
			checker = c.probeProvider(checker)
		}
		report := inspect(ctx, checker, before, instant, o.checkKeys)
		if o.checkKeys {
			if c.afterProbes != nil {
				c.afterProbes()
			}
			after, err := p.Snapshot(ctx)
			if err != nil {
				if !sameLock(lockPath, lock) {
					return fail(stderr, exitChanged, "changed during inspection")
				}
				return providerFailure(stderr, err)
			}
			if before.Revision != after.Revision || before.ReleaseRevision != after.ReleaseRevision || before.DeniedIdentityCount != after.DeniedIdentityCount || !reflect.DeepEqual(before.Keys, after.Keys) {
				return fail(stderr, exitChanged, "changed during inspection")
			}
		}
		if ctx.Err() != nil {
			return providerFailure(stderr, ctx.Err())
		}
		raw, err = document(report)
		if err != nil {
			return fail(stderr, exitUnavailable, "cannot encode inspection")
		}
		if report.Result == "fail" {
			result = exitResultFail
		}
	} else {
		var receipt exportReceipt
		bundle, receipt, err = prepareExport(before, instant, filepath.Base(o.out))
		if err != nil {
			return fail(stderr, exitExport, "cannot prepare public trust export")
		}
		raw, err = document(receipt)
		if err != nil {
			return fail(stderr, exitExport, "cannot encode export receipt")
		}
	}
	if c.beforeResult != nil {
		c.beforeResult()
	}
	if !sameLock(lockPath, lock) {
		return fail(stderr, exitChanged, "changed during inspection")
	}
	if ctx.Err() != nil {
		return providerFailure(stderr, ctx.Err())
	}
	if o.subcommand == "export-trust" {
		if err = publish(o.vault, o.out, bundle, c.publication); err != nil {
			return fail(stderr, exitExport, "public trust publication failed")
		}
	}
	if n, err := stdout.Write(raw); err != nil || n != len(raw) {
		if o.subcommand == "export-trust" {
			return fail(stderr, exitOutput, "output failed; published file remains")
		}
		return fail(stderr, exitOutput, "output failed")
	}
	return result
}
