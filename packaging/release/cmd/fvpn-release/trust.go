package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/TED-H-com/Freedom-VPN/packaging/release"
	"github.com/TED-H-com/Freedom-VPN/pkg/decstr"
)

type trustUsageError struct{ message string }

func (e *trustUsageError) Error() string { return e.message }

func trustUsage(message string) error { return &trustUsageError{message: message} }

// singleTrustFlag retains flag's normal syntax, including -name=value and
// boolean flags, but rejects every repeated flag, even repeated equal values.
type singleTrustFlag struct {
	flag.Value
	seen bool
}

func (v *singleTrustFlag) Set(s string) error {
	if v.seen {
		return errors.New("flag may be given at most once")
	}
	v.seen = true
	return v.Value.Set(s)
}

func (v *singleTrustFlag) IsBoolFlag() bool {
	b, ok := v.Value.(interface{ IsBoolFlag() bool })
	return ok && b.IsBoolFlag()
}

func parseTrustFlags(fs *flag.FlagSet, args []string) error {
	fs.VisitAll(func(f *flag.Flag) { f.Value = &singleTrustFlag{Value: f.Value} })
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return trustUsage(err.Error())
	}
	if fs.NArg() != 0 {
		return trustUsage("unexpected positional arguments")
	}
	return nil
}

func trustTime(value, name string) (time.Time, error) {
	at, err := time.Parse(time.RFC3339, value)
	if err != nil || at.IsZero() {
		return time.Time{}, trustUsage(name + " requires a nonzero RFC3339 time")
	}
	return at.UTC(), nil
}

func trustRevision(value, name string, positive bool) (decstr.U64, error) {
	n, err := decstr.Parse(value)
	if err != nil || (positive && n == 0) {
		return 0, trustUsage(name + " requires a canonical decimal uint64" + map[bool]string{true: " greater than zero"}[positive])
	}
	return n, nil
}

// New commands have strict usage errors without changing the legacy commands'
// flag handling or exit codes.
func runTrust(command string, args []string, stdout, stderr io.Writer) int {
	var err error
	if command == "trust-update" {
		err = cmdTrustUpdate(args, stdout, stderr)
	} else {
		err = cmdTrustVerify(args, stdout, stderr)
	}
	if errors.Is(err, flag.ErrHelp) {
		return 0
	}
	if err != nil {
		fmt.Fprintf(stderr, "fvpn-release: %v\n", err)
		var usage *trustUsageError
		if errors.As(err, &usage) {
			return 2
		}
		return 1
	}
	return 0
}

func trustFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: fvpn-release %s [flags]\n", name)
		fs.PrintDefaults()
		fmt.Fprintln(stderr, "Offline check only: the centre decides against its own ledger at apply time.")
		fmt.Fprintln(stderr, "The public trust store omits the stored release revision, retirement states and revocation times.")
		if name == "trust-update" {
			fmt.Fprintln(stderr, "Exactly one operation: -add FILE | -retire KEY_ID -not-after TIME | -revoke KEY_ID.")
			fmt.Fprintln(stderr, "-allow-no-usable-root permits deliberate emergency removal; recovery from loss of every root is not provided.")
		}
	}
	return fs
}

func cmdTrustUpdate(args []string, stdout, stderr io.Writer) error {
	fs := trustFlags("trust-update", stderr)
	keyPath := fs.String("key", "", "release signing key (build machine only, required)")
	trustPath := fs.String("trust", "", "centre's current exported node trust store (required)")
	revision := fs.String("revision", "", "explicit next release revision, canonical decimal uint64 >= 1 (required)")
	out := fs.String("out", "", "NEW public update file; parent must exist (required)")
	add := fs.String("add", "", "one-key node trust store to authorize")
	retire := fs.String("retire", "", "current release key ID to retire")
	revoke := fs.String("revoke", "", "current release key ID to revoke")
	end := fs.String("not-after", "", "retirement end, RFC3339 (required with -retire only)")
	atText := fs.String("at", "", "one decision instant, RFC3339 (default: current UTC whole second)")
	allow := fs.Bool("allow-no-usable-root", false, "permit deliberate emergency removal of the last usable root")
	if err := parseTrustFlags(fs, args); err != nil {
		return err
	}
	if *keyPath == "" || *trustPath == "" || *revision == "" || *out == "" {
		return trustUsage("trust-update requires -key, -trust, -revision and -out")
	}
	opt := release.TrustUpdateOptions{At: time.Now().UTC().Truncate(time.Second), AllowNoUsableRoot: *allow}
	var err error
	if opt.Revision, err = trustRevision(*revision, "-revision", true); err != nil {
		return err
	}
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	operations := 0
	for _, op := range []string{"add", "retire", "revoke"} {
		if seen[op] {
			operations++
			opt.Operation = op
		}
	}
	if operations != 1 || (seen["add"] && *add == "") || (seen["retire"] && *retire == "") || (seen["revoke"] && *revoke == "") {
		return trustUsage("trust-update requires exactly one of -add, -retire or -revoke")
	}
	if seen["not-after"] != seen["retire"] {
		return trustUsage("-not-after is required with -retire and allowed only with -retire")
	}
	if seen["at"] {
		if opt.At, err = trustTime(*atText, "-at"); err != nil {
			return err
		}
	}
	if seen["not-after"] {
		if opt.NotAfter, err = trustTime(*end, "-not-after"); err != nil {
			return err
		}
	}
	opt.TargetKeyID = *retire
	if seen["revoke"] {
		opt.TargetKeyID = *revoke
	}
	if seen["add"] {
		if opt.AddTrust, err = os.ReadFile(*add); err != nil {
			return err
		}
	}
	trust, err := readReleaseTrust(*trustPath)
	if err != nil {
		return err
	}
	key, err := release.LoadKey(*keyPath)
	if err != nil {
		// The existing loader may mention malformed field names or values.
		// Never reflect any part of a private key file into CLI diagnostics.
		return errors.New("trust-update: cannot load release signing key")
	}
	raw, result, err := release.BuildTrustUpdate(key, trust, opt)
	if err != nil {
		return err
	}
	if err := release.WriteNewTrustUpdate(*out, raw); err != nil {
		return err
	}
	target, _ := json.Marshal(result.Update.Keys[0].ID)
	fmt.Fprintf(stdout, "kind=%s\nrevision=%s\nsigner_key_id=%s\noperation=%s\ntarget_key_id=%s\nupdate_sha256=%s\n",
		result.Update.Kind, result.Update.Revision, key.KeyID, opt.Operation, target, result.UpdateSHA256)
	return nil
}

func readReleaseTrust(path string) (*release.ReleaseTrust, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return release.ParseReleaseTrust(raw)
}

func cmdTrustVerify(args []string, stdout, stderr io.Writer) error {
	fs := trustFlags("trust-verify", stderr)
	update := fs.String("update", "", "signed trust update to check offline (required)")
	trustPath := fs.String("trust", "", "centre's current exported node trust store (required)")
	atText := fs.String("at", "", "one decision instant, RFC3339 (default: current UTC whole second)")
	revision := fs.String("current-revision", "", "known centre release revision, canonical decimal uint64 (optional)")
	if err := parseTrustFlags(fs, args); err != nil {
		return err
	}
	if *update == "" || *trustPath == "" {
		return trustUsage("trust-verify requires -update and -trust")
	}
	at := time.Now().UTC().Truncate(time.Second)
	var current *decstr.U64
	var err error
	seen := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	if seen["at"] {
		if at, err = trustTime(*atText, "-at"); err != nil {
			return err
		}
	}
	if seen["current-revision"] {
		n, err := trustRevision(*revision, "-current-revision", false)
		if err != nil {
			return err
		}
		current = &n
	}
	raw, err := os.ReadFile(*update)
	if err != nil {
		return err
	}
	trust, err := readReleaseTrust(*trustPath)
	if err != nil {
		return err
	}
	result, err := release.VerifyTrustUpdate(raw, trust, at, current)
	if err != nil {
		return err
	}
	// Prepare every line before printing any result.
	var output strings.Builder
	fmt.Fprintf(&output, "kind=%s\nrevision=%s\nsigner_key_ids=%s\nupdate_sha256=%s\n", result.Update.Kind,
		result.Update.Revision, strings.Join(result.SignerKeyIDs, ","), result.UpdateSHA256)
	for _, record := range result.Records {
		line, err := release.TrustRecordJSON(record)
		if err != nil {
			return err
		}
		fmt.Fprintf(&output, "key_record=%s\n", line)
	}
	fmt.Fprintf(&output, "usable_root_after=%t\n", result.UsableRootAfter)
	_, err = io.WriteString(stdout, output.String())
	return err
}
