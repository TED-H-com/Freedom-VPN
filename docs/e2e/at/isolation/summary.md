# Isolation follow-up 15 — 2026-11-01

One assembled run **`0b9e4510-776f-4c51-9e66-500404ce9f7e`**, runtime consumer
**`9031ebd4044059db25ea4ea7a397b87e6d6c6aa5`**, source dirty **false**.
Baseline **`aa7f306c008d8021b3921d083b70e19db7c8f865`**, through #388, includes
#383 and #386. Driver **1072.06 s**; full command **1210.552 s**, exit **2**.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0**, ISO-F6-01 **6 / 6** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0**, carried native-platform matrix |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0**, carried full fault matrix |

**D-140 qualification is not complete.** The unchanged D-104 evaluator passes
this packet trace, but the added schedule has one ingress/egress interleaving.
The raw case verdict and this separate qualification failure are both retained.
No retry run or loss waiver. Historical **isolation F14-01** in run 56d20530
stays failed and remains distinct from **members F14-01**.

Requirements: §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04,
ENT02, SEC01, DNS03. Central hold observations support §5.1–5.2 / AT29.

## D-098 and host context

First commit **`877ea4e3`** archived all **117** root files and subtrees from
run 56d20530, byte-identically against baseline Git blobs, under
[follow-up 14](history/follow-up-14/archive.json). SHA-256 pins include both gzip
forms. `history/follow-up-14-stream/` and all earlier archives remain untouched.
Handover claims already use immutable sources; root references are rejection
inputs, so no D-132 one-file exception edit was needed. A historical replay
link now points to that archive; current large JSON links point to lossless gzip.

[RUN.md](RUN.md) records the baseline and actual `go env GOCACHE` output.
[Preflight](host-preflight.json): six 5-second interval rows r=0–4,
id=83–90%; **not quiet**, because two samples have r=4. The since-boot first
row has id=79%. Load averages: **4.68 / 5.56 / 7.41**. One updates suite was
active at admission and after both stack startups; no two-other-suite wait was
needed (**0 s**), and no third suite was started. Host state is recorded, not a
criterion or causal attribution. [Cleanup](cleanup-inventory.json) is zero for
owned containers, images, networks and volumes.

## D-140 measured boundary and qualification failure

[Schedule/calibration](boundary-schedule.json),
[all 546 probe timelines and interleavings](boundary-review.json),
[unchanged evaluator](deadline-verdict.json), and
[retained qualification finding](follow-up-15-findings.json).

Before the boundary, **103** original-cadence packets measured member-send →
node ingress latency **0.103281–3.778979 ms**, spread **3.675698 ms**.
Maximum ingress→egress transit was **0.221923 ms**. The measured recipe selected
**0.6 ms** spacing: at least twice maximum transit plus 0.1 ms, rounded upward.
It added **344** probes, using median latency to predict ingress from before
D−101 ms through after D+100 ms, extended by spread plus three spacings.
The original **101 phased + 101 arbitrary_37ms** scheduled sends remain exact.
All planned added sends, actual sends, ingress, egress, latency and transit are
retained individually. Observation adds no node nft table/rule or `nftrace`.

The evaluator records **173 positives / 373 drops**, last positive
**D−98.630032 ms**, first drop **D−97.597715 ms**, no later positive and no
post-D positive. Its predicate and existing tests are byte-identical.

However, original phased packet **171** left at **D−98.651615 ms**, while
added packet **172** entered at **D−98.655436 ms**: **3.821 µs** of overlap.
Both were positive, before the first drop. Packet 173 then supplied the last
positive and packet 174 the first drop. This does not manufacture a later
positive after a drop, but it violates the requested no-interleaving condition;
the schedule is **unqualified**. Send spacing alone, even exceeding observed
node transit, did not cover arrival-latency variation.

The final source corrects the spacing formula to include measured latency
spread: `ceil_0.1ms(max(0.5ms, spread + 2*max_transit + 0.1ms))`. Applied to this
calibration it would choose **4.3 ms**, leaving a worst-case predicted ingress
gap of **0.624302 ms** within the observed latency range. That is an **offline
calculation**, not another run or an observed fix. Actual future captures still
must qualify; no bound on unobserved latency is assumed. Only the boundary
schedule changed, and the original sample stays unqualified.

Run 56d20530 remains failed: last positive D−162.220103 ms, first drop
D−97.638039 ms, closure straddling D−101 ms. Its renewal packets **15–18**
remain four counted losses outside guards in the **30→31** baseline bracket.

## D-102: four parent receipts, separate container outcomes

[Joined stops](stop-receipt-review.json), [ordinary restart](restart-lifecycle.json),
[clone fixture](identity-freeze.json), [terminal samples](receiver-closure.json),
and [container events](container-lifecycle-events.json).

| Original stop | Agent parent outcome | Observed interval |
|---|---|---:|
| Whole-container restart | exit **0**, no signal | container TERM→parent wait **201.515825 ms**; stop command **497.176989 ms** |
| Clone-source fixture stop | exit **0**, no signal | container TERM→parent wait **172.452077 ms** |
| Terminal source pidfd sample | **signal 15**, no exit code | signal→exit upper bound **71.529909 ms** |
| Terminal clone pidfd sample | exit **0**, no signal | signal→exit upper bound **6.304797 ms** |

All launch IDs, PIDs and start ticks match the pre-stop identities. Whole-node
receipts were copied with `docker cp`, including while the container was stopped.
The direct samples use the 15-second budget; whole-container/clone grace is
25 seconds. Ordinary and clone-source PID1 exit 0 is recorded separately from
the agent result. No escalation sidecar was retained; its presence would record
an attempt, not the outcome. The source's signal-15 result is **not** relabelled
cooperative because it was fast. The cause of signal termination is not proved.
All four durable-lease checks pass. Step cancellation and exact kernel exit
instants remain unavailable. Historical 10.087 s / 20.153 s exit-137 rows are
unchanged; follow-up 14's separate 20.053570446 s clone row also stays recorded.

## Core 34 reservation: receipt-to-revocation and recovery

[Full timelines](pre-apply-reservation-review.json) retain the
`host.before_apply.end_ns` → `timing.start_ns` bracket for all **950** Applies.
Original conntrack commands prove four nonempty batches:

| Producer sequence / generation | Pre-Apply bracket (ms) | Context |
|---|---:|---|
| 3804 / 18, first process | **1.511606** | populated policy-missing recovery; baseline→Apply **242.859786 ms** |
| 4188 / 25, first process | **0.988798** | established-flow revoke; baseline→Apply **244.170002 ms** |
| 1488 / 28, restarted process | **1.002519** | finite-member expiry; no intervening baseline since preceding successful Apply |
| 2527 / 33, restarted process | **1.117402** | foreign-TC recovery; baseline→Apply **2137.465146 ms**, failed Apply2519 retained |

These are reservation-plus-common-setup brackets, not isolated SQLite timings.
They precede the restricting install and are included in the full intervals.
The revoke's desired response completed at **15:01:50.758444418 UTC**;
producer anchors place first baseline completion at **+52.818181…52.818352 ms**,
Apply entry at **+165.927785…165.928184 ms**, install completion at
**+196.659208…196.659607 ms**, and Apply completion at
**+248.735465…248.735864 ms**. The initial baseline already restricts forwarding;
the later **0.988798 ms** reservation is part of recovery, not omitted work.
The request/response and established-stream failures remain in
[revocation-request.json](revocation-request.json); route withdrawal and kernel
revocation are separate observations, not assigned one causal instant.

The first two batches have later verified live and serving-heartbeat receipts.
Generation33 has live2528 after Apply2527; no serving heartbeat was observed in
that immediate baseline-free interval. The separate ISO-F6-01 final counted
recovery still has all six original assertions. Every baseline cause, failed
Apply and later recovery endpoint remains distinct. A desired receipt may
precede a finite member expiry by minutes; it is not necessarily the deletion
trigger. Proxy delivery completion is not an exact node parse timestamp.

The renewal packet window contains **69** successful Applies, and whole-node
expiry overlaps **one** successful Apply; neither has revoke commands or a
failed Apply. Those pure Steps add no reservation. Failed preparation elsewhere
cannot prove an empty pending queue, so its entire pre-Apply bracket remains.
Agent-core's mechanism benchmarks are not presented as an assembled pass.

## D-087, ISO-F6-01, renewal and completed guards

D-087 passes at fixed generation **14**: byte-equal invariants, **269** original
wire hashes, **32** desired responses, true→accepted serving **200 ms**, counted
bytes **2632→5262**. No forced generation, synthetic report or lease, projector
stop or transport fault supplies the seam result.

ISO-F6-01 passes **6/6**. Restored-control readiness **2.412609117 s**, three
dirty samples, unchanged 90-second/twelve-sample budgets. Accepted transactions
**1541/1542**, candidate32/ACK31, join desired1539 and failed Apply2519,
commands0–36; clean recovery generation33. Pre-install rejection is separate
from installed/live verifier evidence.

Renewal: **1396** probes, **1387** positives, **9 losses**, two actual renewals,
69 completed guards. All nine lost ingresses overlap an outer guard; all stay
counted. No baseline window in this diagnostic has captured ingress coverage;
unsampled intervals remain unavailable. The separate D-120 mechanism row passes
with **101** Applies and one permitted boundary loss (datagram44, next succeeds,
no TCP timeout). Mechanism D-104 also passes; neither substitutes for D-140.

Producer: **7879** contiguous records, **950** Applies, one pre-install failure,
zero observed install overruns. Per-Apply command contexts **950/950**;
**53520** commands, **53513** successful, seven conntrack exit-1 commands retained
inside successful Applies. D-106 recomputation and producer joins **634/634**.
The 949 install commands retain before/after `skip_reason=install_bound`;
resource samples are unavailable there. Exact child starts, hidden host
ancestors and cancellation callbacks remain unavailable.

Nearest-rank milliseconds, **949** completed guards, the next independent sample:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer | 31.412703 | 40.291449 | 48.674027 | 55.125778 |
| Install | 16.080628 | 22.054019 | 26.999839 | 30.329121 |
| Render→installed | 16.163952 | 22.167203 | 27.131280 | 30.411308 |

[Follow-up 14](history/follow-up-14/summary.md), 979 guards: outer
31.128482 / 40.689136 / 49.218688 / 52.588252; install
15.849010 / 21.898948 / 27.440421 / 31.241963; render→installed
15.953637 / 21.986037 / 27.534820 / 31.329338. The earlier 951-guard
[follow-up 13](history/follow-up-13/summary.md) also remains unchanged.
No A/B cause or historical ceiling is inferred.

## Central holds and remaining scope

[Central export](generation-holds-central.json.gz), [review](generation-holds-review.json)
and [operation history](publication-operation-history.json) retain **634**
accepted reports. No accepted hold report, sample, incident, escalation or
matching-end event was produced. All accepted report IDs are compared with the
lease history. Exact hold joining and classification are **not exercised**,
not passed vacuously. Six publication completions remain recorded.

All **21** policy-missing windows are retained; **17** are populated transitions,
with baseline→Apply completion **198.673869–278.193183 ms**. None has a reported
hold to join; none has ingress coverage in these packet streams. No incident or
packet denial is invented from a baseline. The single existing-seam attempt
found no accepted open hold, and was not repeated.

Populated hold: **not exercised — devstack 27's fixture pending**. Alarm,
escalation, active-stop retention, unrelated-failure precedence, local-counter
reset and a later distinct incident remain unexercised. AT11 and AT13 operator
matrices remain carried without renewed asks. F13-01 follows the unchanged
[netlab23 decision](../../../poc/G2-followup23-evidence/decision.json): typed
`duplicate_chain`, origin unproven, 0/3300 reproduced; failed reads never prove
frozen counters or closed forwarding.

[Requests](requests.md) contains two carried operator rows and one new D-140
qualification follow-up. [Validation](validation-results.json) records the real
gates; [immutable gzip pins](history/follow-up-15-stream/archive.json) preserve
both forms. No push, PR or tag.
