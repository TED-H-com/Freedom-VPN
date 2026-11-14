package agent

import "testing"

// closeAdoptionControl ends network I/O before a bubble's local restart work.
// An idle Accept or HTTP connection is not durably blocked in synctest and
// would prevent timers (including a watchdog lock release) from advancing.
// Build the entire rig inside the bubble so certificates, trust windows and
// signed documents all use its epoch. Close engines with that bubble's T too,
// so their usage loops and client connections end before the bubble returns.
// Call this only after the last control request needed by the test.
func closeAdoptionControl(t *testing.T, r *rig) {
	t.Helper()
	if err := r.srv.Close(); err != nil {
		t.Fatal(err)
	}
}
