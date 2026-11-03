# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 33](../handover/follow-up-33.md) (2026-11-03 EST):** Through #415: F27-02's declared fixture reading failed (F21-01); original reading supported, not closed. F28-01 closed as test isolation; B1 stays FAIL. r22 returned to agent-core 39, none confirmed. F30-01 and r20 H-1 remain open. Nine owner requests.
**Previous cut — [follow-up 32](../handover/follow-up-32.md):** #408–#412 at b7449ec1; 20 subtests, five rows, nine frozen inputs, 301 payload checks, seven documents, 19 symlinks, 14 navigation sources and 75 findings.

## Follow-up 33 acceptance snapshot (2026-11-03 EST)

Cut **#413–#415**, main **339db172**; no unbriefed PR inside the range or later
merge at task start. [Evidence and nine owner requests](../handover/follow-up-33.md).

| Lane / PR | Result and boundary |
|---|---|
| Updater 21 (#413) | Declared fixture-defect reading failed: corrected replaced setup took about 533 ms against 250 ms, Recover 15.1 ms; F21-01, no renewal. Quiet legs never qualified; both load legs 50 / 0 qualified. Original F27-02 reading supported, not closed; updater 22 owns the fixture and re-measurement. |
| Handover 32 (#414) | #408–#412 at b7449ec1; 20 subtests / five rows / nine frozen inputs / 301 checks (81 members, 220 netlab) / seven documents / 19 symlinks / 14 navigation sources / 75 findings / nine requests. One codex gpt-6-astra session, 05:22 EST launch; repeat gate waited seven minutes behind netlab 29. |
| Netlab 29 (#415) | F28-01 closed as test isolation: 57-byte retransmitted request + 40-byte RST from AT10's old admitted socket. AT12 closes old sockets; assertion unchanged, two full-suite passes with zero deny bytes. Follow-up 28 gate 2 stays FAIL. Single guard startup attempt failed, not retried; F27-01 unresolved, B1 FAIL, netlab 30 owns the next window. |

r22 is archived and returned to agent-core 39 before merge; none of its five
findings is confirmed. Agent-core 37/38 remain held with 39; r20's five
agent-core findings and r21's five remain open on main. The review's stop-budget
erratum is additive in the [reviews index](../reviews/README.md#r22-dispositions-at-the-follow-up-33-cutoff-2026-11-03-est).
[81-row findings register](../handover/README.md#findings-register-at-the-follow-up-33-cutoff).
F30-01 stays known behaviour with no merged fix; r20 H-1's restart gap stays open.
Main records no scheduled-run result. The nightly reading guide remains
applicable; every historical failure stays failed.
