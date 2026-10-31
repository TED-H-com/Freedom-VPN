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
