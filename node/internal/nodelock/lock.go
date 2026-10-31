// Package nodelock is a compatibility alias for the importable lock in
// github.com/TED-H-com/Freedom-VPN/node/nodelock. Callers under node/ keep
// compiling. packaging/installer can import node/nodelock and drop its copy.
package nodelock

import (
	"context"
	"time"

	real "github.com/TED-H-com/Freedom-VPN/node/nodelock"
)

// DefaultPath is the default lock file (§4 node paths).
const DefaultPath = real.DefaultPath

// ErrBusy means another process holds the lock (UPDATE_BUSY, §13.11).
var ErrBusy = real.ErrBusy

// Lock is a held exclusive flock.
type Lock = real.Lock

// TryAcquire takes LOCK_EX|LOCK_NB before it writes holder metadata.
func TryAcquire(path string) (*Lock, error) { return real.TryAcquire(path) }

// Acquire retries TryAcquire until attempts run out or ctx ends.
func Acquire(ctx context.Context, path string, attempts int, interval time.Duration) (*Lock, error) {
	return real.Acquire(ctx, path, attempts, interval)
}
