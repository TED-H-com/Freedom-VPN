// Package agent is the exit-node agent (§5, §6, §14.7, §16.3).
//
// Boot order is fail-closed: the baseline is installed before identity,
// a lease or a policy can enable member forwarding. Desired state is verified
// before it is stored or written for the updater. A 304 does not move a lease.
// Wall-clock jumps never extend a deadline. The usage meter and the updater
// are separate programs; this package only leaves them a lock, a verified
// desired-state file and the forwarding latch.
package agent
