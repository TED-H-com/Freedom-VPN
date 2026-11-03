# F27-01 exact guard-drop window — follow-up 30 declaration

TEST ONLY. One reservation, one AT10 100-reload run, no retry. Execute only after
this follow-up's live collector prerequisite passes. Historical B1 remains FAIL;
F27-01 remains unresolved unless B1's actual failure is explained. No historical
status, D-120 predicate, probe, 1.5-second timeout or analyzer changes.

The [follow-up 29 declaration](../G2-followup29-evidence/guard-declaration.md)
stands for instrumentation, correlation and cost. Keep its `f29g` prefixes,
group 29, snaplen 128 and queue-threshold 1: changing identifiers offers no
measurement benefit. Use the corrected collector's backend default precision,
exactly as live-checked here. Only the owned-container prefix/evidence destination
and the orchestrator's replacement admission rules change.

The executor still decorates just the two `fv_deadline_guard` rules with logging
immediately before their unchanged terminal drops. No synchronous read is added
inside install/release. Capture original TCP, UDP and producer/executor timing;
retain all 101 Apply scripts and hashes, including partial/failing results.

For every missing node egress, match NFLOG tuple, IPv4 ID, sequence/ACK, flags and
length to independent ingress and endpoint observations. Reverse-direction
matching accounts for DNAT between ingress capture and the forward hook. Name
rule direction, generation, install-start/installed-by/release-start/release-end,
and the bracket containing the kernel timestamp. If that timestamp is absent,
report matched ingress bounds and explicitly do not substitute delivery time.
Retain ambiguous, absent and contradictory matches. Absence of logs never proves
no guard drop; a missing segment without a matching log stays unattributed.

Disclose install/release/outer-guard distributions, exactly added script bytes,
log counts and bytes, actual timestamp precision/presence, loss statistics,
collector start/stop command elapsed times, and TCP-snapshot cost outside Apply.
The external command timer writes only after start/stop returns. These numbers
bound the instrumented diagnostic; there is no uninstrumented control and no
isolated incremental-overhead claim.

Reserve at most 20 minutes, wait 60 seconds, then require a quiet check before
startup and a fresh check after startup immediately before the test. Continue
checks at four-second intervals throughout execution. No other lane suite or
container may run; foreign CPU must be at most 0.5 cores in **every** interval.
Exclude only runner descendants, owned-container cgroups, children of kthreadd
and the named container-runtime processes from CPU; record each excluded class.
Record vmstat without a threshold. Retain checks and wait times. Any in-window
breach makes the window unqualified and still spends it. No renewed reservation.

If guard logs identify missing segments, distinguish demonstrated candidate
mechanism from an explanation of B1's 1.501408-second read. If no segment is
missing, record that without promoting B1. Any guard hand-over improvement is
only a proposal with a separate D-133 comparison, never a production edit here.
