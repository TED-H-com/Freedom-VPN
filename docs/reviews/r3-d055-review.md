# Adversarial Review `r3-d055-review`: Management Peer Discovery (`D-055`)

- **Branch:** `r3-d055-review` (commit `f710843`)
- **Date:** 2026-10-29 (`2026-10-29 22:05:58 -0400`)
- **Reviewing model:** `grok/grok-4.7` (recorded in commit `f710843` `Agent:` trailer; not stated in `.agent/REVIEW.md` body)
- **Decision / scope reviewed:** `D-055` management peer discovery through one-off enrollment correlation groups (`§6.1`, `§6.2`, `§6.4`, `§7.2`, `§14.5`, `§15.3`; `internal/netbird/discovery.go`, `internal/netbird/operations.go`, `apps/worker/jobs_nodes.go`, `internal/nodes/httpwire/revocation_adapter.go`, `docs/netbird/ADAPTER.md`)
- **Adopted proof files on `main`:**
  - [`internal/netbird/discovery_review_test.go`](../../internal/netbird/discovery_review_test.go) (adopted in `#109`)
  - [`apps/worker/jobs_nodes_discovery_review_integration_test.go`](../../apps/worker/jobs_nodes_discovery_review_integration_test.go) (adopted in `#109`)
  - Supporting HTTP/integration fake hooks in [`internal/netbird/adapter_test.go`](../../internal/netbird/adapter_test.go) and [`apps/worker/jobs_nodes_discovery_integration_test.go`](../../apps/worker/jobs_nodes_discovery_integration_test.go) (adopted in `#109`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | 1. A torn setup-key / correlation-group read in `internal/netbird/discovery.go` and `apps/worker/jobs_nodes.go` sees `used_times=0` with one group member when the peer joins between the two HTTP GETs and permanently freezes the live enrollment with `group_peer_without_key_use`. | closed | #109 |
| High | 2. A second peer that joins after the initial group read in `internal/netbird/discovery.go` and `apps/worker/jobs_nodes.go` is not re-checked before `recordPeer`, so the first peer is bound and the correlation group is deleted. | closed | #109 |
| High | 3. Revocation before `recordPeer` in `internal/netbird/discovery.go` and `internal/nodes/httpwire/revocation_adapter.go` revokes the setup key and deletes the correlation group while leaving the enrolled machine peer in `fvpn-unpublished-nodes` and auditing `node.identity_withdrawn`. | closed | #109 |
| High | 4. Correlation-group isolation in `internal/netbird/discovery.go` (`ensureCorrelationGroupUnreferenced`) ignores policy rule `authorized_groups` maps and `/api/dns/nameservers` group references. | closed | #109 |
| High | 5. Legacy one-group setup keys without a `node-enroll-group:` binding in `internal/netbird/discovery.go` (`loadSetupKeyAndGroup`) return retryable `management_peer_unconfirmed` instead of a terminal conflict requiring revoke and re-enroll. | closed | #109 |
| Medium | 6. Group `peers_count` disagreement with `len(group.Peers)` in `internal/netbird/discovery.go` (`validateCorrelationPeer`) is ignored, binding `Peers[0]` from a truncated or inconsistent group payload. | closed | #109 |
| Medium | 7. A policy-reference conflict during correlation cleanup after `recordPeer` in `internal/netbird/discovery.go` (`cleanupEnrollmentCorrelation`) leaves the `node:` management peer binding committed in `external_bindings`. | closed | #109 |
| Medium | 8. Correlation cleanup and withdrawal in `internal/netbird/discovery.go` (`cleanupEnrollmentCorrelation` and `WithdrawEnrollmentCorrelation`) mutate NetBird before writing an `external_operations` receipt. | closed | #109 |
| Low | 9. Adapter documentation in `docs/netbird/ADAPTER.md` still stated that `CreateNodeSetupKey` assigns only `fvpn-unpublished-nodes` instead of the two `D-055` auto-groups. | closed | #109 |

## What Held

- **Exact two-group check on the setup key:** `validateCorrelationSetupKey` (`internal/netbird/discovery.go`) rejects a missing correlation group, a missing shared group, a third auto-group, a non-`one-off` type, `ephemeral: true`, `revoked: true` when `used_times != 1`, and `used_times > 1`. Covered by `TestAT06_Discovery_RejectsSetupKeyDriftAndExtraUses` in `internal/netbird/discovery_safety_test.go`.
- **Candidate peer identity checks:** `validateCorrelationPeer` rejects an empty peer ID, a non-empty `user_id` (`correlation_peer_user_owned`), and `ephemeral: true` (`correlation_peer_ephemeral`), and never consults `name`, `ip`, `dns_label`, `hostname`, `serial_number`, or timestamps. Covered by `TestAT06_Discovery_IgnoresPeerMetadataAndOrder` and `TestAT06_Discovery_RejectsUserOwnedAndEphemeralPeers`.
- **Direct policy rule and route references:** `ensureCorrelationGroupUnreferenced` rejects a correlation group that appears in `rule.Sources`, `rule.Destinations` (`GroupRef`), or `route.Groups`. Covered by `TestAT06_Discovery_RejectsReferencedCorrelationGroup`.
- **Central incarnation fences before `recordPeer`:** `recordManagementPeerTx` (`apps/worker/jobs_nodes.go`) locks `nodes` and `node_incarnations` `FOR UPDATE`, rejects a revoked node, a revoked incarnation, a superseded `current_incarnation_id`, and an existing `node_incarnations.peer_id` for a different peer, and rolls back `recordPeer` when `commit` returns an error. Covered by `TestAT06_D055_DiscoveryNeverRunsAfterRevoke`, `TestAT06_D055_DiscoveryClaimLostBeforeBinding`, and `TestAT06_Discovery_FenceFailureBeforeBindDoesNotWriteBinding`.
- **Revocation after `recordPeer` committed:** Once `node:<node>:<incarnation>` is recorded in `external_bindings` (even if `node_incarnations.peer_id` is still `NULL`), `RevocationAdapter.DeletePeer` finds the binding, calls `DELETE /api/peers/{id}`, verifies `404` on `GET /api/peers/{id}`, and marks the binding `withdrawn`. Covered by `TestAT07_D055_Review_RevokeAfterCentralCommitDeletesPeer`.
- **Heartbeat cannot clear a frozen incarnation:** `internal/nodes/observations.go` preserves `provisioning_status = 'failed'`, `safety_status = 'failed'`, and the conflict detail once `management_peer_conflict_code` is set, and does not enqueue another `node.discover_peer` job. Covered by `TestAT06_D055_ConflictIsTerminalAndHeartbeatDoesNotClear`.
- **Malformed upstream responses:** Truncated JSON, HTML error pages, and empty bodies from `/api/setup-keys/{id}`, `/api/groups/{id}`, `/api/peers/{id}`, `/api/policies`, and `/api/routes` return retryable `BadResponse` errors and never write a binding or freeze the incarnation. Covered by `TestAT06_D055_Review_MalformedReadsDoNotBind`.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### 1. [High] A torn setup-key / correlation-group read freezes a live enrollment as a permanent conflict
- **Where:** `internal/netbird/discovery.go:79-111`, `internal/netbird/discovery.go:258-262`, `apps/worker/jobs_nodes.go:234-268`.
- **Scenario:** Enrollment commits `node.provision` and `node.discover_peer`. `DiscoverNodePeer` runs while the node's `netbird up` is in flight: `c.getSetupKey` reads `GET /api/setup-keys/{id}` and receives `used_times: 0, state: "valid"`; the node finishes registering with NetBird (`used_times` becomes `1` and the peer is added to `fvpn-enroll-peer-<hash>`); `c.getGroup` reads `GET /api/groups/{id}` and receives `peers: [{id: "nb-peer-live"}]`. `validateCorrelationSetupKey` sees `key.UsedTimes == 0` and `peerCount == 1` and returns `conflictFailure("group_peer_without_key_use")`, causing `freezeDiscoveryConflict` to permanently freeze the incarnation.
- **Proof:** `TestAT06_D055_Review_TornReadKeyZeroThenPeerAppears` (`internal/netbird/discovery_review_test.go`), `TestAT06_D055_Review_StaleKeyUseFreezesLiveEnrollment` (`apps/worker/jobs_nodes_discovery_review_integration_test.go`).
- **Fix (#109):** Paired fresh key/group reads after the candidate read and treated `used_times == 0 && peerCount == 1` as retryable `management_peer_unconfirmed` when a re-read still disagrees.

### 2. [High] A second peer that joins after the initial group read is not seen; the first peer is bound and the group is deleted
- **Where:** `internal/netbird/discovery.go:88-167`, `internal/netbird/discovery.go:349-388`, `apps/worker/jobs_nodes.go:196-232`.
- **Scenario:** After `c.getGroup` returns one peer (`peer-a`), a second peer (`peer-b`) joins the correlation group while `c.getPeer`, `ensureCorrelationGroupUnreferenced`, and `check()` run. `recordPeer` writes `node:<node>:<incarnation> -> peer-a` from the initial snapshot and `cleanupEnrollmentCorrelation` deletes the correlation group without re-reading its membership.
- **Proof:** `TestAT06_D055_Review_SecondPeerJoinsAfterGroupReadStillBinds` (`internal/netbird/discovery_review_test.go`), `TestAT06_D055_Review_PeerJoinsAfterSnapshotBindsFirst` (`apps/worker/jobs_nodes_discovery_review_integration_test.go`).
- **Fix (#109):** Re-read `/api/setup-keys/{id}` and `/api/groups/{id}` immediately before `recordPeer` and again inside cleanup before deleting the group.

### 3. [High] Revocation before `recordPeer` deletes the correlation group, leaves the peer in the shared group, and audits the incarnation as withdrawn
- **Where:** `internal/netbird/discovery.go:184`, `internal/netbird/operations.go:397, 639`, `internal/nodes/httpwire/revocation_adapter.go:96, 133, 170`, `apps/worker/jobs_nodes.go:393, 406`.
- **Scenario:** A node consumes its setup key and joins `fvpn-unpublished-nodes` and `fvpn-enroll-peer-<hash>`. Before `DiscoverNodePeer` reaches `recordPeer`, an admin revokes or reinstalls the node. `WithdrawEnrollmentCorrelation` revokes the setup key and deletes `fvpn-enroll-peer-<hash>` without reading its `peers` list, then `c.DeletePeer` finds no `node:<node>:<incarnation>` binding and returns `nil`, leaving the peer in `fvpn-unpublished-nodes` while auditing `node.identity_withdrawn`.
- **Proof:** `TestAT07_D055_Review_RevokeBeforeDiscoveryOrphansEnrolledPeer` (`internal/netbird/discovery_review_test.go`), `TestAT07_D055_Review_RevokeBeforeDiscoveryLeavesPeer` (`apps/worker/jobs_nodes_discovery_review_integration_test.go`).
- **Fix (#109):** Read the correlation group before deleting it during revocation, delete the single correlated machine peer with `404` read-back, and audit withdrawal truthfully with `incarnation_id` and `reason=setup_key_and_peer_withdrawn`.

### 4. [High] Policy `authorized_groups` and DNS nameserver groups are not checked for correlation-group isolation
- **Where:** `internal/netbird/discovery.go:155, 169`, `internal/netbird/reconcile.go:39`, `internal/netbird/testdata/nameserver.json`.
- **Scenario:** `ensureCorrelationGroupUnreferenced` only inspects `rule.Sources`, `rule.Destinations`, and `route.Groups`, ignoring `rule.AuthorizedGroups` maps and `/api/dns/nameservers` `groups` lists.
- **Proof:** `TestAT06_D055_Review_PolicyAuthorizedGroupsReferenceIgnored`, `TestAT06_D055_Review_NameserverGroupReferenceIgnored` (`internal/netbird/discovery_review_test.go`).
- **Fix (#109):** Added `AuthorizedGroups` and `/api/dns/nameservers` group checks to `ensureCorrelationGroupUnreferenced`.

### 5. [High] Legacy one-group setup keys retry as `management_peer_unconfirmed` instead of failing terminally
- **Where:** `internal/netbird/discovery.go:229-247`, `docs/nodes/WIRING.md:97`.
- **Scenario:** When the `node-enroll-group:<node>:<incarnation>` binding is missing, `loadSetupKeyAndGroup` returns retryable `NotFound` (`management_peer_unconfirmed`) instead of a terminal conflict.
- **Proof:** `TestAT06_D055_Review_LegacySingleGroupKeyIsNotTerminalConflict` (`internal/netbird/discovery_review_test.go`).
- **Fix (#109):** Treated a missing correlation-group binding or one-group setup key as a terminal conflict (`setup_key_auto_groups_changed`).

### 6–9. [Medium / Low] `peers_count` mismatch, post-bind cleanup conflict, missing cleanup/withdrawal operation receipts, and adapter doc drift
- **Proofs:** `TestAT06_D055_Review_GroupPeersCountMismatchStillBinds`, `TestAT06_D055_Review_ConflictDuringCleanupLeavesPeerBinding`, and `internal/netbird/discovery.go:349-388` (`internal/netbird/discovery_review_test.go`).
- **Fix (#109):** Verified optional counts against array lengths, rolled back tentative bindings on cleanup conflicts, persisted durable `enrollment-cleanup` / `enrollment-withdrawal` receipts in `external_operations`, and updated `docs/netbird/ADAPTER.md`.
