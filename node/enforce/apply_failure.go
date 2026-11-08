package enforce

import (
	"context"
	"errors"
	"os/exec"
)

// ApplyFailure is bounded local diagnosis, outside serving evidence and its
// serialized digest. ContextState observes the context at the failing boundary;
// it does not assert that cancellation caused the returned error.
type ApplyFailure struct {
	Stage          string
	Classification string
	ContextState   string
	ExitCode       *int
}

type applyFailureError struct {
	diagnostic ApplyFailure
	cause      error
}

func (e *applyFailureError) Error() string { return e.cause.Error() }
func (e *applyFailureError) Unwrap() error { return e.cause }

// ReadApplyFailure returns a detached value; the original error remains
// available to errors.Is and errors.As, never to diagnostic serialization.
func ReadApplyFailure(err error) (ApplyFailure, bool) {
	var failure *applyFailureError
	if !errors.As(err, &failure) || failure == nil {
		return ApplyFailure{}, false
	}
	d := failure.diagnostic
	if d.ExitCode != nil {
		code := *d.ExitCode
		d.ExitCode = &code
	}
	return d, true
}

func applyFailure(stage string, contextErr, err error) error {
	if err == nil {
		return nil
	}
	d := ApplyFailure{Stage: stage, Classification: "operation_error", ContextState: "none"}
	switch contextErr {
	case context.DeadlineExceeded:
		d.ContextState = "deadline"
	case context.Canceled:
		d.ContextState = "canceled"
	}
	var exit *exec.ExitError
	var check *CheckError
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		d.Classification = "deadline"
	case errors.Is(err, context.Canceled):
		d.Classification = "canceled"
	case errors.As(err, &exit) && exit != nil:
		d.Classification = "command_exit"
		if code := exit.ExitCode(); code >= 0 && code <= 255 {
			d.ExitCode = &code
		}
	case errors.As(err, &check) && check != nil:
		d.Classification = "check"
	}
	return &applyFailureError{diagnostic: d, cause: err}
}
