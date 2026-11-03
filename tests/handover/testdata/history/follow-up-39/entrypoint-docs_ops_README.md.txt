# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 38](../handover/follow-up-38.md) (2026-11-03 EST):** #427 only: handover 37 recorded; r25 archived, both findings returned to agent-core 42 before merge, not yet judged. Operator guidance unchanged: no hitless hand-over or C1 benefit; r24 H-1/H-3, F30-01 and r20 H-1 stay open on main. Ten owner requests.
**Previous cut — [follow-up 37](../handover/follow-up-37.md):** #424–#426 at 792e3e41; 16 subtests, three final rows, ten frozen inputs, three PR excerpts, two immutable documents, 138 archive payloads and separate source audit; 97 findings, 47 decisions, ten requests.

## Follow-up 38 acceptance snapshot (2026-11-03 EST)

Cut **#427 only**, main **77626af4**, immediately after #426; no unbriefed PR.
[Evidence and ten owner requests](../handover/follow-up-38.md).

| Item | Result and boundary |
|---|---|
| Handover 37 (#427) | #424–#426 at 792e3e41; 16 subtests, three final rows, ten frozen inputs, three PR excerpts, two immutable documents, 138 archive payloads and separate audit; 97 findings / 47 decisions / ten requests. Two commits, 30 files; 258 digest values only; no new request. |
| r25 archive | 31,868 bytes, unchanged; 0 Critical / 1 High / 1 Medium / 0 Low. Both RETURNED — not yet judged; agent-core 42 before merge. Reviewed 8cc27609 is not on main; no proof copied or acceptance result adopted. |

[99-row findings register](../handover/README.md#findings-register-at-the-follow-up-38-cutoff):
r25 H-1/M-1 added; all 97 prior rows unchanged. H-1's possible overlap with
main is an unanswered D-141 question for agent-core 42. No r25 operator warning.
Agent-core 37–41 were delivered and held; 42 was running since 11:36 EST at
briefing, also handling Netlab F32-01. No held-branch result enters this cut.
r24 H-1 stays open known behaviour on main; F30-01 stays known behaviour and
r20 H-1's restart gap stays open while agent-core 37–42 are held. Follow-up 37's
guidance stays: no hitless hand-over is implemented, no C1 benefit or fix date
promised. Production keeps the current guard pending the owner's G5/G6 decision.
#424's at-isolation fix is merged; #427 records no first scheduled exercise.
All 47 decisions and ten owner requests remain unchanged. Every historical
failure stays failed.
