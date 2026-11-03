# Isolation follow-up 16 — 2026-11-01

One assembled run **`83b457ef-ea16-490b-9923-1fea0df86b36`**, runtime consumer
**`a8394027002771be64f023d062ea47b2da1447d8`**, source dirty **false**.
Baseline **`9efe9773e67d35839649e9f21e1c3a9b680633b2`**, including #392 and
#394. Driver **1272.20 s**; full command **1406.378 s**, exit **2**. No retry.

| Case | Result: pass / blocked / fail |
|---|---|
| [AT10](AT10.json) | **pass: 9 / 0 / 0**, ISO-F6-01 **6 / 6** |
| [AT11](AT11.json) | **blocked: 2 / 1 / 0**, carried native-platform matrix |
| [AT12](AT12.json) | **pass: 10 / 0 / 0** |
| [AT13](AT13.json) | **blocked: 6 / 1 / 0**, carried full fault matrix |

**D-140 remains unqualified:** one positive transit overlaps the next ingress
by **3.609 µs**. The unchanged evaluator separately passes this trace.
**F16-01** retains a separate shared D-104 fixture assertion before that
mechanism's packet window. **F16-02** retains the D-140 qualification failure.
The two populated seam incidents together pass all **seven** requested coverage
categories. Missing per-incident rows remain explicitly not exercised.
[Findings](follow-up-16-findings.json) retain both failures. Historical isolation
F14-01 and follow-up 15's failed qualification stay failed, with no loss waiver.

Requirements: §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04,
ENT02, SEC01, DNS03. Central hold observations support §5.1–5.2 / AT29.

## D-098, admission and publication

First commit **`3005c383`** archived all **123** root files and subtrees of
run 0b9e4510 byte-identically against baseline Git blobs under
[follow-up 15](history/follow-up-15/archive.json). SHA-256 pins include both gzip
forms. Follow-up-15-stream and every earlier archive remain untouched.
Handover claims already read immutable snapshots; living-root strings are
rejection inputs, so no D-132 exception edit was needed. The inherited D-104
trace was removed from the living tree because no fresh trace was emitted;
its original bytes remain in the archive. The mechanism summary's inherited
prefix was likewise removed, retaining this run's one appended row.

[RUN.md](RUN.md) records baseline and the final queried GOCACHE. The final cache
read differs from the pre-run note; no separate runtime cache receipt was
retained, so the assembled cache path is unproven. [Host preflight](host-preflight.json)
retains **52** vmstat windows. Admission became quiet after **1561.452 s**:
six current rows r = **3,3,1,0,1,0**, id = **87,87,86,87,87,87%**;
load averages **8.66 / 11.78 / 12.50**. The since-boot row was ignored.
All four suite admission/startup checks saw zero other suite roots; waits were
**0.084 / 0.088 / 0.092 / 0.129 s**, with no two-other-suite wait. Host state
is context, not a pass criterion or an explanation of later latency.
[Owned inventory](cleanup-inventory.json) is zero for containers, images,
networks and volumes. Large JSON is lossless gzip; both-form pins name only
[immutable follow-up 16 streams](history/follow-up-16-stream/archive.json).

## D-140: predeclared trade-off and failed qualification

[Schedule](boundary-schedule.json), [all 292 timelines](boundary-review.json),
and [unchanged evaluator](deadline-verdict.json) retain the only sweep.
Before running, [RUN.md](RUN.md) declared that one sweep cannot guarantee both
noninterleaving and a deciding probe in the roughly 3 ms historical positive
window. The driver has one whole-node boundary. We did not add another
lifecycle, merge sweeps, alter the evaluator, or retry.

Same-run calibration used **103** packets. Member-send→ingress minimum / median /
maximum was **0.141740 / 0.722673 / 1.840658 ms**; spread **1.698918 ms**;
maximum node transit **0.181849 ms**. The unchanged corrected formula
`ceil_0.1ms(max(0.5ms, spread + 2*max_transit + 0.1ms))` selected **2.2 ms**.
The added grid is anchored at predicted ingress D−100 ms, excludes sends too
near either unchanged original cadence, and added **90** dense probes to
**101 phased + 101 arbitrary_37ms**. No node observer rule was added.

The original D−101 ms send predicted ingress at **D−100.277327 ms**, inside the
historical positive window. Actual packet 165 entered at **D−99.821353 ms** and
left at **D−99.796618 ms**. A deciding positive was present. The evaluator
records **165 positives / 127 drops**, first drop **D−97.671468 ms**, no later
positive and no post-D positive. Its predicate and tests are byte-identical.
All **43** drops before the literal deadline remain counted; a tolerance pass
is not a zero-pre-deadline-loss claim.

Qualification nevertheless fails: dense packet 162 left at **D−106.286141 ms**,
while dense packet 163 entered at **D−106.289750 ms**. The **3.609 µs** overlap
violates D-140. Packet162's latency **3.150529 ms** exceeded calibration maximum
by **1.309871 ms**; packet 163's was **0.987690 ms**. Their actual send gap was
**2.178277 ms**, ingress gap **15.438 µs**, and packet 162 transit **19.047 µs**.
The formula was applied correctly but calibration did not bound future latency.
The network/host cause is unproven. Capture-read timestamps bound userspace
observations, not exact kernel packet instants. No post-run schedule change,
waiver or qualification promotion was made.

Run0b9e4510's **3.821 µs** overlap remains failed. Run56d20530 isolation F14-01
remains failed (last positive D−162.220103 ms, first drop D−97.638039 ms);
its four renewal losses15–18 outside guards in the 30→31 bracket stay counted.
These are distinct from members F14-01.

## Populated stale-lease incidents

[Full row/ID joins](populated-holds.md), [review](populated-holds-review.json),
[original seam attempts](populated-holds.json.gz), [central originals](generation-holds-central.json.gz),
and [general hold review](generation-holds-review.json) carry the evidence.
Both incidents began with **two leased members**, fresh arm UUIDs, a genuine
capacity PATCH/read-back and the unchanged devstack 27 TEST ONLY proxy.
The owned observer forwards through that proxy. Signed bodies and reports were
neither edited nor replayed; no seam implementation was changed.

The **15→16** incident produced six accepted hold reports. Step 3 alarm elapsed
**1905 ms**, and its single audit occurred at incident age **1.899726 s**,
triggered by the reported alarm, not the 30-second fallback. Temporarily moving
the conntrack executable inside the owned container produced a same-incident
`ISOLATION_POLICY_INVALID` apply failure; central retained **check_failed**
precedence over stale_generation. After restoring the executable, the supervised
agent-only stop kept the proxy armed and the incident open. Local counters reset
from **7 Steps / 5917 ms** to **1 Step / 0 ms**, with unchanged first_seen.

Release forwarded a fresh central-issued matching lease whose body digest
matches the local read. The accepted generation 16 report ended the incident
`generation_matched` at **16:59:18.323291Z**. That first matching report still
had **forwarding=false**; it is not claimed as serving recovery. The separate
later counted probe proves **7935→10566** bytes under a later genuine lease.

Only after recovery, fresh arm **bd5cfd83-dc18-43c9-be15-e8fd551ef788** opened
**16→17**, a distinct incident with four accepted hold reports. Step 3 alarm
elapsed **1874 ms**; its one audit was at age **1.868719 s**, again alarm-triggered.
Release/local digest/accepted matching report ended it at **17:00:00.461554Z**;
counted recovery was **13199→15832** bytes. Independent probes in both holds
captured ingress with no egress and timeout. Unrelated failure and active stop
were deliberately exercised only in the first incident, not inferred in the second.

All **784** accepted central report IDs exactly match lease-history receipts.
The complete central export has **11** accepted hold reports across **three**
incidents: the two explicit incidents plus a later brief unarmed incident
`01a19813-2d0a-7ac4-b657-4a3c4917e181` (first/last **17:05:55.977801Z**,
ended **17:05:56.831437Z**). The latter's one report had no valid alarm summary,
no escalation, and is not substituted for either seam incident. All 11 join
their own report, signed lease, sample and incident; all 11 applied generations
match the latest delivered desired document (10 match the later central read).
Reverse-array classification replay agrees. Seven publication completions are
retained, six succeeded and one superseded; none credited a hold report or
completed successfully during an open incident.

Pure `stale_generation` classification, central's 30-second fallback, kernel
reboot and the original opportunistic control-cut hold are **not exercised**.
The general review remains partial for these gaps; its populated retention
field now explicitly distinguishes the successful seam stop from the absent
opportunistic stop. All **24** policy-missing windows remain, **19** populated,
baseline→Apply **202.627285–6082.503527 ms**. The separate renewal/expiry streams
do not sample those windows; denial evidence comes from the two incident probes.

## Five parent stop receipts

[Joined receipt review](stop-receipt-review.json) retains launch IDs, PIDs,
start ticks, parent outcomes and separate container lifecycle observations.

| Stop | Agent parent outcome | Observed interval |
|---|---|---:|
| Whole-container restart | exit 0, no signal | TERM→parent wait **101.420032 ms** |
| Clone-source fixture | exit 0, no signal | TERM→parent wait **110.234416 ms** |
| Terminal source pidfd | exit 0, no signal | signal→exit upper **43.040857 ms** |
| Terminal clone pidfd | exit 0, no signal | signal→exit upper **6.790853 ms** |
| Populated-hold agent-only | exit 0, no signal | signal→exit upper **5.798490 ms** |

All five parent identities match; no escalation attempt was retained. Original
ordinary/clone PID1 outcomes are separately exit 0. The terminal source and clone
retain durable-lease survival checks; original restart/clone durable-lease survival checks also pass. Direct budget 15 s and container grace 25 s are unchanged.
Step cancellation at these stops and exact kernel exit instants remain unknown.
Follow-up15's signal 15 terminal-source receipt stays as observed, cause unproven
and assigned to agent-core 36; these new exit 0 receipts do not reclassify it.
Historical exit 137 and slow-stop receipts remain unchanged.

## Core34 reservation, recovery and original rows

[Reservation timelines](pre-apply-reservation-review.json) retain every
`host.before_apply.end_ns`→Apply entry bracket, including failures.
Four original command batches prove pending conntrack reservations:

| Producer sequence / generation | Pre-Apply ms | Baseline→Apply completion ms |
|---|---:|---:|
| 2796 / 21, post-hold-restart process | **0.973747** | **250.245008** |
| 3191 / 29, established-flow revoke | **0.969850** | **951.648072** |
| 1487 / 32, ordinary restarted process, finite-member expiry | **0.987099** | no intervening baseline since preceding successful Apply |
| 2578 / 37, foreign-TC recovery | **1.178520** | **2065.514125**, failed Applies2561/2570 retained |

These are reservation-plus-common-setup brackets, not isolated SQLite timings.
All remain inside full recovery intervals. The revoke's desired response finished
**17:05:56.649067108Z**; Apply entry followed by **40.591652–40.591978 ms**,
restricting install completion **76.086355–76.086681 ms**, and Apply completion
**125.897038–125.897364 ms**. Its preceding baseline already restricted forwarding;
no later baseline followed this receipt. The broader baseline→Apply interval and
[original revocation request/stream](revocation-request.json) remain separate.
No route-withdrawal instant is substituted for kernel revocation.

The first three batches have later verified live and serving heartbeat receipts.
Generation37 has live 2579, but no serving heartbeat in the immediate baseline-free
interval; ISO-F6-01's separate counted recovery passes all six assertions.
The renewal window has **70** successful Applies, no revoke command and no failed
Apply; whole-node expiry has **zero** Applies. A pure Step adds no reservation.
Failed preparation elsewhere cannot prove an empty queue. Desired delivery may
precede a finite-member expiry by minutes and is not its deletion trigger.

D-087 passes fixed generation **15**, byte-equal invariants, **281** original wire
hashes, **33** desired responses, true→accepted serving **716 ms**, counted
**2632→5304** bytes. No synthetic report/lease or forced generation supplies it.
ISO-F6-01 passes **6/6**; readiness **8.953790903 s**, five samples, unchanged
90-second/twelve-sample budgets. Accepted transactions **1710/1711**, candidate 36 /
ACK 35, join desired 1708 and pre-install failed Apply 2570 (commands 0–36).
Clean generation 37 recovery remains separate from pre-install rejection.

Renewal: **1396** probes, **1370** positives, **26** losses, two renewals,
**70** completed guards. Every lost ingress overlaps an outer guard; all losses
stay counted. No unresolved attribution in this diagnostic. The separate D-120
mechanism passes with **no UDP losses**. Shared D-104 fails its precision fixture:
requested 1001…1008 ms produced remaining 699,700,701,702,703,704,706,707 ms;
the sixth adjacent difference is2 ms. The assertion runs before sockets,
deadline Apply or trace emission. Cause unproven; no product trace is invented.
[Original failure](mechanism-d104-failure.txt) goes to w1-netlab, outside owned paths.

## Producer timing and diagnostic install failure

**9164** contiguous producer records; **1082** Applies, **1079** completed guards,
two foreign-TC pre-install failures and one install timeout. D-106 has **786/786**
recomputed and matched producer digests. **60413** child command receipts,
**60405** successful; all have exit/signal/CPU/switch evidence. Resource samples
are omitted before 1080 installs and after 1079 bounded installs; the failed
install's after-sample says context_deadline_exceeded. One cancellation callback
is observed. Exact child starts and hidden host ancestors remain unavailable.

Producer run `956a6be4-5690-409f-8021-1656c7be624b`, Apply 1296 / generation 15,
has **120.466876 ms** install duration against the 100 ms bound. It records
install_ok=false and guard_released=false, no release command, nft context
**deadline_exceeded**, cancel observation135850900739458 ns, wait return
135850920959758 ns, child **signal 9**. Successful apply_rollback baseline1297
and safety baselines1298/1299 follow. Live1300 fails and heartbeat 1301 is blocked;
Apply 1302, live 1303 and serving heartbeat 1304 recover. This failure remains in
all-command timing; it is excluded only from the explicitly completed-guard
population. Delay cause is unproven; no host blame or fixed historical ceiling.

Nearest-rank milliseconds for **1079 completed guards**:

| Measure | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Outer | 32.859847 | 41.426615 | 48.262200 | 54.718584 |
| Install | 16.887700 | 22.630281 | 28.030774 | 36.828926 |
| Render→installed | 16.982410 | 22.733032 | 28.116746 | 36.924154 |

[Follow-up15](history/follow-up-15/summary.md), 949 guards: outer
31.412703 / 40.291449 / 48.674027 / 55.125778; install
16.080628 / 22.054019 / 26.999839 / 30.329121; render→installed
16.163952 / 22.167203 / 27.131280 / 30.411308. Follow-ups14/13 remain unchanged.
These are independent samples, not an A/B causal comparison.

AT11 native platforms and AT13's full fault matrix remain carried operator rows.
F13-01 retains netlab23's typed duplicate_chain decision and unproven origin;
no failed counter read proves frozen counters or closed forwarding.
[Requests](requests.md) contains two carried assignments, continued D-140 sampling,
and the shared D-104 fixture request. [Validation](validation-results.json) records
actual gates and retained failures. No push, PR or tag.
