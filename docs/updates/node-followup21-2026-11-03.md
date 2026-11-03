# Updater follow-up 21 — 2026-11-03

Base: `63a30c41` (main through #409). Scope: F27-02's qualified comparison
only, measured in one reserved host window. No production or test source
changes. Requirement scope: UPD03/UPD05, MTR03, §13.5–§13.9, §19 AT25
(component level; no assembled acceptance or physical arm64 claim).

Follow-up 20 (#408) corrected the fixture. The one-second assertion in
`TestAT25_FileMeterWaitRestartRechecksCurrentFile/missing` timed all of
`Recover`. It now bounds only the meter wait, deterministically, in
`TestAT25_FileMeterWaitRestartKeepsDeadline`. Its four 200-iteration runs
stay recorded as unqualified in
[follow-up 20](node-followup20-2026-11-03.md) and its
[timing record](node-followup20-timing-2026-11-03.json). Netlab 27's F27-02
stays failed. Neither file is edited here.

## 1. Inputs checked on current `main`

Checked at 2026-11-03T09:22Z, before the declaration and before any
measured run.

- **Overlay.** `.agent/tmp/f20/before-overlay.json` has a single `Replace`
  entry, `node/updater/meter_barrier_test.go`, mapped to the retained
  `before-meter_barrier_test.go`. Compared with that file as of `ea7a6f35^`
  (the original test), the only difference is in the `missing`/all-subcase
  timing block: `elapsed := time.Since(start)`, a
  `t.Logf("F20 recover_elapsed_ns=%d original_remaining_ns=%d", ...)` line,
  and `if elapsed > time.Second` in place of `if time.Since(start) >
  time.Second`. The old assertion and its message are unchanged. With
  admission `free`, `go test -count=1 -overlay=<that file> -run '^$'
  ./node/updater` built and exited 0 on `63a30c41`.
- **Corrected tests.** `git log -- node/updater/meter_barrier_test.go`
  ends at `ea7a6f35` (#408), and `git diff ea7a6f35 HEAD -- node/updater/`
  is empty. Nothing under `node/updater` has changed since #408. The
  corrected parent test still logs the same `F20 recover_elapsed_ns=` line,
  so both variants are parsed the same way.
- **Deterministic budget, pre-window.**
  `go test -v -count=50 -run '^TestAT25_FileMeterWaitRestartKeepsDeadline$'
  ./node/updater` passed (200 samples). Each subcase reported its exact
  original remainder: 0 → 250 ms, 200 ms → 50 ms, 250 ms → 0 s, 1 s → 0 s.
- `go test -count=1 -run '^$' ./node/agent/...` also built. That warmed
  the build cache, so compile time does not land inside the measured legs.
  None of these four commands is a measured leg.

## 2. Declaration (committed before any measured run)

### Legs, in order, in one reservation

All legs use follow-up 20's commands with `CGO_ENABLED=0 GOTOOLCHAIN=local
GOMAXPROCS=4 GOFLAGS=-p=4`, worktree `.agent/tmp` as `TMPDIR`/`GOTMPDIR`,
and the cache from `go env GOCACHE` (`~/.cache/go-build`). `PARENT` is
`'^TestAT25_FileMeterWaitRestartRechecksCurrentFile$'`, and `OVERLAY` is
`-overlay=.agent/tmp/f20/before-overlay.json`.

| # | Leg | Processes | Watchdog |
|---|---|---|---:|
| 1 | quiet baseline | `go test -v OVERLAY -run PARENT -count=200 ./node/updater` | 360 s |
| 2 | quiet corrected | `go test -v -run PARENT -count=200 ./node/updater` | 360 s |
| 3 | bounded-load baseline | `go test -count=1 ./node/agent/...` started, then immediately `go test -v OVERLAY -run PARENT -count=50 ./node/updater`, concurrently | 300 s |
| 4 | bounded-load corrected | the same, without `OVERLAY` | 300 s |

**Budget.** One reservation:
`bash "$FVPN_HOLD_SH" --reserve 30 "updater f21 F27-02 comparison"`.
Then wait 60 s and run the legs back to back. Release with
`bash "$FVPN_HOLD_SH" --release` as soon as the last leg ends.

- **Worst case:** 60 s + 4 × 35 s of preflight + 2 × 360 s + 2 × 300 s
  = 25.3 min, which fits in 30 minutes. The load legs therefore stay at
  `-count=50`.
- **Run times:** follow-up 20's 200-iteration runs took 216–239 s of test
  time. Its agent suite took about 80 s.
- **Leg start rule:** a leg starts only if its whole watchdog, plus a
  30 s release margin, still fits before the reservation ends. Otherwise
  it is recorded as not started.

**Preflight before each leg** (the brief's quiet checks), recorded with its
wait time:

1. `bash "$FVPN_HOLD_SH"` prints `free`.
2. `pgrep -af 'gates.sh|mk/at-|go test'` shows no other lane's suite
   process.
   - **Excluded:** processes descended from the measurement runner.
   - **A suite process** has `go` as its executable with `test` as its
     subcommand, or an argument naming a `gates.sh` script or an `mk/at-`
     makefile.
   - **Not a suite:** a shell whose script text merely contains these
     strings. It is still recorded.
3. `vmstat 5 7`: every interval after the first (since-boot) row has
   r ≤ 3 and id ≥ 80.
4. Steps 1 and 2 are repeated after the 35 s vmstat.

A failed preflight is retried every 30 s, for at most 300 s per leg, within
the budget.

### What qualifies a leg

- **Quiet leg.** All of the following:
  - the preflight passed;
  - at every in-leg sample, taken every ≤ 10 s (nominally 5 s, admission
    plus process scan), admission printed `free` and no other lane's suite
    process was present;
  - the leg ran to completion;
  - every in-leg `vmstat 5` interval kept r ≤ 3 and id ≥ 80. This is the
    fail-closed reading of "keep the vmstat thresholds". Our own test
    process counts here.
- **Load leg.** The same, except the in-leg vmstat thresholds. Our own
  agent load is expected to breach them, so in-leg vmstat is recorded but
  does not decide qualification. The preflight vmstat still applies.
- **Test failures do not disqualify a leg.** They are the measurement.
  Every leg log is retained, whether it passed, failed or was stopped.

### Stop rules

During a leg, either of these ends it:

- another lane's suite process appears;
- admission stops printing `free`.

Then both of our processes are terminated (process group SIGTERM, then
SIGKILL after 10 s), and the leg is recorded as **unqualified**. It is not
retried in the same reservation; the remaining legs continue after their
own preflight.

- **Watchdog or reservation end:** the leg is stopped and recorded as
  unqualified (incomplete) in the same way.
- **Second reservation:** at most one more, only for missing or
  disqualified legs, at least 10 minutes after the first is released.

### Deterministic budget check

The deterministic meter-wait test is not part of any leg, so the legs stay
exactly follow-up 20's commands. It runs pre-window (above) and again after
the window, at `-count=50` each time. The exact remainders must be
250/50/0/0 ms both times. The test runs on a synctest clock. A pass
therefore shows that the budget logic does not drift. It does not show
that wall-clock waits stay exact under load.

### What each outcome would show

- **The fixture-defect reading predicts:**
  - The old assertion (baseline legs) *can* fail under load: some
    subcase's whole-`Recover` time exceeds 1 s, with no deadline renewal.
    It is not required to fail in any particular run.
  - The corrected parent test passes in every leg. It no longer has a
    whole-`Recover` wall-time assertion; its recovery, error, receipt and
    forwarding checks remain.
  - Before and after the window, the deterministic budget is exactly
    250/50/0/0 ms.
  - Recover's `missing` distribution sits near the 250 ms wait plus
    recovery work, with a tail that grows under load.
- **What would contradict it:**
  - any corrected-leg failure, above all a `missing` failure with a
    meter-wait detail other than `meter_checkpoint_timeout`;
  - any change in the deterministic remainders;
  - a `missing` Recover time near the configured one minute, which would
    mean the restart had renewed the deadline.
- **Verdict rule, fixed now:**
  - **Failed:** any of the contradictions above, in any leg, qualified or
    not. The corrected test has to pass under any load.
  - **Held:** there is no contradiction, both corrected legs qualified,
    and both deterministic checks are exact. A baseline failure under load
    supports the reading but is not required. A baseline pass under load
    only means the old assertion's flakiness did not recur in this window,
    and the report says so.
  - **Undecided:** there is no contradiction, but a corrected leg is still
    unqualified after the permitted reservations.

## 3. Results

### Reservation 1 (09:23:58Z–09:37:15Z, 797 s held)

Reserved with `reserved for w2-updater until 04:53:58` (local), and
released with `released w2-updater` right after leg 4. All four legs ran to
completion. No stop rule fired at run time. Every preflight passed on its
first attempt. Each took 30.1 s: `vmstat 5 7` spans 30 s, not the 35 s
budgeted above. There was no further wait.

| Leg | Iterations / failures | In-leg samples | Qualification |
|---|---|---|---|
| quiet baseline | 200 / 1 (`expired`) | 42, max gap 5.06 s, all `free`, no foreign suite | **unqualified**: one in-leg vmstat interval at r = 4 (id 95), t = 5 s |
| quiet corrected | 200 / 0 | 42, max gap 5.06 s, all `free`, no foreign suite | **unqualified**: one in-leg vmstat interval at r = 5 (id 95), t = 60 s |
| load baseline | 50 / 0 | 17, all `free`, no foreign suite | **qualified**; 88.4 s active agent load, covering the whole updater process (0–58.9 s) |
| load corrected | 50 / 0 | 17, all `free`; from t = 15.2 s, `bash .agent/tmp/followup31/run-gates.sh` (another lane) present at 15 samples | **unqualified** (see the correction below); 87.6 s active agent load |

**Runner correction, after the fact.** The runner's classifier checked
only whether an argument's basename was exactly `gates.sh`. The declaration
counts any argument that names a gates script. `run-gates.sh` names one, and
the brief's own pattern `gates.sh` matches it. The fail-closed reading is
that this is another lane's suite runner, not a shell that merely mentions
the strings. Leg 4 therefore should have been stopped at 15.2 s. It is
recorded as **unqualified**, and its completed data is kept as unqualified.
The classifier now counts any argument ending in `gates.sh`. The timing JSON
keeps the runner's original verdict as `qualified_as_recorded`, beside the
corrected `qualified`. That other lane started no `go test` during the leg,
and the in-leg vmstat never breached (max r 3, min id 87).

### Reservation 2 addendum (committed before reservation 2)

This is the one further reservation the brief allows, for the missing legs
only. It is taken at least 10 minutes after the 09:37:15Z release.

- **Legs, in the declared order:** quiet baseline, quiet corrected, load
  corrected (`-count=50`). The commands, watchdogs, preflight, sampling,
  stop rules and the in-leg vmstat rule for quiet legs are the same as in
  §2. Nothing is loosened.
- **The only change is the classifier fix above,** which is stricter.
- **Budget:** 60 + 3 × 35 + 2 × 360 + 300 s = 19.8 min, which fits in 30.
- **No retries:** a leg that is disqualified again is not retried. It
  stays unqualified.

Reservation 1's results stand as recorded. Raw logs and window records are
under `.agent/tmp/f21/` (`r1-*`).

### Reservation 2 (09:47:33Z–09:59:59Z, 746 s held)

The host was free at the first check, 09:47:33Z, 10 min 18 s after the
first release. Reserved with `reserved for w2-updater until 05:17:33`
(local), released with `released w2-updater`.

| Leg | Preflights (wait) | Iterations / failures | Qualification |
|---|---|---|---|
| quiet baseline | 2 (90.2 s). The first failed vmstat (intervals r = 13 and r = 5); no foreign suite | 200 / 0 | **unqualified**: one in-leg vmstat interval at r = 5, id 75, t = 50 s. Admission `free` and no foreign suite at all 42 samples |
| quiet corrected | 1 (30.1 s) | 178 completed / **1 failure** (`replaced`), then stopped | **unqualified, stopped:** at 09:57:28.8Z (t = 197.3 s), `go test -v -run ^TestR22_ ./node/agent` from another lane appeared during our reservation, while admission still printed `free`. Both of our processes were terminated. No in-leg vmstat breach |
| load corrected | 2 (60.2 s). The first saw that same foreign `go test`; the second passed at 09:57:58.9Z | 50 / 0 | **qualified**; 88.8 s active agent load, covering the whole updater process (0–59.1 s) |

The quiet-corrected failure happened at about t = 101–106 s, between
Recover samples 446 and 469; it was the 462nd. That is about 90 s
*before* the foreign process was first seen. Every sample up to
t = 192.2 s was `free` with no foreign suite, and no in-leg vmstat interval
breached.

```text
meter_barrier_test.go:285: F20 recover_elapsed_ns=15122804 original_remaining_ns=-283098718
meter_barrier_test.go:290: failure journal: {... Stage:rolled_back ... ErrorCode:CHECKPOINT_FAILED
  ErrorStage:final_sample ErrorDetail:meter_checkpoint_timeout ...}
```

**Reading the failure.** `replaced` expects `meter_checkpoint_uncorrelated`.
Here the persisted 250 ms deadline had expired 283.1 ms **before**
`Recover` started. So the test's own setup took about 533 ms from writing
the wait note to calling `Recover`; normally that takes 10–25 ms. The setup
here means the simulated crash, closing and reopening SQLite, and writing
the replacement checkpoint. `Recover` itself took 15.1 ms. The deadline
was already gone before any checkpoint read, so `waitMeter` correctly
reported a timeout instead.

The failure is fail-closed. Forwarding stayed blocked and the receipt
stayed null-usage. Only the detail differed. This is **not** a renewal: a
renewal would lengthen `Recover` towards one minute, but it took 15 ms. It
is a second wall-clock dependency in the corrected parent test. The test
needs its pre-`Recover` setup to finish inside the 250 ms budget. The same
pre-`Recover` expiry happened twice in `withdrawn`: −991.5 ms (r2 quiet
corrected, sample 3) and −772.5 ms (r2 load corrected, sample 113). Both
passed, because the withdrawn gate is reported first. Across all
3,716 non-`expired` samples (4,644 `Recover` samples in all), these
three are the only ones that started `Recover` after the deadline.

## 4. Comparison

`Recover` wall time per subcase, in milliseconds. These are nearest-rank
quantiles. With n = 50, p99 is the maximum. Every leg is listed, qualified
or not; "Q" marks a qualified leg. Full distributions, original remaining
budgets, preflight and sample records are in the
[timing record](node-followup21-timing-2026-11-03.json).

| Window / leg | Q | Iter. / fail | missing min / med / p99 / max | replaced | withdrawn | new_boot | expired |
|---|---|---|---|---|---|---|---|
| r1 quiet baseline | no | 200 / 1 | 253.1 / 260.8 / 264.1 / 264.6 | 239.1 / 248.5 / 252.4 / 254.5 | 12.6 / 14.5 / 16.7 / 272.8 | 10.8 / 12.4 / 15.0 / 17.0 | 13.2 / 20.8 / 24.7 / **1443.3** |
| r1 quiet corrected | no | 200 / 0 | 253.2 / 261.0 / 264.1 / 265.6 | 240.0 / 248.9 / 252.1 / 252.5 | 12.5 / 14.4 / 16.7 / 18.1 | 10.6 / 12.4 / 15.0 / 23.5 | 12.7 / 21.0 / 24.5 / 27.7 |
| r1 load baseline | **yes** | 50 / 0 | 248.7 / 254.4 / 259.0 / 259.0 | 229.2 / 236.6 / 686.5 / 686.5 | 13.6 / 22.4 / 26.2 / 26.2 | 11.9 / 19.2 / 23.3 / 23.3 | 12.5 / 20.2 / 26.4 / 26.4 |
| r1 load corrected | no | 50 / 0 | 249.0 / 254.6 / 755.0 / 755.0 | 229.1 / 236.1 / 257.5 / 257.5 | 13.1 / 21.7 / 27.8 / 27.8 | 10.9 / 18.3 / 23.4 / 23.4 | 13.6 / 19.5 / 24.2 / 24.2 |
| r2 quiet baseline | no | 200 / 0 | 252.3 / 261.0 / 267.7 / 761.8 | 240.4 / 248.9 / 253.2 / 253.5 | 12.4 / 14.2 / 17.0 / 17.8 | 10.8 / 12.2 / 14.8 / 17.0 | 13.4 / 21.0 / 23.7 / 24.6 |
| r2 quiet corrected (stopped) | no | 178 / **1** | 254.0 / 261.2 / 275.9 / 789.5 | **15.1** / 248.8 / 252.6 / 252.7 | 12.6 / 14.4 / 16.8 / 404.4 | 10.8 / 12.3 / 14.2 / 14.6 | 12.7 / 20.9 / 25.0 / 31.2 |
| r2 load corrected | **yes** | 50 / 0 | 249.9 / 254.0 / 262.7 / 262.7 | 210.8 / 236.1 / 250.8 / 250.8 | 12.9 / 22.5 / 26.0 / 26.0 | 11.0 / 18.4 / 22.8 / 22.8 | 12.3 / 19.9 / 25.3 / 25.3 |

The r2 quiet-corrected leg has 179 samples for four subcases and 178 for
`expired`, because it was stopped inside iteration 179.

- **Qualification record.**
  - Qualified: load baseline (r1) and load corrected (r2).
  - Not qualified: quiet baseline, twice, each time for one in-leg vmstat
    interval with our own test running (r = 4 and r = 5). Quiet corrected
    was disqualified once by vmstat (r1, r = 5) and once by another lane's
    `go test` (r2, stopped). Load corrected in r1 was disqualified by
    another lane's `run-gates.sh`.
  - Neither quiet leg ever qualified. No leg is retried beyond the two
    reservations allowed.
- **Active load seconds:** 88.4 (r1 load baseline), 87.6 (r1 load corrected,
  unqualified), 88.8 (r2 load corrected). Total 264.8 s. In every load leg
  the updater process (58.8–59.1 s) ran entirely inside the agent suite,
  so all 250 `Recover` calls ran under load. The agent suite passed each
  time: `ok node/agent 88.133s / 87.340s / 88.458s`.
- **The old assertion failed once, on a quiet host, not under load.** In
  r1 quiet baseline, `expired` took 1443.3 ms. Its persisted deadline had
  already passed (−1.9 ms) before `Recover` started, so a renewed deadline
  could not have caused the delay: a renewal would mean a one-minute wait.
  It did not fail in the qualified load-baseline leg. In fact, the load legs
  are not heavier for this test: in-leg max r was 1–3 and min id 87–94,
  while the quiet legs had min id 75–89. With 24 CPUs and `GOMAXPROCS=4`,
  the agent suite is light load. Whole-`Recover` stalls happen in both conditions.
  The quiet legs show 0.27–1.44 s outliers (for example, `missing`
  761.8 ms in r2 quiet baseline). The load legs show 0.69–0.76 s outliers
  (`replaced` 686.5 ms, `missing` 755.0 ms).
- **Deterministic budget:** exact both times. Before the window
  (09:22Z) and after it (10:00:54Z), all 50 iterations of every subcase
  reported 0 s → 250 ms, 200 ms → 50 ms, 250 ms → 0 s and 1 s → 0 s. Not
  one sample drifted.

## 5. Verdict on F27-02's fixture-defect reading: **failed** (by the declared rule)

The declaration in §2 says the reading **fails** on any corrected-leg
failure, qualified or not. The corrected parent test failed once:
r2 quiet corrected, `replaced`, iteration 93. That happened while the leg
was still clean, about 90 s before the foreign process stopped it. By the
rule fixed before measurement, the reading as stated ("the corrected test
does not fail") **failed**.

What the failure does and does not mean:

- **The original diagnosis is not overturned.** The old one-second
  assertion fired again on a stall that cannot be a renewal (r1 `expired`,
  1443 ms, with the deadline already past). Nothing in 4,644 `Recover`
  samples looked like a renewal. No `Recover` took over 1.5 s, and none
  came close to the one-minute configuration. The deterministic budget
  was exact before and after.
- **But the corrected fixture is not free of wall-clock flakiness.**
  `TestAT25_FileMeterWaitRestartRechecksCurrentFile/replaced` still needs
  its setup (crash, SQLite close and reopen, checkpoint replacement) to
  finish within the 250 ms persisted budget. When the setup stalls for
  about 0.5 s, the expected detail changes from `uncorrelated` to `timeout`.
  Product behaviour is fail-closed in both cases. This is new finding
  **F21-01**.
- **The quiet-versus-load comparison stays incomplete.** Neither quiet leg
  qualified in the two permitted reservations. Both load legs qualified.
  In this window the old assertion's failure rate does not depend on the
  light agent load.

History stays as recorded: netlab 27's F27-02 stays failed, and
follow-up 20's four runs stay unqualified. Neither file is edited.

## 6. Known gaps and later work

- **F21-01 (updater-owned, not fixed here; the task was measurement
  only):** make the corrected parent test independent of wall-clock setup
  time. Proposed shape:
  - Give the cases that do not need expiry (`replaced`, `withdrawn`,
    `new_boot`) a persisted budget long enough that setup cannot consume
    it, for example 30 s. They return on the first read or validation, so
    the test does not get slower.
  - Keep 250 ms only for `missing` and `expired`. Their outcome is a
    timeout either way.
  - Keep the exact budget checks in the synctest test.
  - Then measure again under the same rules. Do not relax the expected
    detail.
- **The qualified quiet legs are still missing.**
  - The fail-closed in-leg vmstat rule (r ≤ 3 and id ≥ 80, own process
    included) disqualified three of the four quiet attempts. Each time it
    was a single interval with id still at or above 75.
  - The orchestrator should confirm whether the brief's thresholds apply
    in-leg to quiet legs, or only to the preflight.
  - Under the preflight-only reading, r1 quiet baseline, r1 quiet
    corrected and r2 quiet baseline would have qualified. This report does
    **not** relabel them.
- **The reservation did not keep other lanes out.** During reservation 2,
  another lane's `go test -v -run ^TestR22_ ./node/agent` ran while
  admission printed `free`. During reservation 1, a `run-gates.sh` was
  present, apparently waiting. See the request below.
- **Not covered:** physical arm64 runs, power-loss durability and
  worst-case production load. These runs are not a whole-`Recover`
  wall-time SLO.

## 7. Requests to orchestrator

- **2026-11-03 — host reservation (scheduling, outside updater paths):**
  during `w2-updater`'s reservation (09:47:33Z–09:59:59Z), a foreign
  `go test -v -run ^TestR22_ ./node/agent` ran. It was first seen at
  09:57:28.8Z, was absent at 09:57:23.7Z, and was gone by 09:57:58.9Z. It stopped our quiet-corrected leg. Admission kept printing
  `free` to us. Please check whether that lane bypassed admission, or
  whether non-timing-sensitive runs pass a reservation. If a further quiet
  comparison is scheduled, please also confirm the in-leg vmstat reading.
- **2026-11-03 — F21-01 scheduling:** the corrected
  `TestAT25_FileMeterWaitRestartRechecksCurrentFile/replaced` can fail
  fail-closed with `meter_checkpoint_timeout` when its setup exceeds the
  250 ms budget. Please schedule the updater-owned fixture fix above and a
  re-measurement in a later follow-up. Netlab and merge gates may see it
  as a flaky `replaced` failure in the meantime.
