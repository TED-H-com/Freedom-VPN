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

Pending: added after the window. Raw logs and window records are under
`.agent/tmp/f21/`.
