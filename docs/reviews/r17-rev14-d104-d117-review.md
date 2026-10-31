# Adversarial Review r17: retired-stream hand-off, deterministic installer receipts, fence-aware grants, and the meter barrier

Branch: `r17-rev14-d104-d117-review` (from `main` at `62f24e6d`)
Date: 2026-10-31
Reviewing model: `grok/grok-4.7`

Scope is contract revision 14 (#289, API 1.10.0), `revokeMemberGrant` and finite grants (D-117, #297), the retired-stream usage hand-off (D-105 / D-112 / D-115, #296 and #299, activation D-118 in #312), installer job identity and legacy receipts (D-110 / D-111 / D-113 / D-114, #300 and #307), fence-aware grants (r16 H-1, #291), the durable meter checkpoint and candidate release identity (#305) with the ISO-F6-01 recovery latch (#295), deadlines at the renderer’s millisecond quantum (D-104, #290 and #312), tc read-back evidence (D-106, #312), and the installer marker outside `inet fvpn` (D-116, #302 and #304).

Proofs are `TestReviewR17_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. The r16 findings H-1, M-1, M-2, and L-1 were re-run on this tree and are not refiled. Product code was not changed.

Commands (2026-10-31, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, go1.27.3). Integration tests used their own databases through `dbtest` (`FVPN_TEST_DATABASE_URL`). `pgrep` showed `mk/at-usage.mk` and `mk/at-members.mk` active, so no netlab and no stack was started. No container was created.

```
go test -count=1 -timeout 240s -run 'TestReviewR1[67]_' \
  ./internal/httpapi/ ./internal/config/ ./internal/entitlement/ \
  ./node/enforce/ ./node/updater/ ./node/agent/ ./node/internal/meter/ ./packaging/installer/
go test -tags integration -count=1 -timeout 15m -p 1 -run 'TestReviewR1[67]_' \
  ./internal/usage/ ./internal/fleetupdate/ ./internal/members/ ./internal/desiredstate/ ./internal/nodes/
go test -count=1 -timeout 120s -run 'TestReviewR16_StateCacheDropsGenerationOnlyAdvance' ./internal/nodes/httpwire/
go test -tags integration -count=1 -timeout 180s -p 1 \
  -run 'TestReviewR16_GenerationAdvanceAfterCacheIsServed|TestReviewR16_InstallerResultReplayReturnsStoredAck' \
  ./internal/nodes/httpwire/
```

The unit command exited 0 (`internal/httpapi` 0.081s, `internal/config` 0.008s, `internal/entitlement` 0.009s, `node/enforce` 0.007s, `node/updater` 0.208s, `node/agent` 15.052s, `node/internal/meter` 0.006s, `packaging/installer` 0.006s). The integration command exited 1 only on `TestReviewR17_LaterInstallerOutcomeCanCorrectTheRecordedRelease`. `internal/usage` 2.131s, `internal/members` 0.481s, `internal/desiredstate` 1.164s, and `internal/nodes` 0.509s exited 0. `internal/fleetupdate` exited 1 at 4.974s. The two `httpwire` commands exited 0 (0.007s and 0.882s).

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| M-1 | Medium | fleet-update | `internal/installerjob/id.go:13-16`, `internal/fleetupdate/installer_results.go:36-38` | The derived installer job id ignores outcome. A later eligible success after a pre-health seed, and a later eligible rollback after success, on the same incarnation are `ACK_CONFLICT` and leave `current_release_id` and `last_good_release_id` on the first receipt. |

## Findings

### M-1 (Medium): one installer receipt per release, manifest, and incarnation cannot record the release the node is actually running

- **Where:** `installerjob.ID` is a UUIDv5 of the requested release, the lowercased manifest digest, and the incarnation (`internal/installerjob/id.go:13-16`). State, result id, and attempt are not inputs. `acceptInstallerResult` loads any existing `installer_update_results` row for `(node_id, job_id)` and, when the payload hash differs, returns `ACK_CONFLICT` with “Installer job already has different evidence” (`internal/fleetupdate/installer_results.go:33-41`). The version write that would move `current_release_id` and `last_good_release_id` runs only after a new row is inserted (`internal/fleetupdate/installer_results.go:140-144`). The updater keeps an `ACK_CONFLICT` as `OutboxConflict` and continues; `OutboxRejected` is reserved for `409` `STATE_CONFLICT` whose reason is exactly `job_id_mismatch` (`node/updater/result.go:236-238`).
- **Scenario:** D-111 lets a failed-before-health installer result on a null baseline record the old release as current and last-good. The same incarnation then submits `succeeded` for the eligible candidate. The derived job id matches the seed, the payload hash does not, and central answers `ACK_CONFLICT`. Both version columns stay on the release the node has left. The inverse is the same id: an accepted `succeeded` moves the columns to the candidate, and a later eligible `rolled_back` onto the still-serving previous release is `ACK_CONFLICT`. The fleet record stays on the candidate the node has abandoned. The node keeps the later evidence and retries it. Retries hash to the same id, so they cannot land. Re-enrollment creates a new incarnation and therefore a new id. That is why `TestReviewR16_InstallerRollbackMovesRecordedVersion` still passes: it revokes the incarnation and enrolls another before the rollback receipt. A node that stays on one incarnation cannot correct the receipt. An ineligible release is still refused on the first receipt (What held §3); this finding is the stuck projection after a legitimate first receipt.
- **Proof:** `TestReviewR17_LaterInstallerOutcomeCanCorrectTheRecordedRelease` in `internal/fleetupdate/review_r17_installer_integration_test.go`. Isolated databases `fvpn_t6ae5da6c_d2a432fa02894495a69469cf7a7055ad` and `fvpn_t6ae5da6c_4733f90686da4245a86c0705cc86cd5e`.

  ```
  TODO(review): later eligible installer success after a pre-health seed was not recorded (ack={Accepted:false AcceptedAsHistory:false AppliedToCurrentObservation:false ObservedUpdateGeneration:0 ReceivedAt:0001-01-01 00:00:00 +0000 UTC ResultId:00000000-0000-0000-0000-000000000000} err=ACK_CONFLICT: Installer job already has different evidence current=01a191dd-388b-70d9-b4ee-4cc1df2b2410 last_good=01a191dd-388b-70d9-b4ee-4cc1df2b2410 want 01a191dd-3894-7c30-9712-2485d6ee53d0); D-110's one immutable receipt per release, manifest and incarnation leaves the fleet on the failed baseline
  TODO(review): later eligible installer rollback on the same incarnation was not recorded (ack={Accepted:false AcceptedAsHistory:false AppliedToCurrentObservation:false ObservedUpdateGeneration:0 ReceivedAt:0001-01-01 00:00:00 +0000 UTC ResultId:00000000-0000-0000-0000-000000000000} err=ACK_CONFLICT: Installer job already has different evidence current=01a191dd-3a0b-7360-a191-7682e841a7f6 last_good=01a191dd-3a0b-7360-a191-7682e841a7f6 want 01a191dd-3a06-7723-a840-10d195775e9b)
  ```
- **Fix:** fleet-update. A later eligible outcome for the same derived job has to move `current_release_id` and `last_good_release_id` when the new evidence is the correction the node is running: success after a pre-health seed, or `rolled_back` after success, with the same eligibility and rollback checks the first insert already applies. The early return at `installer_results.go:36-38` is what blocks both. The updater should keep treating a bare `ACK_CONFLICT` as retained evidence; this id is the correct id.

## What held

### 1. The hand-off is not a second ledger for the retired stream

`TestReviewR17_HandoffDoesNotRewriteOrExtendTheRetiredStream`, `TestReviewR17_OlderRetiredTailUsesThatOriginsWindow`, `TestReviewR17_HandoffOriginIdentityAndIdempotency`, `TestReviewR17_HandoffBoundIsTheLiveServiceConfig`, `TestReviewR17_HandoffRejectionsAreAuditedAndBounded`, `TestReviewR17_UsageOriginGateIsOperationScoped`, and `TestReviewR17_SamplingIntervalBoundConfig` passed.

`selectBatchStream` requires the batch node and presenting incarnation to match mTLS, then requires the presenter to be the node’s current active incarnation (`internal/usage/handoff.go:37-46`). An origin is eligible only when it is a retired incarnation of that same node, the presenter was created after that retirement, and the node is not revoked or retired (`internal/usage/handoff.go:48-62`). The stream identity becomes the origin, and `interval_end` must be at or before `revoked_at` plus the service bound (`internal/usage/ingest.go:199-209`). `After` is strict, so the equal endpoint is accepted and one nanosecond past it is `rejected` with reason `retired_window`. Postgres timestamps are microseconds; the proof’s one-nanosecond step is still past the stored retirement. The same sequence and hash is `duplicate` and adds no `usage_samples` row. A different hash at that sequence is per-event `conflict` and also adds no sample. An origin equal to the presenter, and a presenter that omits the origin while authenticating as the retired incarnation, are rejected (`STATE_CONFLICT` / `origin_ineligible`, and `INCARNATION_MISMATCH`).

Two retired incarnations each keep their own window. The newest presenter can deliver the older incarnation’s tail inside that older retirement plus the bound, including after an intermediate incarnation was current. The intermediate retirement does not widen the older window. The intermediate tail uses the intermediate `revoked_at`.

The same idempotency key with a different origin is HTTP 409 `IDEMPOTENCY_CONFLICT` and does not add a sample. `LoadUsage` rejects an explicit 0, a negative, null, and `9223372037` (the value does not fit a duration) on `sampling_interval_bound_seconds` (`internal/config/usage.go:105-109`). `2147483648` fits and is accepted; the documented rule is any positive integer that fits a duration (`docs/usage/WIRING.md`). `NewService` rejects a negative bound and treats 0 as the 10s default (`internal/usage/service.go:97-103`). The bound is the live service value. A second service constructed at 30s accepts an event the 10s service rejected. The node does not choose that value.

Each rejected hand-off event writes one `usage.ingest_rejected` audit carrying the request id and only `error_code`, `reason`, `event_seq`, `presenting_incarnation_id`, and `origin_incarnation_id` (`internal/usage/handoff.go:79-110`). The rejected batch wrote no samples. The audit transaction commits on the batch-level reject path even when the command does not.

D-115 is operation-scoped. `usageOriginSelected` is true only when the top-level `origin_incarnation_id` unmarshals to a UUID (`internal/httpapi/middleware.go:821-826`). Null, absent, a nested field, and a non-UUID do not select a hand-off. A non-UUID usage body is HTTP 400 before the usage service. Null fails schema validation at 422. An absent origin with a presenting-identity mismatch is 403, and the probe is not reached. A lease renewal body that carries `origin_incarnation_id` is not exempt: the mismatch check at `middleware.go:689` applies the exemption only when the operation id is `SubmitUsageBatch`.

### 2. The node ledger does not remint a retired tail

`TestReviewR17_UsageLedgerDoesNotRemint`, `TestReviewR17_TwoUsageStoresDoNotDoubleCommit`, `TestReviewR17_RetiredUploadReplayKeepsTheOriginACK`, `TestReviewR17_MeterIntervalIsNotRuntimeCapped`, `TestReviewR17_FinalSampleDoesNotInventAnEpoch`, and `TestReviewR17_OpenEpochDoesNotPublishCheckpoint` passed.

Opening the same `usage.db` as another incarnation with a zero `RetirePreviousAt` returns `ErrIdentity`. Opening it as another node id returns `ErrIdentity` even when a retirement time is supplied. A non-zero `RetirePreviousAt`, including a clock stepped 24h backward, copies the previous stream into the retired table and starts the new stream at sequence 1 with `AckedThrough` 0. The retired events stay at their original sequence. Reusing the retired incarnation id returns `ErrIdentity`. Production `ensureUsage` always passes `RetirePreviousAt: e.clock.Wall()` (`node/agent/usage.go:132`).

Two `Open` calls on one path are not an exclusive directory lock. Concurrent `Commit` of the same new epoch lets one writer succeed. A lost HTTP response is not stored as an ACK. The next attempt posts the same idempotency key and the same body. The 200 whose `node_incarnation_id` is the origin advances the retired stream’s `AckedThrough` and leaves the current stream at 0. A 422 leaves the un-ACKed tail in place. A 409 `STATE_CONFLICT` sets the retired stream’s final code and a later `Once` does not upload it again.

`meter.New` keeps any positive `Interval` and uses `DefaultSampleInterval` (10s) when the interval is not positive. Production `meter.New` omits `Interval` (`node/agent/usage.go:154-156`), so the running agent uses the 10s default. There is no runtime comparison with the central bound inside `meter.New`. A second `FinalSample` after the epoch is closed returns `ErrNoEpoch` and the checkpoint’s closed epoch is the epoch that was open. An engine that has not closed an epoch does not publish `meter.checkpoint`, and `FileMeter.FinalSample` rejects that absence.

### 3. A node cannot choose another job’s id, and an ineligible release cannot become the baseline

`TestReviewR17_InstallerJobIDIsNotAChosenAlias`, `TestReviewR17_InstallerRollbackAndSeedRefuseIneligibleRelease`, and `TestReviewR17_LegacyJobIDDoesNotBlockTheDerivedID` passed. `TestReviewR16_InstallerRollbackMovesRecordedVersion`, `TestReviewR16_InstallerResultCardinalityIsBounded`, and `TestReviewR16_InstallerCannotSelfPromote` passed in the same `fleetupdate` package run. `TestReviewR16_InstallerResultReplayReturnsStoredAck` passed.

A body whose `job_id` is an open central job is `409 STATE_CONFLICT` without reason `job_id_mismatch`, and no installer row is inserted. The updater therefore does not retire that delivery as `OutboxRejected`. A random UUID is `409 STATE_CONFLICT` with reason `job_id_mismatch` and is not inserted. A forged manifest digest that is then run through `installerjob.ID` is `422 HASH_MISMATCH`. An exact replay returns the stored ack. The same job id with a different `result_id` is `ACK_CONFLICT` and the version columns stay on the accepted release. Two incarnations of the same release derive different ids. The primary key is `(node_id, job_id)`, so one node’s id does not occupy another node’s row.

Rollback onto a revoked release is `422 MINIMUM_VERSION_VIOLATION` and writes no installer row; the recorded version stays the previous eligible release. A D-111 seed whose baseline is revoked is rejected and both version columns stay null. A pre-D-110 row inserted under a different job id does not block the derived id. The derived success is accepted, the version columns move to the candidate, and the latest row is the derived job. That is two historical rows for one install, which is the legacy-receipt case. It is distinct from M-1, where the second outcome uses the same derived id.

r16 M-2’s version write for `rolled_back` holds once the rollback has its own incarnation. r16 L-1’s cardinality bound holds: a ninth random job id is rejected. Those two are not refiled. M-1 is the hole that remains when the incarnation does not change.

### 4. A fenced device is absent from the signed state and the lease device list, and a withdrawn grant stays withdrawn

`TestReviewR16_FencedDeviceReceivesNoGrant` and `TestReviewR17_FencedDeviceIsAbsentFromSignedState` passed. `TestReviewR16_StateCacheDropsGenerationOnlyAdvance` and `TestReviewR16_GenerationAdvanceAfterCacheIsServed` passed. `TestReviewR17_RevokedGrantIsNotResurrected`, `TestReviewR17_Year9999GrantFailsClosed`, and `TestReviewR17_NoGrantRevocationIsCleared` passed.

The binding query sets `is_revoked` when `member_devices.status` is not `authorized`, when `revoked_at` is set, or when a `device_authorization_fences` row exists (`internal/desiredstate/interfaces.go:485-486`). The signed envelope contains the open device id and does not contain the fenced device id. `member_devices.status` stays `authorized`. `ListLeaseEntitlementsForGroups` excludes a fenced device (`internal/leases/leases.go:168`). The NetBird sync sets `allow=false` while a fence row exists (`internal/devices/worker.go:97-98`). `PublicStatus` reports `pending` while an authorized device’s read-back is not current (`internal/devices/service.go:70-76`), and `AuthorizationUnconfirmed` is the fence flag (`internal/devices/service.go:64-65`). That public status is the D-077 device view. The grant readers that r16 H-1 named now follow the fence. The generation-only state-cache advance and the keyed installer-result replay that r16 reconfirmed now pass; #277 is present on this tree.

`RevokeMemberGrant` locks the grant, rejects an already-revoked or already-expired row, and sets `revoked_at` (`internal/members/service.go:527-533`). Replaying the same idempotency key returns the same operation. The same key on another member’s grant withdraws that grant. A later `Grant` inserts a new row and leaves the withdrawn row revoked. `ValidPeriod` and `Compute` reject a grant whose end is year 9999 under the 366-day `MaxGrantDuration`; `Time.Sub` saturates and still fails the positive maximum. Production Go under `internal/entitlement` and `internal/members` contains no assignment that clears `revoked_at`.

### 5. A crafted checkpoint does not seal the barrier, and an open epoch does not publish one

`TestReviewR17_CraftedCheckpointCannotSealTheBarrier`, `TestReviewR17_UpdaterDoesNotWriteEtcVpnNode`, `TestReviewR17_ReleaseIdentityFollowsTheRunningBinaryAndJournal`, and `TestReviewR17_OpenEpochDoesNotPublishCheckpoint` passed.

`FileMeter.FinalSample` opens with `O_NOFOLLOW|O_NONBLOCK`, requires a regular file, and rejects a checkpoint whose sequence fields are not decimal strings, whose new epoch is non-empty, or whose central ack runs ahead of the durable sequence. `waitMeter` still returns `meter_checkpoint_uncorrelated` when `FinalSample` succeeds (`node/updater/meter_barrier.go:187-192`). A stale, future, duplicate, or well-formed checkpoint does not complete the barrier. Fresh success is blocked on the agent-core 26 correlation field. That refusal is fail-closed and is not filed as a regression of this revision.

`loadActiveRelease` requires the running executable to be the `fvpn-agent` in the evaluated current-release directory and the journal current release to equal that id. A candidate tree that is not the running binary does not change `ReleaseID`. `matchingChallenge` requires the challenge release id to equal the configured release UUID. The updater’s production sources name `/etc/vpn-node` only in `layout.go`, `probe_addresses.go`, and `host.go`. `probe_addresses.go` does not open that path for write. `Layout.Ensure` creates the parent directory of `Trust` (`node/updater/layout.go:57`, `layout.go:65`), which is `/etc/vpn-node`, and does not write `release-trust.json`. `DefaultLayout().Trust` is `/etc/vpn-node/release-trust.json`.

### 6. Deadlines use a 1 ms quantum and the 100 ms install guard

`TestReviewR17_DeadlineUsesMillisecondQuantumAndInstallGuard` passed.

`memberTimeout` takes the earliest of the binding end and the forwarding deadline, then the monotonic deadline when `DeadlineMono` is set, then subtracts `timeoutReserve` and truncates to `DeadlineGranularity` (`node/enforce/deadline.go:51-61`). A 1500 ms wall deadline is returned as 1500 ms. With the reserve set to `DeadlineInstallBound` (100 ms), a 2 s deadline yields 1900 ms, a deadline 101 ms out yields 1 ms, and a deadline equal to the guard is not admitted. Stepping `Model.Now` an hour backward while `NowMono` and `DeadlineMono` stay one second apart yields `1s - 100ms`. The monotonic cap is what the renderer uses. `Apply` bounds the install context by the same 100 ms (`node/enforce/apply.go:171`).

### 7. The deciding tc read survives the cap, and central stores the digest the node sent

`TestReviewR17_DecidingTCReadSurvivesTheCap` and `TestReviewR17_CentralStoresTCEvidenceWithoutRehashing` passed.

`retainTC` sorts nonempty and error reads ahead of empty ones. When the check carries a deciding attachment, that row is bounded to one object and four actions and placed first, before the 16-read cap (`node/enforce/evidence.go:144-166`). An interface name outside `^[a-zA-Z0-9_:-]{1,32}$`, a `sk_` token, and a fifth action are absent from the wire JSON. The SHA-256 is `canonjson.HashHex` of the projection `{reads, objects, truncated, tc}` (`node/enforce/evidence.go:173-178`). Two independent retains of the same reads at generations 7 and 8 produce the same digest; the generation string is retained beside that digest. `SubmitObservation` stores `report_payload` and keys idempotency on the whole report. The same `report_id` with a different `tc_evidence.sha256` is `ACK_CONFLICT` and the row count stays 1. A new `report_id` that copies the first digest is stored as its own row. Central does not recompute the digest.

### 8. A dirty generation is not acknowledged, and a failed freeze persist stays latched

`TestReviewR17_DirtyGenerationIsNotAcknowledged`, `TestReviewR17_RetainedApplyErrorStillRenewsTheDesiredGeneration`, and `TestReviewR17_FreezeIntentAndMetaBothFailStaysLatched` passed. `TestReviewR16_FailedFreezePersistIsLostOnRestart` and `TestReviewR16_FrozenIncarnationDoesNotAckOrKeepServing` passed.

`applyError` is cleared only after serving verification has succeeded (`node/agent/engine.go:1032`), and `noteApply` sets it (`node/agent/engine.go:1124-1141`). `ackApplied` returns while `applyError != nil` (`node/agent/engine.go:1109-1111`). `needRenew` is true when an apply error is retained and the lease generation differs from the desired generation (`node/agent/sync.go:143-145`, `leaseBehindDesired` at `164-167`). A renewal can be requested for the desired generation while the applied ACK stays blocked.

`persistFreeze` writes `identity-freeze.intent` and then `identity_frozen` meta (`node/agent/freeze.go:42-51`). `freezeIdentity` retries the durable write when the memory latch is already set (`node/agent/freeze.go:17-19`). `loadFreeze` treats a matching intent file as frozen and calls `persistFreeze` again (`node/agent/freeze.go:60-77`). With the SQLite writer held, the in-process latch is set and `storageOK` is false. After restart the incarnation is still frozen and the meta value is `"1"`. A second failure mode, an enqueued observation whose attempt was bumped and a state directory that then rejects the write, also restarts frozen.

### 9. `inet fvpn_install` is not a special fail-closed lever, and the legacy delete stays on `inet fvpn owner`

`TestReviewR17_InstallerMarkerIsNotASpecialForeignLever`, `TestReviewR17_LegacyMarkerDeleteCannotBeRedirected`, and `TestReviewR17_HookedInstallTableIsRefusedBeforeDelete` passed.

`verifyForeign` fails closed for a base chain outside `inet fvpn` whose hook can see forwarded member traffic. `ip`, `ip6`, `bridge`, and `netdev` forward hooks are in that set. An empty hook and family `arp` are not. A hookless `inet fvpn_install` table is ignored, the same way any hookless foreign table is ignored. An accept-only empty chain that already has a priority is tolerated on the same rule as any other foreign chain. A hooked install table with policy drop is refused before any delete. The legacy marker parser recognizes an empty `inet` chain named `owner` with the installer comment. Extra keys, rules, or a non-`inet` family do not redirect the delete. The delete argv is `nft delete chain inet <NFTTable> owner`, and the default table name is `fvpn`.

### 10. Revision 14 codes have consumers, and the new detail reasons stay out of the error-code enum

`TestReviewR17_Revision14CodesHaveConsumers` passed.

Portal `ERROR_CODE_MESSAGES` contains `STATE_CONFLICT`, `ACK_CONFLICT`, `USAGE_EVENT_CONFLICT`, `LEDGER_EPOCH_MISMATCH`, `INCARNATION_MISMATCH`, `TC_FOREIGN_FILTER`, and `VALIDATION_FAILED` as code keys (`apps/portal/src/api/errors.ts`). It does not contain `job_id_mismatch` or `origin_ineligible`. `pkg/api` has no `JOB_ID_MISMATCH` error code. Those two strings are detail reasons on `STATE_CONFLICT`. The worker handles `member.grant_revoked` (`apps/worker/jobs_members.go`). The entitlement command handles `member.grant_revoke` (`internal/entitlement/command.go`). The updater compares `ack.Reason` to `job_id_mismatch` (`node/updater/result.go:237`).

### r16 dispositions re-run on this tree

| Finding | This run |
|---|---|
| H-1 fenced device still bound and leased | `TestReviewR16_FencedDeviceReceivesNoGrant` passed. The fence is part of `is_revoked` and of the lease device query. |
| M-1 failed identity-freeze persist | `TestReviewR16_FailedFreezePersistIsLostOnRestart` passed. The intent file recovers the latch. See What held §8. |
| M-2 installer rollback version write | `TestReviewR16_InstallerRollbackMovesRecordedVersion` passed after the fixture re-enrolls. Same-incarnation correction is M-1 above. |
| L-1 installer row cardinality | `TestReviewR16_InstallerResultCardinalityIsBounded` passed. A random ninth job id is rejected. |
| D-099 generation-only cache | `TestReviewR16_StateCacheDropsGenerationOnlyAdvance` and `TestReviewR16_GenerationAdvanceAfterCacheIsServed` passed. |
| Keyed installer replay 500 | `TestReviewR16_InstallerResultReplayReturnsStoredAck` passed. |

## Not examined

- Live NetBird group propagation timing after a fence. The sync sets `allow=false` when the fence row exists; this review did not run that worker against a management API.
- DNS documents as a projection separate from the signed desired-state envelope. The envelope proof shows the fenced device id is absent from the signed JSON.
- `ReplaceFiniteGrant` in `tests/e2e/grant_fixture.go`, and the ordering of `device.authorized` against `member.grant_revoked` inside one worker batch. The product `Grant` path inserts a new row and does not clear `revoked_at`.
- A full G2 dataplane. `TestReviewR17_DirtyGenerationIsNotAcknowledged` is the in-process negative control. No netlab was started.
- Suspend and resume of `CLOCK_MONOTONIC`, and a publisher that flaps generations across repeated guarded installs. The unit proof covers the quantum, the guard, and a wall-clock step against `DeadlineMono`.
- The updater completing an update. `waitMeter` refuses every current-format checkpoint as `meter_checkpoint_uncorrelated`. Acceptance of a correlated checkpoint is agent-core 26.
- Portal rendering beyond the `errors.ts` key scan. Handover register sentences in `tests/handover/handover_test.go` still describe activation as pending; that file was not edited and the drift is not filed.
- Whether an operator-sized audit table grows under a usage-rate flood of rejected batches. One batch of three rejected events wrote three bounded audits and no samples. The usage rate limiter was not driven.
- Host networking, shared `fvpn-pg`, and any Docker object. None were used.
