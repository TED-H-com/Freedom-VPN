# Follow-up 5 — publication served; three cases stay blocked

AT10, AT11, AT12 and AT13 stay **blocked**. Each blocked AT has a subset that passed and one case that stayed blocked. The aggregate Go test fails after writing every product record. Mechanism passes do not promote a blocked product result.

Live run: `c0106e5d-d8e8-400b-abe0-181d6428fc55`, source
`7fa0443c532b1c9a395cd93620152fb6902584ae` (`source_dirty=false`).
Main baseline of that commit is `9e493e8` (`roadmap: w3-devstack-e2e follow-up 14 merged (#240)`).
`node/enforce/verify_foreign.go` in the built tree is `2473acf` (#220).
`git log -1 origin/main -- node/enforce/verify_foreign.go` at record time is
`a4191fa` (`enforce: tolerate loopback-scoped foreign nft rules (D-079)`, #241).
That commit is not an ancestor of this run.
Product UTC window: `2026-10-31T01:58:35Z`–`2026-10-31T02:13:10Z`.
Node incarnation `01a18fae-3353-7c1d-8c83-28a8fba176fd`.
Image IDs are in each JSON record. Mechanism lab `ok` in 18.262 s; `TestIsolationAssembled` 875.17 s.
`make -f mk/at-isolation.mk at-isolation` exited 1 at the prefix leftover check.
The Go test had already written the records. `fvpn-w4-isolation-isolation-target` was still attached to `fvpn-w4-isolation-origin`, so `fvpn-w4-isolation-cache-node:local` could not be removed.

| Product AT | Status / evidence | Counted positive control in the same record | Deciding observation |
|---|---|---|---|
| AT10 | blocked — subset passed, case blocked — [JSON](AT10.json), [Markdown](AT10.md) | six destination cases and the interface change, device `01a18faf-79c7-795e-ae29-a7941d760977`, bytes `0`→`2633` at `2026-10-31T02:00:37Z` | Destination cases passed (`destination_positive_controls` `without_positive=` empty). `provisional lease reports AWAITING_MEMBER_GRANTS` blocked: `windows=0`, `codes=` empty. Owner `w2-agent-core`. |
| AT11 | blocked — subset passed, case blocked — [JSON](AT11.json), [Markdown](AT11.md) | metadata and deprecated-6to4, same counted device | `IPv6 platform matrix` stays blocked. Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. |
| AT12 | blocked — subset passed, case blocked — [JSON](AT12.json), [Markdown](AT12.md) | peer, management, forged source, and publication-withdrawal checks | `designated DNS exception` blocked. Resolver `172.28.33.40` still routes `dev eth0`. Owner `w3-devstack-e2e: resolver on member underlay`. Forged `100.64.7.7` passed: `ctunku` +1 and zero forwarded bytes; admitted `u_01a18faf79c7795eae29a7941d760977` +1. |
| AT13 | blocked — subset passed, case blocked — [JSON](AT13.json), [Markdown](AT13.md) | disable, revoke, post-revoke ingress, restart, finite-member expiry, whole-node expiry | `continuous crash/NetBird/reboot/updater transitions` stays blocked. Owner `acceptance / operator`. Post-revoke: revoked `100.93.66.239` moved only `ctunku` by 1 packet / 29 bytes; sibling `u_01a18faf79c7795eae29a7941d760977` +1 and `d_01a18faf79c7795eae29a7941d760977` +1. Whole-node expiry passed at ceiling 60, then restored 86400. |

## What let publication succeed

Devstack follow-up 14 (PR #240, base `9e493e8`) is the change that let this run pass the serving gate. The node entrypoint deletes `table ip nat` before the agent starts. Publication `2d52a3de-5fc9-4dbb-b85e-b931e1ece186` (`node.publish`) succeeded at `2026-10-31T01:59:18Z`. The accepted read-back at `2026-10-31T01:59:47.09586838Z` (`report_id` `01a18faf-3697-7d4b-8bd9-7ea44b5ae7e5`) has `safety_status=passed`, `provisioning_state=ready`, `member_forwarding_enabled=true`, `forwarding_block_reasons=[]`, and lease/qos/source_binding `not_run`. No check carries `FLOWTABLE_BYPASS`.

D-079 was not in the binary. `loopback_scoped` does not appear in the observation report or in `verify_foreign.go` at `2473acf`. The AT10 observation `d079_classifier` is the string the run stored:

`verify_foreign.go ; loopback_scoped_in_source=false; this main's observation report has no foreign-rule classification field`

The git subject in that string is empty because `d079Note` ran `git log` and `ReadFile` from the test working directory `tests/e2e`. The `loopback_scoped_in_source=false` half matches the tree that was built. The classifier state of the run is `2473acf`. The later origin/main tip `a4191fa` was not built.

## Ruleset read-back

`foreign_rules` on AT10, verbatim:

```json
[{"table":{"family":"inet","name":"fvpn","handle":2}}]
```

`foreign_rule_classification`, verbatim:

`foreign rules are listed verbatim; none is Docker 127.0.0.11 NAT; D-079 is not in this tree so the observation report has no per-rule classification`

The single listed object is the product's own `inet fvpn` table declaration. The helper reads JSON field `table`, and a table object stores its name in `name`, so the declaration was kept. `lab-AT10-ruleset.nft` contains one table, `table inet fvpn`, including that table's own postrouting SNAT. It contains no `DOCKER_` chain and no `127.0.0.11`. The full `node_observations.report_payload` is AT10 observation `observation_report`. The harness did not write nft.

## Measured subchecks

Convergence: 12 samples in [convergence-timeline.json](convergence-timeline.json), from `2026-10-31T01:59:48.48183166Z` until the first counted byte at `2026-10-31T02:00:37Z` (`0`→`2633` on `01a18faf-79c7-795e-ae29-a7941d760977`). Early samples have `lease_members=0` and `desired_bindings=0`, with lease check `not_run` and `error_code` null. At `2026-10-31T01:59:52.509632594Z` `lease_members` is already 1. No sample has `desired_bindings>0`, `lease_members=0`, and `lease_generation` equal to `desired_generation` together, so the provisional assertion recorded `windows=0`.

Finite-member expiry ran first, while `lease_max_offline_seconds` was still 86400. Control stopped at `2026-10-31T02:11:12Z`, grant deadline `2026-10-31T02:11:36Z`. Five post-deadline TCP attempts timed out, each captured on `fvmb0` and absent from `fvup0`. Admission bytes stayed `2692`.

Whole-node expiry then `PUT` `/api/v1/admin/fleet-config` to 60 (`200`, etag `"3"`, `2026-10-31T02:11:59.302654Z`). The renewed lease is generation 28, `max_offline_seconds=60`, `member_count=2`, `expires_at=2026-10-31T02:13:04.059177147Z`. Forwarding still counted near that deadline. Baseline returned at `2026-10-31T02:13:07.082353801Z` (agent tick 5s; `early_baseline_at` is zero). Post-deadline ingress was on `fvmb0`, nothing on the uplink, admission bytes flat at `63560`. The ceiling was restored to 86400 (`200`, etag `"4"`, `2026-10-31T02:13:10.733645Z`). Both audits are `fleet_config.update` / `succeeded`: `01a18fba-62c5-7967-bccd-a7deede7af0d` (86400→60) and `01a18fbb-79cc-79ac-a845-149d6efe7c8d` (60→86400).

Post-revoke inject used `tun-mirred` on `fvmb0`, link type 65534, UDP/9. The revoked source's counter delta is only `ctunku` +1. The admitted sibling's `u_` counter moved by 1 and its `d_` counter moved by 1.

## AT13 evaluator

The live AT13 record repeats `SHA256(concatenated capture.go, capture_review_test.go)=d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`. The singular/plural parser from follow-up 2 stayed in force. Those two files were not edited.

| Mechanism control | Status | Assembled result it supports |
|---|---|---|
| [lab-AT10](lab-AT10.json) | pass | destination and interface/apply isolation; does not replace a counted product control |
| [lab-AT11](lab-AT11.json) | pass | IPv6 block mechanism; does not prove a real member-platform path |
| [lab-AT12](lab-AT12.json) | pass | peer/source/DNS isolation mechanism |
| [lab-AT13](lab-AT13.json) | pass | conntrack revocation and baseline/crash mechanisms; does not replace assembled ingress or expiry |

See [requests](requests.md) for owners and [RUN.md](RUN.md) for reproduction.
Follow-up 4's fail-closed records are under [history/follow-up-4](history/follow-up-4/).
Follow-up 3's measured passes stay under [history/follow-up-3](history/follow-up-3/).
No enforcement bypass is claimed.
