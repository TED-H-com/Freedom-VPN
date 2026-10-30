package agent

import (
	"context"
	"errors"

	"github.com/TED-H-com/Freedom-VPN/node/internal/usagestore"
)

// Snapshot exit codes. ExitUsage (64) is still the host_reserve_bps failure.
const (
	// ExitSnapshotArgs is a bad invocation or an unusable config file.
	ExitSnapshotArgs = 2
	// ExitSnapshotAbsent means the usage database file is not there.
	ExitSnapshotAbsent = 3
	// ExitSnapshotLocked means SQLite stayed busy for the whole bounded wait.
	ExitSnapshotLocked = 4
	// ExitSnapshotCorrupt means the file is not a readable usage ledger.
	ExitSnapshotCorrupt = 5
)

const (
	msgSnapshotAbsent  = "fvpn-agent: usage store is absent"
	msgSnapshotLocked  = "fvpn-agent: usage store is locked"
	msgSnapshotCorrupt = "fvpn-agent: usage store is corrupt"
)

// UsageSnapshot reads the ledger at cfg.UsageDBPath and returns the process
// status, the stdout bytes and a stderr line. Stdout is empty on failure.
// The stderr line is a fixed sentence: it has no path, SQL text or secret.
//
// The read does not create the database, does not migrate it, does not take
// the node update lock, and does not call the upload or ACK paths (§10.6).
func UsageSnapshot(ctx context.Context, cfg Config, jsonOut bool) (code int, stdout []byte, stderr string) {
	snap, err := usagestore.ReadSnapshot(ctx, cfg.UsageDBPath())
	if err != nil {
		switch {
		case errors.Is(err, usagestore.ErrAbsent):
			return ExitSnapshotAbsent, nil, msgSnapshotAbsent
		case errors.Is(err, usagestore.ErrLocked):
			return ExitSnapshotLocked, nil, msgSnapshotLocked
		default:
			return ExitSnapshotCorrupt, nil, msgSnapshotCorrupt
		}
	}
	var body []byte
	if jsonOut {
		body, err = snap.JSON()
	} else {
		body = []byte(snap.Text())
	}
	if err != nil || len(body) == 0 || body[len(body)-1] != '\n' {
		return ExitSnapshotCorrupt, nil, msgSnapshotCorrupt
	}
	return 0, body, ""
}
