# Adversarial Review r14: provisional leases, revoke completion, and D-067 / D-068 / D-070

Branch: `r14-d067-d070-review` (from `main` at `bf652d8`)
Date: 2026-10-30
Reviewing model: `grok/grok-4.7`

Scope is the code that closed r13 and the three decisions taken while closing it: D-067 on the node (agent follow-up 15, #193), central readiness and revoke steps plus D-070 (wire-nodes follow-up 18, #194), D-068 null-only release seeding (fleet-update follow-up 9, #189), the forwarded-address rule and certificate-keyed ingress (control follow-up 19, #190), the enforce verifier (netlab follow-up 8, #196), updater recovery (follow-up 12, #192), and contract revision 10 / ENR-02 (#191, #195).

Proofs are `TestReviewR14_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. Known in-flight work is reconfirmed there and is not filed again.

Commands (2026-10-30, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`):

```
go test -count=1 -timeout 180s -run 'TestReviewR14_' ./node/agent/ ./node/enforce/ ./node/updater/ ./internal/httpapi/ ./internal/audit/
go test -count=1 -tags integration -timeout 300s -run 'TestReviewR14_' ./internal/nodes/ ./internal/fleetupdate/
```

`node/updater`, `internal/audit`, and `internal/fleetupdate` exited 0. The other four packages exited 1 only on the findings below. Integration tests used their own databases through `dbtest` and logged the database name. The netlab was not started.

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| H-1 | High | wire-nodes | `internal/nodes/observations.go:723-744`, continue at `891` | `otherNodePendingIso` holds a `member_device.revoke` for a device the dirty node never bound, including an enrolled node with no `exit_groups`, and a same-group peer that has been silent for an hour. |
| M-1 | Medium | agent | `node/agent/revocation.go:43-70`, `node/internal/agentstore/store.go:116-121` | A v2 `applied_address` migrated with nil device and member ids is skipped by `planRevokes` when that address is reassigned, and the next successful apply erases the row. |
| M-2 | Medium | wire-nodes | `internal/nodes/observations.go:237-248` | After `provisioning_state` is `ready`, a repeated `safety=passed` report of an older applied generation stays `ready` while desired has moved on. |
| M-3 | Medium | netlab | `node/enforce/parse_nft.go:104-154`, `node/enforce/apply.go:97` | `VerifyServing` accepts a second inet table with a higher-priority forward accept. The agent read-back lists only `inet fvpn`, and the parser drops every other table before verification. |
| M-4 | Medium | control | `internal/httpapi/middleware.go:444-468`, `809-814`; `internal/pki/service.go:159-160` | One node with two still-usable leaf certificates gets two ingress bursts. The key is `sha256(leaf.Raw)`, spent before `AuthenticateAgent`. Renewal overlap is what keeps the older leaves usable. |
| L-1 | Low | control | `internal/httpapi/middleware.go:207-212`, `276-291` | Eight depleted buckets at the LRU tail hide a full idle bucket. A newcomer is refused while the table is full. The comment on `evictIdleWindow` already states this. |
| L-2 | Low | control | `internal/httpapi/middleware.go:232`, `265-274`, `792-807` | One anonymous `/48` may hold `max/2` `/64` keys. The next `/64` in that prefix is refused while the principal table still has free slots. |

## Findings

### H-1 (High): an exit with no groups, or a silent peer, holds revokes for devices it does not have

- **Where:** `internal/nodes/observations.go:723-744` (`otherNodePendingIso`), continue at `internal/nodes/observations.go:891`. Enrollment inserts the `exits` row at `internal/enrollment/service.go:266`. The pending step the operator reads is written at `internal/nodes/observations.go:830-835`.
- **Scenario:** Completing a revoke's `nodes` step requires that no other registered node is forwarding with `recent_errors` containing `ISOLATION_POLICY_INVALID`. The EXISTS matches that other node when it has no `exit_groups` row, when the member's `exit_group_ids` is empty, or when a group overlaps. It does not look at the device, and it does not look at `last_observed_at`.

  Every new node gets an `exits` row at enrollment. A node enrolled with no member groups has that row and no `exit_groups` row, so the `NOT EXISTS` arm is true for every member revoke. One such node, forwarding, with a pending delete of addresses that are not this device, keeps every `member_device.revoke` at `nodes` status `pending`, code `ISOLATION_POLICY_INVALID`, message `member_forwarding_unconfirmed`, reason `conntrack_delete_pending`. The member operation read stays `running`.

  The same EXISTS has no freshness bound. A peer that shared the device's exit group, reported the pending delete, and then never reported again still matches an hour later. The clean node in the group has already applied the revoke.

  A dirty peer published only to a different group, with the member plan limited to the clean node's group, does not hold the revoke. That boundary is the passing subtest under What held. Wire-nodes follow-up 20's 60 s grant-less expiry does not cover this fence.
- **Proof:** `TestReviewR14_ExitGroupFenceHoldsRevokesForNodesThatDoNotHaveTheDevice` in `internal/nodes/review_r14_observation_integration_test.go`.

  ```
  unpublished_node_holds_a_device_it_never_bound
  review_r14_observation_integration_test.go:350:
  TODO(review): member_device.revoke 01a18ea3-0217-7e07-80aa-7f165b71a870 for a device bound only to 01a18ea3-01dc-766e-822a-21475108fe9c stayed running (status=pending code=ISOLATION_POLICY_INVALID message=member_forwarding_unconfirmed); member read running; unpublished 01a18ea3-021d-71b2-809e-b3a1613719eb has no exit_groups and a pending delete while forwarding

  silent_same_group_peer_holds_the_revoke_an_hour_later
  review_r14_observation_integration_test.go:394:
  TODO(review): peer 01a18ea3-0525-7313-8843-f1051c3b49bc last reported at 2026-10-30T21:06:54Z and then stayed silent; an hour later member_device.revoke 01a18ea3-0560-7054-850d-d51bebfc4163 is still running (status=pending code=ISOLATION_POLICY_INVALID message=member_forwarding_unconfirmed)
  ```
- **Fix:** wire-nodes. Match the other node on a shared exit group, and stop matching an observation that has gone stale. An enrolled exit with no `exit_groups` row must not satisfy the fence for every member.

### M-1 (Medium): a nil-UUID migrated address is never queued for `conntrack -D` after reassignment

- **Where:** `node/agent/revocation.go:43` (`known`), skip at `node/agent/revocation.go:64-70` (`planRevokes`). The migration that writes the nil ids is `node/internal/agentstore/store.go:116-121` (`migrateAppliedAddressIdentity`).
- **Scenario:** v2 `applied_address` rows are copied forward with both `device_id` and `member_id` nil. `known()` is false for that row. `planRevokes` then returns without queueing the address while any current binding holds it, because the row cannot name the previous device. The same overlay address reassigned to a new device before the first v3 `SaveAddresses` therefore produces an apply with an empty revoke list. That apply succeeds and `SaveAddresses` replaces the nil row with the new `(addr, device_id, member_id)`. The delete is not retried.

  The twelve-attempt conntrack bound is a separate path and held. See What held.
- **Proof:** `TestReviewR14_NilUUIDMigrationShieldsReassignedAddress` in `node/agent/review_r14_agent_test.go`.

  ```
  review_r14_agent_test.go:391: TODO(review): planRevokes did not queue conntrack -D for a nil-UUID address reassigned to device 019a0000-0000-7000-8000-0000000000b2 (revoke=[])
  review_r14_agent_test.go:421: TODO(review): engine apply did not queue conntrack -D after the migrated address moved to device 019a0000-0000-7000-8000-0000000000b2 (revoke=[] serving=true)
  review_r14_agent_test.go:433: TODO(review): the nil-UUID row was replaced by the new identity without a conntrack delete (rows=[{Addr:100.64.1.1 DeviceID:019a0000-0000-7000-8000-0000000000b2 MemberID:019a0000-0000-7000-8000-0000000000c2}])
  ```
- **Fix:** agent. Queue `conntrack -D` for a nil-identity row when the address is now bound to a different device, and keep the row until that delete succeeds.

### M-2 (Medium): a repeated stale `safety=passed` report keeps `provisioning_state=ready`

- **Where:** `internal/nodes/observations.go:237-248`. The predicate is `($2::numeric>0 AND ($2::numeric>=n.desired_config_generation OR n.provisioning_state='ready'))`.
- **Scenario:** The OR is the concurrent-advance contract: one in-flight passed report of a generation that was already applied stays `ready` when desired moves. The same predicate then accepts every later report. A second report that still says `safety_status=passed` for applied generation 1, after desired has moved to 5, leaves `provisioning_state=ready` and `safety_status=passed`.

  `ServingReadiness.IsServing` (`internal/nodes/transitions.go:125-131`) still requires `applied >= desired`, so the composite serving flag is false. `CheckPublishable` (`internal/nodes/transitions.go:144-148`) reads `provisioning_state` and `safety_status` and does not read the generation lag, so a gate that only asks whether the node is publishable still sees a ready, passed node.

  A report of the same old generation with `safety_status=failed` is downgraded to `applying`. That subtest passed. The hole is a reporter that keeps attesting the previous generation as passed and never applies the new one.
- **Proof:** `TestReviewR14_RepeatedStalePassedReportKeepsReady/second_stale_passed_report` in `internal/nodes/review_r14_observation_integration_test.go`.

  ```
  review_r14_observation_integration_test.go:497:
  TODO(review): second safety=passed report of applied generation 1 kept provisioning_state=ready safety=passed while desired is 5 (compile had reached 3)
  ```

  The first in-flight report staying `ready` is asserted and passed. `desired is 5` is the generation on the node row after the later compiles; `compile had reached 3` is the generation captured before those compiles.
- **Fix:** wire-nodes. Bound how far, and how often, an already-ready incarnation may acknowledge an applied generation below desired. A repeated stale passed report must leave `provisioning_state=applying`.

### M-3 (Medium): a second table with a higher-priority accept is invisible to `VerifyServing`

- **Where:** `node/enforce/apply.go:97` (also `175` and `207`) runs `nft -j list table inet fvpn`. `node/enforce/parse_nft.go:104` documents that input. `node/enforce/parse_nft.go:145-154` skips a chain whose table is not `fvpn` or whose family is not `inet`.
- **Scenario:** `VerifyServing` was given the serving fixture plus a second inet table `not_fvpn`: hook `forward`, priority `-200`, policy `accept`, and an accept rule. The parser drops that chain before the chain set is checked, so `VerifyServing` returns nil. The same listing is what the agent would fail to see in production, because the read-back never asks nft for any table except `inet fvpn`. A hook that runs before `inet fvpn` can accept member traffic, and both the bytes the agent verifies and the parser that reads them leave that hook out.

  The other constructs in the question are rejected, or they are in-model renderer output. Those subtests passed and are listed under What held. The renderer's own rules do not create a second table.
- **Proof:** `TestReviewR14_VerifierModel/second_table_higher_priority_accept` in `node/enforce/review_r14_enforce_test.go`.

  ```
  review_r14_enforce_test.go:201:
  TODO(review): VerifyServing accepted inet table not_fvpn with a forward hook at priority -200 and policy accept
  ```
- **Fix:** netlab. Fail closed when the verified bytes contain another table or another family, and widen the read-back so a higher-priority hook is part of the bytes the verifier sees. The agent still must not modify any table other than `inet fvpn`.

### M-4 (Medium): renewal overlap gives one node one ingress budget per leaf

- **Where:** `internal/httpapi/middleware.go:449-468` spends `ingress[class]` on `certRateKey(leaf)` and takes the class concurrency slot before `AuthenticateAgent`. `certRateKey` is `internal/httpapi/middleware.go:809-814` (`sha256(cert.Raw)`). `internal/pki/service.go:159-160` caps every older serial's `usable_until` at `now+overlap` on each new renewal, so every certificate issued inside the overlap stays authenticable. The overlap is `RenewalOverlapSeconds` (`internal/config/nodes.go:22`); production value is still 待決策, test value 5 minutes (`internal/pki/ca.go`).
- **Scenario:** The agent listener has no per-node cap. Two leaves that `AuthenticateAgent` maps to the same node and incarnation are two principals. After leaf A spent a burst of 2 (one 429, two admitted), leaf B was admitted. The handler then returned 422 on the empty body, which means the limiter had already allowed the request. The two keys differ, and both are the hex SHA-256 of that leaf's `Raw` only.

  One hundred distinct leaves, one request each, against production defaults (`IngressRate` 20/s burst 60, `MaxConcurrent` 128, `MaxRatePrincipals` 10000) produced zero 429s. That is the 100-node first-wait case and it fits. The finding is the overlap multiplier on one node, not the 100-node storm. The fingerprint itself is the raw leaf DER, computed from the certificate Go has already parsed, with no chain bytes and no second parse. That test passed.
- **Proof:** `TestReviewR14_DistinctLeavesMultiplyOneNodeBudget` in `internal/httpapi/review_r14_limits_test.go`.

  ```
  review_r14_limits_test.go:105:
  TODO(review): leaf B of node 01a18ea2-355e-7556-b502-13c7feec1c75 was admitted after leaf A spent the ingress burst (status 422, keys [ce3ae2d6ef91c6d13f6aae18c286700d43863a49b601bd781422794961eb5477 ce3ae2d6ef91c6d13f6aae18c286700d43863a49b601bd781422794961eb5477 a5d2e3cd241f5c29cd9c37f693844e793763ab8a57a57c20329903291c06f161])
  ```
- **Fix:** control. Cap the agent-listener ingress budget per node so a second still-usable leaf cannot buy another burst. pki-enroll owns the overlap that keeps the older leaves authenticable (`internal/pki/service.go:160`); the budget that multiplies is the limiter.

### L-1 (Low): eight depleted tail buckets hide a full idle bucket

- **Where:** `internal/httpapi/middleware.go:207-212` (`evictIdleWindow = 8`) and the walk at `276-291`.
- **Scenario:** The table holds 9 keys. One key has a full burst and has been idle for a second, so it would refill. Eight depleted keys (`PerSecond` 0, burst 1, token already spent) sit at the tail in front of it, because a later touch moved the full key to the head. `evictIdle` examines eight entries, finds none eligible, and refuses the newcomer. Wait is 1s, and the table stays at 9/9. Seven depleted keys in front of the same full bucket do get evicted; that subtest passed. The comment on the constant already describes this residue of the r13 tail-only fix.
- **Proof:** `TestReviewR14_EvictIdleWindowOfEight/eight_depleted_hide_a_full_bucket` in `internal/httpapi/review_r14_limits_test.go`.

  ```
  review_r14_limits_test.go:158:
  TODO(review): newcomer refused while a full idle bucket sat past the 8-key eviction window (wait 1s, entries 9)
  ```
- **Fix:** control. When the window is entirely depleted, keep walking until a full idle bucket is found or the table is exhausted, or document the refusal as the accepted bound and size the window to the largest depleted tail the listener must absorb.

### L-2 (Low): one `/48` is refused at half the table while slots remain

- **Where:** `internal/httpapi/middleware.go:232` (the group check runs before eviction) and `groupBudget` at `265-274`. `anonymousGroup` (`792-807`) masks an IPv6 `/64` rate key to `/48`.
- **Scenario:** With `max_rate_principals` 10 the budget is 5. Five distinct `/64`s inside `2001:db8::/48` fill the budget. The next `/64`, `2001:db8:0:9::`, is refused for 1s while `len(entries)` is 5 and `max` is 10. A hosting provider that puts many legitimate clients in one `/48` hits this cap with half the principal table empty. The comment on `groupBudget` already states the half-cap. This is the anonymous table, not the certificate-keyed agent listener.
- **Proof:** `TestReviewR14_Anonymous48HalfBudget` in `internal/httpapi/review_r14_limits_test.go`.

  ```
  review_r14_limits_test.go:205:
  TODO(review): /64 2001:db8:0:9:: inside hosting /48 2001:db8:: was refused (1s) with 5/10 principal slots free
  ```
- **Fix:** control. Decide whether a legitimate hosting `/48` may occupy more than half of `max_rate_principals`, and apply that number in `groupBudget`.

## What held

### Provisional lease (D-067)

`TestReviewR14_GrantlessControlRenewsForeverAtCap` (`node/agent/review_r14_agent_test.go`). A control that keeps answering with grant-less leases, entered by a restart onto that lease, renews at one heartbeat (10s), then 20s, 40s, then the 60s cap, and stays there. The node serves nothing (`serving` false, zero members). The operator signal is the lease check `failed`, code `LEASE_EXPIRED`, detail `Serving lease has no member grants yet.`, block reason `lease_expired`, safety `passed`. That signal is the known agent follow-up 16 gap (`awaiting_member_grants` is not emitted on this branch). The schedule itself is the D-067 design.

`TestReviewR14_StaleDesiredDoesNotMakeNewGenerationProvisional`. Desired bindings stay non-empty at generation G. A grant-less lease at G+7, accepted by entering the renew lead (`RenewLeadSeconds` 3600, advance three quarters of the lease window), leaves `provisionalStreak` and `provisionalRetryAt` at 0. Another 60s does not arm them. Empty desired bindings at the followed generation are not provisional and do not renew after 60s.

`TestReviewR14_ProvisionalTimerFollowsMonotonicClock`. A 20s wall jump, inside the 60s forward skew, does not move `provisionalRetryAt` and does not renew. A 10s monotonic advance does renew (streak 2). A restart clears the in-memory deadline; the next step renews at streak 1. Production `Mono` is `CLOCK_MONOTONIC`, which stops across suspend and is the same clock as the lease deadline, so the two stay aligned. This process cannot suspend the host. A wall jump past `DefaultForwardSkew` fails the time anchor, and `needRenew` returns before it reads `provisionalRetryAt`; that path was not given its own assertion.

`TestReviewR14_OneHundredCertStormFitsTheAgentListener`. One hundred distinct leaves, one `POST /api/v1/agent/lease-renewals` each, against production defaults: `denied=0`. The 10s first wait of 100 nodes does not exceed the agent-listener budget of follow-up 19.

### Bounded conntrack failure

`TestReviewR14_ConntrackBoundIsPerApplyAndSurvivesPartialSuccess`. The counter is per verified apply, not per address. Eleven applies each delete two addresses and fail the second (`100.64.1.2`); the address whose delete succeeded (`100.64.1.3`) stays queued, and the count does not reset. A restart between attempt 11 and 12 reloads attempts 11 and both pending addresses. Apply 12 fail-closes: `serving` false, `conntrackBlocked` set, detail `Conntrack delete exceeded the retry bound.`, check `internet_only` failed with `ISOLATION_POLICY_INVALID`. `e.report()` has `MemberForwardingEnabled` false, `SafetyStatus` failed, `ProvisioningState` ready, and that error at stage `apply`. `ConntrackDeleteError` still has no `Failed` subset (netlab follow-up 9, not filed).

### Revoke step on the real fail-closed report (D-070)

`TestReviewR14_AgentFailClosedDoesNotCompleteRevoke` (`internal/nodes/review_r14_observation_integration_test.go`) passed. The observation is the agent's report shape: forwarding false, safety failed, provisioning ready, `internet_only` failed, code `ISOLATION_POLICY_INVALID`, detail `Conntrack delete exceeded the retry bound.`. The `member_device.revoke` and its `nodes` step do not succeed. Wire-nodes follow-up 20 is the in-flight alignment; this test will fail when that change starts completing the step, and that failure will mean the characterization moved.

`TestReviewR14_ExitGroupFenceHoldsRevokesForNodesThatDoNotHaveTheDevice/different_group_does_not_hold_the_revoke` passed. A dirty peer published to another group, with the member plan limited to the clean node's group, lets the revoke succeed.

`TestReviewR14_RepeatedStalePassedReportKeepsReady/safety_failed_downgrades` passed. The same old generation reported with `safety_status=failed` becomes `applying` / `failed`.

### Null-only seeding (D-068)

`TestReviewR14_NullCurrentIsTheOnlyHeartbeatSeed` (`internal/fleetupdate/review_r14_seeding_integration_test.go`), all four subtests. `observeTx` (`internal/fleetupdate/serving.go:143`) writes `current_release_id` only where it is NULL, and only when the reported current release is the expected one and `last_good` agrees. A forged non-null last-good does not seed. An unexpected release cited as current, without being added to the serving set, does not seed. A heartbeat replay of a revoked previous incarnation is `IDENTITY_REVOKED` and does not seed. Reinstall has to revoke the old incarnation first (`node_incarnations_active_key`).

`TestReviewR14_SupersededJobHeartbeatDoesNotStarveNodeLiveness`. A heartbeat whose job id is not the node's current job does not move `fleet_node_state.last_observed_at` (`serving.go:237-239` returns before that update). It does move `nodes.last_observed_at`, `nodes.last_seen_at`, `control_status=online`, and inserts `node_observations`. The operator node view is the nodes clock. The fleet version view is the fleet clock, and it stays put. Liveness is not starved.

### Forwarded-address parse and the certificate fingerprint

`TestReviewR14_ForwardedAddressParseAgreesWithNormalForm`. Generated inputs (zones, IPv4-mapped, uppercase, leading zeros, `::` at either end with seven fields, eight fields plus `::`, eight fields plus a port, brackets with a zone) either reject or equal the bracketed or RFC 5952 host, including the documented zero-width collapse of exactly one `::` with eight non-empty fields. The r13 H-2 vector `2001:db8:1::2:3:4:5:6` shares the `/64` of `2001:db8:1:2:3:4:5:6`. Eight fields plus a trailing `::` is rejected.

Designed shared buckets, asserted and not filed: IPv4 and its IPv4-mapped form; case and leading zeros; a zone id stripped before the key (`fe80::1%eth0` and `fe80::1%eth1` are one bucket); the eight-field host and the same host with a trailing port.

`ParseTrustedProxies` rejects `::ffff:192.0.2.8/128`. `192.0.2.8/32` matches both `192.0.2.8` and `::ffff:192.0.2.8` through `ipInNets`. Leading-zero IPv6 CIDRs match. With that v4 network trusted, both `192.0.2.8:443` and `[::ffff:192.0.2.8]:443` select the client hop `203.0.113.9`.

`TestReviewR14_CertFingerprintIsLeafRawOnly`. `certRateKey` is `sha256(cert.Raw)`. A 1 MiB `RawTBSCertificate` does not change it. Distinct leaves do not collide.

### Verifier model, apart from the second table

`TestReviewR14_VerifierModel` rejected: a `return` verdict; `policy accept` on `forward`; an extra base chain `prerouting` inside `inet fvpn`; `ct state established accept` on `forward`; an interval element and an `interval` flag on `@allowed_v4`; `meta nfproto ipv6` on an accept; a second accept that only names the upload counter.

Accepted because they are the renderer's own shape: `ct state invalid drop` and `meta nfproto ipv6 drop` on the member-to-uplink path; `flags timeout` on `@allowed_v4`. An extra `ip version` match on a device accept does not widen it. Rebinding a named counter's packet count does not add an accept.

`tests/netlab/scenario_ingress_test.go` (`TestIngress_InjectedMemberFrames`) is the devstack follow-up 13 fixture. `fvpn-netlab inject` places one frame on the member interface and the test asserts the inject did not rewrite `table inet fvpn`, the WireGuard peer table, addresses, routes, or the link. It is a lab tool. The netlab was not run in this review.

### `_sha256` exemption and ENR-02

`TestReviewR14_SHA256SuffixIsNotAClientBypass` (`internal/audit/review_r14_sha256_test.go`). `sensitive` (`internal/audit/audit.go:26-37`) exempts a name longer than `_sha256` that ends in `_sha256` before the `token` / `secret` / `password` substring check. `Diff` still copies only the caller's allowlist and skips sensitive keys, so `token_hash` and `access_token` stay redacted and a non-allowlisted `evil_sha256` is dropped. A PEM value is redacted by the value rule even under a `_sha256` name. The exact key `_sha256` is not sensitive (the exemption needs a longer name, and the underscore strip leaves `sha256`, which matches no secret term). That key is not a client bypass: every `Before`/`After` map in the writers is a server-built allowlist. No members, device-authorization, operations-metadata, or fleet-config path copies a client-chosen property through `Diff`.

`Enrollment.token_sha256` is the stored SHA-256 of the one-time token, returned on every enrollment read and written on `enrollment.created`. Contract revision 10 and `docs/contracts/README.md` (revision 10) say that digest is the identifier §6.1 stores, and that names ending in `_sha256` are the exemption. Correlation across audit, portal, and logs is that identifier. It is not filed.

### Updater recovery

`TestReviewR14_BlankRollbackProfileCannotRetargetAnotherJob` (`internal/fleetupdate/review_r14_seeding_integration_test.go`). A result with `state=failed`, `stage=rollback`, recovery required, and all five health checks failed is bound to the job's generation, attempt, batch, target, release id, manifest hash, and node incarnation. The same `result_id` with the same payload hash replays. A different hash conflicts. A different node or incarnation mismatches. The blank failed profile does not retarget an unrelated job, and it does not satisfy `rolled_back` (that transition still requires a healthy profile).

`TestReviewR14_ProbeOmissionEventSurvivesMetaDelete` (`node/updater/review_r14_updater_test.go`). `cleanTxnNotes` (`node/updater/apply.go:513`) deletes the probe-omission meta rows with the transaction. The append-only `txn_events` copy remains. The D-063 issuer re-evaluates addresses when it issues the next challenge; it does not need the finished transaction's meta row.

## Known, reconfirmed, not filed

- Agent follow-up 16: a grant-less lease in force still reports `lease_expired` / `Serving lease has no member grants yet.` The renewal schedule held.
- Wire-nodes follow-up 20: the agent's real fail-closed observation does not complete the revoke step today. The 60 s `RenewTx` backstop for grant-less leases of not-ready nodes is not on this branch.
- Netlab follow-up 9: `ConntrackDeleteError` has `Pending` and no `Failed` subset. The per-apply bound and the restart both held.
- Devstack follow-up 13, portal follow-up 12, the harness re-runs, handover follow-up 8, and multi-replica limiter budgets were not re-tested. The lab inject hook was read as source only.
