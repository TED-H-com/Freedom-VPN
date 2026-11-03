# Isolation follow-up 17 — 2026-11-03

One assembled run **`0ec14b39-467e-4a37-a239-1a33ef12eb71`**, consumer
**`de9682a96a296a3b79686db029b3e28f493f08f4`**, source dirty **false**.
Baseline **`b943941157ab6aeba6b04c79377de686c22bfbb4`**, including #399, netlab 26
#403 and agent-core 36 #401. Driver **1338.92 s**, full command **1504.063 s**,
exit **2**. No retry. [Runtime receipts](runtime.json).

| Case | Pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **9 / 0 / 0**, ISO-F6-01 **6/6** |
| [AT11](AT11.json) | **2 / 1 / 0**, carried native-platform matrix |
| [AT12](AT12.json) | **10 / 0 / 0** |
| [AT13](AT13.json) | **6 / 1 / 0**, carried full fault matrix |

**D-140 qualifies under D-142, and the unchanged evaluator passes.** The separate
shared **D-104 mechanism passes** on netlab 26's changed precision fixture.
The original seven hold coverage categories pass, and the third incident proves
central's **30-second fallback**. Pure `stale_generation` remains **not exercised**;
[F17-01](follow-up-17-findings.json) names the diagnostic/classification mismatch.
All **six** agent parent receipts record exit 0. No install exceeds 100 ms.

Requirements: §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02,
SEC01, DNS03. Central holds support §5.1–5.2 / AT29, without promoting its full matrix.

## D-098, host admission and publication

First separate commit **fbe2e789** archived all **129** root files/subtrees of
83b457ef byte-identically under [follow-up 16](history/follow-up-16/archive.json),
with SHA-256 pins including both gzip forms. All **1,378** earlier archive files,
including follow-up-16-stream, stay untouched. Handover claims already use
immutable paths, so no D-132 exception edit was needed.

[Host preflight](host-preflight.json): quiet after **60.040 s**, two `vmstat 5 7`
windows; final six current rows r=**0,1,0,1,1,0**, id=**100,95,95,90,92,95%**;
load **0.50 / 0.95 / 1.78**. The since-boot row was ignored. Before-run admission
found no conflicting suite (0.034 s check). A w1-netlab tagged test appeared during
lab startup; the post-start check waited **30.078 s** until it finished. Other
checks waited **0.036 / 0.035 / 0.041 s** with zero other roots. Shell command
strings were not counted as runs. [Admission receipts](suite-admission.jsonl).
Host state is context, not a pass criterion or causal explanation.

`go env GOCACHE` immediately before and after returned **`$HOME/.cache/go-build`**;
no cache override. TMPDIR/GOTMPDIR used disk `.agent/tmp`. [Owned Docker inventory](cleanup-inventory.json)
is zero for containers, images, networks and volumes. Twelve large streams are
lossless gzip with both-form pins at [immutable follow-up 17 streams](history/follow-up-17-stream/archive.json).
Five capture-text files have only terminal/line whitespace normalized; original
bytes remain in private run records. No historical archive is rewritten.

## D-140: calibrated schedule and deciding pair

[Predeclared procedure](RUN.md), [schedule](boundary-schedule.json),
[all timelines and qualification](boundary-review.json), [unchanged evaluator](deadline-verdict.json).
Exactly one sweep. The corrected formula and original 101+101 cadences are retained;
only prospective qualification/reporting uses D-142. No observer rule was added.

103 same-run calibration packets measured member-send→ingress min / median / max
**0.085664 / 1.017350 / 2.002521 ms**; spread **1.916857 ms**; maximum transit
**0.178905 ms**. Formula:
`ceil_0.1ms(max(0.5ms, 1.916857 + 2×0.178905 + 0.1ms)) = 2.4ms`.
The planner declared that spacing before the boundary, adding **83** dense probes
for **285** total. The original D−101 ms send predicted ingress **D−99.982650 ms**,
which can land in [D−101 ms, closure). Prediction is not a future latency bound.

**Every overlap anywhere in the sweep: zero.** No earlier positive-positive pair
needs a record-only exemption. D-142 judges all adjacent ingress-ordered pairs
from packet 166 (last positive) through sweep end; all qualify.

| Deciding packet | Scheduled offset ms | Send offset ms | Ingress offset ms | Egress offset ms | Send→ingress ms | Transit ms |
|---|---:|---:|---:|---:|---:|---:|
| 166, phased, positive | −101.000000 | −99.954394 | −98.977545 | −98.970225 | 0.976849 | 0.007320 |
| 167, dense, drop | −96.217350 | −96.185352 | −95.769511 | absent | 0.415841 | unavailable |

Offsets are relative to the signed deadline D. Positive egress precedes first-drop
ingress by **3.200714 ms**. Capture-read times bound userspace observations; they
are not exact kernel packet instants. The closure interval lies wholly after
D−101 ms. Evaluator: **166 positives / 119 drops**, no positive after the first
drop or at/after D. All **38** literal pre-D losses remain counted.

Isolation 15, isolation 16 and isolation F14-01 remain failed as originally judged.
No old sample was reevaluated under D-142. `EvaluateDeadline` and its tests remain
byte-identical to the baseline.

## D-104 and hold rows

Netlab 26's read-only precision fixture produced a fresh
[mechanism trace](reload-stream/TestD104_AT13_DeadlineContinuity/trace.json):
**26 positives / 7 drops**, last positive **D−100.807007 ms**, first drop
**D−0.903687 ms**; Apply→installed ≤**27.930007 ms**, render→installed
≤**15.740598 ms**, precise read **1.448963 ms**. No failure or retry.
83b457ef's F16-01 stays failed; this new pass does not reclassify it.

[Full hold rows and IDs](populated-holds.md) retain all three original arms,
node/incarnation/report/own-lease/sample/incident/audit joins, first/last times,
matching recovery and separate counted recovery. First incident **12→13**:
alarm audit at **1.911284 s**, unrelated `ISOLATION_POLICY_INVALID` keeps
`check_failed`, same incident and audit across agent stop, local counters
**7 Steps / 5936 ms → 1 Step / 0 ms**. Second **13→14** is a distinct incident
after recovery, alarm audit at **1.880655 s**. All seven prior coverage categories
pass across those two incidents.

Third **14→15** stops after the accepted Step-1 report while the shared proxy
stays armed. Central audit **01a1a013-7057-7843-b023-47ac112e60c8** occurs at
incident age **30.769780 s**, `reported_alarm=false`, `alarm_report_id=null`,
Steps **1**, elapsed **4 ms**, before restart. Incident
**01a1a012-f825-7e4a-ab4c-268766baa273** ends `generation_matched` at
**06:23:15.458937Z**. Counted recovery is separate. This proves central age,
not another alarm trigger or a local counter decrease.

All **806** accepted central report IDs match lease-history receipts. **16** hold
reports across **five** incidents join correctly; two short unarmed incidents
remain separate. All 16 applied generations match delivered desired, 15 match
the later central read; reverse-array classification agrees. Six publication
completions all succeeded outside open incidents and credited no hold report.

Pure `stale_generation` remains not exercised: `attachServingRead` replaces the
failed QoS detail with `read_at=...`, whereas `generationHoldSafetyFailure`
exempts only `Rate policy could not be applied.`. Classification is **12 check_failed / 4 isolation_unconfirmed**; the four
post-restart samples have `safety_status=expired` even with the expected dependent
QoS text. Neither check nor safety precedence is bypassed. Devstack's seam cannot edit that field. The owner request asks
for a genuine supported pure row, preserving unrelated-failure precedence.
The general review is **partial**; the nine-category populated review is
**incomplete** solely for this missing row, with zero join problems.

## Stop receipts and retained diagnostics

Agent-core 36's signal handling is present. [Six parent receipts](stop-receipt-review.json)
match their launch IDs/PIDs/start ticks, with exit **0**, no signal or escalation:

| Stop | Observed interval ms |
|---|---:|
| Whole-container TERM→parent wait | 232.654411 |
| Clone-source TERM→parent wait | 304.266772 |
| Terminal source signal→exit upper bound | 11.597128 |
| Terminal clone signal→exit upper bound | 6.310802 |
| First populated hold signal→exit upper bound | 7.986207 |
| Central-fallback hold signal→exit upper bound | 6.013990 |

Container PID1 outcomes remain separate. Historical signal-15 and exit-137
receipts stay unchanged. These samples do not establish exact kernel exit
instants or the Step cancellation instant, or close the full AT13 fault matrix.

D-087: generation **12**, **262** original wire hashes / **32** desired responses,
**924 ms** to accepted serving, counted **2632→5304 bytes**. ISO-F6-01 **6/6**:
readiness **6.733178628 s**, accepted candidate **34** / ACK **33**, transaction
**1776**, desired transaction **1774**, pre-install failed Apply **2536**; clean
recovery generation **35**. Budgets remain 90 seconds / 12 dirty samples.

Renewal: **1396 probes**, **1369 positives / 27 losses**, two renewals and
**70 completed guards**. All 27 lost ingresses overlap outer guards; none waived.
Whole-node expiry has **zero Applies**. The D-120 mechanism passes with one
retained loss, datagram **26**, classified boundary with upper bracket
**27.445952 ms**, render→installed **15.583008 ms**, next datagram successful.

Producer: **9433** contiguous records, **1112 Applies**, **1111 completed guards**,
D-106 **807/807** digest matches, **62247** command receipts (**62240** successful).
Two failed Applies remain: **2527** fails verification after an in-bound installed
attempt; **2536** fails before install. **Zero install overruns**, maximum install
**65.869559 ms** against 100 ms. No cause is claimed for historical overruns.

Nearest-rank completed-guard timing, milliseconds:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer guard | 30.876917 | 42.555975 | 65.433700 | 95.005112 |
| Install | 16.273649 | 23.093791 | 38.754088 | 65.869559 |
| Render→installed | 16.358400 | 23.166781 | 39.274548 | 66.200788 |

[Reservation review](pre-apply-reservation-review.json): nonempty pre-Apply brackets
for sequence/generation **2485/19**, **2878/27**, **1496/30**, **2544/35** are
**1.161693 / 1.194054 / 0.968853 / 0.881909 ms**. Full baseline→Apply recovery
intervals are **224.843570 / 1103.923097 / no intervening baseline / 1993.301146 ms**.
These brackets include common setup, not isolated SQLite time; failed preparation
cannot prove an empty queue. All **24** policy-missing windows remain, **19**
populated, baseline→Apply **132.544354–6073.457750 ms**. Renewal/expiry packets do
not cover those windows; independent denial is limited to the three seam probes.

## Nightly diagnostics and verification

[Nightly investigation](nightly-diagnosis.md) distinguishes the two historical
tracebacks and the unproven short-expiry block. Both logs omit the actual reason;
both have zero artifacts. It is not established as an expected GitHub capability
limit and stays off the list. Only the two operator rows are expected blocked.
Scheduled-only D-144 mapping preserves acceptance status, reports unblocked rows,
and keeps any unlisted block/fail/error red. The unresolved pure-hold capability
also keeps its review nonzero; it is not silently allowlisted. The always-run
records-only artifact retains seven days. No workflow was triggered.

[Consumer corrections](consumer-review-corrections.json) preserve the initial
offline hold-review failure and the runtime `capture_evaluator=unavailable`
metadata caused by a fixed-depth path assumption. The final consumer uses the
module root. Raw records are unchanged; predicate/parser source bytes are checked
against baseline and measured commit. No live retry followed either correction.
[Validation](validation-results.json) records gates and retained nonzero results.
[Requests](requests.md): two carried operator rows and the pure-classification
owner item. No shared seam/fixture or other outside-owned path was edited.
