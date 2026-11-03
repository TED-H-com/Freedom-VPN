# Updater follow-up 20 — 2026-11-03

Base: `535ea71a` (main through #404). Scope: F27-02's restart meter-wait
assertion, and updater-side facts/proposal for agent-core's F30-01. No runtime
change is proposed for F27-02; no F30-01 implementation is included.
Requirement scope: UPD03/UPD05, MTR03, §13.5–§13.9, §19 AT25/AT26. These are
component checks, not new live acceptance or physical arm64 evidence.

## Changes since follow-up 19

Read the [roadmap](../ROADMAP.md) rows from #352 through the base. Relevant
changes and evidence are:

- #352/#353/#359 close the installer wrapper/config/reporting hand-offs and
  retain F18-01's intended stopped-producer negative (apply/replay/recover
  1/1/2). The FileMeter deadline remains updater-owned.
- #356/#365 retain contention/restart failures without inferring causes from
  CPU load or a later pass. #369 changes graceful agent shutdown; #376 shares
  its bounded local deadline; #378 gives systemd 15 s and installer stops 20 s.
  #372/#390 exercise updater stops and later add parent wait receipts. These
  stop-path changes do not run in F27-02's missing-checkpoint failure.
- #379's intersection-serving design is not production behavior. #388 adds
  applied-reference persistence and conntrack reservations; #393 checks meter
  exclusion of `ctdenyd` and identifies the restart deny-set gap; #401 closes
  that agent adoption gap and moves signal handling before Boot. The updater
  still starts a candidate under its own exclusive lock.
- Handover #354/#358/#364/#371/#374/#384/#395/#398 preserve the dated cuts and
  historical failures. A subsequent passing merge gate does not invalidate
  netlab 27's failure.

Netlab 27's `docs/poc/G2-followup27-evidence/gate3-agent.txt` and
`updater-failure.json` were read from the local `w1-netlab` ref. They record
`Recover` returning `ErrMembersBlocked` and the later assertion failing; 2.22 s
is the **subtest** duration. They contain no measured `Recover` duration or
phase timings, so its exact historical delay source remains unproven. No test
in this change reads those evidence files.

## F27-02: what was measured

In [`TestAT25_FileMeterWaitRestartRechecksCurrentFile`](../../node/updater/meter_barrier_test.go),
the original `start := time.Now()` occurs **after** setup, the simulated crash,
reopening SQLite, replacing the engine and removing the checkpoint. Therefore
`time.Since(start)` measures the whole call to `Recover` (plus the immediate
error assertion). It measures neither subtest setup nor time since the original
wait deadline. The error text, “restart renewed the original wait deadline”,
is not a valid inference from that whole-call wall time.

The intended property is that restarting with `FileMeter.Wait = time.Minute`
does not replace the persisted 250 ms budget. In the missing-file path:

1. `Recover` calls `open`/`Layout.Ensure`, acquires and fsyncs lock metadata,
   cleans installer snapshots/finished jobs and reads the active transaction.
2. `continueTxn` resumes `blocked` without republishing the latch;
   `stepSample` calls `waitMeter`.
3. `waitMeter` reads the existing note, validates boot/identity/time, creates
   `context.WithTimeout(ctx, time.Until(note.Deadline))`, and checks both the
   context and `note.Deadline` before and after each checkpoint read. The
   100 ms poll can be interrupted by that context. A missing file ends with
   `CHECKPOINT_FAILED: meter_checkpoint_timeout`.
4. Recovery still must persist `rolling_back` and nondestructive `rolled_back`,
   seal the failed receipt, persist the blocked gate and installer completion,
   consume pending state, flush/retain as configured and clean notes. Finally
   it rechecks publication/revocations and returns `ErrMembersBlocked`.

Steps 1 and 4, filesystem stalls and descheduling are included in the old
one-second assertion. None renews the meter deadline. In this fixture the
services are never stopped or started; the failure receipt remains null for
usage and member forwarding remains blocked.

### Fixture correction and discriminating controls

Keep the full recovery integration test and its error, receipt, current-release,
no-service-restart and blocked-forwarding assertions. Log its measured elapsed
time without treating it as proof of renewal. Add
`TestAT25_FileMeterWaitRestartKeepsDeadline` using standard-library
`testing/synctest`: real FileMeter/FileGate, a real serialized wait note, new
engine/adapters and a new one-minute configuration; no network/database in the
clock bubble. Replay preparation and require the note to remain byte-identical.
The **one-second ceiling is unchanged**, now applied to the actual meter wait;
an additional exact assertion requires 250/50/0/0 ms remaining after
0/200/250/1000 ms downtime. Disk I/O and scheduler stalls consume real wall time
but cannot advance this test clock. No production timeout or clock seam changes.

Two separate controls establish what this correction accepts and rejects:

| Controlled intervention | Before | After | Meaning |
|---|---|---|---|
| Existing `AfterStage` hook sleeps 1.1 s at `rolling_back`, after the missing-file wait has already failed | `Recover` 1355.440 ms; old renewal assertion FAIL | `Recover` 1352.424 ms; full recovery checks PASS | Delaying nondestructive rollback/sealing is not deadline renewal. These are single controlled samples, not the natural failure's measured cause. |
| Temporarily reset `note.Deadline = time.Now().Add(m.Wait)` immediately before wait timeout construction | N/A | All four new timing cases FAIL the unchanged one-second ceiling | The replacement detects a real one-minute renewal deterministically, without a real minute's delay. |
| Restore unchanged product after the mutation | N/A | All four PASS: 250/50/0/0 ms | Both fault injections were removed; runtime diff is empty. |

The first 200-iteration campaign began with `hold.sh=free` and no matching
suite process, but another lane was running at the end. Preserve it as
**unqualified for quiet-host comparison**, not a qualified quiet run. It found
one failure in `new_boot`: measured `Recover` 1438.847 ms; all 200 missing-file
samples were 249.773–264.267 ms. `new_boot` returns `meter_wait_invalid` before
constructing the polling timer, so that failure cannot be a renewed meter
wait. It proves the assertion's false attribution; it does not identify which
filesystem/scheduling operation delayed the call. Netlab's historical missing
case remains failed and its individual latency source remains unproven.

A second monitored 200-iteration baseline attempt was also disqualified:
agent-core started at the 141.537 s sample while `hold.sh` still printed free;
16 of 45 admission samples recorded another suite. This attempt reproduced the
**missing** failure, with `Recover = 1759.503 ms` (printed subtest 1.81 s).
It is retained separately, not counted as a quiet or intentionally loaded leg.
No operation-level trace identifies the delayed phase in this sample.

A third 200-iteration baseline attempt passed, but it also lost qualification
(first foreign-suite sample at 40.533 s; 40/49 samples recorded
other suites). It is retained as unqualified too. Passing does not repair the
first two failures or establish a quiet-host wall-time bound.

### Recorded distributions and admission gap

Every row below has 200 `Recover` samples for **missing**, from a complete
`-run '^TestAT25_FileMeterWaitRestartRechecksCurrentFile$' -count=200` run.
Values are milliseconds; quantiles use nearest rank. The
[timing record](node-followup20-timing-2026-11-03.json) also records all other
subcases and original remaining budgets. These runs are **unqualified** for a
quiet-versus-controlled-load comparison. They are not evidence of a performance
improvement or a whole-Recover wall-time guarantee.

| Run | p50 | p95 | p99 | Max | Missing > 1 s | Overall result |
|---|---:|---:|---:|---:|---:|---|
| Before 1 | 260.363 | 262.345 | 263.728 | 264.267 | 0 | FAIL: new_boot |
| Before 2 | 259.695 | 262.896 | 264.212 | 1759.503 | 1 | FAIL: missing |
| Before 3 | 256.551 | 266.165 | 271.497 | 273.487 | 0 | PASS |
| After | 260.198 | 263.084 | 264.049 | 264.833 | 0 | PASS |

The corrected 200-iteration run also encountered another lane after admission
(13/46 observations include pre-admission waiting or foreign suites), so its
PASS remains unqualified. The monitored runners checked `hold.sh` and
`pgrep -af 'gates.sh|mk/at-|go test'` every five seconds while running, and
waited 60 seconds between blocked admission checks. The third baseline used
a Go overlay containing only the elapsed-time logging change; the first two
used that same instrumented source directly. The final committed source was
restored unchanged. All runs used disk scratch, CGO disabled, the local
toolchain, `GOMAXPROCS=4` and `GOFLAGS=-p=4`.

**Unfinished measurement:** no fully qualified quiet 200-iteration leg was
obtained. The intentional parallel load leg never started because other lane
suites repeatedly occupied admission. Its active duration is **0 s**, within
the 15-minute ceiling. No mixed run is relabeled as the requested load leg.
After the final test completed, the owned admission waiter was stopped; no
test or background command was left running. A coordinated window is requested
below. The fixture mechanism and controlled before-fail/after-pass are proved,
but the requested qualified quiet/load comparison remains open.

## F30-01: current updater behavior

The health step is named `stepCheck` in this tree (the prompt's “stepHealth”).
[`engine.go`](../../node/updater/engine.go) retains the lock around recovery and
apply, including service start and health checks:

```go
lk, err := nodelock.AcquireWithBootID(ctx, e.Layout.Lock, 5, 20*time.Millisecond, e.Node.BootID.String())
// on ErrBusy:
return transient(apierr.UpdateBusy, lockBusyDetail(err))
// on success:
defer lk.Release()
e.lockHeld = true
defer func() { e.lockHeld = false }()
return fn()
```

With a valid configuration/journal and a live apply holding `update.lock`, a
separate `fvpn-updater recover` exhausts **five attempts/four 20 ms sleeps**
(nominal 80 ms plus scheduling/file work). `Recover` opens the journal first,
but never enters `recoverLocked` after lock refusal: no transaction advancement,
recovery seal or gate reopening occurs. It returns retryable
`OpError{Code: UPDATE_BUSY, Detail: "lock", Retry: true}`. The CLI prints exactly:

```text
fvpn-updater: updater: UPDATE_BUSY: lock
```

Exit status is **1**, not 2. [`ExitCode`](../../node/updater/host.go) reserves 2
for `ErrMembersBlocked`; [`command`](../../node/updater/cmd/fvpn-updater/main.go)
prints the other errors with `fmt.Fprintf(stderr, "fvpn-updater: %v\n", runErr)`.
Stale holder diagnostics can instead suffix `lock boot id`, `lock pid`,
`lock process` or `lock start time`, still exit 1, and still never remove the
locked inode. Configuration/open failures can preempt lock acquisition; the
80 ms is not a CLI wall-time guarantee.

[`apply.go`](../../node/updater/apply.go) makes the sequence explicit:

```go
// stepStart, while txn.Stage == StageSwitched:
if err := e.Services.Start(ctx, e.Layout.releaseDir(txn.ReleaseID)); err != nil {
    return e.beginRollback(ctx, txn, apierr.StartFailed, api.UpdateStageStart, nil)
}
// open epoch, then:
_ = e.Journal.SetJobPhase(ctx, txn.JobID, txn.Attempt, updatejournal.JobChecking)
return e.advance(ctx, txn, updatejournal.StageChecking, updatejournal.StageUpdate{Checkpoint: cp})
```

`stepCheck` then issues the challenge, calls idempotent `Services.Start` again,
and runs health checks without releasing the lock. The first candidate start
can precede challenge publication. A challenge file's absence at that moment
is not evidence of a leftover transaction.

The candidate can use the read-only WAL
[`updatejournal.Snapshot`](../../node/internal/updatejournal/snapshot.go) without
acquiring the update lock. In an ordinary clean successful-start attempt, it
can observe:

| Apply location | Durable transaction | Snapshot `stage` / `state` | Pointers / evidence |
|---|---|---|---|
| Candidate starts in `stepStart` | `switched`, destructive | `switch` / `applying` | Current is candidate; last-good is prior baseline. Final sample is durable. New epoch/checking may not yet be committed. |
| Health in `stepCheck` | `checking`, destructive | `check` / `checking` | Current is candidate; last-good remains baseline until successful checks. New epoch is recorded; challenge is issued before this step's Start. |

The job phase can briefly be `checking` while the transaction is still
`switched`; Snapshot derives these stage/state fields from the transaction.
It also exposes generation, job, requested release, health and terminal
recovery evidence. It has **no active transaction ID, live owner or process
liveness field**. Prior terminal/result evidence may also be present; this is
not an unconditional `recovery_required=false` claim for every journal.
The forwarding latch also remains blocked through candidate start and checks.
These durable rows are intentionally identical after an updater crash at the
same boundary. Reading them alone cannot prove a live apply. Raw `Txn` has a
transaction ID, but a nonterminal row still says nothing about liveness.

### Can the candidate identify its parent apply today?

No supported launch token identifies an apply. In
[`CommandServices.Start` / `runArgv`](../../node/updater/host.go):

```go
return runArgv(ctx, c.StartCmd, []string{releaseDir})
// runArgv:
args := append(append([]string{}, argv[1:]...), extra...)
cmd := exec.CommandContext(ctx, argv[0], args...)
cmd.Stdout = io.Discard
cmd.Stderr = io.Discard
```

There is no assigned `cmd.Env`, transaction token, parent-lock descriptor or
special recovery flag. The command inherits its environment and receives the
release directory as its last argument. A configured wrapper may add its own
behavior; systemd can be the actual agent parent. OS parent PID therefore is
not the update protocol.

[`node/nodelock/lock.go`](../../node/nodelock/lock.go) writes, only after flock:

```go
type holder struct {
    PID       int    `json:"pid"`
    BootID    string `json:"boot_id"`
    StartTime int64  `json:"start_time"`
}
```

This identifies a process incarnation and supports stale diagnostics. It has
no role, operation or transaction binding and remains on disk after release.
`ErrBusy` proves live **contention**, not that the holder is this candidate's
updater: the agent, installer and watchdog share the same lock. The existing
public API does not return verified holder metadata. Comparing a filename,
a PID alone or an old journal stage cannot safely establish parent ownership.

Agent-core's current [`recoverOnce`](../../node/agent/engine.go) sets
`e.recovered = true` before `cmd.Run()` and sets `e.recoverBlocked = true` for
both exit 2 and every other error. `cmd.Run` does not capture the updater's
stderr for classification. That explains why exit 1 contention becomes a
latched recovery block even after the parent completes. This is a source
explanation of the supplied F30-01 observation, not a new e2e reproduction.

### Proposed minimal signal — not implemented

For retrying after **any** transient contention, a distinct machine-readable
busy outcome from `recover` would suffice; it must never mean recovery
succeeded. To make the stronger claim **an apply is active**, generic busy is
insufficient. Propose a narrow local status, `apply_in_progress`, carrying the
active transaction ID and current owner incarnation:

- Bind existing lock metadata to `operation=apply` and `txn_id` while holding
  the flock, before starting candidate services (including resumed apply and
  rollback). Keep the existing PID/boot/start-time checks. No marker alone is
  authoritative after the flock is released.
- Expose the actual active transaction ID in the read-only journal snapshot,
  not the most recently selected terminal job's ID. Classify the signal only
  while a live foreign flock and matching owner/transaction are observed;
  unstable, stale, missing or mismatched observations remain unknown/blocked.
  A checked read must account for a release/reacquire race; PID-only matching
  is not enough. The wire shape or a new exit code is for agent-core and the
  orchestrator to agree, not assigned here.
- Agent-core may defer boot recovery and retry after the holder releases the
  lock, while keeping the baseline, meter/publication fences and control/
  health response path. Disappearance/crash of the owner requires ordinary
  recovery of the leftover transaction. The signal is never permission to
  serve, skip recovery permanently, or take over the parent's lock.

This is the smallest *bound* signal proposed here: role plus transaction on
the existing owner record, with a read-only active-transaction view; an
inherited environment boolean or persistent `applying=true` cannot distinguish
crash residue. A simpler generic busy/retry solution remains an agent-core
option if it does not need to claim which process owns the apply.

**D-064 implications:** preserve the foreign-lock requirement through health
checks; releasing the parent lock just to make boot `recover` return zero
would remove the probe-exception authority and break exclusive update work.
[`installProbeException`](../../node/agent/probeexc.go) currently requires a
live matching challenge, `updateLockHeld()`, trusted time and the original
nonce/expiry, and installs/removes only a baseline model. The new signal must
not authorize member serving or replace those checks. As already required in
[the agent contract](../node/README.md#health-attestation-d-053), exposing active
transaction identity also requires matching challenge `TxnID` to it. Keep
that binding synchronized with agent-core; do not expose a new ID and leave
probe validation on the older weaker rule. The new status must cover the
pre-challenge `switched` window without fabricating a health challenge.

## Verification and limits

Required gates passed on the corrected source: updater lint/test/integration/
build (linux/amd64 and linux/arm64), fresh updater/nodelock/installer/installerjob
consumers, unchanged agent meter-barrier interoperability and full agent suites,
handover tests, whole-module lint and direct vet. Integration used the supplied
DSN through `dbtest` per-test databases; sweeping was unset. Root lint used
`go-with-disk-tmp=set -eu;` to preserve exported worktree disk scratch without
the unowned helper's shell cleanup trap; gofmt/vet/pinned staticcheck checks
were unchanged. No dependency changed.

All test scratch is under `.agent/tmp`; the cache was discovered using
`go env GOCACHE` (home-relative `~/.cache/go-build`). No container, network,
volume or image was created; filtered owned inventory is empty. Raw campaign,
control, admission and gate logs are retained in `.agent/tmp/f20/` and the
ignored hand-off report contains real gate output tails. There is no new live
lab, physical arm64 or power-loss measurement. The production 30 s meter wait
and 100 ms polling constants are unchanged; these tests do not establish a
worst-case storage/scheduler or whole-Recover wall-time SLO.

### Requests to orchestrator

- **2026-11-03 — host scheduling / F27-02 measurement (open):** reserve a
  roughly 12-minute window with no other lane suite. Three baseline runs and
  one corrected run each completed 200 iterations, but none could be qualified
  as quiet; the monitored runs were joined by other lanes despite `hold.sh`
  remaining free. Run the qualified before/after quiet and bounded parallel
  comparison using the retained instrumentation/commands, with the unchanged
  full agent suite as the sole intentional load and a combined 15-minute
  active-load ceiling. Preserve all four existing runs and the historical
  failure. This scheduling item is outside updater-owned paths; the code,
  controlled proof, facts and required gates are delivered independently.
- **2026-11-03 — agent-core (F30-01):** decide whether generic busy/retry is
  sufficient or the bound `apply_in_progress` signal above is needed. Own the
  boot recovery retry/terminal-state handling in `node/agent`; coordinate any
  updater/lock/snapshot contract change and D-064 `TxnID` validation before
  implementation. This follow-up makes no F30-01 code change. Validate parent
  completion, parent crash, rollback and unrelated lock contention, and
  re-run the blocked devstack lease-expiry row with the real agent.
