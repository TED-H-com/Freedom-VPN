# Follow-up 31 — first qualified D-140 pass, live set verification and nightly lists

Verified against `main` **3573f396d60836f3f203ab5f883575c96a0ec45c**,
through **#407** on **2026-11-03 EST**. This cut is **#404–#407**, in merge
order: isolation 17 (#404), netlab 27 (#405), devstack 30 (#406), handover 30
(#407). #407 is later than #406; no unbriefed PR merged inside this range.
Nothing was merged after the cutoff at task start. Agent-core 38 (carrying
held 37), updater 20, netlab 28 and members 17 were running or briefed; none
of their results enters this cut. Handover ran no new live acceptance.

**D-098 / D-109 / D-132:** `TestHandover_Followup31Claims` has **21 subtests** and pins
[four final roadmap rows](../../tests/handover/testdata/followup31-main.json),
[nine frozen handover snapshots](../../tests/handover/testdata/followup31-snapshots.json),
[nine frozen nightly inputs](../../tests/handover/testdata/followup31-nightly.json)
and [596 archive and run file checks](../../tests/handover/testdata/followup31-evidence.json).
The three dated evidence/review documents and Go test declarations are pinned
separately. No roadmap row was edited. Isolation 0ec14b39, members 0ffb355a
and usage b8a6e9e1 root records remain living; cite them by roadmap row / PR
only, never as claims-test inputs. Earlier snapshots remain byte-identical.

## Previous cut — handover 30 (#407)

[Follow-up 30](follow-up-30.md) covers **#397–#403 at b9439411**.
`TestHandover_Followup30Claims` on main has **23 subtests / 7 final rows /
9 frozen snapshots / 449 archive and run file checks / 6 dated evidence and
review documents**; its register has **61 findings and 9 owner requests**.
The nine handover snapshots are distinct from the five workflow snapshots
added by its merge-gate fix. Its register prose said ten handover snapshots;
the fixture and test hold nine. The frozen copy preserves that discrepancy.

As #407 records it, agy `claude-opus-5-5-high` wrote the work from **01:06 EST**;
its quota ran out at **01:30:43**. At **02:25**, agy `gemini-4-argon-high`
resumed the same conversation, aligned the test with the fixture's nine
snapshots, added three cut statements, ran the gates and committed. The
admission-wait correction is recorded under Errata below.

## Isolation 17 (#404) — first qualified whole-node pass

**D-098 first:** run 83b457ef's **129 root files and subtrees** are archived
byte-identically under [history/follow-up-16/](../e2e/at/isolation/history/follow-up-16/archive.json).
Handover checks every entry, including both gzip forms; owner proof:
`TestD098_Followup16ArchivePins`.

Run **0ec14b39 on #403's tree**, one assembled run with **no retry**, is
**D-140's first qualified pass**. The corrected formula selected **2.4 ms**
spacing before the run from **103 same-run calibration packets**. Member
send → ingress min / median / max was **0.086 / 1.017 / 2.003 ms**.
D-142 qualification begins at the deciding positive, **packet 166**
(**D−98.978 ms**, **0.977 ms** latency), through the sweep's end. The first
drop is **packet 167 (D−95.770 ms)**. There are **zero overlaps anywhere**.
The byte-identical `EvaluateDeadline` passes: **166 positives / 119 drops**,
with no positive at or after D or after the first drop. Isolation 15's and
16's samples and isolation F14-01 stay failed; a new qualified pass closes
neither historical result.

**D-104:** the fresh mechanism trace passes on netlab 26's fixture,
**26 positives / 7 drops** (`TestD104_AT13_DeadlineContinuity`). Run
83b457ef's original **F16-01 stays failed** after the fixture disposition.

Populated holds through devstack 27's TEST ONLY seam cover the two explicit
incidents and a third reaching central's **30 s age fallback**: **one
escalation at 30.770 s, with no alarm report**. **Eight of nine categories
pass**. Pure `stale_generation` is **not exercised (F17-01)**: today's agent
and central classification cannot produce that report without rewriting one.
This is an acceptance-coverage gap, not a defect; routed to agent-core and
wire-nodes. **AT10 PASS 9 / 0 / 0 (ISO-F6-01 6 / 6)** and **AT12 PASS
10 / 0 / 0**; AT11 and AT13 remain blocked on their operator rows. **No install
exceeded the 100 ms guard.** Root evidence for this run is living until its
owner archives it; the final #404 roadmap row is the claims-test input.

The nightly mapping is below. Scope: ISO01–ISO04, §8.2–§8.3 / §16.3 / §19
AT10–AT13; populated holds support AT29 without passing its full matrix.

## Netlab 27 (#405) — r20 H-2 and live drift closed

Apply verification and periodic live reads now compare the **exact interval
coverage** of `denied_v4` and `host_v4` against the model. Missing, extra or
tainted coverage fails closed with `ISOLATION_POLICY_INVALID`. Interval flags
are mandatory; set and element timeouts are rejected. Baselines require empty,
untainted member admission; bare baselines remove retained sets atomically.

`TestAT13_F27_PeriodicLiveSetDriftWithdrawsServing` proves periodic drift
withdraws serving **without another Apply**.
`TestAT13_F27_UnrecordedApplyRefusesRestartAdoption` proves restart after an
unrecorded Apply takes the baseline. Thus **r20 H-2 and the live-drift reading
are closed**. The operator guide's §3.2 warning is revised; follow-up 30's
frozen copies remain unchanged. Scope: ISO01 / ISO02, §8.1 / §8.2 / §19 AT13.

The D-133 quartet, judged under D-138, is **A1 / B1 / A2 / B2 = PASS / FAIL /
PASS / PASS**. B1 fails **D-120 clause (iv), once**: a TCP read timed out at
**1.501 s**, sent **0.4 ms before generation 1064's guarded install ended**.
The final roadmap disposition judges this **not attributable to 27**: serving
render and install are unchanged between arms; 27's checks only read after
the guard is released. **B1 stays a failed leg; D-120 (iv) is not waived.**
The earlier evidence overview's pending disposition is retained as history;
the final #405 row supplies the later D-138 judgment.

**F27-01**, the TCP failure at the install boundary on the unchanged reload
path, is routed to **netlab 28**. **F27-02**,
`TestAT25_FileMeterWaitRestartRechecksCurrentFile/missing`, exceeded its
one-second assertion in the consumer gate on unchanged updater code; routed
to **updater 20**. Both remain routed, not yet judged by those owners.

The per-follow-up [evidence tree](../poc/G2-followup27-evidence/overview.md)
is pinned under D-098: [final manifest](../poc/G2-followup27-evidence/retention-pins.json)
SHA-256 **`e4ec2e07128d2d8b7208094dfee009df40e9459772a8407a38223aba0011f20d`**,
**446 files**. Historical failed legs stay failed.

## Devstack 30 (#406) — scheduled gate and survivor continuation

D-144 applies to the scheduled e2e `stack` job only. It fails on any fail row,
any error row or non-acceptance test failure, any unlisted blocked row, and
any missing, unreadable or malformed summary or list. A **listed blocked
row keeps it green**, while remaining **blocked** in acceptance records.
The step summary and uploaded `nightly-gate.md` name its owner, reason and
source. A **listed row that passes is reported as unblocked, to be removed
from the list**. Local, pull-request and manual runs remain unchanged.

The committed list is [tests/e2e/nightly_expected_blocked.json](../../tests/e2e/nightly_expected_blocked.json):

| Row | Owner | Reason at this cut |
|---|---|---|
| `member-ipv6` | devstack / platform operators | Blocked by construction: the default native member joins with IPv6 disabled and no IPv6 underlay is provisioned. |
| `lease-expiry` | agent-core; updater co-owner | F30-01, a production defect; see the single operator description in [node update execution §4.3](operator-guide.md#f30-01). Listing it does not accept the behavior. |

The chain's merge gate exposed isolation lint copying only `tests/e2e/*.go`:
the embedded JSON list was missing. **The merge-gate fix reads the list at run
time**, from the module root or `-list`; it is **no longer embedded in package
`tests/e2e`**. Missing, unreadable or malformed lists fail the job
(`TestD144_MissingUnreadableOrMalformedExpectedBlockedListFails`). The
original seven `TestD144_*` checks plus this added check are on main.

Members 16's request is delivered as opt-in TEST ONLY
`Stack.ArmMatchingLeaseHoldSurvivor` in the D-102 seam. It holds only the
eligible survivor's successful matching renewal responses for **one further
generation inside the original window**; it never edits, replays or
synthesizes a signed document. `survivor_mismatch` or a second advance releases
it. Live proof: **8576dfe5, 10 pass / 0 blocked / 0 fail**, one attempt; a real
sibling suspension advances to generation 11, an accepted survivor report
arrives while the incident is open, then recovery closes `generation_matched`.
The dated [run directory](../e2e/runs/2026-11-03/followup30/README.md) has
**21 pinned files**. `TestE2E_Followup30SurvivorHold` is the proof declaration.
**Members 17 is its first consumer and is outside this cut**: the members
acceptance row is not yet exercised by that consumer. Scope: D-102 / D-144,
§16.3 / §18.4 / §19 AT02 / AT13 / AT29; no full acceptance promotion.

## Review r21 — returned before merge

The [raw archive](../reviews/r21-adoption-fixes-review.md) is byte-for-byte:
**29,439 bytes**, SHA-256
**`0fcf888bbb59e8e3032abf041f4d600fcb661525a3c390e15850c61e4df2edc0`**.
It carries no host path, username or secret. agy `gemini-4-argon-high` reviewed
agent-core 37's delivered tip **7b3d7bf8**, which holds fixes for r20's H-1,
M-1, M-2, M-3 and L-1. r21 reports **0 Critical / 2 High / 2 Medium / 1 Low**;
each finding has a proof test on `r21-adoption-fixes-review` (**47f75c04**),
not adopted onto main. Recommendation: **DO NOT MERGE — Return to agent-core
(H-1, H-2, M-1, M-2, L-1)**.

**Disposition: returned to agent-core 38 before merge.** Agent-core 37 is not
on main at this cut; it merges together with 38 after 38 is reviewed.
Agent-core 38 judges each finding before fixing it under D-141; **none of
r21's findings is recorded as confirmed**. Consequently r20 **H-1, M-1, M-2,
M-3 and L-1 remain open on main**. The archive records a review judgment, not
new acceptance evidence. See the [disposition table](../reviews/README.md#r21-dispositions-at-the-follow-up-31-cutoff-2026-11-03-est).

## Nightly scheduled jobs

**Main records no scheduled-run result, so none is claimed.** Both D-144
lists are now committed. E2e's list and gate are described above. Isolation's
[expected-blocked list](../../tests/e2e/at/isolation/expected-blocked.json)
contains exactly:

| Case / row | Owner | Established block |
|---|---|---|
| AT11 / IPv6 platform matrix | acceptance / platform operators | Requires native operator devices and route/capture matrix; this Linux container run does not execute it. |
| AT13 / continuous crash/NetBird/reboot/updater transitions | acceptance / operator | Requires the full fault and physical-device matrix; the container restart row proves only its own transition. |

`ScheduledCaseAcceptable` maps scheduled exits only: a nonempty pass/blocked
record, unique checks, valid positive evidence on pass checks and exact listed
case/row matches on blocked checks. Failures, unknown blocks and real Go test
errors stay red. `nightly_summary.py` also rejects missing/wrong-run/empty
records and failed phases; it reports listed passes as unblocked for removal.
Actual block reasons are logged. The **historical short-expiry block is not
listed**: its cause cannot be recovered from status-only logs. The job stays
red until a scheduled run names that reason; no scheduled outcome is inferred.
Owner proofs: `TestD144_ScheduledExpectedBlocksDoNotHideFailures` and
`TestD144_NightlyReviewAndSuiteAdmission`.

Both jobs write **`$GITHUB_STEP_SUMMARY`**, always upload records and keep them
**7 days**. E2e's records artifact contains **`nightly-gate.md`**; isolation's
contains **`scheduled-job.md`** and **`scheduled-job.json`**. Read the
[operations guide](../ops/README.md#reading-nightly-results-2026-11-03-est).
A red nightly is read, never re-run by an operator or an agent.

## Decisions and findings

**No decision was added after D-144 on main.** D-135, D-136, D-137, D-138,
D-140, D-141, D-142 and D-144 stay provisional. D-143 is Ted's decision.
Every existing decision row is unchanged; the lists implement D-144.

The [findings register](README.md#findings-register-at-the-follow-up-31-cutoff)
has **72 rows**, retaining all 61 previous findings. New rows: r21's five,
F27-01, F27-02, F30-01, F17-01, isolation 17's qualified D-140 sample, and the
members 16 survivor continuation. Netlab 27's B1 remains a recorded failed
leg, not attributable under D-138. r20 H-2 and live drift close; the five
agent-core r20 findings remain open pending held 37 and reviewed 38. Isolation
15/16, F14-01 and F16-01 retain their historical failures. Every other row of
follow-up 30's register is unchanged. **Every historical failure stays failed.**

## Errata

**Follow-up 30 admission timing.** `follow-up-30.md` says #404 and #405 merged
“during this run's host-admission wait”. As #407 corrects the session record,
admission checks ran only from about **01:20 to 01:30 EST**. The session was
stalled on its model quota from **01:30:43 to 02:25 EST**; the two merges fell
in that quota stall. Both merges remain outside follow-up 30's cut either
way. Its statements and frozen snapshots are not edited.

**#401 launch time.** The PR body and squash message say “session launched
10:17 EST on 2026-11-01”. The orchestrator corrects that here: **agent-core 36
launched at 11:10 EST**; **10:17 was agent-core 35's launch**. The old wording
is verified in main's squash message; the corrected launch time comes from
this follow-up's orchestrator instruction. That immutable history is not
rewritten.

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


2026-11-03 — Already routed, not new owner decisions: r21 H-1, H-2, M-1,
M-2 and L-1 → agent-core 38; F27-01 → netlab 28; F27-02 → updater 20;
F30-01 → agent-core with updater co-owner; F17-01 → agent-core / wire-nodes.
Members 17 consumes the delivered survivor continuation. These items stop
with their owners. **Nine owner requests**, unchanged. No outside-owned pin
needed changing.
