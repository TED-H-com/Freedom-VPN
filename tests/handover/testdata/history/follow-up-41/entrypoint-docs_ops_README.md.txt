# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 40](../handover/follow-up-40.md) (2026-11-03 EST):** #430–#434 at 3f7dbc25: D-146/D-147 on main (#432); members 18’s three receive-only TCP rows pass under provisional D-151, while AT09 stays failed. Isolation 19’s four source sets pass in the lab; assembled checks blocked by capacity PATCH 412. Agent-core 44 has no comparison; 45 repeats. D-152 answers F19-01; devstack 32 and members 19 requests stay open. 55 decisions.
**Previous cut — [follow-up 39](../handover/follow-up-39.md):** #428–#429 at 1b87ab32; 11 subtests, two final rows/PR excerpts, ten frozen inputs, exact r26 and separate audit, 15 dated documents/75 Go tests; 99 findings, 53 decisions, all ten owner requests answered.

## Follow-up 40 acceptance snapshot (2026-11-03 EST)

Cut **#430–#434**, main **3f7dbc25**, immediately after #429; five briefed
PRs, no unbriefed PR. [Evidence and provisional interpretations](../handover/follow-up-40.md).

| Item | Result and boundary |
|---|---|
| Handover 39 (#430) | 11 subtests; two final rows/PR excerpts, ten frozen #428 inputs, exact r26/separate audit, 15 dated documents/75 Go declarations; 53 decisions/99 findings. Additive baselineDecision range correction changes no finding or disposition. |
| Agent-core 44 (#431) | No scenario reached; zero pooled legs per arm, no comparison. Foreign-container preflight then clock-reader wrapper failure. Repaired and regression-tested; 45 repeats with the same-clock condition. |
| Netlab 34 (#432) | D-146 MSS 1160 and D-147 narrow RELATED fragmentation-needed on main. Fixed IPv4 lab and 13 ordinary agent scenarios pass; initial consumer failure retained, translator-only continuation and gate 3a green. Standard WireGuard headers, new handshakes, IPv4 forwarding; UDP/QUIC receive-only limitations remain. |
| Members 18 (#433) | Run 1 retained: 0/3/7, retire-reader rejection, no native matrix. Reader fixed; authorized run 2: 2/6/2, subchecks 699/5/4. Three receive-only TCP rows qualify under D-151; AT02/AT09 still fail, every historical F8-02 stays failed. |
| Isolation 19 (#434) | Four D-147 source sets and existing lab rows pass. Reader fixed before assembled run; all 12 assembled checks blocked at capacity PATCH 412, real-node D-147 not exercised. F19-01 answered by D-152; F19-02 open with devstack 32. |

[102-row register](../handover/README.md#findings-register-at-the-follow-up-40-cutoff):
four updated, three added, 95 unchanged. **55 decisions**; D-151/D-152 are
provisional and the owner may overturn them. Agent-core 45, devstack 32 and
members 19 were running at briefing; status only, no results in this cut.
Members 18's grant-restoration request stays open with members 19. No scheduled
exercise, complete platform acceptance, C1 implementation or comparison is
claimed. Every historical failure stays failed. Earlier snapshot sections
below describe their own cuts.
