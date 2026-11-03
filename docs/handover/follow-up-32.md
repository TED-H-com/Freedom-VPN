# Follow-up 32 — survivor continuation, fixture diagnosis and two routed findings

Verified against `main` **b7449ec1931a5cc954862908690969cb8bb2305d**,
through **#412** on **2026-11-03 EST**. This cut is **#408–#412**, in merge
order: updater 20 (#408), handover 31 (#409), members 17 (#410), netlab 28
(#411), devstack 31 (#412). #412 is later than #411. No unbriefed PR merged
inside this range. Nothing was merged after the cutoff at task start.
Agent-core 38 (carrying 37, under review r22), updater 21 and netlab 29 were
running or briefed; their results are outside this cut. Handover ran no new
live acceptance and took no timing reservation.

**D-098 / D-109 / D-132:** `TestHandover_Followup32Claims` has **20 subtests** and pins
[five final roadmap rows](../../tests/handover/testdata/followup32-main.json),
[nine frozen handover inputs](../../tests/handover/testdata/followup32-snapshots.json),
[301 archive files, seven dated/additive documents, 19 symlinks and 14 archive navigation sources](../../tests/handover/testdata/followup32-evidence.json),
and [#409's public session excerpt](../../tests/handover/testdata/history/follow-up-32/pr409-session.txt).
All 18 members archive symlinks and the checklist symlink resolve inside the
repository, with target bytes pinned. Earlier claims keep their assertions;
follow-up 31's live reads now use frozen copies of this cut's main bytes.
Run 81b638e8's root files are living, as are isolation 0ec14b39 and usage
b8a6e9e1: cite final roadmap rows / PRs only, never use those roots as test inputs.

## Previous cut — handover 31 (#409)

[Follow-up 31](follow-up-31.md) covers **#404–#407 at 3573f396**.
The test on main verifies **21 subtests / 4 final rows / 9 frozen handover
snapshots / 9 frozen nightly inputs / 596 archive and run file checks /
3 dated documents**, with **72 findings and 9 owner requests**. The 596 checks
are 129 isolation, 446 netlab and 21 devstack files. These counts are read from
the actual previous function and manifests, not inferred from its prose.

#409 records **one codex `gpt-6-astra` session**, launched **03:07 EST**. It
wrote docs and tests during the host hold and ran its gates after admission
printed free at **03:55 EST**, following a **39-minute admission wait**.
The frozen PR excerpt preserves that public record.

## Updater 20 (#408) — F27-02 fixture defect

The one-second assertion in `TestAT25_FileMeterWaitRestartRechecksCurrentFile`
timed all of **Recover**, including lock work, sealing and cleanup; it did not
measure the persisted meter-wait deadline it claimed to guard. The unchanged
one-second ceiling now bounds **the wait itself**, deterministically, in
`TestAT25_FileMeterWaitRestartKeepsDeadline`. Remaining budgets are
**250 / 50 / 0 / 0 ms** after **0 / 200 / 250 / 1000 ms** downtime; the persisted
note is unchanged. The full recovery test retains its error, receipt and
forwarding assertions and logs the whole-call duration.

Controls separate the two properties: a **1.1 s sealing delay** fails the old
assertion while the corrected test passes; a **one-minute renewal mutation**
fails all four deterministic cases. No production timeout or runtime change
was made. **Four 200-iteration runs were retained, none qualified**, and the
load leg never started. The qualified quiet / load comparison remains open
with updater 21. Historical netlab 27 **F27-02 stays failed**, and its individual
latency source stays unproven.

For **F30-01**, under a live apply, `recover` exits **1** with **UPDATE_BUSY**
after five lock attempts. The candidate cannot distinguish its parent's live
apply from a leftover journal. A transaction-bound **apply-in-progress signal**,
with D-064 TxnID binding, is **proposed, not implemented**. No fix has merged;
[the existing operator description](operator-guide.md#f30-01) remains known
behaviour, without a promised procedure or fix date.

Pins: [dated hand-off](../updates/node-followup20-2026-11-03.md),
[timing JSON](../updates/node-followup20-timing-2026-11-03.json), final #408 row,
and both Go test declarations in `node/updater/meter_barrier_test.go`.
Scope: UPD03 / UPD05 / MTR03, §13.5–§13.9 / §19 AT25–AT26; component checks,
not new live acceptance or physical arm64 evidence.

## Members 17 (#410) — first live D-102 survivor continuation

**D-098 first:** run **0ffb355a** is archived byte-for-byte under
[history/follow-up-16/](../e2e/at/members/history/follow-up-16/SHA256SUMS):
**81 root files**, each SHA-256 pinned. **18 relative symlinks** restore the
archive's own links without editing pinned bytes. Each resolves inside the
repository. The additive [publication mapping](../e2e/at/members/history/follow-up-16/PUBLISHED-COMMITS.md)
records **3a8b6159 → 5d9e5f04**, **cbd59ccb → 876c0558**, and
**7852fba6 → f97da33d**; the pre-publication redaction changed one receipt field,
not the archived measured IDs or outcomes.

One assembled run **81b638e8**, measured on **#406**, **no retry**:

| Result | Cases |
|---|---|
| 2 PASS | AT03, AT08 |
| 6 BLOCKED | AT01, AT04, AT05, AT20, AT21, AT22 |
| 2 FAIL | AT02, AT09 |

Subchecks: **691 pass / 7 blocked / 5 fail**. These living run results are
pinned only through the final [#410 roadmap row](../ROADMAP.md).

**D-102 survivor continuation is evidenced live for the first time.** Members
17 is the first consumer of devstack 30's `ArmMatchingLeaseHoldSurvivor`:
sibling suspension, the survivor's accepted report in the **same open
incident**, then recovery at generation 11. This closes that consumer coverage
row. It does not close the separate source-address populated-hold row.
**AT09's 401 did not recur after a fresh OIDC sign-in**; follow-up 16's cause
stays unknown. Historical failed records are unchanged.

**AT02 still fails:** three failed Applies during its bounded TC stimulus leave
provisional availability **measurement_valid=false**. **F11-03 / D-135 / D-136
fail**, with **all 1,253 losses counted**; the first re-grant's counted forwarding
also fails. **AT09 still fails:** both the **D-067 provisional predicate** and
the combined same-generation restart fail, with **serving readiness 6.01 s**
and **counted recovery 12.70 s**, at **generation 63**. The receive-only dual-MTU
row is **blocked**. No closure or new recovery promise follows from these figures.
The separate source-address row remains **not exercised**, as devstack 31 explains.

Go proof names on main: `TestD098_Followup16Archive`, `TestATMembers_Live`,
`TestAT02_F12RecoveryRequiresNextApplyAndCountsEveryLoss`,
`TestAT02_F13D136HistoricalPolicyMissingCountsLosses`,
`TestAT09_D126CapturedProvisionalSequence`; producer proof:
`TestE2E_Followup30SurvivorHold`. These declarations identify coverage; the
#410 row supplies the live verdict. Scope: §8.2–§8.3 / §16.3 / §19 AT02, AT09
and D-102; no complete platform matrix or AT29 claim.

## Netlab 28 (#411) — candidate mechanism, B1 still failed

**F27-01 stays unresolved; B1 stays FAIL.** One predeclared **TEST ONLY**
diagnostic window used opt-in TCP observation: **100 reloads / 101 verified
Applies / 118 TCP reads**, with **zero TCP failures**. It saw **11 retransmissions
on seven reads**, every missing node egress inside an install / release bracket,
and RTO backoff before recovery. This is a recorded **candidate mechanism**;
it **does not explain B1**. Exact packet drop-rule attribution is not established.
Another lane began a unit test in the final roughly 1.4 s of the read window;
no uncontended whole-window or source-matched performance claim is made.

The separate full-lab gate **failed AT12**:
`TestAT12_ISO01_ISO04_AllowDeny` saw deny-phase usage **16,243 → 16,340 bytes
(+97)** on unchanged AT12 and production code. Retained **without retry**,
unattributed: **F28-01 → netlab 29**, routed, not yet judged. This gate failure
neither closes B1 nor changes the diagnostic's D-120 result.

**Publication mapping, as the final #411 roadmap row records it:** before
publication, the orchestrator rewrote unpushed commits to remove a private
location from one evidence tool line. **b398b5a0 → c91eb3d5** is the measured
harness; **20ad4553 → aa25dd8a** is the delivery. Declaration **ef58364c is
unchanged**. Two evidence files, `overview.md` and `window/claim.json`, still
name **b398b5a0**; read it as **c91eb3d5**. The final manifest on main is:

```text
dda5a42a27bd1e215aeb7384d71cdb5ef69286b9bfa873172d318138749f8e5d
```

The [per-follow-up tree](../poc/G2-followup28-evidence/overview.md) and its
[220-file manifest](../poc/G2-followup28-evidence/retention-pins.json) remain
unchanged. Netlab 29 is to add its own additive mapping note after this cut;
no such result is adopted here. Test names: `TestAT12_ISO01_ISO04_AllowDeny`,
`TestAT10_AT13_ReloadRevokeBaseline`, `TestD104_AT13_DeadlineContinuity`.
Scope: ISO01 / ISO04, §8.1–§8.2 / §19 AT10, AT12–AT13, D-120 / D-133 / D-138.

## Devstack 31 (#412) — source-binding feasibility and archive navigation

Members 17's source-binding fault request **stopped on feasibility**: **no hook
was added and zero stack attempts** were made. Verified on main,
`bindingOK` in `node/agent/decide.go` comes from constructing the model from
signed binding and lease inputs. It is **not a live source-address check**.
Both false paths also fail `qos`; retained Apply errors fail lease, qos and
source_binding together. During a generation hold, `source_binding` already
fails through `leaseOK`, so a source-only restoration cannot be reported there.

**F31-01 → agent-core, design first**, routed, not yet judged: define genuine
source-binding verification with its own cause-specific reporting, separate
from Apply / qos and lease dependence. The separate source-address populated-hold
row stays **not exercised**. This is a verification/reporting gap, not a claim
that packet-level source isolation is absent. A future devstack hook needs
that design and fresh-stack proof.

The relative symlink `docs/e2e/at/poc/G1-app-checklist.md` resolves to
`docs/poc/G1-app-checklist.md`. It repairs the checklist reference in **all 14
archived members OPERATORS.md files (follow-ups 3 to 16)** without changing
pinned bytes. The symlink, target hash and every source digest are checked.

Pins: [dated run directory](../e2e/runs/2026-11-03/followup31/README.md), including
[source analysis](../e2e/runs/2026-11-03/followup31/source-binding.md) and
[archive link verification](../e2e/runs/2026-11-03/followup31/archive-links.md),
final #412 row, and `TestCommittedFixtureEvidence_ValidAndHygieneClean`.
Scope: ISO02, §8.2 / §10.2 / §16.3 / §19 AT02, AT13; no new live result.

## Operator guidance, decisions and boundaries

Seven entrypoints point here. The operator guide retains F30-01 as written in
follow-up 31; no fix has merged. Its older **36.976 s / 41.695 s** restart values
are explicitly historical members 6 measurements. The current supported
measurement on main is members 17's **6.01 s / 12.70 s at generation 63**, a
**failed combined restart**, never a promised bound. Main records no
scheduled-run result, so none is claimed. Secops owned-document digests are
refreshed separately from immutable evidence pins.

**No decision was added after D-144 on main.** D-135, D-136, D-137, D-138,
D-140, D-141, D-142 and D-144 stay provisional; **D-143 is Ted's decision**.
All 47 decision rows and nine owner requests remain unchanged. The findings
register retains all 72 previous rows, updates five, and adds F28-01, F31-01 and
members 17's D-067 / AT09 result: **75 findings**. Every other follow-up 31 row
is byte-identical. Every historical failure stays failed.

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


2026-11-03 — Already routed, not new owner decisions: F28-01 → netlab 29;
F31-01 → agent-core, design first; F27-02 qualified comparison → updater 21.
F27-01 remains with netlab; F30-01 remains with agent-core / updater. r21 and
r20 agent-core items await reviewed agent-core 38; F17-01 stays with agent-core /
wire-nodes. Members 17 closes survivor-consumer coverage only. All outside-owned
implementation stops with its owner; no outside-owned pin needed changing.
**Nine owner requests**, unchanged.
