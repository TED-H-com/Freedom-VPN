# Adversarial Security & Reliability Review — `r23-intent-fixes-review`

- **Target branch / commit:** `agent-core` at `99f9311271764c86b11a1cc6d42330da7897dce6` (follow-ups 37, 38, and 39 together; compared against `main` `339db172` at #415: 30 files, +23,006 / −107)
- **Follow-up 39 commits under review:** `31862b3a..99f93112` (`0d5fccc4`, `11d8b78f`, `180b9cf0`, `0efa0549`, `edc2f387`, `cd5082a0`, `c5ffb344`, `459dffff`, `99f93112`)
- **Review branch:** `r23-intent-fixes-review`
- **Reviewer:** `agy/gemini-4-argon-high`

---

## 1. Findings by severity

### Critical

None.

---

### High

#### H-1 (`R23-H-1`): Usage-epoch transitions during serving leave `usage_apply_intent` pinned to the superseded epoch, causing restart settlement and same-boot adoption to drop active device bindings or misclassify pure single-generation traffic as `mapping_conflict`

- **Severity:** High
- **Where:**
  - `node/agent/restart_usage.go:54-69` (`recordUsageIntent` binds `usage_apply_intent` in `agent.db` to `(Boot: e.bootID, Epoch: h.OpenEpoch)`)
  - `node/agent/restart_usage.go:86-114` (`captureRestartBindings` only loads `usage_apply_intent` after `reader.RestartGeneration(readCtx)` succeeds and delegates candidate selection to `restartHeadBindings`)
  - `node/agent/restart_usage.go:126-150` (`restartHeadBindings` requires `p.Epoch == h.OpenEpoch` at line 130; when false, it discards `e.restartIntent` and filters `e.restartBindings` by `known[b.DeviceID]` in `h.Counters`)
  - `node/agent/engine.go:1259-1265, 1397-1402` (`enforce` writes `usage_apply_intent` against the pre-Apply `h.OpenEpoch` at line 1260 and then calls `e.switchEpoch(ctx)` at line 1400 when repairing a failed apply or baseline with a pending ruleset-switch nudge)
  - `node/internal/meter/delta.go:258, 421-429` (`planSwitch` inside `Meter.Sample` closes the old epoch and opens a new epoch `nextID` in `usage.db` on any counter reset or ruleset switch without updating `usage_apply_intent` in `agent.db`)
- **Introduced by:** Follow-up 39 (`0efa0549` / `99f93112`).
- **Violated invariant:**
  - Follow-up 39 attribution rule: *"Every byte is attributed to the member and binding revision the kernel counted it under, or recorded as a gap. ... Restart candidates come from the live generation. A device with a single candidate is attributed, not gapped."*
  - `docs/node/followup39-r22-judgments-2026-11-03.md` (`H-2.1`–`H-2.5` & `Design / Restart Attribution And Final-Sample Settlement`): *"Before e.plane.Apply, the agent prepares a fresh unbound usage epoch and records a durable usage_apply_intent keyed by the current boot ID and open usage epoch ID."*
  - `docs/spec/PROJECT_SPEC.zh-TW.md §4 MTR01, MTR02, §10.4`.
- **Scenario:**
  `recordUsageIntent` writes `usage_apply_intent` with `Epoch = h.OpenEpoch` (`E1`) immediately before `e.plane.Apply` (`engine.go:1260`). However, `h.OpenEpoch` is **not** constant for the lifetime of a serving generation:
  1. When `enforce` repairs a prior `e.applyError != nil` or `!e.baselineOK` while `k.nudgePending()` is true, `enforce` writes `usage_apply_intent` with `Epoch = E1` at line 1260 and then calls `e.switchEpoch(ctx)` (`e.meter.NotifyRulesetSwitch(ctx)`) at line 1400, which closes `E1` and opens a new epoch `E2` in `usage.db` while `usage_apply_intent` remains pinned to `E1`.
  2. During normal serving of generation `G2`, any counter reset on any counter in `Meter.Sample` (`delta.go:258, 426`) closes `E1` and opens a new epoch `E2` in `usage.db` while `G2` continues serving on the same boot and `usage_apply_intent` remains on `E1`.
  3. On a subsequent restart on the same boot, `restartHeadBindings(h)` (`restart_usage.go:130`) checks `p.Epoch == h.OpenEpoch` (`E1 == E2`), which is **false**, and discards `usage_apply_intent`:
     - **Subcase A (Unapplied rebinding `desired` + `gen == 0` -> false `mapping_conflict` gap):** While `G2` (`dev -> (member1, rev 1)`) is serving in `E2`, `syncDesired` persists `G3` (`dev -> (member2, rev 2)`), and the node crashes after `Baseline` sets kernel `gen = 0` before `E2` is settled (and before `G3` ever reaches `Apply`). On `Boot`, `captureRestartBindings` sees `gen == 0`, loads `usage_apply_intent` (`Generation = 2, Epoch = E1`), and stores `uniqueUsageBindings(G2, G3)` (empty because `dev` rebound between `G2` and `G3`) in `e.restartBindings`. Because `p.Epoch (E1) != h.OpenEpoch (E2)`, `restartHeadBindings` ignores `usage_apply_intent` and falls back to `e.restartBindings` (empty) plus `h.Counters` (`MemberID == uuid.Nil` when `E2`'s opening sample was unbound or before `dev` was sampled in `E2`), turning 100% of `member1`'s pure `G2` traffic in `E2` into a false `mapping_conflict` gap.
     - **Subcase B (Multi-device serving generation -> refused same-boot adoption + `unknown_device` gap):** Generation `G2` binds `devA -> memberA` and `devB -> memberB`. When `E2` opens (via `switchEpoch` or a counter reset on `devA`), `devA` is in `h.Counters` (`known[devA] = true`) while `devB` has not yet forwarded traffic (`known[devB] = false`). `devB` then forwards `4096 B` in `E2` before a same-boot restart with the kernel dataplane still intact at `G2` (`gen == 2`). On `Boot`, `restartHeadBindings` rejects `usage_apply_intent` (`E1 != E2`) and filters `e.restartBindings` by `known[b.DeviceID]`, dropping `devB`. `adoptStoredServing` (`restart.go:114`) sees `!sameUsageBindings(retained, d.Model.Bindings)` (`1 != 2`), refuses same-boot adoption, withdraws forwarding, and `settleRetainedUsage` records an `unknown_device` gap for `devB`'s `4096 B` instead of attributing it to `memberB`.
     - **Subcase C (Transient `RestartGeneration` error skips loading `usage_apply_intent`):** In `captureRestartBindings` (`restart_usage.go:86-91`), `e.loadUsageIntent()` is only called after `reader.RestartGeneration(readCtx)` returns `err == nil`. If `RestartGeneration` fails transiently at `Boot`, `e.restartIntent` is never loaded (`nil`), even when `usage_apply_intent` is valid for the current `(BootID, OpenEpoch)`.
- **Proof (unit test + failure output against `99f93112`):**
  - Tests:
    - `TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart` (`node/agent/review_r23_test.go:46`)
    - `TestR23_H1b_EpochSwitchDuringServing_RefusesSameBootAdoptionAndDropsNewlyActiveDevice` (`node/agent/review_r23_test.go:164`)
  - Failure output:
    ```text
    === RUN   TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart
        review_r23_test.go:141: R23-H-1a: epoch switch from E1 (01a1a17f-c71f-7039-a255-1479966669a8) to E2 (01a1a17f-c73d-740c-a49a-522c26ffcf99) invalidated usage_apply_intent; member1 attributed=4096 (want 9216), gaps=2 ([{EventSeq:2 Gap:0x3220ad41170 IntervalEnd:2026-11-03 13:01:08.285257 +0000 UTC IntervalStart:2026-11-03 13:01:08.283133 +0000 UTC MeterEpochId:01a1a17f-c71f-7039-a255-1479966669a8 MeteringVersion:forward_admission_v1 NodeId:01a1a17f-c6c2-7670-90df-a74bec72d7ea NodeIncarnationId:01a1a17f-c6ba-73ea-8f8f-a6dc2afbfe8b PayloadSha256:4578cf4736fde2572d7749f0135d817bb34a69a657d5473c99b08f0b21c25567 Quality:gap Records:[] SchemaVersion:1} {EventSeq:3 Gap:0x3220ad411d0 IntervalEnd:2026-11-03 13:01:08.332385 +0000 UTC IntervalStart:2026-11-03 13:01:08.285257 +0000 UTC MeterEpochId:01a1a17f-c73d-740c-a49a-522c26ffcf99 MeteringVersion:forward_admission_v1 NodeId:01a1a17f-c6c2-7670-90df-a74bec72d7ea NodeIncarnationId:01a1a17f-c6ba-73ea-8f8f-a6dc2afbfe8b PayloadSha256:f882360c1e0d1300a6cf3d251f7a9a0c7a3676bdd0998a40c082e83f5c6dc586 Quality:gap Records:[] SchemaVersion:1}])
    --- FAIL: TestR23_H1a_EpochSwitchDuringServing_InvalidatesUsageIntentAndCreatesFalseConflictOnRestart (0.21s)
    === RUN   TestR23_H1b_EpochSwitchDuringServing_RefusesSameBootAdoptionAndDropsNewlyActiveDevice
        review_r23_test.go:268: R23-H-1b: same-boot adoption was refused after switchEpoch left usage_apply_intent on E1; next.serving=false, memberA=2048, memberB=0 (want 4096), gaps=1 ([{EventSeq:2 Gap:0x3220adec8d0 IntervalEnd:2026-11-03 13:01:12.347156 +0000 UTC IntervalStart:2026-11-03 13:01:10.347156 +0000 UTC MeterEpochId:01a1a17f-c831-7ab3-b90a-e390dd02e220 MeteringVersion:forward_admission_v1 NodeId:01a1a17f-c7a0-7a6d-81ca-a5adde49d202 NodeIncarnationId:01a1a17f-c792-7a31-bea0-f1111f8c287b PayloadSha256:f2565ece886bf8332d72fcf9037055bcc5a61fd3465f5f5bf8c833e3565a752c Quality:gap Records:[] SchemaVersion:1}])
    --- FAIL: TestR23_H1b_EpochSwitchDuringServing_RefusesSameBootAdoptionAndDropsNewlyActiveDevice (0.29s)
    ```
- **Fix (owner: `agent-core`):**
  1. In `captureRestartBindings` (`node/agent/restart_usage.go:86`), load `e.restartIntent = e.loadUsageIntent()` **before** checking `reader.RestartGeneration(readCtx)` so a transient kernel read error does not drop the durable intent.
  2. Keep `usage_apply_intent` synchronized with the active serving epoch: update `usage_apply_intent` after `e.switchEpoch(ctx)` in `enforce` (`engine.go:1400`) and after any epoch switch in `readMeter` while serving, and in `restartHeadBindings` accept `p := e.restartIntent` when `p.Boot == e.bootID` and `h.OpenEpoch != uuid.Nil` (since `recordUsageIntent` always runs after settling any prior binding set on the same boot, `p` is the only binding set that could have been active in the currently open epoch on `e.bootID`).

---

#### H-2 (`R23-H-2`): `handoff` resets `e.finalDone = false` after closing the usage epoch, so an early return before publishing `meter.checkpoint` causes the next serving `enforce` to skip `PrepareSampling` and fail admission

- **Severity:** High
- **Where:**
  - `node/agent/usage.go:442-475` (`handoff` calls `e.settleRetainedUsage(ctx)` at line 443, which closes the open epoch in `usage.db` (`head.OpenEpoch = uuid.Nil`), clears `e.setMeterBindings(nil)`, and sets `e.finalDone = true`; line 448 immediately resets `e.finalDone = false` before checking `ClosedCheckpoint`, `meterBarrierIdentity`, and `writeAtomic(e.cfg.MeterCheckpoint)`)
  - `node/agent/engine.go:1231-1265` (`enforce` only calls `e.meter.PrepareSampling(ctx)` when `e.finalDone` is `true` at line 1236, and then unconditionally calls `e.recordUsageIntent(ctx, ...)` at line 1260 when `needApply || !sameBindings`)
  - `node/agent/restart_usage.go:47-53` (`recordUsageIntent` reads `h, err := e.usage.Head(ctx)` and fails with `"usage: no current epoch for admission"` when `h.OpenEpoch == uuid.Nil`)
- **Introduced by:** Follow-up 39 (`0efa0549` / `99f93112`). In follow-up 38, `enforce` called `ResumeSampling` when `e.finalDone` was `true` and did not call `recordUsageIntent`; follow-up 39 added `recordUsageIntent` (requiring `h.OpenEpoch != uuid.Nil`) while keeping `e.finalDone` overloaded as both "barrier checkpoint written for `finalBarrierID`" and "usage epoch closed by `FinalSample`, `PrepareSampling` required before admission".
- **Violated invariant:**
  - `docs/node/followup39-r22-judgments-2026-11-03.md` (`H-1 B`): *"Before e.plane.Apply, the agent prepares a fresh unbound usage epoch and records a durable usage_apply_intent keyed by the current boot ID and open usage epoch ID."*
  - `docs/spec/PROJECT_SPEC.zh-TW.md §4 MTR01, UPD05` (update barrier handoff and recovery when a barrier is replaced or removed).
- **Scenario:**
  1. The node is serving generation 2 when an updater barrier `forwarding.blocked` appears (or `handoff(ctx)` is invoked during a barrier transition).
  2. Inside `handoff(ctx)` (`usage.go:442-475`):
     - `e.baseline(ctx)` withdraws member forwarding;
     - `e.settleRetainedUsage(ctx)` runs `e.meter.FinalSample(ctx)`, which closes the open usage epoch (`head.OpenEpoch = uuid.Nil`), clears `e.setMeterBindings(nil)`, and sets `e.finalDone = true`;
     - At line 448, `handoff` immediately sets `e.finalDone = false` before reading `e.meter.ClosedCheckpoint(ctx)` (line 449), re-checking `meterBarrierIdentity(e.cfg.JournalDir)` (line 455), and writing `meter.checkpoint` (line 470).
  3. If `handoff` returns before line 474 — for example when the updater replaces or removes `forwarding.blocked` while `handoff` is running (the exact D-125 Rule 2 race tested by `TestD125_Rule2LatchReplacementDuringClosePublishesNothing`), or on a transient `ClosedCheckpoint` / `writeAtomic` error — the node is left with:
     - `head.OpenEpoch == uuid.Nil` (epoch closed in `usage.db`),
     - `e.meterBindings() == nil`,
     - `e.retainedPending == false`, and
     - `e.finalDone == false`.
  4. When the barrier is later cleared and `Step(ctx)` runs to resume serving:
     - `enforce` (`engine.go:1226`) sees `!sameBindings && len(e.meterBindings()) > 0` is `false` (because `len(e.meterBindings()) == 0`), so it does not call `handoff`;
     - `enforce` (`engine.go:1236`) sees `if e.finalDone` is `false`, so it **skips `e.meter.PrepareSampling(ctx)`**;
     - `enforce` (`engine.go:1260`) calls `e.recordUsageIntent(ctx, d.Model.Generation, d.Model.Bindings)`, which reads `h.OpenEpoch == uuid.Nil` and fails with `"usage: no current epoch for admission"`;
     - `rejectServing` sets `e.usagePersist = true` and `e.applyError = &api.ObservationError{Code: HEALTH_CHECK_FAILED, Message: "apply failed", Stage: "apply"}`, aborting serving with a false `apply` health-check failure and blocking forwarding until another `Step` runs after `readMeter` reopens an epoch.
- **Proof (unit test + failure output against `99f93112`):**
  - Test: `TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission` (`node/agent/review_r23_test.go:295`)
  - Failure output:
    ```text
    === RUN   TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission
        review_r23_test.go:331: R23-H-2: Step after barrier removal failed admission because handoff left finalDone=false with OpenEpoch=uuid.Nil: serving=false applyError=&{Code:HEALTH_CHECK_FAILED Message:apply failed ObservedAt:2026-11-03 13:01:10.634698826 +0000 UTC Stage:apply} usagePersist=true reasons=[meter_storage_failed safety_check_failed] errors=[{Code:HEALTH_CHECK_FAILED Message:apply failed ObservedAt:2026-11-03 13:01:10.634698826 +0000 UTC Stage:apply} {Code:STORE_UNAVAILABLE Message:persist_failed ObservedAt:2026-11-03 13:01:10.635698826 +0000 UTC Stage:usage}]
    --- FAIL: TestR23_H2_HandoffBarrierRaceLeavesFinalDoneFalse_NextServeFailsAdmission (0.29s)
    ```
- **Fix (owner: `agent-core`):**
  Do not overload `e.finalDone` to mean both "usage epoch was closed and needs `PrepareSampling` before the next `Apply`" and "`meter.checkpoint` was published for `e.finalBarrierID`". Track barrier-checkpoint publication separately (e.g. `e.checkpointDone` paired with `e.finalBarrierID`), keep `e.finalDone = true` whenever `FinalSample` has closed the open epoch until `PrepareSampling` or `ResumeSampling` runs, and in `enforce` also run `PrepareSampling` if `h.OpenEpoch == uuid.Nil`.

---

#### H-3 (`R23-H-3`): Pre-Apply `e.handoff(ctx)` failures in `enforce` leave `e.serving` and `e.decision` inconsistent, emitting contradictory `ObservationReport`s and `Reason: "-"` heartbeats

- **Severity:** High
- **Where:**
  - `node/agent/engine.go:1092-1111` (`enforce`'s `defer` only recomputes `e.decision = Decide(...)` when `baselineOK`, `applyOK`, `dnsOK`, `oversubOK`, `egressOK`, `leaseRefreshPending`, `meterFailed`, `probeExcFailed`, `probeClearFailed`, or `memberPath` changed; it omits `usagePersist != e.usagePersist`, `retainedPending != e.retainedPending`, and `serving != e.serving`)
  - `node/agent/engine.go:1223-1229` (follow-up 39's new pre-Apply `if !sameBindings && len(e.meterBindings()) > 0 { if !e.handoff(ctx) { return nil } }` returns without clearing `e.serving = false` / `e.policyExpires = 0` when `handoff` fails before line 438)
  - `node/agent/usage.go:424-445` (`handoff` returns `false` at line 426 on `invalidateMeterCheckpoint()` failure with `e.usagePersist = true` while `e.serving` is still `true` and forwarding is still active in the kernel; returns `false` at line 436 on `e.baseline(ctx)` failure while `e.serving` is still `true` and `e.baselineOK` is `false`; and returns `false` at line 444 on `e.settleRetainedUsage(ctx)` failure with `e.serving = false`, `e.decision.Serve = false`, `e.retainedPending = true`, and `e.usagePersist = true` without updating `e.decision.Reasons` or `e.decision.Errors`)
  - `node/agent/engine.go:696-702` (`Step` immediately calls `e.beat()` and `e.observe(ctx)` after `enforce` returns `nil`)
- **Introduced by:** Follow-up 39 (`0efa0549` / `99f93112`), which added the pre-Apply `e.handoff(ctx)` call inside `enforce` (`engine.go:1226`) when `d.Serve` is `true` without updating `enforce`'s `defer` decision-recomputation predicate (`engine.go:1102`) or clearing `e.serving` on early `handoff` failure.
- **Violated invariant:**
  - Follow-up 39 Attack Question 3: *"Can a failed final sample keep a member withdrawn, or keep serving off, indefinitely? Is the reason visible in the heartbeat, so an operator can tell it from other outages?"*
  - `node/agent/decide.go:349-352, 480-484` (`UsagePersist` or `RetainedPending` requires `ForwardingBlockReasonMeterStorageFailed`, `SafetyStatus: "failed"`, `Checks["storage"] = "failed"`, and `STORE_UNAVAILABLE`; `!BaselineOK` requires `Serve == false` and `MemberForwardingEnabled == false`).
  - `docs/spec/PROJECT_SPEC.zh-TW.md §4 HB01, ISO01, MTR01`.
- **Scenario:**
  - **Subcase A (`settleRetainedUsage` fails inside pre-Apply `e.handoff` -> `heartbeat.Reason == "-"` and `ObservationReport` with `SafetyStatus: "passed"` and empty `ForwardingBlockReasons`):**
    1. The node is serving with active bindings `B1`. A binding change arrives in `Step` (for example, a second binding `bB` passes `ValidFrom`, or a new generation changes bindings after lease renewal) while `d.Serve` is `true`.
    2. `enforce` (`engine.go:1226`) sees `!sameBindings && len(e.meterBindings()) > 0` and calls `e.handoff(ctx)`.
    3. Inside `handoff`, `e.baseline(ctx)` succeeds (`e.baselineOK == true`, matching the entry `baselineOK` saved at `engine.go:1092`), `e.serving` is set to `false`, `e.decision.Serve` is set to `false`, `e.retainedPending` is set to `true`, and `e.settleRetainedUsage(ctx)` fails (for example, `FinalSample` returns `meter.ErrHold` because the binding change arrived within the same clock instant as the prior sample or because `usage.db` is at `MaxUnackedRows`, or a `usage.db` write error occurs). `settleRetainedUsage` sets `e.usagePersist = true` while `e.meterFailed` remains `false`.
    4. `enforce` returns `nil` at `engine.go:1228`. Its `defer` at `engine.go:1102` checks `baselineOK`, `applyOK`, `dnsOK`, `oversubOK`, `egressOK`, `leaseRefreshPending`, `meterFailed`, `probeExcFailed`, `probeClearFailed`, and `memberPath` — **none of which changed**! So `e.decision` is **not** recomputed; only `e.decision.Serve` was mutated to `false` by `handoff`, leaving `e.decision.Reasons = []` and `e.decision.Errors = []`.
    5. `Step` then calls `e.beat()` and `e.observe(ctx)`:
       - `e.beat()` (`engine.go:1478`) writes a watchdog heartbeat with `Serving: false` and **`Reason: "-"`** (because `e.decision.Reasons` is empty), hiding `meter_storage_failed` from the watchdog and operators;
       - `e.observe(ctx)` (`engine.go:1414`, triggered because `prevServing != e.serving`) sends an `ObservationReport` to Central with `MemberForwardingEnabled: false`, **`ForwardingBlockReasons: []`**, **`SafetyStatus: "passed"`**, `Checks["storage"]: "passed"`, and **`Errors: []`**.
  - **Subcase B (`e.baseline(ctx)` or `invalidateMeterCheckpoint()` fails inside pre-Apply `e.handoff` -> `e.serving` remains `true` alongside `SafetyStatus: "failed"`):**
    1. On the same pre-Apply `e.handoff(ctx)` path (`engine.go:1226`), if `e.baseline(ctx)` at `usage.go:435` fails, `handoff` sets `e.baselineOK = false` and returns `false` **before** line 438 (`e.serving = false`). `enforce` returns `nil` at `engine.go:1228` without clearing `e.serving = false` or `e.policyExpires = 0`.
    2. `enforce`'s `defer` sees `baselineOK != e.baselineOK` and recomputes `e.decision` (`Serve: false`, `Reasons: ["safety_check_failed"]`), **but `e.serving` is still `true`!**
    3. `e.beat()` writes a watchdog heartbeat with **`Serving: true, BaselineOK: false`** (violating `watchdog/heartbeat.go:180`, which rejects `Serving && !BaselineOK` as `unhealthy: baseline not ok`), and `e.observe(ctx)` sends a contradictory `ObservationReport` to Central with **`MemberForwardingEnabled: true`**, **`SafetyStatus: "failed"`**, and **`ForwardingBlockReasons: ["safety_check_failed"]`**.
    4. Worse still, if `e.invalidateMeterCheckpoint()` fails at `usage.go:424`, `handoff` sets `e.usagePersist = true` and returns `false` at line 426 **before `e.baseline(ctx)` even runs**: member forwarding remains live in the kernel (`e.serving == true`), `e.decision` is not recomputed (`e.decision.Serve == true`), and the node continues serving with `e.usagePersist == true`.
- **Proof (unit tests + failure output against `99f93112`):**
  - Tests:
    - `TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat` (`node/agent/review_r23_test.go:358`)
    - `TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety` (`node/agent/review_r23_test.go:461`)
  - Failure output:
    ```text
    === RUN   TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat
        review_r23_test.go:440: R23-H-3a: enforce did not recompute e.decision after pre-Apply handoff failure: heartbeat.Reason="-" (want "meter_storage_failed"), report.MemberForwardingEnabled=false, report.SafetyStatus="passed" (want "failed"), report.ForwardingBlockReasons=[] (want [meter_storage_failed]), report.Errors=[]
    --- FAIL: TestR23_H3a_PreApplyHandoffSettleFailure_EmitsCorruptObservationAndDashHeartbeat (0.29s)
    === RUN   TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety
        review_r23_test.go:498: R23-H-3b: pre-Apply handoff baseline failure left serving=true alongside failed safety: eng.serving=true, hb.Serving=true, hb.BaselineOK=false, rep.MemberForwardingEnabled=true, rep.SafetyStatus="failed", rep.ForwardingBlockReasons=[safety_check_failed]
    --- FAIL: TestR23_H3b_PreApplyHandoffBaselineFailure_LeavesServingTrueWithFailedSafety (0.29s)
    ```
- **Fix (owner: `agent-core`):**
  1. In `enforce` (`node/agent/engine.go:1092-1111`), snapshot `usagePersist := e.usagePersist`, `retainedPending := e.retainedPending`, and `serving := e.serving` on entry and include `usagePersist != e.usagePersist || retainedPending != e.retainedPending || serving != e.serving` in the `defer` condition that recomputes `e.decision = Decide(e.decisionInput(e.clock.Wall(), e.clock.Mono()))`.
  2. In `handoff` (`node/agent/usage.go:424-445`) and at `engine.go:1226`, ensure `e.serving = false; e.policyExpires = 0` is always set whenever `handoff` is entered on a non-serving or rebinding transition, and run `e.baseline(ctx)` before returning on `invalidateMeterCheckpoint()` failure so a checkpoint removal error never leaves forwarding live with `e.usagePersist = true`.

---

### Medium

#### M-1 (`R23-M-1`): `withdraw` double-`baseline` asymmetry leaves the kernel on `Baseline` (`gen = 0`) with an unsettled open usage epoch and `e.retainedPending = false`

- **Severity:** Medium
- **Where:**
  - `node/agent/engine.go:1468-1473` (`withdraw(ctx)` calls `e.handoff(ctx)` and then unconditionally calls `_ = e.baseline(ctx)` a second time before setting `e.serving = false; e.policyExpires = 0`)
  - `node/agent/usage.go:434-445` (`handoff(ctx)` only sets `e.retainedPending = true` and calls `e.settleRetainedUsage(ctx)` **after** `e.baseline(ctx)` succeeds at line 435)
- **Introduced by:** Pre-existing in follow-up 37/38 (`withdraw` / `handoff`), compounded by follow-up 39 (`0efa0549` / `99f93112`), where `enforce` now calls `recordUsageIntent` on the open epoch without `PrepareSampling` whenever `e.retainedPending == false && !e.finalDone`.
- **Violated invariant:**
  - Follow-up 39 attribution rule: *"Every byte is attributed to the member and binding revision the kernel counted it under, or recorded as a gap. ... A failed hand-off gates Apply."*
  - `docs/node/followup39-r22-judgments-2026-11-03.md` (`H-1 B`): *"On failure, settleRetainedUsage keeps the bindings, sets e.retainedPending = true and e.usagePersist = true, and leaves the open epoch unsettled."*
- **Scenario:**
  1. While generation `G1` is serving with active bindings `B1`, `e.withdraw(ctx)` runs (for example, when a lease expires or a safety check fails in `enforce`, or when `ensureUsage` fails its `st.Head` read at `usage.go:220`).
  2. Inside `withdraw(ctx)` (`engine.go:1468-1473`):
     - `e.handoff(ctx)` calls `e.baseline(ctx)` (`usage.go:435`). Suppose that first `e.baseline(ctx)` call fails transiently (`handoff` returns `false` at line 436 **before** setting `e.retainedPending = true` at line 442 and before calling `e.settleRetainedUsage(ctx)` at line 443);
     - Immediately on the next line (`engine.go:1470`), `withdraw` calls `_ = e.baseline(ctx)` a second time, and that second call **succeeds** (`e.baselineOK = true`, kernel `gen = 0`), followed by `e.serving = false; e.policyExpires = 0`.
  3. The node now has `Baseline` installed in the kernel (`gen = 0`, `e.baselineOK == true`, `e.serving == false`), **yet `e.retainedPending` is `false`, `e.finalDone` is `false`, and the open usage epoch in `usage.db` was never settled by `FinalSample`**.
  4. On the next `Step(ctx)` when serving resumes for a new generation `G2`, `Step` does not run `settleRetainedUsage` at line 665 (`e.retainedPending == false`). If `G2` has the same bindings (or if `enforce` runs `Apply`), `enforce` sees `!e.finalDone`, skips `PrepareSampling`, and overwrites `usage_apply_intent` on the still-open epoch without having settled `G1`'s withdrawal boundary.
- **Proof (unit test + failure output against `99f93112`):**
  - Test: `TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse` (`node/agent/review_r23_test.go:517`)
  - Failure output:
    ```text
    === RUN   TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse
        review_r23_test.go:548: R23-M-1: withdraw installed Baseline (gen=0, baselineOK=true) but left OpenEpoch=01a1a17f-cccd-752b-9407-337cb38c1ce4 unsettled with retainedPending=false finalDone=false
    --- FAIL: TestR23_M1_WithdrawSecondBaselineSuccess_LeavesOpenEpochUnsettledWithRetainedPendingFalse (0.25s)
    ```
- **Fix (owner: `agent-core`):**
  In `handoff(ctx)` (`node/agent/usage.go:434`), set `e.retainedPending = true` before `e.baseline(ctx)` whenever `len(e.meterBindings()) > 0` (so any withdrawal attempt latches `retainedPending = true` until `settleRetainedUsage` succeeds), and remove the redundant second `_ = e.baseline(ctx)` call in `withdraw(ctx)` (`node/agent/engine.go:1470`).

---

### Low

#### L-1 (`R23-L-1`): `docs/node/followup39-r22-judgments-2026-11-03.md` inaccurately describes when `adoptStoredServing` calls `invalidateMeterCheckpoint`

- **Severity:** Low
- **Where:**
  - `docs/node/followup39-r22-judgments-2026-11-03.md:71-76` (claims under `M-2`: *"Only after the final controlContext(ctx).Err() == nil && e.decision.Serve && e.verifyServingModel(ctx, d.Model) check succeeds does adoptStoredServing invalidate any stale meter checkpoint, set the live meter bindings, call resumeAdoptionUsage(ctx), and start the uploader."*)
  - `node/agent/engine.go:531` and `node/agent/restart.go:80, 128-185` (`e.latch()` calls `e.invalidateMeterCheckpoint()` at `usage.go:161` during `bootWithBudget` at `engine.go:531` and `adoptStoredServing` at `restart.go:80` whenever no barrier is active, **before** `prepareAdoptionUsage` and **before** `verifyServingModel`; `adoptStoredServing` never calls `invalidateMeterCheckpoint` after `verifyServingModel`)
- **Introduced by:** Follow-up 39 (`0d5fccc4` / `99f93112`).
- **Violated invariant:**
  - Documentation accuracy (`AGENTS.md §9`). Note that removing an orphan `meter.checkpoint` when no `forwarding.blocked` barrier is active is safe, so this is a documentation inaccuracy rather than a runtime defect.
- **Scenario & Proof (code trace):**
  In `node/agent/restart.go:128-185`, after `if controlContext(ctx).Err() != nil || !e.decision.Serve || !e.verifyServingModel(ctx, d.Model)` succeeds at line 132, `adoptStoredServing` calls `e.setMeterBindings(d.Model.Bindings)` (line 138), `e.resumeAdoptionUsage(ctx)` (line 139), and sets `e.serving = true` (line 145); it does not call `e.invalidateMeterCheckpoint()`. Instead, `e.invalidateMeterCheckpoint()` is called inside `e.latch()` (`node/agent/usage.go:161`) at `engine.go:531` and `restart.go:80` when no barrier is active.
- **Fix (owner: `agent-core`):**
  Update `docs/node/followup39-r22-judgments-2026-11-03.md` (`M-2` Fix section) to state accurately that `e.latch()` invalidates any stale `meter.checkpoint` when no update barrier is active, and that `adoptStoredServing` defers `setMeterBindings`, `resumeAdoptionUsage`, and `up.Run` until after `verifyServingModel` and `resumeAdoptionUsage` succeed.

---

## 2. Merge recommendation

**Do not merge follow-ups 37, 38, and 39 as they are; merge after `agent-core` fixes `H-1` (`R23-H-1`), `H-2` (`R23-H-2`), `H-3` (`R23-H-3`), `M-1` (`R23-M-1`), and `L-1` (`R23-L-1`).** Follow-up 39 solidly closes the original `r22` crash and adoption windows when the usage epoch stays fixed and `handoff` completes normally, but epoch transitions during a serving generation invalidate `usage_apply_intent` (`H-1`), an interrupted barrier `handoff` leaves `e.finalDone == false` with a closed epoch so the next serving `Step` fails admission (`H-2`), pre-Apply `handoff` failures bypass `enforce`'s decision recomputation or leave `e.serving == true` alongside `BaselineOK == false` (`H-3`), and `withdraw`'s double `baseline` call can leave `Baseline` installed with an unsettled open epoch and `e.retainedPending == false` (`M-1`).

---

## 3. What held

1. **Attack Question 1 — Completeness of follow-up 39's fixes for confirmed `r22` findings and preservation of `r20`/`r21`/`r22` proofs:**
   - All converted `r22` proof tests in `node/agent/review_r22_test.go` (`TestR22_H1_UnsettledGenerationTransitionAndHandoffMisattributeUsage`, `TestR22_H2_RefusedAdoptionCandidateSelectionCreatesFalseGaps`, `TestR22_M1_BootEnsureUsageFailureLatchesMeterFailedPermanently`, `TestR22_M2_AbortedAdoptionContaminatesBaselineEpochAndBypassesPreserveClosed`, `TestR22_M3_BootFailureHeartbeatTruthfulnessAndProbeClearRetry`, and `TestR22_Held_*`) pass against `99f93112`.
   - Follow-up 39's adjustment to `H-2.5` (`crash_before_retained_applied_write_falls_back_to_stale_generation_and_gaps_new_device` in `node/agent/review_r22_test.go:633-721`) is sound: in follow-up 39, `recordUsageIntent` writes `usage_apply_intent` *before* `e.plane.Apply` and `Apply` atomically embeds the live generation `gen` in the nftables table (`fvpn_meta_` comment), so when `OpenEpoch` has not switched since `recordUsageIntent`, `restartHeadBindings` attributes single-generation traffic even if the process crashes before `saveApplied` writes `retained_applied`.
   - Follow-up 39's edits to `node/agent/review_r20_test.go` (`TestR20_H2_RestartBeforeEpochClosesettlesRetainedKernelCounters`) and `node/agent/review_r21_test.go` (`TestR21_H1_RestartAfterRebindingApplyAttributesRetainedCountersToOldMember`) accurately reflect the new pre-Apply `handoff` settlement and `usage_apply_intent` semantics without weakening the underlying attribution assertions.
   - Corrupt JSON in `usage_apply_intent` and stale intents from a previous OS boot (`Boot != e.bootID`) are rejected fail-closed by `loadUsageIntent` and `restartHeadBindings` (`TestR23_Held_AQ1_UsageIntentCrashWindowsAndCrossRebootIsolation` in `node/agent/review_r23_test.go:559`).

2. **Attack Question 2 — Validity of follow-up 39's two partial rejections (`H-1 B` permanently-paused sampling sub-claim and `M-2` original fixture trace):**
   - **`H-1 B` paused-sampling sub-claim:** Follow-up 39's judgment is right for any `Apply` that succeeds or fails after `PrepareSampling`: `PrepareSampling` (`node/internal/meter/meter.go:258-291`) sets `m.paused = true`, and every successful `enforce` path calls `e.resumeSampling()` at `node/agent/engine.go:1388` unconditionally (not gated on `e.finalDone`). If `recordUsageIntent` or `e.plane.Apply` fails after `PrepareSampling`, `rejectServing` (`node/agent/engine.go:1455-1466`) calls `e.withdraw(ctx)`, so the node is not serving while `m.paused` is `true` (`TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply` in `node/agent/review_r23_test.go:623`). *(The actual defect around `e.finalDone` and `PrepareSampling` is `H-2` above, where an interrupted `handoff` leaves `e.finalDone == false` with `OpenEpoch == uuid.Nil`.)*
   - **`M-2` late adoption abort:** Follow-up 39's split of `prepareAdoptionUsage(ctx)` (`node/agent/usage.go:175-183`, using `PreserveClosedEpoch: true` and starting no uploader) and `resumeAdoptionUsage(ctx)` (`node/agent/usage.go:185-217`, called at `node/agent/restart.go:139` only after `controlContext(ctx).Err() == nil && e.decision.Serve && e.verifyServingModel(ctx, d.Model)` succeeds) ensures that a late adoption abort with either an open epoch or a closed epoch does not open a new epoch, mutate `e.meterSnap`, or start `up.Run` before the final adoption gate (`TestR22_M2_AbortedAdoptionContaminatesBaselineEpochAndBypassesPreserveClosed` in `node/agent/review_r22_test.go:804`).

3. **Attack Question 3 — New mechanisms (accounting intent, withdraw before identity change, live-generation marker, removed/re-added devices, double counting):**
   - **Pre-Apply settlement on binding change:** When bindings change while serving (`!sameBindings && len(e.meterBindings()) > 0`), `enforce` (`node/agent/engine.go:1226`) calls `e.handoff(ctx)` before `PrepareSampling`, `recordUsageIntent`, and `e.plane.Apply`, settling the outgoing bindings' delta in the old epoch and opening a fresh unbound epoch with the post-settlement counter baseline before the new bindings take effect (`TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply` in `node/agent/review_r23_test.go:623`).
   - **No double counting across settlement and fresh epoch:** `FinalSample` (`SampleAndClose` in `node/internal/meter/delta.go:258`) commits the final delta of the old epoch and closes it (`OpenEpoch = uuid.Nil`). `PrepareSampling` (`node/internal/meter/meter.go:258-291`) then opens a new epoch whose baseline `head.Counters` records the current kernel counter readings unbound (`MemberID == uuid.Nil`), and the first sample after `Apply` (`Sample` in `node/internal/meter/delta.go:219-225`) adopts the new binding metadata when `cur == old` or charges only the delta above that baseline (`cur.Bytes - old.Bytes`), so no byte is counted twice (`TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply`).
   - **Removed and re-added devices:** In the real nftables renderer (`node/enforce/nft.go:215-248`), per-device named counters (`u_<device_id>`, `d_<device_id>`) are deleted via `delete counter inet fvpn ...` whenever a device leaves the active binding set (and on `Baseline`, which deletes and recreates `table inet fvpn`). Even in the in-memory fake plane where counter map keys persist across generations, removing a device triggers `!sameBindings` -> `e.handoff(ctx)` -> `FinalSample` (attributing all bytes up to removal to the old member and closing the epoch) and `PrepareSampling` (recording the post-removal counter reading as the unbound baseline of the new epoch), so re-adding the device under a new member only attributes bytes above that baseline (`TestR23_Held_AQ2_BindingRebindSettlesOldBindingsBeforePrepareAndApply`).
   - **Live-generation marker atomicity:** In `node/enforce/nft.go:268`, the generation comment `fvpn_meta_v1;gen=<G>;rules_sha256=<H>` is emitted inside the same `add table inet fvpn { comment "..."; ... }` block passed to `nft -f -`, so kernel installation and the generation marker are atomic in a single netlink transaction (`TestRenderApply_Golden` and `TestNftPlane_RestartInspection` in `node/enforce/nft_test.go`).
   - **`ensureUsage` `st.Head` failure fail-closed withdrawal:** If `st.Head` fails during `ensureUsage` while `e.serving` is `true`, `ensureUsage` (`node/agent/usage.go:219-227`) calls `e.withdraw(ctx)`, sets `e.usagePersist = true`, and recomputes `e.decision` so `Step` blocks `Apply` and reports `meter_storage_failed` (`TestR23_Held_AQ3_EnsureUsageHeadFailureWithdrawsAndBlocksServe` in `node/agent/review_r23_test.go:703`).

4. **Attack Question 4 — `M-1` retry after repeated transient usage initialization failures without resource leaks:**
   - In `ensureUsageWith` (`node/agent/usage.go:203-271`), `st`, `m`, and `up` are held in local variables until `st.Open`, `st.Head`, `meter.NewContext`, and `usageupload.New` all succeed; if any step fails, `cleanup()` closes `m` and `st` before returning, and `e.usage`, `e.meter`, and `e.uploader` remain `nil` until full initialization succeeds (`TestR23_Held_AQ4_EnsureUsageRepeatedFailuresLeakNothingAndRecover` in `node/agent/review_r23_test.go:738`, `TestR22_M1_BootEnsureUsageFailureLatchesMeterFailedPermanently` in `node/agent/review_r22_test.go:725`).

5. **Attack Question 5 — Boot and shutdown budget, heartbeat truthfulness, probe clear retry, and `DatapathValue` validation:**
   - **Boot + `Close` budget inside `TimeoutStopSec=15s`:** Shipped `deploy/node/systemd/fvpn-agent.service` sets `TimeoutStopSec=15s` (pinned by `TestPKG01_AT07ShippedAgentStopBudget` in `packaging/Linux/systemd_stop_budget_test.go`). `bootAdoptBudget` (`4s`) + `bootLocalBudget` (`6s`) = `10s`, with each `beatBoot` bounded by `bootBeatMaxBudget = 100ms` (`node/agent/boot_heartbeat.go:19`) clipped to the remaining stage deadline, and `Close` bounded by `closeTotalBudget = 3s` (`node/agent/engine.go:34`), keeping worst-case `Boot` + `Close` within `13s < 15s` (`TestR21_M2_BootTimeoutAndSignalHandling` in `node/agent/review_r21_test.go:219`).
   - **Boot heartbeat truthfulness and non-overwriting late workers:** `beatBoot` (`node/agent/boot_heartbeat.go:23-61`) serializes heartbeat writes through `e.beat gate` (a 1-slot channel mutex). A timed-out `beatBoot` caller does not release the gate until its background `WriteHeartbeat` finishes, preventing a stale timed-out worker from racing with and overwriting a newer heartbeat written by a subsequent stage (`TestR22_M3_BootFailureHeartbeatTruthfulnessAndProbeClearRetry` in `node/agent/review_r22_test.go:880`, `TestR23_Held_AQ5_BootHeartbeatStopProbeExceptionAndDatapathValidation` in `node/agent/review_r23_test.go:788`).
   - **Probe clear retry and deadline enforcement:** When `RemoveProbeException` fails during `answerHealthOnBoot` (`node/agent/probeexc.go:88-96`), `e.probeExc` is retained (`Cleared: false`), `e.probeClearFailed` is set to `true`, and a subsequent `reconcileProbeException` in `Step` retries `RemoveProbeException` using the original `ExpiresMono` deadline without requiring a new issuance (`TestR22_M3_BootFailureHeartbeatTruthfulnessAndProbeClearRetry/boot_probe_clear_failure_preserves_probe_state_for_step_retry_and_reports_observation_error` in `node/agent/review_r22_test.go:967`). In `syncProbeException` (`node/agent/probeexc.go:177-189`), `ttl := min(rem, maxTTL)` clamps the kernel nftables element timeout so it never exceeds the remaining issuance deadline `rem`. In `Close()` (`node/agent/engine.go:639`), `stopProbeException(closeCtx)` removes any active probe exception on shutdown (`TestR23_Held_AQ5_BootHeartbeatStopProbeExceptionAndDatapathValidation`).
   - **`DatapathValue` validation:** `Config.validate()` (`node/agent/config.go:218`) validates `_, err := c.DatapathValue()` inside `LoadConfig` before `New` / `Boot` in production, and there is no live config-reload path that mutates `e.cfg.Datapath` after startup (`TestR23_Held_AQ5_BootHeartbeatStopProbeExceptionAndDatapathValidation`).

6. **Attack Question 6 — Serving-path changes and historical record integrity:**
   - **Same-identity renewal optimization:** When `enforce` runs with `!needApply && sameBindings` (such as a same-generation lease renewal or clock tick with unchanged bindings), `enforce` (`node/agent/engine.go:1259`) skips `recordUsageIntent` (no `agent.db` write) and only checks `e.usage.Head(ctx)` at lines 1347-1356 when `e.leaseRefreshPending` is being cleared (`TestR23_Held_AQ6_SameIdentityRenewalSkipsRedundantIntentWrite` in `node/agent/review_r23_test.go:833`).
   - **Dated corrections in `docs/node/followup38-r21-attribution-and-boot-2026-11-03.md`:** Follow-up 39 preserved follow-up 38's original text intact and appended a clearly labeled section (`## Follow-up 39 Corrections (2026-11-03)`, lines 169-216) recording what review `r22` disproved (`H-1`, `H-2`, `M-1`, `M-2`, `M-3`).

---

## 4. Not examined

- **Live privileged kernel `nft` / `tc` / WireGuard execution (`tests/netlab`) and D-133 4-leg `A,B,A,B` timing comparison:** Per `.agent/BRIEF.md` and `AGENTS.md §5 & §7`, `go test` was run without root or host network changes using the `node/agent` unit and integration harnesses on temporary SQLite databases, and the incomplete D-133 `A,B,A,B` timing comparison is tracked separately outside this review.
- **Central control plane (`apps/control`, `apps/worker`) and web portal (`apps/portal`):** Out of scope for `r23-intent-fixes-review`, which covers `agent-core` follow-ups 37, 38, and 39 (`git diff 339db172 99f93112`).
