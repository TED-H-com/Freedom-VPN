# Live NFLOG prerequisite — declared before execution

TEST ONLY. This is a collector check, not a measurement or the F27-01 window.
At most three attempts named collector1–collector3; retain each and fix offline
between attempts. Stop the guard item if the third fails. Each attempt reserves
at most ten minutes, waits 60 seconds, checks host quietness, and releases after
teardown. Reservations are separated by offline review.

Use the existing owned lab topology and unmodified `guard_observe.py` collector:
`tcpdump --immediate-mode -U -nn -s 0 -i nflog:29 -w <scratch>.pcap`.
Backend default precision, group 29, snaplen 128 and queue-threshold 1 match the
window exactly. A separate lab-only forward chain in `inet fvpn`, generation
30000, logs and drops five known upload UDP packets and three reverse UDP
packets. Unique IPv4 IDs, lengths and tuples identify each. One additional
upload packet targets an unlogged drop rule as a negative control. These rules
are used only in this check and removed before teardown.

Pass requires all eight records decoded, exactly one match per expected packet,
no extra record, receiver loss zero, and captured/received/decoded counts equal.
Every record retains its actual pcap precision and kernel timestamp presence;
absence of a kernel timestamp is reported, never replaced by delivery time.
Retain rule/readback counters, injection headers and send brackets, parser output,
receiver statistics, command/source receipts and cleanup. No payload or link
addresses are published; raw pcap is deleted in disposable scratch.

Host quietness follows follow-up 30's instructions: no foreign suite process,
no foreign running `fvpn-*` container, foreign CPU at most 0.5 cores per interval.
Sample `/proc` every four seconds and report each excluded class separately;
record vmstat without using it to decide. PID/start-time pairs avoid reuse errors.
Process exits between proc snapshots are a measurement limitation, recorded as
missing final samples. Continuous checks cover startup and execution; a breach
marks the attempt unqualified and is retained. All nested make, docker and go-test
commands consult `FVPN_HOLD_SH`. No private admission location is recorded.

The read-only inventory identifies `fvpn-pg` using `postgres:17-alpine` as the
shared PostgreSQL service described by the engineering contract. That exact
name/image pair is recorded as shared infrastructure, outside the **other lane**
container exclusion. Its CPU still counts as foreign; every other non-owned
`fvpn-*` container blocks admission. See [the identity record](shared-service.json).
