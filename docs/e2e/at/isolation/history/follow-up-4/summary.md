# Follow-up 4 — serving blocked before ingress measurement

AT10, AT11, AT12 and AT13 stay **blocked**. No counted member byte was recorded. The aggregate run exits **2** after writing every product record. A Go failure here is the records-first gate. The four mechanism controls pass and do not promote a product AT.

Live run: `fa0074cc-3b1a-403d-b8c2-3a3439fe2b7e`, source
`491a307dd505e33b14dc0e993aa7e86b28e82f68` (`source_dirty=false`; main baseline `205633f`, which includes #216 agent follow-up 18, #219 devstack follow-up 13, #220 whole-ruleset `VerifyServing`, and #231 members follow-up 6).
Product UTC window: `2026-10-31T00:27:07Z`–`2026-10-31T00:31:43Z`.
Node incarnation `01a18f5a-7265-77d1-b36b-b40cff08d76d`.
Image IDs are in each JSON record. Mechanism lab `ok` in 18.198 s; `TestIsolationAssembled` 276.10 s.
`convergence-timeline.json` has no samples. The sampler starts only after publication succeeds.

| Product AT | Status / evidence | Counted positive control in the same record | Deciding observation |
|---|---|---|---|
| AT10 | blocked — [JSON](AT10.json), [Markdown](AT10.md) | none; publication never served | `serving setup` blocked. Owner `w3-devstack-e2e: node netns table ip nat POSTROUTING xt SNAT for 127.0.0.11`. Accepted observation at `2026-10-31T00:31:16.688055275Z`: lease, qos and source_binding `failed` / `FLOWTABLE_BYPASS`; reasons `["safety_check_failed"]`. `provisional lease reports AWAITING_MEMBER_GRANTS` blocked: agent follow-up 18 is on this main and the grant-less window was not sampled. |
| AT11 | blocked — [JSON](AT11.json), [Markdown](AT11.md) | none | `IPv6 platform matrix` stays blocked. Owner `w3-devstack-e2e: real IPv6 and platform paths are devstack backlog`. `harness-exports.md` exposes only `InjectMemberIngress` and `ObserveMemberIngress`. |
| AT12 | blocked — [JSON](AT12.json), [Markdown](AT12.md) | none | `designated DNS exception` blocked, owner `w3-devstack-e2e: resolver on member underlay`. `unknown/spoofed member source` blocked: `ObserveMemberIngress` was not called because serving setup failed. |
| AT13 | blocked — [JSON](AT13.json), [Markdown](AT13.md) | none | `post-revoke kernel ingress attribution`, `short whole-node lease expiry` and `lease member expiry` were not measured. The fleet-config PUT was not issued. `continuous crash/NetBird/reboot/updater transitions` stays owner `acceptance / operator`. |

## Why serving stopped

Before the publication PUT the node was healthy: provisioning `ready`, safety `passed`, desired and applied generation 1, lease `not_run` (`support/publish.md`, `2026-10-31T00:27:42.469995034Z`). Shared `runner.publish` returned 202 for operation `22ecf90f-847a-481d-a58c-8fcd45cb32b7` (`node.publish`) and that operation was still `running` after the poll, step `pending NODE_NOT_SERVICEABLE member_forwarding_unconfirmed`. The isolation publication PUT then returned `422 NODE_NOT_SERVICEABLE` with failed checks `provisioning_state=applying` and `safety_status=failed`.

`nft -j list ruleset` inside `fvpn-w4-isolation-node` at that moment, read only, showed Docker's embedded DNS NAT and no other foreign flowtable:

- `chain ip nat OUTPUT hook output prio -100 policy accept` jumps to `DOCKER_OUTPUT`, whose `xt` target is `DNAT` for `127.0.0.11`.
- `chain ip nat POSTROUTING hook postrouting prio 100 policy accept` jumps to `DOCKER_POSTROUTING`, whose `xt` target is `SNAT` for `127.0.0.11`.

`#220` `VerifyServing` reads the whole ruleset. A foreign `postrouting` chain that jumps to an `xt` statement fail-closes as `FLOWTABLE_BYPASS`. The agent log line is only that code; the table name is the `foreign_ruleset` observation on `serving setup`. The harness did not write `inet fvpn`, did not call `wg`, and did not delete the Docker table. Devstack run `4c97ebef` is not this result.

Follow-up 3's measured AT10 pass, expiry passes and convergence timeline stay under [history/follow-up-3](history/follow-up-3/). They were taken on `dde4aa6`, before `#220`.

## AT13 evaluator

The live AT13 record repeats `SHA256(concatenated capture.go, capture_review_test.go)=d4cd4dfd0957cace3ef9cd63168a4653c75ac1d0ba07d5c5ca1af00603f38d53`. The singular/plural parser from follow-up 2 stayed in force. Those two files were not edited.

| Mechanism control | Status | Assembled result it supports |
|---|---|---|
| [lab-AT10](lab-AT10.json) | pass | destination and interface/apply isolation; does not replace a counted product control |
| [lab-AT11](lab-AT11.json) | pass | IPv6 block mechanism; does not prove a real member-platform path |
| [lab-AT12](lab-AT12.json) | pass | peer/source/DNS isolation mechanism |
| [lab-AT13](lab-AT13.json) | pass | conntrack revocation and baseline/crash mechanisms; does not replace assembled ingress or expiry |

See [requests](requests.md) for owners and [RUN.md](RUN.md) for reproduction.
Follow-up 3 records are under [history/follow-up-3](history/follow-up-3/).
No enforcement bypass is claimed.
