# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 29](../handover/follow-up-29.md) (2026-11-01 EST):** Through #396: same-boot deny-set adoption gap open; isolation 15 unqualified on a 3.821 µs interleave, signal-15 cause unproven. Devstack 27 proves the TEST ONLY populated hold; consumer rows remain open. D-120 measurement end fixed; every failed run stays failed. D-135–D-138, D-140 and D-141 remain provisional. Seven owner decisions retained.
**Previous cut — [follow-up 28](../handover/follow-up-28.md):** #386–#390 at aae6763f; 19 subtests, five rows, 16 snapshots, 1,644 checks and 47 findings. Historical failures stay failed.

## Follow-up 29 acceptance snapshot (2026-11-01 EST)

Cut **#391–#396**, main **d640ce96**. Nothing was merged after the cutoff at task
start. [Evidence and seven owner requests](../handover/follow-up-29.md).

| Lane / PR | Result and boundary |
|---|---|
| Usage 8 (#391) | Nonzero ctdenyd changes no samples and yields none alone; complete download statements enforce resolver replies → source guards → d_ admission. Coverage only, no live run; b8a6e9e1 roots cited by row / PR. |
| Isolation 15 (#392) | 56d20530 archived. 0b9e4510 on #388: AT10 9 / 0 / 0 (ISO-F6-01 6 / 6), AT12 10 / 0 / 0; AT11 / AT13 operator blocks. EvaluateDeadline passes but the 3.821 µs interleave fails D-140 qualification. Whole-node row unqualified, F14-01 failed. Three agent exits 0, terminal source signal 15 with no exit code; cause unproven, agent-core 36. No populated hold. |
| Agent-core 35 (#393) | ctdenyd inert across usage consumers, no production change. Production-reachable same-boot adoption gap: old smaller deny/host sets can serve until first successful Apply; busy lock or control timeout can delay it. Agent-core 36 then netlab 26 routed; live-drift reading unconfirmed. |
| Devstack 27 (#394) | TEST ONLY hold seam: 13 / 0 / 0, two members, 50.000298 s, Step-3 alarm, one alarm-triggered escalation at 9.887 s, generation_matched recovery. Consumer rows remain open. Serving campaign 8 / 8, 58,619 sends, zero missing node ingress; 183 ingress-without-origin losses counted. ::1 type-4 / type-2 anomalies join owner F11-01 evidence; no historical cause assigned. |
| Handover 28 (#395) | #386–#390 at aae6763f; 19 subtests / five rows / 16 snapshots / 1,644 checks; 47 findings. Seven owner decisions retained. |
| Netlab 25 (#396) | D-120 measurement end fixed: final guard closes first, at most one lost-tail successor cycle, 15 s watchdog. Analyzer unchanged; A2, branch attempt and gate2-replay failed. One quiet-host run: 100 reloads / 101 verified Applies, zero UDP losses / TCP timeouts / extra cycles. Lost-tail proof offline only; lossless gzip pins both forms. |

D-135, D-136, D-137, D-138, D-140 and D-141 stay provisional; no new decision
after D-141. The [51-row findings register](../handover/README.md#findings-register-at-the-follow-up-29-cutoff)
keeps every historical failure failed. The node configuration warning promises
no safe procedure; the fix is routed. No later lane result is included.
