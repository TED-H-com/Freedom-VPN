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
