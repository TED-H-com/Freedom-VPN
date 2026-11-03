# Follow-up 33 — fixture comparison failed, AT12 isolation fixed, r22 returned

Verified against `main` **339db17215db583330a2395f0484f2bdad9681e7**,
through **#415** on **2026-11-03 EST**. This cut is **#413–#415**, in merge
order: updater 21 (#413), handover 32 (#414), netlab 29 (#415).
No unbriefed PR merged inside the range. Nothing was merged after the cutoff
at task start. Agent-core 39 (carrying 37 and 38), updater 22 and netlab 30
were running; their results are outside this cut. No new live acceptance or
timing measurement was run by handover, and no reservation was taken.

**D-098 / D-109 / D-132:** `TestHandover_Followup33Claims` pins
[three final roadmap rows and public PR excerpts](../../tests/handover/testdata/followup33-main.json),
[ten frozen handover inputs](../../tests/handover/testdata/followup33-snapshots.json),
[512 netlab payloads, two dated updater documents and the raw r22 archive](../../tests/handover/testdata/followup33-evidence.json).
Earlier claims keep their assertions and read frozen #415 bytes where this
cut rewrites their inputs. Members 81b638e8, isolation 0ec14b39 and usage
b8a6e9e1 roots remain living; none is a claims input.

## Previous cut — handover 32 (#414)

[Follow-up 32](follow-up-32.md) covers **#408–#412 at b7449ec1**.
The actual `TestHandover_Followup32Claims` and manifests on main verify:
**20 subtests / five final rows / nine frozen handover inputs / 301 archive
payload checks (81 members, 220 netlab) / seven dated or additive documents /
19 relative symlinks inside the repository / 14 archived navigation sources**.
Its register has **75 findings and 9 owner requests**.

[PR #414](https://github.com/TED-H-com/Freedom-VPN/pull/414) records **one codex
`gpt-6-astra` session**, launched **05:22 EST**. Its repeat gate waited
**7 minutes**, **05:33–05:40 EST**, behind **netlab 29's reservation**, with no
test inside that reservation. The public session excerpt is frozen with its
source squash-commit provenance; the count comes from the code and manifests.

## Updater 21 (#413) — declared reading failed; F21-01 is the mechanism

**F27-02's fixture-defect reading failed by the rule fixed before measuring:**
any corrected-leg failure fails that reading, qualified or not. The corrected
parent test failed once. The original F27-02 reading is **supported, not
closed**; the comparison does not turn the historical failure into a pass.

**F21-01** is a second wall-clock dependency in
`TestAT25_FileMeterWaitRestartRechecksCurrentFile/replaced`. Its own setup
(simulated crash, SQLite reopen and replacement checkpoint) took about
**533 ms against the 250 ms persisted deadline**. `Recover` itself took
**15.1 ms** and found the deadline already expired by **283.1 ms**. It failed
closed with **meter_checkpoint_timeout** instead of
**meter_checkpoint_uncorrelated**: forwarding blocked, null-usage receipt.
This was **not a renewal**. The deterministic budget was exact in every sample:
**250 / 50 / 0 / 0 ms** after **0 / 200 / 250 / 1000 ms** downtime, before and
after the windows. The corrected failure preceded the foreign process by
about **90 s**.

| Leg | Result and qualification |
|---|---|
| Quiet baseline | 200 / 1, then 200 / 0; neither qualified, one in-leg vmstat interval each (r = 4–5) with the test's own process counted. |
| Quiet corrected | 200 / 0, then 178 completed / 1; neither qualified. First disqualified by one vmstat interval; second stopped by a foreign go test. |
| Load baseline | 50 / 0, qualified. |
| Load corrected | 50 / 0 twice; first unqualified by a foreign gate wrapper, second qualified. |

The old one-second assertion failed once on the quiet host: **expired,
1443.3 ms**. Whole-`Recover` stalls of **0.27–1.44 s** occurred both quiet and
under load. Neither quiet leg ever qualified; both load legs qualified.
The JSON retains the first load-corrected leg's corrected disqualification.
Netlab 27's F27-02 and updater 20's four unqualified runs stay as recorded.
No production or test source changed in #413.

[PR #413's answered requests](https://github.com/TED-H-com/Freedom-VPN/pull/413)
record reservation enforcement: newly launched lane sessions' `go test` refuses
to start until admission prints free. Updater 22 uses a **foreign-CPU quiet-leg
rule** (processes not descended from the runner); vmstat remains recorded but
no longer decides. These answers do not requalify updater 21's legs.
**F21-01 → updater 22**, fixture fix and re-measurement, also covering every
other short-budget updater test. No result from that assignment enters this cut.

Pins: [dated hand-off](../updates/node-followup21-2026-11-03.md),
[timing JSON](../updates/node-followup21-timing-2026-11-03.json), final #413 row,
and Go declarations `TestAT25_FileMeterWaitRestartRechecksCurrentFile` and
`TestAT25_FileMeterWaitRestartKeepsDeadline`. Scope: UPD03 / UPD05 / MTR03,
§13.5–§13.9 / §19 AT25, component evidence only.

## Netlab 29 (#415) — F28-01 closed as test isolation

**F28-01 is closed as a test-isolation defect**, attributed packet by packet
before the fix. The first reserved full-suite run failed AT12 by **97 upload
bytes in two packets (15,533 → 15,630)**. Both came from AT10's earlier TCP
connection to an admitted destination: a **57-byte retransmitted request**
and a **40-byte RST**. Each was matched independently at **origin, node
ingress, node egress and responder**, strictly between the **before and mid
reads**. AT12 entered with that socket still in **FIN-WAIT-1**, Send-Q 26.
Flushing node conntrack does not clear the member namespace's retransmit
state. The bytes were forwarded under a **re-admitted binding**; this is
**not billing of refused packets**.

The fix destroys and verifies prior connected IPv4 TCP sockets in AT12's
lab namespace **before probes and baseline sampling**. The deny assertion is
**byte-identical**, with no subtraction, reset, tolerance or settling sleep.
Both corrected full suites found a pending old socket and counted **zero
deny-phase bytes**: **16,669 → 16,669** and **14,349 → 14,349**. Production
paths are unchanged. **Follow-up 28's gate 2 stays FAIL**; its original packets
were not captured. Reproducing the mechanism cannot recover those identities.

**F27-01 stays unresolved, and B1 stays FAIL.** The single guard diagnostic
failed at capture startup: the **NFLOG backend rejected nanosecond timestamps**.
It was **not retried**. The microsecond parser was corrected offline only;
there is no live collector validation, exact guard-drop attribution or
measured diagnostic cost. **Netlab 30** owns collector validation and the next
guard window, outside this cut.

Follow-up 28's [publication mapping](../poc/G2-followup29-evidence/publication-mapping.md)
is additive, with **no follow-up 28 file edited**: declaration **ef58364c
unchanged**, measured harness **b398b5a0 → c91eb3d5**, delivery
**20ad4553 → aa25dd8a**. The harness changed only the admission tool line to
read `FVPN_HOLD_SH`; the delivery also changed that file's manifest entry.
The executed fixed location was in the orchestrator's private repository.
The manifest changed from
**0f09a17da7da53c81ba3094570f0bf2d4dd9e832b27671fd743a4a9fd872a66f** to
**dda5a42a27bd1e215aeb7384d71cdb5ef69286b9bfa873172d318138749f8e5d**.
The original overview and window claim retain b398b5a0; its published
counterpart is c91eb3d5.

The [follow-up 29 evidence tree](../poc/G2-followup29-evidence/overview.md)
retains three full suites and one failed guard attempt. Its final
[manifest](../poc/G2-followup29-evidence/retention-pins.json) pins **512 payload
files**, SHA-256
**d9f9e7d539a884c113a461646448539343c1da803cf9dc378c4e0ab59dbddefa**.
Go declarations on main: `TestAT12_ISO01_ISO04_AllowDeny`,
`TestAT10_AT13_ReloadRevokeBaseline`, `TestF27_GuardLogKeepsPredicatesAndDrops`,
`TestCommittedEvidenceFiles_ValidAndHygieneClean`. Scope: ISO01 / ISO04 /
DNS03, §8.1–§8.2 / §19 AT12; MTR01, §10.1–§10.2. No broader acceptance,
physical arm64 or production logging-cost claim.

## Review r22 — returned to agent-core 39 before merge

The [raw review](../reviews/r22-attribution-fixes-review.md) is byte-for-byte:
**37,600 bytes**, SHA-256
**05e7806a1f5dfec8388e4fe2cbe0e5e1ed11ec1d4c291f3bb5236d7c0fe43c02**.
It carries no host path, username or secret. Reviewer **agy
`gemini-4-argon-high`** reviewed agent-core 38's delivered **31862b3a** before
merge. That tip carries 37's r20 fixes and 38's fixes for r21 **H-1, H-2,
M-1, M-2 and L-1**. Findings: **0 Critical, 2 High, 3 Medium, 0 Low**, each
with a proof test on **r22-attribution-fixes-review**, **ef78b9b7**.
Recommendation: **Hold for fixes: H-1, H-2, M-1, M-2, M-3.**

**Returned to agent-core 39 before merge.** Agent-core 37 and 38 are not on
main at #415; they merge together with 39 after 39 is reviewed. Under D-141,
39 judges each finding before fixing it; **none is confirmed**. r20 **H-1,
M-1, M-2, M-3 and L-1**, and r21's five findings, therefore stay **open on
main**. Their historical register rows are retained; this paragraph updates
the routing, not their disposition. No review proof file is adopted here.

The [reviews index erratum](../reviews/README.md#r22-dispositions-at-the-follow-up-33-cutoff-2026-11-03-est)
leaves r22 untouched. Its What held item 6 cites the nonexistent
“deploy/systemd/fvpn-agent.service” with **TimeoutStopSec=20s**, and a
**defaultStopTimeout** in main.go. Neither exists on main. The shipped unit is
`deploy/node/systemd/fvpn-agent.service`, **TimeoutStopSec=15s**, pinned by
`TestPKG01_AT07ShippedAgentStopBudget` in `packaging/installer/followup9_test.go`.
The held test's budget check is only **bootLocalBudget + closeLocalBudget ==
13s**; those constants are on the unmerged agent-core branch. **What held in
r22 is the review's statement, not acceptance evidence.**

## Current guidance and decisions

The seven operator/member entrypoints now point at this cut. **F30-01 stays
known behaviour, no merged fix**, as follow-ups 31 and 32 wrote it. The restart
warning remains: **r20 H-1's gap stays open on main while agent-core 37–39
are held**. Main records no scheduled-run result; none is claimed here.

The [findings register](README.md#findings-register-at-the-follow-up-33-cutoff)
has **81 rows**: all 75 prior findings, updated F27-02 / F28-01 / F27-01,
new F21-01, and r22's five returned findings. Every other follow-up 32 row is
byte-identical. **Every historical failure stays failed.** All **47 decision
rows** are unchanged; **no decision was added after D-144 on main**.
**D-135, D-136, D-137, D-138, D-140, D-141, D-142 and D-144 stay provisional**;
**D-143 is Ted's decision**. The nine owner requests below remain unchanged.

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


2026-11-03 — Already routed implementation: F21-01 and the corrected fixture
comparison → updater 22 (including all other short-budget updater tests);
F27-01 → netlab 30; r22 and held agent-core 37/38 work → agent-core 39,
judgment before fixes and review before merge. F30-01 remains agent-core /
updater; F31-01 remains agent-core, design first. F28-01 is closed as test
isolation only; its historical failed gate remains. No outside-owned pin
needed changing. **Nine owner requests**, unchanged.
