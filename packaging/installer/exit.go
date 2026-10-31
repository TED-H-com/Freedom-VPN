package installer

import "fmt"

// Process statuses. check uses ExitDrifted. A busy lock is ExitBusy and
// does not wait (§13.5). ExitUsage matches the updater's flag errors.
const (
	ExitOK         = 0
	ExitFailed     = 1
	ExitDrifted    = 2
	ExitBusy       = 3
	ExitPlatform   = 4
	ExitDependency = 5
	ExitUsage      = 64
)

// Error is a single-line failure. Error() is safe to print: it has no
// stack and no secret.
type Error struct {
	Exit int
	Code string
	Msg  string
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Msg == "" {
		return e.Code
	}
	return e.Code + ": " + e.Msg
}

func errf(exit int, code, format string, args ...any) *Error {
	return &Error{Exit: exit, Code: code, Msg: fmt.Sprintf(format, args...)}
}

// AsError unwraps an installer error. Other errors become ExitFailed.
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	if ee, ok := err.(*Error); ok {
		return ee
	}
	return &Error{Exit: ExitFailed, Code: "failed", Msg: err.Error()}
}
