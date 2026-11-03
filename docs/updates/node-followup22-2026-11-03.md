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
