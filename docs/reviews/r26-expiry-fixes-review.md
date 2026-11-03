# Adversarial Security Review — `r26-expiry-fixes-review`

- **Reviewed branch / commit:** `w2-agent-core` follow-up 42 at `8dec4100ce9fba96ea8b0436c41ed8d2b0c75545` (final production commit `b42ed80b0dfaa184f20303f13a90a226925228e2`; `main` `77626af4016443c509f3b14e06cfd4f00f74fb49` merged at `a4b76e09f8b8c0f90c56413c638a4a27ef9eaeda`; carrying follow-ups 37 through 42).
- **Focus:** `agent-core` follow-up 42 (`8cc27609..8dec4100ce9fba96ea8b0436c41ed8d2b0c75545`, excluding `main`'s merge diff in `a4b76e09`):
  1. `r25` **H-1** fix (separation of earliest wake/retention boundary `e.policyExpires` from whole-node watchdog limit `e.servingDeadline()`, same-tick survivor `Apply` without intermediate `Baseline`, expired-only binding settlement via `CheckpointForRestart`, and `reportingDecision()` cause refresh).
  2. `r25` **M-1** fix (`usagestore.SameCountingInterval` in `canRetainServing` and durable-before-memory `recordUsageIntent` refresh on retained ticks).
  3. Netlab **F32-01** fix (per-lease uniform millisecond renewal jitter `[0.9 * base, 1.1 * base]` in `node/agent/sync.go` and atomic SQLite persistence in `node/internal/agentstore/state.go`).
  4. Comment-only update in `node/enforce/model.go` and dated follow-up 42 records under `docs/node/`.

---

## 1. Findings by severity

- **Critical:** 0
- **High:** 0
- **Medium:** 0
- **Low:** 0

No defects or regressions were found in follow-up 42 (`8cc27609..8dec4100ce9fba96ea8b0436c41ed8d2b0c75545`) or across the combined follow-up 37–42 changeset (`77626af4..8dec4100ce9fba96ea8b0436c41ed8d2b0c75545`). All six failing proof tests imported from `r25` (`node/agent/review_r25_test.go`), all follow-up 42 unit tests (`node/agent/followup42_*_test.go`, `node/internal/agentstore/followup42_test.go`), and all six `r26` adversarial proof tests (`node/agent/review_r26_test.go`) pass.

---

## 2. Merge recommendation

**Merge 37 to 42 as they are:** follow-up 42 resolves `r25` H-1, `r25` M-1, and Netlab F32-01 without weakening kernel expiry cut-off, watchdog fail-closed enforcement, usage accounting, or restart/retention invariants, and leaves all pre-existing review tests (`r11`–`r25`) untouched and green.

---

## 3. What held

### AQ1 — Is the later whole-node deadline safe?
- **Separation of earliest wake vs. whole-node watchdog deadline:**
  - `buildModel` (`node/agent/decide.go:627-674`) computes `nodeWindow = min(in.Lease.ExpiresAt.Sub(in.Lease.IssuedAt), in.Desired.ExpiresAt.Sub(in.Lease.IssuedAt), time.Duration(in.Lease.MaxOfflineSeconds)*time.Second)` and caps each admitted member's `window = min(timeanchor.Window(in.Lease.IssuedAt, ml.LeaseEnd, ml.EffectiveServiceEnd), nodeWindow)`. Each member's `enforce.MemberLease.DeadlineMono` (`node/agent/decide.go:657-700`) is `in.LeaseMonoSent + window`, and `Decision.PolicyExpires` (`e.policyExpires`) is the minimum across `nodeDeadline` and every admitted binding's `deviceDeadline` (`in.LeaseMonoSent + min(window, b.ValidUntil.Sub(in.Lease.IssuedAt))`).
  - `e.policyExpires` continues to govern `nextStepDelay` (`node/agent/engine.go:637-639`), `expireServing` (`node/agent/serving.go:120`), `canRetainServing` (`node/agent/retain.go:33-34`), and `enforce`'s post-read deadline check (`node/agent/engine.go:1185-1197`).
  - `e.servingDeadline()` (`node/agent/deadlines.go:35-52`) computes the heartbeat's `PolicyExpires` (`node/agent/engine.go:1390`, `node/agent/boot_heartbeat.go:17`) strictly from the installed `*e.appliedModel`: it returns `0` when `!e.serving || e.policyExpires <= 0 || e.appliedModel == nil`, returns `e.policyExpires` when `len(m.Bindings) == 0`, and otherwise returns the maximum `bindingDeadline(m, b)` across `m.Bindings` (failing closed to `0` if any binding returns `<= 0`).
- **Kernel cut-off at `T1` when the agent stops and the watchdog waits until `T2`:**
  - In `enforce.Apply` (`node/enforce/apply.go:177-192`) and `deviceDeadlines` (`node/enforce/render_nft.go:560-597`), each admitted binding's `@allowed_v4` element timeout is `min(memberTimeout, bindingTimeout)` after subtracting `DeadlineInstallBound` (`100ms`) and truncating down to `1ms`, and `releaseDeadlineGuard` (`node/enforce/deadline_install.go:27-50`) verifies that `nft -f -` completed within `100ms` of `RenderStart`. Thus device 1's `@allowed_v4` element expires in the kernel at or before `T1`.
  - In `enforce.Render` (`node/enforce/render_nft.go:134-228`), `forward` drops any upload packet with `iifname <in> oifname <up> ip saddr != @allowed_v4` (`render_nft.go:138-140`) before per-device DNS upload rules (`render_nft.go:146-156`) and per-device upload accepts (`render_nft.go:172-181`), and drops any download packet with `iifname <up> oifname <in> ip daddr != @allowed_v4` (`render_nft.go:188-190`) before `ct state != { established, related } drop` (`render_nft.go:192`), per-device DNS download accepts (`render_nft.go:193-203`), and per-device download accepts (`render_nft.go:212-221`). `postrouting` SNAT (`render_nft.go:227`) also matches `ip saddr @allowed_v4`. Consequently, the kernel cuts off all traffic (upload, DNS, download, and established flows) for the `T1` device at or before `T1` with zero agent action while `T2` continues until `T2`.
- **`bindingDeadline` clock bases, bounds, and saturation:**
  - `bindingDeadline` (`node/agent/deadlines.go:15-30`) requires a matching member lease in `m.Leases` with `l.DeadlineMono != nil && *l.DeadlineMono > 0 && m.NowMono > 0`, and caps `d := *l.DeadlineMono` with `d = m.NowMono + min(d-m.NowMono, b.ValidUntil.Sub(m.Now))`. Because `m.Now = in.Lease.IssuedAt.Add(m.NowMono - in.LeaseMonoSent)` (`node/agent/decide.go:626,722`), `m.NowMono + b.ValidUntil.Sub(m.Now) == in.LeaseMonoSent + b.ValidUntil.Sub(in.Lease.IssuedAt)`. Capping `min(d-m.NowMono, b.ValidUntil.Sub(m.Now))` before adding `m.NowMono` prevents `time.Duration` overflow even when `b.ValidUntil.Sub(m.Now)` saturates to `math.MaxInt64` (`~292y`). Bindings with future `ValidFrom` are excluded by `buildModel` (`node/agent/decide.go:649-650`) and rejected by `validateServing` (`node/enforce/validate.go:130`).
- **Watchdog rules unchanged:**
  - `node/watchdog/watch.go` was not modified; `Check` (`node/watchdog/watch.go:116-157`) still enforces Baseline on missing/corrupt heartbeat, `BootID` mismatch, `!hb.Serving`, `!hb.BaselineOK`, stale `Mono`, or `nowMono >= hb.PolicyExpires`.
- **Proof tests:** `TestR26_Held_AQ1_ServingDeadlineAndKernelCutOff` (`node/agent/review_r26_test.go:53`), `TestAT13_F42_WatchdogAcrossSiblingAndLastDeadline` (`node/agent/followup42_deadline_test.go:108`), `TestAT13_F42_WholeNodeDeadlineKeepsEveryCap` (`node/agent/followup42_deadline_test.go:223`).

### AQ2 — Survivors' `Apply` at a sibling's deadline
- **All three `expireServing` call sites and `enforce`'s retained-serving branch:**
  - `expireServing` (`node/agent/serving.go:119-136`), called at `Step` entry (`node/agent/engine.go:698`), at the start of `verifyCurrentServing` (`node/agent/serving.go:100`), and after `verifyServingModel` in `verifyCurrentServing` (`node/agent/serving.go:111`), refreshes desired state, re-evaluates `e.decision = Decide(e.input())`, and returns `false` without withdrawing when `e.policyExpires > 0 && e.decision.Serve`.
  - In `enforce` (`node/agent/engine.go:1181-1209`), after the locked `verifyServingModel`, `expireServing`, and `checkMemberPath` calls, `enforce` re-evaluates `d = Decide(e.input()); e.decision = d`. When a sibling binding expired before or during those reads while another binding remains valid, `d.PolicyExpires` (`T2`) differs from `e.policyExpires` (`T1`) and `d.Model.Bindings` shrinks, so `canRetainServing(ctx, d)` returns `false` and execution falls through to `Plane.Apply` on that same tick without an intermediate `Baseline`.
- **Last binding or node-level expiry:**
  - When the last binding expires, or the node lease, desired state, or `MaxOfflineSeconds` window expires, `Decide(e.input()).Serve` is `false` (with `LEASE_EXPIRED` or `AUTHORIZATION_EXPIRED`), so `expireServing` (`node/agent/serving.go:130-135`) and `enforce` (`node/agent/engine.go:1193-1195`) withdraw immediately to Baseline (`e.serving = false`, `e.policyExpires = 0`).
- **Early return after a sibling's deadline is crossed:**
  - If `Step` returns early before `Plane.Apply` (e.g., on `nodelock.ErrBusy` at `node/agent/engine.go:1156` or caller context cancellation at `node/agent/engine.go:720`), `finishServingStep` (`node/agent/serving.go:174-194`) runs `verifyCurrentServing` on `*e.appliedModel` (`verifyAllowedPresence` in `node/agent/allowed_presence.go:83-85` permits the expired sibling's IP to be absent from `@allowed_v4` while requiring the surviving sibling's IP) and writes a truthful heartbeat (`Serving: true`, `PolicyExpires: T2`, `Reason: "serving"`). Because `e.policyExpires` remains `T1 <= now` until `Apply` completes, `nextStepDelay` returns `0` so the next tick retries immediately. If verification fails or the last binding has also expired, `finishServingStep` withdraws and `reportingDecision()` (`node/agent/observe.go:425-435`) populates the current block reason (never `"-"`).
- **Proof tests:** `TestR26_Held_AQ2_SurvivorsApplyAndEarlyReturnTruthfulness` (`node/agent/review_r26_test.go:196`), `TestAT13_F42_SiblingDeadlineAppliesWithoutBaseline` (`node/agent/followup42_deadline_test.go:60`), `TestAT13_F42_ReportingRefreshesExistingCause` (`node/agent/followup42_deadline_test.go:203`), `TestR25_H1a_SiblingExpiryOnStepEntryTearsDownWholeNodeToBaselineBeforeControlPass` (`node/agent/review_r25_test.go:64`), `TestR25_H1b_SiblingExpiryDuringVerifyOrRetainLeavesSurvivingSiblingInBaselineWithDashHeartbeat` (`node/agent/review_r25_test.go:144`).

### AQ3 — The expired-only removal accounting path
- **Exactness of `onlyExpiredBindingsRemoved` (`node/agent/deadlines.go:57-81`):**
  - Requires `e.serving`, `e.appliedModel != nil`, `e.appliedModel.Generation == next.Generation`, `sameUsageBindings(e.meterBindings(), e.appliedModel.Bindings)`, `0 < len(next.Bindings) < len(e.appliedModel.Bindings)`, exact `reflect.DeepEqual(old, b)` for every surviving binding in `next.Bindings`, `len(remaining) == 0` (rejecting any added device), and `d := bindingDeadline(*e.appliedModel, old)` with `0 < d <= e.clock.Mono()` for every removed binding.
- **Live verification and failure handling in `settleExpiredBindings` (`node/agent/deadlines.go:83-96`):**
  - Runs `e.verifyServingModel(ctx, *e.appliedModel, e.appliedAssignments)` first; if live verification fails, `verifyServingModel` withdraws to Baseline and `settleExpiredBindings` returns `false`.
  - Then calls `e.meter.CheckpointForRestart(ctx)` (which pauses sampling and commits the current interval deltas under the old bindings without closing the epoch). If `e.meter == nil` or `CheckpointForRestart` returns an error, `settleExpiredBindings` sets `e.usagePersist = true`, calls `e.withdraw(ctx)`, and returns `false` without replacing `e.restartIntent` or running `Apply`.
- **Post-checkpoint counter behavior across samples, resets, withdrawal, and restart:**
  - Because `d <= e.clock.Mono()` and the kernel `@allowed_v4` element timeout was installed with `timeout <= d - RenderStart - 100ms` inside a `< 100ms` guard window, the expired device's IP is already expired in `@allowed_v4` before `settleExpiredBindings` checkpoints its counter, and `forward`'s `@allowed_v4` drop rules (`node/enforce/render_nft.go:140,190`) prevent any further packets from reaching `u_<expired>` or `d_<expired>`.
  - After `Apply` installs the surviving subset and `e.setMeterBindings(model.Bindings)` removes the expired device from `e.meterBindings()`, `u_<expired>` and `d_<expired>` remain in the kernel table at their checkpointed values. On subsequent `Meter.Sample`, `FinalSample`, or restart `planContinuity` / `attribute` (`node/internal/meter/delta.go:98-125, 457-504`), `u_<expired>` has `delta = c.Bytes - p.Bytes == 0`; `attribute` skips zero-delta devices at `delta.go:490-492` before checking `a.member == uuid.Nil`, so removed-device bytes are neither double-counted nor lost nor turned into a `mapping conflict` gap.
  - Across crashes between `settleExpiredBindings` (`engine.go:1251`), `recordUsageIntent` (`engine.go:1296`), `Plane.Apply` (`engine.go:1310`), and `SetMeta("retained_applied")` (`engine.go:1360`), `D_exp`'s final delta is already committed in `usage.db` before `usage_apply_intent` is narrowed to survivors, and `D_surv`'s identity is unchanged in both intents.
- **Proof tests:** `TestR26_Held_AQ3_ExpiredOnlyRemovalExactnessAndAccounting` (`node/agent/review_r26_test.go:259`), `TestMTR03_F42_ExpiredRemovalSettlesBytesWithoutClosingInterval` (`node/agent/followup42_deadline_test.go:155`), `TestMTR03_F42_ExpiredSettlementFailureKeepsIntentAndCloses` (`node/agent/followup42_deadline_test.go:259`).

### AQ4 — M-1's fix (`SameCountingInterval` in `canRetainServing`)
- **Both rotation producers handled:**
  - `canRetainServing` (`node/agent/retain.go:48-62`) takes the fast path when `intent.Epoch == h.OpenEpoch` and otherwise checks `e.usage.SameCountingInterval(ctx, intent.Epoch, h.OpenEpoch, e.bootID.String())`. Both `r25` M-1 rotation producers—post-`Apply` `switchEpoch` (`node/agent/meterreset.go:255-263`) and `Meter.Sample` `planSwitch` (`node/internal/meter/delta.go:374-429`)—close the prior epoch and open its successor on the same `boot_id` with `counter_discontinuity`, which `SameCountingInterval` (`node/internal/usagestore/continuity.go:14-33`) accepts.
- **Real counting boundaries rejected:**
  - `SameCountingInterval` rejects `final_sample`, `baseline`, `after_final_sample`, `continuity_unknown`, `power_loss`, missing origin epochs, closed target epochs, and `boot_id` changes.
- **Durable-before-memory intent update and failure handling:**
  - In `enforce` (`node/agent/engine.go:1197-1208`), after the second `canRetainServing(ctx, d)` check passes, `e.recordUsageIntent(ctx, d.Model)` (`node/agent/restart_usage.go:43-70`) writes `usage_apply_intent` via `e.store.SetMeta` (`restart_usage.go:65`) before updating `e.restartIntent = &intent` in memory (`restart_usage.go:68`). If `SetMeta` fails, `enforce` sets `e.usagePersist = true`, leaves `e.restartIntent` on the prior epoch, and withdraws to Baseline (`engine.go:1201-1205`). A crash before `SetMeta` leaves the older continuity-valid epoch on disk (accepted by `restartHeadBindings` via `SameCountingInterval` at `restart_usage.go:139`); a crash after `SetMeta` leaves `h.OpenEpoch` on disk (`p.Epoch == h.OpenEpoch` at `restart_usage.go:136`).
- **Proof tests:** `TestR26_Held_AQ4_SameCountingIntervalRetentionAndIntentPersistence` (`node/agent/review_r26_test.go:424`), `TestMTR03_F42_RetainedEpochIntentIsDurable` (`node/agent/followup42_epoch_test.go:37`), `TestMTR03_F42_RetentionRejectsCountingBoundaries` (`node/agent/followup42_epoch_test.go:67`), `TestMTR03_F42_IntentRefreshFailureWithdraws` (`node/agent/followup42_epoch_test.go:100`), `TestR25_M1_EpochSwitchDuringServingLeavesIntentEpochStaleAndForcesRedundantApplyOnNext304Tick` (`node/agent/review_r25_test.go:254`).

### AQ5 — F32-01's jitter
- **Bounds, resolution, and uniform sampling:**
  - `leadJitter` (`node/agent/sync.go:264-277`) computes `span := lead / 10 / time.Millisecond` and draws `n` from `[0, 2*span]` via `crypto/rand.Int(source, big.NewInt(int64(2*span+1)))` (rejection sampling, zero modulo bias), returning `(time.Duration(n.Int64()) - span) * time.Millisecond`.
  - At the default 120s lead and a 600s lease window, `renewLead(600s)` (`node/agent/engine.go:1552-1561`) is `120s`, `span = 12000`, and the effective lead `e.renewLead(window) + e.renewalJitter(window)` (`node/agent/sync.go:166`) is uniformly distributed over `[108s, 132s]` in `1ms` steps (`24,001` points).
  - At the `window/2` cap (`window >= 2s`), `base = window / 2` and the effective lead is in `[0.45 * window, 0.55 * window]` (`> 0` and `< window`). At the `1s` floor on sub-2s windows (`0 < window < 2s`), `base = 1s` and the effective lead is in `[900ms, 1100ms]`.
- **Lease-scoped reuse, persistence, legacy fallback, and rollback:**
  - `renewalJitter` (`node/agent/sync.go:286-295`) caches the draw by `(e.lease.LeaseId, e.lease.IssuedAt)` and clamps `e.leaseJitter.value` to `[-span, +span]` of the current `e.renewLead(window)`.
  - `SaveLease` (`node/internal/agentstore/state.go:341-361`) writes the `lease` row and `meta` key `'lease_renew_jitter_ns'` in a single SQLite transaction (`s.tx`), rolling back both on error. `Lease` (`state.go:364-401`) reads `COALESCE((SELECT value FROM meta WHERE key='lease_renew_jitter_ns'), '0')`, falling back to `0` when the meta row is absent and failing closed on corrupt/non-integer values.
- **Proof tests:** `TestR26_Held_AQ5_RenewalJitterBoundsPersistenceAndClamping` (`node/agent/review_r26_test.go:462`), `TestAT13_F32_01_UniformMillisecondRenewalJitter` (`node/agent/followup42_jitter_test.go:23`), `TestAT13_F32_01_RenewalDrawOncePerLease` (`node/agent/followup42_jitter_test.go:56`), `TestAT13_F32_01_FailedRandomReadKeepsSafeLead` (`node/agent/followup42_jitter_test.go:82`), `TestAT13_F32_01_LeaseReplayAndRestartKeepDraw` (`node/agent/followup42_jitter_test.go:99`), `TestAT13_F42_LeaseAndRenewalJitterCommitTogether` (`node/internal/agentstore/followup42_test.go:12`).

### AQ6 — `node/enforce/model.go` and `docs/node/followup42-*`
- `git diff 8cc27609..8dec4100ce9fba96ea8b0436c41ed8d2b0c75545 -- node/enforce/` touches only the doc comment on `MemberLease.DeadlineMono` in `node/enforce/model.go:110-114` (3 comment lines replacing 2 comment lines) and alters no executable code.
- `docs/node/followup42-r25-judgments-2026-11-03.md`, `docs/node/followup42-consumers-2026-11-03.md`, `docs/node/followup42-outcome-2026-11-03.md`, `docs/node/followup42-validation-2026-11-03.json`, and `docs/node/followup42-r25-before-2026-11-03.txt` match the code at `8dec4100ce9fba96ea8b0436c41ed8d2b0c75545` (production commit `b42ed80b0dfaa184f20303f13a90a226925228e2`, merge commit `a4b76e09f8b8c0f90c56413c638a4a27ef9eaeda`, `main` `77626af4016443c509f3b14e06cfd4f00f74fb49`, `functional_only: true`, `D133_measurement_run: false`).
- **Proof test:** `TestR26_Held_AQ6_AQ7_ValidationRecordAndZeroDecisionContract` (`node/agent/review_r26_test.go:533`).

### AQ7 — Edited tests and regressions
- Follow-up 42 (`8cc27609..8dec4100ce9fba96ea8b0436c41ed8d2b0c75545`, excluding `main`'s merge `a4b76e09`) modified zero pre-existing `*_test.go` files; `node/agent/review_r25_test.go` was imported byte-identically from `1afecb29` in `bf68757f`, and all new follow-up 42 tests were added in `node/agent/followup42_{deadline,epoch,jitter}_test.go` and `node/internal/agentstore/followup42_test.go`.
- All invariants across `Step`, `Apply`, heartbeat, watchdog, usage accounting, usage upload, and restart adoption held across the `r11`–`r26` review suites and package test suites.
- **Proof test:** `TestR26_Held_AQ6_AQ7_ValidationRecordAndZeroDecisionContract` (`node/agent/review_r26_test.go:533`).

---

## 4. Not examined

- Privileged Docker netlab execution (`tests/netlab`) and D-133 paired `A,B,A,B` timing benchmarks (not re-run in this review worktree; static code and unit/integration proof suites were verified).
- Physical `linux/arm64` hardware execution and live external NetBird / AdGuard / PostgreSQL services.
- Central control-plane (`apps/control`, `apps/worker`) and web portal (`apps/portal`) packages outside the node/enforcement/watchdog/usage scope.
