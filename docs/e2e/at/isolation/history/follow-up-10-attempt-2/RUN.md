# Isolation run procedure — Follow-up 10

Run from the worktree root:

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
make -f mk/at-isolation.mk at-isolation
python3 -B tests/e2e/at/isolation/review_followup10.py
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint 'go-with-disk-tmp='
```

The runner allocates disk TMPDIR/GOTMPDIR under `.agent/tmp`, checks
`pgrep -af 'gates.sh|mk/at-'` before each stack, and waits while two other suite
process trees are active. Teardown uses Python `finally`, with no shell cleanup
traps. The root lint variable override uses the runner's disk-temp lifecycle
instead of the root recipe's shell trap; vet/staticcheck commands are unchanged.
Only `fvpn-w4-isolation-*` Docker objects are created or removed. The final
inventory must show zero containers, images, networks and volumes.

## AT10 and D-120

The 100-reload mechanism stream and its analyzer are unchanged: one outstanding
UDP datagram, 1.5 s timeout, 200 ms sleep, at most five eligible losses, zero TCP
timeouts. Each tolerated loss must satisfy all D-120 conditions, including
captured ingress inside/on the boundary of exactly one verified guard, guard
and render-to-install bounds each at most 100 ms, and the next datagram succeeding.
The original printed verdict and complete inputs are under `reload-stream/`.
This rule never applies to the separate 80 ms diagnostic below.

The owned stack wrapper sets `FVPN_E2E_GRANT_SEAM=1` while preparing a fresh
runtime; the production control binary remains the normal default elsewhere.
Before enrollment, configure the TEST ONLY offline window to 60 seconds and
leave it fixed for the entire run. The first assembled scenario waits for
65 seconds of identical `MemberGrantInvariants`, then requires clean ready /
passed publication, applied=desired generation, bindings, no freeze and a
native counted control. Baseline/meter/tunnel checks are retained from the real
accepted observation.

Capture the invariant snapshot and arm false using `SetMemberGrantsReady`.
Wait for a natural signed empty lease and an actual HTTP 202 ready/passed
observation naming that lease, forwarding false, exactly `awaiting_member_grants`,
lease/qos/source_binding `AWAITING_MEMBER_GRANTS`, and exactly one lease-stage
error with that code. Only after this receipt arm true. Require a distinct
accepted serving report joined to a new populated signed lease, then count
native forwarding independently. Compare all invariants byte-for-byte.
`member-grants-transition.json` retains snapshots, flips, joins and counters;
`member-grants-wire.json` retains original JSON strings and SHA-256 hashes.
Historical isolation 4,767 / 5,021 / 5,025 ms and devstack 3,980 / 4,112 ms
remain separate samples. No generation forcing, projector stop, mapping freeze,
unpublish, transport fault or manufactured observation drives this transition.

## Exact producer timing and ISO-F6-03

Before enrollment, the owned helper enables agent-core 28's TEST ONLY
`test_timing_path`. The agent appends detached `TimingEvidence()` snapshots
after every real Apply, live VerifyServing and blocked audit, including failures,
plus renewal, baseline-writer and heartbeat records. All source kinds participate
in contiguous per-process sequence validation. The source-node JSONL is exported intact after
the agent stops, retaining every process run UUID, boot UUID, decimal sequence,
wall/monotonic anchor, accepted lease deadline and matching D-106 digest.
Clone wire bodies remain separate from the source timing. The shared supervisor
truncates normal logs on process replacement, so the harness retains
`agent/watchdog-before-restart.log.txt`, `-before-clone.log.txt` and `-final.log.txt`;
the unsuffixed logs concatenate those original segments. Renaming each open log
before replacement preserves shutdown writes; export occurs after measurements.

The helper also retains boot ID, time-namespace identity and offsets for the
producer node, packet recorder and every probe. Offline review rejects sequence
gaps, mismatched boot IDs/clock offsets/digests, invalid timing order, missing ingress or capture
drops. Nanoseconds remain decimal strings; comparisons use integer arithmetic.
Docker namespace inodes may differ; the comparison requires identical kernel boot
and exactly zero monotonic/boottime offsets to the initial namespace, with
namespace IDs retained. After restart, read the live agent's `/proc` clock namespace
and offsets as well. Linux documents this offset basis in
[time_namespaces(7)](https://man7.org/linux/man-pages/man7/time_namespaces.7.html).
No timestamp allowance is introduced. Failed or partial commands never establish a completed guard. Both verifiers
from quarantine recovery remain separate, including a non-serving deny-all read.

The fleet ceiling is already fixed at 60 seconds before enrollment. The later
AT13 scenario requests its own capacity generation after AT10 completes.
A 70 s, dual-phase UDP diagnostic captures at least two real renewals. It uses
an 80 ms read timeout; **every loss remains a loss**, even inside a guard.
`producer_timing.py` correlates each loss's send-to-completion window and captured
ingress with exact Apply, install, release, outer/inner guard and verifier
brackets. It retains the nearest Apply and preceding completed Apply/verifier,
including final render, nft ruleset, millisecond precision text and aggregate
tc reads. These are command/capture correlations, not exact kernel drop times.
Passive netlink receipts are never substituted for producer timing.

The finite-member scenario waits for a natural accepted renewal whose member
lease end equals the real finite grant end and whose node expiry is later.
The 60-second fleet ceiling stays fixed. An earlier lease ending before the
member deadline cannot establish that independent expiry case. Require an
accepted node lease extending at least 25 seconds beyond the member end, stop
control five seconds before the member end, and check that all five original
post-member-expiry probes finish before that held node lease expires. These
are fixture scheduling bounds, not allowances on a signed deadline.

After control loss, read-only SQLite lease/anchor state gives the absolute
monotonic deadline D. Phased 100 ms probes and an arbitrary 37 ms phase share
an explicit D−101 ms boundary sample. D-104 requires last positive ≥ D−101 ms,
first drop ≤ D+100 ms, no positive at/after D or later than the first drop, and
complete send/ingress/read evidence. No cadence allowance is added. The packet
predicate is independent of strict renewal continuity. The fleet setting is never rewritten; its original audit and control-service
restart receipt remain retained.

`producer-operations.jsonl` is the original source stream;
`producer-timing-analysis.json` is the full correlation;
`continuity-analysis.json` and `expiry-timing-analysis.json` expose each window.
Outer-guard and install-command p50/p90/p99/max use nearest rank over completed
guards. `completed_guard_install_command` gives the matching install population;
`failed_applies` separately retains every failed operation and its actual brackets.
The source `baseline` rows name real agent causes; `agent.log.txt` and
`watchdog.log.txt` retain normal producer reasons, decisions and writes. Neither
a successful restart adoption nor a bounded D-104 pass resolves the historical
ISO-F6-03 failure. Every pre-D ingress loss remains counted.

The historical run `7779d83c` had 17 losses among 1,396 probes and no exact
producer timestamps. New timestamps cannot retrospectively classify them.

## Recovery, D-106 and D-093

The existing labelled clsact/flower/mirred injection, dirty newer generation,
removal and clean newer generation sequence is unchanged. The ordinary agent
recovers without a restart, link raise, route/address repair or expectation
override. The six G2 assertions use genuine wire bodies, signed leases, durable
applied/member-path state, kernel topology and counted packets. Historical
rejected tc evidence is separate from current objects. D-106 uses the exact
canonical projection `{reads, objects, truncated, tc}`; total objects are not a
foreign-object count. The original unretained follow-up 6 creator stays unknown.

[D-093 reconciliation](d093-reconciliation.md) keeps simulator claims separate
from these real source-node observations and counted packet checks.

## D-098 archives

The corrected final snapshots are immutable: follow-up 6 (`bf671f3a`, 44 files),
follow-up 7 (`041e9e48`, 52), follow-up 8 (`7779d83c`, 80), and follow-up 9 (`2be2797b`, 86). Each has SHA-256
pins in `history/follow-up-N/archive.json`. Follow-up 8 includes all root,
support and reload-stream files. Its deadline, recovery and continuity pins
match netlab 20's exact requested digests. Handover already reads immutable
archives/fixtures; its living AT10/AT13 path-string rejection tests do not read
the files. No handover edit or claim-pin move is required.

Follow-up 9 expiry remains failed in its archive: 96 early drops, 65 subsequent
positives and 41 final expiry drops. That baseline writer remains unattributed.
New producer evidence describes only the new run.

First follow-up 10 attempt `17c14696` is retained separately under
`history/follow-up-10-attempt-1/`. Its manifest explicitly labels inherited
unchanged files; those files are not new measurements. After archiving, clear
generated root artifacts before a fresh run. The offline reviewer refuses to
use packet files when the current whole-node check did not execute its window.
Raw archived captures retain their terminal blank lines; four explicit archive
attributes disable only Git's blank-at-EOF diagnostic for those files.
