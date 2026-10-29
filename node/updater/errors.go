package updater

import (
	"errors"
	"fmt"

	"github.com/TED-H-com/Freedom-VPN/pkg/apierr"
)

// OpError is a node-local update failure (§13.11). Detail is safe to log:
// no tokens, keys or member data. Retry means a later attempt may repeat
// the same step; permanent errors are not retried automatically (§13.4).
type OpError struct {
	Code   apierr.Code
	Detail string
	Retry  bool
}

func (e *OpError) Error() string {
	if e.Detail == "" {
		return "updater: " + string(e.Code)
	}
	return "updater: " + string(e.Code) + ": " + e.Detail
}

func opErr(code apierr.Code, detail string, retry bool) *OpError {
	return &OpError{Code: code, Detail: detail, Retry: retry}
}

func asOp(err error) *OpError {
	var op *OpError
	if errors.As(err, &op) {
		return op
	}
	return nil
}

// ErrNoWork means the verified desired state carries no update job.
var ErrNoWork = errors.New("updater: no authorized update")

// ErrCrashed is returned when a crash hook stops the process after a stage
// has been made durable. Callers must not clean up as if the step failed.
var ErrCrashed = errors.New("updater: injected crash")

// ErrSuperseded stops a queued or downloading job so a newer generation
// can run (§13.8).
var ErrSuperseded = errors.New("updater: superseded")

// ErrStateUnsafe means a migration cannot promise the previous release can
// read the state (§13.7). The outcome is recovery_required.
var ErrStateUnsafe = errors.New("updater: state is not readable by the previous release")

// ErrMembersBlocked means boot recovery left member forwarding off (§13.5,
// §13.9). The process exit code is 2. Local success is not implied.
var ErrMembersBlocked = errors.New("updater: member forwarding must stay blocked")

func permanent(code apierr.Code, detail string) *OpError {
	return opErr(code, detail, false)
}

func transient(code apierr.Code, detail string) *OpError {
	return opErr(code, detail, true)
}

func wrapCode(code apierr.Code, err error, retry bool) *OpError {
	if err == nil {
		return opErr(code, "", retry)
	}
	return opErr(code, fmt.Sprintf("%v", err), retry)
}
