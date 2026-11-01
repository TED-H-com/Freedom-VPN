# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 27](../handover/follow-up-27.md) (2026-11-01 EST):** Through #385: isolation F14-01 fails the D-104 boundary; members 15 adds serving-node ingress gaps; r19 archived with routed findings. D-140 permits boundary scheduling only. D-141 closes D-139: option (b) is not implemented; option (c) stays. Historical failures stay failed. No new operator guidance.
**Previous cut — [follow-up 26](../handover/follow-up-26.md) (2026-11-01 EST):** Through #382, including the #381 row correction: r18 fixes landed; ISO-F6-01 fixed and AT10 passes; F11-01 closed-node cause unproven; members F14-01 new; members F13-01 origin unproven; updates receipt delivered. D-139 is a provisional design track, option (c) stays in production. Historical failures stay failed.

## Follow-up 27 acceptance snapshot (2026-11-01 EST)

Fixed cutoff **37db5405**, **#383–#385**, with no merged-after-cutoff PR at task
start. Devstack 26, netlab 24 and agent-core 34 were running; no result from them
enters this cut. **Merged after the cutoff: #386.**
[Pinned evidence and decisions](../handover/follow-up-27.md).

| Delivery | Accepted reading at #385 |
|---|---|
| Isolation 14 (#383) | 38074149 archived: 112 roots. 56d20530 on #382: AT10 PASS 9 / 0 / 0, ISO-F6-01 6 / 6; AT12 PASS 10 / 0 / 0; AT11 BLOCKED 2 / 1 / 0. AT13 FAIL 5 / 1 / 1, isolation F14-01, distinct from members F14-01: closure (D−162.2, D−97.6] ms straddles D−101 ms. One zero-member bootstrap hold joined and ended generation_matched; no populated hold, alarm, escalation or active-stop retention exercised. Living roots cited by roadmap row / PR only. |
| Handover 26 (#384) | Previous #374–#382 cut at 9dde5eeb plus 5f4b455c: 21 subtests / 9 exact rows / 17 snapshots / 1,075 archive and run file checks retained. |
| Members 15 (#385) | 258ba504 archived: 70 roots. e80bb592: 2 pass / 6 blocked / 2 fail, 644 / 6 / 2 subchecks. AT02 FAIL: 93 successful Applies / 0 failed, 65 paired baselines, 277 / 4,131 attempts without egress (249 absent at ingress, 28 ingress without egress). AT01 first selection fails after generation-5 readiness: five member SYNs, zero node or origin packets; three re-grants pass. Members F14-01 stays failed historically. 682d4434 relay order live-qualified; receive-only dual MTU-1280 FAIL, 12,012 bytes in 12,011 ms. Gzip and plain AT02 index remain living, claims pinned by roadmap row / PR only. |
| r19 | Exact 41,050-byte archive. H-1 → netlab 24; M-2 / L-2 → agent-core 34, routed; in progress. M-2 production reachability unknown. Design-only findings are D-141 reconsideration conditions. |

D-140 permits only the boundary schedule change; the predicate stays byte-identical
and run 56d20530 stays failed. D-141 closes D-139: option (b) is not implemented;
option (c) stays. D-135 / D-136 / D-137 / D-138 remain provisional. Owner decisions:
D-135 + D-136 confirmation (AT02 sibling outage given D-141), D-137 hardening,
D-138 / D-140 / D-141 overturn options, F11-01 repair direction awaiting devstack
26. Serving-node missing-ingress samples route to devstack separately from the
closed-node timeouts; cause unproven. Historical failures stay failed.
**No new operator guidance.** Older snapshots below retain their own cutoff.
