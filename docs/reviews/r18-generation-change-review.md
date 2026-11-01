# Adversarial Review r18: generation-change window, availability note, and TEST ONLY command observation

Branch: `r18-generation-change-review` (from `main` at `5bd75102`)
Date: 2026-11-01
Reviewing model: `grok/grok-4.7`

Scope is the provisional generation-change window (D-135, D-136), agent-core follow-up 31 (#369, `910c7f2d`) including `docs/node/followup31-generation-change-availability-2026-11-01.md`, and the opt-in TEST ONLY command-lifecycle observation (netlab follow-up 22, #365, D-138) with the agent caller added in that same follow-up.

Proofs are `TestReviewR18_*`. A `TODO(review)` failure is a finding. A passing test is recorded under What held. Product code was not changed.

Commands (2026-11-01, `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, go1.27.3, `TMPDIR` and `GOTMPDIR` under `.agent/tmp`). No integration database was opened. No netlab stack, namespace, or container was started, so no other suite was waited on. `FVPN_LAB_PREFIX` was not needed. No `fvpn-r18` object was created.

```
go test -count=1 -timeout 240s -run 'TestReviewR18_' ./node/agent/ ./node/enforce/
go test -count=1 -timeout 180s -run 'TestF31_AT13_' ./node/agent/
go test -bench=BenchmarkReviewR18_DisabledLifecycle -benchtime=30ms -run '^$' ./node/enforce/
```

The review command exited 1. `node/agent` 16.688s failed on the four `TODO(review)` tests below. `node/enforce` 0.194s failed on the one `TODO(review)` test below. `TestF31_AT13_` exited 0 (0.525s). The disabled-path benchmark exited 0: `3108961` iterations, `11.58 ns/op`, `0 B/op`, `0 allocs/op`.

## Summary

| ID | Severity | Owner | Where | Summary |
|---|---|---|---|---|
| M-1 | Medium | agent-core | `node/agent/engine.go:495`, `node/agent/serving.go:161-182`, `node/agent/serving.go:232-236` | After SIGTERM, `finishServingStep` still delivers an observation. The defer captured the context from before `controlBudgetKey` was attached, so `controlContext` does not see the canceled control budget. |
| M-2 | Medium | agent-core | `node/agent/serving.go:136-152`, `node/agent/serving.go:170-182`, `node/agent/engine.go:543`, `deploy/node/systemd/fvpn-agent.service:12-19` | A serving-read failure with quarantine pending spent 14.021s. The advertised local bound is 10s. Two independent 5s quarantine contexts run back to back, then the observation uses a third 5s context. The unit sets no `TimeoutStopSec`. |
| M-3 | Medium | agent-core, wire-nodes | `node/agent/decide.go:192-196`, `node/agent/decide.go:261-267`, `node/agent/decide.go:467-498` | Replaying the previous lease keeps every member on the node-wide baseline for 40 generation steps. The node ACKs the new generation, reports `policy_missing`, safety `passed`, and provisioning `ready`, and has no cap, hysteresis, or distinct alarm. This is a separate exposure from the 100ms install guard. |
| L-1 | Low | netlab, agent-core | `node/enforce/command_lifecycle.go:149-159`, `node/enforce/command_resources_linux.go:64-85`, `node/enforce/apply.go:181-183` | With the TEST ONLY lifecycle key set, `runCommand` reads cgroup and proc state after the install context has already passed `DeadlineInstallBound`, and those reads do not select on the context. |
| L-2 | Low | agent-core | `docs/node/followup31-generation-change-availability-2026-11-01.md:411`, `docs/node/followup31-generation-change-availability-2026-11-01.md:511-517` | The written D-133 acceptance rule accepts a candidate whose only change is a higher sibling rate ceiling. Option (b) requires the stricter ceiling. The rule compares percentiles, bindings, revocation cutoff, and expiry. |

## Findings

### M-1 (Medium): a canceled Step still completes an observation POST

- **Where:** `Step` defers `finishServingStep(ctx)` before it stores `controlBudgetKey` (`node/agent/engine.go:495`, then `499-501`). The deferred value is that earlier context. `finishServingStep` strips cancellation and, when the shutdown read fails, calls `observe` and `flush` on a fresh 5s context (`node/agent/serving.go:161-182`). `controlContext` returns the context it was given when the key is absent (`node/agent/serving.go:232-236`). `flush` posts through that context (`node/agent/observe.go:241-259`).
- **Scenario:** The node is serving. The first live read succeeds and the caller cancels, which is the SIGTERM point follow-up 31 stops before local store and Apply work (`node/agent/engine.go:534-535`). `Step` returns `context.Canceled` and does not sync desired state or renew. The deferred shutdown read then fails closed and withdraws, and the observation handler still accepts a report. The report's forwarding bit is false. The control timeout is already canceled by the time the POST runs; the POST does not use it. Follow-up 31's comment at `engine.go:537-542` says control I/O stops with the caller. This POST does not.
- **Proof:** `TestReviewR18_Q4_ControlRequestAfterSIGTERM`.

  ```
  TODO(review): observation POST completed after the Step context was canceled (posts 2 -> 3, report_ok=true forwarding=false). finishServingStep is deferred on the pre-budget context (engine.go:495) and observe/flush use controlContext, which falls back to that uncancellable context (serving.go:161-182, serving.go:232-236)
  ```
- **Fix:** agent-core. Give `finishServingStep` the context that carries `controlBudgetKey`, so `observe` and `flush` use `ioCtx` and stop when the caller does. Keep the bounded local read and the baseline on the uncancellable cleanup context. The forwarding bit in this proof was already false; the fix is the control call, not another dataplane close.

### M-2 (Medium): shutdown work stacks past the 10s local bound

- **Where:** `rejectServing` quarantines on a 5s `context.WithoutCancel` (`node/agent/serving.go:136-152`). `finishServingStep` does it again on a new 5s context when `quarantinePending` is still set, then forces `ok` false and observes on a third 5s context (`node/agent/serving.go:170-182`). `quarantine` does not clear `quarantinePending` (`node/agent/quarantine.go:109-145`). The local Apply budget added in `910c7f2d` is `controlPassTimeout()+5s`, which is 10s at the default 30s watchdog (`node/agent/engine.go:543`, `node/agent/serving.go:220-226`). That budget is a different context from the defer. `deploy/node/systemd/fvpn-agent.service` sets `Restart=on-failure` and does not set `TimeoutStopSec`. `Close` adds its own 3s `context.Background` and then `loops.Wait` outside that timeout (`node/agent/usage.go:412-425`). The updater stops the unit with `systemctl stop` (`packaging/installer/config.go:271`).
- **Scenario:** A serving node has quarantine pending. The next live read fails. The first quarantine blocks until its 5s context ends, the Step's control budget is already exhausted, and the defer's second quarantine blocks for another 5s. The observation that follows waits out a 2s control delay inside its own 5s context. Forwarding ends closed. The Step has already overrun the bound follow-up 31 described as at most ten seconds for the local context. This run did not call `Close` and did not approach systemd's 90s default stop timeout.
- **Proof:** `TestReviewR18_Q4_ShutdownExceedsLocalBound`. The fake control plane blocked inside `Quarantine` until the context ended, and the server delayed the observation by 2s.

  ```
  TODO(review): Step spent 14.021469915s after a serving-read failure with quarantinePending set; advertised local bound is 10s (controlPassTimeout+5s). Quarantine ran 2 times on independent 5s contexts (rejectServing then finishServingStep) and the following observation used another fresh 5s context. TimeoutStopSec set=false. fvpn-agent.service has no TimeoutStopSec; systemd's default is 90s and this run did not approach it
  ```
- **Fix:** agent-core. One shutdown budget should cover the serving read, quarantine, baseline, and the final observation, and a second caller should not start a fresh 5s quarantine after the first one has already used its allowance. Set `TimeoutStopSec` on `fvpn-agent.service` from that same budget so `systemctl stop` from the updater matches the process.

### M-3 (Medium): a replayed lease holds the whole node on the baseline, and the ACK hides it

- **Where:** `leaseGenerationStale` is the block, and the only reason it adds is `policy_missing` (`node/agent/decide.go:192-196`, `node/agent/decide.go:553-565`). `safetyOf` does not read that mismatch (`node/agent/decide.go:467-498`). `Decide` still sets `Applied` when desired state is acceptable and no apply error is retained, and provisioning becomes `ready` from that ACK (`node/agent/decide.go:261-267`). The ordinary stale branch baselines, ACKs, and returns (`node/agent/engine.go:949-960`). Nothing in the agent counts how long the serving lease has named another generation.
- **Scenario:** The node is serving generation 3 with two members. The process renews once, and the test server then replays that signed lease. Desired generation 4 drops one member. Forty following steps each renew, receive the same lease, install the node-wide baseline, and ACK generation 4. The removed member is absent from every later Apply. `serving` stays false. The decision reason stays `policy_missing`, safety stays `passed`, and provisioning stays `ready`. There is no streak, ceiling, or other reason. The 100ms `DeadlineInstallBound` is the nft install guard from r17 question 6. It does not bound this hold. Central's one-shot `stale_generation_grace_at` compares the reported applied generation with `desired_config_generation` (`internal/nodes/observations.go:258-268`). After this ACK the two are equal, so that grace does not describe the hold. `PublicationDataPathFailure` returns `stale_generation` only while applied is below desired (`internal/nodes/publication_evidence.go:48-50`); a failed check after that returns `check_failed` before `forwarding_disabled` (`internal/nodes/publication_evidence.go:79-85`). Those two central functions were read and not executed.
- **Proof:** `TestReviewR18_Q5_BaselineHoldHasNoCapOrDistinctAlarm`.

  ```
  TODO(review): 40 stale-lease steps stayed on the node-wide baseline (applied=4 serving=false) with block reason policy_missing, safety passed and provisioning ready. No cap, hysteresis or distinct alarm fired. DeadlineInstallBound is 100ms and does not cover this outage; the reason token is the same one Decide uses when the document is absent (decide.go leaseGenerationStale)
  ```
- **Fix:** agent-core and wire-nodes. Keep the baseline. Add a block reason that is only the generation mismatch, and a node-side cap or alarm once the acknowledged generation stays ahead of the serving lease. Central should treat that report as the generation hold even though applied generation has caught up. Do not waive the losses.

### L-1 (Low): the TEST ONLY command sample ignores the install deadline

- **Where:** The enabled `runCommand` calls `commandResources` before `Start` and does not consult `ctx.Err` (`node/enforce/command_lifecycle.go:149-159`). On Linux that reads `/proc/self/cgroup`, `/proc/self/mountinfo`, and up to 64 ancestor cgroup files (`node/enforce/command_resources_linux.go:64-88`). `Apply` builds the install context with `DeadlineInstallBound` (100ms) and passes it to `NftBatch`, which calls `runCommand` (`node/enforce/apply.go:181-183`). The agent attaches the key only from `measurementEnabled` (`node/agent/engine.go:1040-1043`, `node/agent/measurement.go:121-123`).
- **Scenario:** Both test gates are on and the install context is already past 100ms. `runCommand` still records a resource sample, performs the proc reads, and runs `true`. On this host the call returned in 2.549ms, so the sample itself did not consume the whole guard. The context cannot interrupt the reads. A slow cgroup walk sits inside the guarded nft call. The disabled path is unaffected. This key is not reachable from a production config, the environment, or the installer writer (What held §3).
- **Proof:** `TestReviewR18_EnabledSampleIgnoresInstallDeadline`.

  ```
  TODO(review): runCommand sampled command resources after the 100ms install context was already deadline_exceeded (elapsed 2.548987ms, /proc reads syscr +131). commandResources ignores ctx and runs before Start, so an enabled Apply can stay inside NftBatch past DeadlineInstallBound
  ```
- **Fix:** netlab and agent-core. Take the resource sample only while the caller's context is still live, and keep it outside the 100ms install context. Leave the disabled branch as the single context lookup.

### L-2 (Low): the D-133 rule does not notice a looser QoS ceiling

- **Where:** Option (b) retains the stricter QoS ceilings (`docs/node/followup31-generation-change-availability-2026-11-01.md:411`). The acceptance rule that would compare A and B requires no later revocation cutoff, no new or stale binding, no larger expiry bound, non-worse percentiles, and strictly lower restrictive p90 and p99 (`docs/node/followup31-generation-change-availability-2026-11-01.md:511-517`). A ceiling comparison is not part of that rule. Option (b) is not implemented.
- **Scenario:** A candidate improves every reported percentile, keeps the same bindings, cutoff, and expiry, and raises one sibling ceil from 1,000,000 bps to 5,000,000 bps. The written rule accepts it. Both B legs of an A1,B1,A2,B2 run would pass. The sibling can send more than the frozen A candidate allowed.
- **Proof:** `TestReviewR18_Q2_AcceptanceGateMissesQoSCeiling`. The predicate in the test is that acceptance paragraph.

  ```
  TODO(review): D-133 acceptance at docs/node/followup31-generation-change-availability-2026-11-01.md:511-517 accepts a candidate whose only regression is a sibling ceil from 1000000 to 5000000 bps; percentiles improve and bindings, cutoff and expiry are unchanged, so both B legs would pass
  ```
- **Fix:** agent-core. Put the stricter per-member and node ceilings into the D-133 pass/fail rule before any implementation of option (b). A higher ceil fails the leg even when the loss percentiles improve.

## What held

### 1. The stale window does not forward a grant the new document removed

`TestReviewR18_Q1_OrdinaryStalePathDoesNotForwardRemovedGrant`, `TestReviewR18_Q1_RecoveryCandidateDoesNotForwardRemovedGrant`, `TestReviewR18_Q1_StaleWindowStretchStaysClosed`, `TestReviewR18_Q1_FailedApplyDoesNotRestorePreviousGeneration`, and `TestReviewR18_Q1_MemberPathReasonDoesNotInvertServing` passed.

Follow-up 31's correction holds. On the ordinary path `applyError` is nil, `needRenew` is false before enforce, and enforce baselines and ACKs generation 4 without an Apply. The stored lease id is unchanged. The removed `100.64.1.2/32` grant is absent from the installed model. `renewAppliedLease` then obtains generation 4 and the following Apply serves only the surviving member.

On the retained-error path, enforce runs `recoveryDecision` only because `applyError` is set. The stored lease generation stays on the previous generation. The removed grant is not installed. `recoveryDecision` clears bindings on a copy (`node/agent/recovery.go:8-18`). A separate `Decide` on a copy whose lease generation was rewritten to match, and whose bindings were left in place, does admit a new device of a member the old lease still names. The engine does not install that device. The rewrite is not used as member authorization.

The window does not become a fail-open when it is stretched. After the new desired document is stored and before enforce, a restart adopts nothing: applied, desired, and lease generations disagree, and `Boot` baselines (`node/agent/restart.go:15-19`, `node/agent/engine.go:409-421`). Five steps that keep receiving the cached previous lease stay closed. Five failed renewals stay closed. Desired generations 5, 6, and 7, still without the removed member, stay closed. A wall step backward by one hour and forward by 30 minutes stays closed. Moving the monotonic clock two hours forward, past the old lease deadline, stays closed. No later Apply contains the removed prefix.

Four forced Apply failures after the restrictive generation leave the installed model with zero bindings. The previous generation's grants are not restored (D-135).

`rejectMemberPath` latches `memberPathPending` and withdraws. The following stale step keeps `Serving` false and does not install the removed grant. The heartbeat reason did not come out as `member_path_unconfigured` together with a `policy_missing` decision, which is the split the test treats as a misreport. That combination is not filed.

### 2. Today's code matches option (c), and it does not trust an unauthenticated document

`TestReviewR18_Q2_UnverifiedDesiredAndExpiredLeaseStayClosed` passed, including `corrupt`, `lease_expires_during_stale_window`, `freeze`, and `new_device_not_admitted_from_desired_alone`. The ordinary-path test in §1 is the revocation point for option (c): the baseline is installed before `renewAppliedLease`.

A corrupted signature leaves the previous serving model and applied generation in place. An old lease that expires during the window, by a two-hour monotonic step while the server is still replaying that lease, stays closed and does not install the removed grant. `freezeIdentity` withdraws during the window, and the next step stays closed. A desired generation that adds `100.64.9.9/32` for a member the old lease already names is not installed while `SetLeaseGeneration` keeps the lease on generation 2.

Options (a) and (b) are not in the binary. The note rejects unconstrained make-before-break because it would wait for a lease after a known revocation. The product does not wait: the ordinary path baselines first. Intersection serving is not installed, so it does not authenticate a new desired document early, and it does not keep a per-member subset across meters, QoS classes, member paths, freeze, D-135 rollback, or restart. Restart inside the window baselines (§1). The D-133 gap that remains is L-2.

### 3. The observation keys stay out of production, and the disabled path is a lookup

`TestReviewR18_Q3_ProductionConfigCannotEnableObservation`, `TestReviewR18_DisabledLifecycleDoesNotAllocate`, `TestReviewR18_DisabledRunCommandMatchesRun`, `TestReviewR18_RenderCaptureStaysInMemory`, and `TestReviewR18_EnabledCancelReapsChild` passed. `BenchmarkReviewR18_DisabledLifecycle` reported `0 B/op` and `0 allocs/op`.

`LoadConfig` of a production-shaped `agent.json` leaves `TestMode` and `TestTimingPath` empty while `FVPN_TEST_MODE`, `TEST_MODE`, and `FVPN_TEST_TIMING_PATH` are set. A `test_timing_path` without `test_mode` is rejected (`node/agent/config.go:173-175`). `measurementEnabled` is both flags (`node/agent/measurement.go:121-123`). The only `WithCommandLifecycleForTest` in `node/agent/engine.go` sits inside `if e.measurementEnabled()`. That file does not call `WithApplyCaptureForTest`. `packaging/installer/config.go` does not name `test_mode`, `test_timing_path`, `TestMode`, or `TestTimingPath`. `renderAgent` builds `agent.Config` without those fields (`packaging/installer/config.go:196-221`).

`captureRenderForTest` without the key leaves the capture struct untouched. With the key, the model stays in that struct. The agent observation path does not receive it.

`testing.AllocsPerRun` over `beginCommandMeasurement`, `commandPhase`, and `captureRenderForTest` with both keys absent is 0, and the context value is the same pointer. Disabled `runCommand` stays within one allocation of `exec.Cmd.Run` across 20 runs of `true`. Fifty disabled lookups do not move `/proc/self/io` beyond the counter reads used to measure them. The benchmark result is the same zero-allocation claim.

On the enabled path, a `wait` helper canceled through `CommandContext` returns signal 9, and the goroutine count returns to the baseline. That matches `TestD133_CommandLifecycleOutcomes`. The bound gap on that same path is L-1.

### 4. The graceful TestLab04 stop does not accept a torn or stale checkpoint, and cancel during Apply no longer manufactures a withdrawal

`TestReviewR18_Q4_TornCheckpointFailsClosed` passed (`graceful_restart_keeps_valid_lease`, `partial_heartbeat`, `stale_serving_heartbeat`, `torn_database`). `TestReviewR18_Q4_RevocationDuringShutdownFailsClosed` and `TestReviewR18_Q4_CanceledApplyIsThePreFixWithdrawal` passed. `TestF31_AT13_CancellationAfterLiveReadKeepsRestartAuthorization`, `TestF31_AT13_CancellationDuringLocalWorkKeepsRestartAuthorization`, and `TestF31_AT13_CancellationDoesNotMaskRealApplyFailure` passed.

`910c7f2d` changes `Engine.Step` to run local work on `context.WithoutCancel` for `controlPassTimeout()+5s`, and changes `TestLab04_RestartKeepsVerifiedLease` from `stopAgent` (SIGKILL 200ms after SIGTERM) to `timing-probes.py stop` with a 10s budget (`tests/netlab/agent/scenario_test.go:135-139`). `serving.go` is not in that commit. The lab population was not re-run. The unit mechanism of the note's attribution holds: `enforce` on an already canceled context, with an Apply that returns `ctx.Err()`, withdraws and retains an apply error. The same Apply, reached from `Step` after the parent is canceled inside Apply, sees a live local context and keeps serving. A real Apply error still withdraws under `TestF31_AT13_CancellationDoesNotMaskRealApplyFailure`.

`Close` rewrites a valid heartbeat before it returns. After that write, a truncated heartbeat, a heartbeat that still says serving at generation 1, and an `agent.db` replaced with `not a sqlite database` all fail closed on the next `Boot`. The partial and stale heartbeats come up with `serving` false and an empty binding list. The torn database either refuses `Start` or boots without serving and without bindings. A clean `restartEngine` of an untouched checkpoint adopts the same lease, generation, and bindings. `adoptStoredServing` requires a readable heartbeat whose boot, generation, and serving bit match the stored documents (`node/agent/restart.go:22-26`). These are artifact tests after `Close`, not a SIGKILL of a live process mid-write.

A restrictive generation whose baseline runs after the parent is canceled still uses a live local context, ends with serving false, and does not install `100.64.1.2/32`. A `pendingRevoke` entry for that address is set before the step. The early return in M-1 happens before `syncDesired`, so a revocation that has not been fetched yet is not applied on that interrupted step; the previous authorization stays until a later step. That is the serving-read cancel in `TestF31_AT13_CancellationAfterLiveReadKeepsRestartAuthorization`.

The control POST and the stacked shutdown budgets are M-1 and M-2.

### 5. The 100ms guard is still only the install guard

`TestReviewR18_Q5_BaselineHoldHasNoCapOrDistinctAlarm` is M-3. The same tree still bounds one nft install by `DeadlineInstallBound` (`node/enforce/apply.go:181`, `node/enforce/model.go:97`). The stale-lease hold is the generation replay above, not that guard. Historical losses stay counted. This review does not waive them.

## Not examined

- A live nft/tc packet-loss measurement, a quiet-host A1,B1,A2,B2 run, and the follow-up 31 populations (80 transitions, baseline-to-Apply p99 195.6ms). Those numbers were not reproduced.
- The 30-iteration `TestLab04_RestartKeepsVerifiedLease` rate, including the 5 failures attributed to unchanged `main`. The unit mechanism and the existing `TestF31_AT13_*` tests were run. The lab was not.
- A SIGKILL of a live agent during `SaveLease`, `MarkApplied`, or `WriteHeartbeat`. The proofs use a truncated heartbeat, a stale complete heartbeat, and a replaced database after `Close`.
- Central SQL for `stale_generation_grace_at` and `PublicationDataPathFailure`. The finding cites the source. No Postgres database was opened.
- Desired generation 0. `leaseGenerationStale` returns false when the desired generation is 0 (`node/agent/decide.go:561-563`). No fixture in this review published generation 0.
- Physical arm64, NetBird management propagation, and any host-network or sudo run.
- Options (a) and (b) as implementations. They are a design note. The product still baselines.
