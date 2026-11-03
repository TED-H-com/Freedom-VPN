# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 34](../handover/follow-up-34.md) (2026-11-03 EST):** Through #418: F21-01 fixed in fixtures; fixed-test F27-02 reading undecided. Seven guard drops attributed; B1 stays FAIL, both live runs host-unqualified. r22 unchanged; F30-01 and r20 H-1 open. Nine owner requests.
**Previous cut — [follow-up 33](../handover/follow-up-33.md):** #413–#415 at 339db172; 16 subtests, three rows, ten frozen inputs, 512 payload checks (42 gzip originals), three documents, three PR excerpts, 81 findings and nine requests.

## Follow-up 34 acceptance snapshot (2026-11-03 EST)

Cut **#416–#418**, main **cc0ed4ae**; no unbriefed PR inside the range or later
merge at task start. [Evidence and nine owner requests](../handover/follow-up-34.md).

| Lane / PR | Result and boundary |
|---|---|
| Handover 33 (#416) | #413–#415 at 339db172; 16 subtests / three rows / ten frozen inputs / 512 payload checks (42 gzip originals) / three documents / three PR excerpts / 81 findings / nine requests. One codex gpt-6-astra session, 06:20 EST; admission wait 1 min 48 s during netlab 30, no test inside the reservation. |
| Updater 22 (#417) | F21-01 fixed in test fixtures by removing setup’s wall-clock budget dependency; synctest, unchanged budgets/assertions, deterministic renewal guard. PR reports controls 16 failing before / 26 passing after; JSON lists 15 failing labels, clarification requested. Quiet 200/0 qualified; load 33/0 and 50/0 unqualified; budget exact in 283 iterations. Fixed-test F27-02 reading undecided; updater 21 and netlab 27 failures retained. Updater 23 owns the missing load leg. |
| Netlab 30 (#418) | Live collector prerequisite passed; 101 verified Applies / 118 reads / 381 segments. Seven missing egresses match seven guard drops inside install brackets; recovery worst 416.7 ms. F27-01 unresolved, no fix; B1 1,501.4 ms remains FAIL and unexplained. Guard and full suite host-unqualified (2.48 cores; eleven breaches, max 4.31) despite passing functional assertions. Cost disclosed, not isolated; hitless hand-over is netlab 31 design only. |

[81-row findings register](../handover/README.md#findings-register-at-the-follow-up-34-cutoff).
Only F21-01, F27-02 and F27-01 / B1 change; all other rows and all 47 decisions
are unchanged. Agent-core 39's judgments and r23 are outside the cut;
r22's five findings remain returned, none confirmed on main.
F30-01 stays known behaviour with no merged fix; r20 H-1's restart gap stays open.
Main records no scheduled-run result. Every historical failure stays failed.
