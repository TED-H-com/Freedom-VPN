# Isolation follow-up 11 — 2026-10-31

One assembled run **`edd986c4-16d2-4aab-b92f-baa19f8bd829`**, live source **`2304f77a733978405a35eea3b8d0a075027ce88e`**,
source dirty **false**, product baseline
**`45209609d33d567c5f04d49441799cdd3efac59b`** (main after agent-core 29 / #356
and netlab 21 / #355). Full foreground command: **1199.88 s**,
exit **2**. This is the next independent sample after follow-up 10
and agent-core 29's quiet legs, not a controlled A/B or a qualified quiet run.
No production code or acceptance predicate changed.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0** |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0** |

Spec §19 AT10–AT13, §8.2–8.3, §16.3; ISO01–ISO04, ENT02, SEC01, DNS03.
AT11's native platform matrix and AT13's full continuous fault/platform matrix
remain blocked, with [two dated hand-offs](requests.md#follow-up-11-requests--2026-10-31).
Cross-compilation does not establish physical arm64 execution.

## Archive and source identity

D-098 was the first commit, **`0a9cd15b`**. All **100** root files from run
`bfcc3880` match baseline-main bytes and their SHA-256 pins in the immutable
[follow-up 10 archive](history/follow-up-10/archive.json). Both earlier attempt
directories and all earlier archives are unchanged. Handover uses immutable
claims; its mutable root path strings are rejection-test inputs, not reads.
The D-132 one-file exception was unnecessary. Evidence pins remain at history paths.

[Preflight](host-preflight.json) retains `vmstat 5 7` and sanitized
`pgrep -af 'gates.sh|mk/at-'` inventory. Six interval rows: runnable **1–7**,
idle **78–85%**, iowait **0–12%**; one other suite was active. Suite admission
was checked before both stacks and again after startup. The raw commands
remain locally retained. `go env GOCACHE` and the product baseline are recorded
in [RUN.md](RUN.md). These observations do not identify any failure's cause.

## Real D-087 transition and D-120

The exported TEST ONLY grant seam runs with the fixed 60-second fleet window
set before enrollment. Initial counted traffic precedes the 65-second invariant
stability wait. False → natural signed empty lease → actual HTTP 202
ready/passed receipt, forwarding false, exactly `awaiting_member_grants`,
`AWAITING_MEMBER_GRANTS` on lease/qos/source_binding and one lease-stage error
→ true → distinct accepted serving receipt joined to a new populated signed
lease → independently counted native forwarding.

Fixed generation **12** and all invariants remain byte-equal.
The [original-wire audit](member-grants-audit.json) verifies
**254** hashes and **33**
desired responses during the transition. True completed
**2026-10-31T21:50:01.907097266Z**, serving receipt
**2026-10-31T21:50:02.286280336Z**: **379.183070 ms**.
Native admission bytes **2631→5302**.
[Original wire](member-grants-wire.json), [transition](member-grants-transition.json)
and [node acceptance joins](member-grants-timing.json) retain the independent
server/report/node/counter samples. No synthetic observation, lease, forced
generation, projection stop, freeze, unpublish or transport hold drives D-087.

Historical true→serving samples remain: isolation **4,767 / 5,021 / 5,025 ms**,
devstack **3,980 / 4,112 ms**, first attempt **2,035 ms**, and follow-up 10
`bfcc3880` **962.448080 ms**. The second attempt has no qualifying transition.

D-120 **AT10 PASS under D-120**, unchanged: **101** verified Applies,
**43** datagrams, **1** retained UDP losses,
**0** TCP timeouts. Every eligible loss still
requires all original conditions, including exactly one guard, bounded install,
and next datagram success. [Original verdict](reload-stream/TestAT10_AT13_ReloadRevokeBaseline/d120-verdict.txt).
This tolerance never applies to the separate 80 ms renewal/expiry diagnostics.

## D-134 expiry window, counting and atomic guard

The conservative transition is **[D−101 ms, D]**: 100 ms installation reserve
plus 1 ms truncation. Original D is unchanged. The D-104 D+100 ms first-drop
observation bound never permits a positive at/after D. No new tolerance is
introduced. Require qualifying positives through the D−101 ms boundary and
continuous denial thereafter without a new valid authorization. Every pre-D
loss and every renewal loss stays counted; literal zero pre-D loss is not claimed.

New whole-node D-104 **pass**: D=`67713545711930` ns,
**161 positives / 41 drops**;
last positive **D-99.871612 ms**, first drop
**D-62.925070 ms**.
Pre-D ingress losses **1** remain counted;
literal zero-pre-D-loss assertion **false**.
[Packet verdict](deadline-verdict.json), [producer correlation](expiry-failure-correlation.json),
and [all loss brackets](expiry-timing-analysis.json) retain the evidence.
Historical follow-up 9 **96 early drops / 65 subsequent positives / 41 final
drops** remains failed; its initial baseline writer remains **unattributed**.
A fresh pass or restart adoption cannot resolve that event.

Renewal: **1396 probes / 1378 positives / 18 losses**,
**2** actual accepted renewals,
**70** completed guards. **18** lost
ingresses overlap completed guard brackets; **0** are outside.
All remain losses. These overlaps are capture/command correlations, not exact
kernel drop instants or causal attribution. [Continuity analysis](continuity-analysis.json).

The atomic outer guard is the measured mechanism. Its distributions are the
cost measured in this run; **52.596424 ms is a historical measurement, not a
fixed ceiling**. An install overrun never opens the guard. Baseline can replace
it atomically while member forwarding remains closed.

## Per-Apply host consumer and completed distributions

The [original stream](producer-operations.jsonl) has **7885** contiguous
producer records: baseline=131, heartbeat=3847, renewal=71, apply=952, live=2854, blocked_audit=30.
Every Apply's host samples and sanitized command status join by process run ID,
boot ID and sequence. Host contexts: **952 available / 0 unavailable**. Install overruns:
**0**. Full per-Apply contexts, brackets, command status and
failed records are in [producer-timing-analysis.json](producer-timing-analysis.json).

Nearest-rank **p50 / p90 / p99 / max**, milliseconds, completed guards only:

| Sample / population | n | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|---:|
| Follow-up 11 / Outer guard | 945 | 31.545345 | 41.568870 | 48.385975 | 56.774935 |
| Follow-up 11 / Install command | 945 | 16.520153 | 22.588465 | 28.098082 | 30.353789 |
| Follow-up 11 / Render→installed | 945 | 16.615574 | 22.743450 | 28.185461 | 30.431648 |
| Follow-up 10 first attempt / Outer guard | 678 | 31.634581 | 41.827776 | 51.362069 | 57.630034 |
| Follow-up 10 first attempt / Install command | 678 | 16.459780 | 22.524217 | 28.101933 | 30.792366 |
| Follow-up 10 first attempt / Render→installed | 678 | 16.539935 | 22.623136 | 28.161463 | 30.882274 |
| Follow-up 10 second attempt / Outer guard | 931 | 30.749663 | 38.828002 | 46.306456 | 49.844866 |
| Follow-up 10 second attempt / Install command | 931 | 15.779444 | 21.512728 | 26.901864 | 30.439422 |
| Follow-up 10 second attempt / Render→installed | 931 | 15.911599 | 21.591510 | 27.015158 | 30.531122 |
| Follow-up 10 bfcc3880 / Outer guard | 968 | 31.085118 | 40.528987 | 48.851468 | 52.927957 |
| Follow-up 10 bfcc3880 / Install command | 968 | 16.109632 | 22.319402 | 27.699284 | 30.619698 |
| Follow-up 10 bfcc3880 / Render→installed | 968 | 16.234853 | 22.432804 | 27.918874 | 30.691559 |

Completion does not itself imply later verification success. Failed operations
are retained separately; incomplete guards never enter these populations.
All **seven failed Applies** reject before install during the labelled foreign-tc
scenario; their host samples and missing verifier state remain retained.

- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2307**, Apply `67741524033260`–`67741568610198` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2315**, Apply `67742526653725`–`67742569501555` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2322**, Apply `67743514372144`–`67743564567025` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2329**, Apply `67744516491104`–`67744572440268` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2337**, Apply `67744716512501`–`67744761301657` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2344**, Apply `67745701467459`–`67745745477269` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.
- Process `6131959d-7ae2-438d-8f42-e404c030d279`, sequence **2351**, Apply `67746703740690`–`67746754507742` ns; stage `pre_install`, verification `unavailable`, host `available`, cause `unproven`, completed guard=false.

Missing verification is **unavailable**, never failed verification. A baseline
cause named `verification_HEALTH_CHECK_FAILED` cannot supply absent verifier
records. Missing command status means unknown, not exit zero. Missing or
invalid host samples never mean quiet. The conservative diagnostic rule and
its kernel-aggregate/cgroup limits are explicit in [RUN.md](RUN.md).
Even “fail-closed install overrun with recorded host contention” leaves
`cause_status=unproven`; coincident observations are not a scheduler cause.

The [new first-attempt replay](first-attempt-host-review.json) preserves all
**49** failed installs, each with host label **unavailable**, verification
**unavailable** and cause **unproven**. Apply 712's render→installed remains
**215.890796 ms**, population maximum **340.349614 ms**. All **279** renewal
losses remain: **29** overlapping completed guards, **250** outside. Original
[archive](history/follow-up-10-attempt-1/archive.json) and failure analysis are
unchanged. Agent-core 29's controlled interventions up to 384 workers at 0%
idle did not reproduce an install overrun (maximum 53.133468 ms); chain-gate
correlation is not cause. Its independent quiet/loaded populations remain in
[agent-core 29's assessment](../../../node/followup29-install-contention-2026-10-31.md).

## D-135 rollback and recovery

Failed Apply installs the **node-wide generation-0 baseline**. It preserves
revocation and does **not restore the previous generation**. Admitted siblings
also lose forwarding; recovery requires the next successful Apply and live
checks using current authorization. A completed guard bounds an operation,
not the entire outage, and the 100 ms install budget supplies no retry/outage
ceiling. This wording changes no predicate or historical measured row.

ISO-F6-01: **pass**, **6/6** assertions
(the full assertion outcomes are in the [recovery verdict](recovery-verdict.json)).
Assistance **false**; containment upper bound
**1157.112943 ms**, removal→link
**647.708049 ms**, removal→serving
**3420.111504 ms**, removal→counted completion
**12579.963359 ms**. These are observed upper bounds.
Rejected reports **16**, including **two outside the redirect injection window**.
Those two repeat a single `qdisc/read_failed` observation at
22:04:59.126691364 UTC, during the later identity-frozen phase, with zero
foreign objects recorded. [Exact producer/report join](post-freeze-tc-read.json)
retains process `95b1ada3-c652-4ce4-a274-87c27f4fbd00`, blocked-audit sequence 9,
its neighbors and both original reports. The read-error cause remains unproven;
no creator or host-contention cause is inferred. Both reports stay in the
original outside-window list; ISO-F6-01 completed earlier.
Original normal agent/watchdog log segments and baseline writer rows describe
only this run; the unretained historical foreign-tc creator remains unknown.

D-106: **639/639** canonical digests recompute;
**639** match producer reads. [D-093 reconciliation](d093-reconciliation.md)
keeps all 15 agreements (13 open, two retired only in simulator scope).

## Validation and cleanup

Exact command results and actual output tails: [validation-results.json](validation-results.json).
The assembled exit preserves blocked operator coverage. Final owned-prefix
[inventory](cleanup-inventory.json) shows **zero containers, images, networks
and volumes**. Owned lint/test/build, handover, root lint, formatting, diff and
committed-evidence hygiene pass. The diff check initially found terminal blank
lines in five text captures; only those trailing lines were normalized, with
original copies retained locally. D-120 replay is byte-identical afterward;
original producer JSONL, wire JSON and all archives are untouched. No push, PR or tag; all work remains on the assigned branch.
