# Follow-up 26 — shutdown fixes, closed-node timeouts and intersection design

Fixed `main` cutoff **9dde5eeb60a6cb6b9b10974c3100ef1ea1352698**, through
**#382 (wire-nodes 34)**, checked on 2026-11-01 EST. This cut includes
**#374–#382** and the **5f4b455c** correction to #381's roadmap row. Nothing
was merged after the cutoff at task start. **r19, isolation 14, members 15 and
devstack 26 were running**; none of their results enters this cut. Handover
ran no new live acceptance.

**D-098 / D-109 / D-132:** [nine exact roadmap rows](../../tests/handover/testdata/followup26-main.json)
and [17 immutable snapshots](../../tests/handover/testdata/followup26-snapshots.json)
pin `TestHandover_Followup26Claims`: **21 subtests / 1075 archive and run file
checks**. Snapshots retain source commit, full-source
digest, byte offset, length and excerpt digest. Members 14's **258ba504** and
isolation 13's **38074149** root records are living until members 15 and
isolation 14 archive them: **cite by roadmap row / PR only**, never copy or
pin those living roots. Owner archives and the devstack run inventory are
checked separately. Live secops digests cover owned-document integrity, not
acceptance. Every earlier claims test remains true.

## Previous handover — #374

[Follow-up 25](follow-up-25.md) covered **#370–#373 at 9cd7a053**.
`TestHandover_Followup25Claims` retains **18 subtests / 4 exact roadmap rows /
15 immutable snapshots / 270 archive and run file checks**. It archived r18
byte-for-byte as [r18-generation-change-review.md](../reviews/r18-generation-change-review.md).
Its historical findings and cutoff stay intact; the current dispositions below
supersede them only for this new cut.

## Members 14 — #375

D-098 archive of **4182dd3d**, **61 files**, under
`docs/e2e/at/members/history/follow-up-13/`. Run **258ba504**: **AT03 and AT08
pass**; **AT02 fails** the provisional D-135 / D-136 check, with **58 baselines
paired to recovery, 21 losses**. No loss is waived. **AT04 fails** on a fresh
members F13-01 sample: **four duplicated identical chain definitions**, rejected
by `parse_nft.go`; **raw bytes private, public digest**. The origin is unproven.

**AT09's relay row is confounded by harness order**: the routed hop preceded
the eth0-only relay hold. Reordered in **682d4434**, **not live-qualified**.
**New F14-01:** the first **D-077 re-grant forwarding attempt failed after valid
readiness**; cause unproven. It routes to devstack 26. All these current-run
claims are pinned to #375's roadmap row, not its living evidence files.

## Agent-core 32 and netlab 23 — #376 / #377

[Agent-core 32 hand-off](../node/followup32-shutdown-hold-observations-2026-11-01.md)
fixes four r18 findings and ISO-F6-01:

| Finding | Fix and remaining limit |
|---|---|
| M-1 | No observation POST after the caller cancels; the deferred finalizer shares the control budget. |
| M-2, agent side | One local deadline per Step, at most 10 s, then Close at most 3 s: **cooperative SIGTERM-to-exit worst case 13 s**, plus scheduling and uninterruptible synchronous syscall overhead. |
| M-3, agent side | Every stale-lease hold observation carries **policy_missing plus STALE_GENERATION at stage lease**. A node-side alarm follows after **3 Steps or 30 s**, whichever comes first; the baseline stays closed. |
| L-2 | The D-133 candidate rule fails a leg on **any higher or missing ceiling**, for surviving members and the node, upload and download. |
| ISO-F6-01 | `observationState` omitted the rejected candidate generation, coalescing the newer rejection behind the old pending report. Fixed with separate candidate evidence; no retrospective attribution to #369. |

Adopted proofs are in `node/agent/review_r18_generation_test.go`;
`TestF32_ISOF601_RejectedCandidateSurvivesObservationBackoff` covers the
seven-attempt rejection case. A cooperative budget is not a hard bound on a
wedged system call or proof of physical arm64 behavior.

**Netlab 23 (#377) fixes r18 L-1**:
`TestReviewR18_EnabledSampleIgnoresInstallDeadline` now asserts the corrected
behavior. Install-phase resource sampling is skipped. For **members F13-01**,
the lab observed **0 / 3,300 duplicated-chain listings**, so **origin unproven**.
Every duplicate owned-chain definition, identical or otherwise, stays rejected,
now as a typed **`*enforce.DuplicateChainError` (`duplicate_chain`)**.
**No historical row is reclassified.** `TestF13_01_DuplicateChainKind` pins the
regression name on main.

## Installer 9 — #378

[Installer hand-off](../node/install-followup9-2026-11-01.md), r18 M-2 unit side:
**`fvpn-agent.service` sets `TimeoutStopSec=15s`**; **`fvpn-watchdog.service`
gains `Before=fvpn-agent.service`**, so it outlives the agent on a joint stop.
Binary replacement waits with a **20 s per-unit stop deadline**. On error or
timeout it journals **`stop-failed`** and rolls back **before touching the
candidate, `current` or units**. Other stop paths retain their existing scope.
The agent and unit fixes together close M-2; these bounds do not promise a
15 s total paired stop or an adoptable checkpoint.

This is the cut's **one operator-visible change**: one dated line in
[operator-guide §3.1](operator-guide.md#31-enrolling-a-new-node-or-reinstalling-an-existing-node-post-apiv1adminenrollments),
which already covers installer rollback. **No other new operator guidance.**
Regression names: `TestPKG01_AT07ShippedAgentStopBudget`,
`TestPKG01_StopWaitsThroughUnitBudget` and
`TestPKG01_AT25ReplaceStopFailureIsJournaled`.

## Agent-core 33 — #379, D-139 design track

The [intersection-serving design](../node/followup33-intersection-serving-design-2026-11-01.md)
specifies **option (b)** for adversarial review. Its test-only reference is
`node/agent/followup33_intersection_spec_test.go`: **7 tests, 558 table subtests**.
A signed **`intersection_constraints` snapshot is required, or the node
baselines**. The conservative profile still **baselines before the final full
Apply and on every feature-enabled restart**. **No uninterrupted AT02
transition is promised.**

**Option (c) stays in production.** This is a **proposal under review**, with
**r19 running**; the reference passing is **not an acceptance result**. There
is no contract, central, enforcement or runtime implementation approval yet.

## Isolation 13 — #380

D-098 archive of **1c25ee44**, **108 files**, under
`docs/e2e/at/isolation/history/follow-up-12/`. Run **38074149** on #376:

| Case | Result at this cutoff |
|---|---|
| AT10 | **PASS 9 / 0 / 0**, **ISO-F6-01 6 / 6**; assertion 6 met by accepted reports. |
| AT12 | **PASS 10 / 0 / 0**. |
| AT11 | **BLOCKED 2 / 1 / 0**, operator rows. |
| AT13 | **BLOCKED 6 / 1 / 0**, operator rows. |

#376 regression: hold reports carry **policy_missing plus STALE_GENERATION**;
**the alarm was not exercised**. **1c25ee44's block stays historical**. The new
pass does not rewrite it. Current-run results are pinned to #380's roadmap row
only, not living root records. The node alarm and #382 central escalation still
need assembled-run assertions.

## Devstack 25 — #381, corrected by 5f4b455c

The [campaign analysis](../e2e/runs/2026-11-01/followup25/analysis.md) and
`docs/e2e/followup25-captures.sha256` retain **32 attempts: 26 pass / 6 timeout**.
The **`::1`-source response appears only on two successful kernel IPv6
requests**. It is absent from the **TEST ONLY userspace IPv6 arm**, which still
**timed out 3 / 8**. Absence in this arm is not a forwarding repair.

**Every timeout ends on a closed node: generation 0 / serving=false.**
**Four already sampled that state before the request**; this does not mean
those nodes never served. **1d253f8d had an earlier true heartbeat**;
**0c27cac0 went from generation 5 / serving=true to 0 / false**. Independent
samples do not locate the exact withdrawal instant. **The cause is unproven**;
follow-up 24's d556673b timeout stays unattributed. Devstack 26 investigates
the closed-node reason before the owner chooses the **F11-01 repair direction**.
That choice is an **owner decision, not a recommendation**.

The **TEST ONLY agent wait receipt was delivered**, with a **D-102 answer for
updates**. It records the unique launch, wait status and exit versus signal;
updates adopts it next. **Updates F13-01 is distinct from members F13-01**:
**AT23's historical unobserved exit stays unobserved**.

**TEST ONLY harness stop-path gap:** isolation 13's container stops ended in
**SIGKILL / exit 137**; the devstack supervisor escalates after **about 5 s**.
This is **a harness gap, not a product finding**, routed to devstack 26 for
signal forwarding and alignment with the shipped budget. See the immutable
[#380 routing](../../tests/handover/testdata/history/follow-up-26/isolation13-routing.md)
and [#381 routing](../../tests/handover/testdata/history/follow-up-26/devstack25-routing.md).

## Wire-nodes 34 — #382, cutoff

[Central hold hand-off](../nodes/followup34-generation-hold-2026-11-01.md),
r18 M-3 central side: the stale-lease hold is recognized **whatever the applied
ACK says**, joined **only to its own signed lease** on the current incarnation.
**`stale_generation` comes before generic causes**; only the agent's two
dependent **qos / source_binding** checks yield. Unrelated safety failures
retain priority. **Migration 00194 retains incidents across agent boots**.
Exactly **one `node.generation_hold.escalated` audit escalation per incident**
is recorded, transactionally, on a valid alarm or the central age threshold.

**Assembled-run assertions are requested from isolation and members; no AT
pass is claimed here.** M-3 is fixed on both sides (#376 + #382), while its
alarm and central escalation have **not yet been exercised in an assembled run**.

## Decisions and findings at #382

**D-139 — orchestrator, provisional, 2026-11-01 06:00 EST:** option (b)
intersection serving proceeds **only as a design track**: agent-core 33 writes
the design (#379), r19 attacks it, and **one recorded decision approves or
rejects its representation before contracts, wire-nodes, netlab or agent-core
implement anything**. Option (c) stays in production meanwhile.

Grounds: AT02's remaining failure is the **node-wide baseline's sibling
outage**. Agent-core 31 measured **p50 / p99 173.8 / 195.6 ms, 501 losses**;
members 14 records **58 brackets, 21 losses**. Option (b) is the only path to
an unaffected sibling without a requirement change; this motivates review,
not a guarantee that the conservative proposal meets that goal.

**D-135, D-136 (including its 04:3x correction), D-137 and D-138 stay
provisional and unchanged.** The ordinary stale-lease path is
baseline→ACK→renewal→Apply→verify→heartbeat; a verified deny-all candidate is
only for a retained Apply error. No finite outage ceiling; every loss counts.
D-139 advances the review track described historically in D-136, without
changing that decision's fail-closed or acceptance rules.

The [current findings register](README.md#findings-register-at-the-follow-up-26-cutoff)
closes **r18 M-1 / L-1 / L-2** (#376 / #377), **M-2 on both sides** (#376 +
#378), **M-3 on both sides** (#376 + #382), and **ISO-F6-01** (#376, AT10 pass
#380). It updates F11-01, members F13-01 and updates F13-01, adds **F14-01**
and the separate **TEST ONLY harness gap**. **Every other follow-up 25 finding
is unchanged. Every historical failure stays failed.**

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**, together,
including the 04:3x correction, all losses and no finite outage ceiling.

2026-11-01 — **Owner decision 2: D-137 hardening option**, supported scope and
whether lower tunnel MTU or an MSS clamp is wanted.

2026-11-01 — **Owner decision 3: D-138 overturn option**, confirm or overturn
the provisional observation acceptance and pooled qualified-leg assessment.

2026-11-01 — **Owner decision 4: D-139 overturn option**, confirm or overturn
the provisional design track; after r19, record the representation decision
before any implementation.

2026-11-01 — **Owner decision 5: F11-01 repair direction**, waiting on
**devstack 26's closed-node investigation**. Existing transport options remain
owner choices; no recommendation or production change is made here.

Already assigned, not new requests (2026-11-01): **devstack 26** owns the
F11-01 investigation, **F14-01**, and the TEST ONLY stop-path gap in
`deploy/devstack/agent-service.sh` and `deploy/devstack/entrypoint-node.sh`.
**Updates** adopts the delivered receipt in
`tests/e2e/at/updates/stopobserve/main.go`, preserving historical AT23.
**Isolation and members** own #382's assembled incident/alarm/escalation
assertions under their `tests/e2e/at/` paths; no local test substitutes for
that run. **r19 / orchestrator** review the D-139 representation before any
owner implements it. **Operators** retain AT11 / AT13 and AT29 / AT31 resources.
These outside-owned items stop with their owners; the rest is delivered.
**No outside-owned pin needed changing.** No production value is selected.
