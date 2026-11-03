# Follow-up 36 — enforcement fixes merged; C1 remains the owner's decision

Verified against `main` **d7867928d48a7f75259f84abf1b29a9e37675fcb**, through
**#423** on **2026-11-03 EST**. This cut is **#422–#423**, in merge order:
handover 35 (#422), netlab 32 (#423). No unbriefed PR merged inside the range.
No later merge was on main at task start. At briefing, agent-core 37–40 were
delivered and held (40 judged and fixed r23 on its branch); agent-core 41 and
isolation 18 were running. Their results are outside this cut. Agent-core
37–41's judgments are not on main; r20–r23 retain follow-up 35's dispositions.
Isolation 18 diagnoses the nightly's two unlisted blocked rows; this cut
claims no scheduled-run result. Handover took no reservation and ran no live
acceptance or timing measurement.

**D-098 / D-109 / D-132:** `TestHandover_Followup36Claims` pins
[two final rows and two public PR excerpts](../../tests/handover/testdata/followup36-main.json),
[ten frozen #423 inputs](../../tests/handover/testdata/followup36-snapshots.json),
[every file in both netlab 32 trees and the exact r24 archive](../../tests/handover/testdata/followup36-evidence.json),
and a [separate frozen source audit](../../tests/handover/testdata/history/follow-up-36/source-audit.json).
Earlier assertions stay unchanged; reads of rewritten documents move to frozen
#423 bytes. Members 81b638e8, isolation 0ec14b39 and usage b8a6e9e1 roots remain
living and are excluded from claims inputs.

## Previous cut — handover 35 (#422)

[Follow-up 35](follow-up-35.md) covers **#419–#421 at dff6bdb0**.
Its actual function and manifests on main verify **16 subtests / three final
roadmap rows / ten frozen #421 input snapshots / three documents / six netlab
31 design files / three public PR excerpts / a separate frozen source audit /
87 findings / 47 decisions / ten owner requests**. The three documents are
updater 23's dated hand-off, its timing JSON and the r23 archive.

[PR #422](https://github.com/TED-H-com/Freedom-VPN/pull/422) records **one codex
`gpt-6-astra` session**, launched **08:19 EST**, with admission and suite waits
of **0 s**. It answers the evidence-class request: **r23 L-1 is a code trace**;
the brief was wrong to say every r23 finding has a proof test. **No proof is
added**; agent-core 40 judges L-1 with the other four. The r23 register rows
stay exactly as follow-up 35 wrote them, including their historical wording.
Follow-up 35 recorded only the after-cutoff status of r24 commit **ba5d3004**;
this cut archives that review.

## Netlab 32 (#423) — enforcement fixes and design revision

[PR #423](https://github.com/TED-H-com/Freedom-VPN/pull/423) and the
[pre-edit judgment](../poc/G2-followup32-evidence/judgment.md) record **D-141
judgment before implementation**: the exact r24 enforcement proof file was
imported and committed alone at **cdb384ed**, and its failures were recorded
before any production change.

| Finding | Result on main |
|---|---|
| H-4 | Confirmed and FIXED by #423. Rollback needs neither a fresh state read nor the canceled Apply context. One self-contained baseline batch resets the candidate's fixed chains and sets, empties admission, preserves counters and touches no foreign table. Baseline and quarantine cleanup use a cancellation-independent context with a five-second budget. |
| H-2 tc order | Confirmed and FIXED by #423. Prepare classes before nft; retire them after the guarded commit and budget check, before release. Failed retirement keeps the guard closed and attempts baseline. Extra or missing final classes fail verification. |
| M-2 | Confirmed in part. Both post-shape clock errors now attempt rollback, FIXED by #423. The alleged fatal `Verified:true` return is rejected as currently unreachable: Samples cannot fail and invalid conntrack addresses fail validation before mutation. A regression preserves that boundary. |
| M-3 | The copied classifier test was dropped because it never invoked the tooling. The finding rests on tooling inspection; real classifier and decorator tests are prerequisites for C1. No tooling change. |

[One full lab run passed](../poc/G2-followup32-evidence/lab-summary.md), including
`TestISO01_AT13_F32_UnknownStateBaselineAtomic`. It proves cold/warm/repeated
baseline behavior, transaction abort atomicity and counter continuity on the
recorded lab backend. It is not native arm64 qualification or a D-133 comparison.
The fixed-name baseline does not implement C1 bank cleanup; rollback commands
can still fail. A five-second cleanup budget is not a recovery bound.

The [delivery outcome](../poc/G2-followup32-evidence/outcome.md) and
[consumer hand-off](../poc/G2-followup32-evidence/consumer-handoff.md) retain both
consumer-gate results. **The first gate failed on seven agent fixture tests**:
the nft simulator did not model idempotent chain adds, and the tc model had no
two-phase change. The PR attributes the stop to **the orchestrator's brief**:
its “consumers without edits” rule prevented the fixture work. The resume
clarified the rule and extended test-only scope. The simulator now models
create-if-missing adds, bare chains, atomic nft batches and tc preparation /
commit / retirement. **The unchanged gate then passed**; no production file
or existing consumer assertion changed in the resume. The first failure stays
in the evidence as history, not a current blocker.

Existing tests on main include `TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch`,
`TestR24_M2_PostShapeClockCheckSkipsBaselineInApply` (historical names, fixed
assertions), and the three `TestAT13_F32_Simulator*` tests. The claims test pins
their exact declarations, the first gate-3 output and its rerun, and every
SHA-256 and length in the per-follow-up evidence and design trees.

The [C1 revision](../poc/G2-followup32-design/revision.md), with
[protocol](../poc/G2-followup32-design/protocol.md) and
[owner question](../poc/G2-followup32-design/owner-decision.md), is **documents
only; nothing of C1 is implemented**. Production keeps the current guard.
It permits otherwise eligible identities across rate-only fair-share changes
when class identity is stable and ceilings do not tighten; per-device deadlines
and a separate verified-serving record are agent-core prerequisites. The old
bank is deleted at release; the lock and child reaping fence delayed fallback
installs. Empty eligibility omits both return rules and the eligibility set.
Cadence and exposure figures are corrected as recorded below.

The revision calls L-1 confirmed and changes the lock location, but **that
location does not match main**: agent configuration, shared nodelock, updater
layout and installer layout use `/var/lib/vpn-node/update.lock`; protocol.md
claims `/var/lib/vpn-node/update-journal/update.lock`. The archive also cites
an updater config file absent on main. This discrepancy is returned for an
additive correction; neither immutable source is edited here. The empty-set
design correction remains recorded.

**History:** follow-ups **25–31 are byte-identical**, all **1,830 files**;
**B1 stays FAIL and unexplained**; netlab 30's runs stay **host-unqualified**.
The [PR's answered requests](https://github.com/TED-H-com/Freedom-VPN/pull/423)
keep the G5/G6 trade with the owner and the guard in production. H-1 and H-3 go
to agent-core 41; jitter units go to agent-core after 41. Verified-serving
provenance and cross-lane bank readers, baseline writers and rollback work
wait for the owner's decision; nothing is scheduled. Agent-core 37–41's owed
D-133 slot comes before any C1 comparison, which also needs approval,
implementation and a new frozen baseline on the reviewed fixes. The fixture
request was completed in the resume.

## Review r24 — exact archive and bounded disposition

The [raw archive](../reviews/r24-hitless-guard-review.md) is **46,001 bytes**,
SHA-256 **7ed69f029bb12b526130e26608474f1b1cb15ac6346c3bdab6a7cc875af437f8**,
identical to the supplied review file and the review branch. Hygiene inspection
found no host path, username or secret; its node state paths are product paths.
**agy `gemini-4-argon-high`** reviewed netlab 31 delivered at **52a7e59b**, the
design tree merged by #421. Review branch **r24-hitless-guard-review**, commit
**ba5d3004**, dated **08:19:51 EST**. Findings: **0 Critical, 4 High, 3 Medium,
1 Low**. Proof files named review_r24_test.go are on the review branch under
node/agent, node/enforce and node/watchdog. **M-1 is a counterexample trace;
L-1 is code inspection**. Several agent/watchdog proofs cited by the archive
are on the review branch only, not main; no absent test names are listed here.

The review recommends, in its words: **“Implement C1 after named changes
(`H-1`, `H-2`, `H-3`, `H-4`, `M-1`, `M-2`, `M-3`, `L-1`)”**. Its owner section
says **“Accept the trade and implement C1 after fixing `H-1` through `M-3`”**.
These are **the reviewer's recommendations**. The trade is the owner's decision;
the orchestrator does not decide it.

H-4, H-2's tc order and M-2's post-shape rollback are fixed by #423. H-2's
eligibility (confirmed in part), M-1 (design correction), M-3 (tooling inspection,
copied proof dropped) and L-1 (netlab's confirmed judgment, with the lock-path
discrepancy above) are **documents-only C1 prerequisites**. **H-1 and H-3's
agent-core part are OPEN on main — RETURNED, not yet judged; agent-core 41
before merge**. That lane must judge both under D-141. Netlab 32 describes H-1
as “confirmed by code” and H-3 as a confirmed outcome with a corrected
explanation; those are not agent-core 41's judgments.

The [reviews-index errata](../reviews/README.md#r24-dispositions-at-the-follow-up-36-cutoff-2026-11-03-est)
leave the archive untouched. Verified corrections: memberTimeout caps each
binding's ValidUntil, but Decide first collapses all bindings into one member
DeadlineMono and takes service/node ends from the first binding; M-3 copied the
classifier instead of invoking it. Netlab 30's **101 Applies** record **one D-120
UDP boundary loss, seven matched TCP guard drops, seven retransmissions and a
416.739885 ms maximum read**, not the review's 22 / 28 / 256.8 ms figures.
**28.5 ms median / 94.7 ms maximum are outer command brackets**, not exact
blackouts; at 28.5 ms per 10 s the illustrative complement is **99.715%**.
The watchdog's **30–40 s is conditional, not a bound**. “Every ~23h58m” and
“every 13m” are nominal window-minus-lead illustrations. Default time-triggered
renewal starts at about **108 s remaining**, not ~120 s; other renewal triggers
use the full residual old deadline. QoS and host reserve are **§11**, not §8.3.
Follow-up 31's **G1–G10 ledger is the numbering authority**; r24's own What held
labels do not rename the owner's **G5/G6** trade.

## Guidance, findings and decisions

A serving node on main re-installs unchanged enforcement after every successful
nominal ten-second poll. Each install closes member forwarding for the guard
window. The netlab 30 brackets are observations, not exact outages or future
bounds. Netlab 32 changed the install order, not the guard. **Member TCP segments
can still be dropped while a generation installs; no hitless hand-over is
implemented.** Agent-core 41's fix is unmerged; no date or C1 benefit is promised.
F30-01 stays known behaviour. r20 H-1's restart gap stays open while agent-core
37–41 are held. No scheduled-run result is claimed by this cut.

The [register](README.md#findings-register-at-the-follow-up-36-cutoff) has
**96 findings**: F27-01 / B1 updated; r24's eight and **Netlab F32-01** added.
The chosen jitter ID has no collision. In leadJitter, two random bytes supply
at most **65,535 ns**, but the modulus spans nanoseconds: the default 120 s lead
gives **−12 s to −12 s + 65,535 ns**, effectively fixed −12 s, not uniform ±12 s.
This is the cadence-check defect found in netlab 32's owner-decision.md;
**OPEN on main, routed to agent-core after 41**. No jitter fix is included.
Every other follow-up 35 finding row, including r20–r23, is byte-identical.
Every historical failure stays failed.

All **47 decision rows** are unchanged; **no decision was added after D-144 on
main**. **D-135, D-136, D-137, D-138, D-140, D-141, D-142 and D-144 stay
provisional; D-143 is Ted's decision**. The ten owner requests remain ten:
the first nine verbatim, the tenth restated below from main. It is still
undecided; production keeps the current guard until the owner decides.

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

2026-11-03 — **Owner question 10: netlab 31's G5/G6 trade, restated by netlab 32**.
The following paragraph is quoted verbatim from main's owner-decision.md:

> Once unnecessary ten-second reinstallation is fixed, the guard would briefly
> interrupt members mainly on lease renewals, actual policy changes and recovery
> from restarts; the observed outer window was about 28.5ms at the median and
> 94.7ms at the maximum, with packet recovery sometimes taking longer. C1 would
> let unchanged, still-authorized members continue through that window, but if
> installation is late or the agent dies there, those members would no longer
> close immediately: they could continue until independent shutdown or their
> already-running old deadline, about two minutes for an ordinary default timed
> renewal and at most the old lease's remaining 24 hours in the general case
> (less with a shorter lease). Today's steady state already has that deadline
> exposure when the agent fails outside the guard window; C1 would extend it into
> the window while still closing revoked or tightened members at the candidate
> commit. Do you accept that specific availability-versus-immediate-closure trade,
> or should production keep the current guard?

Still undecided. This is the existing tenth request, not an eleventh. Production
keeps the current guard until the owner decides; r24's recommendation to accept
is the reviewer's, not a decision.

2026-11-03 — **Lock-path discrepancy, outside owned paths:** request an additive
correction from netlab to docs/poc/G2-followup32-design/protocol.md and its L-1
judgment context in revision.md. They claim update-journal/update.lock, while
main's node/agent/config.go, node/nodelock/lock.go, node/updater/layout.go and
packaging/installer/layout.go default to /var/lib/vpn-node/update.lock. The
archive's node/updater/config.go citation is absent. No foreign document or
archive byte is edited; only that correction is held for its owner.

2026-11-03 — **Existing routing, no new owner question:** agent-core 41 judges
H-1/H-3 before merge; agent-core after 41 owns Netlab F32-01. Verified-serving
and cross-lane bank/rollback work wait for the owner's C1 answer. Agent-core's
owed D-133 comparison precedes any C1 comparison. #422 answered r23 L-1's
proof-class request; #423's fixture request was completed in the resume.
