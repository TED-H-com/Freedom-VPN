# Updater follow-up 22 — 2026-11-03

Base: `3d25307d` (main through #413). Scope: fix F21-01 in the test
fixtures by its mechanism, then re-measure the fixed parent test,
`TestAT25_FileMeterWaitRestartRechecksCurrentFile`, under the revised
foreign-CPU quiet rule. No production change. Requirement scope:
UPD03/UPD05, MTR03, §13.5–§13.9, §19 AT25 at component level. There is no
assembled acceptance claim and no physical arm64 claim.

Follow-up 21 (#413) stays as recorded:
[doc](node-followup21-2026-11-03.md) and
[timing record](node-followup21-timing-2026-11-03.json). By its declared
rule, F27-02's fixture-defect reading **failed**, because the corrected
parent test failed once (`replaced`, `meter_checkpoint_timeout`). Follow-up
20's runs and netlab 27's F27-02 also stay as recorded. None of these files
is edited here.

## 1. F21-01 from the code

### How the budget is set and spent

- `stepBlock` calls `prepareMeterWait` (`meter_barrier.go`). That writes the
  wait note with `Started = time.Now()` and `Deadline = Started + Wait`,
  then publishes the gate latch. A note that already exists is never
  rewritten, so a restart cannot get a new budget.
- `waitMeter` validates the note in this order:
  1. the note itself (boot ID, identity, bounds), else `meter_wait_invalid`;
  2. then, on every poll, the latch, else `meter_barrier_withdrawn`;
  3. then the deadline, else the current `reason`, which starts as
     `meter_checkpoint_timeout`;
  4. only then a checkpoint read. A barrier-ID mismatch sets `reason` to
     `meter_checkpoint_uncorrelated`, and the loop keeps polling until the
     deadline.
- Therefore **a detail of `meter_checkpoint_uncorrelated`, or a success,
  needs at least one checkpoint read before the persisted deadline.** If
  the deadline has already passed when `waitMeter` first runs, the result
  is `meter_checkpoint_timeout`. That is correct, fail-closed product
  behaviour.
- In every affected test, that deadline ran on the same wall clock as the
  test's own work done after the block.

### The parent test, per subcase (before this fix)

Between writing the wait note (inside `ApplyPending`) and calling `Recover`,
every subcase does the same thing:

- the blocked-stage hook writes the checkpoint with fsync, reads it once,
  and returns `ErrCrashed`;
- the test reads `ActiveTxn` and the note;
- it closes the engine and reopens the SQLite journal;
- then it does its subcase setup.

After that, `Recover` does its own lock and journal work before
`waitMeter`.

| Subcase | Subcase setup | Expected | Does the expected result depend on the time from note to `Recover`? |
|---|---|---|---|
| `missing` | remove the checkpoint | `meter_checkpoint_timeout` | **No.** It times out either way. Only `Recover`'s duration changes (up to 250 ms, or nearly 0 if the deadline is already gone). |
| `replaced` | `agentBarrierBytes` (a new SQLite usage store, meter, barrier, final sample, read-back), then an fsync write | `meter_checkpoint_uncorrelated` | **Yes. This is F21-01.** The first checkpoint read must happen before the deadline. Follow-up 21's failure: the setup took about 533 ms and the deadline had passed 283.1 ms before `Recover` started. |
| `withdrawn` | `Unblock` (fsync) | `meter_barrier_withdrawn` | **The result does not, but what the case proves does.** The latch is checked before the deadline, so withdrawal is reported either way. When setup outlasts 250 ms, the case only shows "withdrawn after expiry", not withdrawal inside the budget. Follow-up 21 saw this twice: −991.5 ms and −772.5 ms. |
| `new_boot` | change the boot ID | `meter_wait_invalid` | **No.** Note validation comes before any deadline check. |
| `expired` | sleep until deadline + 1 ms | `meter_checkpoint_timeout` | **No.** The deadline has passed either way, on purpose. |

### Every other `node/updater` test with a short meter budget

| Test | Budget | Test's own work between the block and the engine's read | Can it consume the budget? | Action |
|---|---|---|---|---|
| `followup17_test.go` `correlatedApply` (`TestD125_Rule*`): `expired`, `foreign`, `malformed`, `legacy`, `spelling` | 250 ms | The blocked-stage hook runs `closeMeter()`: the real meter's barrier and final sample, the SQLite usage store's closed checkpoint and unacked read. It then publishes the checkpoint with fsync (`expired` does not publish). The engine reads after the hook returns. | **Yes.** `foreign`, `malformed`, `legacy` and `spelling` expect `meter_checkpoint_uncorrelated`, so they need a read before the deadline. This is the class that failed `TestD125_RuleMalformedIdentity` on #411's merge CI (`meter_checkpoint_timeout`). `expired` times out either way. | fixed |
| the same, `fresh`, `late`, `republished`, `restart`, `sampled_restart` | 2 s | The same hook, plus: `late`'s goroutine (sleeps 150 ms, then publishes); the restart cases close the engine and reopen SQLite; in `sampled_restart`, `stepStop` reads again on recovery | **Yes.** They expect success, so they need a read before the 2 s deadline. A stall of 2 s or more fails them. | fixed |
| the same, `withdrawn`, `sampled_withdrawn`, `boot` | 2 s | as above | The result does not depend on it: the latch check and note validation come before the deadline. | fixed with the rest (one function) |
| `TestAT25_FileMeterWaitFailsClosedWithoutCorrelation` (6 scenarios) | 250 ms | The hook, and for `late_within_bound`, `withdrawn` and `withdrawn_reblocked` a goroutine. It sleeps 25 ms, then publishes, or unblocks (and reblocks), with fsync. | **Yes.** `older_boot_epoch` and `late_within_bound` expect `uncorrelated`, so they need a read before the deadline: before the poll at 200 ms for `late`. `withdrawn*` expect `withdrawn`, so the unblock must land before the deadline. `missing` and `after_bound` time out either way. | fixed |
| `TestAT25_FileMeterWaitRestartKeepsDeadline` | 250 ms | none; it already runs in `synctest` (follow-up 20) | No | none |
| `TestAT25_FileMeterSubprocessCrashAndSealedReplay` | 500 ms | Between the crashed child and the recovery child: `agentBarrierBytes` (SQLite) and a process spawn | It can consume the budget. **But the result does not depend on it:** the test accepts any meter-wait failure (`meterWaitFailed`), so either `timeout` or `uncorrelated` passes. A bubble cannot span processes. | none |
| `TestAT25_FileMeterDoesNotTrustLegacyFinalSampled` | 1 ms | — | No. No wait note exists, so the result is `meter_wait_invalid` before any deadline check. | none |
| `host_test.go` configuration (17 ms) | 17 ms | — | No. `Recover` has no transaction, so there is no wait. | none |

**This is a fixture defect, not a product defect.** In every case, the
product did what §13.6 asks: a read that was not before the persisted
deadline does not authorise anything, and the result is fail-closed. Only
the tests' expected details assumed that their own wall-clock work was
free.

## 2. The fix (commit `28821e09`; test files only)

- The parent test, `TestAT25_FileMeterWaitFailsClosedWithoutCorrelation`
  and `correlatedApply` now run each case inside `synctest.Test`. The world
  is built inside the bubble by a new helper, `bubbleWorld`.
- **Why this removes the dependency.** In a bubble, time advances only
  when every goroutine in the bubble is durably blocked. Disk I/O, fsync,
  SQLite, process scheduling and a goroutine waiting in a syscall are not
  durable blocks. So the test's setup takes real time but **none** of the
  budget. Only the engine's own timers advance the clock: the poll timer
  and the `WithTimeout` context. So do the tests' deliberate sleeps:
  `late`'s 150 ms, the 25 ms publishers, and `expired`'s sleep past the
  deadline.
- **`bubbleWorld` closes the loopback TLS server.** An idle `Accept` is
  not durably blocking, so it would stop the bubble clock for good. These
  tests stage the release locally and make no HTTP request. If one ever
  did, it would fail rather than hang.
- **Unchanged:**
  - the persisted budgets: 250 ms, 2 s and the one-minute restart
    configuration;
  - every assertion on `Recover`'s or `ApplyPending`'s result, error,
    journal detail, receipt (null usage), forwarding (still blocked),
    service stop/start and producer evidence;
  - the product.
  - `expired` still publishes only after the bounded wait ends. In the
    parent test it still sleeps past the deadline, now on the bubble clock.
- **Added to the parent test:**
  - **A fixture guard:** `Recover` must start with exactly the remaining
    budget the subcase tests: 250 ms, or −1 ms for `expired`. So
    `withdrawn` now always exercises the in-budget path, and any future
    setup that spends budget fails at once with a clear message.
  - **A bubble-clock bound:** `Recover` must never wait longer than the
    original remaining budget. A renewal mutation (control C3 below) now
    fails `missing`, `replaced` and `expired` deterministically, in about
    0.1 s of wall time.
  - **Logging:** `Recover`'s real wall time is still logged, read from
    `CLOCK_MONOTONIC` through `golang.org/x/sys/unix`, which is already a
    direct dependency. It keeps the `F20 recover_elapsed_ns=` key, now
    with `bubble_elapsed_ns=` beside it.
- **One semantic change outside `Recover`:** in
  `TestAT25_FileMeterWaitFailsClosedWithoutCorrelation`, the existing
  `time.Since(started) > 2*time.Second` bound ("wait exceeded bound") now
  reads the bubble clock. It bounds the engine's own waiting, which is
  what its message claims, instead of disk stalls. The threshold is
  unchanged.
- **No longer budget.** No subcase needs a longer persisted budget to test
  what it claims, so none was lengthened.
- **Side effect:** the package's tests are faster. `node/updater` took
  18.3 s, against 28.1 s in follow-up 21's gate, because the waits no
  longer run in real time.

## 3. Controls (run, then removed)

The controls are `-overlay` variants, never committed. They are built by
`.agent/tmp/f22/mkcontrol.py`: `current` from `main`'s pre-fix tests, and
`fixed` from the working tree. The injected delay is a blocking
`nanosleep` syscall in the test's own work. That is a wall-clock stall
which a bubble does not count, the same kind as follow-up 21's 533 ms
setup stall. No timer is involved.

- **C1 (parent):** 600 ms just before `Recover`; in the fixed test, before
  the guard.
- **C2 (`FailsClosedWithoutCorrelation`):** 600 ms inside the blocked-stage
  hook.
- **C3 (`correlatedApply`):** budget + 350 ms (600 ms or 2.35 s) inside the
  blocked-stage hook, before publication.
- **Diagnostic:** a control-only log of the journal's last detail, to name
  the failing detail where the test's own message does not.

| Test / case | Pre-fix + delay | Fixed + delay |
|---|---|---|
| parent `missing` | PASS (`timeout` expected) | PASS |
| parent `replaced` | **FAIL**: `ErrorDetail:meter_checkpoint_timeout` (remaining −369 to −601 ms at `Recover`) | PASS (remaining exactly 250 ms) |
| parent `withdrawn`, `new_boot`, `expired` | PASS (but `withdrawn` ran after expiry) | PASS (`withdrawn` in budget) |
| `FailsClosed` `older_boot_epoch`, `late_within_bound` | **FAIL** ×2: `meter_checkpoint_timeout` | PASS |
| `FailsClosed` `withdrawn`, `withdrawn_reblocked` | **FAIL** ×2: `meter_checkpoint_timeout` | PASS |
| `FailsClosed` `missing`, `after_bound` | PASS | PASS |
| D125 `foreign`, `malformed`, `legacy`, `spelling` | **FAIL** ×4: `meter_checkpoint_timeout` | PASS (`meter_checkpoint_uncorrelated`) |
| D125 `fresh`, `late`, `republished`, `restart` | **FAIL** ×4: `meter_checkpoint_timeout` (rolled back) | PASS (`locally_succeeded`) |
| D125 `sampled_restart`, `sampled_withdrawn` | **FAIL** ×2: `meter_checkpoint_timeout` before the final-sampled crash point ("did not stop mid-handshake") | PASS |
| D125 `expired`, `withdrawn`, `boot` | PASS | PASS |

- **Wall time:** in both runs the delayed cases took 0.67–0.80 s or
  2.4–2.7 s each, so the stall really ran.
- **Without delays:** the fixed tests pass, and the parent logs
  `original_remaining_ns=250000000` for 4 subcases and `-1000000` for
  `expired`. `bubble_elapsed_ns` is 250 ms for `missing` and `replaced`,
  which poll to the deadline, and 0 for the others.
- **Renewal mutation (C3′):** a product overlay that recomputes the
  deadline from the one-minute restart configuration on every wait.
  - It fails the fixed parent test: `missing`, `replaced` and `expired`
    report `Recover waited 1m0s on the bubble clock`.
  - It fails all 4 `KeepsDeadline` cases.
  - The restored product passes.
- Logs: `.agent/tmp/f22/control-*.log`.

After the controls, the tree has no control code: `git grep -n
f22ControlStall` finds nothing.

## 4. Declaration (committed before any measured run)

### Inputs, checked before this declaration (not measured legs)

All at 2026-11-03T10:56Z on `28821e09`, each after admission printed `free`.

- `go test -count=1 -run '^$' ./node/updater` and `./node/agent/...`
  built. That warmed the build cache, so no compile time lands inside a leg.
- `go test -v -count=50 -run '^TestAT25_FileMeterWaitRestartKeepsDeadline$'
  ./node/updater` passed. All 200 samples were exact: 0 s → 250 ms,
  200 ms → 50 ms, 250 ms → 0 s and 1 s → 0 s.
- One unmeasured run of the fixed parent took 0.74 s per iteration
  (0.11–0.20 s per subcase).
- Idle foreign CPU over 15 s was 0.08–0.10 cores.

### Legs, in order, in one reservation

`PARENT` is `'^TestAT25_FileMeterWaitRestartRechecksCurrentFile$'`. The
setup is follow-up 21's: the runner (`window.py`, adapted for the new rule
as `.agent/tmp/f22/window.py`) and the environment, `CGO_ENABLED=0
GOTOOLCHAIN=local GOMAXPROCS=4 GOFLAGS=-p=4`, with the worktree's
`.agent/tmp` as `TMPDIR` and `GOTMPDIR` and the cache from `go env
GOCACHE` (`~/.cache/go-build`).

| # | Leg | Processes | Watchdog |
|---|---|---|---:|
| 1 | quiet | `go test -v -run PARENT -count=200 ./node/updater` | 360 s |
| 2 | bounded load | `go test -count=1 ./node/agent/...` started, then immediately `go test -v -run PARENT -count=50 ./node/updater`, concurrently | 300 s |

- **Reservation:** `bash "$FVPN_HOLD_SH" --reserve 30 "updater f22 F21-01
  re-measure"`. Then wait 60 s, run the preflight, and run the legs back to
  back. Release with `bash "$FVPN_HOLD_SH" --release` as soon as the last
  leg ends.
- **Worst case:** 60 s + 2 × 35 s preflight + 360 s + 300 s = 13.2 min,
  which fits in 30 min.
- **Leg start rule:** a leg starts only if its whole watchdog plus a 30 s
  margin still fits in the reservation. Otherwise it is recorded as not
  started.

### Preflight before each leg (recorded with its wait time)

1. `bash "$FVPN_HOLD_SH"` prints `free`.
2. No other lane's suite process. The classifier is follow-up 21's
   corrected one:
   - **A suite process** has `go` as its executable with `test` as its
     subcommand, or an argument that ends in `gates.sh` or contains
     `mk/at-`.
   - **Excluded:** processes descended from the runner.
3. Foreign CPU is ≤ 0.5 cores, averaged over a 30 s sample. Its 5 s parts
   are recorded too.
4. `vmstat 5 7` runs at the same time. It is **recorded only** and no
   longer decides.
5. Steps 1 and 2 are repeated after the 30 s sample.

A failed preflight is retried every 30 s, for at most 300 s per leg. That
retries the preflight, not the leg.

### Foreign CPU (the revised quiet rule)

- **Definition:** foreign CPU is the `utime + stime` from
  `/proc/<pid>/stat` of every process **not** descended from the runner,
  divided by the interval's wall time. Our test processes, the agent load,
  admission calls and `vmstat` all descend from the runner, so they never
  count.
- **Sampling:** a snapshot of every process every 1 s. Per-process deltas,
  keyed by pid and start time, are summed into 5 s intervals.
  - A process first seen in a step counts with its whole `utime + stime`.
    This is fail-closed: it may overcount a process that started earlier.
  - **Known limit:** CPU a process uses after its last 1 s snapshot, before
    it exits, is not seen.
- **Recorded beside it, deciding nothing:** host busy CPU from `/proc/stat`
  minus our own CPU, which is an upper bound, and the top foreign command
  names.

### What qualifies a leg (both legs, the same rule)

- the preflight passed;
- at every in-leg sample, taken every 5 s (a gap over 7 s disqualifies),
  admission printed `free` and no other lane's suite process was present;
- **foreign CPU was ≤ 0.5 cores in every 5 s interval of the leg;**
- the leg ran to completion.

In-leg vmstat is recorded only. Test failures do not disqualify a leg:
they are the measurement. Every log is kept, whether it passed, failed or
was stopped.

### Stop rules

- **During a leg,** any of these ends it:
  - another lane's suite process appears;
  - a 5 s interval's foreign CPU exceeds 0.5 cores;
  - admission stops printing `free`.
- **Then:** both of our processes are terminated (process group SIGTERM,
  then SIGKILL after 10 s). The leg is recorded as **unqualified** and is
  not retried in this reservation. The next leg runs after its own
  preflight.
- **Watchdog or reservation end:** the leg is stopped and recorded as
  unqualified (incomplete) in the same way.
- **At most one more reservation,** for the missing leg only, taken at
  least 10 minutes after releasing the first. It uses the same commands
  and rules.

### Deterministic budget

- `TestAT25_FileMeterWaitRestartKeepsDeadline` runs at `-count=50` before
  the window (above) and after it. It must stay exactly 250/50/0/0 ms both
  times.
- In addition, the fixed parent test checks its own budget in every
  iteration of every leg:
  - the guard: `original_remaining_ns` is exactly 250000000 for `missing`,
    `replaced`, `withdrawn` and `new_boot`, and −1000000 for `expired`;
  - the bound: `bubble_elapsed_ns` is never above it.
  - Either violation would be a test failure.

### What each outcome shows (verdict rule fixed now)

This verdict is about the **fixed** test from §2. Follow-up 21's verdict on
follow-up 20's corrected test stays **failed**, as recorded.

- **The fixture reading holds:**
  - the fixed parent test does not fail in either leg;
  - both legs qualified;
  - the deterministic budget is exact before and after the window and in
    every parent sample.
- **What contradicts it, and makes the reading fail:**
  - any failure of the fixed parent test, in any leg, qualified or not;
  - any drift in the deterministic budget.
  - A failure's mechanism is reported the way follow-up 21 did it: detail,
    original remaining budget, bubble and wall elapsed time.
- **Undecided:** there is no failure, but a leg is still unqualified after
  the permitted reservations.

### What is reported per leg

- iterations and failures;
- `Recover`'s wall time per subcase: min, median, p99 and max, as
  nearest-rank quantiles;
- `bubble_elapsed_ns` and `original_remaining_ns`, distinct values;
- the qualification record, the preflights with their waits, and the
  foreign-CPU maximum.

## 5. Results

### Reservation 1 (11:15:57Z–11:20:19Z, 261 s held)

- **Admission:** reserved with `reserved for w2-updater until 06:45:57`
  (local), and released with `released w2-updater` right after leg 2 was
  stopped.
- **Wait for the window:** from 11:13:57Z. Admission printed `reserved for
  w1-netlab` until 11:15:57Z, which was the first check showing `free` with
  no `go test` on the host.

| Leg | Preflights (wait) | Iterations / failures | In-leg record | Qualification |
|---|---|---|---|---|
| quiet | 2 (90.3 s). The first failed on foreign CPU: a 30 s average of 0.678 cores, with 5 s parts 1.51, 1.39, 0.70, 0.21, 0.11, 0.12; no foreign suite. The second passed at 0.157 | **200 / 0** (64.8 s) | 12 samples, max gap 5.06 s, all `free`, no foreign suite. Foreign CPU in 13 intervals: max **0.345**, median 0.143. vmstat (recorded only): r ≤ 1, id ≥ 94 | **qualified** |
| bounded load | 1 (30.1 s, 0.146) | 33 completed / **0** failures, then stopped | 3 samples, all `free`, no foreign suite. At t = 15.1 s, a 5 s interval had foreign CPU **1.66** cores, almost all from a foreign `git` process (1.40 cores) | **unqualified, stopped** (`stopped_foreign_cpu`). Both processes were terminated at 15.2 s. Not retried in this reservation |

- **Nothing of ours made that CPU.** The runner, its tests and its load
  started no `git`; I committed before reserving, not during the window.
- **In-leg "non-suite" matches** were my own wait loop's shell (the
  runner's parent, idle). It is not a suite and used no CPU.

### Reservation 2 (11:30:45Z–11:33:16Z, 151 s held), load leg only

- **Timing and admission:** taken 10 min 26 s after the first release, at
  the first check, when admission printed `free` and no `go test` was
  running. Reserved with `reserved for w2-updater until 07:00:45` (local),
  and released with `released w2-updater`.
- **Rules:** the commands and rules are the declared ones. Nothing was
  changed between the reservations.

| Leg | Preflights (wait) | Iterations / failures | In-leg record | Qualification |
|---|---|---|---|---|
| bounded load | 1 (30.1 s, 0.166) | **50 / 0**. The updater process ran 0–24.0 s, exit 0, entirely inside the agent suite | 12 samples, max gap 5.06 s, all `free`, no foreign suite. Foreign CPU stayed at 0.08–0.37 cores in the 11 intervals up to t = 55.5 s, which include all of the updater process (max 0.335). At t = 60.5 s, with only the agent suite still running, one interval reached **1.80** cores. It came from non-lane host jobs: `_l1l4_rearm_ab.` 0.67, `run_l5_qwen.py` 0.46, `supervise.py` 0.20 | **unqualified, stopped** (`stopped_foreign_cpu`). The agent suite was terminated at 60.7 s, so its result is not recorded |

**Not relabelled.** The declared rule judges the leg, not the updater
process inside it. So the load leg stays unqualified, even though every
interval in which the updater test ran was below 0.5 cores. That
sub-record is reported as a fact only. The brief allows no further
reservation.

### After the window

At 11:33:37Z, `TestAT25_FileMeterWaitRestartKeepsDeadline -count=50` was
exact again in all 200 samples: 0 s → 250 ms, 200 ms → 50 ms,
250 ms → 0 s and 1 s → 0 s.

## 6. Per-leg results

`Recover`'s **wall time** per subcase, in milliseconds: min, median, p99
and max, as nearest-rank quantiles. With n ≤ 50, p99 is the maximum. "Q"
marks a qualified leg. Full records are in the
[timing record](node-followup22-timing-2026-11-03.json).

| Window / leg | Q | Iter. / fail | Foreign CPU max (cores) | missing | replaced | withdrawn | new_boot | expired |
|---|---|---|---:|---|---|---|---|---|
| r1 quiet | **yes** | 200 / 0 | 0.345 | 10.7 / 12.2 / 15.0 / 19.5 | 10.7 / 12.2 / 15.0 / 16.5 | 12.5 / 13.9 / 16.2 / 20.4 | 10.6 / 12.0 / 14.1 / 16.6 | 10.8 / 12.0 / 13.9 / 15.1 |
| r1 load (stopped at 15.2 s) | no | 33 / 0 | 1.66 | 11.5 / 18.7 / 23.0 / 23.0 | 11.1 / 18.9 / 21.5 / 21.5 | 13.8 / 21.9 / 25.1 / 25.1 | 11.1 / 18.7 / 22.4 / 22.4 | 11.0 / 18.5 / 21.1 / 21.1 |
| r2 load (stopped at 60.7 s) | no | 50 / 0 | 1.80 (0.335 while the updater ran) | 11.6 / 19.2 / 24.7 / 24.7 | 11.7 / 19.1 / 23.1 / 23.1 | 13.4 / 22.6 / 28.7 / 28.7 | 11.0 / 19.0 / 23.8 / 23.8 | 11.2 / 18.9 / 22.6 / 22.6 |

**Bubble clock, in every sample of every leg (283 iterations, 1,415
`Recover` calls):**

- `original_remaining` was exactly 250 ms for `missing`, `replaced`,
  `withdrawn` and `new_boot`, and −1 ms for `expired`. The setup consumed
  none of the budget in any iteration.
- `bubble_elapsed` was exactly 250 ms for `missing` and `replaced`, and 0
  for the others. No wait was ever longer than the original budget.

**How to read the wall times.**

- They are no longer comparable with follow-up 21's, whose `missing` and
  `replaced` medians were about 249–261 ms. The 250 ms meter wait now
  passes on the bubble clock in no wall time, so `Recover`'s wall time is
  only its own work: lock, journal, sealing and cleanup.
- That work took 10.6–28.7 ms here. The largest was 28.7 ms
  (`withdrawn`, r2 load), and no sample reached 30 ms.
- Load raised the median from about 12 ms to about 19 ms.

## 7. Verdicts

**F21-01 is fixed, in the test fixtures, by its mechanism.**

- The expected result of the parent test, of
  `TestAT25_FileMeterWaitFailsClosedWithoutCorrelation` and of
  `correlatedApply` no longer depends on how long the tests' own work
  takes:
  - controls C1–C3: the pre-fix tests failed with
    `meter_checkpoint_timeout`, and the fixed ones passed;
  - the guard held in every measured iteration;
  - none of the 283 measured iterations failed.
- The product is unchanged.
- The persisted budgets are unchanged.
- Every assertion on result, error, receipt and forwarding is unchanged.

**F27-02's fixture-defect reading, for the fixed test, stays UNDECIDED**
by the rule declared in §4:

- **No contradiction:** the fixed parent test did not fail in any leg
  (200/0 quiet, 33/0 and 50/0 under load). The deterministic budget was
  exact before and after the window, and in every parent sample.
- **But one leg is unqualified:** the quiet leg qualified. The bounded-load
  leg did not qualify in either permitted reservation. Both times it was
  stopped by foreign CPU from processes outside the lanes' suites: a
  foreign `git` at 1.40 cores in r1, and non-lane host jobs at 1.80 cores
  in r2.
- **So "held" is not reached.** By the rule, the reading neither fails nor
  holds.

**History stays.** By its own declared rule, follow-up 21's verdict on
follow-up 20's corrected test remains **failed**. Netlab 27's F27-02 and
follow-up 20's runs remain as recorded. This verdict is about the fixed
test only.

## 8. Known gaps and later work

- **The qualified load leg is still missing.** It needs a reservation that
  also keeps non-lane host jobs quiet, or an orchestrator decision; see §9.
- **The foreign-CPU sampler has a limit:** CPU a process uses after its
  last 1 s snapshot, before it exits, is not counted.
- **`TestAT25_FileMeterSubprocessCrashAndSealedReplay`** still spends its
  500 ms budget on the wall clock across processes. Its assertion accepts
  either meter-wait detail, so its result does not depend on that.
- **`replaced` in the parent test** writes a checkpoint without a
  `barrier_id`. The pre-restart checkpoint has none either, so `replaced`
  would also be `uncorrelated` without the replacement. That is a coverage
  observation, not part of F21-01, and it is unchanged here.
- **Not covered:** physical arm64 runs, power-loss durability and
  worst-case production load. The wall times above are not a `Recover`
  SLO.

## 9. Requests to orchestrator

- **2026-11-03 — load-leg qualification (scheduling, outside updater
  paths):** both bounded-load attempts were stopped by foreign CPU that
  was not another lane's test suite, and that admission and the reservation
  do not cover:
  - r1, 11:20:19Z: a `git` process at 1.40 cores;
  - r2, 11:33:16Z: `_l1l4_rearm_ab.`, `run_l5_qwen.py` and `supervise.py`,
    1.80 cores in total.
  - **Please decide one of these:**
    1. schedule one more load-leg reservation in which those host jobs
       are paused;
    2. or rule whether a breach after the updater process has exited
       should end a load leg. In r2 the updater ran 0–24.0 s at ≤ 0.335
       foreign cores.
  - Until then, F27-02's fixture-defect reading for the fixed test stays
    undecided.
