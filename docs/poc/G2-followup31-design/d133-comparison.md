# D-133 comparison declaration — fixed before measurement

2026-11-03. **No comparison runs in follow-up 31.** B is C1 as specified in
[candidates.md](candidates.md), conditional on its closure trade being reviewed
and every blocking [proof obligation](proof-obligations.md) resolved. Without
that decision, no B implementation or D-133 acceptance is authorized. A remains
main's current unconditional deadline guard. A performance result cannot waive
a safety proof or an owner decision.

The historical D-133 basis is the [merged-PR excerpt](../../../tests/handover/testdata/history/follow-up-20/D133-PR344.txt)
and [handover interpretation](../../handover/follow-up-20.md). The current
foreign-CPU qualification supersedes earlier vmstat-threshold admission for this
comparison. [D-120's five conditions](../G2-at10-reload-stream-2026-10-31.md#d-120-acceptance-rule--2026-10-31)
and its analyzer remain unchanged. D-138-style disclosure of interleaved timing
populations is not permission to discard a B-only failure.

## Sources, order and populations

Freeze **A1,B1,A2,B2**, one attempted measurement per leg, same host and boot,
kernel, nft/iproute2, topology image, Go/runtime settings, CPU environment,
fixtures, policy sizes, streams, capture modes and command observers. Pin the
full source trees, binary/image digests and scripts before launch. A is the exact
main parent of the reviewed B implementation. This design's reference A is
`cc0ed4ae713f6c8242171f923bdc799a0c07a74c`; if main changes the enforcement behavior
before implementation, review the source delta and re-declare the baseline
**before** any measurement. B cannot be pinned until it exists. Do not combine
different B revisions within the quartet.

Each leg uses a fresh, identically configured lab. Migrate/initialize A or B
under a closed baseline before traffic; record that cost separately. Initial
Apply generation 1000 is still one of the required 101 D-120 records. All 100
timed reloads use the original campaign's models and behavior, including A2
removal/re-addition; do not simplify them to obtain C1 eligibility.

Run these predeclared populations in the same order in every leg:

| Population | Fixed sample and instrumentation |
|---|---|
| Ordinary cost | One untimed warm Apply, then 30 Applies of the spec-shape model and 30 of the same expanded model used for comparison, interleaved by model size in identical order. Freeze exact identities, counts and models before implementation measurement; publish rule/element/script counts. Lifecycle resource sampling and NFLOG off. Retain every attempt and failure. |
| AT10 comparison | One original 100-reload AT10/AT13 scenario; 101 Apply records, original stream/analyzer; mandatory UDP sidecar and three-hop traces plus the same opt-in TCP header observer in every leg. No NFLOG in this acceptance population. |
| Exact-rule diagnostic cost | One additional 100-reload scenario in each leg with **identical** NFLOG decorator on the two terminal guard drops, group/snaplen/queue/precision configuration and collector as follow-up 30. Keep this separate from ordinary/AT10 timing. Its failures count; it cannot replace the first AT10 run. |
| Safety controls | Identical fixed deadline/revocation/late-commit/kill/verifier/rollback vectors from the proof plan, including arriving revocation during hand-over. Record injected-fault results separately, with expected closure versus actual evidence. |
| Consumers | Full existing netlab and real-agent suites, plus assigned updater/installer/meter consumers, with the same observer settings across A/B. Cross-lane owners supply their frozen commands before the quartet. Missing consumer leg blocks the quartet. |

Ordinary spec/expanded shapes and consumer commands must be frozen in a dated
execution annex before the first measurement, with no change to sample counts,
ordering, criteria or original AT10 inputs here. Their source dependencies do
not exist yet; filling in their hashes later is not permission to tune the
experiment after seeing results. Any substantive design/criterion change needs
a new declaration and a separate quartet; keep the old one intact.

No retry of a spent leg to get green. Preserve and finish other safe legs after
a failure for attribution. A demonstrated unauthorized forwarding or containment
failure stops further live traffic, closes the owned lab and leaves the remaining
legs missing. A new quartet requires a new assignment, never replacement rows.
Preflight refusals have their own records and are not test results.

## Admission and qualification

The orchestrator schedules a quiet slot with **no other lane session**. Only
the future assigned task prefix may create Docker objects. Before every make,
docker or go-test command, obtain `free` from `bash "$FVPN_HOLD_SH"`; honor the
lane go wrapper for build/vet/run/install/generate/tool as well. Exit 125 means
not admitted, never a test outcome. Bounded waits and every rejection are saved.
Use the disk-temp wrapper, discover cache via `go env GOCACHE`, keep commands
foreground, and clean all owned objects. No host load injection in this quartet.

After reservation, settle 60 seconds. Require a completed quiet interval before
startup and a fresh one after startup before traffic. Monitor at four-second
cadence, including a final interval covering the end. **Every interval overlapping
the measured population must have foreign CPU ≤ 0.5 cores**, no other lane suite
or non-owned lane container, valid process/cgroup accounting, and interval
length ≤ 5 seconds. A missing sample or coverage gap is unqualified. Record
vmstat/load without using it as a substitute or a new threshold.

Use the follow-up 30 exclusion rule exactly: runner descendants; owned-container
cgroups; children of kthreadd; named runtime processes matching
`dockerd|containerd|containerd-shim.*|docker-proxy|runc`. Publish each excluded
class and CPU. Other work, including git, unrelated jobs and the shared
`fvpn-pg` / `postgres:17-alpine` service, counts as foreign. That exact shared
service is exempt only from the other-lane-container exclusion, never from CPU.
Do not pause unrelated jobs or broaden exclusions. Disappeared processes' final
unread CPU ticks remain a disclosed sampling limit, not evidence of zero cost.

Qualify each population and the whole leg (all its populations). A breach
spends and disqualifies the leg regardless of functional results. Keep its
vectors visibly UNQUALIFIED, never pool them into qualified percentiles. A
quartet with any missing/unqualified leg is **NOT ACCEPTED: incomplete quiet
comparison**. This is deliberately different from saying its tests failed or
that foreign CPU caused a particular packet loss.

## Identical instrumentation and truthful guard semantics

One common, frozen TEST ONLY observer revision runs with both binaries. It
buffers monotonic and wall command brackets, result codes and actual scripts,
then hashes/serializes them outside Apply. Both have the same production timing
surface and disabled resource-sampling settings in ordinary cost. Both get the
same TCP/UDP observers in the AT10 population. NFLOG is enabled in both or in
neither for a population, never just A.

The current [trace recognizer](../../../tests/netlab/cmd/fvpn-netlab/trace.go)
recognizes an exact one-line guard deletion; B's token-fenced release has more
statements. A later common adapter must recognize both **exact approved shapes**,
join their producer phases, and emit the existing `guard_install`/`guard_release`
records for real commands. Missing/duplicate/ambiguous matches fail. No invented
zero-duration guard, synthetic empty batch or renamed inactive interval may be
fed to the analyzer. The adapter changes observation only and is part of both
A and B's instrumentation, with costs disclosed.

C1 still installs a real −151 guard and deletes it. In A it blocks all members;
in B it blocks all except the narrowly capped eligible intersection. Thus the
existing analyzer's timestamps continue to mean **guard-present interval**;
they must not be described as an all-member outage in B. Its kernel commit
remains somewhere inside the command bracket. `Deadline.InstallOK` is command
success, not verification. No D-120 field, loss classifier, deadline or predicate
changes. B's extra zero-loss rule below prevents using D-120's tolerated guard
losses as evidence of a hitless success. If the adapter cannot satisfy the
unchanged evidence checks truthfully, stop and request review; do not modify
the analyzer for a pass.

Disclose observer cost separately: commands and bytes per Apply, number of clock
samples, parser/serialization CPU, script-byte delta, hash/write time outside
Apply, capture/collector start-stop, records/IP/derived bytes, receiver/kernel
drops, timestamp precision, missing NFLOG sequence attributes, and extra
outside-Apply TCP/conntrack snapshots. Follow-up 30's logging adds 138 bytes to
its two guard drops; remeasure B rather than assuming equal cost. Paired
NFLOG-on/off populations describe observer impact with order confounding;
they are not a precise isolated-overhead estimate. A/B policy differences are
separate from common instrumentation overhead. An observation failure is a
retained failure, not zero packet loss.

## Measurements and fixed decision rule

For each leg and model/population publish raw vectors, N, failures, missing/censored
observations and nearest-rank **p50/p90/p95/p99/max**. Keep successful, failed,
baseline, migration and fault-injected Applies separate. Do not divide losses by
only successful reads. Fixed elapsed metrics:

- Whole Apply; final render→installed-by; install command; installed-by→release
  start; release command; outer guard upper/lower brackets; each enforcement
  verifier; nft JSON read; precision timeout read; aggregate tc reads; agent's
  reverify and ACK; actual baseline→next successful forwarding recovery.
- TCP read elapsed/deadline failures, send/ACK/retry identities, reconnects,
  UDP loss/count, packet ingress-without-egress in both directions, exact-rule
  matches, eligible/ineligible classification, all denial/control probes and
  capture completeness. A successful retry still records the original loss.
- Old/new absolute deadlines and read-back remaining lifetimes; last admitted /
  first denied packets; revoke receipt, candidate commit bracket and first
  sustained cutoff; queued revoke wait; per-member and node QoS ceilings, counter
  monotonicity and reset/final-sample evidence.

**Acceptance requires every condition below, fixed now.** These are proposed
experiment criteria for design review, not new production SLOs or spec values.
Do not relax them after measurement.

1. All four legs qualify, all proof obligations and cross-lane consumers are
   complete, and the owner has explicitly resolved P0. No newly unauthorized
   admission, deadline extension, isolation bypass, meter error, higher-than-
   authorized ceiling, false ACK or stale-release reopening. Missing evidence
   fails this condition. Old failure statuses remain unchanged.
2. Both B AT10 runs pass the **unchanged** D-120 analyzer and original AT13
   assertions. Across the full B comparison, zero unexpected TCP read deadlines,
   reconnects or wrong echoes. In each eligible A1 reload window, zero observed
   node-ingress-without-egress packets and zero UDP echo loss. Any B-only timeout
   or new unexplained failure is NOT ACCEPTED, including in the separate
   diagnostic population; a later pass cannot replace it.
3. Every successful B install stays within the unchanged 100 ms render/install
   budget. All measured B completed AT10 guard upper brackets are ≤ 100 ms.
   Failed/late operations stay in the report, never recast as completed guards.
   Revoked-member cutoff, measured from the same trigger, is no later than
   paired A (B1 vs A1; B2 vs A2), and never beyond applicable old/new authority.
   If timestamp uncertainty prevents that comparison, it is unresolved, not pass.
4. In **both** pairs, B has fewer eligible-member missing egresses and fewer
   guard-attributed retransmissions than A, and strictly lower TCP p99 and max
   read elapsed. If A has no loss/retries, the improvement is unestablished;
   do not rerun until it does. B TCP p50/p90 must also be non-worse. Report p95
   even though it is not a separate threshold. This tests a mechanism benefit,
   not reconstruction of historical B1.
5. Cost budget for ordinary spec and expanded shapes separately, and AT10
   separately: B whole-Apply and verifier p50/p90/p99/max must each be no more
   than **1.10 × paired A + 1 ms**. This conservative predeclared trial budget
   exposes a material processing regression; it is not a change to any existing
   time limit. Install/release vectors and bytes are always disclosed, even when
   total cost meets the budget. A failed cost criterion is NOT ACCEPTED for
   performance; owner review may choose a different *future* experiment, not
   relabel this one. Do not pool away a regression in one B repeat or shape.

A failures remain A failures: they may establish an existing limitation but do
not excuse a B failure. Both-pass/no-improvement means **safety results retained,
benefit unestablished**, not a claim that the demonstrated mechanism disappeared.
Both-fail, mixed repeat, missing consumer, instrument discrepancy or quietness
failure means **unresolved / NOT ACCEPTED** with the actual reason. Report
functional results, host qualification, safety and performance as distinct
fields. No causal attribution from load correlation alone.

Historical follow-up 27 B1 stays FAIL/unexplained; follow-up 28 gate 2 stays
FAIL; follow-up 29's failed collector attempt and follow-up 30's two unqualified
live runs remain. All follow-up 25–30 evidence stays immutable. Publish the
entire quartet and refusals with hashes and working links, not only successful
legs. A new clean result cannot fill B1's absent TCP/timer trace.
