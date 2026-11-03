# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 35](../handover/follow-up-35.md) (2026-11-03 EST):** Through #421: F27-02’s fixed-test reading holds; F23-01 fixed in fixtures; #417 count corrected to 15. Hitless guard design awaits the owner; production keeps the current guard. r23 returned to agent-core 40, none confirmed. F30-01 and r20 H-1 open. Ten owner requests.
**Previous cut — [follow-up 34](../handover/follow-up-34.md):** #416–#418 at cc0ed4ae; 14 subtests, three rows, ten frozen inputs, 312 payload hashes and lengths (19 gzip originals), two dated documents, three PR excerpts, 81 findings, 47 decisions and nine requests.

## Follow-up 35 acceptance snapshot (2026-11-03 EST)

Cut **#419–#421**, main **dff6bdb0**; no unbriefed PR inside the range or later
merge at task start. [Evidence and ten owner requests](../handover/follow-up-35.md).

| Lane / PR | Result and boundary |
|---|---|
| Handover 34 (#419) | #416–#418 at cc0ed4ae; 14 subtests / three rows / ten frozen inputs / 312 payload hashes and lengths (19 gzip originals) / two dated documents / three PR excerpts / 81 findings / 47 decisions / nine requests. One codex gpt-6-astra session launched 07:29 EST; admission and suite waits 0 s. Control-count request answered by #420. |
| Updater 23 (#420) | F27-02 fixture defect fixed by #417; fixed-test reading HOLDS: qualified load 50/0 plus quiet 200/0, exact budgets in 352 logged Recover calls. First load reservation 20/0 stays unqualified. F23-01 fixed on bubble clock; 250 ms stall control and first-attestation mutation preserve the assertion, all 176 package tests pass the stall control. No product change. Corrected #417 count: 15 failing cases, 26 test/subtest results. Two sessions, no new measurement by the completing codex session. Historical verdicts and failures retained. |
| Netlab 31 (#421) | C1 retained-deadline intersection guard recommended for design review only. G5/G6 closure trade is the owner's decision; G4/G8 difference explicit. Production keeps the current guard. Proofs and D-133 A,B,A,B declared, not run; r24 reviews first. B1 stays FAIL and unexplained; netlab 30 live runs stay host-unqualified. |

[87-row findings register](../handover/README.md#findings-register-at-the-follow-up-35-cutoff).
r23 archived exactly, returned to agent-core 40 before the combined 37–40
merge; none confirmed. r20/r21 remain open on main, r22 dispositions unchanged.
r24 delivered after the cutoff; no result imported. F30-01 stays known behaviour
with no merged fix; r20 H-1's restart gap stays open. Member TCP segments can
still drop during generation installation; no hitless hand-over is implemented.
Main records no scheduled-run result. All 47 decisions and every other
follow-up 34 finding row are unchanged. Every historical failure stays failed.
