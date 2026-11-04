# R31 Adversarial Code Review — Recovery & Lock Fixes (`#470`, `#471`)

- **Base (`main`) SHA:** `fd0f249418926e875928d4e6799ac2fc05c6ba9f`
- **Reviewed commits:**
  - `79c0bb10ef79dc7f75dd05c8b532f1e1036671a0` (PR `#470`, merge `d2cb54461d90b3388184145020f4a53d96dd6226`) — `agent-core` follow-up 51 (`r30` `H-1`, `M-2`, `M-3`)
  - `4742e6a6c31c4be5859994a21678f60899d8372c` (PR `#471`, merge `fd0f249418926e875928d4e6799ac2fc05c6ba9f`) — `updater` follow-up 26 (`r30` `H-2`, `M-1` under clarified `D-158`)
- **Reviewer:** `agy/gemini-4-argon-high`

---

## 1. Fidelity

| r30 Finding / D-158 Clause | `file:line` | Status | Proof |
|---|---|---|---|
| **r30 `H-1`** — `Boot(ctx)` must not run `fvpn-updater recover` under the caller's short boot context (`context.WithoutCancel(ctx)` is used instead), `Run(ctx)` must check `ctx.Err()` before `Boot`, and usage `uploadCmd` must run under an engine-lifetime context canceled in `Close()` | `node/agent/updaterecover.go:24-27`, `node/agent/engine.go:303-312, 577-587, 730-733, 745-754`, `node/agent/usage.go:34-41, 211-225` | `implemented` | `TestR30_H1_ShortBootContextAbortsUpdaterRecoverAndUsageLoop` (`node/agent/review_r30_test.go:32-119`), `TestFollowup51_StandaloneBootKeepsUpdaterRecoverTimeoutOnShortCallerCtx` (`node/agent/followup51_test.go:27-79`), `TestFollowup51_RunStopBeforeBootDoesNotSpawnUpdaterRecover` (`node/agent/followup51_test.go:84-126`), `TestFollowup51_CloseCancelsBackgroundUsageUploadLoop` (`node/agent/followup51_test.go:130-169`), `TestR31_Held_H1_RecoverLifetimesAndFailClosedGate` (`node/agent/review_r31_test.go:215-270`) |
| **r30 `H-2`** — Stale/dead-holder (`lock pid`, `lock boot_id`, `lock corrupt`) fast-fail in `acquireCommandLock` after 5 consecutive stale/corrupt polls (~80ms across 5 polls at 20ms intervals) returning exit `1` | `node/updater/lock_wait.go:16, 34, 47-54` | `implemented` | `TestR30_H2_StaleOrCorruptLockMetadataWaitsFull10sInsteadOfFastFailing` (`node/updater/review_r30_test.go:21-97`), `TestFollowup26_D158_DeadHolderLockFastFailsExit1` (`node/updater/followup26_test.go:19-60`) |
| **r30 `M-1`** — Live-holder reset of stale counter (`lockBusyDetail(err) == "lock"` resets `staleAttempts = 0`) and `remaining := updateLockWaitBudget - time.Since(started)` deadline clamp before sleeping when `staleAttempts == 0` | `node/updater/lock_wait.go:47-62` | `implemented` | `TestR30_M1_TransientStaleReadsExhaustCountAcross10sAndSleepOvershootsBudget` (`node/updater/review_r30_test.go:108-181`), `TestFollowup26_D158_StaleThenLiveHandoverCompletesWithinBudget` (`node/updater/followup26_test.go:64-116`), `TestFollowup26_D158_LiveTimeoutAfterTransientStaleReadsExits3WithoutOvershoot` (`node/updater/followup26_test.go:121-162`) |
| **r30 `M-2`** — Clear `e.leaseAttempted = false` in `renewAppliedLease` when `e.leaseBehindApplied() && e.leaseAttemptGen < e.appliedGen` so a pre-`enforce` null-ACK renewal attempt (`leaseAttemptGen == 0`) does not block same-`Step` generation catch-up after closed recovery | `node/agent/sync.go:228-235` | `implemented` | `TestR30_M2_ClosedRecoveryAfterUnrecordedWithdrawalSkipsSameStepLeaseCatchUp` (`node/agent/review_r30_test.go:132-194`), `TestFollowup51_ClosedRecoveryAfterFailedPreEnforceSyncRenewsLeaseSameStep` (`node/agent/followup51_test.go:175-211`), `TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport` (`node/agent/review_r31_test.go:278-333`) |
| **r30 `M-3`** — Bypass pending-outbox deduplication in `observe` when an armed closed recovery still needs its own post-recovery observation report bound (`e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil`) | `node/agent/observe.go:38-45` | `wrong` (incomplete guard; see **Finding `M-1`**) | `TestR30_M3_PendingObservationDedupStrandsSecondClosedRecoveryEarlyRenewal` (`node/agent/review_r30_test.go:205-268`) and `TestFollowup51_PendingObservationOutboxDoesNotSuppressSecondRecoveryReport` (`node/agent/followup51_test.go:216-267`) pass for the bindable recovery case, but `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` (`node/agent/review_r31_test.go:49-209`) fails on `main` when the closed recovery is ineligible or its report cannot bind |
| **Clarified `D-158` clause (1)** — Fast-fail stale/corrupt lock metadata (`lock pid`, `lock boot_id`, `lock corrupt`) after 5 consecutive stale/corrupt polls (~80ms at 20ms intervals) with exit code `1` | `node/updater/lock_wait.go:15-16, 34, 39, 47-54`, `node/updater/engine.go:1131-1140` | `implemented` | `TestFollowup26_D158_DeadHolderLockFastFailsExit1` (`node/updater/followup26_test.go:19-60`), `TestR30_H2_StaleOrCorruptLockMetadataWaitsFull10sInsteadOfFastFailing` (`node/updater/review_r30_test.go:21-97`) |
| **Clarified `D-158` clause (2)** — Reset consecutive stale counter (`staleAttempts = 0`) whenever a poll sees a live flock holder (`"lock"`) so handover/truncation windows across a 10s wait do not accumulate to 5 | `node/updater/lock_wait.go:47-51` | `implemented` | `TestFollowup26_D158_StaleThenLiveHandoverCompletesWithinBudget` (`node/updater/followup26_test.go:64-116`), `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` (`node/updater/review_r31_test.go:28-109`) |
| **Clarified `D-158` clause (3)** — Clamp each live-lock sleep to `remaining := updateLockWaitBudget - time.Since(started)` and return `err` (exit `3` for live `"lock"`) when `remaining <= 0` without overshooting the 10s budget | `node/updater/lock_wait.go:55-62`, `node/updater/engine.go:1131-1140` | `implemented` | `TestFollowup26_D158_LiveTimeoutAfterTransientStaleReadsExits3WithoutOvershoot` (`node/updater/followup26_test.go:121-162`), `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` (`node/updater/review_r31_test.go:28-141`) |
| **Clarified `D-158` clause (4)** — Emit one structured `"update lock wait"` record (`command`, `acquired`, `waited_ms`, `budget_ms`) whenever at least one retry occurred (`waited == true`) | `node/updater/lock_wait.go:25-33, 41, 46` | `implemented` | `TestFollowup26_D158_*` (`node/updater/followup26_test.go:59, 108, 158`), `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` (`node/updater/review_r31_test.go:66, 107, 139, 171`) |

**Unrequested behavior changes check:**
- In PR `#471` (`updater`), no unrequested behavior change slipped in. `Recover` and `Retain` still call `withLock` -> `withLockFor(ctx, "", ...)` (`node/updater/engine.go:144-159, 207-208`, `node/updater/retain.go:16-17`), which calls `nodelock.AcquireWithBootID` directly with `promptLockAttempts = 5` and never enters `acquireCommandLock`.
- In PR `#470` (`agent-core`), the pending-outbox deduplication bypass at `node/agent/observe.go:39` was written as `!(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil)` without checking `e.recoveryRenewalEligible()` or whether `noteRecoveryReport(rep)` (`node/agent/recovery_renewal.go:27-61`) can actually bind `rep.ReportId`. That omission introduces **Finding `M-1`** below.

---

## 2. Findings by Severity

- **Critical:** 0
- **High:** 0
- **Medium:** 1 (`M-1`)
- **Low:** 0

### `M-1` (Medium) — Unbindable or ineligible closed recovery bypasses pending-outbox deduplication on every due observation during an observation outage, filling the 32-entry outbox (`outbox_full`) and tearing down the held closed recovery

- **Where:** `node/agent/observe.go:39`, `node/agent/recovery_renewal.go:16-61` (interacting with `node/agent/engine.go:786, 1145-1159, 1386-1390`, `node/agent/recovery.go:16-59`, `node/agent/decide.go:140-142`)
- **Owner:** `agent-core`
- **Scenario:**
  1. Whenever `enforce` completes a closed-recovery `Apply` (`wasClosed == true` and `e.applyError == nil`), `node/agent/engine.go:1386-1390` sets `e.closedRecoveryHeld = true`, `e.closedRecoveryRenewal = recoveryRenewal{armed: true}` (`reportID == uuid.Nil`), and `e.lastObs = time.Time{}`.
  2. To fix `r30` `M-3`, `node/agent/observe.go:38-45` changed pending-outbox deduplication from `if state == e.lastObservationState` to:
     ```go
     if state == e.lastObservationState && !(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil) {
     ```
  3. However, `e.closedRecoveryRenewal` stays `{armed: true, reportID: uuid.Nil}` whenever `e.noteRecoveryReport(rep)` (`node/agent/recovery_renewal.go:27-61`, called at `observe.go:55`) returns early without setting `r.reportID = rep.ReportId`. `noteRecoveryReport` neither disarms `e.closedRecoveryRenewal` nor marks it bound in any of the following states while `e.closedRecoveryHeld` remains `true`:
     - **Case A — Isolation error in `rep.Errors` (`recovery_renewal.go:52-57`):** An earlier `Apply` failed with `&enforce.CheckError{Reason: enforce.ReasonIsolation}` and rollback quarantine, so `recordApplyQuarantine` (`node/agent/quarantine.go:94-107`) latched `e.containment.Code = api.ErrorCodeISOLATIONPOLICYINVALID`, which `attachContainment` (`node/agent/quarantine.go:175-210`, called at `observe.go:142`) appends to `rep.Errors` on every observation report (or `len(e.pendingRevoke) > 0` appends `ISOLATION_POLICY_INVALID` at `decide.go:292-300`). When the next `Apply` succeeds as a grant-less closed recovery, `e.recoveryRenewalEligible()` (`recovery_renewal.go:16-23`) is `true` and `retainClosedRecovery` (`node/agent/recovery.go:30-59`) keeps `e.closedRecoveryHeld = true`, yet `noteRecoveryReport` rejects every report at `recovery_renewal.go:55` (`errItem.Code == api.ErrorCodeISOLATIONPOLICYINVALID`) and leaves `r.reportID == uuid.Nil`.
     - **Case B — Failed meter path (`e.meterFailed == true`, `recovery_renewal.go:34, 48-50, 54-56`):** `recoveryRenewalEligible()` (`recovery_renewal.go:16-23`) does not check `!e.meterFailed`, whereas `noteRecoveryReport` rejects `rep.SafetyStatus != api.SafetyStatusPassed`, `NodeCheckNameMeterPath == HealthCheckStatusFailed`, and `ErrorCodeMETERPATHINVALID`, leaving `r.reportID == uuid.Nil`.
     - **Case C — Ineligible closed recovery with no `awaiting_member_grants` (`!e.recoveryRenewalEligible()`, `recovery_renewal.go:16-23, 29, 40-43, 58`):** `observe.go:39` does not call `e.recoveryRenewalEligible()`. For example, if `/leases:renew` is failing or waiting on `Retry-After` when desired state advances to a new generation, the closed-recovery `Apply` succeeds for the new generation while `e.leaseBehindApplied()` is `true` (`node/agent/sync.go:244-249`), so `ForwardingBlockReasons` is `[policy_missing]` (no `awaiting_member_grants`), or `len(e.desired.DeviceBindings) == 0` where `e.leaseGrantless()` is `true` (`sync.go:188-190`) but `grantsProvisional` is `false` (`decide.go:550-555`), yielding `[no_active_members]`. `noteRecoveryReport` returns early without binding `r.reportID`.
  4. During a Central `/observations` outage or `Retry-After` window (`len(items) > 0` in `e.store.Pending(ctx)`), every due observation (`wall.Sub(e.lastObs) >= interval`, default 10s) at an unchanged `observationState` bypasses pending-outbox deduplication at `observe.go:39` and enqueues another identical report into the 32-entry SQLite outbox (`observe.go:196, 296`), whereas the pre-`#470` deduplication held 1 pending report.
  5. Once `len(pending) >= 32` (`e.outboxFull = true` at `observe.go:296` and `engine.go:786`):
     - `Decide` (`node/agent/decide.go:140-142`) appends `outbox_full` (`d.Serve = false`), which causes `recoveryDecision` (`node/agent/recovery.go:23-24`) and `retainClosedRecovery` (`node/agent/recovery.go:37-39`) to return `false`; `enforce` (`node/agent/engine.go:1145-1159`) then runs `e.handoff(ctx)` and `e.baseline(ctx)`, tearing down the held closed-recovery ruleset (`e.closedRecoveryHeld = false`) and blocking member forwarding even if a populated lease arrives; and
     - `observe` (`node/agent/observe.go:33`: `if !due || e.outboxFull { return nil }`) drops all new observations — including real state transitions — until `e.observationRetryAt` expires, `flush` drains pending observations below 32, and a subsequent `Step` recomputes `e.outboxFull` at `engine.go:786`.
- **Proof:** `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` in `node/agent/review_r31_test.go:49-209`:
  - `isolation_error_in_report_fills_outbox_and_tears_down_recovery` (`review_r31_test.go:50-113`): fails on `main` with `len(pending)=32 (want 1), outboxFull=true, closedRecoveryHeld=false, reasons=[outbox_full awaiting_member_grants]`.
  - `failed_meter_path_fills_outbox` (`review_r31_test.go:115-159`): fails on `main` with `pending outbox grew from 3 to 35 at unchanged state when meterFailed prevented binding`.
  - `no_awaiting_member_grants_lease_behind_applied_fills_outbox` (`review_r31_test.go:161-208`): fails on `main` with `pending outbox grew from 1 to 32 at unchanged state when !recoveryRenewalEligible()`.
- **Narrowest fix:**
  - In `node/agent/recovery_renewal.go:16-61` and `node/agent/observe.go:38-60`:
    1. Make `recoveryRenewalEligible()` require the engine-level prerequisites that `noteRecoveryReport` requires (`!e.meterFailed`, `e.baselineOK`, `!e.conntrackBlocked`, `len(e.pendingRevoke) == 0`, and `grantsProvisional(e.input())`), and extract a pure report check `e.canBindRecoveryReport(rep api.NodeObservationReport) bool` used by `noteRecoveryReport(rep)`.
    2. In `observe` (`node/agent/observe.go:38-60`), only allow the pending-outbox deduplication bypass when `e.recoveryRenewalEligible()` is `true` and `e.canBindRecoveryReport(rep)` is `true` (so an unchanged `observationState` with pending items in the outbox is enqueued only when `noteRecoveryReport(rep)` actually binds `rep.ReportId` to `e.closedRecoveryRenewal.reportID`, while keeping `e.closedRecoveryRenewal.armed = true` if `e.leaseBehindApplied()` is still waiting on a catch-up lease renewal).

---

## 3. Routing

- **`agent-core`:** Fix `r31` `M-1` in `node/agent/observe.go:38-60` and `node/agent/recovery_renewal.go:16-61`, and remove the `// TODO(review r31): M-1` markers in `node/agent/review_r31_test.go`.
- **`updater`:** None (all `r30` `H-2` / `M-1` fixes and `D-158` clauses held).
- **Decisions needed from Ted:** None (`M-1` is an implementation defect against `D-157` / `r30 M-3` and the 32-entry observation outbox contract).

---

## 4. Answers to Brief Questions 1–7

### Question 1 — `H-1` (`Boot` / `Run` / `recoverOnce` / `uploadCmd` context lifetimes)

1. **`recoverOnce(ctx)` (`node/agent/updaterecover.go:16-61`):**
   - `recoverContext(ctx)` (`updaterecover.go:24-27`) wraps a non-nil caller context with `context.WithoutCancel(ctx)`:
     ```go
     func recoverContext(ctx context.Context) context.Context {
         if ctx == nil {
             return context.Background()
         }
         return context.WithoutCancel(ctx)
     }
     ```
   - `recoverOnce` (`updaterecover.go:34`) creates `rctx, cancel := context.WithTimeout(recoverContext(ctx), updaterRecoverTimeout)` where `updaterRecoverTimeout = 15 * time.Second` (`updaterecover.go:14`).
   - Thus a standalone caller calling `Boot(shortCtx)` whose `shortCtx` expires after `Boot`'s own baseline/quarantine steps (`node/agent/engine.go:303-310`) cannot abort `fvpn-updater recover` mid-flight; the child gets its full 15s timeout via `context.WithoutCancel(ctx)`.
2. **Every caller of `Boot`, `Run`, `Step`, and `recoverOnce`:**
   - `Boot(ctx)` (`node/agent/engine.go:303-313`): checks `if err := ctx.Err(); err != nil { e.notePreRecoverStop(); return err }` at lines `304-307`, runs `bootQuarantineOrBaseline(ctx)` at lines `308-310`, then calls `e.recoverOnce(ctx)` at line `311`.
   - `Run(ctx)` (`node/agent/engine.go:577-640`): checks `if err := ctx.Err(); err != nil { e.notePreRecoverStop(); return err }` at lines `578-580` **before** calling `e.Boot(ctx)` at line `581`. If `Run`'s context is already canceled at entry, `notePreRecoverStop()` sets `e.recoverBlocked = true` and recomputes `e.decision = Decide(e.input())` without spawning `fvpn-updater recover`.
   - `Step(ctx)` (`node/agent/engine.go:756-858`): calls `e.recoverOnce(ctx)` at line `767` (before `sync`, `enforce`, `observe`, `flush`). Because `e.recovered` is a one-shot boolean set to `true` only when `recoverOnce` succeeds (`updaterecover.go:38, 60`), `Step` is a no-op once `Boot` succeeded, and retries `recoverOnce` on subsequent `Step` calls if `Boot` failed or was skipped.
3. **`e.recovered`, `e.recoverBlocked`, and `e.input().RecoverBlocked` before, during, and after `Boot`/`Run`/`Step`:**
   - In ` Engine.input()` (`node/agent/engine.go:880-899`), line `896` computes:
     ```go
     RecoverBlocked: e.recoverBlocked || (strings.TrimSpace(e.cfg.UpdaterRecover) != "" && !e.recovered),
     ```
   - In `Decide(in Input)` (`node/agent/decide.go:149-151`), `if in.RecoverBlocked` appends `api.ForwardingBlockReasonRecoveryRequired` (`recovery_required`), which forces `d.Serve = false` (`decide.go:199`) and causes `retainClosedRecovery` (`node/agent/recovery.go:40-42`) to return `false`.
   - Therefore, whenever `cfg.UpdaterRecover` is non-empty:
     - **Before `Boot` / on canceled `Run` or `Boot`:** `e.recovered` is `false` (and `notePreRecoverStop()` also sets `e.recoverBlocked = true`), so `e.input().RecoverBlocked == true` and `Decide` blocks forwarding with `recovery_required`.
     - **When `recoverOnce` fails (`err != nil`):** `updaterecover.go:47-55` sets `e.recoverBlocked = true` and leaves `e.recovered = false`, so `e.input().RecoverBlocked == true`.
     - **Only after `recoverOnce` exits `0`:** `updaterecover.go:59-60` sets `e.recoverBlocked = false` and `e.recovered = true`, clearing `RecoverBlocked`.
     - There is **no** path where `fvpn-agent` serves member traffic before `fvpn-updater recover` has succeeded when `UpdaterRecover` is configured. Verified by `TestR31_Held_H1_RecoverLifetimesAndFailClosedGate` (`node/agent/review_r31_test.go:215-270`).
4. **Usage `uploadCmd` lifetime (`node/agent/engine.go:730-754`, `node/agent/usage.go:34-41, 211-225`):**
   - `uploadContext()` (`node/agent/engine.go:730-733`) lazily creates `e.uploadCtx, e.uploadCancel = context.WithCancel(context.Background())`.
   - `startUsageLoop(ctx)` (`node/agent/usage.go:34-41`) runs the background usage loop only when `!e.cfg.Once` (`if e.cfg.Once || e.meter == nil || e.usage == nil { return }`), passing `e.uploadContext()` (not `Boot`'s caller `ctx`) into `e.usage.Start(e.uploadContext(), e.uploadCmd)`.
   - In `once` mode (`e.cfg.Once == true`), `startUsageLoop` is a no-op and `Step` calls `e.flushUsageOnce(ctx)` synchronously (`node/agent/engine.go:845`, `usage.go:211-225`).
   - In `Close()` (`node/agent/engine.go:745-754`), `e.uploadCancel()` is called before `e.usage.Close()`, which waits on `u.done` (`internal/meter/uploader.go`) before `e.store.Close()` runs. Thus `uploadCmd` cannot outlive `Close()` or race against closed SQLite state.

---

### Question 2 — `M-2` (same-`Step` lease catch-up after closed recovery)

1. **`renewAppliedLease` (`node/agent/sync.go:228-242`):**
   ```go
   func (e *Engine) renewAppliedLease(ctx context.Context) error {
       if e.desired == nil || e.appliedGen == 0 {
           return nil
       }
       if e.leaseBehindApplied() && e.leaseAttemptGen < e.appliedGen {
           e.leaseAttempted = false
       }
       if !e.leaseNeedsRenewal(e.clock.Wall()) {
           return nil
       }
       if err := e.syncLease(ctx); err != nil {
           return err
       }
       e.decision = Decide(e.input())
       return nil
   }
   ```
   - In `Step(ctx)` (`node/agent/engine.go:756-858`), `e.sync(ctx)` runs before `e.enforce(ctx)` (`engine.go:769-819`), and `e.renewAppliedLease(ctx)` runs after `e.enforce(ctx)` (`engine.go:824-829`).
   - Inside `syncLease(ctx)` (`node/agent/sync.go:73-138`):
     - Line `79` computes `ackGen := e.appliedGen` (which is `0` when a prior withdrawal or failed apply cleared `e.appliedGen = 0`, `engine.go:1032, 1300`).
     - Lines `85-86` set `e.leaseAttempted = true` and `e.leaseAttemptGen = ackGen` (`0`) before calling `e.control.RenewLease(ctx, e.cfg.NodeID, req)`.
   - When `enforce(ctx)` then succeeds as a closed-recovery `Apply`, `e.appliedGen` becomes `e.desired.PolicyGeneration` (`> 0`, `engine.go:1357`).
   - In `renewAppliedLease(ctx)`, if the node has no lease or its lease is behind `e.appliedGen` (`e.leaseBehindApplied() == true`, `sync.go:244-249`) and the pre-`enforce` attempt only carried `e.leaseAttemptGen < e.appliedGen` (e.g. `0 < 14`), `e.leaseAttempted` is reset to `false`.
2. **Interaction with `leaseRetryAt`, `leaseWaitTimer`, `leaseRetryErr`, `provisionalRetryAt`, `closedRecoveryRenewal`, and `D-067`:**
   - **Quotes of `D-067`:**
     - `docs/ROADMAP.md:157`: `| D-067 | Grant-less recovery lease retry cadence | When a node holds a valid lease for the applied policy generation whose member_grants are empty and Central has signalled awaiting_member_grants, fvpn-agent retries /leases:renew every 10s (monotonic) before normal half-TTL renewal so member forwarding opens promptly once Central finishes compiling grants after an outage or restart. |`
     - `docs/handover/operator-guide.md:603`: ``When Central issues a grant-less recovery lease while `member_grants` are still compiling (`awaiting_member_grants`), `fvpn-agent` retries `/leases:renew` every `10s` on the monotonic clock (`D-067`) before the normal half-TTL window so member forwarding resumes promptly once Central's reconciler finishes.``
     - `docs/handover/operator-guide.md:748-761`: ``When `syncLease` still receives a grant-less lease for the applied generation (`member_grants: []` while `desired.device_bindings` is non-empty), `fvpn-agent` arms the next `10s` `D-067` retry (`provisionalRetryAt = mono + 10s`) regardless of whether that response was triggered by the initial recovery apply, the `D-157` early renewal after observation acceptance, or a normal `10s` `D-067` poll.``
   - **Pre-`enforce` failure with HTTP `Retry-After` (`e.leaseRetryAt > 0`):** If the pre-`enforce` `syncLease` failed with an HTTP `Retry-After` response, `setLeaseRetryAfter(err)` (`sync.go:90, 294-314`) set `e.leaseRetryAt = e.clock.Mono() + wait` and `e.leaseRetryErr = err`. Even after `renewAppliedLease` clears `e.leaseAttempted = false`, `e.leaseNeedsRenewal(now)` checks `if e.leaseRetryAt > 0 && e.clock.Mono() < e.leaseRetryAt { return false }` at `sync.go:154-156`, so `renewAppliedLease` **respects Central's `Retry-After` backoff** and does not double-hit `/leases:renew` during a `Retry-After` window.
   - **Pre-`enforce` failure without `Retry-After` (`e.leaseRetryAt == 0`):** `e.leaseBehindApplied()` is `true`, so `leaseNeedsRenewal` returns `true` at `sync.go:162-164`. `renewAppliedLease` calls `syncLease(ctx)` once with `applied_policy_generation = &e.appliedGen`.
   - **`closedRecoveryRenewal` and `provisionalRetryAt`:** Because `e.leaseBehindApplied()` is `true` during that catch-up call, `usedEarlyRecovery := e.earlyRecoveryRenewal()` (`sync.go:75`, `recovery_renewal.go:16-25`) is `false`, so the catch-up request **does not** consume `e.closedRecoveryRenewal` (`sync.go:118-120`). When the catch-up response returns a grant-less lease for `e.appliedGen`, `sync.go:121-125` arms `e.provisionalRetryAt = e.clock.Mono() + provisionalLeaseRetryInterval` (`10s`, `D-067`), while `e.closedRecoveryRenewal` stays `{armed: true, reportID: uuid.Nil}` ready for `observe` (`M-3`) and `flush` (`D-157`).
   - **Can `renewAppliedLease` ever loop or send >1 request in a single `Step`?** No: `renewAppliedLease` is not a loop; it calls `syncLease(ctx)` at most once per `Step` (`sync.go:236`), which sets `e.leaseAttempted = true` and `e.leaseAttemptGen = e.appliedGen` (`sync.go:85-86`).

---

### Question 3 — `M-3` (pending-outbox deduplication vs closed-recovery report binding)

1. **`observe.go:38-60` and `recovery_renewal.go`:**
   - `node/agent/observe.go:38-45`:
     ```go
     state := currentObservationState(rep)
     if state == e.lastObservationState && !(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil) {
         items, err := e.store.Pending(ctx)
         if err != nil {
             return err
         }
         if len(items) > 0 {
             return nil
         }
     }
     ```
2. **Answers to the 3 sub-questions in Question 3:**
   - **Sub-question (a): What happens if Central's `/observations` endpoint is unreachable (`503` / network error) across many steps after a closed recovery?**
     - **When the recovery report is bindable (normal `D-157` case):** On the recovery step (`Step` 1), `observe` builds `rep`, `e.noteRecoveryReport(rep)` (`observe.go:55`, `recovery_renewal.go:60`) binds `e.closedRecoveryRenewal.reportID = rep.ReportId` (`!= uuid.Nil`), and `e.store.Enqueue` stores that 1 report (`observe.go:56`). On every subsequent `Step` during the `/observations` outage while `state == e.lastObservationState`, `e.closedRecoveryRenewal.reportID != uuid.Nil`, so `!(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil)` is `false`! Pending-outbox deduplication at `observe.go:39-45` sees `len(items) > 0` and returns `nil` without enqueuing another report.
     - **When the recovery report is unbindable or the closed recovery is ineligible (`r31` Finding `M-1`):** Because `noteRecoveryReport(rep)` returns early without setting `r.reportID` (`r.reportID` remains `uuid.Nil`) and does not disarm `r.armed`, `!(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil)` stays `true` on every subsequent due observation! During an `/observations` outage, `observe` enqueues a duplicate report every `ObservationSeconds` (10s) at an unchanged `observationState` until `len(pending) == 32` (`e.outboxFull = true`), which tears down the held closed recovery (`outbox_full`) and blocks member forwarding and new observations. See **Finding `M-1`** and `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` (`node/agent/review_r31_test.go:49-209`).
   - **Sub-question (b): What happens if `noteRecoveryReport(rep)` refuses to bind `rep.ReportId` (e.g., because `rep` has an isolation error, failed meter path, or no `awaiting_member_grants`) while `closedRecoveryRenewal.armed` is true?**
     - As proved in **Finding `M-1`**, `noteRecoveryReport(rep)` (`node/agent/recovery_renewal.go:27-61`) returns early at lines `29`, `34`, `50`, `55`, or `58` without setting `r.reportID` and without clearing `r.armed`. Consequently `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil` remains `true` on every Step, defeating pending-outbox deduplication at `observe.go:39` across an observation outage until `e.outboxFull` trips at 32 pending reports.
   - **Sub-question (c): Can `closedRecoveryRenewal.armed` remain true with `reportID == uuid.Nil` when `!e.recoveryRenewalEligible()`, and if so, does `observe.go:39` bypass pending-outbox deduplication when no recovery renewal is even eligible?**
     - **Yes.** `e.closedRecoveryRenewal = recoveryRenewal{armed: true}` is set unconditionally on every closed-recovery `Apply` (`node/agent/engine.go:1388`), whereas `observe.go:39` checks `!(e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil)` **without** checking `e.recoveryRenewalEligible()`. Whenever a closed recovery completes while `!e.recoveryRenewalEligible()` (for example, `e.leaseBehindApplied() == true` when `/leases:renew` is failing during an outage, or `len(e.desired.DeviceBindings) == 0` where `ForwardingBlockReasons` is `[no_active_members]`), `observe.go:39` bypasses pending-outbox deduplication even though `e.recoveryRenewalEligible()` is `false` (or `noteRecoveryReport` rejects the report). Proved by subtest `no_awaiting_member_grants_lease_behind_applied_fills_outbox` in `node/agent/review_r31_test.go:161-208`.

---

### Question 4 — `H-2` & `M-1` (`acquireCommandLock` under clarified `D-158`)

1. **Exact quotes of `D-053` and `D-064`:**
   - **`D-053`:**
     - `docs/ROADMAP.md:141`: `| D-053 | Release Ed25519 verification, file lock, and atomic journal | Release manifests verify against a root key ring with min_signatures; mutations take an exclusive flock on /var/lib/vpn-node/update-journal/lock and persist fsync'd JSON state before each step. |`
     - `docs/ROADMAP.md:148`: `| D-060 | Updater CLI and exit codes | fvpn-updater exposes recover, stage, apply, status, retain, and flush with deterministic exit codes (0 ok, 2 verification/rejection, 3 lock busy, 4 rollback completed, 5 rollback failed, 1 other). |`
     - `docs/updates/node-updater.md:82`: `All mutating commands acquire the exclusive lock file at /var/lib/vpn-node/update-journal/lock (`D-053`).`
   - **`D-064`:**
     - `docs/ROADMAP.md:155`: `| D-064 | Agent-to-updater handoff and restart | Before fvpn-updater switches releases, fvpn-agent writes a handoff marker so baseline rules stay intact across the service restart instead of tearing down the node table. |`
     - `docs/ROADMAP.md:182`: `| D-093 | Agent drain and handoff CLI | fvpn-agent exposes drain and handoff subcommands plus SIGUSR1 handoff signalling so fvpn-updater can coordinate service restarts under D-064 without tearing down baseline rules. |`
     - `docs/updates/node-updater.md:409`: `When drain_command or handoff_command is configured, Apply and Rollback invoke the command before switching the current symlink (`D-064`).`
2. **Walk of `acquireCommandLock` (`node/updater/lock_wait.go:24-70`), `withLockFor` (`node/updater/engine.go:148-173`), `ExitCode(err)` (`node/updater/engine.go:1131-1140`), and `runUpdater` (`node/updater/host.go:243-259`):**
   - **Clause (1) — Fast-fail stale/corrupt metadata after 5 consecutive polls (~80ms) with exit `1`:**
     - Each loop iteration calls `nodelock.AcquireWithBootID(ctx, e.Layout.Lock, 1, 0, e.Node.BootID.String())` (`lock_wait.go:39`).
     - When `err` wraps `nodelock.ErrBusy` and `lockBusyDetail(err) != "lock"` (i.e. `"lock pid"`, `"lock boot_id"`, or `"lock corrupt"`), `staleAttempts++` (`lock_wait.go:50`).
     - At `staleAttempts == promptLockAttempts` (`5`, `lock_wait.go:16, 52-54`), `acquireCommandLock` immediately returns `nil, err` after 4 intervals of `20ms` = `80ms` elapsed.
     - `withLockFor` wraps `err` as `fail(CodeBusy, err.Error(), err)` (`engine.go:165`). In `ExitCode(err)` (`engine.go:1135-1137`), `fe.Code == CodeBusy && isLiveLockBusy(fe)` is `false` because `lockBusyDetail(fe)` is `"lock pid"`, `"lock boot_id"`, or `"lock corrupt"` (`!= "lock"`), so `ExitCode(err)` returns `1` (`runUpdater` at `host.go:244-252` treats only exit `3` as `ErrLockBusy`).
   - **Clause (2) — Reset stale counter on live flock holder (`"lock"`):**
     - At `lock_wait.go:47-49`, `if lockBusyDetail(err) == "lock" { staleAttempts = 0 }`. Any poll that observes a live flock holder resets the consecutive stale counter to `0`.
   - **Clause (3) — Clamp live-lock sleep to remaining 10s budget and exit `3` at 10s:**
     - At `lock_wait.go:55-62`, when `staleAttempts == 0`, `remaining := updateLockWaitBudget - time.Since(started)`; if `remaining <= 0`, `acquireCommandLock` returns `nil, err` immediately without sleeping past `10s`; otherwise `delay = min(lockRetryInterval, remaining)`.
     - For a live lock (`lockBusyDetail(err) == "lock"`), `isLiveLockBusy(fe)` (`lock_wait.go:73-75`) is `true`, so `ExitCode(err)` (`engine.go:1135-1137`) returns `3`.
   - **Clause (4) — Single `"update lock wait"` structured log record on contention:**
     - `waited := false` (`lock_wait.go:26`) is set to `true` only after the first attempt fails with `nodelock.ErrBusy` (`lock_wait.go:46`).
     - The deferred closure (`lock_wait.go:27-33`) runs once on function return and logs `"update lock wait"` with `command`, `acquired`, `waited_ms`, and `budget_ms` if and only if `waited` is `true`.

---

### Question 5 — Lock-Wait Edge Cases (`node/updater/lock_wait.go`)

Verified by code inspection of `node/updater/lock_wait.go:24-70` and by `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` in `node/updater/review_r31_test.go:28-174`:

1. **Alternating `stale, live, stale, live, ...` across 10s (`review_r31_test.go:29-68`):**
   - Poll 1 (`t = 0ms`, stale): `staleAttempts = 1`, sleeps full `delay = 20ms` (`staleAttempts != 0`).
   - Poll 2 (`t = 20ms`, live): `staleAttempts = 0`, `remaining = 9980ms`, sleeps `20ms`.
   - Every odd poll (`t = 40ms, 80ms, ..., 10000ms`, stale) sets `staleAttempts = 1` and sleeps `20ms`; every even poll (`t = 60ms, 100ms, ..., 10020ms`, live) resets `staleAttempts = 0` and checks `remaining := 10s - time.Since(started)`.
   - At poll 502 (`t = 10020ms`, live), `staleAttempts` resets to `0`, `remaining = -20ms <= 0`, and `acquireCommandLock` returns `updater: UPDATE_BUSY: lock` with **exit code `3`** at `elapsed = 10.02s` (bounded by at most one `20ms` stale sleep past `10s`) and emits **1** `"update lock wait"` record (`waited_ms: 10020`).
2. **4 stale polls (`80ms`) followed by 1 live poll, repeated across 10s (`review_r31_test.go:70-109`):**
   - Polls `1..4` (`t = 0, 20, 40, 60ms`) increment `staleAttempts` to `4` (never reaching `5`).
   - Poll `5` (`t = 80ms`, live) resets `staleAttempts = 0`.
   - Repeating this 5-poll (`100ms`) cycle 100 times reaches `t = 10000ms` (poll 501, stale `1`), `10020ms` (stale `2`), `10040ms` (stale `3`), `10060ms` (stale `4`), and at poll 505 (`t = 10080ms`, live), `staleAttempts` resets to `0`, `remaining = -80ms <= 0`, and `acquireCommandLock` returns `updater: UPDATE_BUSY: lock` with **exit code `3`** at `elapsed = 10.08s` (bounded by at most `4 * 20ms = 80ms` of trailing stale sleeps past `10s`) and emits **1** `"update lock wait"` record (`waited_ms: 10080`).
3. **Live lock until `9990ms` (or `10000ms`), then holder crashes without releasing (`lock pid`) (`review_r31_test.go:111-141`):**
   - Polls `1..500` (`t = 0..9980ms`) see `"lock"` (`staleAttempts = 0`), with poll 500 at `t = 9980ms` clamping `delay = min(20ms, 20ms) = 20ms`.
   - Before poll 501 at `t = 10000ms`, the holder dies leaving stale PID metadata (`"lock pid"`).
   - Polls `501..505` (`t = 10000, 10020, 10040, 10060, 10080ms`) see `"lock pid"`, incrementing `staleAttempts` from `1` to `5`. Because `staleAttempts > 0`, lines `56-62` do not cut off on `remaining <= 0`; вместо returning a false live-lock exit `3`, it completes the 5-poll (`80ms`) stale confirmation and returns `updater: UPDATE_BUSY: lock pid` with **exit code `1`** at `elapsed = 10.08s` (bounded to `10s + 80ms`) and emits **1** `"update lock wait"` record (`waited_ms: 10080`).
4. **First attempt returns non-`ErrBusy` error vs `ctx.Done()` vs `Recover`/`Retain`:**
   - **First attempt non-`ErrBusy` error (e.g. `EACCES`):** `nodelock.AcquireWithBootID` returns a non-`ErrBusy` error on poll 1; `lock_wait.go:43-45` (`if !errors.Is(err, nodelock.ErrBusy) { return nil, err }`) returns before `waited = true` (`line 46`), so **0** `"update lock wait"` records are emitted, and `withLockFor` wraps it as `CodeIO` (exit `1`).
   - **First attempt `ErrBusy`, subsequent attempt non-`ErrBusy` error (`review_r31_test.go:143-173`):** `waited` was set to `true` on poll 1 (`line 46`), so when poll 2 returns a non-`ErrBusy` error at `line 44`, the deferred log emits **exactly 1** `"update lock wait"` record with `acquired: false, waited_ms: 20` and exits `1`.
   - **`ctx.Done()` fires during wait (`followup26_test.go:240-269`):** `lock_wait.go:64-67` stops the timer and returns `nil, ctx.Err()`; the deferred log emits **exactly 1** `"update lock wait"` record with `acquired: false`.
   - **`Recover` and `Retain`:** Call `withLock` -> `withLockFor(ctx, "", fn)` (`engine.go:144-159, 207-208`, `retain.go:16-17`), which takes the `command == ""` branch (`nodelock.AcquireWithBootID(ctx, e.Layout.Lock, promptLockAttempts, lockRetryInterval, e.Node.BootID.String())`) and never calls `acquireCommandLock`. Nested calls when `e.lockHeld == true` return `fn()` immediately at `engine.go:151-154`.

---

### Question 6 — Proof & Lane Tests Audit

| Test File & Test Name | Target Finding / Clause | Failure Mode Tested | Runtime & Determinism |
|---|---|---|---|
| `node/agent/review_r30_test.go:32` — `TestR30_H1_ShortBootContextAbortsUpdaterRecoverAndUsageLoop` | `r30 H-1` | Subtest 1: `Boot(shortCtx)` where `shortCtx` expires `80ms` into a `250ms` `fvpn-updater recover` script; Subtest 2: background usage `uploadCmd` after `shortCtx` expires | `0.48s` (`0.29s` + `0.19s`); real process exec with deterministic sleep margins (`80ms` vs `250ms` vs `15s` timeout) |
| `node/agent/review_r30_test.go:132` — `TestR30_M2_ClosedRecoveryAfterUnrecordedWithdrawalSkipsSameStepLeaseCatchUp` | `r30 M-2` | Closed recovery #2 on same `Step` after a pre-`enforce` renewal with `applied_policy_generation: null` (`leaseAttemptGen == 0`) | `0.22s`; deterministic `fakeClock` + `fakecontrol` |
| `node/agent/review_r30_test.go:205` — `TestR30_M3_PendingObservationDedupStrandsSecondClosedRecoveryEarlyRenewal` | `r30 M-3` | Closed recovery #2 when recovery #1's observation is still in SQLite `Pending` at the same `observationState` | `0.21s`; deterministic `fakeClock` + `fakecontrol` |
| `node/agent/followup51_test.go:27` — `TestFollowup51_StandaloneBootKeepsUpdaterRecoverTimeoutOnShortCallerCtx` | `r30 H-1` | Standalone `Boot(bootCtx)` detaches child from `bootCtx` cancellation while preserving `updaterRecoverTimeout` | `0.25s`; real script exec |
| `node/agent/followup51_test.go:84` — `TestFollowup51_RunStopBeforeBootDoesNotSpawnUpdaterRecover` | `r30 H-1` | Pre-canceled `Run(ctx)` returns `context.Canceled` without spawning `fvpn-updater recover` | `0.13s`; deterministic |
| `node/agent/followup51_test.go:130` — `TestFollowup51_CloseCancelsBackgroundUsageUploadLoop` | `r30 H-1` | `Close()` cancels `uploadCtx` and aborts an in-flight blocking `uploadCmd` | `0.20s`; deterministic |
| `node/agent/followup51_test.go:175` — `TestFollowup51_ClosedRecoveryAfterFailedPreEnforceSyncRenewsLeaseSameStep` | `r30 M-2` | Pre-`enforce` `syncLease` fails (`FailLeases(1)`) with null ACK, then closed recovery `Apply` succeeds and `renewAppliedLease` catches up in the same `Step` | `0.18s`; deterministic `fakeClock` + `fakecontrol` |
| `node/agent/followup51_test.go:216` — `TestFollowup51_PendingObservationOutboxDoesNotSuppressSecondRecoveryReport` | `r30 M-3` | Second closed recovery binds its fresh report while first report is pending, then dispatches `D-157` early renewal once accepted | `0.23s`; deterministic `fakeClock` + `fakecontrol` |
| `node/watchdog/review_r30_test.go:15` — `TestR30_Held_WatchdogArmingAndConsecutiveFailureReset` | `r30 Held` | Watchdog pre-ready suppression, post-ready arming, and consecutive failure reset on recovery | `0.05s`; deterministic `fakeClock` |
| `node/updater/review_r30_test.go:21` — `TestR30_H2_StaleOrCorruptLockMetadataWaitsFull10sInsteadOfFastFailing` | `r30 H-2` / `D-158 (1)` | `lock pid`, `lock boot_id`, and `lock corrupt` fast-fail in `80ms` with exit `1` | `0.02s`; `testing/synctest` fake clock (zero wall-clock sleep) |
| `node/updater/review_r30_test.go:108` — `TestR30_M1_TransientStaleReadsExhaustCountAcross10sAndSleepOvershootsBudget` | `r30 M-1` / `D-158 (2),(3)` | Subtest 1: 4 stale reads (`60ms`) + live read + 1 more stale read resets counter and acquires lock at `200ms`; Subtest 2: 2 stale reads + live lock for remainder of 10s exits `3` at `10000ms` without overshooting | `0.02s`; `testing/synctest` fake clock |
| `node/updater/followup26_test.go:19` — `TestFollowup26_D158_DeadHolderLockFastFailsExit1` | `D-158 (1),(4)` | `Stage`, `Apply`, `Rollback`, `Flush`, `Status` fast-fail in `80ms` with exit `1` and log 1 `"update lock wait"` record across `pid`, `boot_id`, and `corrupt` | `0.11s`; `testing/synctest` fake clock |
| `node/updater/followup26_test.go:64` — `TestFollowup26_D158_StaleThenLiveHandoverCompletesWithinBudget` | `D-158 (2),(4)` | Two 4-poll (`80ms`) stale handover windows separated by live lock across `1.5s` complete `Flush` and `Status` with exit `0` | `0.08s`; `testing/synctest` fake clock |
| `node/updater/followup26_test.go:121` — `TestFollowup26_D158_LiveTimeoutAfterTransientStaleReadsExits3WithoutOvershoot` | `D-158 (2),(3),(4)` | Two 3-poll stale bursts (`0..60ms`, `500..560ms`) followed by live lock exits `3` at `10000ms` (`waited_ms: 10000`) | `0.05s`; `testing/synctest` fake clock |
| `node/updater/followup26_test.go:166` — `TestFollowup26_D158_UncontendedAndInternalLocksStayPrompt` | `D-158 (4)` | Uncontended `Flush` logs 0 wait records; `Recover` and `Retain` fail in `80ms` on a live lock with 0 command-wait records; `ctx` cancel during wait exits `1` at `120ms` with 1 wait record | `0.04s`; `testing/synctest` fake clock |
| `node/agent/review_r31_test.go:49` — `TestR31_M1_UnbindableClosedRecoveryFloodsPendingObservationOutbox` *(new)* | `r31 M-1` | Proves `observe.go:39` floods the 32-entry pending outbox and tears down closed recovery when `rep` has an isolation error, `meterFailed == true`, or `!recoveryRenewalEligible()` | `0.63s`; deterministic `fakeClock` + `fakecontrol` (fails on `main` at `// TODO(review r31): M-1`) |
| `node/agent/review_r31_test.go:215` — `TestR31_Held_H1_RecoverLifetimesAndFailClosedGate` *(new)* | `r30 H-1` | Verifies `input().RecoverBlocked` before/after `recoverOnce` and that canceled `Run` sets `recoverBlocked` (`recovery_required`) without spawning child | `0.14s`; deterministic |
| `node/agent/review_r31_test.go:278` — `TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport` *(new)* | Questions 3 & 7 (`M-2` + `M-3` + `D-157` + `D-067`) | Proves same-`Step` `M-2` catch-up + `M-3` pending-outbox bypass sends 1 catch-up renewal on recovery step, 0 early renewals before report acceptance, and at most 1 `D-157` early renewal inside the `D-067` slot | `0.22s`; deterministic `fakeClock` + `fakecontrol` |
| `node/updater/review_r31_test.go:28` — `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` *(new)* | Question 5 (`D-158` edge cases) | Proves alternating stale/live (`10020ms`, exit `3`), 4-stale-1-live repeated (`10080ms`, exit `3`), live-until-deadline-then-stale (`10080ms`, exit `1`), and busy-then-non-busy error (`20ms`, exit `1`, 1 record) | `0.13s`; `testing/synctest` fake clock |

---

### Question 7 — Cross-PR / End-to-End Interactions

1. **Scenario A — `fvpn-agent` starts under a short boot context while `fvpn-updater` holds `/var/lib/vpn-node/update-journal/lock` (either live or stale):**
   - `fvpn-agent` enters `Boot(ctx)` (`node/agent/engine.go:303-312`), installs baseline firewall rules (`bootQuarantineOrBaseline`), and invokes `fvpn-updater --config <path> recover` under `context.WithoutCancel(ctx)` with `updaterRecoverTimeout = 15s` (`node/agent/updaterecover.go:14, 24-34`).
   - Inside `fvpn-updater recover`, `Engine.Recover` (`node/updater/engine.go:175, 207`) calls `e.withLock(ctx, ...)`, which calls `e.withLockFor(ctx, "", fn)` (`engine.go:144-159`).
   - Because `command == ""`, `Recover` does **not** use the 10s operator-command wait (`acquireCommandLock`); it calls `nodelock.AcquireWithBootID(ctx, e.Layout.Lock, promptLockAttempts, lockRetryInterval, e.Node.BootID.String())` (`5` attempts * `20ms` = `80ms`).
   - Whether `/var/lib/vpn-node/update-journal/lock` is held by a live process (`"lock"`, exit `3`) or left behind by a dead process/old boot (`"lock pid"` / `"lock boot_id"`, exit `1`), `fvpn-updater recover` fails in `~80ms` (or if `Recover` acquires the lock and executes a rollback + health verification, `context.WithoutCancel(ctx)` gives it the full `15s` `updaterRecoverTimeout` unaffected by the caller's short boot context).
   - If `fvpn-updater recover` exits non-zero, `recoverOnce` (`node/agent/updaterecover.go:47-55`) sets `e.recoverBlocked = true` and leaves `e.recovered = false`, so `fvpn-agent` fails closed with `recovery_required` and retries `recoverOnce` on the next `Step` (`engine.go:767`).
   - Meanwhile, if the usage uploader triggers `/opt/vpn-node/current/bin/fvpn-updater --config /etc/vpn-node/updater.json flush` (`deploy/systemd/fvpn-agent.env.example:21`) under `e.uploadContext()` while `fvpn-updater` (`stage`/`apply`/`recover`) holds the lock:
     - If the holder is live, `Flush` waits up to `10s` (`D-158`) for the holder to finish and then flushes usage records (exit `0`), or exits `3` at `10.00s` if still busy.
     - If the holder is dead (`lock pid` / `lock boot_id` / `lock corrupt`), `Flush` fast-fails in `80ms` with exit `1` (`D-158`).
2. **Scenario B — `fvpn-agent` performs a closed recovery that exercises both `M-2` (same-`Step` lease catch-up) and `M-3` (pending-outbox deduplication exception):**
   - Verified end-to-end by `TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport` (`node/agent/review_r31_test.go:278-333`):
     1. Before `enforce`, `Step`'s `syncLease` fails (or runs while `e.appliedGen == 0`), setting `e.leaseAttempted = true` and `e.leaseAttemptGen = 0`.
     2. `enforce` applies generation 14 as a closed recovery (`e.appliedGen = 14`, `e.closedRecoveryHeld = true`, `e.closedRecoveryRenewal = {armed: true, reportID: uuid.Nil}`).
     3. Post-`enforce`, `renewAppliedLease` (`node/agent/sync.go:230-232`) sees `e.leaseBehindApplied() && e.leaseAttemptGen < e.appliedGen` (`0 < 14`), clears `e.leaseAttempted = false` (`M-2`), and calls `syncLease` once with `applied_policy_generation = 14`. Because `e.leaseBehindApplied()` was `true` at entry to `syncLease`, `usedEarlyRecovery` (`sync.go:75`) is `false`, so `e.closedRecoveryRenewal` is **not** consumed by the catch-up renewal. Central returns a grant-less lease for generation 14, and `syncLease` arms the 10s `D-067` timer (`e.provisionalRetryAt = mono + 10s`).
     4. Next in the same `Step`, `observe` (`node/agent/observe.go:39`) sees `state == e.lastObservationState` (from an earlier pending report in the outbox) and `e.closedRecoveryRenewal.armed && e.closedRecoveryRenewal.reportID == uuid.Nil` (`M-3`), enqueues the new report, and `noteRecoveryReport` binds `e.closedRecoveryRenewal.reportID = rep.ReportId`.
     5. If `flush` has not yet had `rep.ReportId` accepted by Central, `e.closedRecoveryRenewal.accepted` is `false`, so `e.earlyRecoveryRenewal()` (`recovery_renewal.go:16-25`) is `false` — **zero** early renewals fire before `rep.ReportId` is accepted.
     6. Once `flush` receives Central's `AcceptedReportIds` containing `rep.ReportId`, `noteAcceptedRecoveryReport` (`recovery_renewal.go:63-74`) sets `e.closedRecoveryRenewal.accepted = true`. On the next `Step` inside the 10s `D-067` slot (`mono < e.provisionalRetryAt`), `leaseNeedsRenewal` (`sync.go:169-171`) returns `true` via `e.earlyRecoveryRenewal()`, `syncLease` dispatches **exactly one** early `/leases:renew` request, and `sync.go:118-120` clears `e.closedRecoveryRenewal = recoveryRenewal{}` so subsequent steps inside the same `D-067` slot send no further requests.

---

## 5. What Held

- **`r30 H-1` (`Boot`/`Run` context detachment, fail-closed `recovery_required` gating, and engine-scoped usage upload context):**
  - `recoverContext(ctx)` (`node/agent/updaterecover.go:24-27`) detaches `fvpn-updater recover` from short caller contexts while keeping the 15s `updaterRecoverTimeout`.
  - `Run(ctx)` (`node/agent/engine.go:578-580`) and `Boot(ctx)` (`node/agent/engine.go:304-307`) check `ctx.Err()` before spawning `fvpn-updater recover` and call `notePreRecoverStop()` so `e.recoverBlocked` and `e.input().RecoverBlocked` remain `true` (`recovery_required`).
  - `startUsageLoop` (`node/agent/usage.go:34-41`) binds the background `uploadCmd` to `e.uploadContext()`, which `Close()` (`node/agent/engine.go:745-754`) cancels before closing the uploader and SQLite store.
  - **Proofs:** `TestR30_H1_ShortBootContextAbortsUpdaterRecoverAndUsageLoop` (`node/agent/review_r30_test.go:32`), `TestFollowup51_StandaloneBootKeepsUpdaterRecoverTimeoutOnShortCallerCtx` (`node/agent/followup51_test.go:27`), `TestFollowup51_RunStopBeforeBootDoesNotSpawnUpdaterRecover` (`node/agent/followup51_test.go:84`), `TestFollowup51_CloseCancelsBackgroundUsageUploadLoop` (`node/agent/followup51_test.go:130`), `TestR31_Held_H1_RecoverLifetimesAndFailClosedGate` (`node/agent/review_r31_test.go:215`).
- **`r30 M-2` (same-`Step` generation catch-up after closed recovery and interaction with `M-3` / `D-157` / `D-067`):**
  - `renewAppliedLease` (`node/agent/sync.go:228-235`) clears `e.leaseAttempted` only when `e.leaseBehindApplied() && e.leaseAttemptGen < e.appliedGen`, respects `e.leaseRetryAt`, does not consume `e.closedRecoveryRenewal`, and pairs cleanly with `D-157` and `D-067`.
  - **Proofs:** `TestR30_M2_ClosedRecoveryAfterUnrecordedWithdrawalSkipsSameStepLeaseCatchUp` (`node/agent/review_r30_test.go:132`), `TestFollowup51_ClosedRecoveryAfterFailedPreEnforceSyncRenewsLeaseSameStep` (`node/agent/followup51_test.go:175`), `TestR31_Held_M2_M3_InteractionSingleEarlyRenewalAfterAcceptedReport` (`node/agent/review_r31_test.go:278`).
- **`r30 H-2` & `M-1` under clarified `D-158` (`acquireCommandLock` in `node/updater/lock_wait.go`):**
  - Stale/corrupt metadata (`lock pid`, `lock boot_id`, `lock corrupt`) fast-fails after 5 consecutive polls (`80ms`) with exit `1`; live flock (`lock`) resets `staleAttempts = 0`, clamps sleep to `remaining := updateLockWaitBudget - time.Since(started)`, and exits `3` when `remaining <= 0`; contended invocations emit exactly one `"update lock wait"` record; `Recover` and `Retain` stay on the prompt 5-attempt path.
  - **Proofs:** `TestR30_H2_StaleOrCorruptLockMetadataWaitsFull10sInsteadOfFastFailing` (`node/updater/review_r30_test.go:21`), `TestR30_M1_TransientStaleReadsExhaustCountAcross10sAndSleepOvershootsBudget` (`node/updater/review_r30_test.go:108`), `TestFollowup26_D158_*` (`node/updater/followup26_test.go:19-270`), `TestR31_Held_D158_AnswerSequencesAndRecordInvariants` (`node/updater/review_r31_test.go:28`).
- **Watchdog `r30` held proofs (`node/watchdog/review_r30_test.go`):**
  - **Proof:** `TestR30_Held_WatchdogArmingAndConsecutiveFailureReset` (`node/watchdog/review_r30_test.go:15`).

---

## 6. Not Examined

- Central services (`apps/control`, `apps/worker`, `internal/*`), portal frontend (`apps/portal`), and database migrations (`db/migrations`), which were outside the scope of PRs `#470` and `#471`.
- Real-kernel `nft` / `tc` netlab execution (`tests/netlab`) and live systemd service supervision on target Ubuntu 24.04 hosts.
