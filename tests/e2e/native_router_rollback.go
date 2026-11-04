//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type nativeRouterRunner func(context.Context, ...string) ([]byte, error)

// NativeRouterRollback records every completed setup step in reverse order.
// A router sysctl is undone by removing its newly created container, not by
// guessing an earlier sysctl value. Its receipt shares that removal's outcome.
type NativeRouterRollback struct {
	Step       int       `json:"setup_step"`
	Command    []string  `json:"command,omitempty"`
	Outcome    string    `json:"outcome"`
	Error      string    `json:"error,omitempty"`
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at"`
}

// NativeRouterSetupError unwraps to the original setup error. Rollback receipts
// are both accessible with errors.As and included in Error for existing callers.
type NativeRouterSetupError struct {
	Step     int
	Cause    error
	Rollback []NativeRouterRollback
}

func (e *NativeRouterSetupError) Error() string {
	raw, _ := json.Marshal(e.Rollback)
	return fmt.Sprintf("router step %d: %v; rollback=%s", e.Step, e.Cause, raw)
}
func (e *NativeRouterSetupError) Unwrap() error { return e.Cause }

func rollbackNativeRouter(ctx context.Context, run nativeRouterRunner, failed int, cause error, undo [][]string) error {
	out := &NativeRouterSetupError{Step: failed, Cause: cause, Rollback: []NativeRouterRollback{}}
	for i := failed - 1; i >= 0; i-- {
		row := NativeRouterRollback{Step: i, Command: undo[i], StartedAt: time.Now().UTC()}
		if undo[i] == nil {
			row.Outcome = "pending_container_removal"
		} else {
			// Each inverse gets its own bound; cancellation or a failed inverse
			// cannot prevent the remaining cleanup attempts.
			clean, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			_, err := run(clean, undo[i]...)
			cancel()
			row.Outcome = "succeeded"
			if err != nil {
				row.Outcome = "failed"
				row.Error = err.Error()
			}
		}
		row.FinishedAt = time.Now().UTC()
		out.Rollback = append(out.Rollback, row)
		if i == 1 {
			for j := range out.Rollback {
				if out.Rollback[j].Step == 5 {
					out.Rollback[j].Outcome = "container_removal_" + row.Outcome
					out.Rollback[j].FinishedAt = row.FinishedAt
					out.Rollback[j].Error = row.Error
				}
			}
		}
	}
	return out
}
