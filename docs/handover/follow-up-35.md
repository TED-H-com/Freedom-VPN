# Follow-up 35 — fixed-test reading holds; hitless guard design awaits its owner

Verified against `main` **dff6bdb0ddcf6dfd8392a8db364c0120783a48f3**,
through **#421** on **2026-11-03 EST**. This cut is **#419–#421**, in merge
order: handover 34 (#419), updater 23 (#420), netlab 31 (#421).
No unbriefed PR merged inside the range. Nothing was merged after the cutoff
at task start. Agent-core 39, carrying 37 and 38, was delivered and held;
agent-core 40 is working on r23. Neither is on main. Review r23 was delivered
at **08:07 EST** and returned to agent-core 40 before merge. Review r24 has a
delivered review commit **ba5d3004**, dated **08:19:51 EST**, **after the cutoff**;
it has no merged PR at task start and no result is imported here.
No new live acceptance or timing measurement was run by handover, and no
reservation was taken.

**D-098 / D-109 / D-132:** `TestHandover_Followup35Claims` pins
[three final roadmap rows and three public PR excerpts](../../tests/handover/testdata/followup35-main.json),
[ten frozen #421 handover inputs](../../tests/handover/testdata/followup35-snapshots.json),
and [updater 23's two dated documents, r23 and every file in netlab 31's design tree](../../tests/handover/testdata/followup35-evidence.json).
Earlier claims retain their assertions and read frozen #421 bytes where this
cut rewrites their inputs. Members 81b638e8, isolation 0ec14b39 and usage
b8a6e9e1 roots remain living; none is a claims input.

## Previous cut — handover 34 (#419)

[Follow-up 34](follow-up-34.md) covers **#416–#418 at cc0ed4ae**.
The actual `TestHandover_Followup34Claims` and manifests on main verify:
**14 subtests / three final rows / ten frozen handover inputs / 312 netlab
payload hashes and lengths, including 19 gzip originals / updater 22's two
dated documents / three PR excerpts / 81 findings / 47 decisions / nine
owner requests**. The previous test's normalized function hash preserves
its assertions as its mutable inputs move to the #421 archive.

[PR #419](https://github.com/TED-H-com/Freedom-VPN/pull/419) records **one codex
`gpt-6-astra` session**, launched **07:29 EST**, with admission and suite waits
of **0 s**. Its control-count request is answered in that PR: the correct
count is 15. The additive correction is in #420; #417's final row and
follow-up 34's historical pins remain unchanged.

## Updater 23 (#420) — F27-02's fixed-test reading HOLDS

The [dated hand-off](../updates/node-followup23-2026-11-03.md) and
[timing JSON](../updates/node-followup23-timing-2026-11-03.json) retain the
precommitted declaration **f770117d**: **the leg ends when the measured test
process exits**. Every interval starting before that exit decides qualification;
30 seconds of later samples are recorded only. Both meter fixture files were
byte-identical to updater 22's tip **5eea3f02** before both reservations and
after the second.

| Bounded-load leg | Iterations / failures | Qualification and foreign CPU |
|---|---|---|
| Reservation 1 | 20 / 0 | Unqualified, stopped by non-lane foreign CPU at 0.996 cores; two additional Recover calls in the interrupted iteration are retained. |
| Reservation 2 | 50 / 0 | Qualified; maximum 0.105 cores in deciding intervals, 0.385 after exit. |

Reservation 2 pools with updater 22's **qualified quiet leg, 200 / 0**.
The deterministic budget is exact before and after the window, and in **all
352 logged parent Recover calls**: remaining 250 ms except expired at −1 ms;
bubble elapsed 250 ms for missing/replaced and zero for the other cases.
The independent restart check remains **250 / 50 / 0 / 0 ms**.
**F27-02 is a fixture defect with F21-01's mechanism, fixed by #417; its
fixed-test reading is confirmed by #420 under the declared rule.**
Updater 21's failed verdict, updater 22's undecided verdict and netlab 27's
failed run stay recorded. Every unqualified leg stays unqualified; no
whole-Recover production SLO follows from these tests.

**F23-01 is a fixture defect, fixed by #420.**
`TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires`
depended on its own first read and clock calls finishing inside an **80 ms
real-clock window** before the scripted rewrite. A **250 ms stall** in its
clock fails before the fix and passes after. The fix uses a **synctest bubble
clock**, with the **window, poll interval, TTL and assertion unchanged**.
A first-attestation mutation still fails the fixed test. The whole-package
stall control passes **all 176 tests**, with **88 stalls in 28 tests/subtests**.
**No product change.** The 30 s and longer windows and the update-journal
snapshot guard remain recorded observations, not newly proved defects.

[PR #420](https://github.com/TED-H-com/Freedom-VPN/pull/420) records **two
sessions**: agy `claude-opus-5-5-high` ran **07:05–07:30 EST**, committing the
declaration, running both reservations and diagnosing F23-01 until its model
quota ran out at **07:30 EST**. Codex `gpt-6-astra` ran **07:31–07:40 EST**,
completed the work from the retained records and reran controls and gates,
with **no new reservation or measurement**.

**Erratum to #417:** the count is **15 failing cases**, all
meter_checkpoint_timeout: **the parent test's replaced, four
FailsClosedWithoutCorrelation scenarios and ten TestD125_Rule tests**.
The old count of 16 came from counting eleven Rule tests. **26 counts test
and subtest results: 24 cases plus two parent tests.** #420's final roadmap
row and PR carry this correction; #417's row stays as merged.

## Netlab 31 (#421) — design only

The [six-file design tree](../poc/G2-followup31-design/hitless-guard-design.md)
is pinned file by file by SHA-256 and length. Its
[current guarantees](../poc/G2-followup31-design/current-guarantees.md) start
from the code, including its failure limits and crash cuts:

| Guarantee | Current behavior and source |
|---|---|
| G1 deadline | Guard prevents candidate admission before budget approval; approved countdown cannot exceed the signed/anchored deadline. [Apply](../../node/enforce/apply.go), [deadline guard](../../node/enforce/deadline_install.go). |
| G2 revocation | Candidate commit closes both member directions; after release, the new allowset rejects revoked addresses before established/related accepts. [Renderer](../../node/enforce/render_nft.go). |
| G3 baseline | No adoptable verified same-boot generation means generation 0 and member closure, with control retained and only a separately bounded synthetic probe exception. [Engine](../../node/agent/engine.go), [baseline renderer](../../node/enforce/render_nft.go). |
| G4 rejection | Early errors can precede a closure attempt; later failures attempt baseline/quarantine. An Apply error alone does not prove closure. [Apply](../../node/enforce/apply.go), [serving](../../node/agent/serving.go). |
| G5 kill/crash | From guarded commit until guard deletion, every member is closed even if the agent dies. Before commit old state may remain; after deletion safe candidate timers remain. [Deadline guard](../../node/enforce/deadline_install.go). |
| G6 late commit | Timeout, error, negative elapsed or over-100 ms install prevents release and attempts baseline. A late committed candidate stays guarded even if cleanup never runs. [Apply](../../node/enforce/apply.go). |
| G7 verification | Guard release precedes live nft/tc verification; verification failure closes via baseline/quarantine. An unverified candidate may briefly forward. [Apply](../../node/enforce/apply.go), [verifier](../../node/enforce/verify.go). |
| G8 rollback | Node-wide baseline, never old-policy restoration; counter retention and owned tc cleanup. Rollback can fail, so no finite outage or infallible cleanup guarantee. [Apply](../../node/enforce/apply.go), [renderer](../../node/enforce/render_nft.go). |
| G9 isolation | Atomic old/new nft policy with both member directions closed during the guard. Foreign tc requires quarantine. [Renderer](../../node/enforce/render_nft.go), [quarantine](../../node/enforce/quarantine.go). |
| G10 accounting/QoS/control | Named counters retained; denied packets are not ordinary usage. tc is sequential and precedes nft, so partial shaping can survive a crash; host control remains independent. [Renderer](../../node/enforce/render_nft.go), [tc renderer](../../node/enforce/render_tc.go). |

The ledger enumerates death before install, after commit, before release,
after release but before verification, during failed rollback and during
conntrack/durable ACK work. It does not equate a killed command with proof
that no transaction committed. Netlab 30's measured medians remain **13.7 ms
install / 13.2 ms release / 28.5 ms outer guard**, outer maximum **94.7 ms**.
The seven attributed drops and worst recovery **416.7 ms** remain observations;
both live runs remain host-unqualified. **B1 stays FAIL and unexplained**.

The [candidates](../poc/G2-followup31-design/candidates.md) are:

- **C1: retained-deadline intersection guard.** Already verified, unchanged
  identities are capped by the old timed elements, never refreshed or copied.
  Revocations commit with the full candidate; other members wait for release.
- **C2: staged inactive generation with one atomic switch.** It adds
  revocation delay; changed QoS still requires today's closure.
- **C3: absolute kernel deadline, rejected.** The design finds stock nft
  expires relative, with no proven absolute monotonic primitive in that interface.

**No candidate keeps today's all-member closure after a late commit, or a
kill or stop before the budget check, while being hitless.** C1 lets eligible
members continue to their **unchanged old deadline**. A stopped lock holder
can defer watchdog closure until then. This is the **G5/G6 trade**, with the
**G4/G8 failure-containment difference** while baseline is pending or fails.
It grants no blanket established-flow exemption and never turns failed Apply
evidence into success. **C1 is recommended for design review only. Production
keeps the current guard; nothing is implemented.**

The [proof plan](../poc/G2-followup31-design/proof-obligations.md) and
[D-133 A,B,A,B comparison](../poc/G2-followup31-design/d133-comparison.md) are
**declared, not run**. The rule requires four qualified legs, owner resolution
and proofs/consumers complete, unchanged D-120 and AT13, no new unauthorized
admission or B-only failure, zero eligible-member observed loss, no later
paired revoke cutoff, improvement in both pairs, and the declared cost budget.
Its proposed trial criteria are not production SLOs. Future test names in
that plan are proposals, not existing tests on main.

[PR #421](https://github.com/TED-H-com/Freedom-VPN/pull/421) answers the requests:
the trade is **the owner's decision**. Agent-core provenance/adoption and
updater, installer and acceptance work **wait for the owner's acceptance**;
nothing is scheduled now. The quiet D-133 slot follows **design review and
passing implementation proofs**. **Review r24 reviews the design first**,
before any implementation. F27-01 remains unresolved, with no fix.

## Review r23 — archived, returned before merge

The [raw archive](../reviews/r23-intent-fixes-review.md) is byte-for-byte:
**41,791 bytes**, SHA-256
**23f899f8e0b834994785e8c8a1f774b5aeb1a18309bf35439ba1a4f3261b25d0**.
It carries no host path, username or secret. **agy `gemini-4-argon-high`**
reviewed delivered agent-core 39 tip **99f93112**, carrying 37's r20 fixes,
38's r21 fixes and 39's r22 fixes. Findings: **0 Critical, 3 High, 1 Medium,
1 Low**, on review branch **r23-intent-fixes-review**, proof commit **26ffc36c**.
The archive gives executable proofs for H-1, H-2, H-3 and M-1; **L-1 is a code
trace, not a named proof test** in that commit. The brief's all-five-test
statement needs clarification; no proof is invented here.

Recommendation: **Do not merge follow-ups 37, 38, and 39 as they are; merge
after `agent-core` fixes `H-1` (`R23-H-1`), `H-2` (`R23-H-2`), `H-3`
(`R23-H-3`), `M-1` (`R23-M-1`), and `L-1` (`R23-L-1`).**

**Returned to agent-core 40 before merge.** Agent-core 37–39 are not on main;
they merge together with 40 after its fixes. D-141 requires owner judgment
before fixing; **none of r23's five findings is confirmed**. r20's five
agent-core findings and r21's five stay open on main. r22's five dispositions
stay exactly as follow-up 33 recorded them. No held-branch judgments or proof
files are adopted.

The [reviews index](../reviews/README.md#r23-dispositions-at-the-follow-up-35-cutoff-2026-11-03-est)
records the What held errata without editing the archive: actual renderer and
forward-chain generation marker, persistent counters and the epoch-baseline
limit, actual stop-budget test location, and nonexistent cited proof names.
These corrections add no acceptance claim; agent-core 40 checks the epoch
baseline argument for removed and re-added devices.

## Current guidance, register and decisions

Seven entrypoints point at this cut. Member TCP segments can still be dropped
while a generation installs. TCP recovered by retransmission in netlab 30's
window; there is no guaranteed recovery bound and **no hitless hand-over is
implemented**. **F30-01 stays known behaviour, no merged fix.** The restart
warning remains: **r20 H-1's gap stays open on main while agent-core 37–39 are
held**. Main records **no scheduled-run result**.

The [findings register](README.md#findings-register-at-the-follow-up-35-cutoff)
has **87 rows**: F27-02, F21-01 and F27-01 updated; F23-01 and r23's five added.
**Every other follow-up 34 finding row is byte-identical. Every historical
failure stays failed.** All **47 decision rows** are unchanged; **no decision
was added after D-144 on main**. **D-135, D-136, D-137, D-138, D-140, D-141,
D-142 and D-144 stay provisional; D-143 is Ted's decision.** Nine owner
requests are carried verbatim, with netlab 31's owner trade as the tenth.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**.

2026-11-01 — **Owner decision 2: D-137 hardening option**; keep the failed
receive-only MTU-1280 limitation without upgrade or waiver.

2026-11-01 — **Owner decision 3: D-138 overturn option**.

2026-11-01 — **Owner decision 4: D-140 overturn option**; unchanged evaluator,
failed qualification and historical F14-01 retained.

2026-11-01 — **Owner decision 5: D-141 overturn option**, retaining its witness,
qualifying-share, promotion-cost and design-finding prerequisites.

2026-11-01 — **Owner decision 6: F11-01 repair direction and upstream report**,
including devstack 27's `::1` data and handshake-response evidence. Members 16's
5.222 s first SYN adds no cause; F11-01 / F14-01 stay unproven.

2026-11-01 — **Owner question 7: RELATED ICMP from spec-denied outer sources**
under §8.1 / §8.2 / §19 AT13. Keep the fail-closed source guard pending an owner
decision; no exception, defect disposition or fix is promised.

2026-11-03 — **Owner decision 8: D-142 overturn option**; isolation 15's and
16's samples stay failed qualification either way.

2026-11-03 — **Owner decision 9: D-144 overturn option**; acceptance status,
pull-request and local runs are unchanged either way.

2026-11-03 — **Owner question 10: netlab 31's G5/G6 trade** — whether G5/G6
require today's all-member closure at an interrupted or late candidate commit,
or whether only already verified, unchanged identities may continue to their
immutable old deadline. Production keeps the current guard pending the owner's
acceptance; r24 reviews the design first.

2026-11-03 — **Review-proof clarification, outside owned paths:** the r23 brief
says each finding has a proof test. The exact archive and proof commit 26ffc36c
supply executable proofs for four findings; L-1 is a code trace. Please confirm
that evidence class or route an additive proof from the review/agent-core owner.
No foreign test or archive byte is edited. This item alone remains outstanding.

2026-11-03 — #419's control-count request is answered additively by #420;
no outstanding count request remains. Agent-core 40 judges r23 before fixes
and before the combined 37–40 merge. F30-01 remains agent-core/updater and
F31-01 remains agent-core, design first. The ten owner requests are distinct
from the evidence clarification above.
