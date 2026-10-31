# Adversarial Review r16: installer-applied update results, the identity-freeze latch, foreign-tc containment, and the restore hold

Branch: `r16-rev13-d089-d099-review` (from `main` at `5575470`)
Date: 2026-10-31
Reviewing model: `grok/grok-4.7`

Scope is contract revision 13 (#261, API 1.9.0, hotfix #264), installer-applied results (D-089 / D-097: fleet-update #273, updater #269, installer #239), the durable identity-freeze latch (agent-core #272, wire-nodes #267), foreign-tc containment and the member-path check (D-090 / D-096 / D-100, netlab #262, policies/usage #263), the restore recovery hold (D-095, policies/usage #270, security-ops #265) and the generation-advance regression it exposed (D-099), members follow-up 7 (#268), and devstack follow-up 15 (#266).

Proofs are `TestReviewR16_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. Known in-flight work is reconfirmed at the end and is not filed again. Product code was not changed.

Commands (2026-10-31, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`). Integration tests used their own databases through `dbtest` (`FVPN_TEST_DATABASE_URL`). No netlab was started. No container was created.

```
go test -count=1 -timeout 180s -run 'TestReviewR16_' ./node/agent/ ./node/enforce/ ./node/updater/
go test -count=1 -timeout 60s -run 'TestReviewR16_StateCacheDropsGenerationOnlyAdvance' ./internal/nodes/httpwire/
go test -tags integration -count=1 -timeout 15m -p 2 -run 'TestReviewR16_' ./internal/fleetupdate/ ./internal/nodes/httpwire/ ./internal/desiredstate/ ./internal/members/
go test -tags integration -count=1 -timeout 180s -run 'TestReviewR16_InstallerCannotSelfPromote' ./internal/fleetupdate/
```

`node/enforce` and `node/updater` exited 0. `node/agent` exited 1 only on `TestReviewR16_FailedFreezePersistIsLostOnRestart`. The integration run exited 1 on the `TODO(review)` lines quoted below. `internal/members` exited 0. `TestReviewR16_InstallerCannotSelfPromote` and `TestReviewR16_RestoreHoldBoundaries` exited 0 on the run after the fixture fixes (incarnation revoke before re-enroll; the post-freeze node's receipt clock set after the recovery record).

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| H-1 | High | wire-members | `internal/desiredstate/interfaces.go:471-488`, `internal/desiredstate/compiler.go:507-511`, `internal/leases/leases.go:162-168` | An `authorization_unconfirmed` device stays `member_devices.status=authorized`. The desired-state binding and the lease device list both still grant it. |
| M-1 | Medium | agent | `node/agent/freeze.go:10-21`, `node/agent/engine.go:241`, `node/agent/engine.go:333-338` | A failed `identity_frozen` write is not retried. The next process starts unfrozen with `storageOK` and can enforce before the next observation. |
| M-2 | Medium | fleet-update | `internal/fleetupdate/installer_results.go:91-102` | An accepted installer `rolled_back` leaves `current_release_id` and `last_good_release_id` on the candidate the node just left. |
| L-1 | Low | fleet-update | `internal/fleetupdate/installer_results.go:99-114`, `db/migrations/00191_installer_update_results.sql:3-18` | One node credential can insert an unbounded number of `installer_update_results` rows. Each acceptance also appends an audit and bumps `nodes.resource_version`. |

## Findings

### H-1 (High): a fenced device still receives a desired-state binding and a lease device id

- **Where:** `PublicStatus` reports `pending` while `status=authorized` and the read-back is not current (`internal/devices/service.go:70-77`). The NetBird sync sets `allow=false` when a `device_authorization_fences` row exists (`internal/devices/worker.go:92-99`). The desired-state binding query sets `is_revoked` only from `md.status <> 'authorized' OR md.revoked_at IS NOT NULL` (`internal/desiredstate/interfaces.go:471-488`, the flag at line 481). The compiler drops a binding only when that flag is set (`internal/desiredstate/compiler.go:507-511`). `ListLeaseEntitlementsForGroups` selects `member_devices` with `status='authorized' AND revoked_at IS NULL` and does not read the fence (`internal/leases/leases.go:162-168`).
- **Scenario:** D-077 leaves `member_devices.status` at `authorized` and records the fence separately. Public status and the next NetBird sync treat the device as unconfirmed. The signed desired state and the lease entitlement do not. A device that still has an open binding is compiled into the exit's member bindings, and its id is attached to the member's lease device list, while an actually revoked device is absent from both. The NetBird `allow=false` is a later sync. Until that sync, and on any path that enforces the signed binding rather than the NetBird peer flag, the unconfirmed device is a granted peer.
- **Proof:** `TestReviewR16_FencedDeviceReceivesNoGrant` in `internal/desiredstate/review_r16_fence_grant_integration_test.go`. The open device was bound and leased. The revoked device was in neither list.

  ```
  TODO(review): authorization_unconfirmed device 29b64654-4618-4d13-97ac-757074f3c9af is still granted (desired-state binding=true, lease device id=true) while member_devices.status stays authorized
  ```
- **Fix:** wire-members. Both grant readers have to treat a row in `device_authorization_fences` as revoked for compilation: the `is_revoked` expression in `internal/desiredstate/interfaces.go` and the device query in `internal/leases/leases.go`. Public status and the NetBird sync already do.

### M-1 (Medium): a failed identity-freeze persist is lost on the next process

- **Where:** `freezeIdentity` sets the memory latch and returns immediately on a later call (`node/agent/freeze.go:10-14`). `SetMeta(ctx, "identity_frozen", "1")` runs after `withdraw`. On error it sets `storageOK=false` and does not clear the memory latch (`node/agent/freeze.go:17-21`). A new engine starts with `storageOK: true` (`node/agent/engine.go:241`) and sets `identityFrozen` only when the meta value is `"1"` (`node/agent/engine.go:333-338`). `Decide` blocks serving on `!StorageOK` or `IdentityFrozen` (`node/agent/decide.go:134`, `node/agent/decide.go:468`). `ackApplied` returns while the memory latch is set (`node/agent/engine.go:1015-1018`). `flush` calls `freezeIdentity` and then `MarkAccepted` (`node/agent/observe.go:231-237`). `Step` flushes before enforce (`node/agent/engine.go:502`) and again after enforce (`node/agent/engine.go:535`, `544-545`).
- **Scenario:** Central accepts an observation with `identity_frozen: true`. The agent withdraws, then `SetMeta` hits the 5 s timeout because another connection holds `BEGIN IMMEDIATE` on `agent.db`. The memory latch stays set, so the process does not serve (`storageOK` is false) and does not try the write again. `MarkAccepted` can still commit, which drops the pending report. After restart the meta key is absent, `identityFrozen` is false, and `storageOK` is true. The next `Step` has nothing frozen to replay, so it enforces before the post-enforce observation can freeze it again. The window lasts until that new observation round-trip. A disk-full or locked store during the one write, followed by any restart, is enough. While the process stays up, the memory latch plus `storageOK` keep it non-serving.
- **Proof:** `TestReviewR16_FailedFreezePersistIsLostOnRestart` in `node/agent/review_r16_freeze_test.go`. A second SQLite connection held `BEGIN IMMEDIATE`. `freezeIdentity` returned with the memory latch set, `storageOK` false, and no meta row. The second call did not write. After `ROLLBACK` and `restartEngine` the new process was unfrozen.

  ```
  TODO(review): identity_frozen meta="" present=false after SetMeta failed; freezeIdentity does not retry once the memory latch is set, and the next process starts unfrozen with storageOK (meta key was not 1)
  ```
- **Fix:** agent. A failed `SetMeta` of `identity_frozen` has to stay pending across `Step` and across process start. The early return at `freeze.go:11` must not skip the durable write. `loadRuntime` needs a fail-closed signal when the freeze was decided and the meta row never landed; `storageOK` alone does not survive the process.

### M-2 (Medium): an accepted installer rollback does not record the restored release

- **Where:** `acceptInstallerResult` computes `lastGood = &current` when the state is `succeeded`, `already_current`, or `rolled_back` (`internal/fleetupdate/installer_results.go:91-94`). The node update writes `current_release_id` and `last_good_release_id` only when `$3` is `success` (`internal/fleetupdate/installer_results.go:99-102`). `success` is `succeeded` or `already_current`. The central path writes `current_release_id` from the reported current release and `last_good_release_id` when `success || rolled_back`, and only when `observed_update_generation` is not newer (`internal/fleetupdate/results.go:242-243`). The installer path has no generation high-water check.
- **Scenario:** The node submits a verified installer `succeeded` for an eligible release. Central moves `current_release_id` and `last_good_release_id` to that release. The node then submits a second installer result, new `job_id` and `result_id`, state `rolled_back`, `current_release_id` the release it is actually running, `previous_release_id` the candidate, `error_code=HEALTH_CHECK_FAILED`. The rollback evidence is accepted. Both version columns stay on the candidate. Fleet scheduling, the node detail, and the next update decision treat the node as running a release it has already abandoned. The restored release in this proof was still in the serving set and was not revoked; the bug is the missing write, not a path onto an ineligible release.
- **Proof:** `TestReviewR16_InstallerRollbackMovesRecordedVersion` in `internal/fleetupdate/review_r16_installer_integration_test.go`.

  ```
  TODO(review): accepted installer rolled_back left current_release_id=01a190ce-2c52-785e-b215-f06ae448f24f last_good_release_id=01a190ce-2c52-785e-b215-f06ae448f24f; want both 01a190ce-2c4c-765a-a3e1-fb44220ef2f4 (the release the node is running)
  ```
- **Fix:** fleet-update. The installer version write has to apply `current_release_id` and `last_good_release_id` for an accepted `rolled_back`, using the same `success || rolled_back` condition as `internal/fleetupdate/results.go:242`, and it has to keep the central generation high-water so an older installer receipt cannot move a newer observation.

### L-1 (Low): installer evidence rows, audits, and the node ETag are unbounded per node

- **Where:** Every new `(node_id, job_id)` inserts one immutable row (`internal/fleetupdate/installer_results.go:111`, `db/migrations/00191_installer_update_results.sql:3-18`). The primary key is `(node_id, job_id)`. The latest-result index is `(node_id, incarnation_id, received_at DESC, job_id DESC)`. There is no per-node cap. Each acceptance updates `nodes` and therefore increments `resource_version` (`installer_results.go:102`) and appends one `update_result.accept` audit (`installer_results.go:116-122`). The agent HTTP budget is 5/s, burst 10 (`internal/config/config.go:87`). The domain `SubmitResult` path used by this proof has no row cap of its own. The node detail reads one row (`ORDER BY … LIMIT 1` in `internal/nodes/update_view.go`).
- **Scenario:** A node mints a new `job_id` per submission of the same eligible release. All nine were accepted. Nine rows, nine installer accept audits, and `resource_version=11` (the node started at 2 after insert and incarnation bind). Nothing rejected the ninth. The projection stays one indexed row, so this is not a promotion bug and not a per-read scan. It is steady growth of an immutable table, of `audit_events`, and of the node ETag, at the agent rate, for as long as the credential is valid. Exact replay of one `job_id` does not add a row; the growth is new ids.
- **Proof:** `TestReviewR16_InstallerResultCardinalityIsBounded` in `internal/fleetupdate/review_r16_installer_integration_test.go`. The bound under test was 8.

  ```
  TODO(review): one node credential inserted 9 installer_update_results rows, 9 installer accept audits, resource_version=11, with no cardinality rejection (last=<nil>)
  ```
- **Fix:** fleet-update. Cap retained installer rows per node (or per node and release) and stop bumping `resource_version` when the installed version and the latest installer summary did not change. The HTTP agent rate is a speed limit, not a cardinality limit.

## What held

### 1. Installer self-promotion, aside from rollback recording and row growth

`TestReviewR16_InstallerCannotSelfPromote` passed. Eligible, in `releases.Service.Eligible` (`internal/releases/service.go:209-223`), means `release_verifications.state='verified'`, `fleet_config.supported_release_policy='allowlist'`, the id is in `approved_release_ids` and in `supported_release_ids`, and the id is not in `revoked_release_ids`, then the manifest is revalidated and its hash is checked. There is no per-node channel, plan, or arch filter on this path. Central update intent still filters artifacts by the node's arch and OS. This review did not build a release whose artifacts omit the node's arch.

A result whose `job_id` is an open central job is `409 STATE_CONFLICT`. `fleet_node_state.current_job_id` and the job state stayed put, and the batch did not complete. An unknown release is `404 RESOURCE_NOT_FOUND` and the recorded version stays the previous release. A release that is no longer approved is `409 STATE_CONFLICT`. A revoked release, submitted by a node that is still on the previous release, is `409 STATE_CONFLICT` and that node's version stays put. A body incarnation that is not the authenticated one is `403`. An exact replay returns the stored ack and does not move `resource_version`. The same `job_id` with a different `result_id` is `409 ACK_CONFLICT` on the first try and on the repeat. After the current incarnation is revoked and a replacement is enrolled, the old credential and the new credential both get `403` for the old body, and `installer_update_results` stays at one row.

The server does not check that `job_id` is the UUIDv5 the updater derives. Any UUID is accepted as an installer job id once the release is eligible. That is how a node records evidence for a release it was not offered. The eligible-release gate is what stops the version move.

### 2. The verified baseline

`TestReviewR16_DownloadRedirectCannotBlessDifferentBytes` passed. A redirect to bytes that do not match the signed SHA-256 is rejected. A redirect to the signed bytes is stored as those bytes. Existing `TestD089_SeedRefusesUnverifiedTree`, `TestD089_SeedRefusesDifferentJournal`, and `TestD089_SeedTakesSharedLock` already refuse a tree that does not match the signed release, a journal that belongs to another install, and a seed that runs while an apply holds the update lock. `TreeHash` covers path, mode, and bytes, and seed rejects symlinks. Idempotence is the tree hash, not the archive name. `TestD089_SeedThenInstallerApply` and `TestD089_SeedCLIAndStatus` leave `LastGoodRelease` empty and `Health` nil after seed. A seeded baseline is not a health pass. This review did not re-run those existing tests.

### 3. The durable 404

`TestReviewR16_Installer404BackoffIsCapped` passed. `backoff` doubles from `Initial` and stops at `MaxWait`. Jitter `0` yields a zero wait. Jitter `0.5` scales the capped delay. A `404` with `RESOURCE_NOT_FOUND` leaves the outbox row `pending`, schedules the next send one initial interval out, and does not move journal pointers. Reopening the journal and flushing before that deadline does not send again and does not drop the payload. `ApplyPending` in this harness sets `CurrentRelease` and `LastGood` to the installed release before the flush; the 404 does not change them. The node keeps that installed tree and keeps the pending result until central accepts a known release. Forwarding is the agent's decision; this updater path does not withdraw it. `TestD089_Installer404IsRetriable` on `main` is the classification of that HTTP status. This review did not re-run it.

### 4. The identity latch, aside from the failed persist

`TestReviewR16_FrozenIncarnationDoesNotAckOrKeepServing` passed. `freezeIdentity` withdraws in the same call. `ackApplied` of a generation the store had already accepted leaves `appliedGen` unchanged. The heartbeat is non-serving with reason `identity_frozen`. The meta value is `"1"`. After `restartEngine` the latch is still set, and a later `ackApplied` still no-ops. `TestD088_IdentityFrozenTLSAndRestart`, `TestD088_AbsentAndFalseDoNotChangeIdentity`, `TestD088_AT07_AT29_HTTPObservationFreezeLiveReceipt`, and `TestD088_AT07_HTTPFrozenRenewalThenObservationAndRetiredIncarnation` on `main` cover the live annotation (present only when `mapping_frozen` is set on the current incarnation, absent otherwise, identical on report-id and idempotency-key replay) and the replacement incarnation. This review did not re-run them. `identity_frozen` does not appear under `apps/portal`. The central lease clamp of a frozen incarnation is follow-up 27, migration `00190`. The r15 finding that the detecting renewal minted a 24 h member-grant lease was closed on `main` and is not refiled.

### 5. Containment, aside from the in-flight isolation follow-ups

`TestReviewR16_QuarantineNeverLowersUplink` passed. `Quarantine` lowers `Datapath.MemberIngress` only (`node/enforce/quarantine.go:62-70`). `validateDatapath` requires `MemberIngress != Uplink` and both names match `^fv[a-z0-9_-]{1,13}$` (`node/enforce/validate.go:24-37`), so a shared, bonded, or VLAN interface that is also the uplink is rejected before any link change. `TestReviewR16_ContainmentEvidenceIsCapped` passed. `recordForeign` keeps at most 16 tc reads, 4 objects, and 4 actions, and replaces any token outside `^[a-zA-Z0-9_:-]{1,32}$` with `redacted` (`node/agent/quarantine.go:24-68`). `InitialQuarantine` is kept once (`node/agent/quarantine.go:128-130`). Creating a tc filter or renaming an interface takes `CAP_NET_ADMIN` or an equivalent privileged daemon (NetBird, container runtime, networkd). There is no hysteresis counter: a foreign filter withdraws every time, and a clean verified Apply plus the member-path check is the recovery. That repeat is the containment, not a new outage amplifier, for a caller that is already root on the node. ISO-F6-01 and ISO-F6-03 stay with their owners; this review did not start a netlab.

### 6. Member-path expectations

`TestReviewR16_MemberPathDoesNotServeAReplacedAddress` passed against `SystemPlane.MemberPath`, not the memory plane. The first clean topology is stored. A superset that still contains the learned address keeps serving and does not replace the stored hashes. Replacing the learned address stops serving and leaves the stored hashes unchanged. A plane error does not write a partial expectation. Restoring the original address clears the hold on the next check, so a flapping member route does not stick the node in the D-096 hold once the learned route is back. `checkMemberPath` assigns `e.memberPath` only while it is nil (`node/agent/memberpath.go:125-132`). `Step` captures only in that same case, before `Apply` (`node/agent/engine.go:899-903`). A failed Apply does not publish a half-updated expectation. Refresh of a legitimately new address on a newer published generation is D-100, owned by agent-core follow-up 23. This build fails closed on that change. It is recorded under Known and is not filed.

### 7. Restore-hold boundaries, aside from the cached generation

`TestReviewR16_RestoreHoldBoundaries` passed. The cutoff is `coalesce(max(operations.created_at) for that node and principal, the recovery record's created_at, the ledger updated_at while mode is not normal)` (`internal/desiredstate/compiler.go:172-201`). Receipt time is `node_observations.received_at`, the central clock at insert (`internal/nodes/observations.go:198`), and it must be strictly after the cutoff.

A replay of the pre-freeze receipt left the hold in place. A new receipt whose central clock was still behind the cutoff left the hold in place while the node's `observed_at` was a day ahead. Another node's fresh receipt released only that node. A node created during recovery has no freeze-operation row, so its cutoff is the recovery record, and it stayed held until its own receipt. `UPDATE exits SET publication_intent='published'` and a device revocation both committed while the holds were in place, and neither released a node. The new node's receipt did not release the restored node. The restored node's own later receipt did.

### 8. Revision 13 error-code consumers

`TestReviewR16_ForeignFilterCodeIsNotHealthFallback` passed. `noteApply` maps `enforce.CheckError.Reason` through `api.ErrorCode.Valid()` (`node/agent/engine.go:1030-1043`), so `TC_FOREIGN_FILTER` is stored as that code. `contracts/revision13_test.go` pins the enum, `UpdateResultSource` `central|installer`, and `TC_FOREIGN_FILTER`. Portal `ERROR_CODE_MESSAGES` includes both `HEALTH_CHECK_FAILED` (`apps/portal/src/api/errors.ts:194`) and `TC_FOREIGN_FILTER` (`apps/portal/src/api/errors.ts:270`). `isKnownErrorCode` falls through for an unknown code. `view-models.ts` special-cases `AWAITING_MEMBER_GRANTS` and `LEASE_EXPIRED`; other codes, including `TC_FOREIGN_FILTER`, render as the code plus the dictionary text. The mock at `apps/portal/src/mocks/data.ts:685` hardcodes `HEALTH_CHECK_FAILED` on a fixture job. The worker sets `DEPENDENCY_UNAVAILABLE` or `STATE_CONFLICT` for discovery failure (`apps/worker/jobs_nodes.go:486-488`) and does not switch on the new members. `TestD093_ISO02_HTTPTCForeignFilterStoredEvidence` on `main` is the stored-evidence HTTP proof. This review did not re-run it. No consumer found by this search drops the new code into the old health fallback.

### 9. Members HTTP path, aside from the fence grant

`TestReviewR16_AdminDeviceCursorIsStable` passed. Five pending devices, page size 2. Authorizing the first id and inserting a newer UUIDv7 during the walk neither skipped nor duplicated an original id, and the new id appeared once. Device ids are `uuid.NewV7()` (`internal/devices/service.go`). The keyset is `(member_id, id)`. A lower id inserted after page 1 would sit behind the cursor; production mint does not do that.

`TestReviewR16_ConcurrentAuthorizeHitsDeviceLimit` passed. Two pending devices, `device_limit` 1, each with a closed historical binding so the winner can open a new interval. One call succeeded and one returned `devices.ErrLimit`. The authorized count stayed 1. `check_device_authorization` locks the member and the entitlement and raises `23514` when the authorized count is already at the limit (`db/migrations/00005_evidence_and_deadlines.sql`). Re-authorizing the winner after a fence returned the same operation id, and the operations count stayed 1.

The grant hole for a device that is already authorized and then fenced is H-1. These two proofs are the cursor and the concurrent limit.

## Known, reconfirmed, not filed

- **D-099, wire-nodes follow-up 29.** `cachedStateSigner.Sign` copies the cached `ConfigGeneration`, `IssuedAt`, and `ExpiresAt` onto the candidate before the JCS compare (`internal/nodes/httpwire/state_cache.go:27-38`). A generation-only advance compares equal and returns `errStateUnchanged`. The HTTP cache then serves the previous signed generation after `applied_config_generation` has moved.

  `TestReviewR16_StateCacheDropsGenerationOnlyAdvance`:

  ```
  TODO(review): D-099 cachedStateSigner dropped a generation-only advance from 128 to 134 (err=desired state inputs unchanged signed=false)
  ```

  `TestReviewR16_GenerationAdvanceAfterCacheIsServed` (first `GET` generation 1, then `applied_config_generation` set to 6 with `resource_version` bumped):

  ```
  TODO(review): D-099 HTTP cache served generation 1 after applied_config_generation 6; want a signed generation strictly above the applied generation (first offer was 1)
  ```

  SECOPS-21 stays open until follow-up 29. The hold itself, proved above, is not this bug.

- **Keyed HTTP replay of an installer result, wire-nodes follow-up 29.** `replayAgentReceipt` for `SubmitUpdateResult` reads `update_results` only (`internal/nodes/httpwire/replay.go:48-59`). The stored ack is in `installer_update_results`. The first `POST /api/v1/agent/update-results` with an idempotency key was 202. The repeat of that key was 500, which is `pgx.ErrNoRows` from that select surfaced as `INTERNAL_ERROR`. Domain replay of the same body, proved in `TestReviewR16_InstallerCannotSelfPromote`, still returns the stored ack.

  `TestReviewR16_InstallerResultReplayReturnsStoredAck`:

  ```
  TODO(review): keyed replay of an installer result did not return the stored ack: 500 {"error":{"code":"INTERNAL_ERROR","message":"Internal error. Quote the request ID when reporting this problem.","request_id":"01a190ce-2f3e-7bec-8ef1-e90d0729ccf2","retryable":false,"details":{}}}
  ```

- **D-100 non-refresh, agent-core follow-up 23.** Same-generation re-check does not replace a stored member-path expectation. The node stops serving when the learned address is gone. See What held §6. No out-of-band override was added.

- **ISO-F6-01** (latched interface-change `TC_FOREIGN_FILTER`, netlab follow-up 14) and **ISO-F6-03** (forwarding loss before the whole-node deadline, agent-core follow-up 23). Not duplicated.

- **Installer follow-up 3.** Devstack #266: the packaged write allowlist does not include the NetBird socket, so the product service failed its first start. Not refiled.

- **r15 H-1.** The detecting renewal no longer mints a 24 h member-grant lease. Follow-up 27 on `main` clamps the frozen incarnation. Not refiled.
