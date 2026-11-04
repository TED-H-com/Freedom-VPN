# Review r29: Candidate Boot `recoverOnce` Contention Against Parent Updater (`F30-01`)

- **Base commit:** `dd052c6012f10fbde147f61931a50c00bec1f98b` (`main`)
- **Branch:** `r29-update-recover-review`
- **Scope:** `node/agent/**`, `node/updater/**`, `node/nodelock/**`, `node/internal/updatejournal/**`, `packaging/installer/**` (read-only), `tests/e2e/**` (read-only), `docs/updates/node-followup20-2026-11-03.md` (read-only)

---

## 1. Options

| Option | Change, with files | Owner or owners | Six invariants (1–6: kept or broken, with why) | Contract or CLI change, if any | Proof |
|---|---|---|---|---|---|
| **Option A:** Updater-side bypass (`fvpn-updater recover` exits `0` when `update.lock` is held by a live same-boot process) | `node/updater/engine.go:143-160, 194-199` (`Recover` treats `nodelock.ErrBusy` with a live same-boot holder as `nil` exit `0`) | `w-updater` | **1:** Kept only while `forwarding.blocked` is already present (`e.latch()` blocks forwarding); **broken** if `update.lock` is held before `stepBlock` writes `forwarding.blocked`.<br>**2: BROKEN.** If the candidate's boot `recoverOnce` gets exit `0` (`e.recovered = true, e.recoverBlocked = false`) and the parent `fvpn-updater` then crashes during `stepStart` or `stepCheck` (leaving `switched` or `checking` in `journal.db`), the candidate agent never runs `recover` again after `update.lock` frees.<br>**3:** Kept if `recover` later runs under a free lock, **broken** if `update.lock` was held while `journal.db` was already in `recovery_required`.<br>**4:** Kept.<br>**5:** Kept on happy path.<br>**6: BROKEN.** Operator/systemd invocation of `fvpn-updater recover` during an active update or rollback falsely reports `0` success without recovering anything. | Changes `fvpn-updater recover` CLI exit status on `UPDATE_BUSY` from non-zero to `0` (`docs/updates/node-updater.md`, `docs/handover/operator-guide.md`). | Rejected by Invariants 2 and 6 (`TestR29_Held_CrashedSwitchedAndCheckingTxnsRecoveredUnderFreeLock`). |
| **Option B (Recommended):** Agent-side deferral and bounded retry while `update.lock` is held, with explicit `ExitBusy = 3` from `fvpn-updater recover` | **Updater:** `node/updater/host.go:243-251` (`ExitBusy = 3` in `ExitCode` when `OpError{Code: apierr.UpdateBusy, Detail: "lock"}`), `node/updater/engine.go:192-245` (`e.open` inside `withLock`).<br>**Agent:** `node/agent/engine.go:534, 681, 1030-1059` (`recoverOnce` called in `Boot` and at top of `Step` while `!e.recovered`; skips spawning or defers with `e.recovered = false, e.recoverBlocked = true` when `e.updateLockHeld()` or exit `3`; runs `recover` in a single tracked background goroutine with a short synchronous fast-path wait so open crashed txns never deadlock `Boot`/`Step`). | `w-updater`, `w-agent` | **1: Kept.** While `update.lock` is held (`stepStart`/`stepCheck`), `e.recoverBlocked = true` and `e.latchBlocked = true` keep `d.Serve = false` (`recovery_required`).<br>**2: Kept.** If the parent updater crashes during `stepStart`/`stepCheck` (releasing `flock`), the next `Step` sees `!e.updateLockHeld()`, runs `fvpn-updater recover`, and completes/rolls back the open `switched`/`checking` txn before clearing `recoverBlocked`.<br>**3: Kept.** Exit `2` (`ErrMembersBlocked`) sets `e.recovered = true, e.recoverBlocked = true` permanently.<br>**4: Kept.** D-064 (`installProbeException` in `node/agent/probeexc.go:55`) and D-053 (`answerHealth` in `node/agent/attest.go:102`) do not check `recoverBlocked`.<br>**5: Kept.** On the first `Step` after `seal` removes `forwarding.blocked` and releases `update.lock`, `recoverOnce` exits `0`, clears `recoverBlocked`, and `enforce` serves without restarting the process.<br>**6: Kept.** `update.lock` stays held across `stepStart` and `stepCheck`. | `fvpn-updater` adds exit code `3` (`ExitBusy`) for live same-boot `UPDATE_BUSY: lock` (`0` = ok, `1` = error, `2` = `ErrMembersBlocked`, `3` = `UPDATE_BUSY`); update `docs/updates/node-updater.md` and `docs/handover/operator-guide.md`. No OpenAPI/JSON Schema change. | `TestR29_H1_CandidateBootUnderHeldUpdateLockDefersRecoverAndServesAfterCommit`, `TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy`, `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery`, `TestR29_Held_D053AndD064PassWhileRecoverBlockedUnderHeldUpdateLock`, `TestR29_Held_TerminalRecoveryRequiredExit2KeepsForwardingOffAfterLockRelease` |
| **Option C:** Narrow `update.lock` scope in `fvpn-updater` (release `update.lock` before `stepStart` / `stepCheck`) | `node/updater/engine.go:106-160`, `node/updater/apply.go:188-258` (drop `update.lock` across `stepStart` and `stepCheck`) | `w-updater` | **1:** Kept only via `forwarding.blocked`.<br>**2: BROKEN.** If `update.lock` is free during `stepStart`, the candidate's `recoverOnce` acquires `update.lock`, sees `ActiveTxn` in `switched`/`checking`, and enters `continueTxn` -> `stepCheck` concurrently with its own parent updater!<br>**3:** Kept.<br>**4: BROKEN.** D-064 `installProbeException` (`node/agent/probeexc.go:55`) requires `e.updateLockHeld() == true` to install the baseline probe exception.<br>**5: BROKEN** (candidate and parent updater race on `continueTxn`).<br>**6: BROKEN.** Explicitly violates Invariant 6; timer-driven `fvpn-updater run` or `packaging/installer` can acquire `update.lock` mid-transaction. | None, but breaks D-064 and single-updater exclusion. | Rejected by Invariants 2, 4, and 6 (`TestReviewR11_SameUIDChallengeOpensProbeHole` in `node/agent/review_r11_d064_test.go:104-140`). |
| **Option D1:** Updater restarts `fvpn-agent` a second time after `seal` releases `update.lock` | `node/updater/apply.go:415-522` (`seal` calls `e.Services.Start`/restart after releasing `update.lock`) | `w-updater` | **1–4:** Kept.<br>**5: BROKEN.** Violates Invariant 5 and D-053: the process that serves after `seal` is no longer the process that attested health in `stepCheck`, and every update incurs a second service restart and control re-sync.<br>**6:** Kept. | None. | Rejected by Invariant 5 and D-053. |
| **Option D2:** Updater writes a completion marker after `seal`; agent clears `recoverBlocked` from the marker without running `recover` | `node/updater/apply.go:415-522`, `node/agent/engine.go:681, 1020-1059` | `w-updater`, `w-agent` | **1:** Kept.<br>**2: BROKEN.** If the parent updater crashes during `stepStart` or `stepCheck` before writing the completion marker, the running candidate agent has `e.recovered = true, e.recoverBlocked = true` and never invokes `fvpn-updater recover` to roll back the stranded `switched`/`checking` transaction once `update.lock` frees.<br>**3–6:** Kept on happy path. | Adds a second ad-hoc IPC file alongside `forwarding.blocked` and `journal.db`. | Rejected by Invariant 2. |

---

## 2. Recommendation

- **Pick:** **Option B** (agent-side deferral and bounded retry of `recoverOnce` while `update.lock` is held, paired with `ExitBusy = 3` from `fvpn-updater` and moving `e.open` inside `withLock`).
- **Owners:** `w-updater` (`node/updater/**`) and `w-agent` (`node/agent/**`), followed by orchestrator/E2E cleanup of `tests/e2e/nightly_expected_blocked.json`.
- **Order of work:**
  1. **`w-updater` first (or in parallel if `w-agent` checks both `e.updateLockHeld()` and exit code `3`):**
     - In `node/updater/host.go:243-251`, define `const ExitBusy = 3` and return `ExitBusy` from `ExitCode(err)` when `err` is an `*OpError` with `Code == apierr.UpdateBusy && Detail == "lock"` (live same-boot holder).
     - In `node/updater/engine.go:192-245`, move `e.open(ctx)` inside `e.withLock` so contended `Recover`/`Run`/`Flush`/`Retain` calls fail on `update.lock` before opening or migrating `journal.db` (`L-1`).
  2. **`w-agent` second (can land in parallel because checking `e.updateLockHeld()` before spawning `fvpn-updater recover` already detects the held lock in-process, and checking `ee.ExitCode() == 3` handles any race between `updateLockHeld()` and `flock`):**
     - In `node/agent/engine.go:534, 681, 1030-1059`, call `e.recoverOnce(ctx)` at the start of `Step` (right before `e.latch()`) whenever `!e.recovered`.
     - In `recoverOnce`, if `e.updateLockHeld()` is true, keep `e.recovered = false` and `e.recoverBlocked = true` without spawning a child process.
     - When `!e.updateLockHeld()`, run `fvpn-updater recover` in a single tracked background goroutine (`e.recoverInFlight`, canceled/joined on `Close()`), waiting synchronously for a short fast-path slice (`250ms`–`500ms`) so normal `<80ms` runs finish in the same `Boot`/`Step` while a crashed open transaction (`StageBlocked`/`StageSwitched`/`StageChecking`) waiting up to 30s–90s on `meter.checkpoint` or `health-attest.json` never self-deadlocks or gets `SIGKILL`ed by `Boot`/`Step`'s 10s local budget (`M-1`).
     - Map exit `0` to `e.recovered = true, e.recoverBlocked = false`; exit `3` (`ExitBusy`) to `e.recovered = false, e.recoverBlocked = true` (retry on next `Step`); exit `2` (`ErrMembersBlocked`) or `1` to `e.recovered = true, e.recoverBlocked = true` (terminal fail-closed).
  3. **Orchestrator / E2E after merge:** Remove the `lease-expiry` (`F30-01`) entry from `tests/e2e/nightly_expected_blocked.json:10-16` (`L-2`) and re-run `TestE2E_Acceptance`.

---

## 3. Verification of `docs/updates/node-followup20-2026-11-03.md` & Answers to Brief Questions 1–6

### Verification of Updater 20 (`docs/updates/node-followup20-2026-11-03.md:150-311`) and `tests/e2e/nightly_expected_blocked.json` against `dd052c60`

- **Every updater-side line number and claim in `docs/updates/node-followup20-2026-11-03.md:150-311` is exact on `dd052c60`:**
  - `Engine.withLock` is at `node/updater/engine.go:143-160`; `nodelock.AcquireWithBootID(ctx, e.Layout.Lock, 5, 20*time.Millisecond, e.Node.BootID.String())` is at `engine.go:148`; `transient(apierr.UpdateBusy, lockBusyDetail(err))` is at `engine.go:151`.
  - `Engine.Recover` is at `node/updater/engine.go:194-199`; `recoverLocked` is at `engine.go:201-230`.
  - CLI `command` is at `node/updater/cmd/fvpn-updater/main.go:138-154`; `eng.Recover(ctx)` is at `main.go:143`, `fmt.Fprintf(stderr, "fvpn-updater: %v\n", runErr)` is at `main.go:152`, and `updater.ExitCode` is at `node/updater/host.go:243-251`.
  - `stepStart` (`node/updater/apply.go:188-219`), `stepCheck` (`apply.go:221-258`), `seal` (`apply.go:415-522`), and `enforce` (`node/updater/engine.go:890-926`) match the note line-for-line.
- **Three stale/inaccurate citations exist in `tests/e2e/nightly_expected_blocked.json:13-14` (`L-2`):**
  1. `node/agent/engine.go:432,884-909` and `node/agent/decide.go:143` shifted in `#429`: on `dd052c60`, `e.recoverOnce(ctx)` in `boot` is at `node/agent/engine.go:534`, `recoverOnce` is at `node/agent/engine.go:1030-1059`, and `if in.LatchBlocked || in.RecoverBlocked { add(api.ForwardingBlockReasonRecoveryRequired) }` is at `node/agent/decide.go:146-148`.
  2. `tests/e2e/member_admit_test.go:175-210` is the wrong file (`member_admit_test.go` is 180 lines ending in `readProjectedBinding`); the 90s readiness poll (`admission_counter: unavailable after 90 seconds`) is in `countedFlow` at `tests/e2e/traffic_scenario_test.go:18-38`, called from `controlStop` (`lease-expiry`) at `tests/e2e/scenarios_test.go:725`.
  3. The reason string says `"until node-restart restarts the agent"`, but in `TestE2E_Acceptance` (`tests/e2e/scenarios_test.go:71-86`), `run.restartNode(ctx)` (`node-restart`) runs at line 71 *before* `run.updater(ctx, good, bad)` at line 85 and `run.controlStop(ctx)` (`lease-expiry`) at line 86; nothing in `TestE2E_Acceptance` restarts the candidate agent after `update-good`.

### Answer to Question 1 (`fvpn-updater recover` behavior across lock and journal states)

- **Before acquiring `update.lock`:** `Recover` (`node/updater/engine.go:194-199`) calls `e.open(ctx)` (`engine.go:232-245`) *before* `e.withLock`. `e.open` calls `e.Layout.Ensure()` and `updatejournal.Open(e.Layout.Journal, ...)` (`node/internal/updatejournal/journal.go:53-112`), which opens `journal.db` RW and runs `migrate(db)` (`CREATE TABLE IF NOT EXISTS ...` and `INSERT INTO gate ... ON CONFLICT(id) DO NOTHING`). It does not mutate active transactions or touch `forwarding.blocked` before `withLock`, but it does open and migrate `journal.db` RW without holding `update.lock` (see `L-1`).
- **When `update.lock` is held by a live same-boot PID:** `nodelock.AcquireWithBootID` (`node/nodelock/lock.go:41-94, 201-227`, 5 attempts x 20ms = ~80ms) gets `EWOULDBLOCK` on `syscall.Flock(LOCK_EX|LOCK_NB)`, reads the holder JSON, sees `holderStaleReason` is `("", false)`, and returns `nodelock.ErrBusy`. `withLock` (`engine.go:151`) returns `&OpError{Code: "UPDATE_BUSY", Detail: "lock", Retry: true}`. `main.go:152` prints `fvpn-updater: updater: UPDATE_BUSY: lock\n` to stderr and exits **`1`** (`host.go:243-251`).
- **When `update.lock` on disk has a dead PID or a different boot ID:** If no process holds the kernel `flock` FD, `syscall.Flock(LOCK_EX|LOCK_NB)` (`node/nodelock/lock.go:59`) succeeds immediately, `writeHolder` (`lock.go:74, 115-131`) overwrites `update.lock` with the current PID/bootID/`/proc/<pid>/stat` start time, and `recoverLocked` runs normally. If the dead process leaked its open `flock` FD to a surviving child so `syscall.Flock` still returns `EWOULDBLOCK`, `holderStaleReason` (`lock.go:145-163`) returns `("boot id" | "pid" | "process" | "start time", true)` without unlinking the locked file, `withLock` returns `&OpError{Code: "UPDATE_BUSY", Detail: "lock <reason>", Retry: true}`, and the CLI prints `fvpn-updater: updater: UPDATE_BUSY: lock <reason>\n` to stderr and exits **`1`**.
- **When `update.lock` is free and `journal.db` has an open transaction in `switched` or `checking`:** `recoverLocked` (`node/updater/engine.go:208-212`) finds `ActiveTxn` and calls `continueTxn(ctx, txn)` (`node/updater/apply.go:20-39`), which blocks `forwarding.blocked` (`apply.go:24`) and runs `stepStart` / `stepCheck` -> `seal`.
  - If candidate health checks pass and the release is in `ServingSet`, `seal` (`apply.go:452`, `engine.go:921`) removes `forwarding.blocked` (when published) and `Recover` returns `nil` (exit **`0`**, empty stderr).
  - If candidate health checks fail and rollback to `LastGood` succeeds (`StageRolledBack`), `seal` restores `LastGood`, removes `forwarding.blocked` (when `LastGood` is published and in `ServingSet`), and returns `nil` at `apply.go:521` (exit **`0`**, empty stderr).
  - If rollback fails (`StageRecoveryRequired`), `seal` blocks `forwarding.blocked` and returns `ErrMembersBlocked` at `apply.go:518-520` (exit **`2`**, stderr `fvpn-updater: updater: member forwarding must stay blocked\n`).
- **When `update.lock` is free and `journal.db` is in `locally_succeeded`:** `ActiveTxn` returns `updatejournal.ErrNotFound`; `ensureReported` (`engine.go:841-860`) seals the job if `job.Phase != JobDone`; `BlocksForwarding` is `false`; `reapplyPublication` (`engine.go:862-888`) unblocks `forwarding.blocked` (when published and in `ServingSet`) and returns `nil` (exit **`0`**, empty stderr).
- **When `update.lock` is free and `journal.db` is in `recovery_required`:** `ActiveTxn` returns `ErrNotFound`; `ensureReported` seals if needed; `e.Journal.BlocksForwarding(ctx)` (`engine.go:222`) returns `true`; `recoverLocked` calls `e.Gate.Block(ctx)` and returns `ErrMembersBlocked` (exit **`2`**, stderr `fvpn-updater: updater: member forwarding must stay blocked\n`).

### Answer to Question 2 (`fvpn-agent` treatment of `recoverOnce` vs `latch`)

- In `boot` (`node/agent/engine.go:534-535`), `e.recoverOnce(ctx)` runs once before `e.latch()`.
- `recoverOnce` (`node/agent/engine.go:1030-1059`) sets `e.recovered = true` at line 1034 and only resets `e.recovered = false` if `ctx.Err() != nil` (line 1045). Any non-zero exit from `fvpn-updater recover` — both exit `2` (lines 1053–1055) and exit `1` on `UPDATE_BUSY` (lines 1057–1058) — sets `e.recoverBlocked = true`.
- On every `Step` (`node/agent/engine.go:681`), `e.latch()` (`engine.go:1020-1028`) re-reads `forwarding.blocked` via `meterBarrierIdentity(e.cfg.JournalDir)`. When the parent updater finishes `stepCheck` and `seal` removes `forwarding.blocked`, the next `Step`'s `e.latch()` clears `e.latchBlocked = false`.
- However, `Step` never calls `recoverOnce`, and `e.recovered` is already `true`. Thus `e.recoverBlocked` stays `true` for the rest of the process lifetime. `e.input()` (`engine.go:920`) passes `RecoverBlocked: e.recoverBlocked` into `Decide`, which at `node/agent/decide.go:146-148` executes `if in.LatchBlocked || in.RecoverBlocked { add(api.ForwardingBlockReasonRecoveryRequired) }`, keeping `d.Serve = false` with reason `recovery_required` forever.

### Answer to Question 3 & Question 4 (Option B Mechanics and Retry Details)

- **4a (Retryable vs terminal outcomes and exit code):**
  - Only `UPDATE_BUSY` caused by a live same-boot holder of `update.lock` (`OpError{Code: apierr.UpdateBusy, Detail: "lock"}`) is retryable.
  - `fvpn-updater` should define `ExitBusy = 3` in `node/updater/host.go:243-251` and return `3` when `errors.As(err, &op) && op.Code == apierr.UpdateBusy && op.Detail == "lock"`.
  - In `fvpn-agent`, `recoverOnce` first checks `e.updateLockHeld()` (`node/agent/probeexc.go:258-269`): if true, it does not even spawn `fvpn-updater recover` and keeps `e.recovered = false, e.recoverBlocked = true`. If `fvpn-updater recover` is spawned and exits `3` (`ExitBusy`), `recoverOnce` likewise keeps `e.recovered = false, e.recoverBlocked = true`.
  - Exit `0` is terminal success (`e.recovered = true, e.recoverBlocked = false`). Exit `2` (`ErrMembersBlocked`), exit `1` (corrupt journal, missing config, stale/leaked lock holder `"lock process"`, etc.), or empty `UpdaterConfig` are terminal fail-closed (`e.recovered = true, e.recoverBlocked = true`).
- **4b (What stays blocked vs unblocked while retry is pending):**
  - While `!e.recovered`, `e.recoverBlocked` is `true` (and `e.latchBlocked` is `true` while `forwarding.blocked` exists). `Decide` (`node/agent/decide.go:146-148`) adds `recovery_required` and keeps `d.Serve = false`, so member forwarding and `adoptStoredServing` (`node/agent/restart.go:57`) stay off.
  - D-064 baseline probe exception (`prepareProbe` / `installProbeException` in `node/agent/probeexc.go:44-86, 233-256`) and D-053 health attestation (`answerHealthOnBoot` / `answerHealth` in `node/agent/attest.go:86-172`) do **not** read `e.recoverBlocked`; `installProbeException` requires `e.updateLockHeld() == true`, which holds during `stepCheck`. Both remain completely unblocked (`TestR29_Held_D053AndD064PassWhileRecoverBlockedUnderHeldUpdateLock`).
- **4c (Where in `Step`/`Boot` the retry runs, execution time bound, and at-most-once guarantee):**
  - Call `e.recoverOnce(ctx)` in `boot` (`node/agent/engine.go:534`) and at the top of `Step` (`node/agent/engine.go:681`, immediately before `e.latch()`).
  - When `!e.updateLockHeld()`, spawn `fvpn-updater recover` in a single tracked background goroutine (`e.recoverInFlight`, using a context derived from `e.runCtx` and canceled/waited in `Close()`), and wait synchronously for up to `250ms`–`500ms` (or `ctx.Done()`).
  - In the normal post-update case (`locally_succeeded`) or `recovery_required`, `fvpn-updater recover` finishes in `<20ms`, so `recoverOnce` updates `e.recovered = true` within that same `Boot` or `Step`.
  - In the crashed-updater open-transaction case (`StageBlocked` waiting up to 30s for `meter.checkpoint`, or `StageSwitched`/`StageChecking` waiting up to 3x30s for `health-attest.json`), bounding the synchronous wait to `250ms`–`500ms` prevents `Boot`/`Step` from self-deadlocking or killing `recover` at the 10s local budget (`M-1`), never stalls lease renewal or watchdog heartbeats, and lets `Boot`/`Step` write `meter.checkpoint` and `health-attest.json` while the single in-flight `recover` child holds `update.lock`.
  - Once `fvpn-updater recover` exits with any non-busy code (`0`, `2`, `1`), `e.recovered` is set to `true` and `recoverOnce` never runs again for the life of the process.
- **4d (What state `Boot` computed while `recoverBlocked` was true):**
  - During `stepStart`, member forwarding was already withdrawn before `stepStop`:
    - `adoptStoredServing` (`engine.go:543`) returned `false` and `boot` installed the generation-0 baseline (`e.baselineOK = true`, `engine.go:552-554`). That is the exact required state during an update; adoption must not re-run after an update switch.
    - `ensureUsage` and `settleRetainedUsage` (`engine.go:572-575`) opened `usage.db` with `PreserveClosedEpoch: true`, started the uploader, and set `e.epochClosed = true`. When `recover` later succeeds on `Step` and `Decide` returns `d.Serve = true`, `enforce` (`engine.go:1266-1276, 1442-1445`) calls `e.meter.PrepareSampling(ctx)`, `e.resumeSampling()`, and `e.switchEpoch(ctx)`, opening the new meter epoch automatically.
    - `publishMeterEpoch()` (`node/agent/meterreset.go:294-306`) only drains kernel counter-discontinuity reports and runs on every `Step` (`engine.go:777`).
  - Therefore, no one-time `Boot` setup needs to be re-run when `recoverBlocked` clears on `Step`.

### Answer to Question 5 (Interactions with `D-136`/`r27 M-1`, `D-064`/`D-053`, and `packaging/installer`)

- **5a (`D-136` `recoveryDecision` and `r27-members-failures-review` `M-1`):**
  - `recoveryDecision` (`node/agent/recovery.go:8-19`, called at `engine.go:1139`) only runs when `e.applyError != nil`, and still feeds `in.LatchBlocked` / `in.RecoverBlocked` into `Decide(in)`, which returns `d.Serve = false` while recovery is blocked.
  - While `recoverBlocked` is `true` during `stepStart`/`stepCheck`, the candidate agent reports `provisioning_state = "applying"` (`node/agent/decide.go:341-345`). If the candidate renews its lease during that window, central `RenewTx` (`internal/leases/leases.go:440-441`) issues a 0-grant provisional lease (`members: 0`) and the agent arms D-067 (`provisionalRetryAt`, 1 heartbeat = 5s in devstack, 10s default).
  - On the first `Step` after `seal` releases `update.lock`, `recoverOnce` clears `e.recoverBlocked = false` and `e.latch()` clears `e.latchBlocked = false`. If the agent holds a 0-grant provisional lease, `Decide` transitions `d.Provisioning` to `ready` (`awaiting_member_grants`) and `flush` reports `provisioning_state = "ready"` to central on that step; on the next D-067 renewal slot (`5s` in devstack, or `10s` before `r27 M-1` fixes the `remainingStepDelay` wakeup race at `engine.go:616`), `syncLease` receives the populated lease and `enforce` applies the serving ruleset.
- **5b (`D-064` probe exception and `D-053` health attestation):**
  - Neither `prepareProbe`/`installProbeException` (`node/agent/probeexc.go:44-86, 233-256`) nor `answerHealth` (`node/agent/attest.go:102-172`) checks `e.recoverBlocked`. Keeping `e.recoverBlocked = true` while `update.lock` is held keeps member forwarding off while preserving D-064 and D-053 during `stepCheck`.
- **5c (`packaging/installer` hand-off):**
  - Both `packaging/installer/install.go:83, 255-265` and `packaging/installer/upgrade.go:53, 115-127` release `/var/lib/vpn-node/update.lock` *before* starting `fvpn-agent.service` or `fvpn-updater.service`. When `upgrade.go:127` starts `fvpn-updater.service` to consume `pending-release.json`, `fvpn-updater` acquires `update.lock` and runs the same `continueTxn` -> `stepStart` -> `stepCheck` path as a central update, which Option B handles identically.

### Answer to Question 6 (End-to-End Consequence in `TestE2E_Acceptance`)

- In `tests/e2e/scenarios_test.go:85-86`, `run.updater(ctx, good, bad)` (`update-good` and `update-bad`) is followed immediately by `run.controlStop(ctx)` (`lease-expiry`, `scenarios_test.go:723-813`), which calls `r.countedFlow(ctx, ...)` (`tests/e2e/traffic_scenario_test.go:18-38`) and polls up to **90 seconds** for `r.admissionBytes()` (`u_`/`d_` counters) and `Generation > 0`.
- With Option B in place, after `update-good` finishes `stepCheck` and releases `update.lock`, the candidate agent clears `recoverBlocked` on its next `Step` (`<= 5s` in devstack) and — even if it must complete one D-067 provisional lease renewal cycle (`+5s` to `+10s`) — installs the serving ruleset and `u_`/`d_` counters within **5–15 seconds**, well inside `countedFlow`'s **90-second** readiness bound. `lease-expiry` then passes and can be removed from `tests/e2e/nightly_expected_blocked.json`.

---

## 4. Findings by Severity

### Critical

None.

### High

#### `H-1`: Candidate `fvpn-agent` runs `recoverOnce` once at `Boot` while parent `fvpn-updater` holds `update.lock` during `stepStart`, latching `recoverBlocked = true` (`recovery_required`) for the life of the process (`F30-01`)

- **Where:** `node/agent/engine.go:534, 681, 1030-1059`; `node/agent/decide.go:146-148`; `node/updater/engine.go:148-153, 194-199`; `node/updater/apply.go:188-219`; `node/updater/host.go:243-251`; `node/updater/cmd/fvpn-updater/main.go:142-154`.
- **Scenario:** During `fvpn-updater run`, `withLock` (`node/updater/engine.go:148`) acquires `/var/lib/vpn-node/update.lock` and holds it across `stepStart` (`node/updater/apply.go:188-219`) and `stepCheck` (`apply.go:221-258`). At `apply.go:200`, `stepStart` starts the candidate `fvpn-agent`. The candidate's `boot` (`node/agent/engine.go:534`) calls `recoverOnce` (`engine.go:1030-1059`), which sets `e.recovered = true` (line 1034) and runs `fvpn-updater recover -config /etc/vpn-node/updater.json`. `fvpn-updater recover` fails to acquire `update.lock` after 5 x 20ms (`nodelock.ErrBusy` -> `OpError{Code: "UPDATE_BUSY", Detail: "lock", Retry: true}`), prints `fvpn-updater: updater: UPDATE_BUSY: lock` to stderr, and exits `1` (`ExitCode` in `node/updater/host.go:243-251`). `recoverOnce` sets `e.recoverBlocked = true` (line 1057). During `stepCheck`, the candidate still answers the D-053 health challenge (`answerHealth` does not check `recoverBlocked`), so the updater seals the transaction (`StageLocallySucceeded`), removes `forwarding.blocked`, and releases `update.lock`. On subsequent `Step` calls (`engine.go:681`), `e.latch()` clears `e.latchBlocked = false`, but `recoverOnce` is never called again and `e.recovered` remains `true`, leaving `e.recoverBlocked = true` (`recovery_required`, `d.Serve = false`) until the agent process is externally restarted.
- **Proof:** `TestR29_H1_CandidateBootUnderHeldUpdateLockDefersRecoverAndServesAfterCommit` in `node/agent/review_r29_test.go`; `TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy` in `node/updater/review_r29_test.go`.
- **Narrowest fix and owner:**
  - **`w-updater` (`node/updater/host.go:243-251`):** Define `const ExitBusy = 3` and return `ExitBusy` from `ExitCode(err)` when `errors.As(err, &op) && op.Code == apierr.UpdateBusy && op.Detail == "lock"`.
  - **`w-agent` (`node/agent/engine.go:534, 681, 1030-1059`):** Call `e.recoverOnce(ctx)` at the start of `Step` (before `e.latch()`) whenever `!e.recovered`. In `recoverOnce`, if `e.updateLockHeld()` is true or `fvpn-updater recover` exits `3` (`ExitBusy`), set `e.recovered = false` and `e.recoverBlocked = true` so `Step` retries after `update.lock` is released. When `fvpn-updater recover` exits `0`, set `e.recovered = true, e.recoverBlocked = false`; when it exits `2` or `1`, set `e.recovered = true, e.recoverBlocked = true`.

### Medium

#### `M-1`: Synchronous `recoverOnce` in `Boot` and `Step` self-deadlocks and hits the 10s local budget (`SIGKILL`) when recovering a crashed open updater transaction (`StageBlocked`, `StageSwitched`, or `StageChecking`)

- **Where:** `node/agent/engine.go:439, 534, 574, 587, 673, 780, 1030-1059`; `node/updater/meter_barrier.go:22, 175-225`; `node/updater/apply.go:200, 232, 245`; `node/updater/attest.go:25, 147-170`.
- **Scenario:** When `fvpn-updater` crashes mid-transaction (releasing `update.lock`) and `fvpn-agent` runs `fvpn-updater recover` in `recoverOnce`, `recoverLocked` calls `continueTxn`:
  1. If the open transaction is in `StagePrepared` or `StageBlocked`, `stepSample` -> `waitMeter` (`node/updater/meter_barrier.go:175-225`) polls up to `MeterCheckpointWait = 30s` for `fvpn-agent` to publish `meter.checkpoint` for the `forwarding.blocked` barrier UUID. In `boot` (`node/agent/engine.go:534-575`), `recoverOnce` runs at line 534 *before* `ensureUsage`/`handoff` at line 574 writes `meter.checkpoint`.
  2. If the open transaction is in `StageSwitched` or `StageChecking`, `stepCheck` (`node/updater/apply.go:232-245`) writes a fresh `health-challenge.json` and polls in `NodeChecks.poll` (`node/updater/attest.go:147-170`) for up to `HealthAttestWait = 30s` per attempt (3 attempts = 90s) waiting for `fvpn-agent` to write `health-attest.json` in `answerHealthOnBoot` (line 587) or `Step`'s `answerHealth` (line 780).
  Because `recoverOnce` blocks synchronously in `exec.CommandContext(ctx, ...).Run()` under `Boot`'s `bootLocalBudget = 10s` (`engine.go:439`) or `Step`'s `controlPassTimeout() + 5s = 10s` (`engine.go:673`), `fvpn-agent` cannot reach `ensureUsage` or `answerHealth` while `recover` is waiting for it, and after 10s `ctx` expires and `SIGKILL`s `fvpn-updater recover` before the 30s updater timeout can even trigger rollback (failing `Boot` with `ErrBoot: local deadline: context deadline exceeded` or stalling `Step` for 10s every tick).
- **Proof:** `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery` in `node/agent/review_r29_test.go`.
- **Narrowest fix and owner:** **`w-agent` (`node/agent/engine.go:322-363, 534, 681, 1030-1059`):** Run `fvpn-updater recover` in a single tracked background goroutine (`e.recoverInFlight`, canceled and joined in `Close()`), waiting synchronously in `Boot`/`Step` for a short fast-path slice (`250ms`–`500ms`) so normal `<80ms` recoveries complete immediately while an open transaction waiting on `meter.checkpoint` or `health-attest.json` continues in the background with `e.recoverBlocked = true` (`d.Serve = false`) while `Boot` and `Step` publish `meter.checkpoint`, refresh heartbeats/leases, and answer `health-challenge.json`.

### Low

#### `L-1`: `Engine.Recover` (and `Run`/`Flush`/`Retain`) opens and migrates `journal.db` read-write before acquiring `update.lock`

- **Where:** `node/updater/engine.go:106-110, 192-245`; `node/internal/updatejournal/journal.go:53-112`.
- **Scenario:** `Recover`, `Run`, `Flush`, and `Retain` call `e.open(ctx)` before `e.withLock(ctx, ...)`. `e.open` calls `updatejournal.Open`, which opens `journal.db` RW and executes `migrate(db)` (`BEGIN`...`COMMIT` write transaction) before checking `update.lock`. When `Recover` or `Run` is called while another process holds `update.lock` and writes to `journal.db`, the contended caller unnecessarily opens `journal.db` RW and contends on SQLite before failing with `UPDATE_BUSY: lock`.
- **Proof:** `TestR29_L1_ContendedRecoverChecksUpdateLockBeforeOpeningJournal` in `node/updater/review_r29_test.go`.
- **Narrowest fix and owner:** **`w-updater` (`node/updater/engine.go:106-245`):** Move `e.open(ctx)` inside `e.withLock` (after `nodelock.AcquireWithBootID` succeeds) so contended callers fail fast on `update.lock` without opening `journal.db`.

#### `L-2`: Stale line numbers, wrong test file citation, and inaccurate `node-restart` ordering note in `tests/e2e/nightly_expected_blocked.json` entry for `lease-expiry` (`F30-01`)

- **Where:** `tests/e2e/nightly_expected_blocked.json:10-16`.
- **Scenario:** The quarantine entry for `lease-expiry` (`F30-01`) cites `node/agent/engine.go:432,884-909` and `node/agent/decide.go:143` (now `node/agent/engine.go:534,1030-1059` and `node/agent/decide.go:146-148`), cites `tests/e2e/member_admit_test.go:175-210` for the 90s `admission_counter` poll (actually `countedFlow` in `tests/e2e/traffic_scenario_test.go:18-38`, called from `tests/e2e/scenarios_test.go:725`), and says `recoverBlocked=true` persists `"until node-restart restarts the agent"` (whereas in `tests/e2e/scenarios_test.go:71-86`, `node-restart` runs at line 71 before `updater` at line 85 and `lease-expiry` at line 86).
- **Proof:** Direct inspection of `node/agent/engine.go:534,1030-1059`, `node/agent/decide.go:146-148`, `tests/e2e/traffic_scenario_test.go:18-38`, and `tests/e2e/scenarios_test.go:71-86`.
- **Narrowest fix and owner:** **Orchestrator / E2E lane (`tests/e2e/nightly_expected_blocked.json`):** Remove the `lease-expiry` (`F30-01`) entry once `H-1` is fixed by `w-updater` and `w-agent`.

---

## 5. Routing

- **`w-updater`:** Implement `H-1` (updater half: add `ExitBusy = 3` in `node/updater/host.go:243-251` for `OpError{Code: apierr.UpdateBusy, Detail: "lock"}` and update `docs/updates/node-updater.md` / `docs/handover/operator-guide.md`) and `L-1` (move `e.open(ctx)` inside `e.withLock` in `node/updater/engine.go:106-245`), turning `TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy` and `TestR29_L1_ContendedRecoverChecksUpdateLockBeforeOpeningJournal` green.
- **`w-agent`:** Implement `H-1` (agent half: retry `recoverOnce` on `Step` while `!e.recovered`, deferring with `e.recovered = false, e.recoverBlocked = true` when `e.updateLockHeld()` is true or `fvpn-updater recover` exits `3`) and `M-1` (run `fvpn-updater recover` in a single tracked background goroutine with a short synchronous fast-path wait in `Boot`/`Step`, joined on `Close()`), turning `TestR29_H1_CandidateBootUnderHeldUpdateLockDefersRecoverAndServesAfterCommit` and `TestR29_M1_RecoverOnceDoesNotSelfDeadlockOrKillOpenTransactionRecovery` green.
- **Orchestrator / E2E lane:** After `w-updater` and `w-agent` merge, remove `lease-expiry` (`F30-01`) from `tests/e2e/nightly_expected_blocked.json` (`L-2`) and verify `TestE2E_Acceptance`.
- **Decision for Ted:** None needed unless you prefer not to add exit code `3` (`ExitBusy`) to `fvpn-updater`'s CLI contract (recommendation: approve Option B with `ExitBusy = 3`, as it is node-local between `fvpn-updater` and `fvpn-agent` and avoids stderr text parsing).

---

## 6. What Held

- **D-053 health attestation and D-064 baseline probe exception succeed while `recoverBlocked` and `latchBlocked` are true under a held `update.lock`, without enabling member forwarding:** proved by `TestR29_Held_D053AndD064PassWhileRecoverBlockedUnderHeldUpdateLock` in `node/agent/review_r29_test.go`.
- **Terminal `fvpn-updater recover` exit `2` (`ErrMembersBlocked` / `recovery_required`) after `update.lock` release keeps `recoverBlocked = true` and member forwarding off:** proved by `TestR29_Held_TerminalRecoveryRequiredExit2KeepsForwardingOffAfterLockRelease` in `node/agent/review_r29_test.go`.
- **A contended `Engine.Recover` call during `stepStart`/`stepCheck` returns `UPDATE_BUSY: lock` without mutating the active transaction stage or unblocking `forwarding.blocked`, and succeeds with exit `0` after `seal` releases `update.lock`:** proved by `TestR29_Held_ContendedRecoverDuringStepStartAndCheckDoesNotMutateTxnOrGate` in `node/updater/review_r29_test.go`.
- **When `fvpn-updater` crashes in `StageSwitched` (releasing `update.lock`), `Engine.Recover` reclaims `update.lock`, drives rollback to `LastGood` when candidate health fails, and unblocks `forwarding.blocked` only after rollback completes:** proved by `TestR29_Held_CrashedSwitchedAndCheckingTxnsRecoveredUnderFreeLock` in `node/updater/review_r29_test.go`.

---

## 7. Not Examined

- Live Docker/netlab execution of `TestE2E_Acceptance` (`tests/e2e` was inspected statically alongside the `followup29` run artifacts in `docs/e2e/runs/2026-11-03/followup29/`).
- Central control-plane services (`apps/control`, `apps/worker`, `apps/portal`), except for read-only verification of how `provisioning_state = "applying"` interacts with provisional 0-grant leases (`r27-members-failures-review` `M-1`).
