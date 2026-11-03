# Follow-up 39 — #428–#429; r26 archived; owner answers recorded

Verified against `main` **1b87ab32344fa3be3216df0d5f7b489eb853e011**, the
final roadmap commit through **#429**, on **2026-11-03 EST**. This cut begins
immediately after #427 (**77626af4**) and contains exactly **#428 and #429**.
**No unbriefed PR** merged inside the range. Main and remote main were at this
cut at task start.

**Netlab 34 and agent-core 44 were running when briefed**, launched with this
session. Netlab 34 implements D-146/D-147; agent-core 44 performs D-150's
measurement after the merge. Status only: neither is part of #429 and no
result from either enters this cut. No live acceptance or timing measurement
is made by this handover.

**D-098 / D-109 / D-132:** `TestHandover_Followup39Claims` pins two final
roadmap rows and public PR excerpts, ten frozen #428 inputs, dated agent-core
records, exact r26 bytes, a separate frozen source audit, all changed register
rows, decision status-only confirmations, the ten answers and Go proof names
on main. Earlier claims keep their assertions, with rewritten-document reads
redirected to frozen #428 bytes. Living members **81b638e8**, isolation
**9f1bc0ff** and usage **b8a6e9e1** evidence roots are never claims inputs.

## Handover 38 (#428)

[PR #428](https://github.com/TED-H-com/Freedom-VPN/pull/428) records the
previous cut at **#427 / 77626af4**, handover 37 only. Its actual
`TestHandover_Followup38Claims` and manifests verify **12 subtests**, **one
final roadmap row and public PR excerpt**, **ten frozen #427 inputs**, the
**exact r25 archive** and a **separate frozen source audit**. The register is
**99 findings: 97 unchanged, two added**, **47 decisions**, **ten owner
requests**. No new request.

Its body records **codex `gpt-6-astra`**, launched **11:43 EST**, delivered
**12:00**, **two commits and 28 owned files**. Admission waited **235.1 s
once**, for agent-core 42's netlab suite; **suite wait 0 s**. No continuation
is recorded. The r25 archive is **31,868 bytes**, SHA-256
**7514218895b014d62d09693b5de38193b3a412c5fa65c2aeb3b44d720d414286**,
with additive citation errata in the reviews index. Its scope-review refresh
changed **258 values, all digests**, independently checked against the before
and after objects. Its r25 main-overlap question is now answered by 42 below;
the ten owner requests are answered in this cut.

## Agent-core 37–43 (#429)

[PR #429](https://github.com/TED-H-com/Freedom-VPN/pull/429) merges seven
follow-ups on one branch, preserving their commits and merges of main:
**65 non-merge commits and 95 owned files** against main **567080be**.
37 started **00:37 EST** on #401's **05b2c2f2**; 43 delivered **13:07 EST**.
37 used agy `claude-opus-5-5-high`; 38–43 used codex `gpt-6-astra`.

| Follow-up | Scope from its dated record and the PR |
|---|---|
| 37 | [r20 adoption/signal judgments](../node/followup37-r20-adoption-review-2026-11-03.md): refused-restart usage settlement, bounded Boot baseline, checks before lock/probe and withdrawal heartbeat; L-1 rejected with guards. The appended 38 correction supersedes the original attribution and stop-budget claims. |
| 38 | [r21 attribution and Boot](../node/followup38-r21-attribution-and-boot-2026-11-03.md): pre-refresh provenance, final-settlement gates, open/closed epochs, one 10 s Boot worker and 3 s Close, truthful failures; later 39 corrections remain attached. |
| 39 | [r22 judgments](../node/followup39-r22-judgments-2026-11-03.md): settle before identity change, durable boot/epoch intent, all-handoff pending gate, corrected restart recovery, initialization retry, read-only adoption preparation and truthful Boot/probe-clear receipts. |
| 40 | [r23 judgments](../node/followup40-r23-judgments-2026-11-03.md): reset-only counting continuity, intent surviving transient marker failure, epochClosed separate from barrier completion, withdrawal/reporting fixes and L-1 clarification. |
| 41 | [r24 judgments](../node/followup41-r24-judgments-2026-11-03.md): unchanged genuine 304 ticks retain verified enforcement; allow-element completeness and independent device deadlines; approved setup-only correction to the original H-1 fixture. |
| 42 | [r25 judgments](../node/followup42-r25-judgments-2026-11-03.md) and [outcome](../node/followup42-outcome-2026-11-03.md): two deadlines, same-tick survivor Apply, expired-only settlement, continuity-aware retention and persistent uniform millisecond renewal jitter (Netlab F32-01). Enforcement change is only the approved DeadlineMono comment. |
| 43 | [outcome](../node/followup43-outcome-2026-11-03.md): imports r26's six proofs, merges main once, attempts two D-133 windows with no qualifying leg, then passes the ordinary B lab's 13 scenarios and other gates including plain whole-module Go tests; no production change. |

The [r20–r25 disposition table](../reviews/README.md#r20-to-r25-dispositions-at-the-follow-up-39-cutoff-2026-11-03-est)
records **each of the 25 requested rows**, its verdict, sub-traces, rejected
claims, corrected attribution, fix and named tests. r20 L-1 is **closed as
rejected**; r23 L-1 is **clarified**; r23 H-1b is a **provenance limitation
with a retained guard**, not a proven healthy-renderer outage. No requested
row is absent. The table also records 41's additional r24 H-2 fair-share
judgment (expected behavior, not an agent defect) and 42's stale-comment
correction. Netlab's separate r24 H-2 disposition stays unchanged.

**Follow-up 38's open r25 H-1 question is closed.** 42 judges the verify-path
whole-node withdrawal and contradictory report, and the watchdog's close at
the earliest binding deadline, as **pre-existing on main**. 41 added the
**Step-entry and retained-read variants**. The fix retains survivors and
reports actual state; every binding still has its independent kernel cutoff.
Both **node/agent/review_r25_test.go** and **node/agent/review_r26_test.go**
are now on main, and their six names each are pinned as Go test declarations.

### Follow-up 43 comparison boundary

[Timing results](../node/followup43-timing-results-2026-11-03.md) retain both
windows, including all partial distributions and failed operations:

| Window (UTC reservation → release) | Leg qualification | Partial outer guard p99 / max | Partial whole serving Step p99 / max |
|---|---|---|---|
| 1: 17:44:03 → 17:45:45 | A1 unqualified after 36.144 s; foreign CPU 0.691858 cores > 0.5; 1/13 scenarios | 37.742 / 37.742 ms | 205.757 / 205.757 ms |
| 2: 17:55:45 → 17:58:23 | A1 unqualified after 92.415 s; foreign CPU 1.311445 cores > 0.5; 1/13 scenarios | 81.643 / 81.643 ms | 1636.113 / 1636.113 ms |

**B1, A2 and B2 never started in either window.** Each partial A1 retains
**13 failed operations (three Apply, ten Verify)**; no scenario failure printed
before interruption. Neither completes D087/D104 packet predicates: those
are unavailable, not passing. Both reservations are exhausted, with no retry
or third reservation. Window 2 starts 600.037 s after the first release.

| Qualified pool | Outer guard | Install command | Render to installed | Whole serving Step | Pre-Apply accounting |
|---|---|---|---|---|---|
| A | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles |
| B | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles | n=0; no percentiles |

**No A/B comparison or attributable shift is supported.** Partial distributions
remain descriptive and excluded from both pools. The separate ordinary B lab
passes all 13 scenarios after both reservations are released, including
D087/D104; it is functional evidence, not a timing leg or a historical upgrade.

**The merge decision is D-150**, the orchestrator's provisional decision, not
r26's recommendation: 37–43 merged without the comparison. Every delivery's
full production node-test-netlab passed D087/D104; D-133 attributes failed
live timing gates. Periodic foreign jobs outside the project stopped both
windows, and the orchestrator never pauses them. Under agent-core 39's
every-interval rule this host cannot qualify a complete leg. **The comparison
is not dropped:** 44 measures the same arms after merge, marks foreign-CPU
intervals over 0.5 cores and excludes overlapping spans from both arms.
Every other rule of agent-core 39 and D-138 stays unchanged. An attributable
B shift or B-only failure routes a fix to a later follow-up. This is lab-only;
there is no operator procedure or measurement result from 44 here.

## Review r26

[Exact archive](../reviews/r26-expiry-fixes-review.md), fetched from
**r26-expiry-fixes-review** at **5c6d6abe6af7ebba4707ef64c56c157362f74b7b**,
**12:28:35 EST**: **19,545 bytes**, SHA-256
**8cd7cc8be881389d5f50e70abe2e18eb13136b2281d66a261d9ad5820a3f0d28**.
Inspection found no host path, username or secret. Product paths remain.
agy **`gemini-4-argon-high`** reviewed **8dec4100ce9fba96ea8b0436c41ed8d2b0c75545**
and the combined **77626af4..8dec4100** changes. Findings:
**0 Critical, 0 High, 0 Medium, 0 Low**. **“Merge 37 to 42 as they are”** is
the reviewer's recommendation; D-150 governs what actually merged.

[The reviews index](../reviews/README.md#r26-dispositions-at-the-follow-up-39-cutoff-2026-11-03-est)
records the reviewer's **AQ1–AQ7** claims individually: deadline separation,
same-tick survivors/early-return truth, expired-only accounting, interval
continuity/intent persistence, jitter, comment/record consistency and edited-test
coverage. It lists the **three not-examined items** and what 43 supplied.
Every copied citation/name was checked at its cited commit, with additive
errata for stale locations, timeout/guard responsibilities, future-binding
omission and the watchdog description. The archive remains byte-for-byte
unchanged. 43 brings all six `TestR26_Held_*` proofs to main.

<a id="owner-answers"></a>

## Owner answers

**Ted, 2026-11-03 about 12:30 EST, relayed by the orchestrator**: five new
owner decisions **D-145–D-149**. **D-150** is the orchestrator's provisional
decision, which the owner may overturn. Main had no decision after D-144;
**53 decisions** now, including the six additions. **D-135, D-136, D-137's
supported scope, D-138, D-140, D-141, D-142 and D-144** are confirmed with
substance unchanged; **D-143 stays Ted's**. The
[decision table](README.md#follow-up-39-decisions-register-2026-11-03-est)
contains the full decision text and confirmation dates.

| Historical request | Answer at this cut |
|---|---|
| 1: D-135 + D-136 confirmation | Confirmed by the owner. |
| 2: D-137 hardening option | D-146: TCP MSS clamp approved; no lower tunnel MTU. |
| 3: D-138 overturn option | Kept and confirmed. |
| 4: D-140 overturn option | Kept and confirmed. |
| 5: D-141 overturn option | Kept and confirmed. |
| 6: F11-01 repair direction and upstream report | D-148: upstream-dependent, no product/pin/daemon/architecture change; [discussion 8473](https://github.com/netbirdio/netbird/discussions/8473) filed by the orchestrator on 2026-11-03. |
| 7: RELATED ICMP from spec-denied outer sources | D-147: only RELATED IPv4 fragmentation-needed type 3/code 4 for an allowed member connection; time-exceeded and all other ICMP stay dropped. |
| 8: D-142 overturn option | Kept and confirmed. |
| 9: D-144 overturn option | Kept and confirmed. |
| 10: the G5/G6 trade | D-145: keep current guard; C1 unimplemented; reopen only for field complaints or measured member-visible impact after r24 H-1's fix merges. |

**All ten owner requests are answered; the open owner-request list is empty.**
Their exact original text remains in [follow-up 38](follow-up-38.md#requests-to-orchestrator).
D-149 additionally dismisses Dependabot #22's development-scope minimatch
risk as tolerable per #349; revisit when eslint-plugin-jsx-a11y accepts
minimatch ≥ 11.0.1. D-146 leaves receive-only UDP/QUIC without PTB as a
limitation and all F8-02 failures failed. D-145 keeps netlab 31/32's C1 design
and revision on main as history, with no C1 benefit promised.

## Register and guidance

The [99-row register](README.md#findings-register-at-the-follow-up-39-cutoff)
changes **25 agent-core judgment rows and five owner/status rows**;
**69 other rows are byte-identical**. Historical failures stay failed.
The seven entrypoints point here. r24 H-1 is fixed: healthy unchanged
verified enforcement is retained, while changed authorization/model, renewal
and required recovery still install. r20 H-1's refused-restart accounting
warning is replaced by the merged settlement/provenance behavior. Unknown
ownership still produces honest gaps; no universal lossless restart claim.
**F30-01 stays known behaviour** with no merged fix.

**D-145 keeps the current guard; no hitless hand-over is implemented.**
D-146 and D-147 are decided and being implemented by netlab 34, **not on main
at this cut**, with no date promised. F11-01 remains upstream-dependent under
D-148 with no product change. D-150 changes no operator guidance.

### Requests to orchestrator

2026-11-03 — **No new scope exception or outside-owned pin request.** No
item is stopped. The ten owner requests are answered, not carried as open.

2026-11-03 — **Existing routed work:** netlab 34 implements D-146/D-147;
agent-core 44 measures under provisional D-150 after merge. Their results
belong to a later cut. D-148's upstream-dependent F11-01 remains open.

2026-11-03 — **Carried operator work:** confirm the first scheduled
at-isolation exercise after #424; AT11 native platform and AT13 full fault
matrices and Isolation F17-01 retain their existing owners. Neither PR in
this cut records that scheduled exercise. No historical result is upgraded.
