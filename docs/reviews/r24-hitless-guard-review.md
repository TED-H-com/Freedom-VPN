# Review `r24-hitless-guard-review`: Adversarial Review of Follow-Up 31 Hitless Deadline-Guard Hand-Over Design (`52a7e59b`)

- **Reviewer:** `agy/gemini-4-argon-high`
- **Target commit:** `52a7e59be947422e5e7c39a52d3c227c84050718` (`main` at `#418` `cc0ed4ae` plus `docs/poc/G2-followup31-design/`)
- **Scope:** `docs/poc/G2-followup31-design/*`, `docs/poc/G2-followup30-evidence/*`, `node/enforce/*`, `node/agent/*`, `node/watchdog/*`, `tests/netlab/*`, `internal/leases/leases.go`, `docs/handover/calibration.md`, `docs/handover/README.md`, `docs/spec/PROJECT_SPEC.zh-TW.md` (§8.1–§8.2, §10, §11, §16.3, §19 AT10/AT13/AT29).

---

## 1. Findings by severity

### Critical

None (0).

---

### High

#### `H-1` — `Engine.Step` re-executes full `Plane.Apply` (and the ~28.5 ms `fv_deadline_guard` blackout) on every 10-second `304 Not Modified` poll/heartbeat tick in healthy steady state

- **Where:** `node/agent/engine.go:619-632`, `node/agent/engine.go:928-1088`; also missed in `docs/poc/G2-followup31-design/current-guarantees.md` and `docs/poc/G2-followup31-design/candidates.md`.
- **In design or already on `main`:** Already on `main` (and unaddressed by the follow-up 31 design).
- **Scenario:**
  In `Engine.Step` (`node/agent/engine.go:619-632`), the unchanged-state retention check is written as:
  ```go
  retain := ioCtx.Err() != nil && e.serving && e.decision.Serve && e.applyError == nil &&
      previousDesired == e.desired && previousLease == e.lease && e.appliedModel != nil &&
      e.appliedGen == e.decision.Model.Generation && e.policyExpires == e.decision.PolicyExpires &&
      len(e.pendingRevoke) == 0 && !e.memberPathPending && !e.quarantinePending
  ```
  Because `retain` is gated on `ioCtx.Err() != nil` (the `D-100` control-plane timeout freeze branch), every normal 10-second `Step` (`PollSeconds = 10`, `HeartbeatSeconds = 10` in `node/agent/config.go:133-138`) where control I/O succeeds (`ioCtx.Err() == nil`) evaluates `retain = false` even when `syncDesired` received HTTP `304 Not Modified` (`previousDesired == e.desired`) and `syncLease` did not renew (`previousLease == e.lease`). `Step` then enters `else` and calls `e.enforce(ctx, e.decision)` (`node/agent/engine.go:928-1088`), which has no unchanged-generation skip and unconditionally calls `e.plane.Apply(applyCtx, model, prev)`.
  As a result:
  1. On `main` today, a healthy serving node with zero policy changes and zero lease renewals runs `tc -batch` and `guardedNFT` (flushing `@allowed_v4`, replacing all chains, and installing `fv_deadline_guard`) **every 10 seconds** (**6 times/minute, 360 times/hour, 8,640 times/day**), subjecting all active member flows to the ~28.5 ms median (up to 94.7 ms) guard blackout every 10 seconds.
  2. Under C1 (`docs/poc/G2-followup31-design/candidates.md` §§"C1.2", "C1.4"), every 10-second no-op tick would allocate a new bank `fv_allow_N`, rebuild all chains, and execute two `nft -f` transactions unnecessarily.
- **Proof (unit test & failure output):**
  Test `TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick` in `node/agent/review_r24_test.go`:
  ```text
  === RUN   TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick
      review_r24_test.go:52: TODO(review r24): H-1: 3 steady-state 10s Steps with HTTP 304 and an unchanged lease executed 3 extra Plane.Apply calls (applies 2 -> 5, want 2)
  --- FAIL: TestR24_H1_SteadyStateNoOpStepReappliesEveryPollTick (0.19s)
  ```
- **Fix (Owner: `agent-core`):**
  In `node/agent/engine.go:619-632`, when `ioCtx.Err() == nil` and the node is already serving the exact same verified desired generation, lease (`previousDesired == e.desired && previousLease == e.lease`), local config/probe state, and has no pending revocations, member-path repairs, or quarantine flags, verify the live datapath (`e.verifyServingModel` + `e.checkMemberPath`) under `nodelock` without re-running `e.plane.Apply`.

---

#### `H-2` — C1's QoS eligibility rule disqualifies every member revocation and addition (including `AT13`), while `Apply` deletes orphan member HTB classes in `TCBatch` before `NftBatch` removes their `meta priority` rules (routing removed-member traffic into `1:fffe` `HTBDefaultMinor` host reserve)

- **Where:** `docs/poc/G2-followup31-design/candidates.md` ("Eligibility (per member device)" lines 38–40 and "C1.1 Shape" lines 78–80), `docs/poc/G2-followup31-design/proof-obligations.md` ("Netlab proof obligations" item 2, lines 120–125), `node/agent/decide.go:766-812`, `node/enforce/apply.go:151, 183`, `node/enforce/render_tc.go:59-79, 96`.
- **In design or already on `main`:** Both: the eligibility contradiction is in the follow-up 31 design (`candidates.md` vs. `proof-obligations.md` and `decide.go:766-812`); the `TCBatch`-before-`NftBatch` orphan class deletion into `1:fffe` is already on `main` (`node/enforce/apply.go:151`, `render_tc.go:62-67, 96`) and carried into C1.1 (`candidates.md:76-86`).
- **Scenario:**
  1. **Design contradiction on `AT13` / member churn:** `candidates.md:38-40` states that any member QoS rate/class change selects the unconditional guard for the whole `Apply`, and `candidates.md:78-80` ("C1.1 Shape") requires: *"In a C1-eligible transition no roots/leaves/classes/rates change; any emitted idempotent class replacements must be byte-equivalent in meaning. Do not slip a changed ceiling or class deletion into this case."* Meanwhile, `proof-obligations.md:120-125` requires proving C1 on `AT13` (*"two independent members plus an unchanged sibling; revoke `A2` while `A1` continues... Unchanged `A1` has zero guard-attributed drops"*).
     However, `buildQoS` in `node/agent/decide.go:767-809` computes `fairUp = (upParent - host) / uint64(len(admitted))` and `fairDown = (downParent - host) / uint64(len(admitted))` across the admitted member set, and `classRate` (`decide.go:829-832`) sets each member's HTB guaranteed `RateBPS = min(fair, ceil)`. Whenever any independent member is revoked or added, `len(admitted)` changes, mutating `MemberQoS.Up.RateBPS` and `Down.RateBPS` for **every remaining member**, and `renderTC` (`node/enforce/render_tc.go:62-67`) emits `qdisc del` + `class del` for the removed member's class. Under C1's own eligibility rule, **every member revocation or addition (including `AT13`) is disqualified from C1 and falls back to the unconditional guard**.
  2. **Pre-`nft` orphan class deletion routes removed member traffic into the host control-plane reserve (`1:fffe`):** In `enforce.Apply` (`node/enforce/apply.go:151`), `ex.TCBatch` executes **before** `ex.NftBatch` (`apply.go:183`). When a member is removed, `renderTC` (`node/enforce/render_tc.go:62-67`) deletes the removed member's HTB class `1:<minor>` and leaf `fq_codel` qdisc during `TCBatch` while the old `forward` chain in the kernel is still matching that member's packets with `meta priority set 0x1<minor> accept`. Because the HTB root qdisc is configured with `default fffe` (`node/enforce/render_tc.go:96`, `HTBDefaultMinor = 0xfffe`, the host control-plane reserve class under §8.3), packets from the about-to-be-revoked member classify into a non-existent HTB class and fall back into `1:fffe` (the host control-plane reserve!) for the ~25–50 ms until `NftBatch` commits.
- **Proof (unit tests & failure output):**
  1. `TestR24_H2_MemberRevocationMutatesSiblingQoSRateInDecide` in `node/agent/review_r24_test.go` (proving `Decide` doubles remaining member `A`'s `RateBPS` when independent member `B` is removed).
  2. `TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch` in `node/enforce/review_r24_test.go`:
  ```text
  === RUN   TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch
      review_r24_test.go:178: TODO(review r24): H-2: orphan member HTB class 1:30 was deleted in TCBatch before NftBatch replaced forward (order=[tc_delete_member_class:1:30 nft_guarded_install]), routing packets with meta priority 0x10030 into HTB default class 1:fffe (HTBDefaultMinor)
  --- FAIL: TestR24_H2_OrphanMemberTCClassDeletedBeforeNftBatch (0.00s)
  ```
- **Fix (Owner: `isolation` & `agent-core`):**
  1. In C1 eligibility (`candidates.md`), allow a member to remain eligible when its `ClassID(minor)` is unchanged and its effective ceiling (`CeilBPS`) did not tighten, even if `RateBPS` (guaranteed fair share) changed due to another member joining/leaving.
  2. In `node/enforce/apply.go` and `render_tc.go`, never delete orphan member HTB classes before `NftBatch` removes their `meta priority set` rules from `forward`: run `class replace` / new-class additions in `TCBatch` before `NftBatch`, and delete unreferenced orphan HTB classes/qdiscs only *after* `NftBatch` commits (before `verifyServing`).

---

#### `H-3` — C1 eligibility cannot be safely computed from what `Engine` and `Plane.Apply` hold under the lock today (`DeadlineMono` collapsed per-member in `decide.go:682-695`, `e.appliedModel` retained across baseline/reboot, and value-copy `NowMono` drift)

- **Where:** `docs/poc/G2-followup31-design/candidates.md` ("Eligibility (per member device)" lines 27–45, "C1.4 Release" line 140), `docs/poc/G2-followup31-design/proof-obligations.md` (Obligation 3, lines 44–53), `node/agent/decide.go:682-695`, `node/agent/engine.go:649-657, 982-994, 1028, 1087, 1126, 1289-1294`, `node/agent/plane.go:16`, `node/enforce/apply.go:68, 169-171`.
- **In design or already on `main`:** Both on `main` (`node/agent/decide.go:682-695`, `node/agent/engine.go`, `node/enforce/apply.go`) and in the follow-up 31 design (`candidates.md`, `proof-obligations.md` Obligation 3).
- **Scenario:**
  Three concrete state mismatches in the current codebase break per-device eligibility calculation under the lock:
  1. **Per-member `DeadlineMono` collapse in `decide.go:682-695` truncates sibling device timeouts and corrupts `D_new >= D_old` across multi-device reloads (`AT10`):**
     `enforce.Model.Leases` has one `MemberLease` per `MemberID`, not per `DeviceID`. In `node/agent/decide.go:683-695`, when a member has two devices with different `ValidUntil` timestamps (e.g., `dev1` with `ValidUntil = T+1h` visited first, `dev2` with `ValidUntil = T+5m` visited second), `decide.go:686-688` overwrites `leases[i].DeadlineMono` with `min(deadline_1, deadline_2) = mono + 5m` while leaving `EffectiveServiceEnd` and `NodeLeaseEnd` at `T+1h`. Because `memberTimeout` (`node/enforce/deadline.go:39-40`) computes `remaining = *l.DeadlineMono - m.NowMono` whenever `l.DeadlineMono != nil` (ignoring `b.ValidUntil`), `dev1` is given a 5-minute kernel timeout instead of 1 hour! Furthermore, when `dev2` is removed in one generation and re-added in the next (as `AT10` does with `devA2` every 10th reload), `leases[memberA].DeadlineMono` fluctuates between `dev1`'s and `dev2`'s deadlines, causing unchanged `dev1` to fail `D_new >= D_old` whenever `dev2` has an earlier `ValidUntil`.
  2. **`e.appliedModel` (`retained_applied`) survives baseline withdrawal, rollback, and OS reboot:**
     `Plane.Apply(ctx, m, prev)` (`node/agent/plane.go:16`) does not receive the prior verified model at all. In `Engine`, `e.appliedModel` is intentionally kept non-nil across `withdraw`, `baseline`, `handoff`, and `rejectServing` (`node/agent/engine.go:982-994, 1117-1121, 1289-1294`) for foreign-tc containment auditing (`serving.go:185-207`), and `loadRuntime` (`engine.go:649-657`) loads `retained_applied` from `agent.db` unconditionally on `Boot` without a `BootID` check (so after an OS reboot, `e.appliedModel.Leases[i].DeadlineMono` holds `CLOCK_MONOTONIC` values from the previous boot epoch!).
  3. **Value-copy `NowMono` drift between `Engine.appliedModel` and `Apply`:**
     `enforce.Apply(ctx, ex, m, prev)` (`node/enforce/apply.go:68`) takes `m Model` by value and updates `m.NowMono = timing.RenderStart` (`apply.go:170`) only on its local copy. `Engine.enforce` (`node/agent/engine.go:1126`) stores `e.appliedModel = &model` with the pre-lock `Decide` timestamp rather than `res.Deadline.RenderStart`.
- **Proof (unit test & failure output):**
  Test `TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse` in `node/agent/review_r24_test.go`:
  ```text
  === RUN   TestR24_H3_AppliedModelRetainedAcrossWithdrawAndMultiDeviceDeadlineCollapse/multi_device_same_member_collapses_deadline_mono_and_leaves_service_end_inconsistent
      review_r24_test.go:164: TODO(review r24): H-3: Device 1 (ValidUntil=1h) received kernel element timeout 5m0s (same as Device 2's 5m0s) because decide.go:682-695 collapses DeadlineMono per MemberID while leaving EffectiveServiceEnd=1h0m0s
  ```
- **Fix (Owner: `agent-core` & `isolation`):**
  1. In `node/agent/decide.go` and `node/enforce/model.go`, store per-device monotonic deadlines (`Binding.DeadlineMono *time.Duration`) so each device's `ValidUntil` caps only that device's kernel timeout and `MemberLease.DeadlineMono` holds the member-level lease deadline (`min(leaseEnd, ml.LeaseEnd, ml.EffectiveServiceEnd)`).
  2. Pass an explicit `*VerifiedServing` struct (carrying `BootID`, `ActiveBank`, post-render `RenderStart`, `Model`, and `Assignments`) into `Plane.Apply`, and clear it in `Engine` whenever `e.serving` transitions to `false` or `Boot` does not adopt.

---

#### `H-4` — `installBaselineModel` ignores `loadState` errors and falls back to stale pre-install `State{}` (omitting `delete chain forward` and `flush set allowed_v4` on cold-start rollback), and `Apply` passes its expired `ctx` to `installBaseline`

- **Where:** `node/enforce/apply.go:152, 174, 192, 202, 222, 305-315` and `node/enforce/render_nft.go:276-395, 510-536`; contradicts `docs/poc/G2-followup31-design/current-guarantees.md` (G4, G8, G10, lines 61, 87, 97) and affects C1.6 (`candidates.md:158-167`).
- **In design or already on `main`:** Already on `main` (and inherited by C1.6 in the design).
- **Scenario:**
  1. **Stale `State{}` rollback leaves candidate `accept` rules and `@allowed_v4` live:** When `Apply` starts from a cold table (`state = State{}` at `apply.go:87`), `ex.NftBatch(installCtx, guarded)` (`apply.go:183`) creates `forward`, `input`, `output`, `postrouting`, and `allowed_v4`, and `releaseDeadlineGuard` (`apply.go:199`) removes `fv_deadline_guard` so the candidate is live in the kernel (`state` only records then deletes `state.Chains[deadlineGuard]`, leaving `state.hasChain("forward") == false` and `state.hasSet("allowed_v4") == false`). When `verifyServing` (`apply.go:213`) subsequently fails, `Apply` calls `installBaseline(ctx, ex, m.Datapath, state)` (`apply.go:222`). Inside `installBaselineModel` (`apply.go:307`), `if fresh, err := loadState(ctx, ex); err == nil { state = fresh }` silently ignores any `loadState` error and falls back to the stale pre-install `state` (`State{}`). Because `replaceChain` (`render_nft.go:524-529`) and `RenderBaseline` (`render_nft.go:292-294`) only emit `delete chain inet fvpn forward` and `flush set inet fvpn allowed_v4` when `state.hasChain` / `state.hasSet` is true, `RenderBaseline` emits only `add set inet fvpn allowed_v4`, `add chain inet fvpn forward`, and `add rule inet fvpn forward ... drop`. In nftables, `add chain` on an existing chain is a no-op and `add rule` appends to the end of the chain — leaving the unverified candidate's `accept` rules at the top of `forward` and all member IPs in `@allowed_v4` while `NftBatch` exits 0!
  2. **Expired `ctx` aborts rollback after guard release:** At `apply.go:152, 174, 192, 202, 222`, `Apply` passes its own `ctx` into `installBaseline`. If `ctx` expires or is canceled during `verifyServing` (after `releaseDeadlineGuard` has already opened the candidate), `installBaseline(ctx, ...)` immediately fails with `context.DeadlineExceeded` / `context.Canceled` without executing a rollback batch.
- **Proof (unit test & failure output):**
  Test `TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen` in `node/enforce/review_r24_test.go`:
  ```text
  === RUN   TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/cold_start_rollback_with_failed_loadState_omits_delete_chain_and_flush_set
      review_r24_test.go:67: TODO(review r24): H-4: post-release rollback rendered against stale cold State{} omitted `delete chain inet fvpn forward` and `flush set inet fvpn allowed_v4`
  === RUN   TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen/expired_ctx_during_verify_aborts_rollback_after_guard_release
      review_r24_test.go:108: TODO(review r24): H-4: guard was released (batches=2, GuardReleased=true) and verification failed on canceled ctx, but installBaseline reused the canceled ctx and executed no baseline rollback batch
  --- FAIL: TestR24_H4_RollbackWithStaleStateOrExpiredCtxLeavesCandidateOpen (0.00s)
  ```
- **Fix (Owner: `isolation`):**
  1. Make `RenderBaseline` (and `replaceChain` / `ensureSet`) self-contained without relying on `state.hasChain` or `state.hasSet`: in nftables, emitting `add chain inet fvpn <c> { ... }` + `delete chain inet fvpn <c>` + `add chain inet fvpn <c> { ... }` and `add set` + `flush set` works unconditionally in a single atomic transaction whether or not the chain/set previously existed. For dynamic C1 sets (`fv_allow_*`, `fv_eligible_*`, `fv_token_*`), record all bank/token/eligible set names created by the current `Apply` into `state.Sets` before `NftBatch` and require `loadState` to succeed in `ApplyBaselineModel`.
  2. In `Apply`, invoke `installBaseline` with an independent bounded cleanup context (`context.WithoutCancel(ctx)` with a timeout) so an expired `Apply` context never prevents post-install rollback.

---

### Medium

#### `M-1` — C1's bank retirement leaves `fv_allow_A` in the kernel after C1.4 release (creating an unnecessary crash cut before `SetMeta("retained_applied")`), and `candidates.md:204` overstates fencing when a delayed unconditional-fallback install races `Baseline`

- **Where:** `docs/poc/G2-followup31-design/candidates.md` (C1.0 lines 54–60, C1.2 lines 88–112, C1.4 lines 133–154, Failure matrix line 204), `node/agent/engine.go:1087-1141`.
- **In design or already on `main`:** In the follow-up 31 design.
- **Scenario & Proof (counterexample trace):**
  1. **Unnecessary two-bank state after C1.4 release:** In C1.2 (`candidates.md:108-109`), `forward` and `postrouting` are replaced to reference `@fv_allow_N`; the only chain in `inet fvpn` still referencing `@fv_allow_A` is `fv_deadline_guard`. Yet C1.4 (`candidates.md:134-139`) only deletes `fv_token_g`, `fv_deadline_guard`, and `fv_eligible_g`, stating: *"The old bank remains unreferenced until cleanup in the next install or baseline."*
     - **Counterexample trace:** During a same-generation lease renewal for generation `42` (switching from `fv_allow_42_1` to `fv_allow_42_2`), C1.2 and C1.4 commit in the kernel, and C1.5 (`VerifyServing`) passes with both `fv_allow_42_2` (active) and `fv_allow_42_1` (unreferenced) present. The agent process is killed (`SIGKILL`) at `node/agent/engine.go:1135` before `SetMeta("retained_applied")` persists the new model/bank. On restart or next `Apply`, `retained_applied` in `agent.db` still names `fv_allow_42_1` while the kernel's `forward` chain references `fv_allow_42_2` and `<retired-bank>` was never persisted. Worse, if the next `Apply` fails in C1.1 (`TCBatch`) before C1.2 runs `delete set inet fvpn <retired-bank>`, the unreferenced bank remains stranded.
     - Because `delete chain inet fvpn fv_deadline_guard` in C1.4 removes the **last** reference to `fv_allow_A` in the kernel, `delete set inet fvpn fv_allow_A` can and should be executed inside the **same C1.4 atomic release batch** immediately after `delete chain inet fvpn fv_deadline_guard`! That eliminates `<retired-bank>` state across `Apply` calls, eliminates the "at most one identified unreferenced previous bank" exception in `VerifyServing` (`candidates.md:148-149`), and closes the crash cut between C1.4 and `SetMeta("retained_applied")`.
  2. **Delayed unconditional install vs. `Baseline` (`candidates.md:204`):** The failure matrix at `candidates.md:204` claims that if `Baseline` commits while a delayed install is in flight, *"install references a removed old bank. Such transactions must fail atomically."*
     - **Counterexample trace:** Process 1 starts an **unconditional fallback** install (`C1.0`, or an install from `Baseline`, or an install when a global deny/host/QoS setting changed). Because there is no eligible prior bank in unconditional fallback, the install script creates `fv_allow_N` and `fv_token_g` from scratch and **does not reference `@fv_allow_A`**. If an orphaned `nft -f` child from that unconditional install were to commit after `Baseline`, it would **not** fail on a missing old bank — it would succeed and overwrite `Baseline`! (In practice, `command_lifecycle.go` kills and reaps the `nft` child via `SIGKILL` + `Wait` before `NftBatch` returns and before `nodelock` is released; `candidates.md:204` must rely on `nodelock` + child reaping before lock release or an explicit baseline epoch set, not on `@fv_allow_A` being referenced by every install).
- **Fix (Owner: `isolation`):**
  1. Delete `fv_allow_A` directly inside the C1.4 release batch (`delete set inet fvpn fv_token_g`, `delete chain inet fvpn fv_deadline_guard`, `delete set inet fvpn fv_eligible_g`, `delete set inet fvpn fv_allow_A`) so `VerifyServing` requires **zero** unreferenced allow banks after release.
  2. In `candidates.md:204`, state that `nodelock` is never released until any timed-out `nft` child has been `SIGKILL`ed and reaped (`command_lifecycle.go`), because unconditional-fallback installs do not reference `@fv_allow_A`.

---

#### `M-2` — `enforce.Apply` checks `timing.RenderStart < anchor` only *after* mutating live `tc` in `TCBatch` and returns without calling `installBaseline`

- **Where:** `node/enforce/apply.go:151-168`; `docs/poc/G2-followup31-design/current-guarantees.md` (G4, G8, G10, lines 61, 87, 97).
- **In design or already on `main`:** Already on `main` (and overstated in `current-guarantees.md:61, 97`).
- **Scenario:**
  In `enforce.Apply` (`node/enforce/apply.go:151-175`), `ex.TCBatch(commandPhase(ctx, "shape"), plan.TC)` mutates the live kernel `tc` hierarchy at line 151. Immediately afterward, at lines 158–168, `Apply` samples `timing.RenderStart, err = MonotonicNow()` and checks `if timing.RenderStart < anchor` (`m.NowMono`). Unlike the error branches immediately before (line 152) and after (line 174), lines 158–168 return an error **without calling `installBaseline`**, leaving the mutated `tc` hierarchy and any previously serving `nft` ruleset un-rolled-back by `Apply` (and similarly at lines 245–273 if `Samples` or a non-`ConntrackDeleteError` fails after `verifyServing`, where `Apply` returns `Verified: true` alongside a fatal error).
- **Proof (unit test & failure output):**
  Test `TestR24_M2_PostShapeClockCheckSkipsBaselineInApply` in `node/enforce/review_r24_test.go`:
  ```text
  === RUN   TestR24_M2_PostShapeClockCheckSkipsBaselineInApply
      review_r24_test.go:228: TODO(review r24): M-2: Apply mutated live TC (tcBatches=1) then failed at apply.go:164 (ISOLATION_POLICY_INVALID: monotonic render precedes model) without calling installBaseline (nftBatches=0)
  --- FAIL: TestR24_M2_PostShapeClockCheckSkipsBaselineInApply (0.00s)
  ```
- **Fix (Owner: `isolation`):**
  Call `installBaseline` on the `apply.go:158-168` error paths (or sample/validate `RenderStart >= anchor` before `TCBatch`), and never return `Verified: true` on a non-`ConntrackDeleteError` failure.

---

#### `M-3` — Netlab's `traceExecutor` and `guardLogScript` (`trace.go:72-89, 157-177`) reject C1's 4-rule guard and multi-line fenced release, and the follow-up 31 proof plan omits `ctunku`/`ctunkd` drop attribution and explicit B1 exclusion

- **Where:** `tests/netlab/cmd/fvpn-netlab/trace.go:72-89, 157-177`, `docs/poc/G2-followup31-design/proof-obligations.md` (lines 114–142), `docs/poc/G2-followup31-design/d133-comparison.md` (lines 16–36, 65–75).
- **In design or already on `main`:** In the follow-up 31 design (`proof-obligations.md`, `d133-comparison.md`) and `tests/netlab/cmd/fvpn-netlab/trace.go`.
- **Scenario:**
  1. **`traceExecutor.NftBatch` misclassifies C1.4 release as `"other"` (`trace.go:77`):**
     `trace.go:77` checks `if strings.TrimSpace(script) == "delete chain inet fvpn fv_deadline_guard" { kind = "guard_release" }`. C1.4's fenced release batch (`candidates.md:134-137`) has multiple lines (`delete set inet fvpn fv_token_g`, `delete chain inet fvpn fv_deadline_guard`, `delete set inet fvpn fv_eligible_g`, plus `delete set inet fvpn fv_allow_A`). `traceExecutor` therefore classifies C1.4 as `"other"`, causing `reload_analysis.py:55` (`incomplete guard brackets`) and `requireProducerTiming` (`tests/netlab/reload_trace_test.go:125`) to fail every C1 run.
  2. **`guardLogScript` (`trace.go:157-177`) errors out on C1.2's `return` rules:**
     `d133-comparison.md:20` and `proof-obligations.md:116` require running C1 with `FVPN_NETLAB_GUARD_LOG=1` (`NFLOG` group 29). However, `guardLogScript` matches every line starting with `"add rule inet fvpn fv_deadline_guard iifname "` or `"oifname "` and returns `"TEST ONLY unexpected guard drop shape"` if the line does not end with `" drop"` or if `iifname` / `oifname` appears more than once. In C1.2 (`candidates.md:104-107`), the first two rules in `fv_deadline_guard` start with `add rule inet fvpn fv_deadline_guard iifname ... return`! Thus `guardLogScript` unconditionally fails on every C1 install.
  3. **Incomplete drop attribution & B1 confounding in `d133-comparison.md`:**
     - `NFLOG` group 29 only decorates the terminal `drop` rules in `fv_deadline_guard`. If a bug in C1 includes an eligible device in `@fv_eligible_g` and `@fv_allow_A` (so its packet `return`s from `fv_deadline_guard`) but omits it from `@fv_allow_N`, the packet is dropped in `forward` by `ip saddr != @fv_allow_N counter name "ctunku" drop` or `ip daddr != @fv_allow_N counter name "ctunkd" drop` with **zero** NFLOG group-29 events.
     - `d133-comparison.md:65-75` keeps B1 (`TCP_RTO_MIN` / socket pacing) as a transport follow-up without explicitly forbidding B1 during the primary C1 acceptance run.
- **Proof (unit test):**
  Passing proof test `TestR24_M3_NetlabTraceAndGuardLogRejectC1Scripts` in `node/enforce/review_r24_test.go`, demonstrating that `trace.go`'s classifier labels C1.4 as `"other"` and `guardLogScript` fails on C1.2 with `"TEST ONLY unexpected guard drop shape"`.
- **Fix (Owner: `netlab`):**
  1. In `tests/netlab/cmd/fvpn-netlab/trace.go`, classify a batch as `"guard_release"` when it contains `delete chain inet fvpn fv_deadline_guard` and does not contain `add chain`, and update `guardLogScript` to skip rules ending in `" return"` so only the two terminal `" drop"` rules in `fv_deadline_guard` are decorated with NFLOG group 29.
  2. In `proof-obligations.md` and `d133-comparison.md`, require `ctunku` and `ctunkd` counter deltas to be `0` across the reload campaign alongside `NFLOG group 29 == 0`, and explicitly forbid enabling B1 during the C1 acceptance run.

---

### Low

#### `L-1` — Lock path typo in `current-guarantees.md:27` and ambiguous rule shape when `fv_eligible_g` is empty in `candidates.md:54, 97, 104-107`

- **Where:** `docs/poc/G2-followup31-design/current-guarantees.md:27`, `docs/poc/G2-followup31-design/candidates.md:54-60, 96-107`.
- **In design or already on `main`:** In the follow-up 31 design.
- **Scenario & Proof (code inspection):**
  1. `current-guarantees.md:27` states that `enforce.Apply` runs after `/var/lib/vpn-node/update.lock` is held, whereas `Config.ApplyDefaults` (`node/agent/config.go:130`) and `node/updater/config.go:53` place the shared lock at `/var/lib/vpn-node/update-journal/update.lock`.
  2. `candidates.md:97` (`add element inet fvpn fv_eligible_g { <eligible address list> } # omit if empty`) leaves `fv_eligible_g` empty when no device is eligible while still emitting the two `@fv_eligible_g` `return` rules (`candidates.md:104-105`), whereas `candidates.md:54, 158` refers to *"the unconditional guard"* when no prior verified bank exists (where `@fv_allow_A` does not exist in the table and therefore cannot be referenced by a `return` rule!).
- **Fix (Owner: `netlab` / `isolation`):**
  Update `current-guarantees.md:27` to cite `/var/lib/vpn-node/update-journal/update.lock`, and specify in `candidates.md` that whenever the eligible set is empty (including cold start, post-baseline, or unconditional fallback), `fv_eligible_g` and the two `return` rules are omitted from `fv_deadline_guard` (leaving only the two `iifname`/`oifname` `drop` rules and `fv_token_g`).

---

## 2. The owner's decision

- **What today's guard costs and how often:** Each policy/lease apply blackholes all member traffic on the node for **28.5 ms median (up to 94.7 ms)**, causing **22 UDP losses, 28 TCP retransmissions, and a 256.8 ms TCP stall across 100 reloads (`AT10`)**. Worse, due to bug `H-1` on `main`, nodes currently pay this blackout **every 10 seconds (8,640 times/day)** even when nothing changed; once `H-1` is fixed, it is paid on lease renewals (**every ~23h58m** at the 24h default ceiling, or **every 13m** at a 15m lease setting) plus member/FUP changes.
- **What C1 gives up, for how long, and compared with today's steady state:** If an apply commits slowly (`> 100 ms`) or the agent dies/stops (`SIGKILL`/`SIGSTOP`) in the ~28.5 ms window before guard release, today's guard drops all members immediately (`0 ms`), whereas C1 lets **unchanged, non-revoked** members keep forwarding until the local watchdog closes the node (**30–40 s** if the lock is free) or until their **already-verified previous kernel timer (`D_old`)** expires (at most `max_offline_lease_seconds`: **60 s to 24 h**, and only **~120 s** during normal lease renewals). Crucially, if the agent dies or stops **1 ms before or after `Apply`** (the other **99.9%+ of the time**), `main` **already** has that exact same 30–40 s watchdog / `D_old` exposure (`scenario_at10_test.go:135-154`); C1 adds zero exposure beyond `D_old` and still cuts off revoked/tightened members immediately.
- **Recommendation — Accept the trade and implement C1 after fixing `H-1` through `M-3`:** No stock-kernel design can be both hitless during the netlink commit-to-verify bracket and close unchanged members at `0 ms` if the agent dies before the second transaction, and C1's worst-case crash exposure is identical to `main`'s existing steady-state crash exposure (`<= D_old`).

---

## 3. Design recommendation

**Implement C1 after named changes (`H-1`, `H-2`, `H-3`, `H-4`, `M-1`, `M-2`, `M-3`, `L-1`)**: C1 is the only stock-kernel design that eliminates reload-window packet loss (`AT10` / `D-133`) while enforcing revocations and `D_old` kernel deadlines atomically, but it will fail its own `AT13` proof and netlab harness unless `H-1`–`M-3` are resolved in the implementation.

---

## 4. What held

### Question 1 — Today's guarantees (`G1`–`G10` in `current-guarantees.md`)

- **What held on `main`:**
  - **`G1` (Monotonic deadline anchor):** `Decide` (`node/agent/decide.go:625-663`) anchors every member deadline to `in.LeaseMonoSent + window` on `CLOCK_MONOTONIC` (`node/internal/timeanchor`) and blocks serving with `ForwardingBlockReasonLeaseExpired` if `in.MonoNow < in.LeaseMonoSent`. `memberTimeout` (`node/enforce/deadline.go:36-62`) subtracts `m.timeoutReserve` (`DeadlineInstallBound = 100ms`, `deadline.go:17`), truncates to `DeadlineGranularity = 1ms`, and caps by `TimeoutCeilSeconds`.
    - *Proofs:* `TestR24_Held_Q1_MonotonicAnchorAndSameBootAdoption` (`node/agent/review_r24_test.go`), `TestD104_AT13_MillisecondDeadline` (`node/enforce/deadline_test.go`), `TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity` (`node/enforce/review_r24_test.go`).
  - **`G2` (Install bound before open):** `guardedNFT` (`node/enforce/deadline_install.go:24-56`) atomically installs `fv_deadline_guard` at priority `-151` (`policy accept`, dropping `iifname` and `oifname` on `MemberIngress`) in the same `nft -f` transaction that replaces `forward` (`priority -150`). `Apply` (`node/enforce/apply.go:189-205`) checks `InstalledBy - RenderStart <= 100ms` *before* calling `releaseDeadlineGuard`.
    - *Proofs:* `TestD104_AT13_InstallOverrunNeverOpensGuard` (`node/enforce/deadline_test.go`), `TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity` (`node/enforce/review_r24_test.go`).
  - **`G3` (Kernel element ceiling):** `verifyAllowedLifetimes` (`node/enforce/deadline_read.go:112-145`) parses `nft list set inet fvpn allowed_v4` and rejects any element with `el.timeout > original` or `el.expires > remaining`.
    - *Proofs:* `TestD104_AllowedPrecisionRead` (`node/enforce/deadline_test.go`), `TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity` (`node/enforce/review_r24_test.go`).
  - **`G5` & `G7` (Immediate revocation, deny growth, and post-verify conntrack flush):** `Render` (`node/enforce/render_nft.go:27-35, 69-106`) flushes and repopulates `@allowed_v4`, `@denied_v4`, `@host_v4`, and `@resolver_v4` and replaces `forward` inside the guarded batch; `deleteConntrack` (`node/enforce/apply.go:254-273`) runs after `verifyServing` and returns `ConntrackDeleteError` (`Verified: true`) on command failure so retries do not tear down verified serving.
    - *Proofs:* `TestApply_ConntrackFailureKeepsServing` (`node/enforce/conntrack_test.go`), `TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth` (`node/agent/followup35_restart_test.go`).
  - **`G6` (Counter continuity):** `Render` (`node/enforce/render_nft.go:116-123`) only emits `add counter` for counters absent from `state` and never emits `delete counter` or `flush table`.
    - *Proofs:* `TestRender_SecondPassKeepsCounters` (`node/enforce/render_test.go`), `TestR24_Held_Q1_GuardInstallBoundLifetimeCeilingAndCounterContinuity` (`node/enforce/review_r24_test.go`).
  - **`G9` (Same-boot adoption and watchdog):** `adoptStoredServing` (`node/agent/restart.go:18-95`) adopts an already-verified ruleset on the same `BootID` without calling `Plane.Apply`; `Watch.Once` (`node/watchdog/watch.go:53-98`) installs `Baseline` when `now - beat.Mono > stale` (`30s`) or `now >= beat.PolicyExpires`.
    - *Proofs:* `TestR24_Held_Q1_MonotonicAnchorAndSameBootAdoption` (`node/agent/review_r24_test.go`), `TestWatch_StaleAppliesBaselineOncePerCheck` (`node/watchdog/watch_test.go`).
- **What did not hold on `main`:** Findings `H-1`, `H-2` (pre-`nft` orphan `tc` class deletion), `H-3` (multi-device `DeadlineMono` collapse), `H-4` (stale `State{}` and expired `ctx` in `installBaseline`), and `M-2` (post-`TCBatch` clock check skipping `installBaseline`).

### Question 2 — C1's safety (Deadline, Revocation, Eligibility, Fencing, Isolation)

- **What held in C1's design (`candidates.md`):**
  - **Deadline intersection (`@fv_allow_A` $\cap$ `@fv_allow_N`):** During `[InstallCommit, ReleaseCommit]`, `fv_deadline_guard` at priority `-151` only `return`s packets matching `@fv_eligible_g` **and** `@fv_allow_A`, which then must also pass `@fv_allow_N` in `forward` at priority `-150`. Because `@fv_allow_A` was verified by `verifyAllowedLifetimes` when generation `A` was installed, its kernel element timeout expires in the kernel's monotonic GC timer at `<= D_old` even if generation `N`'s install exceeded `100 ms` or if the agent is killed/stopped before C1.4 release. Once `@fv_allow_A` expires at `D_old`, `fv_deadline_guard` drops the packet at `-151` regardless of `@fv_allow_N`.
  - **Revocation and changed bindings:** Any revoked member/device, removed IP, reassigned IP (`planRevokes` in `node/agent/revocation.go`), or device with `D_new < D_old` is excluded from `@fv_eligible_g` and dropped at priority `-151` from the instant C1.2 commits.
  - **Release token fencing (`fv_token_g`):** Including `delete set inet fvpn fv_token_g` in C1.4's single atomic release batch guarantees that a duplicate or stale release for an earlier guard `g0` fails with `ENOENT` in the kernel without deleting a newer `fv_deadline_guard`.
  - **Isolation (`ISO01`/`ISO02`), SNAT, DNS, and metering inside the window:** In nftables, `return` in a base chain (`fv_deadline_guard` at `-151`, `policy accept`) returns to the base chain's policy and continues evaluation at the next base chain on the `forward` hook (`forward` at `-150`, `policy drop`). Every packet that passes `fv_deadline_guard` therefore traverses `forward` (`@denied_v4`, `@host_v4`, `@resolver_v4` TCP/UDP 53 exception, `ct state established,related` on download, IPv6 drop, member-to-member drop, `u_<dev16>`/`d_<dev16>` usage counters, and `meta priority set` QoS classification) and `postrouting` (`snat to` using `@fv_allow_N`).
    - *Proofs:* `TestR20_Held_VerifyServingCatchesResolverBindingQoSAndSNATDrift` (`node/enforce/review_r20_test.go`), `TestRender_Golden` (`node/enforce/render_test.go`).
- **What broke in C1's design:** Findings `H-2`, `H-3`, `M-1`, and `L-1`.

### Question 3 — The trade, quantified from the code

- **How often is the guard cost paid?**
  - **Current `main` (with bug `H-1`):** Every **10 seconds** (`PollSeconds = 10`, `HeartbeatSeconds = 10` in `node/agent/config.go:133-138`) on every serving node (**360 times/hour, 8,640 times/day**), because `Engine.Step` (`node/agent/engine.go:619`) only sets `retain = true` when `ioCtx.Err() != nil`.
  - **Production cadence once `H-1` is fixed:**
    1. *Serving lease renewal:* `renewLead(window) = min(RenewLeadSeconds=120s, window/2)` (`node/agent/sync.go:142-151`, `node/agent/config.go:145-147`). Central issues leases with `max_offline_lease_seconds` in `60..86400` s (`DefaultMaxOfflineLeaseSeconds = 86400` = 24h in `internal/leases/leases.go:28`; Admin Portal warning threshold `900` s = 15m in `docs/handover/calibration.md:53`). Thus lease renewal triggers `Apply` once every **86,280 s (~23h58m)** at the 24h ceiling, once every **780 s (13m)** at a 15m setting, or once every **30 s** at the 60s test-lab floor.
    2. *Desired-state generation changes:* Triggered when a member authorizes/revokes a device (`device_sync_seconds` recommended `60 s`, `calibration.md:37`) or crosses a FUP tier (`fup_sweep_seconds` recommended `60 s`, `calibration.md:47`).
- **How long is the C1 exposure at the cut?**
  - Central computes each member's lease ceiling in `ComputeMemberLeaseEnd` (`internal/leases/leases.go:193-208`) as `min(signed_server_time + max_offline_lease, effective_service_end, node_lease_end)`, where `max_offline_lease_seconds <= 86400` (24h). The agent anchors `DeadlineMono = in.LeaseMonoSent + min(leaseWindow, b.ValidUntil - IssuedAt)` (`node/agent/decide.go:625-657`) and renders `timeout = min(DeadlineMono - RenderStart - 100ms, TimeoutCeilSeconds)` (`node/enforce/deadline.go:36-62`).
  - If the agent is killed (`SIGKILL`) after C1.2 commits and before C1.4/C1.6, the OS releases `update.lock` when the process exits, and `fvpn-watchdog` (`WatchdogStaleSeconds = 30` in `node/agent/config.go:139`, polling every `10 s` in `node/watchdog/cmd/fvpn-watchdog/main.go:46`) installs `Baseline` within **30–40 seconds**.
  - If the agent is stopped (`SIGSTOP` / hung in D-state) **while holding `update.lock`** (or if `fvpn-watchdog` is also dead), `Watch.Once` (`node/watchdog/watch.go:64-73`) gets `nodelock.ErrBusy` and defers `Baseline`; eligible members then continue forwarding until their old verified kernel element timeout `D_old` in `@fv_allow_A` expires:
    - During a routine lease-renewal `Apply` (which starts at `renewLead = 120 s` before old lease expiry), `D_old - RenderStart` is at most **~120 seconds (2 minutes)**.
    - During an `Apply` triggered by a desired-state change immediately after a fresh lease was issued, `D_old - RenderStart` is bounded by `min(max_offline_lease_seconds, EffectiveServiceEnd, NodeLeaseEnd, ValidUntil, TimeoutCeilSeconds)` — at most **24 hours (`86,400 s`)** under the §16.3 ceiling, or **15 minutes (`900 s`)** at the portal warning threshold.
- **Comparison with today's steady state on `main` (and watchdog behavior):**
  - On `main` today, if `fvpn-agent` is killed (`SIGKILL`) or stopped (`SIGSTOP`) while serving **outside** the ~28.5 ms guard window (as tested in `tests/netlab/scenario_at10_test.go:135-154`), the kernel ruleset stays live for **30–40 seconds** until `fvpn-watchdog` fires (if `update.lock` is free), or up to **`D_old` in `@allowed_v4`** (if `update.lock` is held by a stopped process or the watchdog is dead).
  - Therefore, C1 **does not increase** the node's maximum exposure bound beyond `D_old`; it only extends steady-state `D_old` survival across the ~28.5 ms `Apply` window for unchanged members.
  - *Proofs:* `TestR24_Held_Q3_WatchdogStaleAndHeldLockBehavior` (`node/watchdog/review_r24_test.go`), `TestAT13_ISOF603_WatchdogCannotOverwriteRenewal` (`node/watchdog/followup28_test.go`).

### Question 4 — The rejected candidates (`C3` and `C2`)

- **`C3` (Kernel absolute or `CLOCK_MONOTONIC` rule predicate):**
  - Verified against Linux kernel `net/netfilter/nft_meta.c` (`NFT_META_TIME_NS` evaluates `ktime_get_real_ns()`, i.e. `CLOCK_REALTIME` wall clock, which jumps on NTP/clock steps and violates `D-104` / `AT13`) and `net/netfilter/nf_tables_api.c` (`nft_set_elem_init` converts relative `NFTA_SET_ELEM_TIMEOUT` milliseconds into a kernel expiration timestamp at netlink commit time). Stock Linux nftables on Ubuntu 24.04 (`amd64`/`arm64`) provides no rule expression that compares packet arrival against a caller-supplied absolute `CLOCK_MONOTONIC` timestamp. The rejection of `C3` holds.
- **`C2` (Unfenced stage-then-swap without a guard chain):**
  - Staging `@fv_allow_N` in transaction 1 and swapping `forward` in transaction 2 without a guard chain delays revocations and deny-list growth until transaction 2 commits (`candidates.md:169-180`). Attempting to restrict `C2` by replacing `forward` in transaction 1 while keeping `@fv_allow_A` until transaction 2 either (a) delays newly authorized devices and deadline extensions until transaction 2 while still requiring two full chain replacements, or (b) if `@fv_allow_N` is referenced in transaction 1 without a guard chain, exposes newly added or deadline-extended members to unverified `D_new` timeouts if transaction 1 overruns 100 ms. The rejection of `C2` holds.

### Question 5 — Is there a candidate that keeps every guarantee while remaining hitless?

- **Why no stock-kernel design can keep strict `G2`/`G4` (`0 ms` post-install exposure on kill/late-commit) while being hitless during `[InstallCommit, ReleaseCommit]`:**
  - Any netlink transaction `T1` that installs relative timeouts into a kernel set commits at an exact kernel instant unknown to user space until `nft -f` returns and user space samples `InstalledBy = MonotonicNow()`.
  - If the agent process is killed (`SIGKILL`) immediately after `T1` commits in the kernel (before user space can issue a second transaction `T2`), the kernel remains in the state left by `T1`.
  - For an unchanged member `A1`'s packets arriving after `T1` commits:
    - To be **hitless**, `T1`'s kernel state must **accept** `A1`'s packets before `T2` runs.
    - To satisfy **strict `G2`/`G4`** (*"if `T1` took `> 100 ms` from `RenderStart` and the agent died before `T2`, `A1` must be dropped at `0 ms`"*), `T1`'s kernel state must **drop** `A1`'s packets before `T2` runs.
  - Because a single kernel state after `T1` cannot simultaneously accept and drop `A1`'s packets before `T2`, any design that is hitless during `[T1, T2]` must admit `A1` during `[T1, T2]` under a kernel timer that was already verified before `T1` (`@fv_allow_A`, i.e. C1).
  - Note that fixing `H-1` eliminates >99% of `Apply` calls in steady-state production without relaxing `G2`/`G4`, but `AT10` explicitly executes 100 consecutive generation reloads with device churn, which still requires C1 to achieve zero reload-window drops.

### Question 6 — The proof obligations and the `D-133` comparison plan

- **What held in `proof-obligations.md` and `d133-comparison.md`:**
  - The paired-comparison structure in `d133-comparison.md` (same 100-reload `AT10` workload, same `stream.py` cadence, same producer/executor timing brackets from `reload_trace_test.go`, and explicit retention of `B1`'s unexplained `1,501.432 ms` failure record from follow-up 29) is sound in structure.
  - *Proofs:* `TestR24_M3_NetlabTraceAndGuardLogRejectC1Scripts` (`node/enforce/review_r24_test.go`) proves the exact recognizer/decorator bugs (`M-3`) that must be fixed in `tests/netlab/cmd/fvpn-netlab/trace.go` before running the C1 campaign.

### Question 7 — Additional implementation hazards surfaced for the owner and implementer

- Surfaced as Findings `H-1` (every-10s `Apply` on `304 Not Modified`), `H-2` (`buildQoS` fair-share recalculation + pre-`nft` orphan `tc` class deletion into `1:fffe`), `H-3` (per-member `DeadlineMono` collapse in `decide.go:682-695` + `e.appliedModel` retention across withdraw/reboot), `H-4` (`installBaselineModel` ignoring `loadState` errors and using stale `State{}` / expired `ctx`), `M-1` (deleting `fv_allow_A` directly inside C1.4 release), `M-2` (post-`TCBatch` clock check skipping `installBaseline`), and `M-3` (`trace.go` recognizer and `guardLogScript`).

---

## 5. Not examined

- Live privileged Docker `netlab` kernel execution (`tests/netlab` with `-tags=netlab`): C1 is a design proposal with no production code implemented yet, and all findings against `main` (`H-1`–`H-4`, `M-2`) and against the C1 design/harness (`H-2`, `H-3`, `M-1`, `M-3`, `L-1`) were proven deterministically via unit tests in `node/enforce`, `node/agent`, and `node/watchdog` without launching Docker containers.
- Central control-plane, worker, portal, updater, and installer packages outside lease window calculation (`internal/leases/leases.go`) and node lock paths.
