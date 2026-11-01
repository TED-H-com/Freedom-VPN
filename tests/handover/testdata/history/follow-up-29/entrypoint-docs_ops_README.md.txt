# Operations Runbooks Index (`docs/ops/`)

**Current cut — [follow-up 28](../handover/follow-up-28.md) (2026-11-01 EST):** Through #390: r19 H-1 / M-2 / L-2 fixed; devstack 25 closure explained as tracer interference; native delivery delay remains. Updates 14 adopts parent receipts, AT23–AT27 5 / 5. D-135–D-138, D-140 and D-141 remain provisional. Node diagnostics and traceroute guidance added; RELATED ICMP owner question open. Historical failures stay failed.
**Previous cut — [follow-up 27](../handover/follow-up-27.md) (2026-11-01 EST):** Through #385: isolation F14-01 fails the D-104 boundary; members 15 adds serving-node ingress gaps; r19 archived with routed findings. D-140 permits boundary scheduling only. D-141 closes D-139: option (b) is not implemented; option (c) stays. Historical failures stay failed. No new operator guidance.

## Follow-up 28 acceptance snapshot (2026-11-01 EST)

Fixed main **aae6763f**, **#386–#390**. Devstack 27 and isolation 15 were running
at task start; no later results enter this cut. [Evidence and dated requests](../handover/follow-up-28.md).

| Delivery | Accepted reading at #390 |
|---|---|
| Devstack 26 (#386) | Devstack 25's closed node was its TEST ONLY foreign nft tracer: ISOLATION_POLICY_INVALID, deny baseline, kernel 0 / central applied 5. Harness artifact, not a product defect. Corrected member-only 4 / 0; F14-style re-grants 17 / 1, af718646 delivery 5.671 s late behind a loopback-source response. Native delay remains; owner repair/upstream direction unblocked. duplicate_chain classification and 15 s stop path with PID1 TERM forwarding delivered. Dated run evidence pinned. |
| Handover 27 (#387) | #383–#385 at 37db5405; 19 subtests / 3 exact rows / 14 snapshots / 182 archive file checks retained. #381 live row equals its frozen snapshot, sha256 400d5cf9…; no action. |
| Agent-core 34 (#388) | r19 M-2 production-reachable, fixed by applied snapshot committed with ACK and generation-matched refs. L-2 fixed by durable reservation before Apply; about 1.1 ms, p99 1.59 ms before restricting install when deletes are queued. Accepted under D-138, cost explicit. |
| Netlab 24 (#389) | r19 H-1 fixed; download denied/host/resolver source guards precede generic established / related accepts, exact DNS replies stay ahead. ctdenyd is diagnostic_v1, never usage. A,B,A,B accepted under D-138; no B-only failure. A2 D-120 on unchanged main stays failed; both missing-successor runs route to netlab 25. RELATED ICMP question is open, not a defect or decision. |
| Updates 14 (#390) | 759d244a archived: 48 roots. 2704600f on #387's tree: AT23–AT27 PASS 5 / 5; installer 27 / 27; 13 POSTs, AT26 physical rollback, uncached fallback. All 11 stops returned 0; six separate agent exit-0 parent receipts (AT23 first is a new sample), five repeat reuse. No SIGKILL attempt; stored lease present at every stop/start. Parent receipt and 15 s path adopted. Current run cited by roadmap row / PR only until updates 15 archives it. Updates F13-01 remains the unfilled historical gap, unrelated to members F13-01. |

No new decision since D-141; D-135 / D-136 / D-137 / D-138 / D-140 / D-141
remain provisional. Members 15's serving-node signature remains with devstack
27, cause unproven. New operator guidance: diagnose from the member side or
capture without changing the node ruleset; the verifier closes an invalidated
node. Traceroute gets no reply from spec-denied on-path sources under the new
source guard; owner question open, no fix promised. Historical failures stay
failed. Older snapshots below retain their own cutoff.
