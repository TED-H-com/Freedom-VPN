# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 36](../handover/follow-up-36.md) (2026-11-03 EST):** Through #423: r24 enforcement fixes merged; C1 revised as documents only, owner G5/G6 trade undecided, production keeps the current guard. r24 H-1/H-3 returned to agent-core 41; jitter open. F30-01 and r20 H-1 open. Ten owner requests.
**Previous cut — [follow-up 35](../handover/follow-up-35.md):** #419–#421 at dff6bdb0; 16 subtests, three rows, ten frozen inputs, three documents, six design files, three PR excerpts and separate source audit; 87 findings, 47 decisions, ten requests.

## Follow-up 36 acceptance snapshot (2026-11-03 EST)

Cut **#422–#423**, main **d7867928**; no unbriefed PR inside the range or later
merge on main at task start. [Evidence and ten owner requests](../handover/follow-up-36.md).

| Lane / PR | Result and boundary |
|---|---|
| Handover 35 (#422) | #419–#421 at dff6bdb0; 16 subtests, three final rows, ten frozen inputs, three documents, six design files, three PR excerpts, separate source audit, 87 findings, 47 decisions and ten owner requests. One codex gpt-6-astra session launched 08:19 EST, admission/suite waits 0 s. r23 L-1 code-trace clarification answered; no proof added. |
| Netlab 32 (#423) | D-141 judgments precede fixes: H-4 rollback, H-2 tc order, M-2 post-shape rollback fixed. One full lab passes. First consumer gate fails seven fixture tests; test-only resume repairs simulator, unchanged gate passes, no production change in resume. C1 revision is documents only; production keeps the current guard. B1 stays FAIL/unexplained, netlab 30 host-unqualified. |

[96-row findings register](../handover/README.md#findings-register-at-the-follow-up-36-cutoff).
r24 archived exactly; H-1/H-3 returned to agent-core 41, not yet judged by that
lane on main. Its recommendation to accept C1 is the reviewer's; the G5/G6
owner decision remains open. Netlab F32-01 jitter is routed after 41. The
revision's lock-path discrepancy is returned for an additive correction.
F30-01 stays known behaviour; r20 H-1's restart gap stays open while agent-core
37–41 are held. A serving node re-installs unchanged enforcement on each
successful nominal ten-second poll. The 28.5 ms median / 94.7 ms maximum
measured windows are outer command brackets, not exact outages. Netlab 32
changed install order, not the guard; member TCP segments can still drop.
No hitless hand-over is implemented and no date is promised for agent-core 41.
This cut claims no scheduled-run result; isolation 18's nightly diagnosis is
outside the cut. All 47 decisions and the other 86 finding rows are unchanged.
Every historical failure stays failed.
