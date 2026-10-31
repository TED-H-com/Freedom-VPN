# Central update acceptance evidence

Run from the repository root:

```sh
make -f mk/fleetupdate.mk fleetupdate-verify
make go-lint
```

Integration tests require `FVPN_TEST_DATABASE_URL` for PostgreSQL 17. Each uses
`internal/db/dbtest`, creates an isolated database and drops it on cleanup.
The Make targets use on-disk scratch under `.agent/tmp`; Go removes its temporary
build/test directories on normal completion.
Ordinary `go test ./...` requires neither PostgreSQL nor network/root/Docker.
No real node or host networking is changed.

| Case | Central executable evidence | Remaining external evidence |
|---|---|---|
| AT23 / UPD01, UPD02 | `TestAT23_AllStatesArchitecturesAndEvidence`: both architectures; published/unpublished/disabled/enrolling; revoked/retired excluded; matching artifact; missing health rejected; success counted only after result | Real amd64/arm64 installation and tunnel probes |
| AT24 / UPD01 | `TestAT24_OfflineEnrollmentAndFirstSync`: offline catch-up; bind once; immutable null snapshot; after-snapshot one-node system batch; no job before binding; reinstall isolation. `TestAT24_UnboundSupersessionAndCancellation` and `TestAT24_FirstSyncSnapshotGuards` cover unbound counts/cancellation and rejected system snapshots | Physical reconnect and installer enrollment |
| AT25 / UPD02, UPD03, UPD05 | `TestUPD02_ManifestValidation`, `TestUPD02_RegisterImmutableAndVerify`, `TestUPD02_RejectedRegistrationsPersistIneligible`: signatures, origins, architecture completeness, immutable content and unique product version (D-013), length and digest verification | Disk exhaustion, crash/fsync ordering, journal recovery, migrations and usage continuity belong to updater/netlab |
| AT26 / API01, UPD02 | `TestAT26_IdempotencyConcurrencyAndLateResults`: six concurrent identical commands; conflict body; supersede queued while retaining applying; original ACK replay; result conflict evidence; no pointer regression | Node exclusive lock and power-loss recovery |
| AT27 / UPD03–UPD05, EXIT03 | `TestAT27_ServingIndependentOfDesired`, `TestAT27_ObservedVersionRollbackAndDisabled`: supported old serving; minimum policy; valid/refused rollback; revoked last-good recovery; heartbeat outranks delayed result. AT23 also asserts disabled unchanged after success | Real forwarding enforcement and rollback flow |
| §13.12 counting | `TestUPD05_CountingInvariantRandomSequences`: 10,000 random changes over 100 targets; every bucket sum equals snapshot size. Integration covers attempts, cancellation, pagination and operation completion | Portal rendering |
| §13.8/§17.4 | `TestUPD02_ResultBindingExpiryAndRestore`: identity, incarnation, target, attempt, generation, digest, health, usage checkpoint, expiry, backward progress and restored-generation protection | Reconcile restored job IDs and on-device journals |
| §15.5.6 | `TestUPD05_RetryCancellationWatchdogAndAtomicity`: injected outbox failure rolls back snapshot transaction; retry denominator and attention-only watchdog | Production latency/capacity measurement |
| §16.2 | `TestSEC01_FetcherRejectsNonpublicAddresses`: local/private/link-local/metadata/documentation/NAT64 rejected; validator checks exact origins | Controlled storage TLS/DNS deployment |

CI runs all central unit/integration tests, pinned staticcheck, vet and cross-builds
both linux/amd64 and linux/arm64. Cross-compilation is not evidence of a real
arm64 device run. All fixture keys are generated at test time; the signed manifest
shape derives from `contracts/vectors/release-manifest-signing.json` (TEST ONLY).

Migration 00160 retains the base unique release version, non-null job identities,
composite target FK, phase guards and frozen terminal evidence. The database
owner's schema/review integration suite passes with this migration applied.
`TestUPD02_JobReferencesAndForwardPhases` also proves unknown job references fail
and late or invalid phase reports cannot rewrite current terminal evidence.

The database helper now reparses isolated database DSNs; the earlier CLI-test
workaround is no longer needed. All domain tests continue to create and drop
only their own databases.

Review regression coverage:

| Finding | Executable evidence |
|---|---|
| 12.1 | `TestAT23_ResultAfterRoutineHeartbeat`: older heartbeat permits result advancement; same/newer observed generations stay authoritative |
| 12.2–3 | `TestAT24_IntentRefreshAndTargetNames`: returned/stored timestamps match exactly; named and unnamed nodes are distinguished |
| 15.2 | `TestUPD02_RegistrationValidationBeforeTypedDecode`: oversize request stops before database work; invalid signature precedes typed uint64 overflow |
| 15.3 | `TestUPD02_FailedRegistrationAndOperationEvidence`: pending reuse, failed re-registration conflict and original-command replay |
| 4 / D-029 | `releasetest.AssertLedger`: every operation reads through `operations.Store.Get`; audit pages include all persisted events with zero anomalies; verification failure has a safe step and top-level error |
| Applied policy generation | `TestAT26_AppliedPolicyEvidenceIsImmutable`: exact large decimal and matching policy refs, conflict handling, history cannot rewrite accepted evidence |

## Installer-applied results (D-089)

`TestD089_InstallerAcceptReplayAndIsolation` validates the canonical installer
example against OpenAPI, accepts both architecture fixtures, checks exact and
concurrent ACK replay, conflicting content, immutable rows, audit provenance,
node release/summary projection and unchanged central-table rows/counts.
`TestD089_InstallerRejections` covers identity/incarnation, publication,
central-issued IDs, source-less unknown jobs, hash and terminal-evidence fences.
`TestD089_InstallerDoesNotCompleteQueuedBatch` preserves queued work and proves
explicit `source: central` still completes its authorized job.
`TestD089_InstallerPreHealthBaseline` checks the unchanged-release failure rule.
`TestD089_InstallerAcceptanceClosesHeartbeatSeeding` fences later heartbeat seeding
even after an accepted failure on a node whose current version remains null.
`TestD089_InstallerAdminNodeHTTP` exercises admin node detail and list responses.
`TestD089_InstallerFailuresAndTransaction` proves failure preserves node
versions, rollback records the restored release, already-current establishes last-good, and caller rollback undoes both
receipt and projection. `TestD097_InstallerProjectionCentralPrecedenceAndIncarnation`
proves central-job precedence, unchanged confirmed batch evidence, latest
accepted source selection, and rejection/old-incarnation exclusion. These are
central integration proofs, not physical installer or real ARM64 runs.

## D-127 correction coverage (2026-10-31)

The unchanged `TestReviewR17_LaterInstallerOutcomeCanCorrectTheRecordedRelease`
now passes alongside the other copied r17 installer proofs and all r16 proofs.
`TestD127_SuccessAfterSeedMovesBothVersions` and
`TestD127_RollbackAfterSuccessMovesBothVersions` cover both architecture identities,
immutable history, node detail/summary and ETags. The remaining `TestD127_*`
tests cover both exact stored ACKs (including the real HTTP keyed replay reader),
concurrent delivery, third flips, changed same-outcome evidence, other disallowed
transitions, ineligible corrections, unchanged rollback/health checks, nine
random job IDs, the central generation high-water, and migration 00192's
round-trip, row bound, immutable guards and lossless downgrade refusal.
See [D-127 lifecycle and devstack hand-off](central-lifecycle.md#installer-outcome-corrections-d-127-2026-10-31).

## Live installer central acceptance — updates follow-up 10 (2026-10-31)

Updates follow-up 10, merged as #337, ran the installer → updater → central
extension on product main `fb5e002549f79e8ef61bebcf42cce1678ab65cb1`, harness
`0df8599dc84d4b85e48182f20aa4b0fc741bc72a`, run
`695f977c-b587-4768-88f1-0514f3a4f1ac` at `2026-10-31T15:27:22Z`.
The [JSON record](../e2e/at/updates/installer-handoff.json),
[rendered hand-off](../e2e/at/updates/installer-handoff.md) and
[follow-up report](../e2e/at/updates/follow-up-10.md) on main record all 23
installer extension checks passing. These living links identify the cited run;
no digest is pinned to a living evidence file. This is measured central
acceptance for UPD02, UPD03 and UPD05 (§13.6, §13.8, §13.9).

| Stimulus | Live central result |
|---|---|
| Real sealed installer success after sequence-11 apply | HTTP 202; `source=installer`, `batch_id=null`, current = last-good = sequence 11 from a null central baseline. The D-110 job ID was recomputed and matched the receipt. |
| Same-key and new-key exact receipt replays | HTTP 202 / 202 with stored ACKs; ETag remains `"8"`. |
| Modified receipt; wrong job ID | HTTP 409 `ACK_CONFLICT`; HTTP 409 `STATE_CONFLICT` with `details.reason=job_id_mismatch`. Release pointers, update summary and version stay unchanged at ETag `"8"`. |
| Failed sequence-12 apply after real checkpoint withdrawal | `CHECKPOINT_FAILED` at `final_sample`, null checkpoint and health; failed evidence accepted HTTP 202. Current = last-good stays sequence 11; the failure summary advances the ETag to `"9"`. |
| D-127 / r17 M-2 later rollback correction | **TEST ONLY HTTP stimulus** derived from the real success; accepted HTTP 202. Central current = last-good returns to sequence 10, state becomes `rolled_back`, ETag `"9"` → `"10"`. |
| First-receipt and correction replays; third changed outcome | HTTP 202 / 202 with both stored ACKs retained; the correction also replays under a new key with HTTP 202. A third changed outcome returns HTTP 409 `ACK_CONFLICT`; correction ETag stays `"10"`. Two outcomes remain stored for the success job. |

The source report labels the correction “D-127 / r16 M-2”; the same measured
correction supplies the r17 M-2 central acceptance evidence recorded here.
The correction did not perform a local rollback or measure restored baseline
health. **Physical rollback (AT26) and the regular AT23–AT27 rows remain
blocked** by updates F10-01, the devstack fixture layout item: the regular
builder places the executable under `bin/`, while active-release validation
expects it at the release root. AT23 cannot establish last-good, so AT24–AT27
cannot execute their later scenarios. The full live make target exits 2.

The extension used a quiet unpublished amd64 agent and an explicitly configured
central endpoint, CA and allowed origins. Generated installer reporting remains
F10-02 because its `result_url` is empty. Serving-traffic withdrawal/nonzero
accounting is separately cited by updates 10 from devstack 19; physical arm64
execution remains open. These results claim no live production rollout.

## Fixture ownership and evidence paths (D-132, 2026-10-31)

`TestD062a_PrecheckFailureHarnessDelivery` and
`TestD062c_RollbackHarnessRequiresRestoredHealthEmptyAllowlist` now read
byte-identical redacted captures in `internal/fleetupdate/testdata/`. Each test
comments its immutable follow-up 9 source and SHA-256 digest. State, stage,
error and health assertions are retained, with IDs, hashes and times rebound
to real authorized jobs. An empty precheck record now fails explicitly before
indexing. `TestD132_EvidenceReferencesUseImmutableHistory` scans every file
under `internal/fleetupdate/` during ordinary tests and rejects AT evidence
references outside a `history/` path, including comments and fixture files.

The recursive audit of fleetupdate and installerjob tests found no remaining
cross-lane AT file reads. Shared contract example reads remain:
`ObservationReport.json` in `d062_integration_test.go` and
`review_r9_d062_integration_test.go`, and `UpdateResult.installer.json` in
`installer_results_integration_test.go`. The shared `releasetest.New` helper
reads the canonical release-manifest signing vector in `contracts/vectors/`.
Installerjob creates its pending record under `t.TempDir()` and exercises the
updater against that isolated tree. These contract and local scratch inputs
do not depend on another lane's evolving evidence files.
