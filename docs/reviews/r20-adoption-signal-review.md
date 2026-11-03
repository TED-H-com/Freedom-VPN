# Review r20: adversarial review of agent-core follow-up 36

- **Reviewer:** `r20-adoption-signal-review` (`agy/gemini-4-argon-high`)
- **Date:** 2026-11-03
- **Target:** `agent-core` follow-up 36 (`2f80722a..7d34c885`, commits `6715bf46` and `7d34c885`)
- **Scope:**
  1. Same-boot serving adoption refusal on changed `DeniedV4`, `ExtraDeniedV4`, and `HostV4` sets (`node/agent/restart.go`).
  2. Signal handling held from before `open`/`Boot` until after `eng.Close()` (`node/agent/cmd/fvpn-agent/main.go`).
  3. Durability and live-lab proofs (`node/agent/followup36_test.go`, `node/agent/followup36_durable_test.go`, `node/agent/cmd/fvpn-agent/signal_test.go`, `node/agent/followup35_restart_test.go`, `tests/netlab/agent/scenario_f36_test.go`).

---

## 1. Findings by severity

### High

#### `H-1` — Refused same-boot adoption drops unsampled member usage bytes into an unattributed `mapping conflict` gap (`MTR03` / `AT13`)

- **Where:** `node/agent/engine.go:437-451, 649-657`, `node/agent/restart.go:64-82`, `node/agent/usage.go:160-169`, `node/internal/meter/delta.go:453-497`, `node/agent/followup36_durable_test.go:174-186`.
- **Scenario:**
  1. Process 1 serves generation $g$ with member device $D$ and samples upload counter `u_<D>` at $4096\text{ B}$.
  2. Before Process 1 stops — either because `Close()` was interrupted before `FinalSample` committed (the exact scenario of `TestAT13_F36_InterruptedCloseDurability`), or because `preserveServingOnStop` intentionally left the serving ruleset active in the kernel while packets continued traversing it until Process 2's `Boot` — the live kernel counter `u_<D>` advances to $8192\text{ B}$ ($4096\text{ B}$ unsampled delta).
  3. Process 2 starts on the same boot (`boot_id` unchanged) with a changed `DeniedV4`, `ExtraDeniedV4`, or `HostV4` set.
  4. In `Boot`:
     - `loadRuntime` (`engine.go:649-657`) unmarshals `retained_applied` into `e.appliedModel`, but never calls `e.setMeterBindings(retained.Model.Bindings)`.
     - `adoptStoredServing` (`restart.go:64`) returns `false` at `sameRestartSets(*installed, d.Model)` before reaching `e.setMeterBindings(d.Model.Bindings)` (`restart.go:80`).
     - `Boot` (`engine.go:446-450`) installs `Baseline` (which replaces chains while preserving named per-device counters in `inet fvpn`) and calls `e.ensureUsage(ctx)`, wiring `kernelSource{bindings: e.meterBindings}` while `e.meterSnap` is still `nil`. The same-boot usage epoch remains open at the $4096\text{ B}$ counter checkpoint.
  5. On the first `Step`:
     - If desired state does not serve (`PublicationIntent = unpublished`/`draining`, expired lease, or barrier), `enforce` (`engine.go:971`) runs `e.handoff(ctx)` $\to$ `e.meter.FinalSample(ctx)`. `kernelSource.Read` observes `u_<D> = 8192` with `obs.Bindings = nil`.
     - If desired state serves a new generation where device $D$ was removed or its `BindingRevision` changed, `enforce` (`engine.go:1115`) overwrites `e.meterSnap` with the new generation's bindings before any sample runs.
     - In both cases, `attribute()` (`node/internal/meter/delta.go:490`) cannot map `u_<D>`'s $+4096\text{ B}$ delta (`conflict = true`), emits `UsageQualityGap` (`Detail: "mapping conflict"`, `Records: nil`), and drops the member's $4096\text{ B}$ of usage.
  6. `TestAT13_F36_InterruptedCloseDurability` (`followup36_durable_test.go:174-186`) missed this because the parent test did not boot a second `Engine`; it opened `meter.New` directly with a pre-populated `before.Observation.Bindings` slice from before the crash.
- **Proof (failing unit test):** `TestR20_H1_RefusedAdoptionDropsPendingUsageDelta` in `node/agent/review_r20_test.go`.
  ```text
  === RUN   TestR20_H1_RefusedAdoptionDropsPendingUsageDelta
      review_r20_test.go:143: H-1: refused same-boot adoption turned 4096 unsampled member bytes into an unattributed gap event (seq=2 records=0)
  --- FAIL: TestR20_H1_RefusedAdoptionDropsPendingUsageDelta (0.14s)
  ```
- **Fix (owner: `agent-core`):**
  1. In `loadRuntime` (`node/agent/engine.go:652`), populate `e.setMeterBindings(retained.Model.Bindings)` when restoring `e.appliedModel` from `retained_applied`.
  2. In `Boot` (`node/agent/engine.go:446-451`), when `adoptStoredServing` returns `false` after `Baseline` is installed and `e.ensureUsage(ctx)` opens the meter on the same boot with a retained `e.appliedModel != nil`, sample (or `handoff`) the open same-boot epoch against the retained `e.appliedModel.Bindings` before the first `Step` can overwrite `e.meterSnap` or run `handoff` with `nil` bindings.

---

#### `H-2` — `VerifyServing` and `VerifyBaseline` do not verify `@denied_v4` or `@host_v4` contents or set flags, and `VerifyBaseline` does not require `@allowed_v4` to be empty (`ISO01` / `AT13`)

- **Where:** `node/enforce/parse_nft.go:1180-1254`, `node/enforce/verify.go:14-176, 835-850`, `node/enforce/render_nft.go:53-57, 294-303`.
- **Scenario:**
  1. `sameRestartSets` in `node/agent/restart.go` only compares the stored `retained_applied` model against `d.Model`. It trusts that the kernel's live `@denied_v4` and `@host_v4` sets match `retained_applied`.
  2. In `node/enforce/parse_nft.go:1185-1197`, `parseSetElems` marks `denied_v4` and `host_v4` tainted (`rs.TaintedSets[name] = true`) whenever they contain prefix/range elements and discards their elements; `verifyServingRuleset` (`verify.go:14-122`) and `VerifyBaseline` (`verify.go:146-176`) never inspect `rs.Elements["denied_v4"]` or `rs.Elements["host_v4"]`.
  3. Consequently, until `netlab 27` lands, live kernel drift in `@denied_v4` or `@host_v4` passes both `adoptStoredServing` and steady-state `VerifyServing` / `VerifyBaseline` in two concrete windows:
     - **Window A (external or out-of-band nft mutation):** `nft flush set inet fvpn denied_v4` or `nft delete element inet fvpn host_v4 ...` while `fvpn-agent` is running or stopped leaves the forward chain drop rules `ip daddr @denied_v4 drop` / `ip daddr @host_v4 drop` matching an empty set.
     - **Window B (crash between `e.plane.Apply` and `SetMeta("retained_applied")`):** In `Engine.enforce` (`node/agent/engine.go:1087-1137`), `e.plane.Apply` updates the kernel ruleset at line 1087 before `e.store.SetMeta(ctx, "retained_applied", ...)` at line 1137. If `fvpn-agent` crashes in that window and the operator rolls back `agent.json` before starting `fvpn-agent`, `retained_applied` still holds the old model matching the rolled-back `agent.json`, so `sameRestartSets` returns `true` while the kernel holds the unrecorded Apply's sets.
  4. In addition, static inspection of `verify.go` and `render_nft.go` identified four requirements that `netlab 27`'s brief must include so the pair (`sameRestartSets` + kernel verifier) is complete:
     - **Interval merging (`auto-merge`):** `denied_v4` and `host_v4` are declared with `type ipv4_addr; flags interval; auto-merge;` (`render_nft.go:24-25`). The kernel merges adjacent or overlapping prefixes (e.g. `192.0.2.0/25` + `192.0.2.128/25` $\to$ `{"prefix": {"addr": "192.0.2.0", "len": 24}}` or `{"range": ["192.0.2.0", "192.0.2.255"]}`). `netlab 27` must normalize both the expected model prefixes and the parsed nft elements (`"prefix"`, `"range"`, and `/32` string elements) into canonical disjoint closed `[start, end]` IPv4 intervals before comparing.
     - **`Datapath.SNAT/32` in `@host_v4`:** Both `Render` (`render_nft.go:54-56`) and `RenderBaseline` (`render_nft.go:297-299`) append `netip.PrefixFrom(m.Datapath.SNAT, 32)` to `m.HostV4` when populating `@host_v4`. The verifier must include `m.Datapath.SNAT/32` in the expected `@host_v4` interval set whenever `m.Datapath.SNAT.IsValid()`.
     - **Set flags on `denied_v4` and `host_v4`:** `verifySetFlags` (`verify.go:835-850`) currently checks only `allowed_v4`, `probe_v4`, `probe_dst_v4`, and `resolver_v4`. It omits `denied_v4` and `host_v4`, so adding a `timeout` flag (and expiring elements) on `denied_v4` or `host_v4` is not rejected. `verifySetFlags` must require `"interval"` and reject `"timeout"` (and any element with `timeout`/`expires`) on `denied_v4` and `host_v4`.
     - **Baseline bare vs probe sets and `@allowed_v4` emptiness:** `ApplyBaseline` (`apply.go:280-281`) calls `ApplyBaselineModel(ctx, ex, Model{Datapath: dp})` with `m.Probe == nil`, where `RenderBaseline` (`render_nft.go:295`) does not emit `denied_v4` or `host_v4`, whereas `ApplyBaselineModel` with `m.Probe != nil` does emit and reference them. `VerifyBaseline` (`verify.go:153`) also currently checks `verifySetFlags(rs)` without checking `len(rs.Elements["allowed_v4"]) == 0` or `!rs.TaintedSets["allowed_v4"]`.
- **Proof (failing unit test):** `TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets` in `node/enforce/review_r20_test.go`.
  ```text
  === RUN   TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets
  === RUN   TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets/serving_emptied_and_timed_deny_and_host_sets
      review_r20_test.go:44: H-2: verifyServingRuleset accepted emptied @denied_v4 and @host_v4 with timeout flags against a model requiring 2 extra-denied and 2 host prefixes
  === RUN   TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets/probe_baseline_emptied_deny_host_and_nonempty_allowed_v4
      review_r20_test.go:78: H-2: VerifyBaseline accepted a probe baseline with emptied @denied_v4/@host_v4 and a non-empty @allowed_v4
  --- FAIL: TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets (0.00s)
  ```
- **Fix (owner: `netlab`):** In `netlab 27`, parse `"prefix"`, `"range"`, and `/32` address elements for `denied_v4` and `host_v4` in `node/enforce/parse_nft.go` into canonical disjoint IPv4 intervals (rejecting any timed element or unparseable expression), enforce `denied_v4` / `host_v4` interval equality (including `m.Datapath.SNAT/32` in `host_v4`) and set flags (`interval` required, `timeout` forbidden) in `verifyServingRuleset` and in `VerifyBaseline` when `m.Probe != nil`, and require `allowed_v4` to be untainted and empty in `VerifyBaseline`.

---

### Medium

#### `M-1` — `SIGTERM` during `Boot` runs `e.plane.Baseline(ctx, dp)` with the canceled signal `ctx`, overwriting the valid serving heartbeat with `Serving: false, BaselineOK: false` without installing the baseline (`AT13`)

- **Where:** `node/agent/cmd/fvpn-agent/main.go:98-126`, `node/agent/engine.go:425-457`.
- **Scenario:**
  1. `main.go:98` now installs `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` before `open` and `eng.Run(ctx)` (which calls `eng.Boot(ctx)`).
  2. Unlike `Engine.Step` (which creates a 10 s `newLocalBudget(ctx)` using `context.WithoutCancel(ctx)` for local completion), `Engine.Boot(ctx)` has no `localBudget` and passes the caller's `ctx` directly to `adoptStoredServing(ctx)` and `e.plane.Baseline(ctx, dp)` (`engine.go:446`).
  3. If `SIGTERM` arrives while `Boot(ctx)` is in `adoptStoredServing(ctx)` (for example during `nodelock.Acquire(ctx, ...)` at `restart.go:42` or `e.evaluate(ctx)` at `restart.go:53`), `adoptStoredServing(ctx)` returns `false`.
  4. `Boot(ctx)` then calls `e.plane.Baseline(ctx, dp)` (`engine.go:446`) with the already-canceled `ctx`. `KernelPlane.Baseline` fails immediately with `context.Canceled` without installing the baseline in the kernel.
  5. `Boot(ctx)` sets `e.baselineOK = false` and `e.serving = false`, calls `e.beat()` (`engine.go:452`) — overwriting the previous process's valid `Serving: true, BaselineOK: true` heartbeat file on disk with `Serving: false, BaselineOK: false, Reason: "safety_check_failed"` — and returns `nil`!
  6. `Run(ctx)` then sees `ctx.Done()` and returns `context.Canceled`, and `runWithOpen` (`main.go:121-125`) returns exit code `0`. A subsequent start can no longer adopt the serving ruleset because the on-disk heartbeat now says `Serving: false, BaselineOK: false`.
- **Proof (failing unit test):** `TestR20_M1_TermDuringBootPoisonsHeartbeatWithoutBaseline` in `node/agent/review_r20_test.go`.
  ```text
  === RUN   TestR20_M1_TermDuringBootPoisonsHeartbeatWithoutBaseline
      review_r20_test.go:230: M-1: canceled Boot returned nil without installing Baseline (baselines=4) and overwrote heartbeat with Serving=false BaselineOK=false Reason="safety_check_failed"
  --- FAIL: TestR20_M1_TermDuringBootPoisonsHeartbeatWithoutBaseline (0.11s)
  ```
- **Fix (owner: `agent-core`):** Give `Engine.Boot` a bounded local completion context (`newLocalBudget(ctx).localContext(ctx, localApplyBudget)`) for `e.plane.Baseline` (and `loadRuntime`), or when `ctx.Err() != nil` prevented `adoptStoredServing` from completing while the retained serving heartbeat was still valid, do not overwrite the valid heartbeat with `BaselineOK: false` without first attempting `Baseline` under a bounded `WithoutCancel` context.

---

#### `M-2` — `adoptStoredServing` runs `sameRestartSets` and `installed.NowMono` checks after `nodelock.Acquire` and `e.evaluate(ctx)`, extending the live-forwarding window of old deny/host sets during `Boot` and causing `e.updateLockHeld()` to self-contend (`D-064` / `AT13`)

- **Where:** `node/agent/restart.go:34-70`, `node/agent/probeexc.go:56-68`.
- **Scenario:**
  1. In `adoptStoredServing` (`restart.go:34-66`), `prelim := Decide(in)` at line 37 already constructs `prelim.Model` containing `DeniedV4`, `ExtraDeniedV4`, and `HostV4`, and `installed` (`e.appliedModel`) is already in memory with `installed.NowMono` and `installed.Now`.
  2. However, `sameRestartSets(*installed, d.Model)` and the `installed.NowMono` check are placed at lines 64-70, *after* `nodelock.Acquire` (line 42, up to $2.01\text{ s}$) and `e.evaluate(ctx)` (line 53).
  3. During `Boot(ctx)`, there is no `controlBudgetKey` deadline on `ctx`. `e.evaluate(ctx)` runs `refreshProbe` (which dials TCP probe targets with 2 s timeout per target) and `applyResolver` (which performs an HTTP fetch to `ResolverURL` with a 30 s client timeout). Throughout that entire probe/resolver duration, the old kernel ruleset with the smaller deny/host sets remains active and forwarding packets in the kernel before `sameRestartSets` finally refuses adoption and `Boot` installs `Baseline`.
  4. Furthermore, calling `e.evaluate(ctx)` while holding `lock` (`nodelock.Acquire` on `e.cfg.LockPath` at line 42) causes `e.updateLockHeld()` (`probeexc.go:56`, called from `evaluate` $\to$ `refreshProbe` $\to$ `loadProbe` when a `health-challenge.json` is present) to call `nodelock.TryAcquire(e.cfg.LockPath)`, which opens a new file descriptor and hits `EWOULDBLOCK` against `adoptStoredServing`'s own `flock`, falsely reporting that the updater holds `update.lock` (`D-064`).
- **Proof (failing unit test):** `TestR20_M2_SameRestartSetsRunsAfterProbeAndSelfLocksD064` in `node/agent/review_r20_test.go`.
  ```text
  === RUN   TestR20_M2_SameRestartSetsRunsAfterProbeAndSelfLocksD064
      review_r20_test.go:277: M-2: adoptStoredServing ran live Prober 1 time(s) while old deny sets were still active before checking sameRestartSets
  --- FAIL: TestR20_M2_SameRestartSetsRunsAfterProbeAndSelfLocksD064 (0.11s)
  ```
- **Fix (owner: `agent-core`):** Check `sameRestartSets(*installed, prelim.Model)` and `installed.NowMono <= 0 || in.NowMono < installed.NowMono || in.WallNow.Before(installed.Now)` immediately after `prelim := Decide(in)` at `restart.go:37` (before `nodelock.Acquire` and `e.evaluate(ctx)`), and track self-ownership of `e.cfg.LockPath` (or run `e.evaluate(ctx)` before `nodelock.Acquire`) so `e.updateLockHeld()` cannot self-contend on the agent's own lock.

---

#### `M-3` — Failed verification in `adoptStoredServing` latches `e.applyError` across `Boot`'s baseline, blocking ACKs on subsequent `unpublished` / `draining` transitions, and `shutdownUsage` `handoff` leaves `Serving: true` in the on-disk heartbeat (`AT13`)

- **Where:** `node/agent/restart.go:72-86`, `node/agent/serving.go:45-48, 127-156`, `node/agent/engine.go:446-457, 971-994, 1228`, `node/agent/usage.go:417-419`.
- **Scenario:**
  1. When `Close()` runs `shutdownUsage(ctx)` (`usage.go:404-420`) and `e.preserveServingOnStop(ctx)` returns `false` (for example because the lease or policy deadline expired right before shutdown, or a barrier arrived), line 418 runs `e.handoff(ctx)`, which installs `Baseline` (`fvpn_generation=0`) and sets `e.serving = false`, but never calls `e.beat()`. The on-disk `HeartbeatPath` retains `Serving: true, BaselineOK: true`.
  2. When the next process starts on the same boot (or whenever the kernel ruleset drifted before `Boot`) and enters `adoptStoredServing`, `verifyServingModel` (`restart.go:72`) fails with `GENERATION_MISMATCH` (`fvpn_generation=0 != installed.Generation`) and calls `e.rejectServing(err)` (`serving.go:45-48, 127-146`), which latches `e.applyError = &applyFailure{Code: "GENERATION_MISMATCH", Stage: "apply"}`. (Note also that inside `rejectServing`, `e.kernel` is still `nil` at `restart.go:72` and `e.expectServing` is `false` until line 90, so `e.kernel.rejectedGeneration()` is skipped.)
  3. `adoptStoredServing` returns `false`, and `Boot` (`engine.go:446-452`) installs a verified `Baseline` (`e.baselineOK = true`), but does not clear non-quarantine `e.applyError`.
  4. If the node's desired state is or becomes non-serving (`PublicationIntent = unpublished` or `draining`), `enforce` (`engine.go:971-994`) takes the `!d.Serve` branch, installs `Baseline`, and never clears `e.applyError` (which is only cleared at `engine.go:1139` after a serving `Apply` + `MemberPath`). Consequently, `e.ackApplied(d)` (`engine.go:1228`, which requires `e.applyError == nil`) refuses to advance `e.appliedGen` to the new `unpublished`/`draining` generation, and `Decide` permanently reports `SafetyStatusFailed` (`ISOLATION_UNVERIFIED` / `GENERATION_MISMATCH`).
- **Proof (failing unit test):** `TestR20_M3_FailedAdoptionLatchesApplyErrorAndBlocksUnpublishedAck` in `node/agent/review_r20_test.go`.
  ```text
  === RUN   TestR20_M3_FailedAdoptionLatchesApplyErrorAndBlocksUnpublishedAck
      review_r20_test.go:348: M-3: stale adoption applyError=&{GENERATION_MISMATCH GENERATION_MISMATCH 2026-11-03 05:13:31.998842884 +0000 UTC apply} blocked unpublished generation ACK: appliedGen=2 want=3 safety=failed reasons=[safety_check_failed not_published]
  --- FAIL: TestR20_M3_FailedAdoptionLatchesApplyErrorAndBlocksUnpublishedAck (0.14s)
  ```
- **Fix (owner: `agent-core`):**
  1. In `shutdownUsage` (`node/agent/usage.go:418`), call `e.beat()` after `e.handoff(ctx)` withdraws serving so the on-disk heartbeat records `Serving: false`.
  2. When `adoptStoredServing` fails verification (`restart.go:72-78`) and `Boot` subsequently succeeds in installing the clean fail-closed `Baseline` (with `!e.quarantinePending`), do not leave a stale adoption `e.applyError` latched that blocks non-serving (`unpublished`/`draining`) baseline ACKs (or clear non-quarantine `e.applyError` once `!d.Serve` baseline verification succeeds).

---

### Low

#### `L-1` — `sameRestartSets` accepts non-nil empty `DeniedV4: []netip.Prefix{}`, and `adoptStoredServing` does not check `installed.Generation == e.appliedGen`

- **Where:** `node/agent/restart.go:19-70, 97-108`, `node/enforce/model.go:240-247`, `node/agent/engine.go:1135-1165`.
- **Scenario:**
  1. `enforce.Model.denyV4()` (`node/enforce/model.go:240-247`) treats `DeniedV4 == nil` as `SpecDeniedV4()` and explicitly rejects a non-nil empty slice (`len(m.DeniedV4) == 0` $\to$ `"deny prefix list is empty"`). However, `sameRestartSets` (`restart.go:97-108`) only replaces `a.DeniedV4 == nil` / `b.DeniedV4 == nil` with `enforce.SpecDeniedV4()`. If both `a.DeniedV4` and `b.DeniedV4` are non-nil empty slices `[]netip.Prefix{}`, `samePrefixSet` returns `true` (`len(set) == 0`).
  2. In `Engine.enforce` (`node/agent/engine.go:1135-1165`), `retained_applied` is persisted at line 1137 *before* `checkMemberPath` (line 1150) and `ackApplied` (line 1161, which persists `applied_snapshot`). If `fvpn-agent` crashes between line 1137 and line 1161, `installed.Generation` (`retained_applied`) is ahead of `e.appliedGen` (`applied_snapshot`). `adoptStoredServing` checks `d.Model.Generation != installed.Generation` (line 61), which happens to match the new desired generation, and adopts serving while `e.appliedGen` and `e.appliedSnapshot` still reflect the older generation until the next `Step`.
- **Proof (failing unit test):** `TestR20_L1_SameRestartSetsEmptyNonNilDeniedV4AndGenerationMismatch` in `node/agent/review_r20_test.go`.
  ```text
  === RUN   TestR20_L1_SameRestartSetsEmptyNonNilDeniedV4AndGenerationMismatch
      review_r20_test.go:361: L-1: sameRestartSets accepted non-nil empty DeniedV4=[]netip.Prefix{} (which enforce.Model.denyV4 rejects)
  --- FAIL: TestR20_L1_SameRestartSetsEmptyNonNilDeniedV4AndGenerationMismatch (0.00s)
  ```
- **Fix (owner: `agent-core`):** In `sameRestartSets` (`restart.go:97-108`), return `false` when `(a.DeniedV4 != nil && len(a.DeniedV4) == 0) || (b.DeniedV4 != nil && len(b.DeniedV4) == 0)`, and in `adoptStoredServing` (`restart.go:23`), require `e.appliedGen == installed.Generation` (or write `retained_applied` only after `checkMemberPath` succeeds alongside `ackApplied`).

---

## 2. Merge recommendation

**Merge after named fixes (`H-1`, `M-1`, `M-2`, `M-3`, `L-1` in `agent-core`), and route `H-2` to `netlab 27`:** agent-core 36's core set-comparison and signal-lifetime mechanisms work as designed for unchanged/changed restart sets and repeated `SIGTERM`s during `Close`, but refusing adoption currently leaves `e.meterSnap` nil and drops pending same-boot member bytes into a `mapping conflict` gap (`H-1`), and `SIGTERM` during `Boot` (`M-1`), pre-comparison probe execution (`M-2`), and latched adoption `applyError` (`M-3`) should be fixed before merge.

---

## 3. What held

### Q1 — Is the comparison complete?

Every field of `enforce.Model` and of the retained, desired, and lease structures reused by `adoptStoredServing` falls into one of three categories:

| Structure / Field | Category | How adoption handles a change |
|---|---|---|
| `Model.DeniedV4`, `Model.ExtraDeniedV4`, `Model.HostV4` | **(a) Compared before adoption** | `sameRestartSets(*installed, d.Model)` (`restart.go:64`) compares all three as sets of masked IPv4 prefixes before adoption. |
| `Model.Generation` | **(a) Compared + (b) Verified** | `d.Model.Generation != installed.Generation` checked at `restart.go:61`; `verifyGeneration` (`verify.go:54`) checks `fvpn_generation=<G>` in kernel `forward` chain. |
| `Model.IPv6Mode` | **(b) Rebuilt + verified** | `Decide` sets `IPv6Block`; `validateServing` (`validate.go:19`) and `verifyForward` (`verify.go:476`) require `IPv6Block` and unconditional IPv6 drop rules before any accept. Neither `enforce.Model` nor `Config` carries IPv6 deny/host exception sets (`IPv6Block` drops all member IPv6 traffic). |
| `Model.Datapath` (`MemberIngress`, `Uplink`, `SNAT`) | **(b) Rebuilt + verified** | Rebuilt from `e.cfg.DatapathValue()` in `Decide`; `verifyForward`, `verifyInputDrops`, `verifyPostrouting`, `VerifyQoS`, and `checkMemberPath` (`serving.go:53`) verify interface names, SNAT address (`render_nft.go:54` also adds `SNAT/32` to `@host_v4`, see `H-2`), and link identity (`Ifindex`, `LinkNetnsID`). |
| `Model.Resolvers` (`resolver_v4`) | **(b) Rebuilt + verified** | Rebuilt in `evaluate` (`applyResolver`) and `Decide`; `verifyResolverSet` (`verify.go:124-144`) verifies exact set equality between `rs.Elements["resolver_v4"]` and `m.Resolvers`. |
| `Model.Bindings` (`MemberID`, `DeviceID`, `PeerID`, `AllowedV4`, `BindingRevision`, `ValidFrom`, `ValidUntil`) | **(b) Rebuilt + verified** (except `BindingRevision` on unchanged `ConfigGeneration`) | Rebuilt in `Decide` from stored `Desired` + `peerCheck`; `verifyForward` and `verifyAllowedElements` verify every active device's `/32`, counter names `u_<dev>`/`d_<dev>`/`nu_<dev>`/`nd_<dev>`, and `@allowed_v4` membership; `verifyAllowedLifetimes` (`verify.go:99`) verifies kernel `expires <= TimeoutFor + 5s`. (`BindingRevision` is not encoded in nft rules, which is safe because `ConfigGeneration` changes whenever `Desired` changes; if adoption is refused, see `H-1`.) |
| `Model.Leases` (`EffectiveServiceEnd`, `NodeLeaseEnd`, `SignedServerTime`, `MaxOfflineSeconds`, `TimeoutCeilSeconds`) | **(b) Rebuilt + verified** | Rebuilt in `Decide` from stored `Lease`; `restart.go:66-70` verifies `installed.NowMono` and `d.PolicyExpires > in.NowMono`; `verifyAllowedLifetimes` (`verify.go:99`) verifies no `@allowed_v4` element outlives the remaining lease ceiling. |
| `Model.QoS` (`Up`, `Down`, `HostUp`, `HostDown`, `Members`, `Assignments`) | **(b) Rebuilt + verified** | Rebuilt in `Decide`; `e.assignments` restored from `retained_applied` (`engine.go:654`); `verifyPriorities` (`verify.go:63`) and `VerifyQoS` (`verify_tc.go:25`) verify every HTB class rate/ceil and `fq_codel` leaf against `d.Model` and `e.assignments`. |
| `Model.Probe` (`ProbeException`) | **(b) Rebuilt + verified** | Rebuilt in `evaluate` (`refreshProbe`); `verifyForward`, `verifyProbeSets`, and `verifyPostrouting` reject any mismatch between `d.Model.Probe` and the kernel's `@probe_v4`/`@probe_dst_v4` sets and rules. |
| `Model.Revoke` | **(c) Neither** | `d.Model.Revoke` is always `nil` in `Decide` (populated only inside `enforce` from `removedAddresses(e.appliedModel, d.Model)`). Any crash before conntrack deletion completed is persisted in `pending_revocations` / `conntrack_blocked`, which sets `in.ConntrackBlocked = true` (`engine.go:635`) so `Decide` returns `Serve: false` and `adoptStoredServing` refuses adoption. |

- **Proof tests:**
  - `TestR20_Held_VerifyServingCatchesResolverBindingQoSAndSNATDrift` (`node/enforce/review_r20_test.go`) — PASS
  - `TestAT13_F36_RestartSetChanges` (`node/agent/followup36_test.go`) — PASS

### Q2 — Set semantics

- `samePrefixSet` (`node/agent/restart.go:110-132`) rejects `!p.IsValid() || !p.Addr().Is4()` (returning `false` on zero-value `netip.Prefix{}`, IPv6 prefixes, and IPv4-mapped IPv6 addresses `::ffff:192.0.2.1/128`), normalizes non-canonical host bits via `p.Masked()`, and compares sets via `map[netip.Prefix]struct{}`.
- `nil` vs empty `[]netip.Prefix{}` for `ExtraDeniedV4` and `HostV4` both produce an empty map and compare equal; `nil` `DeniedV4` is replaced with `enforce.SpecDeniedV4()` and compares equal to an explicit `SpecDeniedV4()` slice (for non-nil empty `DeniedV4: []netip.Prefix{}`, see `L-1`).
- Two models with identical `map[netip.Prefix]struct{}` always render to the same kernel set because `enforce.CanonicalPrefixes` (`node/enforce/prefixes.go:47`) performs the same `p.Masked()` + deduplication + sort before `Render`.
- Two models with split/merged or overlapping prefixes (e.g. `203.0.113.0/24` vs `203.0.113.0/25` + `203.0.113.128/25`) compare unequal (`false`) in `samePrefixSet` even though nft `auto-merge` merges them into the same kernel interval. This is fail-closed (causes a brief baseline + re-apply on config rewrite, availability cost only) and unlikely in practice unless an operator rewrites `agent.json` into split CIDRs.
- **Proof test:** `TestR20_Held_SameRestartSetsPrefixSemantics` (`node/agent/review_r20_test.go`) — PASS.

### Q3 — What the comparison trusts

- When `retained_applied` is missing (`""`), corrupt JSON, has `Generation == 0`, or comes from a previous boot (`boot_id` mismatch) or a different incarnation (`identityFrozen`), `Boot` fails closed to `Baseline` and refuses adoption.
- If a crash occurred during a previous `Apply` after `reserveConntrack` (`engine.go:1082`) when bindings were removed, `conntrack_attempts` (`1`) $\neq$ `conntrack_ack` (`0`) latches `e.conntrackBlocked = true` on `loadRuntime` (`engine.go:635`), forcing `Decide` to return `Serve: false` and refusing adoption.
- Until `netlab 27` lands, live kernel drift in `@denied_v4` / `@host_v4` is not visible to `VerifyServing` or `VerifyBaseline`; the exact remaining windows and the four requirements `netlab 27` must enforce are documented in finding `H-2`.
- **Proof tests:**
  - `TestR20_Held_CorruptOrRebootedRetainedAppliedFailsClosed` (`node/agent/review_r20_test.go`) — PASS
  - `TestR20_H2_VerifierIgnoresEmptiedDenyAndHostSets` (`node/enforce/review_r20_test.go`) — FAIL (proving `H-2`)

### Q4 — Ordering and races inside `adoptStoredServing` and `Boot`

- When `adoptStoredServing` returns `false`, `Boot` (`engine.go:446-456`) never sets `e.serving = true`, never sends an observation or ACK to central (`syncDesired`, `syncLease`, and `sendObservation` run only in `Step`, never in `Boot`), and writes a non-serving heartbeat via `e.beat()`.
- For the ordering of `nodelock.Acquire` and `e.evaluate(ctx)` before `sameRestartSets`, and the latching of `e.applyError` on failed `verifyServingModel`, see findings `M-2` and `M-3`.

### Q5 — Signals

- Holding `signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)` through `eng.Close()` (`main.go:98-120`) prevents a second or third `SIGTERM` or `SIGINT` during `Close()`'s SQLite/usage checkpoint from terminating the process early with signal 15.
- Supervisor escalation remains intact: `Engine.Step` bounds local completion at $\le 10\text{ s}$ after `SIGTERM`, `shutdownUsage` bounds its stop budget at $13\text{ s}$ (`stopTotalBudget`), and `deploy/node/systemd/fvpn-agent.service` + `packaging/installer/units.go` set `TimeoutStopSec=15s`, after which systemd sends `SIGKILL` (which cannot be caught).
- `SIGHUP` is not intercepted by `NotifyContext`, so its default action (terminate) remains unchanged.
- Exit codes match `Restart=on-failure` (`deploy/node/systemd/fvpn-agent.service:17`) and `stopobserve` (`tests/e2e/at/updates/stopobserve/receipt.go`): a `SIGTERM`-initiated stop of `fvpn-agent run` exits `0` (`Result=success`, `ExecMainCode=exited`, `ExecMainStatus=0`), while `fvpn-agent step` exits `1` on cancellation. (For `SIGTERM` arriving during `Boot`, see finding `M-1`.)
- `runWithOpen` is unexported (`package main`), and `run(args)` in `main.go:94-96` hardcodes `open`, so production cannot reach a test opener.
- **Proof tests:**
  - `TestR20_Held_RepeatedSignalsDuringCloseAndStepExitCodes` (`node/agent/cmd/fvpn-agent/review_r20_test.go`) — PASS
  - `TestAT13_F36_SignalLifetime` (`node/agent/cmd/fvpn-agent/signal_test.go`) — PASS

### Q6 — Durability proof soundness

- In `TestAT13_F36_InterruptedCloseDurability` (`node/agent/followup36_durable_test.go`), the child subprocess runs the real `r.eng.Close()` path (`shutdownUsage` $\to$ `preserveServingOnStop` $\to$ `e.meter.Sample`), blocked inside `f36StopSource.Read` before the SQLite transaction commits, and is killed by a real `SIGTERM` (`WaitStatus.Signal() == syscall.SIGTERM`). It proves that interrupting `Close()` before the sample commits does not tear `agent.db`, `usage.db`, or `HeartbeatPath` and does not fabricate `MeterCheckpoint`.
- Across two consecutive crashes on the same boot where adoption succeeds (`sameRestartSets == true`), the open epoch in `usage.db` retains the last committed counter values (`4096`), and the third process's sample attributes the cumulative delta (`8192 - 4096 = 4096`) once (`total == 8192`).
- However, lines 174-186 of `TestAT13_F36_InterruptedCloseDurability` bypassed `Engine.Boot` when testing recovery and passed `before.Observation.Bindings` directly to `meter.New`, which hid finding `H-1` on the refused-adoption path.
- **Proof tests:**
  - `TestR20_Held_TwoCrashesAndRebootCounterRecovery` (`node/agent/review_r20_test.go`) — PASS
  - `TestR20_H1_RefusedAdoptionDropsPendingUsageDelta` (`node/agent/review_r20_test.go`) — FAIL (proving `H-1`)

---

## 4. Not examined

- Privileged Docker network lab execution of `tests/netlab/agent/scenario_f36_test.go` (`restart-sets.py`): reviewed statically; unit-level proofs in `node/enforce`, `node/agent`, and `node/agent/cmd/fvpn-agent` were executed directly.
- Same-host A,B,A,B timing runs on D087/D104 (explicitly out of scope per task prompt).
- Central API (`apps/control`), worker (`apps/worker`), portal (`apps/portal`), and installer/updater packages beyond their stop/signal contracts (`TimeoutStopSec=15s`, `stopobserve`).
