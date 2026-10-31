# Adversarial review archive — `r14-d067-d070-review`

- **Branch:** `r14-d067-d070-review`
- **Commit:** `70efbde6c94fb8d87f2cb88334703cfff4afe004`
- **Date:** `2026-10-30T17:16:53-04:00`
- **Reviewing model:** `grok/grok-4.7` (verified from the commit's `Agent:` trailer)
- **Source report:** `.agent/REVIEW.md`, blob `4a54791914db8704c053a3a91a8e1c60d5a4cc6b`, SHA-256 `96a379e53b8744f71b51322d36ec4f098a96feb2e6e2d9dd29c5e03484b82b60`.
- **Status snapshot:** `origin/main` at `d9e28f8`, 2026-10-30 EDT. Findings below describe the reviewed revision; closing PRs describe the subsequent owner fixes.

## Scope

The review covers the code that closed r13 and the three decisions taken while closing it: D-067 on the node (agent follow-up 15, #193), central readiness and revoke steps plus D-070 (wire-nodes follow-up 18, #194), D-068 null-only release seeding (fleet-update follow-up 9, #189), the forwarded-address rule and certificate-keyed ingress (control follow-up 19, #190), the enforce verifier (netlab follow-up 8, #196), updater recovery (follow-up 12, #192), and contract revision 10 / ENR-02 (#191, #195).

## Findings and disposition

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | **H-1** (wire-nodes): `otherNodePendingIso` holds a `member_device.revoke` for a device the dirty node never bound, including an enrolled node with no `exit_groups`, and a same-group peer that has been silent for an hour. | closed | #217 |
| Medium | **M-1** (agent): A v2 `applied_address` migrated with nil device and member ids is skipped by `planRevokes` when that address is reassigned, and the next successful apply erases the row. | closed | #216 |
| Medium | **M-2** (wire-nodes): After `provisioning_state` is `ready`, a repeated `safety=passed` report of an older applied generation stays `ready` while desired has moved on. | closed | #217 |
| Medium | **M-3** (netlab): `VerifyServing` accepts a second inet table with a higher-priority forward accept. The agent read-back lists only `inet fvpn`, and the parser drops every other table before verification. | closed | #220 |
| Medium | **M-4** (control): One node with two still-usable leaf certificates gets two ingress bursts. The key is `sha256(leaf.Raw)`, spent before `AuthenticateAgent`. Renewal overlap is what keeps the older leaves usable. | closed | #215 |
| Low | **L-1** (control): Eight depleted buckets at the LRU tail hide a full idle bucket. A newcomer is refused while the table is full. The comment on `evictIdleWindow` already states this. | closed | #215 |
| Low | **L-2** (control): One anonymous `/48` may hold `max/2` `/64` keys. The next `/64` in that prefix is refused while the principal table still has free slots. | closed | #215 |

## Adopted proof files on main

| Findings | Fix and retained proof |
|---|---|
| H-1, M-2 (`#217`) | `internal/nodes/observation_readiness_integration_test.go`: `TestReviewR14_ExitGroupFenceHoldsRevokesForNodesThatDoNotHaveTheDevice` and `TestReviewR14_RepeatedStalePassedReportKeepsReady`. Other-node revoke evidence is scoped to relevant groups and fresh observations (90 s); repeated stale reports cannot retain ready indefinitely. D-075 (`#228`) subsequently made the grace marker durable. |
| M-1 (`#216`) | `node/agent/review_r14_m1_test.go`: `TestReviewR14_NilUUIDMigrationShieldsReassignedAddress` and `TestReviewR14_NilUUIDMigrationKeepsPendingUntilDeleteSucceeds`. A migrated address with unknown identity is queued for deletion and remains pending until deletion succeeds. `node/agent/followup15_test.go` also covers the provisional lease behavior shipped in that wave. |
| M-3 (`#220`) | `node/enforce/followup10_review_test.go` (`TestReviewR14_VerifierModel`, `TestFollowup10_ForeignChains`) and `tests/netlab/scenario_foreign_test.go`. Apply/reverify read the whole ruleset. The owner refined the review recommendation: harmless foreign accept/match/counter/log chains are tolerated; dangerous foreign NAT, rewrite, and offload paths fail closed. D-079 (`#241`) subsequently classified proven loopback-only rules as `loopback_scoped`. |
| M-4, L-1, L-2 (`#215`) | `internal/httpapi/followup20_review_test.go`: `TestReviewR14_DistinctLeavesMultiplyOneNodeBudget`, `TestReviewR14_EvictIdleWindowOfEight`, `TestReviewR14_Anonymous48HalfBudget`. Ingress keys use the verified node URI SAN; idle eviction walks the bounded table; the anonymous half-table cap became a reserve of one. The later r15 L-1 questions that reserve policy and remains separately open. |

## What held and limits

The source review retained the 12-attempt conntrack fail-closed bound, D-068 null-only release seeding/current-job attribution, updater recovery receipts and pruning, and the corrected IPv6 forwarded-address parse rule. Its unit/integration proofs ran without a netlab. The whole-ruleset fix's later lab proof belongs to `#220`; this archive does not claim a new live run.
