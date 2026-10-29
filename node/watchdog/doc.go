// Package watchdog is the separate liveness process of §16.3.
//
// It reads the agent heartbeat and, on a stale heartbeat, a missing baseline
// or an expired policy, installs the fail-closed baseline through node/enforce.
// It never contacts the control plane and it never rewrites a serving lease.
// A restart of this process does not restart kernel deadlines: a healthy
// reading leaves the ruleset untouched. If the agent still holds the node
// lock, the watchdog applies the baseline anyway. Fail-closed outranks the lock.
package watchdog
