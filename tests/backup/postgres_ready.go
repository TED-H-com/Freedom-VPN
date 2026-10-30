package backup

import (
	"strings"
	"time"
)

const (
	// postgresReadyTimeout is the bound on waiting for the final postmaster.
	// The official image logs a ready line for its temporary init server and
	// again for the real server; callers must not treat the first as success.
	postgresReadyTimeout = 120 * time.Second
	// postgresReadyConfirmGap is the pause between two successful SELECT 1 polls.
	postgresReadyConfirmGap = time.Second

	postgresReadyToAccept = "database system is ready to accept connections"
	postgresInitComplete  = "PostgreSQL init process complete; ready for start up."
	postgresShutDown      = "database system is shut down"
	postgresShuttingDown  = "database system is shutting down"
)

// postgresFinalServerReady reports whether logs show the real postmaster, not
// the temporary server started by docker-entrypoint to run init scripts.
//
// On a fresh data directory the image logs postgresReadyToAccept once for that
// temporary server, shuts it down, logs postgresInitComplete, then logs
// postgresReadyToAccept again for the server that stays up. A restored data
// directory skips init and logs the ready line once (after promote, not the
// earlier "read-only connections" line).
func postgresFinalServerReady(logs string) bool {
	if initAt := strings.LastIndex(logs, postgresInitComplete); initAt >= 0 {
		return readyLineStands(logs[initAt+len(postgresInitComplete):])
	}
	if initPhaseStarted(logs) {
		return false
	}
	return readyLineStands(logs)
}

// initPhaseStarted is true once initdb has begun and the entrypoint has not yet
// logged postgresInitComplete. Those lines are written before the temporary
// server's ready line, so a single ready line during init is not the final server.
func initPhaseStarted(logs string) bool {
	return strings.Contains(logs, "The files belonging to this database system will be owned by user") ||
		strings.Contains(logs, "/docker-entrypoint-initdb.d/") ||
		strings.Contains(logs, "initdb:")
}

// readyLineStands is true when section contains postgresReadyToAccept and the
// postmaster has not begun shutting down after that last ready line.
func readyLineStands(section string) bool {
	readyAt := strings.LastIndex(section, postgresReadyToAccept)
	if readyAt < 0 {
		return false
	}
	tail := section[readyAt+len(postgresReadyToAccept):]
	return !strings.Contains(tail, postgresShutDown) && !strings.Contains(tail, postgresShuttingDown)
}
