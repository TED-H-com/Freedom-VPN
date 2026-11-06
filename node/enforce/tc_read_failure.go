package enforce

import (
	"context"
	"errors"
	"os/exec"
	"regexp"
)

// TCReadFailure contains local, bounded command diagnostics. Classification
// describes the returned error chain. ContextState is a separate observation
// immediately after Output fails; it does not establish why the command failed.
// This is outside TCReadEvidence and its D-106 serialization and digest.
type TCReadFailure struct {
	Classification string
	ContextState   string
	Interface      string
	Attachment     string
	ExitCode       *int
}

// ReadTCFailure returns detached diagnostics without exposing the original
// command, output or error. It does not change CheckError's error chain.
func ReadTCFailure(err error) (TCReadFailure, bool) {
	var check *CheckError
	if !errors.As(err, &check) || check == nil || check.tcFailure == nil {
		return TCReadFailure{}, false
	}
	d := *check.tcFailure
	if d.ExitCode != nil {
		code := *d.ExitCode
		d.ExitCode = &code
	}
	return d, true
}

var diagnosticInterface = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,15}$`)
var diagnosticAttachment = regexp.MustCompile(`^[0-9a-fA-F]{1,4}:[0-9a-fA-F]{0,4}$`)

func tcReadError(contextErr error, r TCReadEvidence, err error) error {
	d := TCReadFailure{Classification: "read_error", ContextState: "none", Interface: cleanTCToken(r.Interface), Attachment: cleanTCToken(r.Attachment)}
	if !diagnosticInterface.MatchString(d.Interface) {
		d.Interface = "redacted"
	}
	switch d.Attachment {
	case "qdisc", "class", "root", "ingress", "egress":
	default:
		if !diagnosticAttachment.MatchString(d.Attachment) {
			d.Attachment = "redacted"
		}
	}
	switch contextErr {
	case context.DeadlineExceeded:
		d.ContextState = "deadline"
	case context.Canceled:
		d.ContextState = "canceled"
	}
	var exit *exec.ExitError
	if errors.As(err, &exit) && exit != nil {
		d.Classification = "command_exit"
		if code := exit.ExitCode(); code >= 0 {
			d.ExitCode = &code
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		d.Classification = "deadline"
	} else if errors.Is(err, context.Canceled) {
		d.Classification = "canceled"
	}
	check := tcCheck(r, nil).(*CheckError)
	check.tcFailure = &d
	return check
}
