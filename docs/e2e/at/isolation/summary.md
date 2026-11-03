# Isolation follow-up 18 — 2026-11-03

One assembled run **`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`**, consumer
**`c248db966d6870182a7c87cf7e410d7524bf56ff`**, source dirty **false**.
Baseline **`090bb97b88b4062073a487c82676d0e1faca4fac`**. Driver **1316.05 s**,
full command **1477 s**, exit **2**. No retry. [Runtime receipts](runtime.json).

| Case | Pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **9 / 0 / 0**, ISO-F6-01 **6/6** (`foreign ingress redirect containment and D-096 recovery: pass`) |
| [AT11](AT11.json) | **2 / 1 / 0**, carried native-platform matrix |
| [AT12](AT12.json) | **10 / 0 / 0** |
| [AT13](AT13.json) | **6 / 1 / 0**, carried full fault matrix (`short whole-node lease expiry: pass`) |

**Both unlisted blocked rows from scheduled run `41640196805` are diagnosed and
fixed in the owned harness (`Outcome (a)`), and both pass in this assembled run.**
See [dated nightly diagnosis](nightly-diagnosis-2026-11-03.md).

**D-140 qualifies under D-142, and the unchanged evaluator passes.** The separate
shared **D-104 mechanism passes** on netlab 26's precision fixture.
The original seven hold coverage categories pass, and the third incident proves
central's **30-second fallback**. Pure `stale_generation` remains **not exercised**;
[F17-01](follow-up-18-findings.json) names the diagnostic/classification mismatch.
All **six** agent parent receipts record exit 0. No install exceeds 100 ms.

Requirements: §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02,
SEC01, DNS03. Central holds support §5.1–5.2 / AT29, without promoting its full matrix.

## D-098, host admission and publication

First commit **`c248db966d6870182a7c87cf7e410d7524bf56ff`** archived all **138**
root files/subtrees of `0ec14b39-467e-4a37-a239-1a33ef12eb71` byte-identically
under [follow-up 17](history/follow-up-17/archive.json), with SHA-256 pins. All
earlier archive files, including `follow-up-17-stream`, stay untouched. Handover
claims already use immutable paths, so no D-132 exception edit was needed.

[Admission receipts](suite-admission.jsonl): `bash "$FVPN_HOLD_SH"` was `free`
before and after the run, and all five suite admission checks waited
**0.044 / 0.044 / 0.036 / 0.047 / 0.039 s** with zero other suite roots.
`go env GOCACHE` before and after returned **`$HOME/.cache/go-build`**.
[Owned Docker inventory](cleanup-inventory.json) is zero for containers, images,
networks and volumes. Twelve large streams are lossless gzip with both-form pins
at [immutable follow-up 18 streams](history/follow-up-18-stream/archive.json).

## D-140: calibrated schedule and deciding pair

[Predeclared procedure](RUN.md), [schedule](boundary-schedule.json),
[all timelines and qualification](boundary-review.json), [unchanged evaluator](deadline-verdict.json).
Exactly one sweep.

103 same-run calibration packets measured member-send→ingress min / median / max
**0.226815 / 0.867193 / 2.204540 ms**; spread **1.977725 ms**; maximum transit
**1.850952 ms**. Formula:
`ceil_0.1ms(max(0.5ms, 1.977725 + 2×1.850952 + 0.1ms)) = 5.8ms`.
The planner declared that spacing before the boundary, adding **32** dense probes
for **234** total.

**Every overlap anywhere in the sweep: zero.** D-142 judges all adjacent
ingress-ordered pairs from packet 164 (last positive) through sweep end; all qualify.

| Deciding packet | Scheduled offset ms | Send offset ms | Ingress offset ms | Egress offset ms | Send→ingress ms | Transit ms |
|---|---:|---:|---:|---:|---:|---:|
| 164, phased, positive | −101.000000 | −100.920401 | −100.564152 | −100.551358 | 0.356249 | 0.012794 |
| 165, dense, drop | −95.067193 | −95.046463 | −93.753451 | absent | 1.293012 | unavailable |

Offsets are relative to the signed deadline D. Positive egress precedes first-drop
ingress by **6.797907 ms**. Evaluator: **164 positives / 70 drops**, no positive
after the first drop or at/after D. All **15** literal pre-D losses remain counted.

## D-104 and hold rows

Netlab 26's read-only precision fixture produced a fresh
[mechanism trace](reload-stream/TestD104_AT13_DeadlineContinuity/trace.json):
**26 positives / 7 drops**, last positive **D−100.526854 ms**, first drop
**D−0.760365 ms**; Apply→installed ≤**24.993517 ms**, render→installed
≤**11.718073 ms**, precise read **1.494143 ms**.

[Full hold rows and IDs](populated-holds.md) retain all three original arms,
node/incarnation/report/own-lease/sample/incident/audit joins, first/last times,
matching recovery and separate counted recovery. First incident **10→11**:
alarm audit at **1.900313 s**, unrelated `ISOLATION_POLICY_INVALID` keeps
`check_failed`, same incident and audit across agent stop, local counters
**7 Steps / 5910 ms → 1 Step / 0 ms**. Second **11→12** is a distinct incident
after recovery, alarm audit at **1.946133 s**. Third **12→13** stops after the
accepted Step-1 report while the shared proxy stays armed; central audit
`01a1a1c0-9eb9-7df0-86e9-35b1c5d6f120` occurs at incident age **30.579901 s**,
`reported_alarm=false`, `alarm_report_id=null`, Steps **1**, elapsed **4 ms**,
before restart.

All **791** accepted central report IDs match lease-history receipts. **13** hold
reports across **four** incidents join correctly. All 13 applied generations
match delivered desired and the later central read; reverse-array classification
agrees. Five publication completions all succeeded outside open incidents and
credited no hold report. Pure `stale_generation` remains not exercised
(**10 check_failed / 3 isolation_unconfirmed**; carried [F17-01](follow-up-18-findings.json)).

## Stop receipts and retained diagnostics

[Six parent receipts](stop-receipt-review.json) match their launch IDs/PIDs/start
ticks, with exit **0**, no signal or escalation:

| Stop | Observed interval ms |
|---|---:|
| Whole-container TERM→parent wait | 294.574226 |
| Clone-source TERM→parent wait | 233.015384 |
| Terminal source signal→exit upper bound | 39.507101 |
| Terminal clone signal→exit upper bound | 5.826460 |
| First populated hold signal→exit upper bound | 7.750413 |
| Central-fallback hold signal→exit upper bound | 7.164891 |

D-087: generation **10**, **301** original wire hashes, counted
**2671→5344 bytes**. ISO-F6-01 **6/6**: readiness **7.696240747 s**, initial
generation **31**, durable apply generation **33**, pre-install failed Apply
**2561**; clean recovery verified.

Renewal: **1396 probes**, **1391 positives / 5 losses**, two renewals and
**69 completed guards**. All 5 lost ingresses overlap outer guards; none waived.
Whole-node expiry has **zero Applies**.

Producer: **9460** contiguous records, **1116 Applies**, **1115 completed guards**,
D-106 **792/792** digest matches, **62528** command receipts (**62521** successful).
One failed Apply (**2561**, `pre_install`). **Zero install overruns**, maximum
install **52.011013 ms** against 100 ms.

Nearest-rank completed-guard timing, milliseconds:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer guard | 30.855667 | 42.183226 | 61.889484 | 92.729936 |
| Install | 16.500002 | 23.722198 | 31.925108 | 52.011013 |
| Render→installed | 16.609113 | 23.819775 | 31.993765 | 52.127138 |

[Reservation review](pre-apply-reservation-review.json): nonempty pre-Apply brackets
for sequence/generation **2488/17**, **2878/24**, **1488/28**, **2569/33** are
**1.257233 / 0.899646 / 1.078966 / 1.131673 ms**. Full baseline→Apply recovery
intervals are **213.340536 / 213.023237 / no intervening baseline / 2123.628292 ms**.
All **25** policy-missing windows remain, **23** populated, baseline→Apply
**122.819935–6085.452005 ms**.
