# Adversarial Review `r6-d053-review`: Challenge-Bound Update Health Attestation (`D-053`)

- **Branch:** `r6-d053-review` (commit `307deb7`)
- **Date:** 2026-10-29 (`2026-10-29 22:38:49 -0400`)
- **Reviewing model:** `antigravity/gemini` (recorded in commit `307deb7` `Agent:` trailer; not stated in `.agent/REVIEW.md` body)
- **Decision / scope reviewed:** `D-053` challenge-bound update health attestation (`§13.6`, `§13.9`, `UPD03`, `UPD04`, `UPD05`, `AT23`, `AT25`, `AT27`, `SEC02`; `node/internal/healthattest`, `node/updater`, `node/agent`)
- **Adopted proof files on `main`:**
  - [`node/internal/healthattest/review_d053_test.go`](../../node/internal/healthattest/review_d053_test.go) (adopted in `#96`)
  - [`node/updater/review_d053_test.go`](../../node/updater/review_d053_test.go) (adopted in `#96`)
  - [`node/agent/review_d053_test.go`](../../node/agent/review_d053_test.go) (adopted in `#99`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | HIGH-1: Revocation rollback (`restoreRelease` and `blockRevoked`) in `node/updater/revoke.go` never issues or consumes a `D-053` health challenge and reports a false `previous_release_id = target` when the rollback check fails. | closed | #96 |
| High | HIGH-2: Health-attestation file I/O in `node/internal/healthattest/io.go` accepts symlinked and group/world-writable parent journal directories, and `ObserveAttestation` omits the leaf `0600` mode check. | closed | #96 |
| Medium | MED-1: Crash in `stepRollback` in `node/updater/apply.go` after `SetPointers(LastGood: target)` and before `advance(StageRolledBack)` causes `Recover` on the final rollback attempt to enter `recovery_required` (`ROLLBACK_FAILED`) while `last_good` is healthy and running. | closed | #96 |
| Medium | MED-2: Crash in `stepCheck` in `node/updater/apply.go` after `SetPointers(LastGood)` and `SetStateSchema` (before `advance(StageLocallySucceeded)`) mutates `state_schema` to the candidate's version and causes post-crash rollback recovery to fail with `SCHEMA_INCOMPATIBLE`. | closed | #96 |
| Medium | MED-3: `NodeChecks.poll` in `node/updater/attest.go` aborts the 30s wait window on the first bound attestation even if checks are still warming up, and `node/agent/engine.go` and `node/agent/sync.go` prematurely attest `dns_policy: failed` on an expired ETag `304` or first-step transport error. | closed | #96, #99 |
| Medium | MED-4: `healthattest.Validate` in `node/internal/healthattest/validate.go` accepts `GeneratedAt` timestamps in the future of `now`, and `NodeChecks.poll` in `node/updater/attest.go` uses wall-clock `time.Now()` instead of the injected `n.now()` clock. | closed | #96 |
| Low | LOW-1: `healthattest.Validate` (`checkReason`) in `node/internal/healthattest/validate.go` does not enforce a fixed reason vocabulary or reject URLs/member data before `node/updater/attest.go` copies reasons into SQLite `txn_events`. | closed | #96 |

## What Held

- **Nonce / `ReleaseID` / `TxnID` binding (`node/internal/healthattest/validate.go`):** A pre-staged `health-attest.json` from an old agent or previous update attempt with a wrong `Nonce`, wrong `ReleaseID`, wrong `TxnID`, `GeneratedAt < IssuedAt`, or `now > ExpiresAt` is rejected deterministically (`ReasonNonce`, `ReasonRelease`, `ReasonTxn`, `ReasonBeforeIssue`, `ReasonExpired`). Proven by `TestD053_Review_Held_ValidateAndFileDefenses` and `TestAG_D053_Review_Held_OldAgentDoesNotOverwriteCandidateAttestation`.
- **Leaf file symlink, hard-link, size cap, and non-0600 rejection in `ReadAttestation` / `ReadChallenge` (`node/internal/healthattest/io.go`):** When the leaf `health-attest.json` or `health-challenge.json` itself is a symlink, has `Nlink != 1`, has mode `0644` (in `ReadAttestation`), or exceeds `MaxFileBytes` (`8192`), `readRegularFile` rejects it before `json.Unmarshal`. Proven by `TestD053_Review_Held_ValidateAndFileDefenses`.
- **Truncated / partial JSON writes (`node/internal/healthattest/io.go`, `node/updater/attest.go`):** A half-written or corrupted JSON file returns `ReasonMalformed`, which `NodeChecks.poll` treats as transient (`if err.Reason != healthattest.ReasonMissing && err.Reason != healthattest.ReasonMalformed`), continuing to poll until either an atomic rename completes or the wait deadline expires (at which point the candidate rolls back). Proven by `TestUPD_D053_Review_Held_PartialWriteAndChallengeWriteFailureRollBack`.
- **Challenge write failure (`node/updater/apply.go`):** If `issueChallenge` fails (`WriteChallenge` returns an error), `stepSwitch` rolls back the `current` symlink, restores `Pointers.CurrentRelease`, restarts the old service, and advances to `StageFailed` with `SWITCH_FAILED`. Proven by `TestUPD_D053_Review_Held_PartialWriteAndChallengeWriteFailureRollBack`.
- **TTL clamping and NetBird version token matching (`node/updater/attest.go`, `node/agent/attest.go`):** Configuring `HealthChallengeTTL > MaxTTL` (`300s`) clamps `ExpiresAt` to `IssuedAt + 300s` (rather than failing `Validate`), and `extractVersionToken` in `node/agent/attest.go` strips pre-release/build suffixes (`0.68.2-rc1+build7` -> `0.68.2`) while rejecting mismatched versions (`0.68.1` != `0.68.2`) and sanitizing raw dial errors out of `tunnel_probe.reason` (`"probe_failed"`). Proven by `TestUPD_D053_Review_Held_VersionTokenAndTTLClampCannotHideMismatch` and `TestAG_D053_Review_Held_NetbirdVersionAndProbeReasonSanitization`.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### [HIGH-1] Revocation rollback (`restoreRelease`) never issues or consumes a D-053 health challenge and lies about `previous_release_id` on failure
- **Where:** `node/updater/revoke.go:238-272` (`restoreRelease`), `node/updater/revoke.go:108-130` (`blockRevoked`).
- **Scenario:** When the running `CurrentRelease` is listed in `revoked_release_ids`, `rollbackIfRevoked` calls `restoreRelease(ctx, cur, lastGood, ...)`. Unlike `stepSwitch` (`node/updater/apply.go:196`) and `stepRollback` (`node/updater/apply.go:292`), `restoreRelease` never calls `e.issueChallenge` or `e.consumeChallenge`, so `runHealth` always fails with `ReasonMissing` (`no_health_attestation`), and `blockRevoked` enqueues an `UpdateResult` with `PreviousReleaseId == target` instead of `cur`.
- **Proof:** `TestUPD_D053_Review_RevokedCurrentRollbackIssuesChallengeAndReportsTruth` (`node/updater/review_d053_test.go`).
- **Fix (#96):** `rollbackIfRevoked` records the previous release, stops the revoked service, issues and binds a fresh health challenge before starting the restored release, consumes it after the health run, stops the target on failure, and reports `previous_release_id` truthfully.

### [HIGH-2] Health-attestation file I/O accepts symlinked and group/world-writable journal directories, and `ObserveAttestation` skips the `0600` mode check
- **Where:** `node/internal/healthattest/io.go:72-95, 117-147, 194-275`.
- **Scenario:** `WriteChallenge`, `ReadChallenge`, `WriteAttestation`, `ObserveAttestation`, and `ReadAttestation` only `os.Lstat` the leaf file and never validate the parent directory `journalDir`, and `ObserveAttestation` omits the `0600` mode check on `health-attest.json`.
- **Proof:** `TestD053_Review_SymlinkedJournalDirRejected`, `TestD053_Review_SharedWritableJournalDirRejected` (`node/internal/healthattest/review_d053_test.go`).
- **Fix (#96):** One private-directory validator guards every challenge and attestation reader/writer plus layout setup, rejecting symlinked, non-directory, group/other-writable, or foreign-owned directories and enforcing mode `0600`, ownership, `os.SameFile`, and `io.LimitReader(MaxFileBytes+1)` on leaf files.

### [MED-1] & [MED-2] Crash windows in `stepRollback` and `stepCheck` before stage advance
- **Where:** `node/updater/apply.go:227-322`, `node/updater/health.go:150-155`.
- **Scenario:** Updating `LastGood` or `StateSchema` before the SQLite stage-advance transaction meant a crash between the two commits either caused `Recover` on the final rollback attempt to report `ROLLBACK_FAILED` / `recovery_required: true` while `last_good` was healthy and running (`MED-1`), or caused post-crash rollback recovery to reject `last_good` with `SCHEMA_INCOMPATIBLE` (`MED-2`).
- **Proof:** `TestUPD_D053_Review_CrashInStepRollbackAfterSetLastGoodLiesRecoveryRequired`, `TestUPD_D053_Review_CrashInStepCheckAfterSetStateSchemaBreaksRollback` (`node/updater/review_d053_test.go`).
- **Fix (#96):** Committed `LastGood` and `state_schema` inside the same SQLite transaction as the stage advance (`StageUpdate.Pointers` / `StageUpdate.StateSchema`).

### [MED-3], [MED-4], & [LOW-1] Polling warm-up, agent verified desired-state gate, future `GeneratedAt` skew, injected poll clock, and bounded check reasons
- **Proofs:** `TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires`, `TestUPD_D053_Review_PollDeadlineIgnoresInjectedClock` (`node/updater/review_d053_test.go`), `TestAG_D053_Review_ExpiredStoredDesiredETagOn304WritesFailedDNSAttestation`, `TestAG_D053_Review_TransientFirstSyncErrorPrematurelyAttestsFailedDNS` (`node/agent/review_d053_test.go`), `TestD053_Review_ValidateRejectsFutureGeneratedAtClockSkew`, `TestD053_Review_ValidateRejectsFreeFormReasonAndURLs` (`node/internal/healthattest/review_d053_test.go`).
- **Fix (#96, #99):** `NodeChecks.poll` uses the injected clock and keeps polling until all attested checks pass or the deadline expires; `healthattest.Validate` rejects `GeneratedAt` in the future beyond a bounded skew and enforces a fixed reason vocabulary; `node/agent` stores `desiredETag` only after signature and validity-window verification and writes a health answer only once a verified desired state is held.
