# R30 Review: Update Contention, Restart Adoption, and Closed-Recovery Renewal (`D-155`–`D-158`)

- **Branch:** `r30-update-fixes-review`
- **Base commit (`main`):** `9b9546c9502fe21239dde26b156f38dfbb0c1a58`
- **Reviewed PRs:**
  - `#446` (`439dbc0b`, `updater`, `D-155` updater side, `docs/updates/node-followup24-2026-11-03.md`)
  - `#454` (`65099264`, `agent-core`, `D-155` agent side + `D-156`, `docs/node/followup49-2026-11-03.md`)
  - `#461` (`3f39c1cc`, `agent-core`, `D-157`, `docs/node/followup50-2026-11-04.md`)
  - `#463` (`d733b352`, `updater`, `D-158`, `docs/updates/node-followup25-2026-11-04.md`)

---

## Fidelity

| Decision and clause | Code (`file:line`) | Status | Proof |
|---|---|---|---|
| **D-155.1** `fvpn-updater` defines `ExitBusy = 3` and exits `3` only when the error is `*Error` with `Code == apierr.UpdateBusy` and `Detail == "lock"` (live holder); `ErrMembersBlocked` (`2`) takes precedence in joined errors; stale holder metadata (`lock boot id`, `lock pid`, `lock process`, `lock start time`) and non-`lock` busy details remain exit `1`. | `node/updater/errors.go:18-48`, `node/updater/cmd/fvpn-updater/main.go:34-43` | **implemented** | `TestD155_ExitCode`, `TestD155_RecoverExitCodes`, `TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog` |
| **D-155.2** All six `fvpn-updater` entry points (`Recover`, `Run`, `Flush`, `Retain`, `ApplyPending`, `Seed`) acquire `update.lock` before opening or mutating the update journal (`withLockFor` wraps `withJournal`). | `node/updater/engine.go:65-75,92-99,115-120,162-167,201-206,389-424` | **implemented** | `TestD155_AllEntryPointsLockBeforeJournal` |
| **D-155.3** `fvpn-node-install` treats `fvpn-updater` exit code `3` as `UPDATE_BUSY` (retryable / busy, distinct from corrupt state) across `install`, `upgrade`, `reenroll`, `remove`, `check`, and `baseline`. | `packaging/installer/lock.go:70-124`, `packaging/installer/baseline.go:108-123` | **implemented** | `TestD155_ExitCode`, installer lock error classification inspection |
| **D-155.4** `fvpn-agent` `recoverOnce` runs `fvpn-updater recover` with a 250 ms per-pass wait (`recoverWaitBudget`), sets `Setpgid: true` and `Cancel` (`SIGKILL` to `-pid`), keeps a single in-flight background child across `Boot`/`Step`, and kills/waits for it on `Close` (`stopRecovery`). | `node/agent/updaterecover.go:18-167`, `node/agent/engine.go:381,604` | **wrong** — `recoverOnce` returns before setting `e.recoverBlocked = true` when `e.runCtx.Err() != nil` (`updaterecover.go:36-40`), and `bootWithBudget` sets `e.runCtx = caller` without clearing it (`engine.go:471`), allowing a canceled `Boot` context to skip recovery with `recoverBlocked == false` (admitting serving) or wedge later `Step` retries (`H-1`). | `TestR30_H1_CanceledBootContextSkipsRecoverAndAdmitsServing`, `TestR30_H1_PerCallBootContextWedgesStepRecovery` (`// TODO(review r30): H-1`) |
| **D-155.5** `fvpn-agent` pre-checks `update.lock` via `nodelock.Acquire(..., 1, 0)` before spawning `fvpn-updater recover` so an older `fvpn-updater` binary (pre-exit-3) is not spawned while the parent updater holds `update.lock`; live contention (`nodelock.ErrBusy` with detail `"lock"` or child exit `3`) keeps `recoverBlocked = true` (`recovery_required`) and retries on later steps without latching `recovered = true`. | `node/agent/updaterecover.go:45-53,128-167` | **implemented** | `TestD155_OldUpdaterDefersBeforeSpawn`, `TestD155_Exit3RetriesUntilClean`, `TestR30_D155_D156_D157_HeldInvariants` |
| **D-155.6** Stale lock metadata (`lock boot id`, `lock pid`, `lock process`, `lock start time`) or exit `1`/`2` from `fvpn-updater recover` is treated as a real recovery failure and latches `recovered = true, recoverBlocked = true` for the lifetime of the agent process. | `node/agent/updaterecover.go:47-51,132-135,155-167` | **implemented** | `TestD155_StaleLockMetadataLatchesFailure`, `TestD155_RealFailureLatches` |
| **D-156.1** At the start of `boot`, `startupHeartbeat` writes `Reason: "boot_adopting"` (`watchdog.AdoptionPending`) with `Serving: false` and `BaselineOK: false` while preserving `prior.Mono`, `prior.Wall`, `prior.Generation`, and `prior.PolicyExpires` if and only if the prior receipt has matching `BootID`, `Serving == true`, `BaselineOK == true`, `Generation > 0`, `0 < prior.Mono <= now`, `now - prior.Mono <= WatchdogStaleSeconds`, and `prior.PolicyExpires > now`; otherwise it writes `Reason: "boot_unverified"` with zero `PolicyExpires`. | `node/agent/engine.go:502-510`, `node/agent/boot_heartbeat.go:13-46` | **implemented** | `TestD156_AdoptionPreservesServingAgainstConcurrentWatchdog`, `TestR30_D155_D156_D157_HeldInvariants` |
| **D-156.2** A second agent startup over an existing `boot_adopting` receipt cannot refresh or extend `boot_adopting` (`prior.Serving` and `prior.BaselineOK` are `false`), downgrading immediately to `boot_unverified`. | `node/agent/boot_heartbeat.go:16-21` | **implemented** | `TestD156_WatchdogRejectsUnverifiedAndExpiredPendingReceipts`, `TestR30_D155_D156_D157_HeldInvariants` |
| **D-156.3** `fvpn-watchdog` `baselineDecision` accepts `Reason == "boot_adopting"` without installing the baseline only when `!Serving`, `!BaselineOK`, `Generation > 0`, `PolicyExpires > 0`, `w.BootID != uuid.Nil`, `beat.BootID == w.BootID`, `beat.Mono <= mono`, `mono - beat.Mono <= WatchdogStaleSeconds`, `mono < beat.PolicyExpires`, and `Enforce.BaselinePresent` is `true`. | `node/watchdog/watch.go:170-215` | **implemented** | `TestR30_D156_WatchdogAdoptionPendingInvariants`, `TestD156_WatchdogRejectsUnverifiedAndExpiredPendingReceipts` |
| **D-156.4** `adoptStoredServing` still validates the captured `previous` receipt (not the `boot_adopting` file written by `startupHeartbeat`) and on failure `boot` installs the fail-closed baseline and writes `boot_baseline`. | `node/agent/engine.go:502,553-581`, `node/agent/restart.go:18-135` | **implemented** | `TestD156_FailedAdoptionFallsBackToBaseline`, `TestD156_CanceledBootInstallsBaselineAndPublishesNonServing` |
| **D-157.1** A verified closed-recovery `Apply` (`closedRecovery == true`) arms one process-local `recoveryRenewal` opportunity (`e.closedRecoveryRenewal = recoveryRenewal{armed: true}`) and clears `e.lastObs = time.Time{}` so the recovery step queues a fresh observation report after refreshing `e.decision = Decide(e.input())`. | `node/agent/engine.go:792-799,1381-1385`, `node/agent/recovery_renewal.go:10-23` | **wrong** — (1) when a pre-`enforce` `syncLease` in the same `Step` sets `e.leaseAttempted = true` before `syncDesired` advances `ConfigGeneration`, `renewAppliedLease` at `engine.go:789` is suppressed, leaving `e.lease` at the old generation (`Reasons = [policy_missing]`) so `noteRecoveryReport` cannot arm the renewal on that recovery step (`M-2`); (2) `observe()` still short-circuits at `state == e.lastObservationState && len(items) > 0` (`observe.go:38-45`) even when `closedRecoveryRenewal.armed && reportID == uuid.Nil` (`M-3`). | `TestR30_M2_PreEnforceLeaseAttemptBlocksPostRecoveryRenewal` (`// TODO(review r30): M-2`), `TestR30_M3_PendingOutboxDedupSkipsNewClosedRecoveryReport` (`// TODO(review r30): M-3`) |
| **D-157.2** `noteRecoveryReport` binds `reportID` only when `recoveryRenewalEligible()` holds and the newly queued report has `AppliedConfigGeneration == e.appliedGen`, `!MemberForwardingEnabled`, `SafetyStatus == passed`, `ProvisioningState == ready`, `Netbird.Backend == kernel`, `Netbird.PeerId.IsNull()`, `awaiting_member_grants` in `ForwardingBlockReasons`, `internet_only == passed`, `meter_path != failed`, and no isolation/meter errors. | `node/agent/recovery_renewal.go:27-61`, `node/agent/observe.go:59` | **implemented** | `TestD157_OldOrUnsafeReportCannotArmRecoveryRenewal` |
| **D-157.3** `acceptRecoveryReport` sets `accepted = true` only when `rep.ReportId == r.reportID`, `accepted.ReportId == rep.ReportId`, `accepted.Accepted == true`, `IdentityFrozen` is nil or `false`, and `recoveryRenewalEligible()` holds. | `node/agent/recovery_renewal.go:63-70`, `node/agent/observe.go:324` | **implemented** | `TestD157_RecoveryWaitsForObservationAcceptance`, `TestD157_OldOrUnsafeReportCannotArmRecoveryRenewal`, `TestR30_D155_D156_D157_HeldInvariants` |
| **D-157.4** `earlyRecoveryRenewal()` wakes the step loop immediately (`nextStepDelay` returns `min(interval, max(0, e.leaseRetryAt - e.clock.Mono()))`, respecting `Retry-After`), triggers `needRenew()`, consumes `e.closedRecoveryRenewal = recoveryRenewal{}` on dispatch in `syncLease` (including failed or empty responses), and is disarmed by `baseline` or normal `Apply`. | `node/agent/engine.go:645-647,1051-1052,1311-1312`, `node/agent/sync.go:118-121,156-158` | **implemented** | `TestD157_AT02_ClosedRecoveryRenewsAfterAcceptedReadyReport`, `TestD157_EmptyEarlyAnswerKeepsD067AndSecondRecoveryGetsOne`, `TestD157_EarlyRenewalHonorsRetryAfterAndDoesNotRearmOnFailure`, `TestR30_D155_D156_D157_HeldInvariants` |
| **D-158.1** Operator/scheduled updater commands (`run`, `seed`, `apply`, `flush`) wait up to `10s` (`updateLockWaitBudget`, polling every `20ms` via `updateLockPollInterval`) when `update.lock` is held by a live process (`lockBusyDetail(err) == "lock"`), while `recover` and `retain` keep the prompt 5-attempt (~80 ms) lock path. | `node/updater/lock_wait.go:12-67`, `node/updater/engine.go:389-424` | **wrong** — in `acquireCommandLock` (`node/updater/lock_wait.go:47-55`), once `staleAttempts > 0` on poll 1, `if staleAttempts > 0 { staleAttempts++ }` increments `staleAttempts` on every subsequent busy poll even when `lockBusyDetail(err) == "lock"` (live holder), aborting after 5 polls (~80 ms) with exit 3 (`UPDATE_BUSY: lock`) instead of waiting up to the 10 s live lock budget (`H-2`). | `TestR30_H2_StaleThenLiveLockMetadataAbortsEarlyWithExitBusy` (`// TODO(review r30): H-2`), `TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog` |
| **D-158.2** Stale holder metadata (`lock boot id`, `lock pid`, `lock process`, `lock start time`) fails after the prompt 5-attempt (~80 ms) window with exit `1` (`UPDATE_BUSY: lock <detail>`), without waiting 10 s. | `node/updater/lock_wait.go:47-55`, `node/updater/followup25_test.go:192-212` | **implemented** (for persistently stale metadata; see `H-2` for stale-then-live / alternating transitions) | `TestD158_StaleDetailsKeepPromptWindow` |
| **D-158.3** When and only when the first lock attempt in `acquireCommandLock` misses on contention, emit one structured log record `"update lock wait"` with `command`, `waited_ms`, `acquired`, and `budget_ms: 10000`. | `node/updater/lock_wait.go:28-47` | **wrong** — `waited = true` is set at line 43 before `if !errors.Is(err, nodelock.ErrBusy)` at line 45, so a non-busy error on the very first lock attempt (zero contention, zero wait) emits a spurious `"update lock wait"` record (`M-1`). | `TestR30_M1_NonBusyFirstLockErrorEmitsSpuriousWaitLog` (`// TODO(review r30): M-1`) |

---

## Findings by severity

### High

#### `H-1` — `recoverOnce` returns before setting `recoverBlocked = true` when `runCtx.Err() != nil`, and `bootWithBudget` pins `e.runCtx` to `Boot`'s caller context across later `Step` calls (`D-155`)

- **Owner:** `agent-core`
- **Where:** `node/agent/updaterecover.go:36-40`, `node/agent/engine.go:471,544,695,940`
- **Scenario:**
  1. In `recoverOnce` (`node/agent/updaterecover.go:36-40`):
     ```go
     if e.runCtx != nil && e.runCtx.Err() != nil {
         return
     }
     e.recoverBlocked = true
     ```
     On a newly started `Engine`, `e.recoverBlocked` is `false` (zero value). Returning at line 38 before setting `e.recoverBlocked = true` leaves both `e.recovered == false` and `e.recoverBlocked == false`.
  2. In `bootWithBudget` (`node/agent/engine.go:471`), PR `#454` added `e.runCtx = caller` without clearing it when `Boot` returns, and `Step(ctx)` (`node/agent/engine.go:676-695`) never updates `e.runCtx`.
  3. **Fail-closed violation (admitting member forwarding without running `fvpn-updater recover`):** If `Boot(bootCtx)` is called with an already-canceled `bootCtx` (which `bootWithBudget` still executes on its detached 5 s `bootLocal` budget to install the baseline), `e.runCtx` is set to the canceled `bootCtx` and `recoverOnce` returns at `updaterecover.go:38` with `e.recovered == false` and `e.recoverBlocked == false`. When the caller subsequently runs `Step(context.Background())`, `recoverOnce` at `engine.go:695` sees `e.runCtx.Err() != nil` (from the old `bootCtx`), returns at line 38 with `e.recoverBlocked == false`, and `Step` admits member forwarding (`e.serving = true`) without ever invoking `fvpn-updater recover` (even when `UpdaterConfig` is empty or an unrecovered update journal exists on disk).
  4. **Permanent recovery wedge / premature child kill across `Boot(bootCtx)` -> `Step(stepCtx)`:** If a caller passes a per-call timeout context `bootCtx` to `Boot(bootCtx)` while `update.lock` is busy (so `Boot` sets `e.recoverBlocked = true, e.recovered = false`, or spawns a background `recover` child parented to `e.runCtx = bootCtx` at `updaterecover.go:62`) and cancels `bootCtx` after `Boot` returns before calling `Step(stepCtx)`:
     - Any in-flight background `fvpn-updater recover` child is immediately `SIGKILL`ed mid-transaction when `bootCtx` is canceled.
     - Every subsequent `Step(stepCtx)` sees `e.runCtx.Err() != nil` at `updaterecover.go:37` and returns immediately without retrying `fvpn-updater recover`, wedging `recovery_required` for the lifetime of the `Engine` even after `update.lock` is free.
- **Proof:**
  - `TestR30_H1_CanceledBootContextSkipsRecoverAndAdmitsServing` in `node/agent/review_r30_test.go` (`// TODO(review r30): H-1`)
  - `TestR30_H1_PerCallBootContextWedgesStepRecovery` in `node/agent/review_r30_test.go` (`// TODO(review r30): H-1`)
- **Narrowest fix:**
  - In `recoverOnce` (`node/agent/updaterecover.go:33-40`), set `e.recoverBlocked = true` immediately after the `e.cfg.UpdaterRecover == ""` check (before checking context cancellation), and in `e.input()` (`node/agent/engine.go:940`) set `RecoverBlocked: e.recoverBlocked || (e.cfg.UpdaterRecover != "" && !e.recovered)`.
  - Set `e.runCtx` only in `Engine.Run(ctx)` (clearing it when `Run` returns); in `recoverOnce`, gate new spawns on `localWorkContext(ctx).Err() == nil` and parent the background child to `e.runCtx` when `Run` is active or to an engine-lifetime context canceled in `stopRecovery()`/`Close()` so a per-call `Boot` context neither kills the background child after `Boot` returns nor poisons subsequent `Step` calls.

#### `H-2` — Stale-then-live lock metadata transition in `acquireCommandLock` aborts after 5 polls (~80 ms) with `ExitBusy` (`exit 3`) instead of waiting the 10 s live lock budget (`D-158`)

- **Owner:** `updater`
- **Where:** `node/updater/lock_wait.go:47-55`
- **Scenario:**
  In `acquireCommandLock` (`node/updater/lock_wait.go:47-55`):
  ```go
  if staleAttempts > 0 {
      staleAttempts++
  } else if lockBusyDetail(err) != "lock" {
      staleAttempts = 1
  }
  if staleAttempts == promptLockAttempts || (staleAttempts == 0 && time.Since(started) >= updateLockWaitBudget) {
      return nil, err
  }
  ```
  `nodelock.Lock.Release()` (`node/nodelock/lock.go:97-113`) unlocks and closes `update.lock` without removing or truncating the file, leaving the previous exited holder's `{pid, boot_id, start_time}` on disk. When a new live holder (`fvpn-agent` or `fvpn-watchdog`) calls `tryAcquire` (`node/nodelock/lock.go:51-79`), it acquires `flock(LOCK_EX|LOCK_NB)` at line 58 before reading `/proc` and truncating/rewriting `update.lock` at lines 64-73.
  If `fvpn-updater {run,seed,apply,flush}` makes its first poll in that window (or just as an exited holder is replaced by a new live holder), poll 1 sees `"lock process"` and sets `staleAttempts = 1`. On polls 2–5 (`20ms`–`80ms`), the live holder's metadata is in place (`lockBusyDetail(err) == "lock"`), yet `if staleAttempts > 0 { staleAttempts++ }` unconditionally increments `staleAttempts` without checking `lockBusyDetail(err)`. At poll 5 (~80 ms), `staleAttempts == promptLockAttempts` (`5`) triggers and returns poll 5's bare `nodelock.ErrBusy`, which `withLockFor` wraps as `UPDATE_BUSY: lock` (`ExitBusy = 3`). Thus `fvpn-updater` aborts after ~80 ms with exit `3` even when the live holder releases at e.g. 250 ms (well inside the 10 s `updateLockWaitBudget`). Conversely, if poll 1 is stale, polls 2–4 are live, and poll 5 is stale, poll 5 aborts on the very first poll of the new stale incident instead of giving it 5 consecutive stale polls.
- **Proof:**
  - `TestR30_H2_StaleThenLiveLockMetadataAbortsEarlyWithExitBusy` in `node/updater/review_r30_test.go` (`// TODO(review r30): H-2`)
- **Narrowest fix:**
  - In `acquireCommandLock` (`node/updater/lock_wait.go:47-55`), reset `staleAttempts = 0` whenever `lockBusyDetail(err) == "lock"`, and increment `staleAttempts++` only when `lockBusyDetail(err) != "lock"`:
    ```go
    if lockBusyDetail(err) == "lock" {
        staleAttempts = 0
        if time.Since(started) >= updateLockWaitBudget {
            return nil, err
        }
    } else if staleAttempts++; staleAttempts >= promptLockAttempts {
        return nil, err
    }
    ```

---

### Medium

#### `M-1` — Non-busy lock failure on the first attempt in `acquireCommandLock` emits a spurious `"update lock wait"` log record (`D-158`)

- **Owner:** `updater`
- **Where:** `node/updater/lock_wait.go:43-47`
- **Scenario:**
  In `acquireCommandLock` (`node/updater/lock_wait.go:40-47`):
  ```go
  lock, err := e.acquireLock(ctx)
  if err == nil {
      return lock, nil
  }
  waited = true
  for {
      if !errors.Is(err, nodelock.ErrBusy) {
          return nil, err
      }
  ```
  Because `waited = true` is set at line 43 before checking `if !errors.Is(err, nodelock.ErrBusy)` at line 45, any first-attempt lock failure that is not `nodelock.ErrBusy` (for example, `e.Layout.Lock` is a directory, permission/read-only filesystem error, or `ctx` is already canceled on entry) triggers the deferred `"update lock wait"` log (`waited_ms: 0, acquired: false, budget_ms: 10000`) even though no lock contention occurred and the first lock attempt did not miss on `ErrBusy`, violating D-158 / Invariant 8 (*"when and only when the first lock attempt missed"*).
- **Proof:**
  - `TestR30_M1_NonBusyFirstLockErrorEmitsSpuriousWaitLog` in `node/updater/review_r30_test.go` (`// TODO(review r30): M-1`)
- **Narrowest fix:**
  - In `acquireCommandLock` (`node/updater/lock_wait.go:43-47`), move `waited = true` after the initial `if !errors.Is(err, nodelock.ErrBusy) { return nil, err }` check.

#### `M-2` — Pre-`enforce` `syncLease` in `Step` sets `e.leaseAttempted = true` and suppresses `renewAppliedLease` when the same step recovers into a newer desired generation (`D-157`)

- **Owner:** `agent-core`
- **Where:** `node/agent/engine.go:742-766,789`, `node/agent/sync.go:86,123,252-267`, `node/agent/recovery_renewal.go:21-22`
- **Scenario:**
  In `Step` (`node/agent/engine.go:742-799`):
  1. `syncLease` is called at line 743 (when `e.leaseGrantless()` is true, before `syncDesired` at line 752) and at line 764 (after `syncDesired`, before `enforce` at line 784). Whenever either pre-`enforce` call dispatches `RenewLeaseOnce`, it sets `e.leaseAttempted = true` (`node/agent/sync.go:123`) and sends `AppliedConfigGeneration = null` when `e.applyError != nil` (`sync.go:112`).
  2. If that pre-`enforce` lease request fails transiently (or ran at line 743 before `syncDesired` fetched a new desired generation `N+1`), `e.lease` remains at generation `N` while `e.leaseAttempted` is `true`.
  3. At line 784, `enforce` runs closed recovery for generation `N+1` (`leaseGenerationStale(in) == true`), clears `e.applyError = nil`, sets `e.appliedGen = N+1`, and arms `e.closedRecoveryRenewal = recoveryRenewal{armed: true}`.
  4. At line 789, `renewAppliedLease` calls `e.syncLease(ctx)`, which immediately returns `nil` at `sync.go:86` (`if e.leaseAttempted ...`). As a result, `e.lease` remains at generation `N` (`e.leaseBehindApplied() == true`), `e.decision` at line 795 has `Reasons = [policy_missing]` (not `awaiting_member_grants`), `noteRecoveryReport` (`recovery_renewal.go:21,58`) refuses to bind the recovery report, and D-157's `earlyRecoveryRenewal()` fails to arm on that recovery step.
- **Proof:**
  - `TestR30_M2_PreEnforceLeaseAttemptBlocksPostRecoveryRenewal` in `node/agent/review_r30_test.go` (`// TODO(review r30): M-2`)
- **Narrowest fix:**
  - In `renewAppliedLease` (`node/agent/sync.go:252-267`), track the `appliedGen` that was in force when the pre-`enforce` `syncLease` ran (or check whether `enforce` just advanced `e.appliedGen` / cleared `e.applyError` so `AppliedConfigGeneration` changed since the pre-`enforce` attempt) and clear `e.leaseAttempted = false` before calling `e.syncLease(ctx)` when `e.leaseRetryAt <= e.clock.Mono()`.

#### `M-3` — `observe()` pending-outbox deduplication skips queuing a new closed-recovery report when `lastObservationState` matches an older pending clean report (`D-157`)

- **Owner:** `agent-core`
- **Where:** `node/agent/observe.go:38-46`, `node/agent/engine.go:1382-1385`
- **Scenario:**
  When `enforce` completes a closed recovery, `node/agent/engine.go:1382-1385` sets:
  ```go
  e.closedRecoveryRenewal = recoveryRenewal{armed: closedRecovery}
  if closedRecovery {
      e.lastObs = time.Time{} // each recovery needs its own report, even at the same generation
  }
  ```
  However, in `observe` (`node/agent/observe.go:38-46`):
  ```go
  if state == e.lastObservationState {
      items, err := e.store.Pending(ctx)
      if err != nil {
          return err
      }
      if len(items) > 0 {
          return nil
      }
  }
  ```
  If closed recovery #1 at generation `G` queued a clean `awaiting_member_grants` report `R1` (`e.lastObservationState = S_clean_G`) that remained pending in `e.store.Pending(ctx)` during observation backoff, and an intermediate withdrawal + failure was not recorded by `observe()` (for example while `e.outboxFull` was `true`), then when closed recovery #2 succeeds at the same generation `G`, `e.closedRecoveryRenewal` is reset to `{armed: true, reportID: uuid.Nil}` while `e.lastObservationState` is still `S_clean_G` and `len(items) > 0`. `observe()` returns `nil` at line 44 without queuing a report for closed recovery #2 or calling `noteRecoveryReport`, leaving `e.closedRecoveryRenewal.reportID == uuid.Nil` so closed recovery #2 never arms `earlyRecoveryRenewal()`.
- **Proof:**
  - `TestR30_M3_PendingOutboxDedupSkipsNewClosedRecoveryReport` in `node/agent/review_r30_test.go` (`// TODO(review r30): M-3`)
- **Narrowest fix:**
  - In `observe` (`node/agent/observe.go:38`), do not deduplicate against pending outbox items when a closed recovery is waiting to bind its own report ID:
    ```go
    if state == e.lastObservationState && !(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil) {
    ```

---

## Routing

- **`agent-core`:** Fix `H-1` in `node/agent/updaterecover.go` and `node/agent/engine.go` (set `e.recoverBlocked = true` before the context-cancellation return in `recoverOnce`, guard `e.input().RecoverBlocked` with `e.cfg.UpdaterRecover != "" && !e.recovered`, and stop pinning `e.runCtx` to `Boot`'s per-call context in `bootWithBudget`), `M-2` in `node/agent/sync.go` (`renewAppliedLease` when `enforce` just advanced/cleared the ACKable generation after an earlier pre-`enforce` lease attempt), and `M-3` in `node/agent/observe.go` (bypass pending-outbox deduplication when `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil`), making `TestR30_H1_*`, `TestR30_M2_*`, and `TestR30_M3_*` in `node/agent/review_r30_test.go` pass and removing their `// TODO(review r30):` markers.
- **`updater`:** Fix `H-2` and `M-1` in `node/updater/lock_wait.go` (reset `staleAttempts = 0` when `lockBusyDetail(err) == "lock"` so a single initial stale read does not abort a subsequent live-holder wait at ~80 ms with exit `3`, and move `waited = true` after `if !errors.Is(err, nodelock.ErrBusy)` so non-busy first-attempt errors do not log `"update lock wait"`), making `TestR30_H2_*` and `TestR30_M1_*` in `node/updater/review_r30_test.go` pass and removing their `// TODO(review r30):` markers.
- **Questions for Ted:** None — all five findings are implementation bugs against the existing `D-155`, `D-157`, and `D-158` contracts and require no new spec or architectural decision.

---

## What held

- **Q1 (`D-155` updater & installer lock/exit classification):**
  - `ExitCode(err)` in `node/updater/errors.go:18-48` uses `errors.As` over the error tree, gives `ErrMembersBlocked` (`2`) strict priority over `ExitBusy` (`3`) regardless of `errors.Join` order, and returns `ExitBusy` (`3`) if and only if an `*Error` has `Code == apierr.UpdateBusy` and `Detail == "lock"`.
  - All six updater entry points (`Recover`, `Run`, `Flush`, `Retain`, `ApplyPending`, `Seed`) enter `withLockFor` before `withJournal`, so a contended invocation never opens, creates, or mutates `journal.db`.
  - `Recover` and `Retain` stay on the prompt 5-attempt (~80 ms) lock path without entering the 10 s command wait or logging `"update lock wait"`.
  - Proof tests: `TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog`, `TestD155_ExitCode`, `TestD155_AllEntryPointsLockBeforeJournal`, `TestD155_RecoverExitCodes`.
- **Q2 & Q3 (`D-155` agent-side `recoverOnce` under `Run(ctx)`):**
  - When `Engine.Run(ctx)` owns the process lifecycle, `recoverOnce` pre-checks `update.lock` (`nodelock.Acquire(..., 1, 0)`) before spawning `fvpn-updater recover`, preventing a legacy pre-exit-3 updater binary from being invoked while the parent updater holds `update.lock`.
  - While `update.lock` is held by a live process (`detail == "lock"`) or `fvpn-updater recover` exits `3` or is still running past the 250 ms per-pass wait, `e.recoverBlocked` stays `true` (`recovery_required`), `adoptStoredServing` is refused (`prelim.Serve == false`), `Boot` completes its baseline within its 5 s budget without waiting 10 s, only one background `recover` child is kept in flight across steps, and `Close()` kills its process group (`-pid`) and waits for it.
  - Real recovery failures (exit `1`, exit `2`, non-exit errors, or stale lock metadata `lock boot id` / `lock pid` / `lock process` / `lock start time`) latch `e.recovered = true, e.recoverBlocked = true` for the lifetime of the agent process.
  - Proof tests: `TestR30_D155_D156_D157_HeldInvariants/D155_busy_recover_blocks_adoption_and_clears_on_later_step`, `TestD155_OldUpdaterDefersBeforeSpawn`, `TestD155_Exit3RetriesUntilClean`, `TestD155_LongRecoverContinuesInBackgroundWithinBootBudget`, `TestD155_StaleLockMetadataLatchesFailure`, `TestD155_RealFailureLatches`.
- **Q4 (`D-156` `boot_adopting` receipt and watchdog coordination):**
  - `startupHeartbeat` (`node/agent/boot_heartbeat.go:13-24`) writes `Reason: "boot_adopting"` with `Serving: false, BaselineOK: false` while copying `prior.Mono`, `prior.Wall`, `prior.Generation`, and `prior.PolicyExpires` only when the prior receipt was a fresh, unexpired, same-boot `Serving && BaselineOK` receipt; it never resets `Mono` to `now` or extends `PolicyExpires`.
  - A second startup over `boot_adopting` sees `prior.Serving == false && prior.BaselineOK == false` and immediately writes `boot_unverified` with `PolicyExpires: 0`.
  - `Watch.baselineDecision` (`node/watchdog/watch.go:170-215`) preserves live rules for `boot_adopting` only when `!Serving && !BaselineOK && Generation > 0 && PolicyExpires > 0 && w.BootID != uuid.Nil && beat.BootID == w.BootID`, `beat.Mono <= mono`, `mono - beat.Mono <= WatchdogStaleSeconds`, `mono < beat.PolicyExpires`, and `Enforce.BaselinePresent` is `true`; every malformed, stale, expired, cross-boot, or missing-baseline case triggers an immediate watchdog baseline write.
  - Proof tests: `TestR30_D156_WatchdogAdoptionPendingInvariants`, `TestR30_D155_D156_D157_HeldInvariants/D156_second_startup_over_boot_adopting_downgrades_to_unverified`, `TestD156_AdoptionPreservesServingAgainstConcurrentWatchdog`, `TestD156_WatchdogRejectsUnverifiedAndExpiredPendingReceipts`, `TestD156_FailedAdoptionFallsBackToBaseline`.
- **Q5 (`D-157` one-shot early lease renewal after closed recovery):**
  - `noteRecoveryReport` and `acceptRecoveryReport` (`node/agent/recovery_renewal.go:16-74`) strictly verify that the accepted observation report matches the bound `reportID` from the current closed recovery, carries `AppliedConfigGeneration == e.appliedGen`, `!MemberForwardingEnabled`, `SafetyStatus == passed`, `ProvisioningState == ready`, `Netbird.Backend == kernel`, `Netbird.PeerId.IsNull()`, `awaiting_member_grants`, `internet_only == passed`, `meter_path != failed`, no isolation/meter errors, `accepted.Accepted == true`, and `IdentityFrozen` nil or `false`.
  - The one-shot opportunity is consumed on dispatch in `syncLease` (`node/agent/sync.go:119-121`) even if the request fails or central returns another grant-less lease (keeping the D-067 exponential schedule `10s -> 20s -> 40s -> 60s` intact thereafter), honors `Retry-After` (`e.leaseRetryAt`), and is immediately disarmed by `baseline` (`engine.go:1051-1052`) or any subsequent `Apply` (`engine.go:1311-1312`).
  - Proof tests: `TestR30_D155_D156_D157_HeldInvariants/D157_baseline_disarms_early_recovery_renewal`, `TestD157_AT02_ClosedRecoveryRenewsAfterAcceptedReadyReport`, `TestD157_EmptyEarlyAnswerKeepsD067AndSecondRecoveryGetsOne`, `TestD157_RecoveryWaitsForObservationAcceptance`, `TestD157_EarlyRenewalHonorsRetryAfterAndDoesNotRearmOnFailure`, `TestD157_OldOrUnsafeReportCannotArmRecoveryRenewal`.
- **Q6 & Q7 (`D-158` steady-state contention & cross-fix interactions):**
  - When `update.lock` is held continuously by a live holder with valid metadata, `run`, `seed`, `apply`, and `flush` wait up to 10 s and proceed as soon as the holder releases, logging a single `"update lock wait"` JSON record with `command`, `waited_ms`, `acquired`, and `budget_ms: 10000`, while uncontended acquisitions log nothing.
  - Across `D-155` + `D-156`, a busy `update.lock` during `Boot` sets `e.recoverBlocked = true` before `adoptStoredServing`, which causes `adoptStoredServing` to refuse adoption before touching `update.lock` and sends `Boot` straight to its fail-closed baseline and `boot_baseline` heartbeat.
  - Proof tests: `TestD158_CommandsProceedAfter600msHolder`, `TestD158_StaleDetailsKeepPromptWindow`, `TestD158_LiveTimeoutExits3AndLogsFailedWait`, `TestR30_D155_UpdaterRecoverStaysPromptNoWaitLog`.

---

## Not examined

- Live devstack 35 working files, containers, or runtime behaviour (prohibited by review brief while devstack 35 is running on the shared host).
- Privileged network lab (`tests/netlab`), real kernel `nft`/`tc` execution, and real `systemd` daemon-reload/service restart under root.
- Packages outside the node update/agent/watchdog/installer review scope (`apps/control`, `apps/worker`, `apps/portal`, `internal/*`).
