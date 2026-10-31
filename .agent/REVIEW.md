# Adversarial Review r15: cloned identity, the revoke fence, gated publish, and Fair-Use at the exits

Branch: `r15-d072-d075-review` (from `main` at `de18964`)
Date: 2026-10-30
Reviewing model: `grok/grok-4.7`

Scope is the r14 fixes now on `main` and the decisions taken with them. D-072 cloned-identity detection is wire-nodes follow-up 21 (#211) and follow-up 22 (#217). D-073, the durable roaming candidate, and D-075, the grace column, are follow-up 24 (#228). D-074, a gated publish that waits, is follow-up 23 (#224). r14 H-1 and M-2 landed in #217, M-1 in agent follow-up 18 (#216), M-3 in netlab follow-up 10 (#220), M-4 / L-1 / L-2 in control follow-up 20 (#215). Fair-Use recompile and the ACK reader are wire-policies-usage follow-up 4 (#222).

Proofs are `TestReviewR15_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. Known in-flight work is reconfirmed there and is not filed again. Product code was not changed.

Commands (2026-10-30, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`). Integration tests used their own databases through `dbtest`. The netlab used prefix `fvpn-r15-netlab` (`--network none`) and was removed afterwards (`netlab-down`, scratch wiped, image removed).

```
go test -count=1 -timeout 180s -run 'TestReviewR15_' ./internal/httpapi/ ./internal/pki/ ./internal/fup/ ./node/agent/ ./node/enforce/
go test -tags integration -count=1 -timeout 10m -p 2 -run 'TestReviewR15_' ./internal/nodes/ ./internal/nodes/httpwire/ ./apps/worker/ ./node/agent/
FVPN_LAB_PREFIX=fvpn-r15-netlab make -f mk/netlab.mk netlab-up
FVPN_NETLAB=1 FVPN_LAB_PREFIX=fvpn-r15-netlab go test -tags netlab -count=1 -timeout 15m -p 1 -parallel 1 -v -run 'TestReviewR15_TcMirredAndPolicyRoutingBypass' ./tests/netlab
FVPN_LAB_PREFIX=fvpn-r15-netlab make -f mk/netlab.mk netlab-down
```

Unit packages exited 0 except `TestReviewR15_AnonymousReserveLeavesOneSlot`. Integration packages exited 1 only on the `TODO(review)` lines quoted below. The netlab exited 1 on the tc redirect and passed the policy-routing subtest:

```
review_r15_q8_bypass_test.go:53: tc mirred: ctunku=0 uplink="02:07:10.435212 IP 100.64.9.9.40000 > 203.0.113.10.39997: UDP, length 1\n\n"
review_r15_q8_bypass_test.go:55: TODO(review): tc clsact mirred on fv-fh redirected a non-admitted packet onto fv-ih ...
--- FAIL: TestReviewR15_TcMirredAndPolicyRoutingBypass/tc_clsact_mirred (1.02s)
    review_r15_q8_bypass_test.go:79: ip rule: ctunku=1 uplink="\n"
--- PASS: TestReviewR15_TcMirredAndPolicyRoutingBypass/ip_rule (1.07s)
```

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| H-1 | High | wire-nodes | `internal/nodes/httpwire/state.go:177-190`, `internal/enrollment/revoke.go:379-395`, `internal/leases/leases.go:431-444` | The renewal that freezes an incarnation still mints a 24 h member-grant lease. The next observation marks the node failed and does not shorten that lease. |
| M-1 | Medium | wire-nodes | `internal/nodes/commandtx/transaction.go:34-38`, `internal/enrollment/revoke.go:239`, `db/migrations/00005_evidence_and_deadlines.sql:15` | A lease renewal and an observation of the same node take `nodes` and `fleet_config` in opposite orders and deadlock (`40P01`). |
| M-2 | Medium | wire-nodes | `internal/nodes/observations.go:812-821` | After the member's only grant is revoked, `otherNodePendingIso` still matches that grant's plan revision, so a quiet-enough peer in the old group holds `member_device.revoke`. |
| M-3 | Medium | wire-nodes | `internal/exits/exits.go:723-739` | `safety_status=failed` with the generation already applied is gated as `readiness_unconfirmed`. The publish stays running until `worker_attempts`. |
| M-4 | Medium | wire-nodes | `internal/nodes/observations.go:205-227` and `276-278`, `internal/fup/engine.go:1561-1567` | A frozen exit's latest observation row stays `safety_status=passed`, so `fair_use.applying` clears while the node row is `failed`. |
| M-5 | Medium | wire-policies-usage | `internal/fup/engine.go:1513-1518` and `1613-1624`, `848-853` | Unpublishing the only carrying exit makes the ACK reader fall through to `checks.applied_policy_refs`. Production does not store that there, so `applying` stays true. |
| M-6 | Medium | netlab | `node/enforce/apply.go:99-110`, `node/agent/engine.go:832-847` | A `tc clsact` mirred redirect on the member interface forwards a non-admitted packet while `fvpn-netlab verify` still returns 0. |
| L-1 | Low | control | `internal/httpapi/middleware.go:249-252`, `284-290` | `anonymousGroupReserve = 1` lets one IPv6 `/48` hold every anonymous slot but one. |

## Findings

### H-1 (High): the renewal that detects a clone still issues a 24 h member-grant lease

- **Where:** `Handlers.RenewServingLease` calls `observeAgentEndpointTx` and then `RenewTx` in the same transaction (`internal/nodes/httpwire/state.go:177-190`). `recordIdentityConflict` sets `mapping_frozen` and does not change `provisioning_state` (`internal/enrollment/revoke.go:388-395`). `includeMembers` is `published && registered && provisioning_state == ready && len(groupIDs) > 0` (`internal/leases/leases.go:431-434`). The 60 s grant-less cap runs only when the node is published and registered and not ready (`internal/leases/leases.go:436-444`). `DefaultMaxOfflineLeaseSeconds` is 86400 (`internal/leases/leases.go:28`). `LockAuthenticated` does not read `mapping_frozen` (`internal/pki/service.go:87-102`). The agent renews a grant-bearing lease only inside `RenewLeadSeconds`, default 120 (`node/agent/config.go:122-123`, `node/agent/engine.go:994-996`).
- **Scenario:** X→Y→X inside the 120 s window freezes the incarnation on the detecting `POST /api/v1/agent/lease-renewals`. That request still returns 200. Freeze does not flip `provisioning_state`, so `RenewTx` still includes the member grants and signs a lease for the fleet offline cap, 24 h. The following observation does what the doc says: `provisioning_state=failed`, `safety_status=failed`, `member_forwarding_observed=false` (`internal/nodes/observations.go:276-278`). It does not update `serving_leases.expires_at`. A later renewal, now that the node is not ready, is the 60 s grant-less backstop with zero members. Until the agent is inside the 120 s renew lead of the 24 h deadline, it keeps the grant-bearing lease and can keep forwarding. That is about 23 h 58 min after the detecting renewal. The frozen leaf still authenticates. Recovery remains revoke and re-enroll, which withdraws members, and nothing in this path shortens the lease that was just minted.
- **Proof:** `TestReviewR15_FrozenIncarnationStillReceivesAGrantLease` in `internal/nodes/httpwire/review_r15_identity_integration_test.go`.

  ```
  TODO(review): the renewal that froze incarnation 0cf5ceaa-350b-482f-b766-2b293383d525 still minted 1 member grants for 24h0m0s (DefaultMaxOfflineLeaseSeconds=86400). ObserveEndpointTx runs before RenewTx and recordIdentityConflict does not change provisioning_state, so includeMembers stayed true. The next observation set provisioning=failed safety=failed forwarding=false and left serving_leases.expires_at at 2026-11-01T01:59:02Z. A later renewal ttl is 1m0s with 0 members. The agent renews a grant-bearing lease only inside RenewLeadSeconds (default 120) of its deadline, so this lease keeps member forwarding until about 23h58m0s after the detecting renewal. Recovery is revoke and re-enroll.
  ```
- **Fix:** wire-nodes. Once `mapping_frozen` is set, `RenewTx` must not include member grants, and an already stored grant-bearing lease for that incarnation must not stay valid until its original `expires_at`. The grant-less 60 s cap is the existing backstop for a node that is not ready; the freeze has to make the node not ready before `includeMembers` is decided, or `includeMembers` has to read `mapping_frozen`.

### M-1 (Medium): lease renewal and observation of one node deadlock on `nodes` and `fleet_config`

- **Where:** `commandtx.InTx` takes `SELECT id FROM fleet_config FOR UPDATE` before the callback (`internal/nodes/commandtx/transaction.go:34-38`). `SubmitObservation` enters through that helper, so the observation locks the fleet row before `LockAuthenticated` locks the node (`internal/nodes/httpwire/handlers.go:329-334`, `internal/enrollment/revoke.go:239`, `internal/pki/service.go:93`). `RenewServingLease`, on the HTTP command transaction, does not take the fleet lock first (`internal/nodes/httpwire/state.go:177-181`). It calls `ObserveEndpointTx`, which locks `nodes` `FOR UPDATE`, and the `serving_leases` insert then runs `check_node_lease`, which takes `fleet_config` `FOR SHARE` (`db/migrations/00005_evidence_and_deadlines.sql:9-15`). `db.InTx` retries `40001` and `40P01` four times (`internal/db/tx.go:23-49`).
- **Scenario:** Two control processes, one database, one incarnation. The observation session holds `fleet_config` `FOR UPDATE` and waits for `nodes` `FOR UPDATE`. The lease session already holds `nodes` `FOR UPDATE` and waits for `fleet_config` `FOR SHARE`. PostgreSQL reports `40P01` on the lease session. The observation session in this interleaving saw no error of its own; the lease session is the one the deadlock detector cancels. Four retries then return the error to the caller. One agent process is sequential, so a single node does not do this to itself. Two replicas, or a lease and an observation overlapping from two clients of the same leaf, do. Different nodes do not share the node row; the cycle is per node plus the single `fleet_config` row.
- **Proof:** `TestReviewR15_LeaseAndObservationLockOrder` in `internal/nodes/httpwire/review_r15_q2_lock_integration_test.go`. The proof is two raw connections issuing those statements, with `deadlock_timeout = 200ms`, after the lease session is observed in `pg_stat_activity.wait_event_type = Lock`.

  ```
  TODO(review): concurrent lease renewal and observation on node 01a18fab-fe72-7c41-ac24-28a0e2822cf2 deadlock (40P01). Observation holds fleet_config FOR UPDATE and waits for nodes FOR UPDATE; the lease holds nodes FOR UPDATE and waits for fleet_config FOR SHARE inside check_node_lease. db.InTx retries 40001 and 40P01 four times and then returns the error. lease=ERROR: deadlock detected (SQLSTATE 40P01) observation=<nil>
  ```
- **Fix:** wire-nodes. Take `fleet_config` before `nodes` on the lease path as well, the order `commandtx.InTx` already uses for observations. `check_node_lease` can keep its `FOR SHARE`; it must not be the first fleet lock in a transaction that already holds `nodes`.

### M-2 (Medium): a revoked plan revision still holds the 90 s fence when the projection is empty

- **Where:** The grant fallback inside `otherNodePendingIso` (`internal/nodes/observations.go:812-821`) joins `entitlement_grants` to `plan_revisions` when `jsonb_array_length(projection->'exit_group_ids') = 0`. The join has no `eg_m.revoked_at IS NULL`. The projector does filter `revoked_at IS NULL` (`internal/entitlement/store.go:75`).
- **Scenario:** The member has one grant, naming the exit group the dirty peer is published in. `members.Service.RevokeGrant` sets `revoked_at` and recomputes. The active projection's `exit_group_ids` is then exactly `[]` (an empty plan revision cannot be granted: `validPlan` requires at least one exit group). The peer is still forwarding with `ISOLATION_POLICY_INVALID` and is inside the 90 s freshness window. `member_device.revoke` stays `running`, step `pending`, empty code and message. The stale revision is what matches the peer's group. A live projection that still named the group would be the designed hold; this hold survives the revoke.
- **Proof:** `TestReviewR15_RevokedPlanRevisionStillHoldsTheFence` in `internal/nodes/review_r15_q3_fence_integration_test.go`.

  ```
  TODO(review): active projection exit_group_ids is [] after the grant that named group 01a18fae-863f-7072-840f-d36756a9bf9c was revoked, and the peer 01a18fae-8640-72f9-b181-12d201b51c9d is published only in that group. member_device.revoke 01a18fae-8674-70ab-83f8-564c80fb7d6e stayed running (status=pending code= message=). otherNodePendingIso matches entitlement_grants joined to plan_revisions with no revoked_at IS NULL, so the stale revision still holds the fence. The projector itself filters revoked_at.
  ```
- **Fix:** wire-nodes. Add `revoked_at IS NULL` on that grant, the same predicate the projector uses. An empty projection after the only grant is revoked must not keep a peer in the old group holding the revoke.

### M-3 (Medium): a node with `safety_status=failed` waits out the publish budget as `readiness_unconfirmed`

- **Where:** `executePublish` maps every `CheckPublishable` error to `failGatedPublish` on `agent_policy`. The cause is `readiness_unconfirmed` unless `applied < desired`, in which case it is `stale_generation` (`internal/exits/exits.go:723-739`). `CheckPublishable` rejects `safety_status` failed (`internal/nodes/transitions.go:147-148`). `failGatedPublish` keeps the step `pending` with `NODE_NOT_SERVICEABLE` (`internal/exits/publication_gate.go:42-56`). The worker returns "publication execution incomplete" and calls `ExhaustGatedPublish` only when `event.Attempt >= attempts` (`apps/worker/jobs_nodes.go:108-114`). `config.Defaults().WorkerAttempts` is 10 (`internal/config/config.go:84`). There is no cancel operation under `internal/operations`.
- **Scenario:** The node has applied generation 2, desired generation 2, provisioning `applying`, safety `failed`. The publish command is accepted (the PUT gate does not set `RequireOnlineAndACKForRun`). Execution then treats the safety failure as a wait: operation `running`, error null, `publication_observed=absent`, step reason `readiness_unconfirmed`. With the worker budget set to 2 it stays that way until attempt 2, then the operation is `failed` and `publication_observed=error`. It does not wait forever. At the production default the wait is the documented jittered backoff, about 4.3 to 8.5 minutes (`docs/nodes/README.md` §12.3). While the operation is `running` the portal headline, which is the operation status, does not say failed. There is no cancel. A node that is already broken is indistinguishable from a node that has not caught up yet until the budget ends.
- **Proof:** `TestReviewR15_SafetyFailedPublishWaitsAsReadinessUnconfirmed` in `internal/nodes/httpwire/review_r15_q4_publish_integration_test.go`. The dispatcher was given `attempts=2` so the proof does not sleep through the production backoff. The cause and the final `error` transition are what the test asserts.

  ```
  TODO(review): safety_status=failed at applied 2 >= desired 2 (provisioning applying) is gated as agent_policy reason readiness_unconfirmed. The operation stayed running with a null error and publication_observed=absent until worker attempt 2 of 2, then failed with publication_observed=error. It does not wait forever. config.Defaults WorkerAttempts is 10 (backoff 1s doubling to 300s). There is no cancel operation. The portal headline is the operation status, so it does not say failed while the budget remains.
  ```
- **Fix:** wire-nodes. When `safety_status` is `failed` or `expired` and the applied generation is already at desired, fail the publish on the first attempt the way a non-gate `DEPENDENCY_UNAVAILABLE` fails (`docs/nodes/README.md` §12.3), and set `publication_observed=error`. Keep the wait for `stale_generation` and for a data path that is merely unconfirmed.

### M-4 (Medium): Fair-Use ACK clears while the carrying exit is frozen

- **Where:** `SubmitObservationTx` inserts `node_observations` with the safety from `evaluateObservationEvidence` (`internal/nodes/observations.go:205-227`) before the `mapping_frozen` override (`internal/nodes/observations.go:276-278`). The override is written onto the `nodes` row only. `carryingExitAckStatus` reads the latest `node_observations.safety_status` and `control_status` (`internal/fup/engine.go:1561-1567`). It does not read `nodes.safety_status` and it does not read `mapping_frozen`.
- **Scenario:** The carrying exit has acked, so `fair_use.applying` is false. The incarnation is then frozen and submits another observation whose report still says safety passed. The node row becomes `safety_status=failed`. The latest observation row stays `passed` and `online`. The ACK reader counts that exit as acked. `applying` stays false (`pending=0`, `acked=1`) while the exit is frozen and not a safe place to enforce the tier. The unrelated exit's desired generation stays 0; the world setup asserts carry `1`, other `0` before this observation (`assertDesiredGens` in `reviewFairUseWorld`).
- **Proof:** `TestReviewR15_FrozenCarryingExitKeepsFairUseApplying` in `apps/worker/review_r15_q9_fup_integration_test.go`.

  ```
  TODO(review): frozen exit 97df04f1-b178-43a3-b547-c7115c3fe831 has nodes.safety_status=failed but the latest node_observations.safety_status=passed, so carryingExitAckStatus still acks it (applying=false pending=0 acked=1). SubmitObservation inserts the observation before the mapping_frozen override, and the ACK reader never reads nodes.safety_status. fair_use.applying clears while the carrying exit is frozen.
  ```
- **Fix:** wire-nodes. Write the post-override safety onto the observation row, or stop inserting a `passed` observation for a frozen incarnation. wire-policies-usage can also require `nodes.safety_status=passed` and `mapping_frozen=false` in `carryingExitAckStatus`; the reader today is doing what its comment says against a row that does not say the node is frozen.

### M-5 (Medium): unpublishing the only carrying exit leaves `fair_use.applying` set

- **Where:** `GetMemberNodeAckStatus` uses `carryingExitAckStatus` and, when that returns no published carrying exit, calls `legacyRegisteredNodeAckStatus` (`internal/fup/engine.go:1505-1518`). The legacy query reads `node_observations.checks->'applied_policy_refs'->>'fup_generation'` (`internal/fup/engine.go:1615-1624`). The comment on the fall-through says production stores `applied_policy_refs` on `report_payload`. `applying` is true when the generation is positive and (`TotalServingNodes == 0` or `PendingNodes > 0`) (`internal/fup/engine.go:848-853`).
- **Scenario:** One published exit carries the member and has acked, so `applying` is false. That exit's `publication_intent` becomes `unpublished`. No published exit carries the member. The reader falls through to every registered node and looks for `checks.applied_policy_refs`, which the stored observations do not have. Nothing acks. The status this proof read was `total=2 pending=2 acked=0 summary="pending on 2 nodes"`, and `applying` stayed true. It does not clear on a later observation of the unpublished exit, because that exit is no longer in the carrying set and the legacy predicate still misses. There is no generation event that would turn it off. `applying` sticks.
- **Proof:** `TestReviewR15_UnpublishDropsTheCarryingExitAndApplyingSticks` in `apps/worker/review_r15_q9_fup_integration_test.go`.

  ```
  TODO(review): after the only carrying exit ef536a5f-b3cf-44fb-b2e0-083b147fe21f is unpublished, fair_use.applying stays true (total=2 pending=2 acked=0 summary="pending on 2 nodes"). GetMemberNodeAckStatus falls through to legacyRegisteredNodeAckStatus, which reads checks.applied_policy_refs. Production stores applied_policy_refs on report_payload, so a registered node never acks and applying does not clear.
  ```
- **Fix:** wire-policies-usage. When no published exit carries the member, do not count registered nodes through a column the observation writer does not fill. Either `applying` is false when `TotalServingNodes` from the carrying query is 0, or the legacy reader has to use the field production actually stores. Document which one operators should expect after the last carrying exit is unpublished.

### M-6 (Medium): a `tc clsact` mirred redirect forwards non-admitted traffic while verify returns 0

- **Where:** `Apply` installs nft, then reads `nft -j list ruleset` once and calls `VerifyServing` (`node/enforce/apply.go:99-110`). Nothing in that read is a tc qdisc or an `ip rule`. After `Apply` returns, `Engine.enforce` sets `e.serving = true` (`node/agent/engine.go:832-847`). `Step` defers `beat()`, which writes `Serving` from that flag with no second ruleset read (`node/agent/engine.go:436-437`, `973-988`). `docs/poc/G2-netlab.md` line 87 says the agent installs no tc filter. The 待實測 list (line 231) names a real NetBird peer's fwmark routing, not a local `clsact` mirred redirect.
- **Scenario:** The lab prefix `fvpn-r15-netlab` is serving generation 91. `fvpn-netlab verify` returns 0. A `clsact` qdisc on `fv-fh` (the member interface) with `matchall` and `mirred egress redirect dev fv-ih` is then installed. Verify still returns 0. One non-admitted UDP packet injected on `fv-fh` from `100.64.9.9` to `203.0.113.10` is captured on `fv-ih`. The `ctunku` counter does not move. The redirect happens at ingress, before the inet filter forward chain. The heartbeat can say serving for the rest of that step and until the next apply. Installing the qdisc takes `CAP_NET_ADMIN` on the exit, the same capability the agent has. A second process, or a qdisc left in place, is enough.
- **Proof:** `TestReviewR15_TcMirredAndPolicyRoutingBypass/tc_clsact_mirred` in `tests/netlab/review_r15_q8_bypass_test.go`.

  ```
  review_r15_q8_bypass_test.go:53: tc mirred: ctunku=0 uplink="02:07:10.435212 IP 100.64.9.9.40000 > 203.0.113.10.39997: UDP, length 1\n\n"
  review_r15_q8_bypass_test.go:55: TODO(review): tc clsact mirred on fv-fh redirected a non-admitted packet onto fv-ih (capture "02:07:10.435212 IP 100.64.9.9.40000 > 203.0.113.10.39997: UDP, length 1\n\n") without incrementing ctunku, and fvpn-netlab verify still returned 0. The redirect is not in the nft ruleset VerifyServing reads.
  ```
- **Fix:** netlab. Fail closed when the member interface or the uplink has a tc filter or a mirred action the agent did not install, and say so in `docs/poc/G2-netlab.md`. A read of `nft -j list ruleset` does not see this path. The policy-routing case measured in the same test did not bypass; see What held.

### L-1 (Low): one hosting `/48` may hold every anonymous slot but one

- **Where:** `allowGrouped` refuses a new key in a group that already holds keys when `max - len(entries) <= anonymousGroupReserve` (`internal/httpapi/middleware.go:249-252`). `anonymousGroupReserve` is 1 (`internal/httpapi/middleware.go:284-290`). The anonymous class is `EnrollNode` and `ReceiveProviderWebhook` (`internal/httpapi/middleware.go:728-731`). Login is a separate limiter, per IPv6 `/64`. The agent bootstrap is mTLS and is keyed by the node URN (`agentIngressKey`, `internal/httpapi/middleware.go:851-856`), not by the `/48`.
- **Scenario:** With `max_rate_principals` 10 the test admitted 9 new `/64` keys from one `/48` and refused the tenth. Production `max_rate_principals` is 10000, so one hosting `/48` holds 9999 anonymous principal slots and every other `/48` shares the one leftover. The previous cap, adopted in r14 as L-2 and then replaced, was half the table (5 of 10 here, 5000 in production). Those 9999 slots are enrollment and the provider webhook. They are not login and they are not the agent listener. One tenant network can still crowd out enrollment for everyone else.
- **Proof:** `TestReviewR15_AnonymousReserveLeavesOneSlot` in `internal/httpapi/review_r15_q6_ingress_test.go`.

  ```
  TODO(review): anonymousGroupReserve=1 lets one hosting /48 hold 9 of 10 anonymous principal slots (production max_rate_principals 10000 holds 9999). The previous cap was half the table (5 here, 5000 in production). Those slots are the anonymous class only: EnrollNode and ReceiveProviderWebhook. Login is keyed per IPv6 /64 on its own limiter, and the agent bootstrap is mTLS keyed by the node URN. One /48 can therefore take every enrollment and provider-webhook slot but one, and every other /48 shares the single leftover. Recommend the previous half cap (max/2) or a small absolute cap such as 64.
  ```
- **Fix:** control. Restore a reserve of `max/2`, or cap one `/48` at a small absolute number such as 64. Keep the reserve off the login limiter and off the node-URN agent key.

## What held

### 1. Clone detector: forwarded address, who can write, multi-homed cost, quiet takeover

`TestReviewR15_DeployProxyTakesTheRightmostUntrustedHop` (`internal/httpapi/review_r15_q1_xff_test.go`) uses the real middleware and a trusted-proxy CIDR shaped like `deploy/` (`172.28.30.21/32`). Several `X-Forwarded-For` hops resolve to the rightmost hop that is not itself a trusted proxy (`requestRateKey`, `internal/httpapi/middleware.go:890-919`). The leftmost client-supplied entry is not the domain. A TCP peer outside the trusted set ignores the header. A header the proxy does not append is the client's value; `deploy/devstack/nginx.conf` appends `X-Forwarded-For` on the portal `:8443` `/api/` block, and the agent mTLS listener is not that block. A naked header on a connection the gateway does not terminate is the documented trust boundary, not a second finding. Two clones inside one `/24` or `/48` stay indistinguishable (`docs/nodes/README.md` §11.2). That limit is already documented and is not filed.

`TestReviewR15_EndpointWriteRequiresTheNodesLeaf` and `TestReviewR15_RightmostHopIsTheStoredDomain`. No verified leaf is 401 before the handler, and `node_identity_endpoints` stays empty. Another node's leaf, with a body naming the victim, returns 403 and commits no endpoint row (`other-leaf request naming the victim returned 403 and committed no endpoint row`). That other node's own renewal writes only its own row. After HTTP revoke, the revoked leaf is non-200 and the victim's row count does not change. `ObserveEndpointTx` runs only after `AuthenticateAgent` (`internal/httpapi/middleware.go:463-490`).

`TestReviewR15_MultihomedNodeFreezesItself`. Two uplinks, or a mobile and a fixed address, that alternate inside 120 s are the interleaving rule: one `node_identity_conflicts` row, `mapping_frozen`, no unfreeze. The operator cost is `revokeNode` and re-enroll, which withdraws members and starts a new incarnation (`docs/nodes/README.md` §11.4). Section 11 calls that interleaving a cloned key. It does not name a legitimate dual-uplink node. Handover follow-up 10 owns that wording, so the gap is not filed again.

`TestReviewR15_QuietWindowPromotesWithoutAnAudit`. Timeline: domain X is stored; 121 s later Y is promoted to `endpoint_hash` with zero `node_identity_conflicts` rows, zero `node.identity.conflict` audits, and `nodes.resource_version` unchanged. `identity_conflict` therefore stays unfrozen with `conflict_count` 0. Nothing in the operator note (§11.5), which is written for conflicts, tells the operator that the mapped domain moved. That is the documented roaming promotion (`internal/enrollment/revoke.go:307-333`), not a second conflict path. Two further window crossings leave a single endpoint row. The primary key is `incarnation_id`, so a node that roams every day adds no rows. Revoke leaves that row on the retired incarnation and the replacement incarnation starts with none. The candidate columns are what `ObserveEndpointTx` reads on every call (`internal/enrollment/revoke.go:277-283`); this proof does not exec a second OS process. The owner's restart and two-session coverage remains `TestD073_DurableRoamingCandidateMultiReplicaRestartConcurrencyAndMigration`.

### 2. Durable candidate and the grace column

The candidate row, the one-row bound, and revoke retention are the passing test above. The same-node lock cycle is M-1.

`TestReviewR15_StaleGraceCannotStayAtAnOldGeneration`. The first stale `safety=passed` report of an already-applied generation stays `ready` and sets `node_incarnations.stale_generation_grace_at`. The next report at the desired generation stays `ready` and clears the column. Deleting that node's `audit_events` under `SET LOCAL fvpn.retention_purge='on'` does not change the next decision: the same old generation is then `applying`, the column stays NULL, and `applied_config_generation` has already moved forward with `greatest`, so the node is not `ready` at the old generation. Alternating one stale report and one at-desired report therefore cannot sit at the old generation. Re-arming the grace after desired moves ahead again is the one grace per lag from r14 M-2, now on the column (`internal/nodes/observations.go:248-272`, `315-327`). The writer sets the column NULL on every newer observation that is not the granted-grace `ready` case, which is the applying downgrade this test observed. The adopted r14 grace test is not re-filed.

### 3. The 90 s fence, and what a quiet peer still costs

`TestReviewR15_QuietPeerReleasesTheRevokeAfterNinetyOneSeconds`. Two nodes share the member's group. The dirty one is forwarding with `ISOLATION_POLICY_INVALID` at stage `apply`. After `DefaultStaleTimeout` (90 s) plus one second the clean node's observation completes `member_device.revoke`. The log line was:

```
exposure: stale timeout 1m30s released the revoke; peer row still forwarding=true recent_errors=[{"code": "ISOLATION_POLICY_INVALID", "stage": "apply", ...}]; next compile bindings=0; offline lease cap 86400s; conntrack delete bound is 12 attempts (node/agent maxConntrackDeleteAttempts)
```

The peer row can still say forwarding until it reports again. The next desired state has no binding for the device. The offline lease cap is the 24 h fleet maximum and is not shortened by this observation. `conntrack -D` is bounded by `maxConntrackDeleteAttempts` (12) in `node/agent/revocation.go:32`, already covered by the agent tests. That residual is the design those bounds describe. It is not a new finding. The revoked-grant hole, which is not this freshness path, is M-2.

### 4. Gated publish around the safety-failed wait

M-3 is the case that masks `safety_status=failed`. The surrounding contract held as written and was not given a second failing proof.

A node that never becomes serviceable stays `running` with the gated step `pending`, `error_code=NODE_NOT_SERVICEABLE`, and `publication_observed` not `error`, until `worker_attempts`. Production default is 10 attempts, backoff 1 s doubling to 300 s, about 255.5 s to 511 s unjittered (`docs/nodes/README.md` §12.3). The proof used 2 attempts and observed the transition to `failed` / `publication_observed=error`. There is no cancel API.

`ISOLATION_POLICY_INVALID` with safety still passed is the data-path reason `isolation_unconfirmed`, which is one of the documented wait reasons. It is not the safety-failed case.

The r11 empty-attachment withdrawal during a wait is `failGatedPublish` (`internal/exits/publication_gate.go:53-109`): a confirmed withdrawal sets `publication_observed=withdrawn` and keeps the step pending. This review did not re-execute that fixture.

A node that becomes ready after `ExhaustGatedPublish` stays unpublished until an operator re-issues `PUT /api/v1/admin/nodes/{node_id}/publication` (`docs/nodes/README.md` §12.4). That sentence is the documentation. This review did not sleep out a production budget and then repair the node.

The last check is inside the publishing transaction. After NetBird I/O, `executePublish` evaluates the gate again (`internal/exits/exits.go:838-846`) and, under `nodes … FOR UPDATE`, once more before it marks the operation succeeded (`internal/exits/exits.go:856-864`). An observation that commits before that lock is visible. An observation that overlaps it waits on the same node row. A report that commits after the publish commits is a later observation of an operation that already succeeded on the evidence it locked. That is ordinary convergence, not a second publish of a stale gate.

### 5. `AWAITING_MEMBER_GRANTS` is not a failure

`TestReviewR15_AwaitingMemberGrantsReportIsNotAFailure` (`node/agent/review_r15_q5_grants_integration_test.go`) builds the report through the agent's `reported()` helper, not a hand-written JSON body. The submitted report has exactly one `errors[]` entry, `AWAITING_MEMBER_GRANTS` at stage `lease`, safety passed, provisioning ready, forwarding false. `SubmitObservation` keeps safety passed (`evaluateObservationEvidence`, `internal/nodes/observations.go:434-441`). On a running `node.publish` the gate reason was `check_failed` and the operation stayed `running`. The log line was `publication gate reason=check_failed operation=running (wait, not a failure). agent errors=1 safety=passed`. `publicationForwardingFailure` returns `check_failed` on any failed check before it returns `forwarding_disabled` (`internal/nodes/publication_evidence.go:59-67`). The operation is still the D-074 wait, so the reason string is not filed. The FUP observation predicate stayed true (safety passed and control online). The running `member_device.revoke` completed.

The normative contract text requires that one `errors[]` entry. `ObservationReport` in `contracts/openapi.yaml` (the D-071 description, including the line that `errors[]` carries exactly one `AWAITING_MEMBER_GRANTS` entry at stage `lease`) and `docs/contracts/README.md` (the same sentence) are the text. `contracts/examples/valid/ObservationReport.awaiting-member-grants.json` still has `"errors": []`. Contract revision 12 owns the example. It is not filed.

Portal follow-up 14 owns the `awaiting_member_grants` copy and the `identity_conflict` badge. They were not re-tested.

### 6. Node-keyed ingress, aside from the reserve

`TestReviewR15_IssueSetsTheNodeURNAndRejectsACSRURI`. `CA.Issue` sets `urn:fvpn:node:<node id>` and `urn:fvpn:incarnation:<incarnation id>` on the template and does not copy CSR extensions (`internal/pki/ca.go:184`). A CSR URI is rejected. Issued leaves in production have the node URN. `agentIngressKey` falls back to `sha256(leaf.Raw)` only when that URN is absent (`internal/httpapi/middleware.go:851-856`), which this issuer does not produce. Renewal overlap of two leaves of one node therefore shares one budget. Leaves with no URN are not an issuance output of this CA.

`TestReviewR15_UnauthenticatedAgentDoesNotFillTheIngressTable`. Thirty posts with no verified certificate are 401 before `allow()` (`internal/httpapi/middleware.go:463-466`). A following request with a fresh leaf is not 429 and not 401. An unauthenticated client cannot force `evictIdle` and cannot spend a node's burst.

`TestReviewR15_EvictIdleFullTableIsAShortWalk`. A full table of 10000 depleted keys refuses a newcomer in 112.46 µs (wait 1 s). The walk is the whole table (`internal/httpapi/middleware.go:296-308`) and is not an amplification finding at that cost. The reserve size is L-1.

### 7. Re-admitted address

`TestReviewR15_ReassignedAddressDeleteIsOnceAndAddressOnly`. After the nil-UUID migration, reassignment queues one `conntrack -D` for that address. The fake conntrack sees the delete once. A second apply does not queue it again. An address that was never in the previous or pending set is not queued. Changing `MemberID` on the same address still queues that address: the tuple the delete targets is the address, so the new device's established flows on the reassigned address are deleted once. That is the collateral the r14 M-1 fix requires, and it is not filed again. Restart between queue and success remains `TestReviewR14_NilUUIDMigrationKeepsPendingUntilDeleteSucceeds` on `main`.

### 8. Ruleset read-back, aside from the tc redirect

`TestReviewR15_ApplyListsTheRulesetOnceBeforeVerified`. `Apply` lists the ruleset once, then `VerifyServing`. The serving flag is set after `Apply` returns, and the heartbeat writes it with no second list. The gap is the rest of that `Step` plus up to `HeartbeatSeconds` (default 10, `node/agent/config.go:125-126`). A ruleset edited in that window is not re-verified until the next apply. The measurement is that single list; it is not a longer hole, and it is not filed separately from M-6.

Nft families the verifier already classifies, from `TestFollowup10_ForeignChains` and `hookCrosses` (`node/enforce/verify_foreign.go:125-141`), were not re-run in the lab because verify rejects them before serving:

- `netdev` ingress or egress that mangles on the member interface or the uplink fails closed (`netdev_mangle_on_member`, `netdev_mangle_on_uplink_egress`). The same mangle on some other device is tolerated.
- A `bridge` forward chain with `queue` fails closed. An accept that only matches, counts, logs, or continues does not stop a later base chain; `verifyForeign` tolerates that shape on purpose.
- An `arp` table is tolerated (`arp_drop_tolerated`). `hookCrosses` returns not-crossing for family `arp`. An arp table does not forward the IPv4 packet the lab injects.
- A flow statement and a flowtable object that names the member or uplink devices fail closed (`flow_offload`, `ReasonFlowtable`).

`TestReviewR15_TcMirredAndPolicyRoutingBypass/ip_rule` passed. `ip rule add from 100.64.9.9 lookup 100` with `203.0.113.10/32 dev fv-ih table 100` left verify at 0, incremented `ctunku` by 1, and captured nothing on `fv-ih` (`ip rule: ctunku=1 uplink="\n"`). That rule still egresses the policed uplink, so the inet forward drop saw the packet. A rule that egresses a third interface was not installed. The forward chain policy is `drop` (`node/enforce/render_nft.go:104`), so a forwarded packet that matches no accept is dropped on whatever egress the route picked; this review did not capture that third interface. M-6 is the path that skipped the forward hook.

### 9. Fair-Use generation cost, aside from the two ACK holes

`TestReviewR15_TierOscillationEmitsOneUpwardGeneration`. `SelectTier` has no hysteresis; equality enters the new tier (`internal/fup/tier.go:56-82`). Inside one period, ordinary accumulation does not lower the tier. One baseline evaluation plus one threshold crossing plus five oscillations of the same total produces two upward generation increments, not one event per sample. An unchanged tier does not increment (`enforcedOutputChanged`, `internal/fup/engine.go:926-954`). One member's own traffic can emit at most one upward `fup.member_generation_changed` per tier crossing per period. Three tiers means at most two upward crossings after the baseline generation. There is no per-hour storm from oscillating around a threshold the member has already crossed.

`RecompileBatch` on `EventFUPMemberGenerationChanged` calls `exitsCarryingMembers` with that member only (`internal/desiredstate/compiler.go:920-940`). The Fair-Use world setup asserts desired generation 1 on the carrying exit and 0 on the unrelated exit before either ACK proof runs. `TestFairUseActivationRecompileAndNodeAck` on `main` already covers duplicate claimed events and the requirement that both carrying exits ack before `applying` clears. Those are not re-filed. Each generation still costs a new desired generation on the carrying exits, an agent apply, a tc reload, a lease renewal, and an observation. The two holes in that ACK are M-4 (clears early while frozen) and M-5 (sticks after the last carrying exit is unpublished). No observation yet does not ack: `obsOK` is false when the latest row is missing (`internal/fup/engine.go:1561-1587`).

## Known, reconfirmed, not filed

- Portal follow-up 14: `awaiting_member_grants` copy and the `identity_conflict` badge.
- Handover follow-ups 9 and 10: operator wording for D-072, D-074, and D-075, including the dual-uplink sentence §11 does not contain.
- Usage follow-up 3, devstack follow-ups 13 and 14, members follow-up 6, isolation follow-up 4, enrollment follow-up 3, security-ops follow-ups 8 and 9.
- Contract revision 12: the awaiting-member-grants example still has `errors: []`. The OpenAPI description and `docs/contracts/README.md` are the normative text and require one entry.
- Multi-replica limiter budgets are a deployment note. The agent-listener key itself is per node URN.
- Two clones inside one `/24` or `/48` are the documented D-072 limit.
- r14's adopted proofs (group fence freshness, one-shot grace, nil-UUID conntrack retention, whole-ruleset nft read, node-keyed ingress, `evictIdle` walking the whole table) were not re-opened. The new holes are the ones in the table above.
