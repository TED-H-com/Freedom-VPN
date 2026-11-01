# Isolation run procedure — Follow-up 13

Product baseline: `334d56e466c7689fdbf16e078041afbfbd9bf4fb` (main including
agent-core 32 / PR #376 and isolation 12 / PR #373). The preflight command
`go env GOCACHE` returned `/data/ted-h-cache/go-build`; no cache path is
hard-coded in the runner. Runtime source identity is recorded in each AT row.

Before the one assembled run, retain `vmstat 5 7` and
`pgrep -af 'gates.sh|mk/at-'`. The public `host-preflight.json` keeps the CPU
samples and sanitized suite inventory; original command output remains in
`.agent/f13/`. These host observations do not attribute a failure.

Run from the worktree root:

```sh
make -f mk/at-isolation.mk at-isolation-lint at-isolation-test at-isolation-build
make -f mk/at-isolation.mk at-isolation
python3 -B tests/e2e/at/isolation/review_followup10.py
python3 -B tests/e2e/at/isolation/producer_timing.py --replay-first-attempt
python3 tests/e2e/at/isolation/run.py go test -count=1 ./tests/handover/...
python3 tests/e2e/at/isolation/run.py make go-lint 'go-with-disk-tmp='
```

The runner allocates disk TMPDIR/GOTMPDIR under `.agent/tmp`, checks
`pgrep -af 'gates.sh|mk/at-'` before each stack, and waits while two other suite
process trees are active, and rechecks after startup. Teardown uses Python `finally`, with no shell cleanup
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
leave it fixed for the entire run. The first assembled scenario establishes a native counted control, then waits
for 65 seconds of identical `MemberGrantInvariants` so ordinary initial FUP
projection settles before the snapshot. It then requires clean ready / passed
publication, applied=desired generation, bindings and no freeze. Baseline/meter/tunnel checks are retained from the real
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

Before enrollment, the owned helper enables agent-core 29's TEST ONLY
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
Outer-guard, install-command and render→installed p50/p90/p99/max use nearest rank over completed
guards. D-134 treats these as the measured cost of the atomic guard, with no
fixed historical ceiling; 52.596424 ms was a sample, not a guarantee. `completed_guard_install_command` gives the matching install population;
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

The second attempt `5e9e4041` is independently pinned under
`history/follow-up-10-attempt-2/` (98 files). Its initial counted flow incorrectly
followed the stability wait and triggered ordinary projection before false; its
AT10 failure remains evidence. The corrected sequence above precedes the third
run. No product failure is erased by either later run.

## Follow-up 11 host consumer and D-134 / D-135 wording

Each Apply joins its own `host.before_apply` / `host.after_apply` and sanitized
command status by the original process run ID, boot ID and decimal sequence.
The consumer validates sample types, brackets outside Apply and inside its
anchor, counter bounds, and the `nft`/`tc`/`other`, exit-code and signal fields.
Missing exit fields are unknown, never zero. Missing host blocks, `ok=false`,
zero total tick deltas or decreasing counters produce unavailable context,
with no idle percentage. Missing verification is unavailable, never a failed
verification; baseline cause names cannot manufacture verifier records.

The conservative TEST ONLY diagnostic uses a positive aggregate total tick
delta, zero idle **and** iowait tick deltas, and more than one runnable process
in both samples. Only a failed install exceeding its recorded bound, with no
release and those contemporaneous samples, receives “fail-closed install
overrun with recorded host contention”. This reports coincident observations;
it is not a cgroup measurement or scheduler causal proof. All failure causes
remain `unproven`; no failure or lost packet is removed from totals. Partial
and failed operations stay outside completed-guard distributions.

The first-attempt replay reads only `history/follow-up-10-attempt-1/` and writes
`first-attempt-host-review.json`, leaving original bytes and pins untouched.
All 49 failed installs have host label **unavailable** and cause **unproven**;
all 279 renewal losses remain counted. Agent-core 29's controlled contention
experiment did not reproduce an overrun; chain-gate correlation is not cause.

D-134 describes the conservative expiry transition as **[D−101 ms, D]**,
from the 100 ms installation reserve and 1 ms truncation. The unchanged D-104
first-drop bound D+100 ms is an observation bound, never permission for a
post-D positive. Every pre-D and renewal loss remains counted. No literal
zero pre-D-loss claim is made. An overrun never opens the guard; the baseline
may replace it atomically while member forwarding remains closed.

D-135 rollback installs the **node-wide generation-0 baseline**, preserving
revocation without restoring a previous generation. Admitted siblings also
lose forwarding; restoration requires the next successful Apply and live
checks with current authorization. The 100 ms install bound does not bound
that outage. No acceptance predicate is changed.

Before any rewrite, commit `0a9cd15b` archived run `bfcc3880` under
`history/follow-up-10/`: all 100 files byte-identical to the baseline and
SHA-256 pinned. Earlier archives and both attempts are untouched. Handover
reads immutable evidence; no D-132 one-file exception is needed.


The follow-up 11 sample is `edd986c4-16d2-4aab-b92f-baa19f8bd829`, runtime
source `2304f77a733978405a35eea3b8d0a075027ce88e` with `source_dirty=false`.
Full foreground command 1,199.880 seconds, exit 2 (lab=0, stack=1); AT10/AT12
pass, AT11/AT13 retain operator blocks. The completed-guard and host results
are in `summary.md`; these are independent measurements, not a quiet A/B leg.

## Follow-up 12 lifecycle consumer and restart observations

First commit `b597f995` archives all 103 root files of run edd986c4 under
`history/follow-up-11/`, byte-identical with SHA-256 pins. Earlier archives
are untouched. Handover reads immutable evidence; mutable root strings are
rejection inputs, so the D-132 one-file exception is not needed.

Runtime source `99e58bff0cf7276e36cb04f35605835208c8638e` adds consumer validation
and observation only. Agent-core 31 enables netlab 22's per-Apply `commands`
under the existing TestMode/TestTimingPath gate. The consumer checks kind, phase,
availability, start/return/wait/cancel stamps, resource before/after brackets,
started/ok, exit/signal/CPU/switch fields, and containment within the exact
Apply. Records join only by run/boot/sequence, with command indices. Descriptions
name the observed phases; absent records are unavailable. Historical replay
preserves all prior results apart from these additive unavailable fields.
Command cancellation, exit and host observations do not establish scheduler
causality; causes stay unproven. Exact child-start and hidden host ancestors
remain explicitly unavailable when the producer says so. D-106 is unchanged.

The normal restart now uses Docker stop then start, retaining a SQLite lease
snapshot while stopped. This adds a measured snapshot interval to the outage.
`restart-lifecycle.json` records stop, snapshot and restart command durations,
lease ID and original monotonic anchor. It is not an exact SIGTERM/exit trace.
The cloned-identity fixture retains its own whole-call bracket and Docker
finished/started timestamps when available. Terminal source/clone service stops
retain durations and before/after lease snapshots in `receiver-closure.json`.
A stored lease surviving does not prove that it remains eligible for serving.

D-136 correction: only recovery from a retained Apply error uses the verified
deny-all candidate. The ordinary stale-lease path installs the node-wide
baseline, ACKs, renews, then Applies, verifies and heartbeats. `policy_missing`
alone does not establish generation mismatch. Any observed transition is
bracketed from baseline to the next successful Apply, with later verification
and heartbeat/egress evidence kept distinct; every loss remains counted.
There is no finite recovery guarantee and the 100 ms install budget does not
bound the outage. See the agent-core 31 generation-change design note.

The expanded command stream is stored as lossless `producer-operations.jsonl.gz`
when needed for publication. Decompression is compared byte-for-byte to the
export before the raw working copy is moved to private run storage.
`read_operations` accepts raw historical JSONL and gzip; the evidence hygiene
test scans decompressed content and validates every JSONL record. Successful
Apply inventories reference original command indices by run/boot/sequence;
failed/nearest Apply details retain full records. No record is sampled away.

After the assembled run, replay with:

```sh
python3 -B tests/e2e/at/isolation/review_followup10.py docs/e2e/at/isolation
python3 -B tests/e2e/at/isolation/review_policy_missing.py docs/e2e/at/isolation
```

The lifecycle reader/compact review and D-136 review are offline consumer
changes after runtime commit `99e58bff`; they do not alter the running binaries.
Their synthetic regressions include absent recovery and all retained losses.

Final run `1c25ee44-1872-4cce-a9f5-d52491d87095`: full command 1207.405 s,
exit 2; AT10 blocked 8/1/0 (ISO-F6-01 5/6, missing dirty-generation report),
AT11 blocked 2/1/0, AT12 pass 10/0/0, AT13 blocked 6/1/0.
The sixth predicate is unchanged. Local generation-30 failures and kernel
denial do not substitute for the absent current central observation.
Ordinary stop/start retains the lease, but exact signal/exit events were
unavailable. The later clone fixture's event observer records SIGTERM then
SIGKILL after 20.007193 s, exit 137. It is not a graceful-stop qualification.
See `summary.md`, `iso-f6-01-observation-gap.json`, and the three dated hand-offs.


## Follow-up 13 observation boundary and regression sample

D-098 commit `822d56b0` archives all 108 root files of run `1c25ee44`
byte-identically under `history/follow-up-12/`, with SHA-256 pins. Earlier
archives are untouched. Handover reads immutable evidence; its root paths are
rejection inputs, so no D-132 exception edit is needed. The historical sixth
assertion remains blocked.

The existing 90-second control-restoration budget now includes a current
accepted serving report and a read-only SQLite outbox count of zero. The
readiness samples are retained. ISO-F6-01 refuses injection if this boundary
was not reached. The foreign filter remains until an accepted dirty-candidate
`TC_FOREIGN_FILTER` report with forwarding=false and an in-window observation
arrives. The existing maximum twelve samples at 500ms is unchanged; exhaustion
permits cleanup only, not a passing boundary. All six assertion names and
requirements remain. Assertions requiring an installed verifier continue to
consume that verifier. Assertion 6 separately joins `stage=apply_candidate`,
`verified=false`, generation and failure time to the original desired wire
and failed Apply command trace; an old applied ACK is allowed.

Terminal source/clone stops use an owned TEST ONLY helper inside each lab
container. It opens the agent's pidfd, sets the existing supervisor pause
marker, sends SIGTERM to that pidfd and polls exit readiness. It records send
and exit brackets, the 13-second comparison, and /proc zombie exit status when
available. The supervisor's escalation policy is unchanged. Stored leases are
read before and after. Exact Step-context cancellation is not exported, so
r18 M-1 is **not observable here**; receiver completion alone is not a caller
cancellation timestamp. Docker container stop/restart and terminal process
stop measurements remain separate.

D-135 / corrected D-136 use baseline→next successful Apply, with every captured
loss counted and later verifier/heartbeat recovery required. Hold wire review
checks `policy_missing` plus `STALE_GENERATION` at `stage=lease`, and retains
actual streak/alarm summaries. The baseline stays. Operator AT11/AT13 rows
remain blocked. No timeout inflation, loss waiver or retrospective promotion.


Final follow-up 13 run `38074149-1137-4782-80fd-8669c36c6fcb`, runtime source
`c521cb0018844eac1de24ee243c858518bb1a5e3`, source_dirty=false: full foreground
command 1216.858 s, exit 2. AT10 9/0/0 and AT12 10/0/0 pass; AT11 2/1/0 and
AT13 6/1/0 retain operator blocks. All six ISO-F6-01 assertions pass only for
this new sample. Final statistics and proof limits are in summary.md.

The offline hold reviewer (commit `2c51c176`) runs as:

```sh
python3 -B tests/e2e/at/isolation/review_core32.py docs/e2e/at/isolation
```

The full producer stream is available as lossless gzip both at the current
root and under immutable `history/follow-up-13-stream/`. Its manifest pins
compressed and decompressed SHA-256 forms; byte identity was checked against
the original export. All 764 archive file pins verify. Root living evidence is
not used as an immutable claim pin. The old observation-gap root is an explicit
pointer to its historical blocked snapshot, preserving the agent-core design
note's link without changing any outside-path document.
