# Adversarial Security Review `r25-retain-review`: `agent-core` Follow-ups 40 & 41 (`8cc27609`)

- **Reviewed commit:** `8cc2760965abbbb1af3340976c6c4adf1be3dc37` (`w2-agent-core` follow-up 41 continuation 1 tip, carrying follow-ups 37 to 41; last merge of `main` is `c98ea61c` as `6b6a26b0`)
- **Follow-up 40 range:** `99f93112..b17c8c17` (fixes for `r23` in `docs/reviews/r23-intent-fixes-review.md`)
- **Follow-up 41 range:** `b17c8c17..8cc27609` (fixes for `r24` in `ba5d3004:.agent/REVIEW.md`)
- **Combined diff against `main` (`c98ea61c..8cc27609`):** 63 files changed, +27,864 / −164 lines
- **Reviewer:** `agy/gemini-4-argon-high`

---

## 1. Findings by severity

### Summary of findings

| ID | Severity | Title | Where | Introduced in | Owner |
|---|---|---|---|---|---|
| `H-1` | **High** | Single-binding deadline expiry (`expireServing`) while a sibling binding remains valid tears down the entire node to Baseline before the control/probe pass, and — when reached during `verifyCurrentServing` or `enforce`'s `canRetainServing` — returns early from `Step`, leaving surviving bindings unserved in Baseline with a corrupt `Reason: "-"` heartbeat and a contradictory `ObservationReport` | `node/agent/decide.go:663-670`, `node/agent/engine.go:696,709-713,1182-1186,1496-1501`, `node/agent/serving.go:100,111,119-129,167-187`, `node/agent/observe.go:94-132` | Follow-up 41 (`c5b17256`, `77689438`, `19677dc9`), interacting with `expireServing` from follow-up 23 | `agent-core` |
| `M-1` | **Medium** | `canRetainServing` requires strict `intent.Epoch == h.OpenEpoch` instead of `SameCountingInterval` (and `switchEpoch` does not refresh `e.restartIntent`), forcing a redundant `Plane.Apply` on the next steady-state `304 Not Modified` tick after any ruleset-switch or counter-discontinuity epoch rotation | `node/agent/retain.go:47-54`, `node/agent/engine.go:1268,1416`, `node/agent/restart_usage.go:43-70` | Follow-up 41 (`77689438`), interacting with follow-up 40 (`b17c8c17`) | `agent-core` |

No Critical or Low findings were identified.

---

### Finding `H-1` (High): Single-binding deadline expiry (`expireServing`) while a sibling binding remains valid tears down the entire node to Baseline before the control/probe pass, and — when reached during `verifyCurrentServing` or `enforce`'s `canRetainServing` — returns early from `Step`, leaving surviving bindings unserved in Baseline with a corrupt `Reason: "-"` heartbeat and a contradictory `ObservationReport`

- **Severity:** High
- **Where:**
  - `node/agent/decide.go:663-670` (`buildModel` sets `policyExpires = min(policyExpires, deviceDeadline)` across all admitted bindings)
  - `node/agent/engine.go:694-696` (`Step` entry calls `e.expireServing(ctx)`)
  - `node/agent/engine.go:709-713` (`if !e.verifyCurrentServing(ctx)` early return from `Step`)
  - `node/agent/engine.go:1181-1186` (`enforce`'s `canRetainServing` branch returns `nil` when `!e.expireServing(ctx)`)
  - `node/agent/engine.go:1496-1501` (`beat()` writes `Reason: "-"` when `!e.serving && len(e.decision.Reasons) == 0`)
  - `node/agent/serving.go:100,111,119-129,167-187` (`verifyCurrentServing`, `expireServing`, `finishServingStep`)
  - `node/agent/observe.go:94-132` (`report` builds `ObservationReport` with `MemberForwardingEnabled: e.serving` alongside `e.decision`'s `ForwardingBlockReasons`, `DataHealth`, `SafetyStatus`, `Checks`, and `Errors`)
- **Introduced in:** Follow-up 41 (`c5b17256`, `77689438`, `19677dc9`), interacting with `expireServing` from follow-up 23.
- **Scenario:**
  Suppose a node serves two bindings — either two devices of the same member with different `ValidUntil` timestamps (`surviving` with `ValidUntil = T0 + 1h` and `expiring` with `ValidUntil = T0 + 20s`, as in `TestAT13_F41_TimeDrivenModelChanges`'s `"sibling_expiry"`) or two members with different `LeaseEnd` / `EffectiveServiceEnd` timestamps (`"member_expiry"`).
  In `buildModel` (`node/agent/decide.go:663-670`), `e.policyExpires` (which is both the wake deadline in `nextStepDelay` and the whole-node expiry deadline checked by `e.expireServing` and written to `Beat.PolicyExpires`) is set to the earliest single binding's deadline (`T0_mono + 20s`, `expiring`'s deadline).
  Note that in kernel space, each device's element in `allowed_v4` already carries its own per-device `timeout` (`node/enforce/deadline.go:36-62`, `render_nft.go`), so `expiring`'s kernel element expires automatically at `T0 + 20s` without tearing down `surviving`.

  In userspace, however, `e.expireServing(ctx)` (`node/agent/serving.go:119-129`) treats `e.clock.Mono() >= e.policyExpires` as a whole-node expiry and executes:
  ```go
  e.withdraw(cleanup)
  e.decision = Decide(e.input())
  e.lastObs = time.Time{}
  return true
  ```
  This creates two distinct failure modes when `expiring` reaches `T0 + 20s` while `surviving` remains valid for another 59m40s:

  1. **Subcase A — Clock reaches `e.policyExpires` before `Step` line 696 (`e.expireServing(ctx)`):**
     - At `Step` entry (`node/agent/engine.go:696`, added in `77689438` with the comment `// Re-decide on this tick after a planned deadline withdrawal. Surviving siblings must not wait another poll merely because one device expired`), `e.expireServing(ctx)` calls `e.withdraw(cleanup)` (`engine.go:1484-1489`), which runs `e.handoff(cleanup)` (`e.baseline(cleanup)`) and a second `_ = e.baseline(cleanup)`, tearing down the **entire node ruleset to `fvpn_baseline=1`** (`r.plane.baselines` increments by 2, `e.serving = false`, `len(r.plane.last.Bindings) == 0`) **before** `Step` runs its HTTP control and probe pass (`flush`, `syncDesired`, `syncLease`, `syncCert`, `syncBootstrap`, and `evaluate` -> `refreshProbe` at `engine.go:734-766`).
     - Every surviving binding on the node is blackholed in `fvpn_baseline=1` for the entire duration of the control-plane HTTP pass and tunnel probe (up to the 5-second control pass budget if the control plane is slow or unreachable) before `e.enforce(ctx, e.decision)` at line 767 re-applies the surviving bindings.

  2. **Subcase B — Clock reaches `e.policyExpires` during `verifyCurrentServing` (`node/agent/serving.go:100, 111`) or during `enforce`'s `canRetainServing` (`node/agent/engine.go:1182-1186`):**
     - If `Step` starts shortly before `e.policyExpires` (`e.clock.Mono() < e.policyExpires` at `engine.go:696`) and the clock reaches `e.policyExpires` either during `verifyCurrentServing` (`serving.go:100` or `111`, at `Step` line 709) or during `enforce`'s `canRetainServing` live verification (`engine.go:1182-1183`, when `Step` starts `>101ms` before `e.policyExpires` so `Decide` still admits both bindings and the live read in `enforce` at line 1182 spans `e.policyExpires`):
       1) `e.expireServing(ctx)` calls `e.withdraw(cleanup)`, tearing down the entire node to `fvpn_baseline=1` (`e.serving = false`, `e.policyExpires = 0`), and sets `e.decision = Decide(e.input())`.
       2) Because `surviving` is still valid (`now < T0 + 1h`), `Decide(e.input())` returns `Serve: true`, `Reasons: nil`, `DataHealth: api.DataHealthHealthy`, `Safety: api.SafetyStatusPassed`, `Checks: [all 10 passed]`, and `Errors: nil`.
       3) If triggered in `verifyCurrentServing`, `verifyCurrentServing` returns `false` and `Step` (`engine.go:709-713`) calls `_ = e.observe(ctx)`, `_ = e.flush(ctx)`, and **returns `nil` immediately without calling `evaluate` or `enforce`**. If triggered in `enforce` at `engine.go:1183`, `!e.expireServing(ctx)` is `false` and `enforce` **returns `nil` at line 1186 without applying the surviving binding's model** (`enforce`'s `defer` at `engine.go:1095` recomputes `e.decision = Decide(e.input())` with `Serve: true, Reasons: nil`).
       4) On `Step` return, `finishServingStep` (`serving.go:167-187`) runs with `e.serving == false` and `e.decision.Serve == true` (`len(e.decision.Reasons) == 0`):
          - `e.beat()` (`engine.go:1496-1501`) sees `!e.serving && len(e.decision.Reasons) == 0` and writes a corrupt watchdog heartbeat with `Serving: false, PolicyExpires: 0, Reason: "-"`.
          - `e.observe(ctx)` and `e.flush(ctx)` (`observe.go:94-132`) build and POST to Central a contradictory `ObservationReport` with `MemberForwardingEnabled: false`, `ForwardingBlockReasons: []` (empty), `DataHealth: "healthy"`, `SafetyStatus: "passed"`, `Checks: [all 10 passed]`, and `Errors: []`.
          - Surviving binding `surviving` (`ValidUntil = T0 + 1h`) is left completely unserved in Baseline (`eng.serving == false`, `len(r.plane.last.Bindings) == 0`) until the next `Step`!

- **Proof (failing unit tests in `node/agent/review_r25_test.go`):**
  - `TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass`
  - `TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat`
  ```text
  === RUN   TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass
  === RUN   TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass/sibling_device_expiry
      review_r25_test.go:122: TODO(review r25): H-1 (sibling_device_expiry): single-binding expiry while sibling remains valid tore down the entire node to Baseline before the control/probe pass (servingDuringControlPass=false bindingsDuringControlPass=0 baselines=7->9)
  === RUN   TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass/sibling_member_expiry
      review_r25_test.go:122: TODO(review r25): H-1 (sibling_member_expiry): single-binding expiry while sibling remains valid tore down the entire node to Baseline before the control/probe pass (servingDuringControlPass=false bindingsDuringControlPass=0 baselines=7->9)
  --- FAIL: TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass (0.41s)
      --- FAIL: TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass/sibling_device_expiry (0.20s)
      --- FAIL: TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass/sibling_member_expiry (0.21s)
  === RUN   TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat
  === RUN   TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat/during_verifyCurrentServing
      review_r25_test.go:215: TODO(review r25): H-1 (during_verifyCurrentServing): surviving sibling left unserved in Baseline after expireServing early return: eng.serving=false bindings=0 heartbeat={Serving:false PolicyExpires:0s Reason:"-"} lastObs={MemberForwardingEnabled:false Reasons:[] DataHealth:healthy SafetyStatus:passed}
  === RUN   TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat/during_enforce_canRetainServing
      review_r25_test.go:215: TODO(review r25): H-1 (during_enforce_canRetainServing): surviving sibling left unserved in Baseline after expireServing early return: eng.serving=false bindings=0 heartbeat={Serving:false PolicyExpires:0s Reason:"-"} lastObs={MemberForwardingEnabled:false Reasons:[] DataHealth:healthy SafetyStatus:passed}
  --- FAIL: TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat (0.33s)
      --- FAIL: TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat/during_verifyCurrentServing (0.16s)
      --- FAIL: TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat/during_enforce_canRetainServing (0.17s)
  ```
- **Fix & Owner (`agent-core`):**
  1. Separate the per-binding wake deadline (`nextBindingExpiry` for `nextStepDelay`) from the whole-node policy expiry (`e.policyExpires`, when no admitted binding remains valid or the node lease / desired state expires), or in `expireServing(ctx)`: when `e.clock.Mono() >= e.policyExpires` is reached, check `d := Decide(e.input())`; if `d.Serve` is `true` (surviving bindings remain admitted), do **not** call `e.withdraw(cleanup)` at `Step` entry (`engine.go:696`) before the control pass.
  2. In `verifyCurrentServing` (`serving.go:100, 111`) and `enforce`'s `canRetainServing` (`engine.go:1182-1186`): if `e.clock.Mono() >= e.policyExpires` is reached while `Decide(e.input()).Serve` is `true`, do not return early from `Step` or `enforce` in Baseline — let `Step` proceed through `evaluate` and let `enforce` fall through to re-decide (`d = Decide(e.input())`) and apply the surviving bindings' model on the same tick.
  3. In `beat()` (`engine.go:1496-1501`) and `report()` (`observe.go:94-132`), never emit `Serving: false, Reason: "-"` or `MemberForwardingEnabled: false, ForwardingBlockReasons: []` when `!e.serving`.

---

### Finding `M-1` (Medium): `canRetainServing` requires strict `intent.Epoch == h.OpenEpoch` instead of `SameCountingInterval` (and `switchEpoch` does not refresh `e.restartIntent`), forcing a redundant `Plane.Apply` on the next steady-state `304 Not Modified` tick after any ruleset-switch or counter-discontinuity epoch rotation

- **Severity:** Medium
- **Where:**
  - `node/agent/retain.go:47-54` (`canRetainServing` checks `intent.Epoch == h.OpenEpoch`)
  - `node/agent/engine.go:1268,1416` (`e.recordUsageIntent(ctx, model)` runs before `e.plane.Apply`; `e.switchEpoch(ctx)` runs after `e.plane.Apply`)
  - `node/agent/usage.go:349-364` (or `reset.go` where `switchEpoch` calls `e.meter.NotifyRulesetSwitch(clean)` without updating `e.restartIntent`)
  - `node/agent/restart_usage.go:43-70,135-146` (`recordUsageIntent` vs `restartHeadBindings`, which uses `st.SameCountingInterval`)
- **Introduced in:** Follow-up 41 (`77689438`, `node/agent/retain.go:53`), interacting with `SameCountingInterval` / `recordUsageIntent` from follow-up 40 (`b17c8c17`).
- **Scenario:**
  Follow-up 40 fixed `R23-H-1` on the restart path (`restartHeadBindings` in `node/agent/restart_usage.go:135-146`) by introducing `st.SameCountingInterval(ctx, p.Epoch, h.OpenEpoch, e.bootID.String())` (`node/internal/usagestore/continuity.go:14-33`), because the open usage epoch in `usage.db` advances from `E1` to `E2` with `'counter_discontinuity'` while the same serving generation remains active whenever:
  1. `e.switchEpoch(ctx)` at the end of `enforce` (`node/agent/engine.go:1416`) calls `e.meter.NotifyRulesetSwitch(clean)` after `e.recordUsageIntent(ctx, model)` at `engine.go:1268` already recorded the pre-`Apply` epoch `E1` into `e.restartIntent.Epoch` (which happens in production after any baseline fallback recovery, generation repair, or counter decrease observed during `Apply`), or
  2. `Meter.Sample` observes a counter decrease during serving and rotates `h.OpenEpoch` from `E1` to `E2` with `'counter_discontinuity'`.

  However, in follow-up 41 (`77689438`), `canRetainServing` (`node/agent/retain.go:47-54`) was written to check strict equality `intent.Epoch == h.OpenEpoch`:
  ```go
  intent := e.restartIntent
  return intent != nil && h.OpenEpoch != uuid.Nil && h.BootID == e.bootID.String() &&
      intent.Boot == e.bootID && intent.Epoch == h.OpenEpoch &&
      uint64(intent.Generation) == d.Model.Generation && sameUsageBindings(intent.bindings(), d.Model.Bindings)
  ```
  Consequently, after `switchEpoch(ctx)` at the end of `enforce` (or after a counter-discontinuity epoch switch in `Sample`), `e.restartIntent.Epoch` is `E1` while `h.OpenEpoch` is `E2` (with `SameCountingInterval(E1, E2) == true`). On the very next 10-second steady-state `304 Not Modified` poll tick, `canRetainServing` sees `intent.Epoch != h.OpenEpoch`, rejects retention, and executes a redundant full `e.plane.Apply` (inflicting an unnecessary `tc -batch` + `fv_deadline_guard` drop window 10 seconds after the first `Apply` even though `DesiredState` returned `304 Not Modified` and the lease and model are unchanged).

- **Proof (failing unit test in `node/agent/review_r25_test.go`):**
  - `TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick`
  ```text
  === RUN   TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick
  === RUN   TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick/post_apply_switchEpoch
      review_r25_test.go:336: TODO(review r25): M-1 (post_apply_switchEpoch): stale restartIntent.Epoch=01a1a23b-eeb8-7a3b-aabd-db3736a8dee7 != Head.OpenEpoch=01a1a23b-eedd-70ea-b194-d1b70c0a622c (SameCountingInterval=true) forced a redundant Plane.Apply on a steady-state 304 tick (applies 3 -> 4, want 3)
  === RUN   TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick/sample_counter_discontinuity
      review_r25_test.go:336: TODO(review r25): M-1 (sample_counter_discontinuity): stale restartIntent.Epoch=01a1a23b-ef5b-708e-a20c-95a7275e7a7a != Head.OpenEpoch=01a1a23b-ef83-75e0-83fc-39273c8a3299 (SameCountingInterval=true) forced a redundant Plane.Apply on a steady-state 304 tick (applies 3 -> 4, want 3)
  --- FAIL: TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick (0.33s)
      --- FAIL: TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick/post_apply_switchEpoch (0.17s)
      --- FAIL: TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick/sample_counter_discontinuity (0.16s)
  ```
- **Fix & Owner (`agent-core`):**
  Refresh `e.recordUsageIntent` after `e.meter.NotifyRulesetSwitch` in `e.switchEpoch`, and in `canRetainServing` (`node/agent/retain.go:47-54`) accept `intent.Epoch == h.OpenEpoch` or `e.usage.SameCountingInterval(ctx, intent.Epoch, h.OpenEpoch, h.BootID)` (refreshing `e.restartIntent` in place when `SameCountingInterval` is `true` without running `e.plane.Apply`).

---

## 2. Merge recommendation

**Merge after named fixes:** Fix `H-1` and `M-1` in `agent-core` before merging follow-ups 37 to 41 into `main`, because `H-1` causes single-binding expiry on a multi-binding node to blackhole surviving bindings to Baseline across the control/probe pass and, when triggered in `verifyCurrentServing` or `enforce`'s `canRetainServing`, leaves surviving bindings unserved in Baseline with a corrupt `Reason: "-"` heartbeat and contradictory `ObservationReport`.

---

## 3. What held

### Attack Question 1: Does skipping `Apply` ever keep a stale or unsafe ruleset?
- **Renderer input completeness (`node/agent/retain.go:16-26`):** `sameEnforcement` normalizes `Now` and `NowMono` only after checking `applied.NowMono > 0 && next.NowMono >= applied.NowMono && applied.Now.Add(next.NowMono-applied.NowMono).Equal(next.Now)`, substitutes `appliedAssignments` for `applied.QoS.Assignments`, clears consumed `applied.Revoke`, and runs `reflect.DeepEqual(applied, next)` across the entire `enforce.Model` struct (`Generation`, `IPv6Mode`, `Datapath`, `Bindings`, `Leases`, `Resolvers`, `DeniedV4`, `ExtraDeniedV4`, `HostV4`, `QoS`, `Probe`). Any mutation to any renderer input causes `sameEnforcement` to return `false` (`TestISO01_F41_RendererChangesAlwaysApply` in `node/agent/followup41_model_test.go:14-72`).
- **Pending/failure gates (`node/agent/retain.go:30-44`):** `canRetainServing` rejects retention whenever any engine health, recovery, revocation, conntrack, probe-cleanup, or usage-settlement gate is pending (`!e.serving`, `!d.Serve`, `!e.baselineOK`, `!e.storageOK`, `e.applyError != nil`, `e.failedCandidate != nil`, `e.appliedModel == nil`, `e.appliedSnapshot == nil`, `e.leaseBoot != e.bootID`, `e.appliedGen != d.Model.Generation`, `e.policyExpires != d.PolicyExpires`, `e.policyExpires <= e.clock.Mono()`, `e.latchBlocked`, `e.repairing`, `len(e.pendingRevoke) != 0`, `e.conntrackAttempts != 0`, `e.conntrackBlocked`, `len(planRevokes(nil, e.addresses, d.Model.Bindings)) != 0`, `e.memberPath == nil`, `e.memberPathPending`, `e.quarantinePending`, `e.probeCleanup != nil`, `e.retainedPending`, `e.usagePersist`, `e.usageAdopting`, `e.epochClosed`, `e.finalDone`, `e.finalBarrierID != ""`, `e.shutUsage`, `!e.sampleOn`, `!e.expectServing.Load()`, `e.meter == nil`, `e.usage == nil`, `!sameUsageBindings(e.meterBindings(), d.Model.Bindings)`, `e.kernel != nil && e.kernel.nudgePending()`).
- **Node lock, updater latch, live kernel read, and 101ms early-expiry margin (`node/agent/engine.go:1152-1186`, `node/agent/allowed_presence.go:18-90`):** `enforce` acquires `nodelock.Acquire` at line 1152 and re-reads `e.latch()` at line 1170 before calling `e.canRetainServing(ctx, d)` at line 1181. On a retained tick, `enforce` still runs `e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments)` (which wraps `SystemPlane` with `servingPresenceExecutor` / `verifyAllowedPresence` to verify that every admitted prefix is live in `allowed_v4` with remaining timeout `>= 101ms = DeadlineInstallBound + DeadlineGranularity`), runs `e.checkMemberPath`, and updates the watchdog heartbeat via `e.beat()`.
- **Wall-clock jump handling (`node/agent/engine.go:722,989-1001`, `node/internal/timeanchor/anchor.go:57-88`):** Any wall-clock jump exceeding `DefaultBackwardSkew` (2s) or `DefaultForwardSkew` (60s) trips `e.verifyTime()` (`e.anchorOK = false`), causing `Decide` to block serving with `clock_untrusted` (`d.Serve == false`), which withdraws to Baseline unless a fresh signed lease re-anchors the clock.
- **Real control `304 Not Modified` semantics (`internal/nodes/httpwire/state.go:127-147`, `internal/nodes/httpwire/state_cache.go:21-51`):** Real `fvpn-control` returns `errStateUnchanged` (`304 Not Modified`) only when `cached.Signed` has more than half its TTL remaining (`> DesiredStateSeconds/2`), `PolicyRefs` match, no restore operation is pending, and the JCS-canonicalized `DesiredState` (modulo `ConfigGeneration`, `IssuedAt`, `ExpiresAt`) is byte-identical.
- **Proof tests:** `TestR25_Held_AQ1_RetainedTickVerifiesUnderLockAndBlocksOnAnyPendingGateOrClockSkew` (`node/agent/review_r25_test.go`), `TestISO01_F41_RendererChangesAlwaysApply` (`node/agent/followup41_model_test.go`), `TestAT10_F41_SkipVerifiesUnderLockAndHeartbeats` and `TestAT10_F41_Actual304RetainsFullModel` (`node/agent/followup41_tick_test.go`), `TestAT13_F41_LiveDriftOnSkippedTickRepairsOrWithdraws` (`node/agent/followup41_drift_test.go`).

### Attack Question 2: Did the `r24` H-3 fix (per-device deadlines) hold?
- **Per-device timeout vs member-only lease fields (`node/agent/decide.go:663-670,691-696`, `node/enforce/deadline.go:36-62`):** `buildModel` keeps `MemberLease.EffectiveServiceEnd`, `MemberLease.NodeLeaseEnd`, and `MemberLease.DeadlineMono` at the member-only window (`in.LeaseMonoSent + window`), independent of binding slice order, while `memberTimeout` in `node/enforce/deadline.go:36-62` intersects the member deadline (`*lease.DeadlineMono - m.NowMono`) with `b.ValidUntil.Sub(m.Now)` so `enforce.Render` gives each device its own element timeout in `allowed_v4`.
- **Edge cases (`node/agent/decide.go:614-670`, `node/enforce/deadline.go:46-61`):**
  - Distant `ValidUntil` (`+300y`, where `b.ValidUntil.Sub(in.Lease.IssuedAt)` saturates `time.Duration` at `math.MaxInt64`) is capped by `deviceWindow := min(window, b.ValidUntil.Sub(in.Lease.IssuedAt))` before adding `in.LeaseMonoSent`, preventing `int64` wraparound.
  - Zero `ValidUntil` (`b.ValidUntil.IsZero()`) skips per-device tightening and uses the member cap.
  - `ValidUntil` before or equal to `in.Now` (including before `in.Lease.IssuedAt`) and future `ValidFrom` (`in.Now.Before(b.ValidFrom)`) are excluded by `bindingActive(b, in.Now)` at `decide.go:615`, while `nextStepDelay` (`engine.go:649-659`) wakes at `b.ValidFrom.Sub(now)` when a future `ValidFrom` is pending.
  - (When a shorter sibling device `ValidUntil` or member `LeaseEnd` expires while another binding remains valid, `expireServing` exhibits `H-1` above.)
- **Proof tests:** `TestR25_Held_AQ2_PerDeviceDeadlinesAndEdgeCases` (`node/agent/review_r25_test.go`), `TestAT13_F41_DeviceDeadlinesInEitherOrder` and `TestAT13_F41_DistantDeviceDeadlineUsesMemberCap` (`node/agent/followup41_deadline_test.go`), `TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse` (`node/agent/review_r24_test.go`).

### Attack Question 3: Did follow-up 40's `r23` fixes hold?
- **`H-1a` / `H-1b` / `H-1c` (`SameCountingInterval` and `restartHeadBindings` in `node/internal/usagestore/continuity.go:14-33` and `node/agent/restart_usage.go:76-164`):**
  - `SameCountingInterval` queries `rowid` for `first` and `last`, requires `lastRow >= firstRow` and `last.closed_at IS NULL`, and verifies that every epoch in `rowid BETWEEN ? AND ?` has `boot_id = ?`, every non-last row has `closed_at IS NOT NULL AND close_reason = 'counter_discontinuity'`, and every non-first row has `open_reason = 'counter_discontinuity'`. Any intervening `'final_sample'` close, `'after_final_sample'` / `'initial'` open, closed target epoch, reversed epoch order, or `boot_id` change returns `false`.
  - `captureRestartBindings` (`restart_usage.go:80-102`) loads `usage_apply_intent` before `RestartGeneration` so a transient nft read failure keeps the durable intent (`H-1c`), while a live non-zero kernel marker that contradicts the intent clears `e.restartIntent = nil`.
  - (On the steady-state live retention path in follow-up 41, `canRetainServing` did not use `SameCountingInterval`; see `M-1` above.)
- **`H-2` (`epochClosed` vs `finalDone` in `node/agent/usage.go:345-347,474,500` and `node/agent/engine.go:1227-1249`):** `settleRetainedUsage` sets `e.epochClosed = true`, and `handoff` resets `e.finalDone = false` at line 474 until `ClosedCheckpoint`, barrier identity recheck, and `MeterCheckpoint` write succeed (line 500). In `enforce`, lines 1238-1249 require `e.meter.PrepareSampling(ctx)` whenever `e.epochClosed` is `true`, even if a previous `handoff` failed after `FinalSample` before setting `e.finalDone = true`.
- **`H-3a` / `H-3b` / `H-3b′` / `M-1` (`node/agent/usage.go:428-498`, `node/agent/engine.go:1083-1102,1188-1194,1227-1237,1484-1489`):**
  - `handoff`'s `defer` (`usage.go:428-434`) preserves pre-existing `usagePersist`, `meterFailed`, `usageDisk`, and `usageOutbox` flags across a healing `FinalSample` (`H-3a`).
  - `handoff` sets `e.retainedPending = true`, `e.serving = false`, `e.policyExpires = 0`, `e.decision.Serve = false` before `e.baseline(ctx)` (`usage.go:441-450`), so a failed initial baseline in `withdraw` or `handoff` keeps `e.retainedPending = true` and `e.serving = false` (`H-3b`, `M-1`).
  - `handoff` sets `e.usagePersist = true` on `invalidateMeterCheckpoint()`, `ClosedCheckpoint()`, or `writeAtomic(e.cfg.MeterCheckpoint, ...)` failure (`H-3b′`), and `enforce` line 1189 (`e.invalidateMeterCheckpoint()`) blocks `Apply` on subsequent `Step`s if `MeterCheckpoint` invalidation still fails after `settleRetainedUsage` has closed the outgoing epoch.
  - `enforce`'s `defer` (`engine.go:1083-1102`) tracks `usagePersist`, `retainedPending`, and `serving` changes (`usageChanged`) and recomputes `e.decision = Decide(e.input())` and `e.lastObs = time.Time{}`.
- **`L-1` (`node/agent/review_r23_test.go`):** Stale docstrings in `review_r23_test.go` were updated to match the tests.
- **Proof tests:** `TestR25_Held_AQ3_SameCountingIntervalAndHandoffRecoveryInvariants` (`node/agent/review_r25_test.go`), `TestR23_H1c_MarkerReadFailureKeepsIntentAcrossRestart`, `TestR23_H1_IntentCannotCrossSettledBoundaryOrFailedIntentWrite`, `TestR23_H2_PostFinalSampleCheckpointFailureStillPreparesNewEpoch`, and `TestR23_H3_M1_FailedHandoffReportsRecoversAndRestarts` (`node/agent/followup40_test.go`), and all `TestR23_*` tests in `node/agent/review_r23_test.go`.

### Attack Question 4: Did the edited tests lose coverage?
- Audited the full diff (`b17c8c17..8cc27609`) across all 12 edited test files:
  - `node/agent/engine_test.go`
  - `node/agent/followup12_test.go`
  - `node/agent/followup22_test.go`
  - `node/agent/followup23_expiry_test.go`
  - `node/agent/followup24_test.go`
  - `node/agent/followup25_evidence_test.go`
  - `node/agent/followup28_test.go`
  - `node/agent/followup31_restart_test.go`
  - `node/agent/followup35_test.go`
  - `node/agent/review_r15_q7_conntrack_test.go`
  - `node/agent/review_r23_test.go`
  - `node/agent/review_r24_test.go`
- In each edited test, `requireApplyChangeForTest(r)` (`r.eng.cfg.HostReserveBPS++`) or `r.eng.leaseRefreshPending = true` was added only where the test's setup specifically injects a fault into the `Plane.Apply` path on the next `Step` (which previously ran `Apply` unconditionally on every no-op tick due to `r24` H-1).
- Live kernel drift on retained (skipped-`Apply`) ticks (`deleted_element`, `deleted_chain`, `foreign_tc`, `metering_version`) is covered without `requireApplyChangeForTest` by `TestAT13_F41_LiveDriftOnSkippedTickRepairsOrWithdraws` in `node/agent/followup41_drift_test.go:47-105`. No assertions were weakened.

### Attack Question 5: Are the two fixture changes sound?
1. **`node/agent/review_r24_test.go:28-30` (`56829cc4`):**
   - `fakecontrol` (`node/internal/fakecontrol/handlers.go:144-150, 414-420`) re-signs a new `DesiredState` envelope with a new `IssuedAt`, `ExpiresAt`, and `ETag` (`200 OK`) whenever its clock advances, and in `serveBoundGeneration` bumped `Lease.IssuedAt` by `+1ms`, making `Desired.ExpiresAt.Sub(Lease.IssuedAt)` equal `1h - 1ms` on setup and `1h` on the first `+10s` tick.
   - Real `fvpn-control` (`internal/nodes/httpwire/state.go:127-147` and `state_cache.go:21-51`) caches the signed `DesiredState` envelope and returns `304 Not Modified` when `If-None-Match` matches and more than half of `DesiredStateSeconds` remains. Pinning `r.srv.SetClock` after setup makes `fakecontrol` actually return `304 Not Modified` as specified by `TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick`, and `TestAT13_F41_LeaseRenewalApplies` (`node/agent/followup41_tick_test.go:134-152`) separately tests unpinned clock advancement and lease renewal.
2. **`tests/netlab/agent/scenario_at_usage_test.go:45-60` (`f1ed8f3b`):**
   - `"usage checkpoint applied"` is logged by `restoreUsage` at `node/agent/engine.go:758-761` *before* `e.enforce(ctx, e.decision)` at `engine.go:767` finishes `Plane.Apply` (which briefly installs `fv_deadline_guard` at priority `-151`, dropping member packets before `forward` at `-150`).
   - In `inet fvpn` `forward` (`policy drop`), the only rule that accepts member upload traffic (`node/enforce/render_nft.go:179`: `iifname <ingress> oifname <uplink> ip saddr <ip> ip daddr != @denied_v4 counter name "u_<device_id>" accept`) increments `u_<device_id>` in the same atomic rule evaluation as `accept`. Furthermore, line 68 of `scenario_at_usage_test.go` asserts `snap.Bytes == nftAfter` across all bursts sent. Thus, retrying bursts until `nftUsageBytes(t) > before` cannot mask forwarded-but-uncounted traffic.

### Attack Question 6: Anything new?
- Identified `H-1` and `M-1` above.
- Checked `docs/node/followup40-r23-judgments-2026-11-03.md`, `docs/node/followup40-outcomes-2026-11-03.json`, `docs/node/followup41-r24-judgments-2026-11-03.md`, `docs/node/followup41-outcomes-2026-11-03.json`, `docs/node/followup41-test-edit-inventory-2026-11-03.md`, and `## Follow-up 40` / `## Follow-up 41` in the `w2-agent-core` `REPORT.md`; their descriptions match the code as written (subject to `H-1` and `M-1`).

---

## 4. Not examined

- Privileged Docker `netlab` scenarios (`tests/netlab/...`) were inspected statically (`scenario_at_usage_test.go`) but not executed in containers during this review pass.
- Code paths outside the `c98ea61c..8cc27609` diff and its direct callers/callees in `node/agent`, `node/enforce`, `node/internal/usagestore`, `node/internal/fakecontrol`, and `internal/nodes/httpwire` (such as `apps/portal`, `apps/worker`, `node/updater`, and `packaging`) were not re-audited.
