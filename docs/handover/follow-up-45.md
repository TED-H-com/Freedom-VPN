# Follow-up 45 — #464–#468; F30-01 closed, AT02 measured, r30 routed

Verified against main **c148cf349df219124e457113ce90c010016a308e** through
**devstack 35 #468**, on **2026-11-04 EST**. Follow-up 44 ended at
**#463 / 5c21e670**. This range has exactly **five merges and five final roadmap
rows: #464, #465, #466, #467 and #468**. No unbriefed PR merged inside it.
Main and the read-only public PR listing ended at #468 at the initial check.

**Running at briefing, status only:** isolation **23**, launched on #465,
archives **7b0db5f2** under D-098 and runs the assembled D-120 lab with complete
TCP captures judged under D-161 for F22-01; members **23**, armed on #467,
takes F22-AT01 and F22-D146; updater **26** takes r30 H-2/M-1 under clarified
D-158; agent-core **51** takes r30 H-1/M-2/M-3. No delivered or merged PR for
these follow-ups was present at start. Later results are **after the cutoff**
and do not enter this cut. No date is promised.

**D-098 / D-109 / D-132:** `TestHandover_Followup45Claims` binds the full
narrative, final roadmap rows, verbatim PR excerpts, frozen producer records,
review provenance, main test declarations and frozen code readings. Every
lane-document citation below uses a frozen copy committed with this cut.
No assertion reads a living members, isolation, secops or usage evidence root.
Earlier claims keep their assertions; follow-up 44's rewritten document inputs
now read frozen **#466 bytes**.

## Handover 44 (#466): the previous cut and verified counts

[PR #466](https://github.com/TED-H-com/Freedom-VPN/pull/466) records the
**#457–#463 cut, up to 5c21e670**. `TestHandover_Followup44Claims` actually has
**15 subtests**, **seven final roadmap rows and PR excerpts**, **25 frozen
inputs** (nine #457 documents and sixteen producer records), **30 Go test
declarations on main**, and **16 archived code readings**. All **37 earlier
claims functions** stay exact after reversing four input-expression redirects.
The register then has **64 decisions** (60 unchanged and four added) and
**126 findings rows: 10 updated, 2 added, 114 byte-identical**. These are
#466's counts, not this cut's.

The session account is attributed to #466's body: **codex gpt-6-astra**,
launched **04:17 EST** on **2026-11-04**, delivered **04:33**; **6 commits,
52 owned files, +37,107/−325**. Docs, claims, snapshots and digest values only;
no reservation, Docker or live measurement.

## Secops 18 (#464): SECOPS-F17-01 answered, AT29 blocked

[PR #464](https://github.com/TED-H-com/Freedom-VPN/pull/464) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-45/docs_e2e_at_secops_2026-11-04-follow-up-18.md.txt)
answer **SECOPS-F17-01**. The old killed-agent `lease_expiry` case used a
stale cached deadline. It now reads **D at the fault, after the control stop,
from the node's accepted local lease and its serving heartbeat**. It requires
a 60 s cap, lifetime in (0, 60 s], and a matching future monotonic deadline;
the earlier grant end wins. Traffic predicates remain unchanged.
`TestAT29_FaultLease18` and `TestAT29_HistoricalStaleLease18` are on main.

Run **9d927350** passes every runnable case:

- Old case: D read **90 ms after the control stop**, with **58.48 s remaining**.
  The watchdog's **heartbeat_stale** withdrew rules first; both transports
  are quiet from **D + 0.39 s**. This does not isolate kernel-only expiry.
- Live-agent case: **one agent identity throughout**, both transports
  continuous to **D − 0.99 s**, quiet from **D + 2.01 s**. D + 3 s and the
  ten-second quiet requirement pass.
- Grant expiry, device revoke with bounded relay and manual grant withdrawal
  pass. **AT29 goes from failed to blocked**: guest clock rollback still
  needs an independently clocked disposable guest, which is the operator's.

**D-160 is committed harness code:** an admission journal, a free check before
each of six named timed sections, execution receipts but **no admission checks
inside**, and an offline validator. The run retains **509 checks: 507 free,
2 held**, both held checks after the old section, costing 120 s between cases.

**Recorded as it was:** the first pre-subject dispatch stopped before setup
because the admission helper recursed through the wrapped `PATH`. The lane
stopped **11,215 owned descendant processes**, signalled no foreign process,
fixed the helper to use the unwrapped PATH, and added a regression test.
The one allowed rerun produced 9d927350; no sub-case was retried.

The lane records the run as **confounded under D-133**. The orchestrator's
reading in the PR body says **no foreign suite overlapped**: 76 of 77 monitor
matches were its sleeping wait shell and the other was its note command,
with zero CPU ticks. The lane's conservative record **stays as written**;
these are separately attributed readings. Historical **b2de46c8** and
**1bfd5cff** stay failed.

## Netlab 36 (#465): D-161 in the D-120 analyzer

[PR #465](https://github.com/TED-H-com/Freedom-VPN/pull/465) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-45/docs_poc_G2-followup36-2026-11-04.md.txt)
record the merged classifier. A TCP timeout is attributed to the guard only
when complete checked captures prove every lost transmission of the stalled
bytes entered the node and never left it, inside or on the boundary of
**exactly one verified guard bracket**, with both guard and render→installed
**≤100 ms**; later delivery and the echo are in order on the same connection,
with no reset, close or second connection. **Anything else fails (iv), with
its failed conditions named.** UDP losses plus attributed timeouts count
against five per 100 reloads; synthetic 4 + 1 passes and 5 + 1 fails.

**Gate 2, the live campaign: AT10 passes under D-120** with **64 TCP reads,
zero TCP timeouts, zero UDP losses**, **8 boundary losses recovered by
retransmission**, and longest stall **217.7 ms** (217.667 ms, sequence 12).
**D-161's timeout path did not run live; synthetic proofs cover it.**
`TestAT10_AT13_ReloadRevokeBaseline` is the campaign on main, also included
in isolation's assembled run.

Replays of **F22-01 (7b0db5f2)** and **F27-01/B1** still fail: **no TCP
captures**, hence no attribution. The deliberate tightening rejects a campaign
without combined TCP captures as an **evidence error**, even without a timeout.
**F22-01 stays open** until isolation runs with combined TCP captures and is
judged under D-161; isolation 23 runs for it at cutoff. Historical failures
stay failed. D-145's guard, cadence and the other D-120 predicates stay.
D-161 is a test-acceptance definition, not operator behaviour or a latency promise.

## Members 22 (#467): AT02 measured; two new findings

[PR #467](https://github.com/TED-H-com/Freedom-VPN/pull/467) and the
[frozen dated note](../../tests/handover/testdata/history/follow-up-45/docs_e2e_at_members_followup22-2026-11-04.md.txt)
record the **D-098 archive of run 5715939e**, **98 root records** byte-identical
with SHA-256 pins, followed by the **G2 archive alias** resolving three frozen
run-2 links. Earlier archives and outcomes are unchanged.

Run **3eb3960d** measures **55f2765d, main after #461**, harness **8a6dcf34**:
**3 pass, 7 blocked, 0 fail; 702 passed, 8 blocked and 0 failed subchecks**.
The lane disclosed that retry preflight main also contained #462; the
orchestrator answered that #462 changed only docs and netlab tests, so product
and members code matched and **no later-main rerun is needed**.

| Case | 5715939e → 3eb3960d | Current passed / blocked / failed subchecks |
|---|---|---:|
| AT01 | blocked → blocked; first transfer counted → uncounted; four conditional continuation checks not reached | 19 / 0 / 0 |
| AT02 | pass → pass; early renewal measured below | 193 / 0 / 0 |
| AT03 | pass → pass | 98 / 0 / 0 |
| AT04 | blocked → blocked | 113 / 0 / 0 |
| AT05 | blocked → blocked | 31 / 0 / 0 |
| AT08 | blocked → pass; one native-capture check added | 36 / 0 / 0 |
| AT09 | blocked → blocked | 70 / 8 / 0 |
| AT20 | blocked → blocked | 89 / 0 / 0 |
| AT21 | blocked → blocked | 13 / 0 / 0 |
| AT22 | blocked → blocked | 40 / 0 / 0 |

The count is **705 − 4 + 1 = 702**; no predicate was removed. AT02/AT03's
capture-name UUID substitutions do not change their outcomes.

**AT02 under D-157:** one early renewal **67.8 ms after closed-recovery Apply
586**, then a **one-member lease** and **verified serving Apply 592**. Exactly
one request and no baseline write occur between those Applies. **Deny-all is
205 ms with 8 losses**, down from **4.968 s and 241 losses**. Exact central
readiness timing was not retained; the historical measurement stays intact.

**F21-AT08 is answered:** AT08 passes with the evidence the old run lacked:
paired safe peer views and original native capture, one origin receipt and
both counter increments. The peers move from idle/no handshake to connected
and direct. **The old failure's cause stays unknown and its outcome stays.**
`TestAT08_SelectedPeerSummaryDropsIdentity` and
`TestAT08_MissingSelectedPeerIsNeverConnected` are on main.

**F21-D146's failed condition is known in the new run: freshness.** The later
read finds matching generation/incarnation, forwarding enabled and no block
reasons, but the latest observation is **22.39 s old against 15 s**. The
original combined serving read remains failed. Historical indexes **49 and
51 stay undetermined**. `TestD146_LaterServingReadClassifiesEveryCondition`
preserves that distinction and the strict bound.

**New open findings, as the PR names them:** **F22-AT01**, AT01's first transfer
uncounted on an IPv6 pair (member received no node bytes, neither view reports
a handshake; return-path loss is consistent, not proven); **F22-D146**, the
freshness failure (neighbouring accepted observations are 34.85 s apart; why
reporting stopped or failed is not retained). **Members 23 runs for both.**

Attempt 1 waited 21 minutes, then stopped before Go tests or acceptance: an
unmatched quote in a host command line broke the lane's admission parser with
`ValueError: No closing quotation`. The matching line was not retained.
**8a6dcf34** fixes it with four offline tests; the one allowed retry produced
the run. The recorded foreign window was during teardown, after acceptance;
the lane's D-133 record stays, with no measured-case overlap inferred.

## Devstack 35 (#468): F30-01 closed without a contended acquisition

[PR #468](https://github.com/TED-H-com/Freedom-VPN/pull/468) and the
[frozen analysis](../../tests/handover/testdata/history/follow-up-45/docs_e2e_runs_2026-11-04_followup35_analysis.md.txt)
record **8a436a87: 32 pass, 1 blocked (member-ipv6), 0 fail**. Make exits 2 for
that block; **the unchanged nightly evaluator passes**. Only `update-good`
changes from blocked to pass. `fvpn-updater run` **exited 0 with empty output
and no update lock wait record: its first acquisition succeeded**. Therefore
**no merged change is credited with avoiding devstack 34's collision**, and
**D-158's wait has not run live** in this acceptance path.

**F30-01 is closed by updater 24 #446, agent-core 49 #454 and updater 25 #463**.
The [frozen selected records](../../tests/handover/testdata/history/follow-up-45/docs_e2e_runs_2026-11-04_followup35_recovery-proof.txt.txt)
meet every part of the refined closing condition:

- Parent updater **178784** holds `update.lock` from **09:52:28.339Z** through
  at least **09:52:37.441Z**; candidate **181291** boots **09:52:31.986Z**.
- Candidate boot recovery is deferred: **09:52:32.243Z**, `serving=0`,
  **recovery_required**, while its parent holds the lock.
- The lock is gone by **09:52:37.655Z**, parent exit **09:52:37.471Z**.
  Recovery clears by **09:52:42.112Z**, inferred from `policy_missing`, which
  the code ranks below recovery. No recovery-child exit timestamp is invented.
- The same candidate writes its first serving heartbeat **09:52:42.903Z**,
  generation **95**; the next scenario's counted member flow succeeds.
- **lease-expiry passes afterward**. Forwarding closes at the earlier signed
  desired-policy deadline; this is not a packet-exact or member-lease-only claim.

**update-bad exits 1, HASH_MISMATCH: artifact**, without a wait record.
**Every updater invocation's complete output is retained**, with PID, start,
end and exit status. `TestD158_TwoUpdaterInvocationOutputsSurvive` proves the
retention contract on main.

**D-160 in devstack:** all **30 named sections** are admitted before they start;
**2,041 wrapped commands** run inside with **zero admission checks inside**.
`TestD160_AdmissionPrecedesSectionAndNeverRunsInside` and
`TestD160_HeldBeforeSectionWaitsBeforeStart` are on main. The scheduled workflow
is unaffected: without the admission script configured, the wrapper runs the
command directly and sections record `admission_not_configured`.
The evidence-hygiene tightening scans **decompressed bytes of every committed
.gz**, not only follow-up 27; `TestCommittedFixtureEvidence_ValidAndHygieneClean`
is on main.

## Review r30 and decision boundaries

The [raw archive](../reviews/r30-update-fixes-review.md) is **216 lines**, SHA-256
`b7615f2de08ca6914fb85bd957c00ff7b903dcfa1ac03d229711d1eb38efb933`, from
**r30-update-fixes-review**, head **da2560a9**, base **9b9546c9**. It is a
post-merge D-155–D-158 code review of **#446, #454, #461 and #463**, by
**agy/gemini-4-argon-high**, committed by **Ted H** as the branch records.
It is now a D-098 input. **0 Critical, 2 High, 3 Medium, 0 Low.**

[Dispositions](../reviews/README.md#r30-dispositions-at-the-follow-up-45-cutoff-2026-11-04-est):
**H-1/M-2/M-3 → agent-core 51**; **H-2/M-1 → updater 26**, all open and running
at cutoff. The orchestrator reads H-1 as **not reachable from fvpn-agent today**:
run and once modes pass one context to Boot and Step. It is a **latent fail-open
default in the engine API**, routed for repair anyway. The review's question to
the owner is **none**. Its agent, updater and watchdog proof files are **routed,
not adopted**; none is on main at this cutoff.

There are still **64 decisions**. No new row or row after D-161 exists.
Only D-158 and D-161 change; the other **62 rows remain byte-identical**.
**D-158 clarification — 05:3x EST, 2026-11-04, orchestrator; provisional, the
owner may overturn:** exit 1 after **five consecutive stale answers**; a live
answer, detail `lock`, including empty or unreadable metadata, **resets** the
stale count. The **10 s budget starts at the first attempt** and governs live
answers: never abandon a live holder before it, and past it the first live
answer ends with **exit 3**. Emit **update lock wait only if the first attempt
missed on ErrBusy**. Updater 26 is implementing it, running at cutoff;
**main still counts from the first stale answer**. D-161 is implemented by #465.

The [findings register](README.md#findings-register-at-the-follow-up-45-cutoff)
has **133 rows: 7 updated, 7 added, 119 byte-identical**. F30-01 is closed,
SECOPS-F17-01 and AT02 are answered, AT29 is blocked, F22-01 stays open,
F21-AT08 is answered without a historical cause, and F21-D146 points to the
new freshness finding while historical indexes stay undetermined. Seven new
rows are members F22-AT01/F22-D146 and r30's five findings. Every other open
row is unchanged; every historical failure stays failed.

## Operator and member entrypoints

Seven entrypoints point at this cut. The operator guide keeps the F30-01 and
AT02 anchors and states their measured outcomes in place. Updater 25's D-158
sentence stays verbatim: it is still true on main at this cutoff; updater 26's
later change belongs to a later cut. The TCP reload-guard anchor and wording
are unchanged. Nothing else changes what an operator or member sees, and no
date is promised for unmerged work. Scope-review changes only digest values.

### Requests to orchestrator

**2026-11-04 EST:** carry isolation 23's D-098 archive and complete-capture
D-161 run for F22-01, including netlab's AT10 wording hand-off; members 23's
F22-AT01/F22-D146 diagnosis; updater 26's r30 H-2/M-1 under clarified D-158;
and agent-core 51's H-1/M-2/M-3. Preserve the historical failures, secops'
D-133 record and the absence of live D-158 contention in devstack 35.
The operator still supplies AT29's independently clocked disposable guest.
No new outside-owned pin needs changing; no item is stopped. Other open
operator/resource requests remain unchanged.
