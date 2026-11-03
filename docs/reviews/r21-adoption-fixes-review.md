# Adversarial Review: `agent-core` Follow-up 37 (`05b2c2f2..7b3d7bf8`)

**Reviewed range:** `05b2c2f2f17da4cc1e307d342c44a14f66ed494a..7b3d7bf87c0919c74bddf6d6fbf968539b191dda` (`c43fcc4e`, `7b3d7bf8`)
**Scope:** Same-boot serving adoption refusal, retained-usage settlement (`H-1`), `Boot` under `SIGTERM` (`M-1`), early clock-sample and `sameRestartSets` ordering (`M-2`), refused-adoption `applyError` clearing and `shutdownUsage` non-serving heartbeat (`M-3`), and `sameRestartSets` / `installed.Generation` guards (`L-1`). (`H-2` belongs to `netlab 27` and is out of scope here.)

---

## 1. Findings by severity

### Critical

None.

### High

#### `H-1` — Cross-member / cross-revision billing misattribution after a crash between `plane.Apply` and `retained_applied` persistence (`MTR03`, `MTR04`)

- **Where:** `node/agent/engine.go:684-698, 1128-1182`, `node/agent/usage.go:257-264`, `node/internal/meter/delta.go:453-499`.
- **Scenario:**
  Follow-up 37 (`docs/node/followup37-r20-adoption-review-2026-11-03.md:57-59`) claims: *"A crash between `plane.Apply` and the `retained_applied` write (r20 H-2 window B) can leave counters already sampled under the newer bindings; the next refusal then records a gap, not a misattribution."*
  That claim holds only if the background sampler happened to complete a sample in the few milliseconds between `e.plane.Apply` (`engine.go:1128`) and the crash before `e.store.SetMeta(ctx, "retained_applied", ...)` (`engine.go:1178`). Whenever no sample committed in that window (or when `e.plane.Apply` crashes after `NftBatch` / `releaseDeadlineGuard`, or `e.verifyServingModel` at line 1174 times out / fails and the process dies before `withdraw` completes):
  1. Process 1 serves generation 2 where `DeviceID = D` is bound to `(MemberID = M1, BindingRevision = 1)` and samples `u_<D> = 4096` into `usage.db` (`head.Counters["u_<D>"] = {Bytes: 4096, MemberID: M1, BindingRevision: 1}`).
  2. Generation 3 arrives where `DeviceID = D` is bound to `(MemberID = M2, BindingRevision = 2)` (device reassigned to another member or a new binding revision).
  3. Process 1 runs `e.plane.Apply` (`engine.go:1128`), which atomically replaces the kernel ruleset with generation 3 (`fvpn_generation = 3`) while preserving the kernel counter `u_<D>`, and crashes before `e.store.SetMeta(ctx, "retained_applied", ...)` at line 1178.
  4. While generation 3 is active in the kernel (before the crash and during the restart gap before Process 2's `Boot` installs `Baseline`), `4096 B` of traffic are forwarded for Member `M2` (`BindingRevision = 2`), advancing `u_<D>` from `4096` to `8192`.
  5. Process 2 starts on the same boot (`bootID`). `loadRuntime` (`engine.go:684-698`) loads the stale `retained_applied` from generation 2 and calls `e.setMeterBindings(retained.Model.Bindings)` (`MemberID = M1, BindingRevision = 1`).
  6. `adoptStoredServing` returns `false` (either early because `e.appliedGen == 2 != desired.ConfigGeneration == 3`, or in `verifyServingModel` because the kernel has `fvpn_generation = 3 != 2`), and `Boot` installs `Baseline` (preserving `u_<D> = 8192`).
  7. `Boot` calls `e.settleRetainedUsage(ctx)` (`engine.go:482`) $\to$ `e.meter.Sample(ctx)`. Because both `head.Counters["u_<D>"]` and `obs.Bindings` carry `(MemberID = M1, BindingRevision = 1)`, `attribute()` (`node/internal/meter/delta.go:490`) sees no conflict and emits a **`UsageQualityComplete`** event charging **Member `M1` (`BindingRevision = 1`)** for the `4096 B` that the kernel actually forwarded for **Member `M2` (`BindingRevision = 2`)** under generation 3.
- **Proof:** `TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember` (`node/agent/review_r21_test.go`):
  ```bash
  CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE="$(go env GOCACHE)" go test -count=1 -v -run '^TestR21_H1_' ./node/agent
  ```
  ```text
  === RUN   TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember
      review_r21_test.go:181: H-1: Member M1 (rev 1) was billed 8192 bytes as UsageQualityComplete (want 4096; the extra 4096 bytes belonged to Member M2 rev 2 under generation 3)
  --- FAIL: TestR21_H1_CrashBetweenApplyAndRetainedAppliedMisattributesToPreviousMember (1.25s)
  ```
- **Fix (`agent-core`):** Persist a pre-install intent (or update the durable meter binding / generation marker before `e.plane.Apply` opens forwarding for a new generation, and verify the live kernel `fvpn_generation` before `Boot`'s `Baseline` overwrites it) so `settleRetainedUsage` only attributes under `retained_applied` when the kernel ruleset and durable state provably belong to `retained_applied.Model.Generation`, and otherwise closes the epoch as a gap.

#### `H-2` — Cross-member / cross-revision billing misattribution when `settleRetainedUsage` fails before a rebinding `Apply` (`MTR03`, `MTR04`)

- **Where:** `node/agent/usage.go:257-264`, `node/agent/engine.go:481-482, 697, 1152-1157`, `node/internal/meter/delta.go:472-498`.
- **Scenario:**
  Follow-up 37 (`docs/node/followup37-r20-adoption-review-2026-11-03.md:54-57`, `node/agent/usage.go:253-256`) claims: *"a settlement sample that fails (clock hold, source failure) leaves the delta pending under the same bindings; a non-serving hand-off still attributes it, and a rebinding Apply records an honest gap rather than a misattribution."*
  That claim is false whenever `head.Counters["u_<D>"]` in `usage.db` does not yet carry the retained `(MemberID = M1, BindingRevision = 1)` binding — which occurs whenever Process 1 opened the baseline epoch in `ensureUsage` before `Apply` (so `head.Counters["u_<D>"]` was either absent or recorded with `MemberID = uuid.Nil, BindingRevision = 0`), applied generation 2 (`DeviceID = D, MemberID = M1, BindingRevision = 1`), forwarded `4096 B` (`u_<D> = 4096`), and crashed before its first post-`Apply` `Sample` committed to `usage.db`:
  1. Process 2 boots on the same boot (`bootID`) and refuses adoption. `loadRuntime` restores `retained_applied` (`MemberID = M1, BindingRevision = 1`) into `e.meterSnap`.
  2. `ensureUsage` opens the meter (`planContinuity` returns `noop: true`, leaving `head.Counters["u_<D>"]` with `MemberID = uuid.Nil, BindingRevision = 0`).
  3. `settleRetainedUsage` calls `e.meter.Sample(ctx)`. When that single sample returns `meter.ErrHold` (e.g. wall clock stepped back by `< 10s`, which does not set `m.sourceFailed` and does not block `Step`), `settleRetainedUsage` logs a warning and returns without stamping `(M1, rev 1)` into `head.Counters` and without calling `e.readMeter(ctx)`.
  4. On Process 2's first `Step`, `enforce` runs `e.plane.Apply` for generation 3 where `DeviceID = D` is bound to `(MemberID = M2, BindingRevision = 2)`, and `engine.go:1156` (`e.setMeterBindings(model.Bindings)`) overwrites `e.meterSnap` with `(MemberID = M2, BindingRevision = 2)`.
  5. On the next `Sample` once the clock advances, `attribute()` (`node/internal/meter/delta.go:472-498`) sees `a.hadOld == false` (`p.MemberID == uuid.Nil && p.BindingRevision == 0`) and `cur = (M2, rev 2)`. Because `a.hadOld` is `false`, `attribute()` sees no mapping conflict and emits a **`UsageQualityComplete`** event charging **Member `M2` (`BindingRevision = 2`)** for the `4096 B` consumed by **Member `M1` (`BindingRevision = 1`)** under generation 2.
- **Proof:** `TestR21_H2_FailedSettlementThenRebindingApplyMisattributesToNewMember` (`node/agent/review_r21_test.go`):
  ```bash
  CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE="$(go env GOCACHE)" go test -count=1 -v -run '^TestR21_H2_' ./node/agent
  ```
  ```text
  === RUN   TestR21_H2_FailedSettlementThenRebindingApplyMisattributesToNewMember
      review_r21_test.go:294: H-2: Member M2 (rev 2) was billed 4096 bytes as UsageQualityComplete for Member M1's (rev 1) traffic (byMemberRev=map[{[1 154 0 0 0 0 112 0 128 0 0 0 0 0 0 178] 2}:4096] gaps=0)
  --- FAIL: TestR21_H2_FailedSettlementThenRebindingApplyMisattributesToNewMember (0.16s)
  ```
- **Fix (`agent-core`):** Do not allow `enforce` to overwrite `e.meterSnap` with a new generation's bindings (or admit forwarding after a refused adoption) while an open epoch still has unsettled retained-generation counters, and call `e.readMeter(ctx)` after `settleRetainedUsage(ctx)` in `Boot`.

---

### Medium

#### `M-1` — Refused adoption leaves `head.Counters` stamped with the retired `BindingRevision` in an open epoch across `Baseline`, dropping the first traffic interval of a rebinding `Apply` into a false `mapping conflict` gap (`MTR03`, `AT13`)

- **Where:** `node/agent/engine.go:481-482, 697, 1101-1157, 1261`, `node/agent/usage.go:257-264`, `node/internal/meter/delta.go:183-230, 472-493`.
- **Scenario:**
  When `Boot` refuses adoption and installs `Baseline` (`engine.go:469`), member forwarding is stopped in the kernel. However, `Boot` leaves an open epoch in `usage.db` whose `head.Counters["u_<D>"]` is stamped with the retired `retained_applied` binding `(MemberID = M1, BindingRevision = 1)`:
  - **Open-epoch path:** `settleRetainedUsage(ctx)` calls `e.meter.Sample(ctx)` (`usage.go:261`), which attributes the `4096 B` restart-gap delta (`u_<D> = 8192`) to `(M1, rev 1)` and writes `head.Counters["u_<D>"] = {Bytes: 8192, MemberID: M1, BindingRevision: 1}` while leaving `head.OpenEpoch` open and `e.finalDone == false`.
  - **Closed-epoch path:** Even when the previous process closed its epoch cleanly with `FinalSample` (`head.OpenEpoch == uuid.Nil`), `loadRuntime` now calls `e.setMeterBindings(retained.Model.Bindings)` (`engine.go:697`) *before* `ensureUsage` opens the meter; `meter.New` $\to$ `reconcile` $\to$ `planBaseline` (`delta.go:183-230`) then opens the new baseline epoch with `head.Counters["u_<D>"]` pre-stamped with the retired `(M1, rev 1)` from `retained_applied`.
  On the first `Step`, `enforce` runs `e.plane.Apply` for generation 3 where `DeviceID = D` has `BindingRevision = 2` (or a new `MemberID`). Because `e.finalDone` is `false`, `PrepareSampling` (`engine.go:1101-1110`) is skipped. As soon as `e.plane.Apply` installs generation 3 in the kernel (`releaseDeadlineGuard`), packets are forwarded under generation 3 (`BindingRevision = 2`), advancing `u_<D>` (e.g. from `8192` to `9216`) before `enforce` finishes post-install verification/SQLite persistence and reaches `resumeSampling()` at line 1261 (and before the asynchronous sampler's next tick). On the first sample after traffic flows under generation 3, `attribute()` (`delta.go:490`) compares `head.Counters["u_<D>"]` (`BindingRevision = 1`) against `obs.Bindings["u_<D>"]` (`BindingRevision = 2`), declares a false `mapping conflict`, and drops the generation 3 bytes into a `UsageQualityGap` — even though `Baseline` cleanly separated generation 2 from generation 3 and 100% of the post-`Baseline` bytes belonged to `BindingRevision = 2`. (`TestR20_H1_RefusedAdoptionAttributesPendingDeltaOnce/rebinding_first_apply` missed this because it never advanced the counter or sampled under generation 3 after `stepOK`.)
- **Proof:** `TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind` (`node/agent/review_r21_test.go`):
  ```bash
  CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE="$(go env GOCACHE)" go test -count=1 -v -run '^TestR21_M1_' ./node/agent
  ```
  ```text
  === RUN   TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind
  === RUN   TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind/open_epoch_settled
      review_r21_test.go:382: M-1 (open_epoch_settled): post-Baseline traffic under generation 3 (rev 2) was dropped into a mapping-conflict gap: byRevision=map[1:8192] gaps=[{EventSeq:3 ... Quality:gap Records:[] SchemaVersion:1}] (want rev1=8192, rev2=1024, gaps=0)
  === RUN   TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind/closed_epoch_reopened_by_load_runtime
      review_r21_test.go:382: M-1 (closed_epoch_reopened_by_load_runtime): post-Baseline traffic under generation 3 (rev 2) was dropped into a mapping-conflict gap: byRevision=map[1:4096] gaps=[{EventSeq:2 ... Quality:gap Records:[] SchemaVersion:1} {EventSeq:3 ... Quality:gap Records:[] SchemaVersion:1}] (want rev1=8192, rev2=1024, gaps=0)
  --- FAIL: TestR21_M1_RefusedAdoptionLeavesOpenRetainedBindingCausingFalseConflictOnRebind (0.31s)
  ```
- **Fix (`agent-core`):** After `Boot` refuses adoption and installs `Baseline`, settle any open epoch with `FinalSample` (closing the retired generation's epoch at the post-`Baseline` counter reading and setting `e.finalDone = true`) and clear `e.setMeterBindings(nil)` so `planBaseline` / `PrepareSampling` never stamps retired `retained_applied` bindings onto the new post-`Baseline` epoch.

#### `M-2` — `Boot` under `SIGTERM` starts uncanceled background `usageupload` HTTP requests, fails `recoverOnce` on the canceled caller context, stacks two `10s` `bootLocal` budgets (`23s` with `Close`), and exits `0` when `Baseline` or `loadRuntime` fails (`UPD04`, `ISO01`, `MTR04`)

- **Where:** `node/agent/engine.go:422-493, 925-950`, `node/agent/usage.go:188-195, 209`, `node/agent/cmd/fvpn-agent/main.go:121-125`.
- **Scenario:**
  Follow-up 37 (`engine.go:415-422`, `docs/node/followup37-r20-adoption-review-2026-11-03.md:59-67`) states that `bootLocalBudget` (`10s`) plus `Close` (`3s`) stays within the `13s` cooperative stop budget under `TimeoutStopSec=15s`, that `SIGTERM` during `Boot` ends control I/O without breaking local completion, and that `fvpn-agent run` exits `0` only after establishing a verified baseline and truthful heartbeat. Four concrete paths violate this contract:
  1. **Uncanceled outbound `POST /api/v1/agent/usage-batches` after `SIGTERM`:** In `Boot` (`engine.go:481`), `e.ensureUsage(ctx)` runs under `ctx = bootLocal(caller)` even when `caller.Err() != nil`. `ensureUsage` (`usage.go:189-194`) unconditionally spawns `go up.Run(e.loopCtx)` where `e.loopCtx` is `context.WithCancel(context.Background())` (not bound to `caller`). When `usage.db` has due events (including the event just emitted by `settleRetainedUsage`), `up.Run(e.loopCtx)` immediately starts an uncanceled control-plane HTTP `POST /api/v1/agent/usage-batches` after `SIGTERM` until `Close()` cancels `e.loopCtx`. (Similarly, `openUsage(ctx)` at `usage.go:209` passes `ctx = bootLocal(caller)` to `nodelock.Acquire`, ignoring `SIGTERM` during lock wait.)
  2. **`e.recoverOnce(caller)` fails on `SIGTERM` and latches `e.recoverBlocked = true`:** At `engine.go:449`, `e.recoverOnce(caller)` passes `caller` instead of the `bootLocal` context to `exec.CommandContext(ctx, e.cfg.UpdaterRecover, "recover", ...)` (`engine.go:937`). When `UpdaterRecover` is configured (the production default) and `caller` is already canceled when `Boot` reaches line 449, `cmd.Run()` fails immediately with `context.Canceled` and sets `e.recoverBlocked = true`, causing `Boot` to write its heartbeat with `Reason: "recovery_required"` on an ordinary `SIGTERM`.
  3. **Stacked `10s + 10s + 3s = 23s` budgets exceed `TimeoutStopSec=15s`:** Unlike `Step` (which allocates a single `localBudget` at entry), `Boot` allocates two independent `10s` `bootLocal(caller)` budgets back-to-back — `load` at `engine.go:440` and `ctx` at `engine.go:463` — plus `Close()`'s `3s` budget at `engine.go:316` (and if `SIGTERM` cancels `verifyServingModel` or `checkMemberPath` inside `adoptStoredServing(caller)`, `localContext(caller)` has no `localBudgetKey{}` so `cleanupBudget(caller)` / `rejectMemberPath` creates another `5s` `WithoutCancel` budget and runs `e.plane.Baseline` twice).
  4. **Exit code `0` when `Baseline` or `loadRuntime` fails during a `SIGTERM`-canceled `Boot`:** At `engine.go:469-492`, if `e.plane.Baseline(ctx, dp)` fails (`e.baselineOK = false`, leaving the previous process's unverified serving ruleset forwarding packets in the kernel), `Boot` still returns `nil`, `Run` returns `ctx.Err()` on the first `Step`, and `main.go:121` (`if err := eng.Run(ctx); err != nil && ctx.Err() == nil`) exits **`0`**. Likewise, if `e.loadRuntime(load)` fails at `engine.go:441-445` while `ctx.Err() != nil`, `Boot` returns the store error without calling `e.beat()` and `main.go:121` ignores the error because `ctx.Err() != nil` and exits **`0`**.
- **Proof:** `TestR21_M2_BootUnderSIGTERMUncanceledUploadAndRecoverBlocked` (`node/agent/review_r21_test.go`) and `TestR21_M2_RunExitsZeroWhenCanceledBootFailsBaselineOrLoadRuntime` (`node/agent/cmd/fvpn-agent/review_r21_test.go`):
  ```bash
  CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE="$(go env GOCACHE)" go test -count=1 -v -run '^TestR21_M2_' ./node/agent ./node/agent/cmd/fvpn-agent
  ```
  ```text
  === RUN   TestR21_M2_BootUnderSIGTERMUncanceledUploadAndRecoverBlocked
      review_r21_test.go:444: M-2: Boot under SIGTERM sent 2 uncanceled usage batch HTTP POST(s) after SIGTERM and latched recoverBlocked=true heartbeat.Reason="recovery_required"
  --- FAIL: TestR21_M2_BootUnderSIGTERMUncanceledUploadAndRecoverBlocked (0.17s)
  === RUN   TestR21_M2_RunExitsZeroWhenCanceledBootFailsBaselineOrLoadRuntime
      review_r21_test.go:73: M-2: runWithOpen returned exit code 0 when Boot failed with boot failed to load runtime or establish fail-closed baseline during SIGTERM; want exit code 1
  --- FAIL: TestR21_M2_RunExitsZeroWhenCanceledBootFailsBaselineOrLoadRuntime (0.00s)
  ```
- **Fix (`agent-core`):** Share a single `bootLocal(caller)` budget across `Boot` (including `recoverOnce` and `adoptStoredServing`'s local cleanup), do not start background `up.Run` loops in `ensureUsage` when `controlContext(ctx).Err() != nil` (and use `controlContext(ctx)` for `openUsage`'s `nodelock.Acquire`), and return a non-zero exit error from `Boot` / `main.go` whenever `Boot` fails to load runtime state or fails to establish `e.baselineOK` (even when `ctx.Err() != nil`).

---

### Low

#### `L-1` — Refused adoption in `checkMemberPath` clears `e.applyError` in `Boot` without clearing `e.memberPathPending`, leaving `heartbeat.Reason` stuck on `"member_path_unconfigured"` across non-serving generations (`ISO01`)

- **Where:** `node/agent/restart.go:100`, `node/agent/memberpath.go:135-144`, `node/agent/engine.go:473-480, 1348-1350`.
- **Scenario:**
  1. Process 2 runs `adoptStoredServing(caller)`. `verifyServingModel` (`restart.go:97`) succeeds, and `e.checkMemberPath(ctx, d.Model)` (`restart.go:100`) fails (either because the member interface topology drifted during the restart gap or because `SIGTERM` canceled `caller` during `checkMemberPath`).
  2. `checkMemberPath` $\to$ `readMemberPath` $\to$ `rejectMemberPath` (`memberpath.go:135-144`) sets both `e.memberPathPending = true` and `e.applyError = &api.NodeError{Code: HEALTH_CHECK_FAILED, Message: "member_path_unconfigured"}`.
  3. `adoptStoredServing` returns `false`. `Boot` (`engine.go:473-480`) installs `Baseline` and clears `e.applyError = nil` (`!e.quarantinePending`), **without clearing `e.memberPathPending`**.
  4. On subsequent non-serving steps (e.g. when central transitions the node to `PublicationIntent: unpublished` at generation 3), `enforce` takes the `!d.Serve` branch (`engine.go:1023-1035`), which acknowledges generation 3 with `Safety: ok`, `Reasons: ["not_published"]`, and `Errors: []`, but never clears `e.memberPathPending`.
  5. Because `e.beat()` (`engine.go:1348-1350`) checks `if e.memberPathPending && !e.quarantinePending { reason = "member_path_unconfigured" }`, every heartbeat written by `Boot` and by subsequent non-serving `Step`s reports `Reason: "member_path_unconfigured"`, contradicting `e.decision.Reasons` and `e.decision.Safety`.
- **Proof:** `TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending` (`node/agent/review_r21_test.go`):
  ```bash
  CGO_ENABLED=0 GOTOOLCHAIN=local GOCACHE="$(go env GOCACHE)" go test -count=1 -v -run '^TestR21_L1_' ./node/agent
  ```
  ```text
  === RUN   TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending
      review_r21_test.go:514: L-1: orphan memberPathPending=true after Boot cleared applyError=<nil>; unpublished generation 3 heartbeat.Reason="member_path_unconfigured" (decision.Reasons=[not_published], decision.Safety=passed)
  --- FAIL: TestR21_L1_RefusedAdoptionMemberPathLeavesOrphanMemberPathPending (0.14s)
  ```
- **Fix (`agent-core`):** When `Boot` clears the refused adoption's `e.applyError` after a verified `Baseline` (`engine.go:473-480`, `!e.quarantinePending`), also clear `e.memberPathPending = false` if it was not already set before `adoptStoredServing`.

---

## 2. Merge recommendation

**DO NOT MERGE — Return to `agent-core` (`H-1`, `H-2`, `M-1`, `M-2`, `L-1`).**

Follow-up 37 cleanly resolved `r20`'s `M-2` ordering and `planeLocked` self-contention, `L-1` set/generation guards, `M-3` `shutdownUsage` heartbeat update, and the single-process happy path of `H-1` and `M-1`. However, the new retained-usage settlement path in `loadRuntime` + `settleRetainedUsage` introduces two High-severity cross-member / cross-revision billing misattribution bugs (`H-1` when a crash occurs between `plane.Apply` and `retained_applied` persistence without an intervening sample, and `H-2` when `settleRetainedUsage` fails on a clock hold after an unsampled `Apply` and the first `Step` rebinds the device) and one Medium-severity false `mapping conflict` gap (`M-1` because `Boot` leaves `head.Counters` stamped with the retired `BindingRevision` in an open epoch across `Baseline`, dropping the first traffic interval of a rebinding `Apply`). In addition, `Boot` under `SIGTERM` (`M-2`) still spawns an uncanceled background `usageupload` HTTP loop after `SIGTERM`, runs `recoverOnce` under the canceled `caller` context (latching `recoverBlocked = true`), stacks two `10s` `bootLocal` budgets (`23s` with `Close`), and exits `0` in `cmd/fvpn-agent` when `Baseline` or `loadRuntime` fails during a `SIGTERM`-canceled `Boot`; and `Boot`'s `M-3` cleanup (`L-1`) clears `e.applyError` without clearing `e.memberPathPending` when `adoptStoredServing` fails in `checkMemberPath`.

---

## 3. What held

1. **Attack Question 1 — Completeness against `r20`, `M-3` proof reachability, and `L-1` rejections:**
   - **Held:** Follow-up 37 addressed all five `agent-core` findings from `r20` (`H-1`, `M-1`, `M-2`, `M-3`, `L-1`). The state in `TestR20_M3_RefusedAdoptionErrorDoesNotBlockNonServingAck` (a fresh `Serving: true` heartbeat on disk while the live kernel ruleset is at generation 0 / mismatched, without rewinding the monotonic clock) is production-reachable both when an external actor replaces the ruleset during the restart gap within the 30s heartbeat staleness window and when the previous agent process crashes in `Step`/`withdraw` or `shutdownUsage`/`handoff` immediately after `e.baseline(ctx)` replaces the kernel ruleset and before `e.beat()` updates `HeartbeatPath`. Both `L-1` rejections are sound and both guards (`sameRestartSets` rejecting non-nil empty `DeniedV4` on `installed`, `candidate`, or both at `restart.go:125-128`, and `adoptStoredServing` rejecting `installed.Generation != e.appliedGen` at `restart.go:36-38`) hold.
   - **Proof:** `TestR21_Held_Q1_OutOfBandRulesetReplacementReachableAndL1Guards` (`node/agent/review_r21_test.go`).

2. **Attack Question 2 — `H-1` idempotency across repeated refused adoptions and host reboots:**
   - **Held (when `retained_applied` matches the installed kernel generation and `settleRetainedUsage` succeeds before a non-serving step):** Repeated refused-adoption `Boot`s on the same boot with a process crash between them settle the restart-gap delta on the first `Boot` and observe a `0 B` delta on the second `Boot` without double-counting or emitting a gap; and a refused adoption after a host reboot (`boot_id` changed) closes the prior boot's epoch as `power_loss` inside `ensureUsage` (`planReboot`) before `settleRetainedUsage` runs, never carrying counter readings across boots.
   - **Did not hold:** Crash window between `plane.Apply` and `retained_applied` persistence (`H-1`), failed `settleRetainedUsage` sample followed by a rebinding `Apply` (`H-2`), and post-`Baseline` traffic under a rebinding `Apply` (`M-1`).
   - **Proof:** `TestR21_Held_Q2_DoubleRefusedAdoptionAndRebootIdempotency` (`node/agent/review_r21_test.go`).

3. **Attack Question 3 — `M-1` (`bootLocal` bounded budget) across `Boot` stages under `SIGTERM`:**
   - **Held (on the normal `Baseline`-succeeds path without `UpdaterRecover` or due outbox uploads):** When `SIGTERM` cancels `caller` while `adoptStoredServing` is waiting on `nodelock.Acquire` or running `evaluate` (`Prober.Probe` / `ResolverDocument`), `adoptStoredServing` returns `false` (`restart.go:73, 88`), `Boot` installs `Baseline`, opens usage, settles retained usage, writes a non-serving heartbeat (`Serving: false, BaselineOK: true`), and skips `e.answerHealthOnBoot(caller)` (`if caller.Err() == nil` at `engine.go:489`) even when a live health challenge is present.
   - **Did not hold:** `ensureUsage` spawning `up.Run(e.loopCtx)` after `SIGTERM`, `recoverOnce(caller)` failing on canceled `caller`, stacked `10s + 10s + 3s` budgets, and exit status `0` when `Baseline` or `loadRuntime` fails under `SIGTERM` (`M-2`).
   - **Proof:** `TestR21_Held_Q3_BootSIGTERMRefusesAdoptionAndSkipsHealthAnswer` (`node/agent/review_r21_test.go`).

4. **Attack Question 4 — `M-2` (ordering of clock-sample and `sameRestartSets` checks before lock, probe, and DNS, plus `planeLocked`):**
   - **Held:** Nothing between `restart.go:46-48` and `restart.go:96` mutates `e.lease`, `e.leaseMono`, or `e.leaseBoot`. Tampered `installed.NowMono` (`< e.leaseMono`, `> now`) and `installed.Now` (`!= e.lease.IssuedAt.Add(installed.NowMono - e.leaseMono)`) as well as changed `DeniedV4`, `ExtraDeniedV4`, or `HostV4` are refused before `nodelock.Acquire`, `Prober.Probe`, and `ResolverDocument`. `e.planeLocked` is `true` only while `adoptStoredServing` (or `enforce`) holds the exclusive node lock, is always cleared by `defer` before `lock.Release()`, prevents `updateLockHeld()` self-contention (`probeexc.go:60`) and `repairGeneration` re-entrancy (`serving.go:57`), and allows `quarantine` (`quarantine.go:133`) to execute under the already-held lock on `ReasonTCForeignFilter` without deadlocking. The preliminary decision (`prelim := Decide(in)` with `ProbeRan=true, ProbeCheck=Passed, DNSFailed=false`) does not refuse any valid adoption that `main` (`05b2c2f2`) accepted.
   - **Proof:** `TestR21_Held_Q4_AdoptionOrderingClockSampleSetChecksAndPlaneLocked` (`node/agent/review_r21_test.go`).

5. **Attack Question 5 — `M-3` (clearing refused-adoption `applyError` after verified `Baseline` and writing non-serving heartbeat in `shutdownUsage`):**
   - **Held (for `verifyServingModel` failures and `shutdownUsage`):** When `adoptStoredServing` fails `verifyServingModel` with `ReasonTCForeignFilter`, `e.quarantinePending` is `true` and `Boot` preserves `e.applyError` (`!e.quarantinePending` at `engine.go:473`), keeping non-serving ACKs blocked while quarantined; when `verifyServingModel` fails with a non-quarantine mismatch (e.g. `ReasonGenerationMismatch`), `retainServingRead` preserves the failed read for observation while `Boot`'s verified `Baseline` clears `e.applyError` so subsequent non-serving generations can be acknowledged. In `shutdownUsage` (`usage.go:435-446`), `e.serving` cannot be `true` when `e.sampleOn` is `false`; when `handoff` installs `Baseline` (`e.serving = false`), `shutdownUsage` recomputes `e.decision = Decide(e.input())` and writes `e.beat()` (`Serving: false, BaselineOK: true`), and when `handoff`'s `e.baseline(ctx)` fails, `e.serving` remains `true` so the serving heartbeat is not overwritten with a false `Serving: false` claim.
   - **Did not hold:** When `adoptStoredServing` fails in `checkMemberPath` (`restart.go:100`), `Boot` clears `e.applyError` without clearing `e.memberPathPending` (`L-1`).
   - **Proof:** `TestR21_Held_Q5_QuarantineRetainsApplyErrorAndShutdownUsageHeartbeat` (`node/agent/review_r21_test.go`).

6. **Attack Question 6 — Regressions in normal serving `Step` or `Apply`:**
   - **Held (when `Boot` adopts serving rules or starts with no prior `retained_applied`):** Follow-up 37 did not alter `Step` or `enforce`'s `Apply` path directly; normal serving `Step`, `Apply`, same-boot serving adoption, and graceful shutdown (`preserveServingOnStop`) continue to pass all existing unit and `r20` regression suites.
   - **Did not hold (after a refused adoption):** `loadRuntime`'s unconditional `e.setMeterBindings(retained.Model.Bindings)` and `Boot`'s `settleRetainedUsage` affect the first post-refusal `Step` / `Apply` as documented in `H-1`, `H-2`, and `M-1`.
   - **Proof:** `TestR21_Held_Q1_OutOfBandRulesetReplacementReachableAndL1Guards`, `TestR21_Held_Q2_DoubleRefusedAdoptionAndRebootIdempotency`, `TestR21_Held_Q4_AdoptionOrderingClockSampleSetChecksAndPlaneLocked` (`node/agent/review_r21_test.go`).

---

## 4. Not examined

- **`r20` `H-2` (`node/enforce/verify.go` set-element range parsing, `@denied_v4` / `@host_v4` set flags and element verification, and `@allowed_v4` emptiness on `VerifyBaseline`):** Owned by `netlab 27` (`wt/w2-netlab`, branch `w2-netlab`), out of scope for `agent-core` follow-up 37 per `.agent/BRIEF.md`.
- **Live privileged `netlab` container execution:** Covered by unit-level deterministic proofs in `node/agent` and `node/agent/cmd/fvpn-agent`.
