> [!IMPORTANT]
> **Follow-up 27 live run status — NEWEST ATTEMPT, FAILED (`d69f5652-aed6-4fdd-9e7c-edface751237`):**
> The single permitted Follow-up 27 assembled invocation (`run_id` `d69f5652-aed6-4fdd-9e7c-edface751237`, `2026-11-05T10:11:41.296179Z`–`2026-11-05T10:33:52.446700Z`, `1331.15` s, `source_commit: 37b3c25f60194e0d2261cf9aca30dfe8fc5d93be`, `baseline/launch_main: aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`, `source_dirty: false`, `invocations: 1`, `reruns: 0`, `lab_included: true`, `confounded: false`) **failed** (`make` exit `2`; [`scheduled-job.json`](history/follow-up-27-run/scheduled-job.json) status `fail` with **26 pass / 2 operator-blocked / 1 fail** on `AT13` `short whole-node lease expiry` (`packet 145 missing send/ingress/read` after `result: "socket_error"`); auxiliary [`clone-transport-review.json`](history/follow-up-27-run/clone-transport-review.json) status `not established` with 8/9 criteria `established` and `consumer_latch_held` `not established`).
>
> **Root-record decision:** Canonical root `AT10`–`AT13` and `support`/`reload-stream` payloads in `docs/e2e/at/isolation/` remain byte-identical to pre-run `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` and retain the last passing full run (**Follow-up 25**, `f25`, `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`, backed by [`history/follow-up-25/archive.json`](history/follow-up-25/archive.json)). Every Follow-up 27 run artifact is sealed under [`history/follow-up-27-run/`](history/follow-up-27-run/archive.json); see [`## Follow-up 27 — 2026-11-05`](#follow-up-27--2026-11-05-newest-attempt-failed) below and [`followup27-2026-11-05.md`](followup27-2026-11-05.md).

# Isolation follow-up 19 — 2026-11-03

**All four D-147 mechanism rows and the unchanged lab AT10–AT13 pass. The single assembled run is blocked at serving setup.** These are separate results; no blocked acceptance row is promoted.

The focused D-147 run is `1d746f9a-e2df-4baf-af8d-f2d3074c8cbf` on `3e12f913`; the assembled invocation is `70beb28e-4766-4004-932b-510fd01f986a` on `458b7a89f07c22dd4e66f839195b6549db4b883d`, source dirty **false**. Baseline main is `27ff63928222c9642e4f5e180cf225108327a1e5` including netlab 34 / #432. [Dated D-147 evidence](d147-related-frag-needed-2026-11-03.md), [run receipt](follow-up-19-run-receipt.json), [findings](follow-up-19-findings.json), [requests](requests.md).

## Observed assembled rows

| Record | Pass / blocked / fail | Overall |
| --- | --- | --- |
| [AT10](AT10.json) | 0 / 2 / 0 | blocked |
| [AT11](AT11.json) | 0 / 2 / 0 | blocked |
| [AT12](AT12.json) | 0 / 3 / 0 | blocked |
| [AT13](AT13.json) | 0 / 5 / 0 | blocked |

| Record | Row | Observed |
| --- | --- | --- |
| AT10 | serving setup | blocked; positive control false |
| AT10 | provisional lease reports AWAITING_MEMBER_GRANTS | blocked; positive control false |
| AT11 | serving setup | blocked; positive control false |
| AT11 | IPv6 platform matrix | blocked; positive control false |
| AT12 | serving setup | blocked; positive control false |
| AT12 | designated DNS exception | blocked; positive control false |
| AT12 | unknown/spoofed member source | blocked; positive control false |
| AT13 | serving setup | blocked; positive control false |
| AT13 | post-revoke kernel ingress attribution | blocked; positive control false |
| AT13 | short whole-node lease expiry | blocked; positive control false |
| AT13 | lease member expiry | blocked; positive control false |
| AT13 | continuous crash/NetBird/reboot/updater transitions | blocked; positive control false |

All 12 recorded checks are blocked with no counted positive control. Other planned member, hold, restart, revoke, DNS/probe and expiry checks were **not exercised** after setup returned. The old fallback wording about DNS exports is retained verbatim in AT12; this run does not newly establish that capability gap. Full platform and fault-matrix blocks remain carried.

The shared publisher's initial capacity PATCH returned **412 RESOURCE_VERSION_CONFLICT** ([support record](support/publish.json)). Its read-ETag/one-PATCH path is in read-only `tests/e2e/publish_test.go`; it returns without applying capacity or continuing its publish flow. The isolation driver's subsequent publication PUT returned 202, but its operation stayed running for the unchanged 150-second wait: `agent_policy` succeeded, `data_path_probe` was pending with `NODE_NOT_SERVICEABLE / member_forwarding_unconfirmed (forwarding_disabled)`, and NetBird route read-back remained pending. Desired/applied generation reached 3 while publication observed stayed absent. [AT10 setup observations](AT10.json), [operation history](publication-operation-history.json), [receiver](receiver-capture.json).

This is F19-02 for the shared publisher owner, with bootstrap coordination as needed. No outside-owned change, mutation of reports, or retry was performed. No claim is made that host contention caused the 412 or that it is the sole bootstrap cause.

[Boundary review](boundary-review.json) and [producer timing unavailable](producer-timing-unavailable.json) correctly report **expiry not executed**, `inherited_evidence_used=false`. No current continuity/expiry sweep or populated-hold analysis exists. The populated-hold reviewer raised a missing-input FileNotFoundError; clone export also reported the clone had never been created. Those are secondary diagnostics. The generic [recovery verdict](recovery-verdict.json) has no run ID and is blocked; its static creator text does not establish an injection. No redirect injection was reached.

The producer stream contains **155 baseline, 10 renewal and 154 heartbeat records**, no timed Apply/verify operations or command phases. The `retire` compatibility fix passes all 23 synthetic tests, but its runtime phase is **not exercised** in this assembled attempt.

## D-147 and unchanged mechanism gates

The focused run and the assembled lab phase each pass all four exact source memberships: resolver-only `203.0.113.53`, denied-only `192.168.34.2`, host-only `203.0.113.33`, and denied+host `192.168.34.1`. Each transfers 64,064 payload bytes, learns PMTU 1000, delivers ten genuine type 3/code 4 errors totaling 5,760 IP bytes, and reconciles both directions and class `1:2f` exactly. The 56 focused captures have complete decode counts and zero kernel capture drops. [Focused immutable records](history/follow-up-19-lab-final/archive.json), [assembled D-147 record](lab-AT13-d147.json), [assembled capture detail](lab-AT13-d147-detail.json).

Time-exceeded dies at `ctdenyd`, unsolicited fragmentation-needed at `ctinv`. Upload-to-source denies are `ctbadns`, `ctdeny`, `cthost`, `ctdeny` respectively, with zero Internet/DNS upload usage. The literal `ctdeny` requirement for exclusive resolver/host sources is **not exercised** under this production order (F19-01); their actual denial is independently proven. Rules were installed solely through normal Apply and live verification. No predicate was weakened.

Existing lab AT10–AT13 pass in both the final focused session and the assembled lab phase. The assembled reload/revoke and D-104 tests also pass (D-104: 26 positives, 7 drops; last positive −100.770619 ms, first drop −0.875802 ms). These do not substitute for the unexecuted real-node acceptance rows.

## Admission, provenance and final gates

The assembled invocation ran **22:08:56.001043Z–22:13:50.863036Z**, **294.862 seconds**, exit **2** (`lab=0`, `stack=1`). Its two `$FVPN_HOLD_SH` checks returned free in **0.038 / 0.037 seconds**, and its own four suite checks admitted without a contention wait. An earlier continuation reservation wait before recreating the focused lab lasted **30 seconds**. No reservation was requested.

The [vmstat preflight](host-preflight.json) records all six current 5-second rows: idle 87/77/61/76/86/85%, runnable 1/4/32/2/0/3. `pgrep -af 'gates.sh|mk/at-'` showed members' gates process. [30-second overlap receipts](suite-overlap-monitor.jsonl) observed that process at 22:08:56–22:10:56Z; the first absent sample was 22:11:26Z. **The initial assembled window is confounded under D-133.** Periodic samples cannot exclude shorter overlaps or prove scheduler causality. No retry was made.

[Cleanup inventory](cleanup-inventory.json) is zero for owned containers, images, networks and volumes. `go env GOCACHE` was used (reported as `$HOME/.cache/go-build`). All test scratch was on disk under `.agent/tmp`; Python used `-B` or `PYTHONDONTWRITEBYTECODE=1`.

Lane lint/test/build (including amd64/arm64 probe build), handover tests and root `make go-lint` pass. Final publication hygiene and formatting receipts are in [validation results](validation-results.json).

D-098 `f5a5a425` archived all 140 prior root files byte-identically at [follow-up 18](history/follow-up-18/archive.json). All earlier archives remain untouched. The [inherited failed D-147 record](history/follow-up-19-inherited-lab/archive.json), [missing-counter collector failure](history/follow-up-19-lab-attempt-1/archive.json), and [reused-lab contamination](history/follow-up-19-lab-attempt-2/archive.json) remain failed. New retained lab code was committed before its run; the inherited run's dirty embedded-runner provenance is explicitly limited in the dated record.

This run used an empty evidence directory. [Current/previous-only inventory](follow-up-19-evidence-inventory.json) distinguishes current generated files from historical root artifacts; previous-only files are **not evidence for this run**. Large new evidence is lossless gzip with both-form pins only under [immutable history](history/follow-up-19-stream/archive.json). D-144's expected-blocked list and workflow are unchanged.

The session switched from agy Gemini 4 Argon to codex/gpt-6-astra after a content-filter interruption at 16:58 EST (D-143). Note 1's separate rows and unchanged member predicate, and note 2's `retire` fix, are retained. [Reader enum audit](follow-up-19-reader-audit.json).

## Retained follow-up 18 narrative

The following is historical; its links resolve to the archived snapshot. Its passing rows are not this run’s observations.

# Isolation follow-up 18 — 2026-11-03

One assembled run **`9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`**, consumer
**`c248db966d6870182a7c87cf7e410d7524bf56ff`**, source dirty **false**.
Baseline **`090bb97b88b4062073a487c82676d0e1faca4fac`**. Driver **1316.05 s**,
full command **1477 s**, exit **2**. No retry. [Runtime receipts](history/follow-up-18/runtime.json).

| Case | Pass / blocked / fail |
|---|---|
| [AT10](history/follow-up-18/AT10.json) | **9 / 0 / 0**, ISO-F6-01 **6/6** (`foreign ingress redirect containment and D-096 recovery: pass`) |
| [AT11](history/follow-up-18/AT11.json) | **2 / 1 / 0**, carried native-platform matrix |
| [AT12](history/follow-up-18/AT12.json) | **10 / 0 / 0** |
| [AT13](history/follow-up-18/AT13.json) | **6 / 1 / 0**, carried full fault matrix (`short whole-node lease expiry: pass`) |

**Both unlisted blocked rows from scheduled run `41640196805` are diagnosed and
fixed in the owned harness (`Outcome (a)`), and both pass in this assembled run.**
See [dated nightly diagnosis](history/follow-up-18/nightly-diagnosis-2026-11-03.md).

**D-140 qualifies under D-142, and the unchanged evaluator passes.** The separate
shared **D-104 mechanism passes** on netlab 26's precision fixture.
The original seven hold coverage categories pass, and the third incident proves
central's **30-second fallback**. Pure `stale_generation` remains **not exercised**;
[F17-01](history/follow-up-18/follow-up-18-findings.json) names the diagnostic/classification mismatch.
All **six** agent parent receipts record exit 0. No install exceeds 100 ms.

Requirements: §19 AT10–AT13; §8.2–8.3, §10.3, §16.3; ISO01–ISO04, ENT02,
SEC01, DNS03. Central holds support §5.1–5.2 / AT29, without promoting its full matrix.

## D-098, host admission and publication

First commit **`c248db966d6870182a7c87cf7e410d7524bf56ff`** archived all **138**
root files/subtrees of `0ec14b39-467e-4a37-a239-1a33ef12eb71` byte-identically
under [follow-up 17](history/follow-up-17/archive.json), with SHA-256 pins. All
earlier archive files, including `follow-up-17-stream`, stay untouched. Handover
claims already use immutable paths, so no D-132 exception edit was needed.

[Admission receipts](history/follow-up-18/suite-admission.jsonl): `bash "$FVPN_HOLD_SH"` was `free`
before and after the run, and all five suite admission checks waited
**0.044 / 0.044 / 0.036 / 0.047 / 0.039 s** with zero other suite roots.
`go env GOCACHE` before and after returned **`$HOME/.cache/go-build`**.
[Owned Docker inventory](history/follow-up-18/cleanup-inventory.json) is zero for containers, images,
networks and volumes. Twelve large streams are lossless gzip with both-form pins
at [immutable follow-up 18 streams](history/follow-up-18-stream/archive.json).

## D-140: calibrated schedule and deciding pair

[Predeclared procedure](history/follow-up-18/RUN.md), [schedule](history/follow-up-18/boundary-schedule.json),
[all timelines and qualification](history/follow-up-18/boundary-review.json), [unchanged evaluator](history/follow-up-18/deadline-verdict.json).
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
[mechanism trace](history/follow-up-18/reload-stream/TestD104_AT13_DeadlineContinuity/trace.json):
**26 positives / 7 drops**, last positive **D−100.526854 ms**, first drop
**D−0.760365 ms**; Apply→installed ≤**24.993517 ms**, render→installed
≤**11.718073 ms**, precise read **1.494143 ms**.

[Full hold rows and IDs](history/follow-up-18/populated-holds.md) retain all three original arms,
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
(**10 check_failed / 3 isolation_unconfirmed**; carried [F17-01](history/follow-up-18/follow-up-18-findings.json)).

## Stop receipts and retained diagnostics

[Six parent receipts](history/follow-up-18/stop-receipt-review.json) match their launch IDs/PIDs/start
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

[Reservation review](history/follow-up-18/pre-apply-reservation-review.json): nonempty pre-Apply brackets
for sequence/generation **2488/17**, **2878/24**, **1488/28**, **2569/33** are
**1.257233 / 0.899646 / 1.078966 / 1.131673 ms**. Full baseline→Apply recovery
intervals are **213.340536 / 213.023237 / no intervening baseline / 2123.628292 ms**.
All **25** policy-missing windows remain, **23** populated, baseline→Apply
**122.819935–6085.452005 ms**.


## Follow-up 20 — 2026-11-03 (additive current result)

One assembled run **c52bea6c-cd91-4086-a779-656b01277dcc**, measured source **037f4418**, source dirty false; baseline **c359d4a9**. **26 pass / 2 blocked / 1 fail**, exit 2, 1569.901 s, no rerun and no lab invocation. AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **5/1/1** (pass/blocked/fail). [Every row and evidence](followup20-2026-11-03.md#assembled-result).

F19-01 is answered by D-152; F19-02 is answered by devstack 32 and this successful publication. All 14 retained node-write traces are single 202 attempts with no conflict. The successful extra setup publication trace is unavailable, so this is not a complete conflict audit. Its recording fix and the collector's skipped-lab fix are committed in **4fc47dd1**, with offline coverage only. Original failures and missing evidence are retained.

**F20-01 stays failed:** `tunnel_probe / HEALTH_CHECK_FAILED` caused the recorded safety baseline to complete **40.992498819 s before the signed deadline**. Boundary: **0 positives / 202 drops**, no qualified calibration, no last positive. [Owner handoff](requests.md), [findings](follow-up-20-findings.json). Full native platform/fault matrices and pure `stale_generation` remain carried. Native counted forwarding with six installed D-147 PMTU accepts was observed; native fragmentation-needed delivery was not exercised.

Admission wait **0.035 s**; no other-suite overlap sampled; no reservation. Final owned Docker inventory zero. [113-file follow-up-19 archive](history/follow-up-19/archive.json), [86-file current runtime archive](history/follow-up-20-run/archive.json), [current/prior-only inventory](follow-up-20-evidence-inventory.json). The older sections above remain historical; their original root-link context is frozen under follow-up 19.


## Follow-up 21 — 2026-11-03 EST (additive current result)

Run `7599576e-339c-49c9-8bf5-0f710818cbda` on clean source `fe2c68ec`, baseline `9e709518` (#447): **26 pass / 3 blocked / 0 fail**, **1602.678 s**, exit **2**, lab included, no rerun. AT10 **9/0/0**, AT11 **2/1/0**, AT12 **10/0/0**, AT13 **5/2/0** (pass/blocked/fail). [Every row and evidence](followup21-2026-11-03.md#assembled-result).

**F20-01 answered by agent-core 48:** whole-node expiry passes D-140 and qualifies under D-142, **165 positives / 88 drops**, last positive **D−99.541360 ms**, first drop **D−96.362193 ms**, zero interleavings. Historical `c52bea6c` remains failed. **Member expiry stays blocked:** the accepted node lease extends **23.057987 s** after the finite deadline, below the unchanged >25 s bound. No five-probe window or new serving reads occurred. M-1's original-evidence replay rejects `serving 0 / safety_check_failed`, and synthetic serving evidence passes; live coverage remains outstanding.

**Follow-up 20 request 2 answered:** 32 operation snapshots carry this run's ID and AT10 retains successful setup's publication trace. **15 node-write commands / 15 first-attempt 202s / zero conflicts.** Lab and six-assertion recovery pass. The native platform/full fault matrices and pure `stale_generation` remain carried. F48-01's paired signature is not observed in retained logs, not declared fixed.

Admission **0.054 s**, no other-run wait; 54 samples show no overlap; inventory zero. [165-file original root archive](history/follow-up-20/archive.json), [124-file run archive](history/follow-up-21-run/archive.json), [dated requests](requests.md). Earlier sections remain historical; their original root-link context is frozen under follow-up 20.


## Follow-up 22 — 2026-11-04 EST (additive current result)

Run `7b0db5f2-eab5-4534-824a-b54781eed3fd` on clean source `0b95e69a`, baseline `33f0ef8e` (agent-core 49 #454): **27 pass / 2 blocked / 0 fail acceptance rows**, **1578.608 s**, command exit **2**, lab included, no rerun. **The separate D-120 reload/revoke lab failed** on TCP sequence 101 `TimeoutError`; every historical failure stays failed. [Every row and receipts](followup22-2026-11-04.md#assembled-result).

**F21-01 answered:** the fresh-renewal anchor opens the unchanged member window; all five probes time out, bytes **2631→2631**, all six local reads show serving 1 and a non-baseline member chain, and the retained node lease remains valid through completion. This is r28 M-1's first live pass. Whole-node expiry on agent-core 49 passes D-140 and qualifies under D-142: **165 positives / 90 drops**, closure **(D−98.836702 ms,D−96.710276 ms]**, zero interleavings.

**15 node-write commands / 16 attempts / one 412**, resolved by fresh ETag 917→918 and same-key/body retry to 202; all attempts retained. Restart/watchdog records show no F48-01 paired signature. Four source restart-command windows and the initial heartbeat-unreadable watchdog decision are documented; absence of a decision is not proof of a forced D-156 interleaving.

Wait **1414.753 s**, 53 clear overlap samples, zero owned Docker inventory. [173-file root archive](history/follow-up-21/archive.json), [125-file run archive](history/follow-up-22-run/archive.json). **New F22-01** routes the unchanged lab failure to its owner. Native platform/full fault matrices and pure stale_generation remain carried. [Requests](requests.md). Earlier root-link contexts are frozen under follow-up 21.


## Follow-up 23 — 2026-11-04 EST (additive current result)

Run `328ab9e0-7ff4-495a-81c1-410ae8a7d876`, baseline `2d3aaeed`, clean measured source `89accd8d`: **27 pass / 2 blocked / 0 fail acceptance rows**, all 29 statuses unchanged. **D-120 passes under D-161**, with complete checked combined captures: **65 TCP reads, zero timeouts, six guard-attributed lost transmissions, zero attributed or unattributed timeouts, zero UDP losses, longest stall 210.082 ms**. **F22-01 answered**; original run 7b0db5f2 stays failed. [Every row and retained evidence](followup23-2026-11-04.md).

Member expiry opens and passes five denied probes/six serving reads (**r28 M-1’s second live pass**). Whole-node expiry passes D-140 and qualifies under D-142: **164 positives / 69 drops**, bracket **(D−99.957279 ms,D−94.057588 ms]**, zero interleavings. **15 node-write commands / 15 first-attempt 202s / no conflicts**. Four source restart-command windows; no F48-01 paired signature. Watchdog initial heartbeat-unreadable and later policy-expired decisions are retained without inferring a recovery pass.

No other-run wait; admission checks **0.175 s**, **48 clear overlap samples**, run **1430.476 s**, exit **2** for the carried operator blocks, no retry, zero owned Docker inventory. D-160’s static/runtime audit finds no timed-section admission wait; no harness code change. [184-file original root archive](history/follow-up-22/archive.json), [139-file new runtime archive](history/follow-up-23-run/archive.json). Native platform/full fault matrices and pure stale_generation remain carried; no new request. Earlier root-link contexts are frozen under follow-up 22.

## Follow-up 24 — 2026-11-04

Archived run `328ab9e0` byte-identically before root replacement: [203-file archive](history/follow-up-23/archive.json). The 11-04 runner failure remains historically failed; [diagnosis and one assembled run](followup24-2026-11-04.md) distinguish early grant-only renewals from the later binding-bearing renewal. New run `df9dbad7-049d-4eaa-ab98-5490906cc3e9` has scheduled verdict **fail**. The new member schedule and D-163 coverage mapping change no product or acceptance predicate. AT11/AT13 operator matrices and F17-01 remain carried. [Runtime pins](history/follow-up-24-run/archive.json).

**New F24-01 is the reason this local scheduled run stays red:** an empty added-probe schedule causes the unchanged boundary reviewer to raise TypeError. D-142 qualification remains unestablished despite the passing whole-node packet predicate. The owned fix is deferred to an orchestrator-assigned follow-up under the task’s finding-only rule. [Original failed phase](history/follow-up-24-run/scheduled-job.json); [diagnosis](history/follow-up-24-run/boundary-review-failure.json).


## Follow-up 25 — 2026-11-04

Single scheduled-mode run `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db` on clean source `4c495c3b1516022e1de8db2f864432a8a8d171c5`,
current-main baseline `034c82cf991de73db0f3a2f02ebe7270ef04d8c4`: **27 pass / 2 operator-blocked /
0 fail acceptance rows; scheduled job pass, exit 0**. All 29 statuses match
`df9dbad7`; the boundary review now completes and qualifies under D-142. The
current calibration produces 5.5ms spacing and 34 added probes without discounting
any sample. D-120 passes under its own D-161 analyzer. Fifteen node writes used
15 attempts with zero conflicts. Four restart windows retain no watchdog decision;
silence does not establish a retaining pass. No observed foreign overlap and zero
owned Docker inventory at teardown.

[213-file D-098 archive](history/follow-up-24/archive.json),
[diagnosis, every row and limits of F24-01 closure](followup25-2026-11-04.md),
[runtime pins](history/follow-up-25-run/archive.json),
[scheduled summary](history/follow-up-25-run/scheduled-job.md).
Historical failures stay failed. AT11 platform and AT13 full fault matrices remain
operator rows; F17-01 stays open under D-163. F24-01 was not added to D-163.


## Follow-up 27 — 2026-11-05 (NEWEST ATTEMPT, FAILED)

Single assembled run `d69f5652-aed6-4fdd-9e7c-edface751237` ([`follow-up-27-run-receipt.json`](history/follow-up-27-run/follow-up-27-run-receipt.json), [`invocation.json`](history/follow-up-27-run/invocation.json)) on clean source `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` (`source_dirty: false`), current-`main` baseline `aaffdc7fe176c5e3edc521c8ca491bff28bd4a6f`, **2026-11-05T10:11:41.296179Z–2026-11-05T10:33:52.446700Z** (`1331.15` s, `invocations: 1`, `reruns: 0`, `lab_included: true`, `confounded: false`): **26 pass / 2 operator-blocked / 1 fail acceptance rows; `make` exit 2; [`scheduled-job.json`](history/follow-up-27-run/scheduled-job.json) status `fail`**. Canonical root `AT10`–`AT13` and `support`/`reload-stream` payloads in `docs/e2e/at/isolation/` remain byte-identical to pre-run `37b3c25f60194e0d2261cf9aca30dfe8fc5d93be` (`f25`, `bd73d7a4-03f1-4b17-97c2-9c25a39ac0db`, backed by [`history/follow-up-25/archive.json`](history/follow-up-25/archive.json)), while all 135 Follow-up 27 run archive files are pinned in [`history/follow-up-27-run/archive.json`](history/follow-up-27-run/archive.json). [Full diagnosis, clone/stop reviews and receipts](followup27-2026-11-05.md).

| Record (sealed in `history/follow-up-27-run/`) | Pass / blocked / fail | Overall |
| --- | --- | --- |
| [`AT10.json`](history/follow-up-27-run/AT10.json) | 9 / 0 / 0 | pass |
| [`AT11.json`](history/follow-up-27-run/AT11.json) | 2 / 1 / 0 | blocked (`IPv6 platform matrix`) |
| [`AT12.json`](history/follow-up-27-run/AT12.json) | 10 / 0 / 0 | pass |
| [`AT13.json`](history/follow-up-27-run/AT13.json) | 5 / 1 / 1 | fail (`short whole-node lease expiry` fail; `continuous crash/NetBird/reboot/updater transitions` blocked) |

- **Independent `AT13` `short whole-node lease expiry` failure:** Under held lease `4a45a251-aab8-45c3-abc9-5abdd7623c61` (`generation: "34"`, `expires_at: 2026-11-05T10:30:39.921834424Z`, `deadline_mono_ns: 198082369407399`), [`boundary-schedule.json`](history/follow-up-27-run/boundary-schedule.json) qualified (`spacing_ns: 2800000`, `71` dense boundary probes, `273` total). Probe `packet_id: 145` (`source_port: 42145`, `scheduled_mono_ns: 198081468407399`, `send.mono_ns: 198081469013038`, ~`900.4` ms before deadline) recorded `result: "socket_error"` in [`whole-node-probes.json`](history/follow-up-27-run/whole-node-probes.json) when `net.DialUDP` returned an error (`e != nil`; underlying errno/error text not recorded) in `tests/e2e/at/isolation/observer_main.go:292-298` and `c.Write` was not called. Because no UDP datagram was transmitted on the wire for packet `145`, [`whole-node-packets.json`](history/follow-up-27-run/whole-node-packets.json) (`438` entries, `drops: 0`) and [`whole-node-node-capture.txt.gz`](history/follow-up-27-run/whole-node-node-capture.txt.gz) had no ingress/egress record (`ingress_mono_ns: "0"`, `egress_mono_ns: "0"`), causing `EvaluateDeadline` ([`deadline-verdict.json`](history/follow-up-27-run/deadline-verdict.json)) and `review_boundary.py` ([`boundary-review.json`](history/follow-up-27-run/boundary-review.json)) to fail with `errors: ["packet 145 missing send/ingress/read"]` (`positives: 164`, `drops: 108`, `last_positive` packet `165` `egress_offset_ns: -99616258`, `first_drop` packet `166` `ingress_offset_ns: -97485749`, `interleavings: []`). `review_followup10.py` (`tests/e2e/at/isolation/producer_timing.py:459-460`) then raised `ValueError: duplicate probe or missing ingress` on packet `145` and exited `1` before writing `producer-timing-analysis.json` or `expiry-failure-correlation.json` ([`assembled.log.txt.gz`](history/follow-up-27-run/assembled.log.txt.gz)).
- **Included lab, member-expiry and node-write results:** Included netlab ([`lab-AT10.json`](history/follow-up-27-run/lab-AT10.json)–[`lab-AT13.json`](history/follow-up-27-run/lab-AT13.json), [`lab-AT13-d147.json`](history/follow-up-27-run/lab-AT13-d147.json)), `D-120`/`D-161` ([`reload-stream/TestAT10_AT13_ReloadRevokeBaseline/at10-analysis.json`](history/follow-up-27-run/reload-stream/TestAT10_AT13_ReloadRevokeBaseline/at10-analysis.json): `65` TCP reads, `0` timeouts, `8` guard-attributed lost transmissions, `0` UDP losses, longest stall `209.558` ms on seq `63`), `D-104` ([`reload-stream/TestD104_AT13_DeadlineContinuity/trace.json`](history/follow-up-27-run/reload-stream/TestD104_AT13_DeadlineContinuity/trace.json)), and `AT13` `lease member expiry under control loss` (`5` denied probes, `6` local serving reads, `post_deadline_bytes: 2632 -> 2632`, `node_lease_valid_through_last_probe: true`) all passed. All `9` unique logical node-write operations (`idempotency_key`s on `/api/v1/admin/nodes/01a1ab33-4234-7bd4-a7ad-685f191a3e45` and `/publication`) succeeded on `1` attempt each with HTTP `202` (`0` `412` conflicts) across `13` stored receipt occurrences (`8` in `AT10.json` — `3` check-local + `5` root observations; `2` in `AT12.json`; `3` in `AT13.json`) due to `4` cross-record copies.
- **Clone transport and stop reviews ([`auxiliary-reviews.json`](history/follow-up-27-run/auxiliary-reviews.json)):** [`stop-receipt-review.json`](history/follow-up-27-run/stop-receipt-review.json) is `reviewed` (`review_stops.py` exit `0`, `all_parent_receipts_matched: true`, all `6` stops `exit_code: 0`, `signal: null`, including source PID `911` `signal_to_exit_upper_ns: 7155643` and clone PID `1155` `signal_to_exit_upper_ns: 6859340` terminal `direct_agent` stops). [`clone-transport-review.json`](history/follow-up-27-run/clone-transport-review.json) is `not established` (`review_clone.py` exit `1`): **8 of 9 criteria are `established`** (`clone_created` container `226c3cded7d73b907529740b7c6bfba290a5725b699f07081c95200d49914ddd` at `10:32:59.398564296Z`, `identity_fingerprints_match`, `final_agent_stop`, `executing_release_config` on `/usr/local/bin/fvpn-agent` + `/etc/vpn-node/agent-run.json` with `control_url: https://127.0.0.1:9444` and `datapath.snat: 192.0.6.23` matching clone `fvup0` `192.0.6.23/24` with `selected_release_id: ""`, `corrected_transport` with `5` accepted `http_status: "202"` submissions in clone `agent.log` and owned `127.0.0.1:9444`/`172.29.33.30:9443` `resolver-proxy` sockets, `central_conflict_frozen` with `same_identity: true`, `endpoint_rows: 1`, `conflicts: 1`, `mapping_frozen: true`, `source_freeze_response_and_report` on source-only `9445` transactions `[1802, 1804, 1807, 1812]`, `clone_stop_log_retention` for both `agent` and `watchdog` logs). **`consumer_latch_held` is `not established`**: [`identity-freeze.json.gz`](history/follow-up-27-run/identity-freeze.json.gz) `row[1]` (`10:33:01.121Z`–`10:33:01.146Z`) and `row[2]` (`10:33:04.171Z`–`10:33:04.197Z`, `+3.025` s) both show fresh `reason: identity_frozen`, `serving: 0`, `policy_expires_mono_ns: 0`, `baseline_ok: 1` heartbeats and verified drop-baseline rulesets (`ok1 == True`, `ok2 == True`, `contradiction == False`), but `after-clone-creation` in [`clone-runtime.json`](history/follow-up-27-run/clone-runtime.json) (outer `read_start: 10:32:59.621346473Z` / `read_end: 10:32:59.726700358Z`; inner `snapshot.read_start: 10:32:59.674883482Z` / `snapshot.read_end: 10:32:59.723398052Z`) completed ~`107` ms before the parent-waiter `StartedAt` (`10:32:59.833688125Z`, sampled before `cmd.Start()` in `tests/e2e/internal/agentwait/wait.go:59-60`) for restarted source `fvpn-agent` PID `911`, leaving the first source process snapshot `unavailable` (`agent_pid` `"file absent"`, `process_coherent: false`) and `source_bracket` with only `after-clone-observation` (`len == 1 < 2`).
- **Admission, overlap and cleanup:** [`prelaunch-admission.json`](history/follow-up-27-run/prelaunch-admission.json) (`30.639` s quiet window), [`suite-admission.jsonl`](history/follow-up-27-run/suite-admission.jsonl) (`4` checks), [`overlap.json`](history/follow-up-27-run/overlap.json) (`45` samples, `foreign_processes: []`), and [`cleanup-inventory.json`](history/follow-up-27-run/cleanup-inventory.json) (zero owned `fvpn-w4-isolation*` containers, images, networks, volumes) are all clean. [Owner requests](requests.md#follow-up-27--2026-11-05-newest-attempt-failed).
