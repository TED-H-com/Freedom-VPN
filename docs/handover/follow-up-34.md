# Follow-up 34 — fixture fixed, guard drops demonstrated, acceptance limits retained

Verified against `main` **cc0ed4ae713f6c8242171f923bdc799a0c07a74c**,
through **#418** on **2026-11-03 EST**. This cut is **#416–#418**, in merge
order: handover 33 (#416), updater 22 (#417), netlab 30 (#418).
No unbriefed PR merged inside the range. Nothing was merged after the cutoff
at task start. Agent-core 39 (carrying 37 and 38) was delivered and held for
review r23 before merge; r23, updater 23 and netlab 31 were running or held.
Their results are outside this cut. No new live acceptance or timing
measurement was run by handover, and no reservation was taken.

**D-098 / D-109 / D-132:** `TestHandover_Followup34Claims` pins
[three final roadmap rows and public PR excerpts](../../tests/handover/testdata/followup34-main.json),
[ten frozen handover inputs](../../tests/handover/testdata/followup34-snapshots.json),
and [two dated updater documents and the netlab evidence manifest](../../tests/handover/testdata/followup34-evidence.json).
Earlier claims retain their assertions and read frozen #418 bytes where this
cut rewrites their inputs. Members 81b638e8, isolation 0ec14b39 and usage
b8a6e9e1 roots remain living; none is a claims input.

## Previous cut — handover 33 (#416)

[Follow-up 33](follow-up-33.md) covers **#413–#415 at 339db172**.
The actual `TestHandover_Followup33Claims` and manifests on main verify:
**16 subtests / three final rows / ten frozen handover inputs / 512 netlab
payload checks, including 42 gzip originals / three documents / three PR
excerpts / 81 findings / nine owner requests**. The three documents are
updater 21's two dated documents and the raw r22 review.

[PR #416](https://github.com/TED-H-com/Freedom-VPN/pull/416) records **one codex
`gpt-6-astra` session**, launched **06:20 EST**. Its first gate launch was
refused by admission during **netlab 30's reservation**. It waited
**1 min 48 s** and **ran no test inside that reservation**. The public excerpt
is frozen with its squash-commit provenance; the counts come from the code
and manifests.

## Updater 22 (#417) — F21-01 fixed in the fixtures by its mechanism

The persisted meter deadline is **wall-clock time from the block**. The tests'
own setup between the block and the engine's first read ran on that clock:
crash simulation, SQLite reopen, meter close, fsync publication, and publisher
or withdrawer goroutines. That setup could spend the budget before the first
checkpoint read; the product then correctly failed closed.

The parent `TestAT25_FileMeterWaitRestartRechecksCurrentFile`,
`TestAT25_FileMeterWaitFailsClosedWithoutCorrelation` and
`followup17_test.go`'s `correlatedApply` now run on a **synctest bubble clock**.
Disk and scheduler stalls consume real time but no bubble budget. **Budgets
and result, error, receipt and forwarding assertions are unchanged.** The
parent gains a **remaining-budget guard and a bubble bound**, so a deadline
renewal fails deterministically again. **No product change.**

[PR #417](https://github.com/TED-H-com/Freedom-VPN/pull/417) reports the controls
as **16 timing-dependent cases failing with meter_checkpoint_timeout before
the fix; all 26 passing after it**. The blocking-syscall overlays were removed.
There is a **counting discrepancy**: the dated timing JSON enumerates **15
failure labels**, and its fixed summary says **26 test/subtest results**.
The dated control table also totals 15 failing leaf cases. The PR's “16” is
retained as its statement, not independently promoted to a leaf-case count;
clarification is requested below. The mechanism and deterministic renewal
controls are documented independently of that count.

**The PR states** that the fix covers three F21-01-class CI failures on main,
**none re-run**: **41616194943** (`TestD125_RuleMalformedIdentity`),
**41620760976** (`TestAT25_FileMeterWaitFailsClosedWithoutCorrelation/withdrawn_reblocked`),
and **41625190790** (`TestD125_RuleLegacyFourMembers`). This is the PR's
statement, not a new handover CI result.

| Fixed-test leg | Iterations / failures | Qualification |
|---|---|---|
| Quiet | 200 / 0 | Qualified. |
| Bounded load, reservation 1 | 33 / 0 | Unqualified, stopped by non-lane foreign CPU. |
| Bounded load, reservation 2 | 50 / 0 | Unqualified, stopped by non-lane foreign CPU after the measured test exited. |

The deterministic budget stayed exact in **all 283 iterations**: remaining
250 ms for missing/replaced/withdrawn/new_boot, −1 ms for expired; bubble
elapsed 250 ms for missing/replaced and zero for the others. The independent
restart budget stayed **250 / 50 / 0 / 0 ms** before and after the windows.

**F27-02's fixture-defect reading for the fixed test is undecided**: quiet
qualified, load unqualified in both reservations. **Updater 21's verdict stays
failed**, and **netlab 27's F27-02 stays failed**. The original reading is
**supported, not closed**. No historical leg is requalified.

**Requests answered in #417:** the orchestrator does not pause jobs outside
this project. A later leg ends when the measured test process exits; foreign
CPU after it is recorded and does not disqualify. That rule must be declared
before the leg runs. **Updater 23 runs the one missing bounded-load leg** and
handles the other short wall-clock fixture windows; its results are outside
this cut.

Pins: [dated hand-off](../updates/node-followup22-2026-11-03.md),
[timing JSON](../updates/node-followup22-timing-2026-11-03.json), final #417 row,
PR excerpt and the four AT25/D125 test declarations above plus
`TestAT25_FileMeterWaitRestartKeepsDeadline`. Scope: UPD03 / UPD05 / MTR03,
§13.5–§13.9 / §19 AT25, component evidence only.

## Netlab 30 (#418) — guard-drop mechanism demonstrated; B1 still FAIL

The **collector prerequisite passed live on its second slot**. Eight expected
logs matched once each, with a separately counted unlogged negative control,
**8 captured / 8 received / zero kernel drops**. Every record has microsecond
precision and a kernel timestamp. The first slot stopped at preflight;
cleanup completed separately after the successful collector. This finite
check does not prove lossless logging at every load.

AT10's **100-reload scenario** completed **101 verified Applies, 118 TCP
reads and all 381 segments correlated**. Each of the **seven missing node
egresses** matches **exactly one fv_deadline_guard drop, member → responder**,
with no contradictory egress. Each lies inside its generation's
**install-command bracket**: **1002, 1018, 1019, 1041, 1069, 1070, 1093**.
**F27-01's guard-drop mechanism is demonstrated** in this window. TCP recovered
every time by retransmission; the worst read was **416.7 ms** (416.739885 ms).

**F27-01 stays unresolved, with no fix. B1 stays FAIL and unexplained.** Its
**1,501.4 ms failed read has no packet trace**. The successful window cannot
reconstruct its request, reply, retries or timer. Follow-up 28's gate 2 stays
FAIL; F28-01 remains closed as test isolation, as recorded by follow-up 33.

**Neither the guard window nor the full suite is host-qualified.** The guard
window's first interval reached **2.48 foreign cores**. The full suite had
**eleven intervals above 0.5 cores, at most 4.31**. **Every functional assertion
passed**; those passes are separate from host-qualified acceptance.

The guard's observed cost is **disclosed, not isolated**: install, release and
outer-guard medians **13.7, 13.2 and 28.5 ms**, outer maximum **94.7 ms**. These
are command brackets in an instrumented, unqualified run, not isolated
incremental overhead or an outage bound. A **hitless guard hand-over is a
proposal only**: **netlab 31 designs it; nothing is implemented**. Any later
implementation needs its own safety design and D-133 comparison.

**D-098:** **1,506 earlier payload files and five manifests** were verified
byte-identical at both ends. The [per-follow-up evidence tree](../poc/G2-followup30-evidence/overview.md)
has **312 payload files, including 19 gzip originals**, sealed by the final
[manifest](../poc/G2-followup30-evidence/retention-pins.json), SHA-256
**cd23561b8e978f126a213b6fa5ed3eb192ccc9de37058ad430c68e4788cd1254**.
The claims test verifies these retained hashes and lengths, both forms of
every gzip, and the structured collector, attribution, cost and qualification
records. Go declarations on main: `TestAT10_AT13_ReloadRevokeBaseline`,
`TestF27_GuardLogKeepsPredicatesAndDrops`,
`TestCommittedEvidenceFiles_ValidAndHygieneClean`. Scope: ISO01 / ISO04,
§8.1–§8.2 / §19 AT10 and D-104 / D-120 / D-133; no new acceptance claim.

## Current guidance, reviews and decisions

The seven entrypoints point at this cut. The operator guide states that
in-flight member TCP segments can be dropped while a generation installs,
and TCP recovers by retransmission in the observed window. It promises no
recovery bound beyond that evidence.

**F30-01 stays known behaviour, no merged fix**, as follow-ups 31–33 wrote it.
**r20 H-1's restart gap stays open on main while agent-core 37–39 are held**.
Agent-core 39's judgments and review r23 are outside this cut. All five r22
rows remain returned to agent-core 39, **none confirmed on main**. The r22
archive and follow-up 33 stop-budget erratum remain unchanged. Main records
no scheduled-run result; none is claimed here.

The [findings register](README.md#findings-register-at-the-follow-up-34-cutoff)
retains **81 rows**. Only **F21-01, F27-02 and F27-01 / B1** change;
**every other follow-up 33 row is byte-identical**. **Every historical failure
stays failed.** All **47 decision rows** are unchanged; **no decision was
added after D-144 on main**. **D-135, D-136, D-137, D-138, D-140, D-141,
D-142 and D-144 stay provisional**; **D-143 is Ted's decision**. The nine owner
requests below remain unchanged.

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

2026-11-03 — **Control-count clarification, outside owned paths:**
`docs/updates/node-followup22-timing-2026-11-03.json` lists 15 failing labels,
while #417 and its roadmap row report 16 failing cases. Its fixed summary
says 26 test/subtest results. Please reconcile the counting unit with the
updater owner, and clarify #417 / `docs/ROADMAP.md` or supply an additive
erratum. No producer evidence or roadmap row was changed here.

2026-11-03 — Already routed: updater 23 owns the missing bounded-load leg
and remaining short wall-clock fixture windows; netlab 31 designs a hitless
guard hand-over. Agent-core 39 carries 37/38 and awaits r23 before merge;
no judgments are imported. F30-01 remains agent-core / updater; F31-01
remains agent-core, design first. **Nine owner requests**, unchanged, plus
the evidence-count clarification above.
