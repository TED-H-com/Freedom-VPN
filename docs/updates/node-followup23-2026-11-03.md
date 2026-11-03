# Updater follow-up 23 — 2026-11-03

Base: `f50050f3` (main through #417). Scope:

1. the one missing bounded-load leg for the fixed parent test,
   `TestAT25_FileMeterWaitRestartRechecksCurrentFile`, under a declared end
   rule;
2. a judgement of the other short wall-clock windows in `node/updater`'s
   fixtures (D-141: judge before fixing).

Requirement scope: UPD03/UPD05, MTR03, §13.5–§13.9 and §19 AT25 at component
level. There is no assembled acceptance claim and no physical arm64 claim.

**History stays as recorded.** None of these files is edited here:

- Follow-up 21 ([doc](node-followup21-2026-11-03.md),
  [timing](node-followup21-timing-2026-11-03.json)): by its declared rule,
  F27-02's fixture-defect reading **failed** for follow-up 20's corrected
  test (F21-01).
- Follow-up 22 ([doc](node-followup22-2026-11-03.md),
  [timing](node-followup22-timing-2026-11-03.json)): F21-01 fixed by its
  mechanism. For the fixed test, F27-02's reading stayed **undecided**:
  quiet leg qualified at 200/0, both bounded-load legs (33/0, 50/0) stopped
  by non-lane foreign CPU.
- Follow-up 20's runs and netlab 27's F27-02.

## 1. Declaration for the load leg (committed before any measured run)

### The leg

It is follow-up 22's bounded-load leg, unchanged:

| Process | Command |
|---|---|
| load (started first) | `go test -count=1 ./node/agent/...` |
| measured (started immediately after, concurrently) | `go test -v -run '^TestAT25_FileMeterWaitRestartRechecksCurrentFile$' -count=50 ./node/updater` |

- **Environment:** follow-up 22's: `CGO_ENABLED=0 GOTOOLCHAIN=local
  GOMAXPROCS=4 GOFLAGS=-p=4`, the worktree's `.agent/tmp` as `TMPDIR` and
  `GOTMPDIR`, and the cache from `go env GOCACHE`.
- **Runner:** follow-up 22's `window.py` and `lib.py`, copied to
  `.agent/tmp/f23/` (not committed), with only these changes:
  - the end rule below;
  - foreign CPU excludes kernel threads and the container runtime, as this
    follow-up's brief defines it (below);
  - a bounded host-quiet wait before reserving (below);
  - one leg per reservation.
- **The measured test process** is the updater's `go test` process that the
  runner starts. It exits after its test binary has exited and its output
  is written.

### The tree check

Before the window, the runner checks and records that
`node/updater/meter_barrier_test.go` and `node/updater/followup17_test.go`
are byte-identical to follow-up 22's delivered tip `5eea3f02`: same git blob
and same SHA-256. If either differs, item 1 stops, and what changed is
reported.

Checked when this declaration was written, on `f50050f3`:

| File | Blob at `5eea3f02` | Blob in the worktree |
|---|---|---|
| `meter_barrier_test.go` | `e1d911ad96b2` | `e1d911ad96b2` |
| `followup17_test.go` | `98b300dd22b3` | `98b300dd22b3` |

`git diff 5eea3f02 HEAD -- node pkg internal go.mod go.sum` is empty, so the
product and its dependencies are also unchanged since that tip. The runner
repeats the check just before the window.

### The end rule (the one change to follow-up 22's rule)

- **The leg ends when the measured test process exits.**
- After that, the runner keeps sampling for 30 s, exactly as during the leg:
  admission, the suite scan and foreign CPU. Everything is recorded.
  **Nothing in those 30 s disqualifies the leg.**
- The load process keeps running during those 30 s. At their end, if it is
  still running, it is terminated (process-group SIGTERM, then SIGKILL after
  10 s). The load is not the measurement, so its result is then not
  recorded. The reservation is released right after.
- **Which foreign-CPU intervals decide:** every 5 s interval that *starts*
  before the measured process exits. The interval in which it exits
  therefore decides, which is fail-closed. Intervals that start after the
  exit are post-exit and recorded only.

### Foreign CPU

- **Definition:** the `utime + stime` from `/proc/<pid>/stat` of every
  process not descended from the runner, divided by the interval's wall
  time. Our two `go test` processes, their children, admission calls and
  `vmstat` descend from the runner and never count.
- **Excluded, and recorded in their own fields:** kernel threads (pid 2,
  `kthreadd`, and its children) and the container runtime (`dockerd`,
  `containerd`, `containerd-shim*`, `docker-proxy`, `runc`). Follow-up 22's
  runner counted them as foreign, so this is the brief's definition, not
  follow-up 22's. It is stated here before the run, and both excluded
  totals are reported for every interval.
- **Sampling:** follow-up 22's. A snapshot of every process every 1 s;
  per-process deltas, keyed by pid and start time, summed into 5 s
  intervals. A process first seen in a step counts with its whole
  `utime + stime` (fail-closed). Known limit: CPU a process uses after its
  last snapshot, before it exits, is not seen.
- Each interval also records the top foreign commands and processes
  (command basenames only, no paths) and the host's busy CPU minus ours.

### Host-quiet wait, then preflight

1. **Before reserving: bounded checks.** Each check is admission, the suite
   scan and a 30 s foreign-CPU sample. A check that fails is repeated after
   60 s. Every check is recorded with what was running.
   - The total time spent waiting is capped at **60 minutes** for item 1,
     across both permitted reservations.
   - If the host never quiets in that time, item 1 stops and is reported.
     That is not a failure of the test.
2. **Reserve** with `bash "$FVPN_HOLD_SH" --reserve 30 "updater f23 load
   leg"`, right after a check passes. If the reservation is refused, no leg
   starts.
3. **Wait 60 s**, then **the preflight:**
   - admission prints `free`;
   - no other lane's suite process;
   - foreign CPU at or below 0.5 cores, averaged over a 30 s sample (its 5 s
     parts are recorded);
   - `vmstat 5 7` runs beside it and is recorded only;
   - admission and the scan are repeated after the sample.
   - A failed preflight is retried every 30 s for at most 300 s. That
     retries the preflight, never the leg. If it never passes, the leg is
     recorded as **not started**.
4. **Leg start rule:** the leg starts only if its preflight, the 300 s
   watchdog, the 30 s post-exit sampling and a 30 s margin still fit in the
   reservation.

A **suite process** is follow-up 22's classifier: `go` as the executable
with `test` as its subcommand, or an argument ending in `gates.sh` or
containing `mk/at-`. Processes descended from the runner are excluded.

### What qualifies the leg

All of these, judged up to the measured process's exit:

- the preflight passed, so admission printed `free` before the leg;
- at every in-leg sample, taken every 5 s, admission printed `free` and no
  other lane's suite process was present. A gap over 7 s between the start,
  the samples and the exit disqualifies;
- foreign CPU was at or below 0.5 cores in every deciding interval;
- the measured process exited on its own (it was not stopped).

vmstat is recorded and does not decide. A test failure does not disqualify:
it is the measurement.

### Stop rules (before the measured process exits only)

- Any of these ends the leg: another lane's suite process appears, an
  interval's foreign CPU exceeds 0.5 cores, admission stops printing
  `free`, the 300 s watchdog, or the reservation's end.
- Then both processes are terminated (process-group SIGTERM, then SIGKILL
  after 10 s), and the leg is recorded as **unqualified, stopped**.
- **One reservation, one leg, no retry in it.** At most one more
  reservation, taken at least 10 minutes after releasing the first, and only
  if the first leg was stopped before its test process exited. A leg that
  was recorded as not started also never reached its exit; it is treated
  the same way. The second reservation uses the same commands and rules.

### Pooling

Follow-up 22's qualified quiet leg (r1 quiet, 200/0, foreign CPU max
0.345 cores, same tree as checked above) is pooled with this leg, as D-138
allows. Together they make the two legs of follow-up 22's verdict rule. The
two unqualified load legs of follow-up 22 (33/0, 50/0) stay recorded as
they are and are not pooled as qualified.

### Deterministic budget

- `TestAT25_FileMeterWaitRestartKeepsDeadline -count=50` runs before the
  window and after it. It must be exactly 0 s → 250 ms, 200 ms → 50 ms,
  250 ms → 0 s and 1 s → 0 s in every sample, both times.
- The fixed parent test also checks its own budget in every iteration: the
  guard (`original_remaining_ns` exactly 250000000 for `missing`,
  `replaced`, `withdrawn` and `new_boot`, and −1000000 for `expired`) and
  the bound (`bubble_elapsed_ns` never above it). Either violation is a test
  failure.

### What each outcome shows (fixed now)

- **The fixture reading holds:** the leg qualifies, the fixed test does not
  fail, and the deterministic budget stays exact (before, after, and in
  every parent sample). With follow-up 22's qualified quiet leg, F27-02's
  fixture-defect reading for the fixed test then **holds**.
- **It fails:** any failure of the fixed test in this leg, qualified or not,
  or any drift of the deterministic budget. The mechanism is reported the
  way follow-up 21 did: detail, original remaining budget, bubble and wall
  elapsed time.
- **It stays undecided:** the leg does not qualify (in either permitted
  reservation), or item 1 stops for a host that never quiets, and nothing
  failed.

### What is reported

- iterations and failures;
- `Recover`'s wall time per subcase: min, median, p99 and max, as
  nearest-rank quantiles;
- `original_remaining_ns` and `bubble_elapsed_ns`, distinct values;
- the qualification record: every wait check, the preflights, the in-leg
  and post-exit samples;
- the foreign-CPU maximum before and after the measured process exited, and
  the excluded kernel and runtime CPU.

Every run is retained, failed or passed, under `.agent/tmp/f23/`, and
summarised in `node-followup23-timing-2026-11-03.json`.

## 2. Results and verdict from the retained records

**F27-02's fixture-defect reading for the fixed test HOLDS**, by §1's
precommitted rule: r2's bounded-load leg qualified at **50/0**, pooled with
follow-up 22's qualified quiet leg at **200/0**, and the deterministic
budget stayed exact. The unqualified r1 leg had no test failure. This
verdict concerns the fixed test; follow-up 21's failed verdict and
follow-up 22's undecided verdict remain as recorded.

### Completion across two sessions

The first session, **agy / claude-opus-5-5-high**, committed §1 as
`f770117d`, ran the input checks, runner self-test and both permitted
reservations, checked the post-window budget and tree, and drafted the
verdict, timing JSON and item 2. It also diagnosed F23-01, ran its controls,
and parked the patch so r2 used follow-up 22's exact package. Its model
quota ran out at **07:30 EST (12:30Z)** before it wrote its report.

The second session, **codex / gpt-6-astra**, completed this follow-up. It
read and checked the retained records, reran `summarize.py`, independently
recomputed qualification from the recorded intervals, applied the parked
patch unchanged, reran the controls after admission, ran the gates, and
finished this document, JSON and hand-off report. It took **no reservation
and ran no measured leg**. §1 is byte-for-byte unchanged.

### Record verification

The [timing record](node-followup23-timing-2026-11-03.json) is regenerated
from `r1-window.json`, `r2-window.json`, their updater logs, and the
pre/post deterministic logs. `post.json` confirms a successful admitted
post-window check and the unchanged tree. The completion session also
checked the raw interval partitions, maxima, admission/suite samples,
sample gaps and process exits instead of relying only on the stored
`qualified` flags. **No disagreement** with `verdict.json` or the retained
draft timing JSON was found. Original records, controls and logs are kept
unchanged under `.agent/tmp/f23/`; completion records are in its
`completion/` directory.

The two meter fixture files were byte-identical to `5eea3f02` before both
reservations and after r2; the SHA-256 checks in `inputs.json` and
`post.json` agree with the blobs and the current files. The parked item-2
patch was absent during r2. The completion changes only the separate
D-053 test file.

### Reservations and qualification

| Record | r1 | r2 |
|---|---|---|
| Reservation UTC | 12:11:13–12:12:53 | 12:26:11–12:28:34 |
| Held time | 100.310 s | 143.623 s |
| Preflight | `free`, no foreign suite, 30 s foreign-CPU average 0.076 cores | `free`, no foreign suite, 30 s average 0.077 cores |
| Measured process | stopped at 10.129 s, signal 15 | exited 0 at 23.385 s |
| Completed iterations / failures | **20 / 0**, plus two logged `Recover` calls in the interrupted 21st iteration | **50 / 0** |
| In-leg admission / suite samples | 2, all `free`, no foreign suite | 4, all `free`, no foreign suite |
| Maximum sample gap | 5.064 s | 5.066 s |
| Foreign CPU, maximum deciding interval | **0.996 cores** | **0.105 cores** |
| Foreign CPU, maximum post-exit interval | not sampled: stopped leg | **0.385 cores** |
| Post-exit record | none, both processes stopped | six suite/admission samples; 30.067 s recorded after exit |
| Agent/updater overlap | 10.127 s | 23.383 s, the entire measured process |
| Qualification | **unqualified, stopped_foreign_cpu** | **qualified, completed** |

- **r1:** the second deciding interval (ending at 10.089 s) exceeded the
  0.5-core limit. A non-lane `python3` process contributed 0.883 cores;
  aggregate foreign CPU was 0.996. Both lane processes were terminated,
  and no retry ran in that reservation. The 102 logged `Recover` calls
  contain no test failure; the interrupted iteration is not counted as
  completed.
- **r2:** reserved 13 min 17 s after r1's release, more than the required
  ten minutes. The first host-quiet check failed at a 30 s average of
  0.848 cores (one interval 4.487, mainly `grok` and `run_l5_qwen.py`).
  After the bounded wait, the next check passed at 0.114 cores. No foreign
  process was paused, signalled or reniced. Across the three checks, the
  retained wait ledger totals 150.397 s, within the 60-minute cap.
- **Exit boundary:** r2's interval ending at 25.210 s began at 20.169 s,
  before the measured process exited at 23.385 s. Its 0.083 cores therefore
  counts as deciding, as §1 declared. Post-exit intervals begin after the
  exit and are recorded only; none breached the limit here.
- **Load lifecycle:** the agent suite remained running throughout r2's
  post-exit sampling, then was terminated at 53.445 s. Its exit −15 is
  intentional cleanup, not an agent-suite pass or failure claim.
- **Excluded CPU, recorded separately:** kernel/runtime maxima were
  0.260/0.000 cores in r1 and 0.272/0.012 in r2. vmstat was recorded only:
  r ≤ 1 and idle ≥ 88% in r1, r ≤ 1 and idle ≥ 92% across r2's record.
- **Runner self-test:** the first session's retained `dry` run passed
  2/0 with no reservation. It has no in-leg suite sample because its
  measured process exited in 1.221 s; it is not a qualified comparison
  leg and is not pooled.

Both permitted reservations are spent. No further measurement was made.

### `Recover` wall times

Milliseconds, **min / median / p99 / max**, nearest-rank quantiles. The
sample count is per subcase; r1 includes the two calls logged before the
21st iteration was stopped. These times exclude fixture setup and use the
real monotonic clock; the meter wait itself uses the bubble clock.

| Subcase | r1 unqualified, n | r1 wall time | r2 qualified, n | r2 wall time |
|---|---:|---|---:|---|
| `missing` | 21 | 16.703 / 18.354 / 24.722 / 24.722 | 50 | 10.855 / 19.066 / 22.537 / 22.537 |
| `replaced` | 21 | 11.940 / 18.601 / 23.496 / 23.496 | 50 | 11.233 / 18.980 / 22.227 / 22.227 |
| `withdrawn` | 20 | 18.654 / 21.624 / 26.243 / 26.243 | 50 | 12.807 / 22.752 / 33.079 / 33.079 |
| `new_boot` | 20 | 16.349 / 18.655 / 21.752 / 21.752 | 50 | 10.996 / 18.767 / 21.926 / 21.926 |
| `expired` | 20 | 14.841 / 18.889 / 21.988 / 21.988 | 50 | 11.103 / 18.454 / 22.577 / 22.577 |

### Deterministic budget and pooling

`TestAT25_FileMeterWaitRestartKeepsDeadline -count=50` passed before the
window at 12:08:52Z and after it at 12:28:47Z. Both admitted runs contain
200 exact samples, 50 of each:

| Elapsed before restart | Remaining budget and actual wait |
|---|---|
| 0 s | 250 ms |
| 200 ms | 50 ms |
| 250 ms | 0 s |
| 1 s | 0 s |

All **352** logged parent `Recover` calls in r1/r2 also have the exact
budget: remaining 250 ms for `missing`, `replaced`, `withdrawn` and
`new_boot`, −1 ms for `expired`; bubble elapsed 250 ms for `missing` and
`replaced`, zero for the others. There is no deadline renewal or setup
budget consumption in these samples.

Follow-up 22's qualified quiet leg (200/0, foreign CPU maximum 0.345 cores)
is pooled with r2 under the declaration. Its historical unqualified load
legs stay unqualified. These results complete the declared comparison for
the fixed test; they do not establish a whole-`Recover` production SLO.

## 3. Item 2: the other short wall-clock windows in `node/updater`'s fixtures

D-141: judge before fixing. Scope: a test whose expected result depends on
its own work, or the scripted agent's, finishing inside a short (under 1 s)
wall-clock window that the engine reads. The meter budgets were judged in
follow-up 22 (§1 there) and are not repeated.

### How the `AttestWindow` is read (`attest.go`)

- `NodeChecks.poll` sets `deadline = n.now() + Wait` on the **injected**
  clock and opens `context.WithTimeout(ctx, Wait)` on the **real** clock.
- The loop always reads first (`readOnce`), and checks `!now.Before(deadline)
  || ctx.Err() != nil` only **after** that read. So the first read always
  happens, however slow the runner is.
- In the standard world (`newWorld`) the injected clock is fixed
  (`w.now`). The injected deadline therefore never passes, and the only
  thing that can end a failing poll is the real-clock context.
- `Issue` (the challenge's `IssuedAt`/`ExpiresAt`) and `Validate` both use
  that fixed clock, so the challenge TTL is **not** a wall-clock window in
  these tests (including the 10 ms TTL, clamped to 35 ms, below).

### Each window

| Test(s) | Window | Between the window opening and the engine's read | Can a slow runner change the expected result? | Control | Result | Change |
|---|---|---|---|---|---|---|
| `useAttest` (40 ms): every test in `attest_test.go` that uses it, and its users in `candidate_health_test.go`, `d062_test.go`, `followup6_test.go`, `probe_hardening_test.go`, `revocation_r3_test.go`, and the review tests below that build the same wiring | `Wait` 40 ms, real-clock context | Nothing of the test's. The scripted agent answers synchronously inside `Services.Start`, which the engine calls **before** `runHealth` opens the window. The first read inside the window already sees the final file. A pass breaks at that read. A rejection, `missing` or `not_run` gives the same reason at every later read (same file, fixed clock), so fewer reads change nothing | **No** | C2 (supporting): 100 ms blocking stall in the world's injected clock whenever `NodeChecks.poll` is on the stack, i.e. inside every window, over the whole package | **PASS**, 28 tests/subtests, 88 stalls; rerun on the fixed tree: PASS, 88 stalls | none |
| `TestUPD_D053_Review_Held_VersionTokenAndTTLClampCannotHideMismatch` | `Wait` 30 ms; TTL 10 ms clamped to 35 ms | As above; the TTL is on the fixed clock | **No** | covered by C2 (5 stalls) | PASS | none |
| `TestUPD_D053_Review_Held_PartialWriteAndChallengeWriteFailureRollBack` | 40 ms and 20 ms | As above (partial file is `unreadable` at every read; the write-failure path never polls the candidate) | **No** | covered by C2 (4 + 2 stalls) | PASS | none |
| **`TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires`** | `Wait` 80 ms, real-clock context | The first read sees the warm-up attestation (`tunnel_probe=failed`), then the loop checks the context. The test's own `nc.Now` runs inside that read (`Validate`) and in the check. The candidate's rewrite to `passed` happens only in the injected `Sleep`, **after** that check. A stall of more than 80 ms before the check ends the poll with `failed` | **Yes. Named: F23-01** | C1: 250 ms blocking `nanosleep` in the test's `nc.Now`, 2nd call (inside the first read) | current: **FAIL** (`got tunnel_probe="failed", want "passed"`, 0.29 s); fixed: **PASS** (0.30 s, the stall ran) | fixed (below) |
| `TestUPD_D053_Review_PollDeadlineIgnoresInjectedClock` | 30 s real-clock context | Stalls only end the poll earlier | **No**: the assertion is `sleeps <= 3`, which fewer sleeps still meet | — | — | none |

**F23-01 is a fixture defect, not a product defect.** Under the stall, the
engine did exactly what D-053 asks: it read for its whole window, never got
a passing answer inside it, and failed closed. Only the test assumed that
its own first read and clock calls finish inside 80 ms of wall time.

### The fix (test file only)

- `TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires`
  now runs inside `synctest.Test`, with the world built by follow-up 22's
  `bubbleWorld`. The `Wait` context then runs on the bubble clock, which
  advances only when the bubble is durably blocked. Disk and scheduler
  stalls take real time and none of the window.
- `Wait` (80 ms), `Poll` (10 ms), the TTL and the assertion are unchanged.
  No window was lengthened. With `-w`, the diff is +10/−1.
- **Mutation check (M1, product overlay, removed):** `poll` stops at the
  first attestation, the regression this test was written for. Both the
  current and the fixed test fail with the test's own message, so the
  bubble did not weaken it.
- Fixed test, plain: 20/20 passes.
- The controls are `-overlay` files under `.agent/tmp/f23/control/`, never
  in the tree: `git grep -n 'f23Control\|f23ClassStall' -- node/` finds
  nothing.

### Completion-session reruns (not measurements)

The parked patch was applied unchanged and committed as `159329d9`.
`git diff -w` against the declaration commit is **+10/−1**, confined to
`node/updater/review_d053_test.go`. Here, “current” means the retained
pre-fix overlay, even though the working tree now contains the fix.
Every command below ran after admission printed `free`; no reservation
was taken. The overlays under `.agent/tmp/f23/control/` were used as-is,
and their hashes remain unchanged.

| Control | First session | Completion session | Evidence |
|---|---|---|---|
| C1 current, 250 ms stall | FAIL (0.292 s package time) | FAIL (0.287 s) | stall logged; `got tunnel_probe="failed", want "passed"` |
| C1 fixed, same stall | PASS (0.305 s) | PASS (0.295 s) | stall logged in both runs |
| M1 current, first-attestation mutation | FAIL (0.041 s) | FAIL (0.037 s) | test's own warm-up message |
| M1 fixed, same mutation | FAIL (0.044 s) | FAIL (0.036 s) | same message; assertion still detects the regression |
| Fixed plain, `-count=20` | 20/20 PASS | 20/20 PASS | no control overlay |
| C2, full fixed package | PASS, 88 stalls in 28 tests/subtests | PASS, 88 stalls in the same 28 tests/subtests | all 176 top-level package tests pass |

There is **no result disagreement**. Wall runtimes differ as recorded;
they are control diagnostics, not measured comparison legs. Both sets of
logs are retained. The initial pre-fix C2 package run also passed with
88 stalls, as the draft records. The JSON lists the exact C2 cases and
stall counts, and both sessions' log tails.

To reproduce only these controls, use the declared environment and
admission before each command:

- C1/M1: `go test -v -count=1 -run
  '^TestUPD_D053_Review_PollBreaksOnTransientFailedCheckBeforeWaitExpires$'
  -overlay=.agent/tmp/f23/control/<control>-overlay.json ./node/updater`,
  where `<control>` is `c1-current`, `c1-fixed`, `m1-current` or `m1-fixed`.
- Plain: the same test at `-count=20`, without `-overlay`.
- C2: `go test -v -count=1
  -overlay=.agent/tmp/f23/control/c2-fixed-overlay.json ./node/updater`.

After the reruns, `git grep -n 'f23Control\|f23ClassStall' -- node/`
returns no matches (exit 1). No control was applied to a tracked product
file. **F23-01's fixture-defect reading is supported by C1, and the
unchanged parked fix removes that dependency while M1 preserves the
regression check.**

### Examined, outside the short-window class (recorded, not changed)

- `TestUPD_D053_PollRetainsLastValidFailure` (`followup6_test.go`): the same
  mechanism with a **30 s** real-clock context and an exact `sleeps == 3`
  assertion. Only a stall over 30 s could change it. Not a short window, so
  not controlled or changed; listed under known gaps.
- Download `AttemptTimeout` 5 s and the probe's 2 s DNS deadline: long
  windows. The 1–5 ms download backoff runs through the world's no-op
  `Sleep` (real timers only where `useAttest` clears it), which delays but
  never decides a result.
- Test-owned deadlines, not engine reads: the 5 s lock-holder waits
  (`followup13_test.go`, `review_fix_test.go`) and the 1 s "reader must not
  block" guard in `probe_hardening_test.go`.
- Outside `node/updater`: `node/internal/updatejournal`'s
  `TestD062_SnapshotDuringWriteTransaction` asserts `time.Since(start) <
  SnapshotTimeout` (1 s) on the wall clock. Recorded under known gaps.

## 4. Gates and hygiene

All gates ran in the completion session, in the foreground, each after
admission printed `free` (no admission wait). Environment:
`CGO_ENABLED=0 GOTOOLCHAIN=local GOMAXPROCS=4 GOFLAGS=-p=4`, disk-backed
`.agent/tmp` as `TMPDIR` and `GOTMPDIR`. The cache was read with
`go env GOCACHE` (`~/.cache/go-build`); Go was 1.27.3 on linux/amd64.

| Gate | Exit | Command wall time |
|---|---:|---:|
| `go test -count=1 ./node/updater/... ./node/nodelock/...` | 0 | 17.129 s |
| `go test -count=1 ./tests/handover/...` | 0 | 3.466 s |
| `make go-lint go-with-disk-tmp='set -eu;'` | 0 | 2.102 s |
| `go vet ./...` | 0 | 0.377 s |
| tracked Go files: `gofmt -l` empty | 0 | 0.225 s |
| `git diff --check` and `git diff --check main...HEAD` | 0 | 0.006 / 0.002 s |
| `make -f mk/updater.mk updater-lint` | 0 | 0.957 s |
| `make -f mk/updater.mk updater-build` (amd64 and arm64) | 0 | 1.421 s |
| `GOOS=linux GOARCH=arm64 go test -c` for `node/updater` | 0 | 1.098 s |

Real output tails:

```text
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	16.820s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	0.704s
?   	github.com/TED-H-com/Freedom-VPN/node/updater/fsutil	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/nodelock	0.092s
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.184s
```

The lint override is follow-up 22's: only the root helper's shell cleanup
trap is bypassed, as required by the brief. Gofmt, vet and the pinned
staticcheck v0.8.2 still run unchanged. All gates passed; no new
`meter_checkpoint_timeout` failure appeared in the fixed meter fixtures.

The final diff and hygiene checks pass. Only three owned files change
against the lane base: the D-053 test, this dated document and its JSON.
No dependencies or production code change. No control symbols remain
under `node/`. The timing JSON contains no private or home location and
no expanded admission-script path. Owned Docker inventory
(`fvpn-w2-updater-f23*`) is zero for containers, networks, volumes and
images; each read-only listing was admitted. No process started by the
completion session remains running.

## 5. Limits and hand-off

- No physical arm64 run, power-loss test, production load claim or
  whole-`Recover` SLO is added. The arm64 evidence here is cross-compilation.
- The CPU sampler still misses CPU used between a process's last 1 s
  snapshot and its exit. Qualification uses its recorded samples under
  the declared rule.
- The parent's `replaced` checkpoint still lacks `barrier_id`, as noted
  in follow-up 22; no coverage change is made here.
- The 30 s `PollRetainsLastValidFailure` context and the test-owned
  guards listed in §3 were inspected but not controlled or changed. The
  update-journal snapshot test is outside this item’s `node/updater`
  scope. These are recorded observations, not newly proven defects.
- Spec questions: none.
- Requests to orchestrator, **2026-11-03**: none. The previously missing
  qualified bounded-load leg is now supplied by r2; both reservations
  remain spent. Historical verdicts and unqualified runs stay intact.
