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
