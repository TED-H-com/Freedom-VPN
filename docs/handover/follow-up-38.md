# Follow-up 38 — handover 37 recorded; r25 returned before merge

Verified against `main` **77626af4016443c509f3b14e06cfd4f00f74fb49**, through
**#427** on **2026-11-03 EST**. The range starts immediately after #426
(**792e3e41**) and ends at #427: **handover 37 is the only merged PR**.
No unbriefed PR merged inside the range. Main and remote main were at #427
at task start.

At briefing, **agent-core 37–41 were delivered and held before merge**.
**Agent-core 42 was running since 11:36 EST**, on the same unmerged branch:
r25 H-1 and M-1 (item 2), plus Netlab F32-01's renewal jitter. This is status
only. Its results are outside this cut; no judgment from agent-core 37–42 is
on main here. r20–r24's agent-core findings stay as follow-up 37 recorded them.
No live acceptance or timing measurement is made by this handover.

**D-098 / D-109 / D-132:** `TestHandover_Followup38Claims` pins
[the final roadmap row and public PR excerpt](../../tests/handover/testdata/followup38-main.json),
[ten frozen #427 inputs](../../tests/handover/testdata/followup38-snapshots.json),
the [exact r25 archive](../reviews/r25-retain-review.md) and a
[separate frozen source audit](../../tests/handover/testdata/history/follow-up-38/source-audit.json).
Earlier claims keep their assertions; reads of rewritten documents move to
frozen #427 bytes. Living members **81b638e8**, isolation **9f1bc0ff** and
usage **b8a6e9e1** roots are never claims inputs. Review-branch proof names
are pinned only as text in the archive, never as main tests.

## Handover 37 (#427) — previous cut preserved

[Follow-up 37](follow-up-37.md) covers **#424–#426 at 792e3e41**: isolation 18,
handover 36 and netlab 33. Its actual claims function and manifests verify:

- **16 subtests**; **three final roadmap rows**, **ten frozen #426 input
  snapshots** and **three public PR excerpts**.
- **Two immutable documents**: isolation's dated diagnosis and netlab 33's
  `corrections.md`; **all 138 payloads** of isolation's follow-up 17 archive.
- A **separate frozen source audit**; **97 findings**, **47 decisions** and
  **ten owner requests**. Follow-up 36's exact r24 archive and 174 netlab files,
  including 12 gzip originals, remain sealed.

[PR #427](https://github.com/TED-H-com/Freedom-VPN/pull/427) records codex
**gpt-6-astra**, launched **11:23 EST**, delivered **11:36**, **two commits**
and **30 owned files**. Admission waited **87.1 s once**, during the
orchestrator's verification reservation for r25; suite wait **0 s**.
No continuation is recorded. Its `scope-review.json` refresh changed
**258 values, all digests**; the before/after objects verify no other change.

The register addition was **Isolation F18-01**, the at-isolation harness
readiness defect, **FIXED by #424**, with the first scheduled exercise pending.
Its ID is namespaced because bare **F18-01** already exists. Its requests:
**none new**; netlab 33's linking request was completed in that cut. The ten
owner requests remain undecided. #427 records no scheduled-run result after
#424, so this cut claims none.

## Review r25 — exact archive; no adopted findings

**agy `gemini-4-argon-high`** reviewed agent-core follow-ups **40 and 41** at
**8cc2760965abbbb1af3340976c6c4adf1be3dc37**: 41's delivered tip, carrying
**37–41**, which is **not on main**. Source branch **r25-retain-review**,
commit **1afecb29c82bcc8cea6f39a16fa724f0acc7a5d5**, dated **11:30:07 EST**.
The [archive](../reviews/r25-retain-review.md) is copied unchanged from
that commit's `.agent/REVIEW.md`: **31,868 bytes**, SHA-256
**7514218895b014d62d09693b5de38193b3a412c5fa65c2aeb3b44d720d414286**.
Inspection found no host path, username or secret; product paths are retained.

The review's proofs are **node/agent/review_r25_test.go**, **680 lines** at
1afecb29, not on main. This handover does not copy that file. Agent-core 42's
brief requires importing it alone onto the agent-core branch first.

Findings: **0 Critical, 1 High, 1 Medium, 0 Low**. These are the reviewer's
claims, not confirmed or adopted findings on main:

| Finding | Reviewer's claim and attribution | Disposition at #427 |
|---|---|---|
| r25 H-1 | A single binding's deadline, while a sibling remains valid, withdraws the whole node to baseline before the control/probe pass. Reached in verifyCurrentServing or the retained branch, it returns early, leaving survivors unserved with a `-` heartbeat reason and a contradictory observation report. Attributed to follow-up 41 interacting with expireServing, which r25 dates to follow-up 23. | RETURNED — not yet judged; agent-core 42 before merge |
| r25 M-1 | canRetainServing requires the intent epoch to equal the open epoch exactly. A same-interval epoch rotation therefore makes the next unchanged tick force a redundant Apply. Attributed to follow-up 41 interacting with follow-up 40. | RETURNED — not yet judged; agent-core 42 before merge |

The recommendation is **“Merge after named fixes”**: H-1 and M-1 before
agent-core 37–41 merge. It is **the reviewer's recommendation, not a decision**.
**Open question, unanswered:** which part of H-1, if any, already exists on
main? The candidate is main's whole-node withdrawal when the earliest admitted
binding's deadline passes. The orchestrator asked agent-core 42 to judge this
under **D-141**. This handover makes no judgment and adds no operator warning
for r25's findings about unmerged code.

The review's five answered questions are summarized by number in the
[dispositions and additive citation errata](../reviews/README.md#r25-dispositions-at-the-follow-up-38-cutoff-2026-11-03-est):
**Q1** retention's equality, pending gates and fresh live checks;
**Q2** per-device deadlines and edge cases, subject to H-1;
**Q3** r23 restart/settlement fixes, subject to M-1's live-retention gap;
**Q4** claimed preservation of edited-test coverage;
**Q5** claimed soundness of the control-clock and counted-burst fixtures.
These remain the reviewer's claims, not acceptance results. Citation mismatches
are corrected additively in the index; no archive byte is changed. Netlab was
inspected statically by the reviewer, not executed.

## Main status, operator guidance and owner boundary

The [register](README.md#findings-register-at-the-follow-up-38-cutoff) has
**99 findings**. **r25 H-1** and **r25 M-1** are new, collision-free IDs;
**all 97 prior rows are byte-identical**, including r24 H-1/H-3 and Netlab
F32-01. Netlab F32-01 stays open on main, routed to agent-core 42; its older
row label remains historical. Every historical failure stays failed.

All **47 decision rows** remain byte-identical; **no decision was added after
D-144 on main**. **D-135, D-136, D-137, D-138, D-140, D-141, D-142 and D-144
stay provisional; D-143 is Ted's decision**. The ten owner requests below
are carried verbatim from follow-up 37. The tenth G5/G6 trade remains undecided;
production keeps the current guard until the owner decides.

Seven entrypoints point here. Follow-up 37's operator guidance is unchanged
in substance: **no hitless hand-over is implemented**, and **no C1 benefit is
promised**. **r24 H-1 stays open known behaviour on main**; agent-core 41's
fix is not merged and no date is promised. **F30-01 stays known behaviour**;
**r20 H-1's restart gap stays open** while agent-core **37–42 are held**.
#424's at-isolation fix is merged; its first scheduled exercise remains pending
in this cut. Local acceptance is not promoted to a scheduled-run result.

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

2026-11-03 — **Carried agent-core 42 judgment:** decide r25 H-1 and M-1
before merge, including the unanswered D-141 question about H-1 on main;
Netlab F32-01 remains routed there. No judgment or fix from that branch enters
this cut. This is existing routed work, not a new owner question.

2026-11-03 — **Carried operator work:** confirm the first scheduled
at-isolation exercise after #424; AT11 native platform and AT13 full fault
matrices and Isolation F17-01 remain with their existing owners.

2026-11-03 — **No new scope exception or outside-owned pin request.**
Netlab 33's linking request was already completed in #427. No item is stopped.
