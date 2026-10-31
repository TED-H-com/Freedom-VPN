# Current requests — Follow-up 5

Run `c0106e5d-d8e8-400b-abe0-181d6428fc55`, source
`7fa0443c532b1c9a395cd93620152fb6902584ae`, based on main `9e493e8` (#240).
`verify_foreign.go` in that tree is `2473acf` (#220).
`git log -1 origin/main -- node/enforce/verify_foreign.go` at record time is
`a4191fa` (#241, D-079). That commit is not an ancestor of the run.
Publication succeeded. The serving gate passed because follow-up 14 removed
`table ip nat` from the node before the agent started. `loopback_scoped` was
not emitted. The ruleset read-back has no Docker NAT rule.

1. **Provisional `AWAITING_MEMBER_GRANTS` was not sampled.** Owner `w2-agent-core`. Twelve convergence samples exist. The two grant-less samples (`2026-10-31T01:59:48Z` and `2026-10-31T01:59:50Z`) have `desired_bindings=0`, lease check `not_run`, and `error_code` null. At `2026-10-31T01:59:52Z` `lease_members` is already 1. No sample has `desired_bindings>0`, `lease_members=0`, and the same lease and desired generation. The check records `windows=0` and an empty code list. AT10 is subset passed, case blocked. The six destination cases passed on this run.

2. **Designated DNS exception did not land.** Owner `w3-devstack-e2e: resolver on member underlay`. The member route to `172.28.33.40` is `dev eth0`. No export places the designated resolver behind the exit. `harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress`. AT12 is subset passed, case blocked. The forged `100.64.7.7` source passed on this run (`ctunku` +1, zero forwarded bytes).

3. **IPv6 and member-platform rows did not land.** Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. Same export limit as request 2. AT11 is subset passed, case blocked. The metadata and deprecated-6to4 cases passed on this run.

4. **Acceptance/operator: the continuous fault matrix stays open.** Crash, NetBird restart, reboot, updater transition, and physical member-platform/arm64 runs are still owner `acceptance / operator`. AT13 is subset passed, case blocked. Post-revoke ingress, finite-member expiry, and the 60 s whole-node expiry (ceiling restored to 86400, both `fleet_config.update` audits) passed on this run.

No `node/enforce` change is requested. This run did not delete or rewrite nft. D-079 on later `origin/main` was not required for the pass: the lab node presented no Docker NAT rule for the classifier to scope.

# Current requests — Follow-up 4

Run `fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e`, source
`491a307dd505e33b14dc0e993aa7e86b28e82f68`, based on main `205633f`
(#216, #219, #220, #231).

1. **Devstack follow-up 14: Docker embedded DNS NAT in the node netns blocks serving.** Owner `w3-devstack-e2e: node netns table ip nat POSTROUTING xt SNAT for 127.0.0.11`. After publication, `nft -j list ruleset` in `fvpn-w4-isolation-node` shows `table ip nat` chain `POSTROUTING` (hook `postrouting`, priority 100, policy accept) jumping to `DOCKER_POSTROUTING`, whose statement is `xt` target `SNAT` for `127.0.0.11`. Chain `OUTPUT` jumps to `DOCKER_OUTPUT` with `xt` target `DNAT` for the same address. `#220` `VerifyServing` fail-closes that `xt` statement as `FLOWTABLE_BYPASS`. The accepted observation at `2026-10-31T00:31:16.688055275Z` has lease, qos and source_binding `error_code=FLOWTABLE_BYPASS` and `reasons=["safety_check_failed"]`. Shared publish operation `22ecf90f-847a-481d-a58c-8fcd45cb32b7` stayed `running` with `NODE_NOT_SERVICEABLE member_forwarding_unconfirmed`. The isolation PUT returned `422 NODE_NOT_SERVICEABLE`. This is why post-revoke ingress, the forged source, the convergence window, and both expiry measurements were not taken. The harness must not delete or rewrite that table. The needed capability is a node network namespace whose ruleset `VerifyServing` can accept, with this Docker DNS NAT absent from the listing `nft -j list ruleset` returns inside the node.

2. **Post-revoke kernel ingress attribution stays unmeasured.** The export landed: `Stack.ObserveMemberIngress` / `InjectMemberIngress` (`docs/e2e/scenarios/harness-exports.md`). It was not called. Owner is the serving gap in request 1, not a missing inject primitive. Devstack run `4c97ebef` is not an AT13 pass.

3. **Unknown/spoofed member source stays unmeasured.** The same export would inject forged `100.64.7.7` and the live member, with `ctunku` +1 and zero forwarded bytes for the forged source and `u_<hex>` +1 for the admitted source. `ObserveMemberIngress` was not called. Owner is request 1.

4. **Designated DNS exception did not land.** Owner `w3-devstack-e2e: resolver on member underlay`. No export places the designated resolver behind the exit. `harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress`.

5. **IPv6 and member-platform rows did not land.** Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. Same export limit as request 4.

6. **Provisional lease code was not sampled.** Agent follow-up 18 is on this main (`205633f` includes #216). The harness asserts lease-check `AWAITING_MEMBER_GRANTS` and block reason `awaiting_member_grants` on a grant-less same-generation sample. This run produced no sample: publication never succeeded, and `convergence-timeline.json` has `samples: null`. The recorded code is null. Build `491a307dd505e33b14dc0e993aa7e86b28e82f68`.

7. **Whole-node and finite-member expiry were not re-measured.** The fleet-config PUT was not issued, so the ceiling was not changed and was not restored by this run. Owner is request 1. Follow-up 3's passing measurements remain under [history/follow-up-3](history/follow-up-3/).

8. **Acceptance/operator: the continuous fault matrix stays open.** Crash, NetBird restart, reboot, updater transition, and physical member-platform/arm64 runs are still owner `acceptance / operator`.

No `node/enforce` bypass request is filed. `#220` fail-closed on the ruleset it read. Deleting `table ip nat` or weakening `VerifyServing` is not an isolation-harness change.

## Devstack follow-up 15 response (2026-10-31)

Request 2: `ObserveDesignatedDNS` is exported through `labstack`. The prepared product configuration names the isolated origin-network resolver at port 53; its TCP/UDP forwarder reaches the node's filtering AdGuard. The member cannot use its Docker underlay for this destination. The scenario requires replies and both device DNS counters to move while Internet counters hold. This closes the lab's missing designated exit path, not the full AT12 case or native-client fallback.

Request 3: `member-ipv6` records the native member's IPv6 configuration and remains blocked. A dual-stack native fixture, IPv6 underlay route/capture matrix and per-platform leak measurements have not been provisioned. An interface address alone is never a pass. IPv4-only mode's `ctv6` mechanism tests in netlab do not substitute for AT11's client-platform evidence. Owner remains devstack/platform operators.

Request 1 remains agent-core's provisional-grant observation window. Request 4's continuous crash/restart/reboot/update/platform campaign remains acceptance/operator work; the new established transport helpers can measure active TCP and UDP during those faults. The control-loss scenario now uses those helpers. The node entrypoint retains deletion of Docker's `ip nat`; it does not claim D-079 classifier evidence, which belongs to netlab.
