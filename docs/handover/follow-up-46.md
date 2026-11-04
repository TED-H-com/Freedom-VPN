# Follow-up 46 — #469–#473; F22-01 answered, r30 fixed, D-162 routed

Verified against main **52b93845b5cfeac1dd63e04c700857fcd098a7f6**, through
**members 23 #473 / af6d7828**, on **2026-11-04 EST**. Follow-up 45 ended at
**#468 / c148cf34**. There are exactly **five merges and five final roadmap
rows: #469, #470, #471, #472 and #473**. No unbriefed PR merged inside the range.

**Running at briefing, status only:** review **r31**, a post-merge code review
of agent-core 51 and updater 26 by **agy/gemini-4-argon-high**; **devstack 36**,
implementing D-162 in the fixture, auditing consumers and If-Match writers,
and running full acceptance on agent-core 51 and updater 26. No delivered or
merged PR for either was present at the initial read-only check. Later results
are **after the cutoff** and do not enter this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup46Claims` pins the narrative,
final roadmap rows and PR excerpts, frozen producer documents, original
archived evidence, main Go test declarations and frozen source readings.
Every lane-document quotation below comes from a committed frozen copy.
No assertion reads living members, isolation, secops or usage evidence roots.
Follow-up 45's rewritten inputs now read frozen **#472 bytes**; its assertions
and every earlier claims test remain unchanged.

## Handover 45 (#472): the previous cut and verified counts

[PR #472](https://github.com/TED-H-com/Freedom-VPN/pull/472) records the
**#464–#468 cut, up to c148cf34**. On main, `TestHandover_Followup45Claims`
has **15 subtests**, **five final roadmap rows and PR excerpts**, **27 frozen
inputs** (nine #466 documents and eighteen producer records), **13 Go test
declarations**, **17 frozen code readings** and **nine archived artifacts**.
It preserves **38 earlier claims functions**, after reversing five input
redirects in follow-up 44. Its register contains **64 decisions** (two updated,
62 byte-identical) and **133 findings rows: 7 updated, 7 added, 119 byte-identical**.
These are the verified #472 counts, not this cut's changes.

The session account is attributed to #472's body: **codex gpt-6-astra**,
launched **06:48 EST** on **2026-11-04** on #468's roadmap row, delivered
**07:46**; **8 commits, 54 owned files, +107,301/−327**. Final gates waited
**45 minutes** for host admission while members 23's assembled run was active.
Docs, claims, snapshots and digest values only; no reservation, Docker or
live measurement.

## Isolation 23 (#469): F22-01 answered on complete captures

[PR #469](https://github.com/TED-H-com/Freedom-VPN/pull/469) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-46/docs_e2e_at_isolation_followup23-2026-11-04.md.txt)
record the **D-098 archive of run 7b0db5f2's 184 files**, byte-identical under
**history/follow-up-22/**. Earlier archives and historical verdicts remain.

**D-160 was checked without a harness change.** Runner admission checks
precede the lab, the test, the stack and the driver; none is inside the D-120
campaign or AT13's member and whole-node timed expiry sections. The run's
admission records confirm this. Follow-up 22's private nested wrappers were
not reused.

Run **328ab9e0**, measured source **89accd8d** on main after netlab 36,
records **27 pass, 2 blocked, 0 fail**, with **all 29 statuses identical to
7b0db5f2**. The blocks remain **AT11's IPv6 platform matrix** and **AT13's
full continuous fault matrix**, the two operator matrices.

**F22-01 is answered:** the included D-120 lab passes under D-161 on complete
checked combined TCP captures: **65 TCP reads, zero TCP timeouts, zero UDP
losses**, **six lost transmissions**, each attributed to a verified guard
and recovered by retransmission, and **longest stall 210.1 ms** (210.082 ms).
**D-161's accepted-timeout branch did not run live.** The allowance count is
zero, since recovered transmissions are not timeout events. Historical
**7b0db5f2 stays failed and its cause unproven**; it retained no TCP packets.
`TestAT10_AT13_ReloadRevokeBaseline` is the campaign on main.

**AT13 member expiry gives r28 M-1 its second live pass:** five requests
time out with bytes flat and six serving-node reads; denial completes
**13.9 s before the node's own lease expires**. Whole-node expiry **passes
D-140 and qualifies under D-142 at generation 32**; its measured closure
bracket is not an exact mathematical switch-at-D claim.

**Node writes: 15, all first-attempt 202.** Follow-up 22's one 412 remains
recorded; no merged change is credited with its absence. A 202 alone remains
separate from operation and observed-state completion.

## Agent-core 51 (#470): r30 H-1, M-2 and M-3 fixed

[PR #470](https://github.com/TED-H-com/Freedom-VPN/pull/470) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-46/docs_node_followup51-2026-11-04.md.txt)
record diagnosis first and the three repaired findings:

- **H-1:** only **Run** sets the run context. Outside Run, **no caller's
  context owns the recover child**; **Close still cancels and joins it**.
  Uploads keep Boot's caller as their parent. The orchestrator's reading
  stays unchanged: **neither CLI mode reaches H-1**.
- **M-2:** a held closed recovery may make **one catch-up lease request for
  a newly available ACK**, despite an earlier attempt in the same Step.
  Recording that ACK prevents a second bypass; retry bound and Retry-After
  remain. The accepted clean report arms D-157's existing one-shot.
- **M-3:** deduplication has **one exception**, a closed recovery waiting to
  bind its own report. **The 32-entry capacity guard still comes first**;
  ordinary coalescing is unchanged.

**Unchanged:** D-155, D-156, D-157, D-067, D-135 and D-136. Forwarding still
opens only on a **populated lease and a verified Apply**. The r30 agent and
watchdog proof names are on main, including
`TestR30_H1_CanceledBootContextSkipsRecoverAndAdmitsServing`,
`TestR30_H1_PerCallBootContextWedgesStepRecovery`,
`TestR30_M2_PreEnforceLeaseAttemptBlocksPostRecoveryRenewal` and
`TestR30_M3_PendingOutboxDedupSkipsNewClosedRecoveryReport`.
Every imported assertion is retained; two manual-clock corrections preserve
the retry floor and Retry-After. Its ordinary lab passes **13/13 on one
attempt**, functional evidence only, not a new AT02 timing campaign.

## Updater 26 (#471): r30 H-2 and M-1 fixed under clarified D-158

[PR #471](https://github.com/TED-H-com/Freedom-VPN/pull/471) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-46/docs_updates_node-followup26-2026-11-04.md.txt)
record **five consecutive stale answers, 20 ms apart**, giving **exit 1
with the final stale detail**. **Every live answer**, including empty or
unreadable metadata, **resets the stale count**. Live answers use the **10 s
budget from the first attempt**: **no exit 3 before it**, and **exit 3 on the
first live answer at or past it**. Stale answers can extend the wait nominally
about 100 ms past the budget; filesystem work and scheduling add elapsed time.

One **update lock wait** record is written **only when the first attempt
misses on ErrBusy**, including a later cancellation. A non-busy first error
and cancellation before any attempt write none. **Unchanged:** recover and
standalone Retain keep their prompt five-attempt window; nested callers do
not wait; the error lines, exit table and nodelock remain.

Main carries `TestR30_H2_StaleThenLiveLockMetadataAbortsEarlyWithExitBusy`,
`TestR30_M1_NonBusyFirstLockErrorEmitsSpuriousWaitLog`, and the four
`TestD158_Consecutive*` proofs. Virtual retry-clock results do not measure
device latency. **D-158's wait remains unmeasured live.** Devstack 35 closed
F30-01 without contended acquisition; updater's earlier-base note calling
F30-01 open does not override that closure.

## Members 23 (#473): fixture cadence and the corrected IPv6 boundary

[PR #473](https://github.com/TED-H-com/Freedom-VPN/pull/473) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-46/docs_e2e_at_members_followup23-2026-11-04.md.txt)
record the **D-098 archive of run 3eb3960d's 122 root records**, byte-identical
with original gzip and SHA-256 pins. `TestD098_Followup22Archive` is on main.

Run **6a25221b on c148cf34**, measured source **8bde2b6a**, has **3 pass,
7 blocked, 0 fail; 706 passed, 8 blocked and 0 failed subchecks**. The only
count change against **3eb3960d's 702/8/0** is **AT01's four conditional
continuation checks** (19 to 23 passing subchecks). All ten case statuses
stay; AT02, AT03 and AT08 pass, the other seven remain blocked. This baseline
precedes agent-core 51 and updater 26; their repairs receive no credit for it.

**F22-D146 is located in the lab's fixture and stays open.** The regular
node's prepared config omits **observation_seconds**, so the agent reports
every **30 s**, while the serving query requires an observation **under 15 s
old**. **IPv6 1280 indexes 49 and 51 pass**; **dual 1280 indexes 44 and 46
are blocked on an observation 22.61 s old**, the sole failed condition.

**Two orchestrator readings in #467 were wrong, as #473 corrects them:** the
5 s observation override belongs to the **installer node only**, not the lab's
regular agent; and **34.85 s between two retained latest rows was not proof
of a gap in accepted reports**. The complete row windows show roughly 30 s
reporting. The 15 s bound is an acceptance predicate, not a product rule.

**F22-AT01:** the original node's **IPv6 loopback-source WireGuard response
reached member ingress**, then was dropped **before WireGuard counted it**.
Node packet 158 and member packet 161 share an IP hash; member header errors
rose from zero to one while WireGuard receive bytes stayed zero. **Where it
was dropped is unproven**: there is no packet-specific kernel drop trace or
proven source-selection mechanism. This replaces the old “consistent with
IPv6 return-path loss” reading. The historical outcome stays. F22-AT01 and
**Members F11-01 remain open, upstream-dependent under D-148**; the **::1
response-source boundary is already reported upstream**. The new counted
initial transfers add a sample, without an inferred merged fix or closure.

**AT02 under D-157:** one early renewal **63.9 ms after closed-recovery Apply
544**, then a populated generation-15 lease and verified serving Apply 550;
one intervening request and no baseline write. **Deny-all: 179 ms with 7
losses**, alongside #467's 205 ms / 8 losses. D-157 was in both runs; no timing
change is attributed to a merge. **Node writes: 30, every one a first-attempt
202.** Every historical failure stays failed.

## Decisions and findings

The [decision table](README.md#follow-up-46-decisions-register-2026-11-04-est)
has **65 rows**: **D-162 is the only new row**, D-158's clarification is
implemented by #471, and **63 other rows are byte-identical to #472**.
No other decision after D-161 was present on main.

**D-162 — 08:1x EST, 2026-11-04, orchestrator; provisional, the owner may
overturn:** set **observation_seconds: 5 in both prepared configs**, so the
regular node reports every 5 s like its heartbeat, poll and installer node.
**TEST ONLY:** the product default **30 s** and **every acceptance predicate,
including the 15 s bound, are unchanged**. The bound was written for a lab
reporting every 5 s; widening it would let a stale node pass. Every accepted
observation increments the node's ETag, so **If-Match writers meet a newer
ETag six times as often**. **Devstack 36 implements and audits it, running at
cutoff**; no result is imported.

The [findings register](README.md#findings-register-at-the-follow-up-46-cutoff)
keeps **133 rows: 10 updated, 0 added, 123 byte-identical**. Members 23 names
no new finding. F22-01 is answered; r30's five findings are fixed, with
[review r31 running at cutoff](../reviews/README.md#r30-dispositions-at-the-follow-up-46-cutoff-2026-11-04-est).
F22-D146 is located and open; F22-AT01 and F11-01 are open with the new evidence;
AT02 adds its measurement. Every other open row stays unchanged.

## Operator and member entrypoints

Seven entrypoints point at this cut. The operator guide replaces updater 25's
D-158 sentence with **updater 26 Requests item 1's exact sentence from the
frozen copy**, followed by one sentence narrowing the logging guidance to
first-attempt ErrBusy. AT02's existing anchor gains the **179 ms / 7-loss**
measurement after one early renewal. **The F30-01 paragraph and TCP reload-guard
anchor and wording stay as written.** Nothing else in this cut changes what
an operator or member sees. D-162 is TEST ONLY. Scope-review changes only
digest values. No date is promised for anything not on main.

### Requests to orchestrator

**2026-11-04 EST:** carry devstack 36's D-162 fixture, consumer and If-Match
audit and full acceptance; r31's post-merge review remains status only.
Members #473 routes missing per-attempt observation-submission timestamps
and outcomes in the agent/control client to agent-core. That outside-owned
instrumentation item remains with its owner; this lane adds no product edit.
The D-148 owner retains F22-AT01's new packet-boundary evidence and F11-01.
No new outside-owned pin needs changing. Deliver the rest; no permission or
new run is requested. Historical failures, AT11/AT13 operator matrices,
AT29's independently clocked guest, AT31, physical arm64, power loss and
other carried resource gaps remain. D-158 live contention stays unmeasured.
