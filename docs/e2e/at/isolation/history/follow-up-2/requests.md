# Current requests — Follow-up 2

Run `422cdd6d-090b-4bb7-b92b-d4883259ac87`, source
`11698fc4d7205a5c902592660ccb346d7f4db750`, based on main `be2976c`.

1. **Agent and wire-nodes: investigate the serving/lease feedback during native
   admission.** Before any publication-withdrawal or revocation probe, the node
   lost its serving path. At `2026-10-30T17:30:34.707709Z`, its accepted report
   named generation 8, `policy_missing`, and lease `STALE_GENERATION`; the lease
   issued immediately beforehand named generation 9. Later leases for
   generations 9, 11 and 12 had zero members despite active entitlements.
   Generation 12 desired state had three bindings and two member policies.
   The node reported `lease_expired`, lease `LEASE_EXPIRED`, and QoS/source
   binding `VALIDATION_FAILED`; its actual forward chain was baseline generation
   0 and publication was observed `withdrawn`. All four dependencies were
   present. The sampled node row still said `ready` / safety `passed`.
   Preserve the distinction between the local forward chain, accepted checks
   and projected node status. See the complete [read-only snapshot](node-state-before-publication-probes.json)
   and current AT records. This predates the revocation test; it is not attributed
   to conntrack deletion or to a particular newly merged PR.
   The counted path later recovered at generation 13, before the harness's
   capacity PATCH. Its first valid control increased admission bytes from 0 to
   2632. Later checks pass, but the six earlier destination checks retain their
   missing-positive blocker. A [later snapshot](node-state-after-recovery.json)
   preserves the recovered state during the A/B publication probes.
2. **Devstack: supply the remaining assembled measurement paths.** A designated
   resolver on the member's Docker underlay cannot prove the exit's DNS
   exception. Controlled IPv6/member-platform coverage, authenticated stale or
   spoofed client injection, and persistent post-revoke node ingress remain
   needed. #170's native route-withdrawal limitation applies. The mechanism
   controls remain available, with independent positive controls.
3. **Wire-nodes: complete the D-065 fleet configuration handler.** #176 added
   the contract but left the generated fallback; the current run received
   501 `NOT_IMPLEMENTED` from `GET /admin/fleet-config`. Short whole-node lease
   expiry needs the public API. The finite-member deadline subset was measured
   successfully. No `fleet_config` or other projected row was changed.
4. **Acceptance/operator: complete the continuous fault/platform matrix** once
   the counted serving precondition is stable: crash, NetBird restart, reboot,
   updater transition and physical member-platform/arm64 runs.

No `node/enforce` bypass request is filed. The current assembled negatives lack
a counted positive wherever marked blocked; the mechanism target-ingress
captures show no bypass. A timeout or an outbound packet captured on a member
interface is insufficient evidence for an enforcement defect.
