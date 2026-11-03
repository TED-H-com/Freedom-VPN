# Follow-up 37 — isolation readiness fixed; netlab citations corrected

Verified against `main` **792e3e4186d00db72aaaf0ee309e6716332b8dd3**, through
**#426** on **2026-11-03 EST**. This cut is **#424–#426**, in merge order:
isolation 18 (#424), handover 36 (#425), netlab 33 (#426). No unbriefed PR
merged inside the range. Main and remote main ended at #426 at task start.
#424 merged after follow-up 36's #423 cutoff and before #425; follow-up 36's
statement that isolation 18 is outside its cut remains true.

At briefing, **agent-core 37–41 were delivered and held before merge**.
#425 records H-1/H-3 fixed on agent-core 41's unmerged branch. **Review r25
was running**, reviewing 37–41 at **8cc27609**. This records status only;
its archive belongs to a later cut. Their judgments are not on main and no
held-branch acceptance result enters this cut. r20–r24's agent-core findings
stay as follow-up 36 recorded them. Handover took no reservation and ran no
live acceptance or timing measurement.

**D-098 / D-109 / D-132:** `TestHandover_Followup37Claims` pins
[three final roadmap rows and three public PR excerpts](../../tests/handover/testdata/followup37-main.json),
[ten frozen #426 inputs](../../tests/handover/testdata/followup37-snapshots.json),
[two immutable documents and the follow-up 17 archive](../../tests/handover/testdata/followup37-evidence.json),
and a [separate frozen source audit](../../tests/handover/testdata/history/follow-up-37/source-audit.json).
Earlier claims keep their assertions; reads of rewritten documents move to
frozen #426 bytes. Follow-up 36's netlab 32 and r24 pins still hold.
Members **81b638e8**, isolation **9f1bc0ff** and usage **b8a6e9e1** roots are
living, never claims inputs. Isolation's new root identity was checked on main;
#424's figures below are pinned through its final roadmap row and dated diagnosis.

## Isolation 18 (#424) — one harness defect behind two unlisted blocks

[PR #424](https://github.com/TED-H-com/Freedom-VPN/pull/424) and the
[dated diagnosis](../e2e/at/isolation/nightly-diagnosis-2026-11-03.md) record
scheduled run **41640196805**, on **487c698e** on **11-03**. It failed as
**D-144** requires: **four blocked rows, two unlisted**. The diagnosis copies
in every log line and record excerpt it relies on because the run's artifact
expires on **2026-11-10**.

Both unlisted rows have one cause: `deadline()` restarted control and the
worker in a deferred `docker start`, with no readiness wait. The next row,
**AT13 short whole-node lease expiry**, attempted admin login through the
gateway **406.6 ms** after restart and got **502**. Control served its first
request about **7.9 s** later. The deferred pre-enrollment fleet audit hit the
same 502 as a consequence, not a second cause. **AT10 foreign ingress redirect**
read a readiness record that only `waitControl` writes, but no step had called
it. It blocked with **no injection**, despite current accepted observations
and a drained retry queue about **17 s** earlier. This is a **harness defect,
not runner slowness and not a product defect**.

The previous night's **41456092596** is consistent with the same cause:
AT13 blocked **337.7 ms** after restart. AT10 proceeded using the checkout's
follow-up 16 readiness file. **That AT10 pass is not a qualified result**;
no acceptance status changes. No existing owned document or register row
records that nightly pass as acceptance; this qualification is added here.

**Outcome (a), fixed in owned harness paths:** each control restart and each
step needing control now waits for current-run readiness. The predicate is
main's existing predicate plus a **non-empty desired generation**: successful
admin login, fleet-config HTTP 200, available and drained observation queue,
and a post-wait-start accepted observation matching the desired generation
with member forwarding enabled. Stale or foreign readiness records are not
trusted. More precisely, the helper compares **started_at**, rejects a
mismatched run ID when both IDs are non-empty, and rechecks the persisted
record after waiting. It does not check safety_status, forwarding_block_reasons,
kernel_ruleset or readback. With no injection, the review script emits
**passed: false / status: blocked**, skips the missing-window join and exits
**0** instead of crashing. **expected-blocked.json is unchanged**; no row was
added to it.

The new Go regression is
`TestAT13_AT10_Nightly41640196805ControlRestorationSequence` in
[scheduled_test.go](../../tests/e2e/at/isolation/scheduled_test.go). It replays
the recorded 502 / queue-drain sequence, missing, foreign and pre-restart
readiness, and reuse after a successful wait. The helpers are new, so this
has no before-fail; the assembled run proves the control-flow fix. The Python
regression is
`test_unexecuted_foreign_redirect_writes_blocked_verdict_without_traceback` in
[nightly_test.py](../../tests/e2e/at/isolation/nightly_test.py). #424 records its
failure against the pre-fix script and pass on the branch. The Python name
is pinned through the dated diagnosis and frozen source audit, not a living
foreign-prose assertion.

Follow-up 17's root snapshot was archived first, **138 files**, under
[history/follow-up-17](../e2e/at/isolation/history/follow-up-17/archive.json).
One local assembled run, **9f1bc0ff** on **c248db96**, then records:

| Case | Pass / blocked / fail |
|---|---|
| AT10 | 9 / 0 / 0 |
| AT11 | 2 / 1 / 0 |
| AT12 | 10 / 0 / 0 |
| AT13 | 6 / 1 / 0 |

Only the two listed operator rows block. The short whole-node expiry passes;
its **D-140 sample qualifies under D-142**, and the unchanged evaluator passes
with **164 positives, 70 drops, no overlap**. This is a local result.
**The next scheduled run after merge is the first to exercise the fix on
GitHub's runner. No PR in this cut records that exercise.** AT11's native
platform matrix and AT13's full fault matrix stay operator rows. **Isolation
F17-01** (pure stale_generation not exercised) stays carried with wire-nodes
and agent-core.

#424 also records **one documents-only continuation** returned by the
orchestrator. The first diagnosis overstated the readiness predicate with
three unsupported checks, named completed_at instead of started_at, and said
the review script exited 1 with fields it does not write. The session corrected
the diagnosis in place, rechecked every code quote and line reference, and
added a dated report correction. Those initial statements are not adopted here.

## Handover 36 (#425) — previous cut preserved

[Follow-up 36](follow-up-36.md) covers **#422–#423 at d7867928**.
Its actual claims function and manifests verify **16 subtests**, **two final
roadmap rows**, **ten frozen #423 input snapshots**, the **exact r24 archive**,
**174 netlab 32 design/evidence files** (**12 gzip originals**), **two public
PR excerpts**, a **separate frozen source audit**, **96 findings**, **47
decisions** and **ten owner requests**. The 174 files are three design and
171 evidence files. The raw r24 archive remains **46,001 bytes**, SHA-256
**7ed69f029bb12b526130e26608474f1b1cb15ac6346c3bdab6a7cc875af437f8**.

[PR #425](https://github.com/TED-H-com/Freedom-VPN/pull/425) records a codex
**gpt-6-astra** session, launched **09:57 EST**, delivered **10:16**.
Admission waited **269.4 s once**; suite wait **0 s**. It records two
continuations on the same thread:

1. **Lock citation:** the first delivery said the original follow-up 31 lock
   path matches main, but follow-up 31 names no lock path. The orchestrator
   returned it; the session corrected **three documents and one pin**.
2. **Digest refresh:** the orchestrator's brief omitted scope-review.json,
   causing the session to stop and ask. It then applied the approved refresh:
   **53 values, and nothing else**.

Its requests are answered: netlab's additive correction is **#426** below;
the tenth owner request was relayed and remains undecided; H-1/H-3 are fixed
on **unmerged agent-core 41**, with **r25 in progress**; **Netlab F32-01 is
routed to agent-core 42**. These routing updates do not import the held branch's
judgments or change its findings' main status.

## Netlab 33 (#426) — additive corrections, documents only

[PR #426](https://github.com/TED-H-com/Freedom-VPN/pull/426) adds exactly one
file: [corrections.md](../poc/G2-followup33-design/corrections.md), SHA-256
**cc644de260d63f480c52833ddc4cffc50ddce7363b7bffa66df8484d4d7fc50a**.
Both netlab 32 trees remain **byte-identical**, verified against all 174
follow-up 36 pins, including the 12 gzip originals. No C1 implementation or
new lab result is part of #426.

Follow-up 32's protocol.md:93 names
`/var/lib/vpn-node/update-journal/update.lock`. Every default on main uses
**/var/lib/vpn-node/update.lock**: agent configuration, nodelock (including its
internal re-export), updater and installer layouts, and watchdog configuration.
**/var/lib/vpn-node/update-journal is the update journal directory**.

**r24 L-1 is confirmed in part.** Its lock-location half is rejected: follow-up
31's cited line starts the timing definition and names no lock; the design
names no lock location; r24's cited node/updater/config.go does not exist.
The empty eligible-set half stands as follow-up 32 specified it, documents
only. **No follow-up 32 conclusion depends on the path string.** Delayed
fallback fencing requires every writer to lock the same path and timed-out
children to be killed and reaped before releasing it. The shared default
also preserves the watchdog's try-lock contract.

**r24 M-3's B1 characterization has no source.** Neither TCP_RTO_MIN nor
socket pacing appears anywhere at **52a7e59b**, the tree r24 reviewed.
Its cited d133-comparison.md:65–75 gives stop and admission rules. B1 is the
first candidate arm of the **A1,B1,A2,B2 D-133 quartet**; follow-up 27's B1
holds **F27-01's failed sequence-43 read, 1,501.407973 ms**. Follow-up 32's
protocol.md and revision.md adopted r24's wording. The substantive constraint
stays: **no transport or socket change in any C1 acceptance run or comparison
arm**. B1 stays FAIL and unexplained; netlab 30 stays host-unqualified.

The complete citation recheck in corrections.md §3.2 has **16 rows: 14
Verified, two Corrected** (L-1 and M-3/B1). It checks every other r24 citation,
test name, quote and figure that follow-up 32 adopted or addressed against
main and the reviewed tree. Its whole-tree search (§4) is commit-scoped:
**e54700f9** and **9f85c3fb**. The wrong path's only adoption is follow-up 32's
protocol.md:93. The r24 archive preserves it; six handover 36 hits quote it as
the claim being corrected. **Zero other hits; zero in all 411 tracked gzip
archives.** The ignored brief quotes the path too, as the document records.

#426 records four continuations: **three returns by the orchestrator and one
correction of its own brief**:

1. Invented test names/file and figures in the first citation table were
   replaced with real regressions, lines and recorded figures; all names,
   links and quotes were mechanically checked.
2. A figures row attributed netlab 30's 100 reloads / 101 verified Applies /
   118 reads / zero failures to follow-up 32 lines that lack them, and quoted
   follow-up 32's summary as r24's words. Each statement now names its source.
3. The search was refreshed after #425, with each searched commit and the
   new handover hits stated explicitly.
4. The orchestrator's brief had called all 31 intervening files handover's.
   The session checked and corrected it: **30 owned files plus docs/ROADMAP.md**,
   #425's roadmap row committed by the orchestrator's merge.

The [reviews index](../reviews/README.md#follow-up-37-additive-r24-errata-2026-11-03-est)
links this correction from r24's entry and adds L-1 and M-3 errata. The raw
archive is untouched. Netlab's request to link corrections.md is completed.

## Operator guidance, findings and decisions

Follow-up 36's reload guidance stays: a serving node on main re-installs
unchanged enforcement on every successful nominal ten-second poll, closing
member forwarding for the guard window. **r24 H-1 stays open known behaviour
on main**; agent-core 41's fix is unmerged, with no date promised. **No hitless
hand-over is implemented** and no C1 benefit is promised. **F30-01 stays known
behaviour**; **r20 H-1's restart gap stays open** while agent-core 37–41 are held.
The at-isolation harness fix is merged; its first scheduled exercise is pending.

The [register](README.md#findings-register-at-the-follow-up-37-cutoff) has
**97 findings**: r24 L-1 and M-3 updated, **Isolation F18-01 added**, **94 other
rows byte-identical**, including r24 H-1/H-3 and Netlab F32-01. The new ID is
namespaced to avoid the existing bare **F18-01**. It is **FIXED by #424**, with
the runner's first exercise pending. Every historical failure stays failed.

All **47 decision rows** remain byte-identical; **no decision was added after
D-144 on main**. **D-135, D-136, D-137, D-138, D-140, D-141, D-142 and D-144
stay provisional; D-143 is Ted's decision**. Follow-up 36's ten owner requests
are carried verbatim below. The tenth is still undecided; production keeps
the current guard until the owner decides.

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

2026-11-03 — **Carried implementation and operator work, outside owned paths:**
AT11 native platform and AT13 full fault matrices stay operator rows; Isolation
F17-01 stays with wire-nodes and agent-core. Confirm the first scheduled
at-isolation exercise after #424 without relabeling the local run as a runner
result. Agent-core 37–41 remain held for review; r25's archive belongs to a later
cut. Netlab F32-01 is routed to agent-core 42. No new owner question is added.

2026-11-03 — **Completed handover request:** #426's corrections.md is linked
here and from r24's reviews-index entry. No outside-owned pin needs changing;
no item is stopped for a scope exception. C1 provenance/bank/rollback work and
any C1 comparison remain subject to the existing owner decision and prerequisites.
