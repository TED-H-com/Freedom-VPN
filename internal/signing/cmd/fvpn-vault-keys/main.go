// fvpn-vault-keys conditionally rotates/revokes central FileProvider keys.
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
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/TED-H-com/Freedom-VPN/internal/signing"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
	"github.com/google/uuid"
	"golang.org/x/sys/unix"
)

const defaultLockTimeout = 5 * time.Second
const (
	exitOK             = 0
	exitAlreadyRevoked = 0 // Idempotent success; outcome distinguishes it.
	// Exit 1 is deliberately unused. Keep all failure codes stable.
	exitUsage       = 2
	exitUnavailable = 3
	exitTimeout     = 4
	exitRefused     = 5
	exitNotApplied  = 6
	exitDurability  = 7
	exitUnknown     = 8
	exitOutput      = 9
)
const usage = "usage: fvpn-vault-keys rotate -vault EXISTING_VAULT -purpose PURPOSE -expect-revision N -expect-active-key KEY_ID|none -validity DURATION -overlap DURATION [-lock-timeout DURATION] | fvpn-vault-keys revoke -vault EXISTING_VAULT -key-id KEY_ID -expect-purpose PURPOSE -expect-revision N [-revoke-active] [-lock-timeout DURATION]"

type options struct {
	subcommand, vault, keyID, active string
	purpose                          signing.Purpose
	revision                         decstr.U64
	validity, overlap, timeout       time.Duration
	revokeActive                     bool
}
type onceFlag struct {
	seen, boolean bool
	set           func(string) error
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
func central(p signing.Purpose) bool {
	return p == signing.DesiredState || p == signing.Lease || p == signing.Recovery || p == signing.LedgerEpoch
}
func canonicalID(s string) bool {
	id, err := uuid.Parse(s)
	return err == nil && id.String() == s
}
func parse(args []string) (o options, ok bool) {
	if len(args) == 0 || (args[0] != "rotate" && args[0] != "revoke") {
		return o, false
	}
	o.subcommand, o.timeout = args[0], defaultLockTimeout
	fs := flag.NewFlagSet("fvpn-vault-keys", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var required []*onceFlag
	add := func(name string, boolean, mandatory bool, set func(string) error) {
		f := &onceFlag{boolean: boolean, set: set}
		fs.Var(f, name, name)
		if mandatory {
			required = append(required, f)
		}
	}
	add("vault", false, true, func(s string) error { o.vault = s; return nil })
	add("expect-revision", false, true, func(s string) (err error) { o.revision, err = decstr.Parse(s); return })
	add("lock-timeout", false, false, func(s string) (err error) { o.timeout, err = time.ParseDuration(s); return })
	purpose := func(s string) error { o.purpose = signing.Purpose(s); return nil }
	if o.subcommand == "rotate" {
		add("purpose", false, true, purpose)
		add("expect-active-key", false, true, func(s string) error { o.active = s; return nil })
		add("validity", false, true, func(s string) (err error) { o.validity, err = time.ParseDuration(s); return })
		add("overlap", false, true, func(s string) (err error) { o.overlap, err = time.ParseDuration(s); return })
	} else {
		add("key-id", false, true, func(s string) error { o.keyID = s; return nil })
		add("expect-purpose", false, true, purpose)
		add("revoke-active", true, false, func(s string) (err error) { o.revokeActive, err = strconv.ParseBool(s); return })
	}
	if fs.Parse(args[1:]) != nil || fs.NArg() != 0 {
		return o, false
	}
	for _, f := range required {
		if !f.seen {
			return o, false
		}
	}
	if o.vault == "" || hasParentElement(o.vault) || !central(o.purpose) || o.timeout <= 0 {
		return o, false
	}
	if o.subcommand == "rotate" {
		if o.validity <= 0 || o.overlap < 0 || (o.active != "none" && !canonicalID(o.active)) {
			return o, false
		}
		if o.active == "none" {
			o.active = ""
		}
	} else if !canonicalID(o.keyID) {
		return o, false
	}
	o.vault = filepath.Clean(o.vault)
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

// Pin the entry without following a symlink or opening contents. In particular,
// a FIFO cannot block this check. The provider validates the lock it actually
// holds during mutation; do not recheck this reference after a change (§16.2).
func holdLock(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, err
	}
	f := os.NewFile(uintptr(fd), path)
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		_ = f.Close()
		return nil, signing.ErrUnavailable
	}
	return f, nil
}

type changeProvider interface {
	RotateConditional(context.Context, signing.Purpose, decstr.U64, string, time.Duration, time.Duration) (signing.ChangeResult, error)
	RevokeConditional(context.Context, string, signing.Purpose, decstr.U64, bool) (signing.ChangeResult, error)
	Snapshot(context.Context) (signing.FileSnapshot, error)
}

// Invocation-local dependencies let tests exercise failures without any runtime
// environment switch or exported provider test hook.
type command struct {
	open func(context.Context, string) (changeProvider, error)
}

func main() {
	signal.Ignore(syscall.SIGPIPE)
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
func run(args []string, stdout, stderr io.Writer) int { return (command{}).run(args, stdout, stderr) }
func fail(stderr io.Writer, code int, message string) int {
	fmt.Fprintln(stderr, "fvpn-vault-keys: "+message)
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
		return fail(stderr, exitUsage, usage)
	}
	lock, err := holdLock(filepath.Join(o.vault, "lock"))
	if err != nil {
		return fail(stderr, exitUnavailable, "vault unavailable")
	}
	defer lock.Close()
	ctx, cancel := context.WithTimeout(context.Background(), o.timeout)
	defer cancel()
	open := c.open
	if open == nil {
		open = func(ctx context.Context, dir string) (changeProvider, error) {
			return signing.OpenFileProviderContext(ctx, dir)
		}
	}
	p, err := open(ctx, o.vault)
	if err != nil {
		return providerFailure(stderr, err)
	}
	var result signing.ChangeResult
	if o.subcommand == "rotate" {
		result, err = p.RotateConditional(ctx, o.purpose, o.revision, o.active, o.validity, o.overlap)
	} else {
		result, err = p.RevokeConditional(ctx, o.keyID, o.purpose, o.revision, o.revokeActive)
	}
	if err != nil {
		return providerFailure(stderr, err)
	}
	// No mutation retry, probe, cleanup or context check after the change. The
	// sole follow-up to an uncertain replacement is a fresh locked snapshot.
	report, code := classify(o, result)
	if result.Outcome == signing.ChangeUncertain {
		readCtx, readCancel := context.WithTimeout(context.Background(), o.timeout)
		snapshot, readErr := p.Snapshot(readCtx)
		readCancel()
		if readErr != nil {
			report.ReadBack = "failed"
		} else if confirms(snapshot, result) {
			report, code = confirmed(o, result, true)
		} else {
			report.ReadBack = "different"
		}
	}
	return writeResult(stdout, stderr, report, code)
}
