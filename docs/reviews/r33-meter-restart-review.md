# Adversarial Review r33: `agent-core 53` Meter Restart Checkpoint (`b094ffac`)

## Scope and Immutable Base

| Item | Value |
|---|---|
| Review task / branch | `r33-meter-restart-review` |
| Immutable review base (`HEAD` at start) | `29cd267162fa614c444122865b62f405e2f58802` (`agent: correct D-141 diagnosis and reproduce held-sample timing exposure`) |
| Meter restart change commit | `b094ffac72f2f4f247a3e2b592a6659a84f1d51a` (`agent: fix D-141 node CI failures and record repeat proofs`) |
| Pre-change comparison parent | `9e52ea20675daae89710bc4630906e1c70b2e650` (`agent: diagnose recent node CI failures (D-141)`) |
| `29cd2671:node/internal/meter/delta.go` blob | `112260bd296d31cfd382d4faf03bf1efe50d7729` (byte-identical to `b094ffac`) |
| `29cd2671:node/internal/meter/meter.go` blob | `ac6ff0b548b5c0bdd6a3c5e44ad4b04d34dec3f9` (byte-identical to `b094ffac`) |
| Reviewed product diff scope | `9e52ea20675daae89710bc4630906e1c70b2e650..b094ffac72f2f4f247a3e2b592a6659a84f1d51a` on `node/internal/meter/{delta,meter}.go` (with byte identity verified at `29cd267162fa614c444122865b62f405e2f58802`) |

### Exact Product Change Under Review (`9e52ea20..b094ffac` on `node/internal/meter/`)

1. `node/internal/meter/delta.go:243-284`:
   - `planSample(head, obs, now, interval, final)` (`delta.go:243-245`) delegates to `planSampleWithBarrier(head, obs, now, interval, final, false)` (`delta.go:247-341`).
   - Inside `planSampleWithBarrier`:
     - First, if `head.OpenEpoch == uuid.Nil` (`delta.go:249-255`), `final` returns `ErrNoEpoch` and non-`final` runs `planContinuity(head, obs, now)`.
     - Second, `cur := usageCounters(obs)` and `resets := enforce.DetectResets(counterMap(head.Counters), counterMap(cur))` (`delta.go:256-257`) are computed, and if `bootChanged(head.BootID, obs.BootID) || len(resets) > 0` (`delta.go:258-263`), `planSampleWithBarrier` returns `planSwitch(head, obs, cur, now, false, interval)`.
     - Third, when `!head.LastSample.IsZero() && !now.After(head.LastSample)` (`delta.go:264-291`) and `head.LastSample.Sub(now) <= interval` (`delta.go:265-284`):
       - The unchanged-counter condition was widened from `if final && sameReadings(head.Counters, cur)` to `if (final || restart) && sameReadings(head.Counters, cur)` (`delta.go:270`).
       - When that condition holds, both `final` and `restart` construct `usagestore.Commit{Epoch: head.OpenEpoch, Counters: head.Counters, BootID: obs.BootID, SampledAt: head.LastSample}` (`delta.go:271-276`), preserving `OpenEpoch`, `Counters`, and `LastSample` without emitting an event (`Event: nil`).
       - Only `if final` sets `c.CloseEpoch = true` and `c.CloseReason = "final_sample"` (`delta.go:277-280`).
       - Otherwise (when `!final && !restart`, or when `!sameReadings(head.Counters, cur)` after passing the no-boot-change and no-reset/no-disappearance checks), `planSampleWithBarrier` returns `plan{hold: true}, nil` (`delta.go:283`), which `Meter.sample` translates into `ErrHold` (`meter.go:249-251`).
2. `node/internal/meter/meter.go:216,245`:
   - Under `m.mu` in `m.sample(ctx, final)` (`meter.go:201-217`), `restart := m.inFinal && !final` (`meter.go:216`) is captured and passed to `planSampleWithBarrier(head, obs, m.nowUTC(), m.interval, final, restart)` (`meter.go:245`).

---

## Fidelity Table

| Area | Requirement / Contract | Code (`29cd267162fa614c444122865b62f405e2f58802`) | Observed Behavior | Independent Proof & Structural Limits |
|---|---|---|---|---|
| **Clock domain** | `MTR01`–`MTR04`, `§10.5`, `§10.7`, `§10.9`, `D-112` (10 s default sampling interval bound), `D-130` (newly minted event intervals have UTC whole-microsecond endpoints `>= 1 µs` via `eventInterval`; central validates canonical nanosecond `interval_order` and stores microsecond projection; sealed 1 ns events are never rewritten) | `node/internal/meter/delta.go:23-25,148-165,243-299,343-363`, `node/internal/meter/meter.go:166,245-256,485-519`, `node/internal/meter/period.go:6,27-41` | When `head.OpenEpoch != uuid.Nil`, `BootID` is unchanged, `DetectResets` is empty, `!head.LastSample.IsZero()`, and `0 <= head.LastSample.Sub(now) <= interval` (including `now == head.LastSample`, `0 < rollback < interval`, and exact equality `rollback == interval`) with unchanged readings: `Sample` gets `plan{hold: true}` and returns `ErrHold`; `CheckpointForRestart` commits `Epoch: head.OpenEpoch, Counters: head.Counters, SampledAt: head.LastSample` (`CloseEpoch: false, Event: nil`) and returns `nil`; `FinalSample` commits the same checkpoint with `CloseEpoch: true, CloseReason: "final_sample"` and returns `nil`. When `head.LastSample.Sub(now) > interval` (`interval + 1ns` or `interval + 1µs` with unchanged `BootID` and no reset), all three run `planClockGap` (`delta.go:285-290,343-363`), emitting a `1µs` `clock_anomaly` gap at `[head.LastSample, head.LastSample + 1µs)` on `head.OpenEpoch` (and `FinalSample` closes that same `head.OpenEpoch` in the same commit via `delta.go:286-289`). For `now.After(head.LastSample)` and for `head.LastSample.IsZero()`, `restart` is not consulted. Under `D-130`, when `LastSample == T` (whole microsecond), a forward call at `T + 331ns` has `now.After(LastSample) == true` and `eventInterval` advances `LastSample` to `T + 1µs`; a subsequent call at `T + 800ns` then sees `!now.After(LastSample)` (`LastSample == T + 1µs`) and enters the equal/rollback branch. | Proven by `TestR33_Q1_ClockDomainAndFidelity` (`equal_timestamp`, `bounded_rollback_5s`, `exact_interval_boundary_10s`, `over_interval_by_1ns`, `over_interval_by_1us`, `forward_time_unchanged_readings`, `sub_microsecond_same_microsecond_as_last_sample`, `open_epoch_with_zero_last_sample`; subtest 8 uses direct SQLite `UPDATE meta SET last_sample = NULL` to reach `head.LastSample.IsZero()` with an open epoch). Note: comments at `delta.go:23-24` and `docs/node/usage-meter.md:84` say "less than one sample interval" while `delta.go:265` implements `<= interval`; `§10.9` does not mandate `<` over `<=`. |
| **Restart durable checkpoint** | `MTR02`, `MTR03`, `§§10.4–10.6`, `§13.6`, `AT14`–`AT17` (atomic SQLite checkpoint and outbox event stream; exact-once accounting; immutable `payload_sha256`; decimal strings up to `2^64-1`) | `node/internal/meter/delta.go:270-281`, `node/internal/meter/meter.go:257-260,285-314`, `node/internal/usagestore/store.go:446-630,690-848`, `node/internal/usagestore/ack.go:88-219,623-670`, `node/internal/usagestore/snapshot.go:39-193` | An accepted unchanged `CheckpointForRestart` at `now <= head.LastSample` returns `plan{commit: ...}` with `p.hold == false, p.noop == false` and `Event == nil, CloseEpoch == false, SampledAt == head.LastSample`. `m.persist` (`meter.go:285-314`) executes `s.Commit` (`store.go:446-472`), which runs a real SQLite write transaction (`StepRead`, `StepPlan`, `StepCheckpoint`, `StepBeforeCommit`, `StepAfterCommit`, skipping `StepInsert` at `store.go:531-539`), deletes and re-inserts the `counters` rows and updates `meta.last_sample` (`writeCursor`, `store.go:690-754`), and preserves `OpenEpoch`, `OpenReason`, `LastCloseReason`, `HasHistory`, `BootID`, `LastSample`, `Counters` (including `> 2^53` byte totals), `NextSeq`, `AckedThrough`, `LedgerEpoch`, `BlockedSeq`, archived and outbox event payloads/hashes, `ReadSnapshot`, and `OutboxStatus` across close/reopen. Repeated calls are idempotent and the next forward positive delta is counted once from `head.LastSample`. | Proven by `TestR33_Q2_DurableCheckpointAndExactOnce` and `TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability`. |
| **Ordinary `Sample`** | `MTR01`, `§10.5`, `§10.9` (periodic sampling holds without error log on bounded wall-clock rollback `<= interval` when epoch/boot/counters are continuous; quiet forward interval updates checkpoint without emitting an event) | `node/internal/meter/meter.go:195-199,216,245-251,451-456`, `node/internal/meter/delta.go:243-284` | `Sample` calls `m.sample(ctx, false)` with `final = false` and `restart = m.inFinal && !final` (`false` when not inside `pausedSample`). Provided `head.OpenEpoch != uuid.Nil`, `!head.LastSample.IsZero()`, `BootID` is unchanged, and `DetectResets` reports no counter decrease or disappearance, `planSampleWithBarrier` at `0 <= head.LastSample.Sub(now) <= interval` returns `plan{hold: true}` (which `Meter.sample` returns as `ErrHold` without writing SQLite) whether readings are unchanged, increased, or have an added counter. (If `head.OpenEpoch == uuid.Nil`, `BootID` changed, or a counter decreased/disappeared, those earlier branches execute instead of `ErrHold`.) `Run` (`meter.go:453`) ignores `ErrHold` without logging a warning. | Proven by `TestR33_Q1_ClockDomainAndFidelity` and `TestR33_Q3_ChangedReadingsAndAttribution`. |
| **`FinalSample`** | `§10.7`, `§13.6` steps 8 and 11, `D-125` (`ClosedCheckpoint` requires `OpenEpoch == uuid.Nil && LastCloseReason == "final_sample"`) | `node/internal/meter/meter.go:216,241-244,261-281,353-356`, `node/internal/meter/delta.go:249-290`, `node/internal/usagestore/checkpoint.go:23-46` | `FinalSample` calls `m.pausedSample(ctx, true)` (`final = true, restart = false`). When `head.OpenEpoch == uuid.Nil`, `Meter.sample` returns `ErrNoEpoch` (`meter.go:241-244`). At `0 <= head.LastSample.Sub(now) <= interval` (with unchanged `BootID` and no reset/disappearance), if `sameReadings(head.Counters, cur)` is true, it commits `CloseEpoch: true, CloseReason: "final_sample"` on `head.OpenEpoch` at `SampledAt: head.LastSample`; if `sameReadings` is false, it returns `ErrHold`. On rollback `> interval` (with unchanged `BootID` and no reset), `planClockGap` emits a `clock_anomaly` gap on the existing `head.OpenEpoch` (`p.commit.Open == nil`) and `delta.go:286-289` sets `p.commit.CloseEpoch = true, p.commit.CloseReason = "final_sample"` in that single commit. By contrast, when `bootChanged` or `len(resets) > 0` runs `planSwitch` (`delta.go:258-263,382-438`), the first commit closes the old epoch and opens a new epoch (`p.commit.Open != nil`), which triggers the second commit in `Meter.sample` (`if final && p.commit.Open != nil`, `meter.go:261-281`) to close that newly opened epoch with `"final_sample"`. | Proven by `TestR33_Q1_ClockDomainAndFidelity`, `TestR33_Q4_PrecedenceAndClosedEpochs`, and `TestR33_Q6_ConcurrentCallersAndBarrierSemantics`. |
| **Reset/boot precedence** | `MTR02`, `§10.7`, `§13.7`, `AT17` (counter reset/decrease/disappearance or `boot_id` change must emit `counter_discontinuity` or `power_loss` gap and open a new epoch; never hidden by unchanged-counter shortcut) | `node/internal/meter/delta.go:249-263,382-438` | In `planSampleWithBarrier`, `head.OpenEpoch == uuid.Nil` (`delta.go:249`), `DetectResets(counterMap(head.Counters), counterMap(cur))` (`delta.go:257`), and `bootChanged(head.BootID, obs.BootID) || len(resets) > 0` (`delta.go:258`) execute strictly before `if !head.LastSample.IsZero() && !now.After(head.LastSample)` (`delta.go:264`). Equal-looking counter numbers across a `BootID` change at `now <= head.LastSample` emit a `power_loss` gap (zero records) and open a new epoch; a counter byte/packet decrease or counter disappearance at `now <= head.LastSample` emits a `counter_discontinuity` gap (preserving positive deltas of non-reset devices) and opens a new epoch. | Proven by `TestR33_Q4_PrecedenceAndClosedEpochs` and `TestR33_Q3_ChangedReadingsAndAttribution/counter_removed_or_renamed_triggers_reset_switch`. |
| **Attribution** | `MTR01`, `MTR02`, `§§10.1–10.3`, `§10.7`, `D-019` (per-device `forward_admission_v1` upload/download attribution to `(member_id, binding_revision)`; DNS `nu_`/`nd_` and diagnostic counters excluded; mid-epoch binding change or missing/conflicting binding on positive delta emits `gap`) | `node/internal/meter/delta.go:37-125,270-283,301-340,365-380,440-513` | At `0 <= head.LastSample.Sub(now) <= interval` (after the no-boot-change and no-reset/no-disappearance checks), any increase in `Bytes` or `Packets` on an existing counter or any newly added counter (`len(head.Counters) != len(cur)`) causes `sameReadings` (`delta.go:365-380`) to return `false` (`plan{hold: true}` -> `ErrHold`), leaving the uncheckpointed delta intact for the next forward sample. `sameReadings` compares `(Name, Packets, Bytes)` and ignores `(MemberID, BindingRevision)`, while the accepted unchanged commit writes `Counters: head.Counters` (`delta.go:273`). Thus an unchanged checkpoint at `now <= head.LastSample` never overwrites stored attribution in `head.Counters` if `obs.Bindings` changed or had duplicate conflicting entries; if a later forward sample in the same epoch has a positive delta under a different `(MemberID, BindingRevision)`, `attribute()` (`delta.go:485-505`) sees `a.hadOld && (a.oldMem != a.member || a.oldRev != a.rev)` and emits a fail-closed `gap` (`reason: unknown, detail: "mapping conflict"`). If `head.Counters` had `(uuid.Nil, 0)` from an initially unbound baseline, `a.hadOld` is `false` so the next forward positive delta attributes cleanly to `cur`'s binding. | Exercised in `TestR33_Q3_ChangedReadingsAndAttribution` (byte increase `> 2^53`, packet-only increase, added zero/non-zero counter, removed counter, mid-epoch binding change followed by forward positive delta, duplicate conflicting bindings at rollback followed by clean forward delta, and initially unbound baseline followed by unchanged checkpoint and clean forward delta). Forward positive delta with missing (`uuid.Nil`) binding (`delta.go:503`) and DNS/diagnostic counter exclusion (`parseUsageCounter` at `delta.go:51-75`) were verified by source inspection and existing `meter_test.go`. |
| **Health/errors** | `MTR04`, `§10.7`, `§10.8`, `§13.6` step 11, `AT18`, `AT19` (fail closed on counter read failure `meter_path_invalid` or SQLite failure `persist_failed`; `ErrHold` must not clear active failure latches) | `node/internal/meter/meter.go:225-260,285-300,425-439,521-545`, `node/internal/usagestore/store.go:153-179,474-485`, `node/internal/usagestore/ack.go:418-594`, `node/agent/restart.go:174-195`, `node/agent/deadlines.go:83-96` | `src.Read` failure calls `m.noteSource(err)` (`sourceFailed = true` -> `ReasonMeterPath` in `m.Health` when the store has no gate reason of its own, `meter.go:430-437`). Store `Head` or non-injected `Commit` failure calls `m.store.NotePersistFailure()` (`ReasonPersistFail` in `s.EvaluateGate`, `ack.go:505-507`). When `planSampleWithBarrier` returns `plan{hold: true}`, `m.sample` returns `ErrHold` at `meter.go:249-251` before `m.persist`, `m.clearSource`, and `m.markSampled`, so `ErrHold` neither clears `sourceFailed` / `persistFailed` nor updates `lastMono`. A subsequent successful `CheckpointForRestart` executes both `src.Read` and `store.Commit`, clearing `sourceFailed` and `persistFailed` and updating `lastMono`. In `node/agent/`, `preserveServingOnStop` (`restart.go:174-195`, calling `CheckpointForRestart` at line `190`) and `settleExpiredBindings` (`deadlines.go:83-96`, calling `CheckpointForRestart` at line `90`) both require `CheckpointForRestart(ctx) == nil` alongside their verification checks before retaining serving rules or applying a surviving sibling model. | Proven by `TestR33_Q5_ErrorsHealthAndBarriers` (using a `BEFORE INSERT ON counters` SQLite trigger for non-injected write failure), `TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback`, and `TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback`. Note: `usagestore.Store.hookStep` (`store.go:474-485`) wraps all `Options.Hook` errors in `ErrInjectedCrash`, and `meter.go:288-294` checks `if seq != 0 || p.commit.Event == nil` and calls `NotePersistSuccess()`; because `Options.Hook` is test-only (nil in production), production SQLite errors never take the `ErrInjectedCrash` branch. |
| **Concurrent callers** | `§10.5`, `§13.6` (`Barrier` pauses periodic `Run` and waits for in-flight `Sample`; `CheckpointForRestart` / `FinalSample` run under `pausedSample` and leave `m.paused = true` until `ResumeSampling`) | `node/internal/meter/meter.go:201-223,319-384,414-420,443-478` | When an in-flight `Sample` is blocked in `src.Read`, `CheckpointForRestart` (or `FinalSample`) sets `m.paused = true` and waits on `m.cond.Wait()` in `Barrier(ctx)` (`meter.go:319-351`). Once the in-flight `Sample` commits at timestamp `T`, `CheckpointForRestart` runs at `now == T == head.LastSample` with unchanged readings and returns `nil`. While `CheckpointForRestart` is in flight (`m.sampling = true, m.inFinal = true`), concurrent `Sample` calls hit `if m.sampling` (`meter.go:211-214`) and return `ErrBusy` without modifying `m.inFinal`. On return, `m.inFinal` is `false` and `m.paused` remains `true` (`Sample` returns `ErrPaused` at `meter.go:207-210`) until `ResumeSampling()`. After `Close()`, all methods return `ErrClosed`. | Proven by `TestR33_Q6_ConcurrentCallersAndBarrierSemantics` (`checkpoint_waits_for_in_flight_sample_then_succeeds_at_same_timestamp` and `sample_called_while_checkpoint_in_flight_returns_errbusy_and_does_not_steal_restart`). Structural limit: in `pausedSample` (`meter.go:367-384`), `m.mu` is unlocked between `Barrier(ctx)`, setting `m.inFinal = true` (`line 376`), and locking `m.mu` in `m.sample` (`line 202`), whereas `PrepareSampling` (`meter.go:390-412`) sets `m.sampling = true` before unlocking `m.mu` (see *Not Examined*). |
| **Period boundary** | `MTR01`, `§§10.2–10.4`, `§10.9` (UTC calendar month `YYYY-MM`; sample within one interval after UTC month boundary is stamped on boundary as `complete` for prior month; missed boundary `> interval` is marked `boundary_estimated`; `gap` takes precedence over `boundary_estimated`) | `node/internal/meter/period.go:11-41`, `node/internal/meter/delta.go:270-281,292-340,395-424,492-495`, `node/internal/usagestore/payload.go:109-127` | After a sample at the exact UTC month boundary `2026-11-01T00:00:00Z` (which emits a `complete` event with `PeriodId = "2026-10"` via `usagestore.PeriodID(head.LastSample)` and sets `head.LastSample = 2026-11-01T00:00:00Z`), an unchanged `CheckpointForRestart` at `2026-11-01T00:00:00Z` or rolled back across the boundary into October (`2026-10-31T23:59:55Z` or `23:59:50Z`) preserves `LastSample = 2026-11-01T00:00:00Z` instead of moving `LastSample` back into October. Consequently, the next forward sample at `2026-11-01T00:00:10Z` has `IntervalStart = 2026-11-01T00:00:00Z`, `PeriodId = "2026-11"`, and `Quality = complete` (neither `"2026-10"` nor `boundary_estimated`). A forward sample that misses the boundary by `> interval` (`now.Sub(boundary) > interval`, `period.go:37-40`) keeps `end = now` and is marked `boundary_estimated` (with `PeriodId` set from `head.LastSample`'s UTC month by `usagestore.PeriodID` at `delta.go:492` for central projection), and `gap` takes precedence over `boundary_estimated` (`delta.go:323-334,420-424`). | Proven by `TestR33_Q7_PeriodsAndMonthBoundary`. |

---

## Findings by Severity

- **High:** 0
- **Medium:** 0
- **Low:** 0

No product defect was found in `b094ffac72f2f4f247a3e2b592a6659a84f1d51a` (`29cd267162fa614c444122865b62f405e2f58802`).

### Pre-Change vs. Post-Change Behavioral Verification (`9e52ea20` vs. `29cd2671`)

Against the pre-change parent `9e52ea20675daae89710bc4630906e1c70b2e650`:
- When `head.OpenEpoch != uuid.Nil`, `BootID` was unchanged, and `DetectResets` was empty, whenever `CheckpointForRestart` ran at `now == head.LastSample` (for example, immediately after a periodic `Sample` completed in `Barrier`, or within the same microsecond) or during a bounded wall-clock rollback `0 < head.LastSample.Sub(now) <= interval` with unchanged counter readings, `planSample` evaluated `if final && sameReadings(head.Counters, cur)` with `final == false` and returned `plan{hold: true}` (`ErrHold`).
- In `node/agent/`, that `ErrHold` caused `preserveServingOnStop` (`restart.go:190`) to return `false` and fall back to `shutdownUsage` baseline handoff on clean stop, and caused `settleExpiredBindings` (`deadlines.go:90`) to fail closed (`e.usagePersist = true; e.withdraw(ctx)`) when removing an expired sibling binding.
- At `29cd267162fa614c444122865b62f405e2f58802` (`b094ffac`), `planSampleWithBarrier` accepts `(final || restart) && sameReadings(head.Counters, cur)` for `0 <= head.LastSample.Sub(now) <= interval`, committing a durable unchanged checkpoint (`OpenEpoch = head.OpenEpoch, Counters = head.Counters, SampledAt = head.LastSample, CloseEpoch = false`) while still returning `plan{hold: true}` (`ErrHold`) when `sameReadings(head.Counters, cur)` is false (after the earlier open-epoch, boot-change, and reset/disappearance checks).

---

## Routing

- **Required routing:** `none` (0 findings).
- **Optional non-blocking notes for `agent-core`:**
  1. `node/internal/meter/delta.go:23-24` and `docs/node/usage-meter.md:84` describe the bounded rollback hold window as "less than one sample interval", whereas `delta.go:265` implements `head.LastSample.Sub(now) <= interval` (inclusive of exact equality). Aligning the comment/doc wording with `<= interval` would remove any reader ambiguity.
  2. In `node/internal/meter/meter.go:367-384` (`pausedSample`), acquiring `m.sampling = true` under the same `m.mu` lock that sets `m.inFinal = true` after `Barrier(ctx)` (matching `PrepareSampling` at `meter.go:394-404`) would eliminate the unhooked lock-handoff window between `pausedSample` and `m.sample`.

---

## What Held

### Test Implementation Scope and Helpers Used

The 10 parent `TestR33_*` tests live in three same-package test files:
- `node/internal/usagestore/review_r33_test.go` (`package usagestore`): uses same-package test helpers (`openEpoch`, `counterAt`, `testNode`, `testInc`, `testDev`, `testMem`) to test `Store.Commit`, `Store.ApplyACK`, `Store.Head`, `Store.Event`, `ReadSnapshot`, and `Store.OutboxStatus`.
- `node/internal/meter/review_r33_test.go` (`package meter`): uses same-package test helpers and private fields (`fakeSrc`, `clock`, `observe`, `reading`, `start`, `m.mu`, `m.lastMono`, `m.paused`, `m.inFinal`) plus direct SQLite connections (`sql.Open("sqlite", ...)`) to clear `meta.last_sample` in Q1 subtest 8 and to install/drop the `r33_fail_counters` `BEFORE INSERT ON counters` trigger in Q5.
- `node/agent/review_r33_test.go` (`package agent`): uses same-package test rigs, helpers, and private methods/fields (`newRig`, `f42Siblings`, `serveBoundGeneration`, `r20DynamicSource`, `r20Successor`, `r21ReadLedger`, `r.eng.enforce`, `r.eng.onlyExpiredBindingsRemoved`, `r.eng.meter`, `r.eng.usage`, `r.eng.serving`, `r.eng.retainedPending`).

### Parent Test Summary on Review Base vs. Pre-Change Parent

- **On immutable review base `29cd267162fa614c444122865b62f405e2f58802` (Step 440):**
  - **Packages:** 3 passed, 0 failed (`node/internal/usagestore` `0.038s`, `node/internal/meter` `0.878s`, `node/agent` `1.522s`).
  - **Parent `TestR33_*` tests:** 10 passed, 0 failed:
    1. `TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability` (`node/internal/usagestore/review_r33_test.go`, `0.04s`)
    2. `TestR33_Q1_ClockDomainAndFidelity` (`node/internal/meter/review_r33_test.go`, `0.43s`)
    3. `TestR33_Q2_DurableCheckpointAndExactOnce` (`node/internal/meter/review_r33_test.go`, `0.04s`)
    4. `TestR33_Q3_ChangedReadingsAndAttribution` (`node/internal/meter/review_r33_test.go`, `0.16s`)
    5. `TestR33_Q4_PrecedenceAndClosedEpochs` (`node/internal/meter/review_r33_test.go`, `0.09s`)
    6. `TestR33_Q5_ErrorsHealthAndBarriers` (`node/internal/meter/review_r33_test.go`, `0.03s`)
    7. `TestR33_Q6_ConcurrentCallersAndBarrierSemantics` (`node/internal/meter/review_r33_test.go`, `0.07s`)
    8. `TestR33_Q7_PeriodsAndMonthBoundary` (`node/internal/meter/review_r33_test.go`, `0.05s`)
    9. `TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback` (`node/agent/review_r33_test.go`, `0.84s`)
    10. `TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback` (`node/agent/review_r33_test.go`, `0.68s`)
- **On pre-change parent `9e52ea20675daae89710bc4630906e1c70b2e650` (Step 444):**
  - **Packages:** 1 passed (`node/internal/usagestore` `0.028s`), 2 failed (`node/internal/meter` `0.602s`, `node/agent` `1.313s`).
  - **Parent `TestR33_*` tests:** 2 passed (regression coverage), 8 failed (on the new `CheckpointForRestart` equal/rollback invariants):
    - **Passed (2/10):** `TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability` (`0.03s`, testing `usagestore.Store.Commit` directly) and `TestR33_Q4_PrecedenceAndClosedEpochs` (`0.06s`, testing closed-epoch, `BootID`-change, and counter-reset precedence prior to the rollback branch).
    - **Failed (8/10):** `TestR33_Q1_ClockDomainAndFidelity` (`0.36s`), `TestR33_Q2_DurableCheckpointAndExactOnce` (`0.02s`), `TestR33_Q3_ChangedReadingsAndAttribution` (`0.10s`), `TestR33_Q5_ErrorsHealthAndBarriers` (`0.01s`), `TestR33_Q6_ConcurrentCallersAndBarrierSemantics` (`0.03s`), `TestR33_Q7_PeriodsAndMonthBoundary` (`0.03s`), `TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback` (`0.67s`), and `TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback` (`0.64s`).

*(During Correction 1, only `.agent/REVIEW.md`, `.agent/REPORT.md`, `.agent/last.md`, and comments in `node/internal/meter/review_r33_test.go` and `node/agent/review_r33_test.go` were updated; no executable test code changed and no tests were rerun.)*

### Authentic Proof Execution Output on Immutable Review Base (`29cd267162fa614c444122865b62f405e2f58802`, Step 440)

```text
$ CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS= go test -v -count=1 -run '^TestR33_' ./node/internal/usagestore ./node/internal/meter ./node/agent
=== RUN   TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability
--- PASS: TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability (0.04s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	0.038s
=== RUN   TestR33_Q1_ClockDomainAndFidelity
=== RUN   TestR33_Q1_ClockDomainAndFidelity/equal_timestamp
=== RUN   TestR33_Q1_ClockDomainAndFidelity/bounded_rollback_5s
=== RUN   TestR33_Q1_ClockDomainAndFidelity/exact_interval_boundary_10s
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/sample
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/restart
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/final
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/sample
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/restart
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/final
=== RUN   TestR33_Q1_ClockDomainAndFidelity/forward_time_unchanged_readings
=== RUN   TestR33_Q1_ClockDomainAndFidelity/sub_microsecond_same_microsecond_as_last_sample
=== RUN   TestR33_Q1_ClockDomainAndFidelity/open_epoch_with_zero_last_sample
--- PASS: TestR33_Q1_ClockDomainAndFidelity (0.43s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/equal_timestamp (0.06s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/bounded_rollback_5s (0.06s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/exact_interval_boundary_10s (0.06s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns (0.07s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/sample (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/restart (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/final (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us (0.06s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/sample (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/restart (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/final (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/forward_time_unchanged_readings (0.04s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/sub_microsecond_same_microsecond_as_last_sample (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/open_epoch_with_zero_last_sample (0.05s)
=== RUN   TestR33_Q2_DurableCheckpointAndExactOnce
--- PASS: TestR33_Q2_DurableCheckpointAndExactOnce (0.04s)
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/bytes_changed_above_2pow53_returns_errhold_and_preserves_delta
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/packets_changed_bytes_unchanged_returns_errhold
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/counter_added_zero_or_nonzero_returns_errhold
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/counter_removed_or_renamed_triggers_reset_switch
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/unchanged_counters_with_changed_missing_or_conflicting_bindings
--- PASS: TestR33_Q3_ChangedReadingsAndAttribution (0.16s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/bytes_changed_above_2pow53_returns_errhold_and_preserves_delta (0.02s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/packets_changed_bytes_unchanged_returns_errhold (0.02s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/counter_added_zero_or_nonzero_returns_errhold (0.04s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/counter_removed_or_renamed_triggers_reset_switch (0.02s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/unchanged_counters_with_changed_missing_or_conflicting_bindings (0.07s)
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/no_open_epoch_precedes_rollback_branch
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/boot_id_change_with_equal_counters_at_rollback_switches_epoch
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/counter_reset_at_rollback_preserves_other_device_delta_and_final_sample_closes
--- PASS: TestR33_Q4_PrecedenceAndClosedEpochs (0.09s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/no_open_epoch_precedes_rollback_branch (0.02s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/boot_id_change_with_equal_counters_at_rollback_switches_epoch (0.02s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/counter_reset_at_rollback_preserves_other_device_delta_and_final_sample_closes (0.04s)
=== RUN   TestR33_Q5_ErrorsHealthAndBarriers
--- PASS: TestR33_Q5_ErrorsHealthAndBarriers (0.03s)
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics/checkpoint_waits_for_in_flight_sample_then_succeeds_at_same_timestamp
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics/sample_called_while_checkpoint_in_flight_returns_errbusy_and_does_not_steal_restart
--- PASS: TestR33_Q6_ConcurrentCallersAndBarrierSemantics (0.07s)
    --- PASS: TestR33_Q6_ConcurrentCallersAndBarrierSemantics/checkpoint_waits_for_in_flight_sample_then_succeeds_at_same_timestamp (0.05s)
    --- PASS: TestR33_Q6_ConcurrentCallersAndBarrierSemantics/sample_called_while_checkpoint_in_flight_returns_errbusy_and_does_not_steal_restart (0.02s)
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary/unchanged_checkpoint_at_or_rolled_back_into_prior_month_preserves_november_start
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary/missed_boundary_marks_boundary_estimated_and_gap_takes_precedence
--- PASS: TestR33_Q7_PeriodsAndMonthBoundary (0.05s)
    --- PASS: TestR33_Q7_PeriodsAndMonthBoundary/unchanged_checkpoint_at_or_rolled_back_into_prior_month_preserves_november_start (0.03s)
    --- PASS: TestR33_Q7_PeriodsAndMonthBoundary/missed_boundary_marks_boundary_estimated_and_gap_takes_precedence (0.02s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.878s
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_0s
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_1s
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_2s
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/changed_counters_at_rollback_refuses_preservation_and_withdraws
--- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback (0.84s)
    --- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_0s (0.22s)
    --- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_1s (0.22s)
    --- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_2s (0.21s)
    --- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/changed_counters_at_rollback_refuses_preservation_and_withdraws (0.19s)
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_0s
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_1s
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/changed_counters_at_rollback_withdraws_and_keeps_retained_pending
--- PASS: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback (0.68s)
    --- PASS: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_0s (0.24s)
    --- PASS: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_1s (0.23s)
    --- PASS: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/changed_counters_at_rollback_withdraws_and_keeps_retained_pending (0.20s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	1.522s
```

### Authentic Pre-Change Comparison Output (`9e52ea20675daae89710bc4630906e1c70b2e650`, Step 444)

```text
=== RUN   TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability
--- PASS: TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability (0.03s)
PASS
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	0.028s
=== RUN   TestR33_Q1_ClockDomainAndFidelity
=== RUN   TestR33_Q1_ClockDomainAndFidelity/equal_timestamp
    review_r33_test.go:94: CheckpointForRestart at rollback=0s: got meter: wall clock jitter, sample held, want nil
=== RUN   TestR33_Q1_ClockDomainAndFidelity/bounded_rollback_5s
    review_r33_test.go:94: CheckpointForRestart at rollback=5s: got meter: wall clock jitter, sample held, want nil
=== RUN   TestR33_Q1_ClockDomainAndFidelity/exact_interval_boundary_10s
    review_r33_test.go:94: CheckpointForRestart at rollback=10s: got meter: wall clock jitter, sample held, want nil
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/sample
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/restart
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/final
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/sample
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/restart
=== RUN   TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/final
=== RUN   TestR33_Q1_ClockDomainAndFidelity/forward_time_unchanged_readings
=== RUN   TestR33_Q1_ClockDomainAndFidelity/sub_microsecond_same_microsecond_as_last_sample
    review_r33_test.go:245: second sub-microsecond CheckpointForRestart: got meter: wall clock jitter, sample held, want nil
=== RUN   TestR33_Q1_ClockDomainAndFidelity/open_epoch_with_zero_last_sample
--- FAIL: TestR33_Q1_ClockDomainAndFidelity (0.36s)
    --- FAIL: TestR33_Q1_ClockDomainAndFidelity/equal_timestamp (0.03s)
    --- FAIL: TestR33_Q1_ClockDomainAndFidelity/bounded_rollback_5s (0.04s)
    --- FAIL: TestR33_Q1_ClockDomainAndFidelity/exact_interval_boundary_10s (0.04s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns (0.07s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/sample (0.03s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/restart (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1ns/final (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us (0.07s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/sample (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/restart (0.02s)
        --- PASS: TestR33_Q1_ClockDomainAndFidelity/over_interval_by_1us/final (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/forward_time_unchanged_readings (0.04s)
    --- FAIL: TestR33_Q1_ClockDomainAndFidelity/sub_microsecond_same_microsecond_as_last_sample (0.02s)
    --- PASS: TestR33_Q1_ClockDomainAndFidelity/open_epoch_with_zero_last_sample (0.05s)
=== RUN   TestR33_Q2_DurableCheckpointAndExactOnce
    review_r33_test.go:386: CheckpointForRestart at equal timestamp: meter: wall clock jitter, sample held
--- FAIL: TestR33_Q2_DurableCheckpointAndExactOnce (0.02s)
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/bytes_changed_above_2pow53_returns_errhold_and_preserves_delta
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/packets_changed_bytes_unchanged_returns_errhold
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/counter_added_zero_or_nonzero_returns_errhold
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/counter_removed_or_renamed_triggers_reset_switch
=== RUN   TestR33_Q3_ChangedReadingsAndAttribution/unchanged_counters_with_changed_missing_or_conflicting_bindings
    review_r33_test.go:638: CheckpointForRestart with changed binding and equal counters: meter: wall clock jitter, sample held
--- FAIL: TestR33_Q3_ChangedReadingsAndAttribution (0.10s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/bytes_changed_above_2pow53_returns_errhold_and_preserves_delta (0.02s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/packets_changed_bytes_unchanged_returns_errhold (0.02s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/counter_added_zero_or_nonzero_returns_errhold (0.03s)
    --- PASS: TestR33_Q3_ChangedReadingsAndAttribution/counter_removed_or_renamed_triggers_reset_switch (0.02s)
    --- FAIL: TestR33_Q3_ChangedReadingsAndAttribution/unchanged_counters_with_changed_missing_or_conflicting_bindings (0.01s)
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/no_open_epoch_precedes_rollback_branch
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/boot_id_change_with_equal_counters_at_rollback_switches_epoch
=== RUN   TestR33_Q4_PrecedenceAndClosedEpochs/counter_reset_at_rollback_preserves_other_device_delta_and_final_sample_closes
--- PASS: TestR33_Q4_PrecedenceAndClosedEpochs (0.06s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/no_open_epoch_precedes_rollback_branch (0.02s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/boot_id_change_with_equal_counters_at_rollback_switches_epoch (0.01s)
    --- PASS: TestR33_Q4_PrecedenceAndClosedEpochs/counter_reset_at_rollback_preserves_other_device_delta_and_final_sample_closes (0.03s)
=== RUN   TestR33_Q5_ErrorsHealthAndBarriers
    review_r33_test.go:939: CheckpointForRestart recovery: meter: wall clock jitter, sample held
--- FAIL: TestR33_Q5_ErrorsHealthAndBarriers (0.01s)
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics/checkpoint_waits_for_in_flight_sample_then_succeeds_at_same_timestamp
    review_r33_test.go:1103: isFinal=false after in-flight Sample at same timestamp: got meter: wall clock jitter, sample held, want nil
=== RUN   TestR33_Q6_ConcurrentCallersAndBarrierSemantics/sample_called_while_checkpoint_in_flight_returns_errbusy_and_does_not_steal_restart
    review_r33_test.go:1167: CheckpointForRestart completion: meter: wall clock jitter, sample held
--- FAIL: TestR33_Q6_ConcurrentCallersAndBarrierSemantics (0.03s)
    --- FAIL: TestR33_Q6_ConcurrentCallersAndBarrierSemantics/checkpoint_waits_for_in_flight_sample_then_succeeds_at_same_timestamp (0.01s)
    --- FAIL: TestR33_Q6_ConcurrentCallersAndBarrierSemantics/sample_called_while_checkpoint_in_flight_returns_errbusy_and_does_not_steal_restart (0.01s)
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary/unchanged_checkpoint_at_or_rolled_back_into_prior_month_preserves_november_start
    review_r33_test.go:1233: CheckpointForRestart at 2026-11-01 00:00:00 +0000 UTC: meter: wall clock jitter, sample held
=== RUN   TestR33_Q7_PeriodsAndMonthBoundary/missed_boundary_marks_boundary_estimated_and_gap_takes_precedence
--- FAIL: TestR33_Q7_PeriodsAndMonthBoundary (0.03s)
    --- FAIL: TestR33_Q7_PeriodsAndMonthBoundary/unchanged_checkpoint_at_or_rolled_back_into_prior_month_preserves_november_start (0.01s)
    --- PASS: TestR33_Q7_PeriodsAndMonthBoundary/missed_boundary_marks_boundary_estimated_and_gap_takes_precedence (0.01s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/node/internal/meter	0.602s
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_0s
    review_r33_test.go:87: Close at rollback=0s with unchanged counters withdrew to baseline (4 -> 5)
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_1s
    review_r33_test.go:87: Close at rollback=1s with unchanged counters withdrew to baseline (4 -> 5)
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_2s
    review_r33_test.go:87: Close at rollback=2s with unchanged counters withdrew to baseline (4 -> 5)
=== RUN   TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/changed_counters_at_rollback_refuses_preservation_and_withdraws
--- FAIL: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback (0.67s)
    --- FAIL: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_0s (0.13s)
    --- FAIL: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_1s (0.18s)
    --- FAIL: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/unchanged_counters_2s (0.18s)
    --- PASS: TestR33_AgentPreserveServingOnStopAtSameTimestampAndRollback/changed_counters_at_rollback_refuses_preservation_and_withdraws (0.18s)
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_0s
    review_r33_test.go:265: survivor transition failed at rollback=0s: serving=false bindings=0 baselines=7->9 applies=3->3
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_1s
    review_r33_test.go:265: survivor transition failed at rollback=1s: serving=false bindings=0 baselines=7->9 applies=3->3
=== RUN   TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/changed_counters_at_rollback_withdraws_and_keeps_retained_pending
--- FAIL: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback (0.64s)
    --- FAIL: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_0s (0.22s)
    --- FAIL: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/unchanged_counters_1s (0.21s)
    --- PASS: TestR33_AgentSettleExpiredBindingsAtSameTimestampAndRollback/changed_counters_at_rollback_withdraws_and_keeps_retained_pending (0.21s)
FAIL
FAIL	github.com/TED-H-com/Freedom-VPN/node/agent	1.313s
```

### Package Unit Suites and Vet Output (Step 448)

```text
$ CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS= go test -count=1 ./node/internal/usagestore ./node/internal/meter
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/usagestore	1.008s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/meter	1.263s

$ CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS= go vet ./node/internal/usagestore ./node/internal/meter ./node/agent
(exit 0)
```

---

## Proof Development and Execution Attempt Ledger

All nine `go test` invocations during the session are recorded below. Intermediate uncommitted test-file edits prior to `7bac86538bd9f0108a3bb441205b9ba979eae134` were not separately committed or retained as git snapshots; all failures in Steps 212, 230, 341, 370, 378, and 424 were test-authoring compile, fixture, or expectation errors, not product defects.

| Session Step | Target Tree | Exit | Package Outcomes & Durations | Cause of Failure & Test Correction Applied |
|---|---|:---:|---|---|
| **Step 212** | `29cd2671` | `1` | `usagestore`: `[build failed]`<br>`meter`: `[build failed]`<br>`agent`: `FAIL 1.195s` (both parents failed: `0.61s`, `0.58s`) | **Compile & fixture errors:** (1) `usagestore` and `meter` test files referenced nonexistent type `Step` / `usagestore.Step` (hook step names are `string`) and `usagestore.StepRead` / `usagestore.StepPlan` (defined in `package meter`). (2) `agent` tests did not align `r.clock` to a whole microsecond before sampling (causing a `1µs` `eventInterval` shift on `0s` subtests) and jumped wall clock back by `5s` and `10s`, which exceeded `timeanchor.DefaultBackwardSkew = 2s` (`Reasons=[clock_untrusted]`). Fixed by using `string` / `meter.StepRead` / `meter.StepPlan`, adding `alignMicrosecond(r)`, and testing `0s`, `1s`, `2s` rollbacks in `node/agent/review_r33_test.go`. |
| **Step 230** | `29cd2671` | `1` | `usagestore`: `ok 0.039s` (`0.04s`)<br>`meter`: `FAIL 0.386s` (all 7 parents failed)<br>`agent`: `ok 1.408s` (`0.74s`, `0.66s`) | **Test fixture & assertion errors in `meter`:** (1) In Q1 (`0.16s`), Q2 (`0.03s`), Q3 (`0.09s`), Q4 (`0.04s`), and Q7 (`0.02s`), the test helper opened `meter.New` on a fresh store (`!head.HasHistory`) with non-zero initial counters (`anyBytes(cur) == true`), causing `planBaseline` (`delta.go:208-223`) to emit an initial `unknown` gap event (`seq 1`) and advance `LastSample` to `t0 + 1µs`. (2) In Q5 (`0.01s`), the test injected a hook error at `StepRead` (`"sample.read"`) and asserted `ReasonPersistFail`, whereas `m.store.Fault(StepRead)` returns before `m.src.Read` and `m.store.Head` without latching `persistFailed`. (3) In Q6 (`0.03s`), `m.Sample` was called after the `Barrier` goroutine had already set `m.paused = true`, returning `ErrPaused` instead of `ErrBusy`. Fixed by starting `meter.New` fixtures with zero counters at `t0` and moving the `ErrBusy` check before launching `Barrier`. |
| **Step 341** | `29cd2671` | `1` | `usagestore`: `ok 0.178s` (`0.18s`)<br>`meter`: `FAIL 0.788s` (5 parents passed: Q1 `0.41s`, Q2 `0.03s`, Q3 `0.14s`, Q4 `0.08s`, Q6 `0.06s`; 2 failed: Q5 `0.02s`, Q7 `0.04s`)<br>`agent`: `ok 1.235s` (`0.77s`, `0.46s`) | **Test expectation errors in Q5 and Q7:** (1) In Q5, `Options.Hook` errors at `StepCheckpoint` are wrapped in `ErrInjectedCrash` by `usagestore.Store.hookStep` (`store.go:474-485`), and `Meter.persist` (`meter.go:288-294`) calls `NotePersistSuccess()` when `errors.Is(err, ErrInjectedCrash)` and `p.commit.Event == nil`. (2) In Q7, the missed-boundary subtest sampled at `novStart.Add(5 * time.Second)` (`<= 10s` interval), which `stampEnd` (`period.go:37-39`) stamps onto `novStart` as `complete` in `"2026-10"` rather than `boundary_estimated`. |
| **Step 370** | `29cd2671` | `1` | `usagestore`: `ok 0.035s` (`0.03s`)<br>`meter`: `[build failed]`<br>`agent`: `ok 0.997s` (`0.52s`, `0.47s`) | **Compile error in Q5:** While refactoring Q5 to open `s` at `dbPath`, called `New(s, src, Options{...})` instead of `New(Config{...})` (`review_r33_test.go:881`). |
| **Step 378** | `29cd2671` | `1` | `usagestore`: `ok 0.030s` (`0.03s`)<br>`meter`: `[build failed]`<br>`agent`: `ok 1.414s` (`0.74s`, `0.67s`) | **Compile error in Q5:** Passed `Hook` in `meter.Config{...}` literal (`review_r33_test.go:885`); `Hook` belongs on `usagestore.Options`, not `meter.Config`. |
| **Step 424** | `29cd2671` | `1` | `usagestore`: `ok 0.030s` (`0.03s`)<br>`meter`: `FAIL 0.932s` (5 parents passed: Q1 `0.47s`, Q2 `0.04s`, Q3 `0.18s`, Q4 `0.10s`, Q6 `0.07s`; 2 failed: Q5 `0.03s`, Q7 `0.05s`)<br>`agent`: `ok 1.646s` (`0.90s`, `0.74s`) | **Trigger target in Q5 and `PeriodID` month in Q7:** (1) In Q5, the SQLite trigger was placed `BEFORE UPDATE ON meta`, which also caused `m.Health` -> `s.EvaluateGate` (`UPDATE meta SET gate_stopped = ...`, `ack.go:585`) to return a SQLite error. (2) In Q7, `novStart.Add(15 * time.Second)` produced `Quality = boundary_estimated`, with `PeriodId = "2026-10"` (and `"2026-11"` for the December cross-boundary gap) because `attribute()` (`delta.go:492`) calls `usagestore.PeriodID(head.LastSample)`. Fixed by moving Q5's trigger to `BEFORE INSERT ON counters` and updating Q7's expected `PeriodId` values to `"2026-10"` and `"2026-11"`. |
| **Step 440** | `29cd2671` | `0` | `usagestore`: `ok 0.038s`<br>`meter`: `ok 0.878s`<br>`agent`: `ok 1.522s` | **All 10 parent `TestR33_*` proofs passed on the immutable review base.** |
| **Step 444** | `9e52ea20` (temp checkout) | `1` | `usagestore`: `ok 0.028s`<br>`meter`: `FAIL 0.602s`<br>`agent`: `FAIL 1.313s` | **Intended pre-change comparison:** 2 parent tests passed (`TestR33_StoreUnchangedCheckpointPreservesLedgerAndDurability` and `TestR33_Q4_PrecedenceAndClosedEpochs`) and 8 parent tests failed on `CheckpointForRestart` returning `ErrHold` (`meter: wall clock jitter, sample held`). Temporary checkout removed immediately after run. |
| **Step 448** | `29cd2671` | `0` | `usagestore`: `ok 1.008s`<br>`meter`: `ok 1.263s`<br>`go vet` & `gofmt -l`: `exit 0` | **Full package unit suites (`node/internal/usagestore`, `node/internal/meter`), `go vet`, and `gofmt -l` passed.** |

---

## Not Examined

1. **Live kernel / netlab / arm64 hardware execution:** No Docker, privileged netlab container, or host `nft`/`tc` commands were run during this unit-level adversarial review. Real kernel `nft` counter reads and netlab restart adoption remain covered by the `netlab` suite (`AT14`–`AT19`).
2. **Unproven multi-controller concurrency outside `Engine`'s serialization model:** In `pausedSample` (`node/internal/meter/meter.go:367-384`), `m.mu` is released between `Barrier(ctx)` (`line 327`/`348`), setting `m.inFinal = true` (`line 376`), and acquiring `m.mu` in `m.sample` (`line 202`), and again between `m.sample`'s deferred unlock (`line 222`) and `pausedSample`'s deferred unlock (`line 381`). Because `Engine` serializes control operations (`CheckpointForRestart`, `FinalSample`, `PrepareSampling`, `ResumeSampling`) on a single control goroutine and `Meter.Run` checks `m.paused` under `m.mu` before calling `Sample`, and because there is no synchronization hook inside that instruction gap, a race between `pausedSample` and an external concurrent caller that bypasses `Run`'s pause check was not reachable via deterministic hooks and is recorded as an unproven concurrency assumption rather than a finding.
3. **Source-inspected meter paths not re-tested with dedicated `TestR33_*` subcases:** Exclusion of DNS (`nu_` / `nd_`) and diagnostic counters (`parseUsageCounter` / `usageCounters` at `delta.go:51-125`), a forward positive delta when `cur` has a missing (`uuid.Nil`) binding (`delta.go:503`), and readings changing again between an in-flight `Sample`'s read and a waiting `CheckpointForRestart`'s read were verified by source inspection (and existing `meter_test.go` coverage) rather than separate `TestR33_*` subtests.
4. **Other commits on the `agent-core 53` branch and later `agent-core` changes:** Only the meter restart checkpoint diff `9e52ea20675daae89710bc4630906e1c70b2e650..b094ffac72f2f4f247a3e2b592a6659a84f1d51a` (verified byte-identical on `node/internal/meter/{delta,meter}.go` at `29cd267162fa614c444122865b62f405e2f58802`) was audited in this review. Other files and commits in `9e52ea20..29cd2671` and any subsequent `agent-core` commits after `29cd267162fa614c444122865b62f405e2f58802` remain outside this review, with root responsible for final replay.
