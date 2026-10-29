# Central update acceptance evidence

Run from the repository root:

```sh
make -f mk/fleetupdate.mk fleetupdate-verify
make go-lint
```

Integration tests require `FVPN_TEST_DATABASE_URL` for PostgreSQL 17. Each uses
`internal/db/dbtest`, creates an isolated database and drops it on cleanup.
The Make targets use private on-disk scratch under `.agent/tmp` and remove it.
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
