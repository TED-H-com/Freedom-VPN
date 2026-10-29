// Package agent is the exit-node agent (§5, §6, §14.7, §16.3).
//
// Boot order is fail-closed: the baseline is installed before identity,
// a lease or a policy can enable member forwarding. Desired state is verified
// before it is stored or written for the updater. A 304 does not move a lease.
// Wall-clock jumps never extend a deadline. The usage ledger is opened from
// this process: sampling starts only after forwarding is admitted, and a
// stop or hand-off takes a final sample before the next epoch. A ruleset
// cleared outside this process (table, forward chain, or generation marker)
// opens one meter epoch through the meter's sample path when a usage counter
// was live; idle counters do not. The updater
// stays a separate program. This package leaves it the node lock, a verified
// desired-state file and the forwarding latch.
package agent
