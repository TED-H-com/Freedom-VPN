# Node updater follow-up 24 — 2026-11-03

## Summary and status

D-155 (provisional) selects r29 Option B for F30-01. This change completes the
updater half: `updater.ExitBusy = 3` identifies `UPDATE_BUSY` with detail exactly
`lock`, and lock-taking operations check `update.lock` before opening or migrating
`journal.db`. The agent half remains pending in agent-core. F30-01 stays open
until the agent change lands and an acceptance run passes without the nightly
`lease-expiry` quarantine entry. That entry is unchanged here.

Base: `ff5efb69`. Review: `r29-update-recover-review`, `5b1ca726`, based on
`dd052c60`. The four updater proofs were imported byte-for-byte in `03e24246`
before any fix. The exit change is `a5daf7b3`; journal ordering is `664ce6ca`.
Earlier follow-up documents, consumers, units and acceptance harnesses are unchanged.

## Judgment before fixing (D-141)

H-1's updater half is confirmed. On this base, `Engine.withLock` made five bounded
acquisition attempts, translated a live holder to `OpError{Code: UPDATE_BUSY,
Detail: "lock", Retry: true}`, and `ExitCode` returned 1. The CLI forwarded that
status. `stepStart` starts the candidate while the outer updater still owns the
lock; agent `recoverOnce` treated this exit as a terminal block. Nothing here
narrows that lock, returns false success, restarts the attesting agent after seal,
or introduces a completion marker.

L-1 is confirmed for `Run`, `Recover`, `Flush`, `ApplyPending` and `Seed`: they
called `open` before `withLock`. `open` calls `Layout.Ensure` and
`updatejournal.Open`, which opens SQLite read-write and calls `migrate`
(`node/internal/updatejournal/journal.go:166,208`). A current schema may need no
migration writes, but the open and schema read still preceded the lock.

One r29 detail needs qualification: `Retain` did **not** call `open` on this base.
It returned early when `Journal == nil`, otherwise locking before pruning. It
still never opens a journal. Its nil-journal no-op now runs inside `withLock`, so
all six entry points report contention consistently; with a free lock and no
journal it remains a no-op. The imported proof's historical comments and all its
assertions remain unchanged.

## Exit contract and command matrix

`node/updater/host.go:243` exports `ExitBusy = 3`. `ExitCode` checks nil first,
then `errors.Is(err, ErrMembersBlocked)`, then `errors.As(err, &op)` and the exact
`op.Code == apierr.UpdateBusy && op.Detail == "lock"` pair. Wrapped busy errors
qualify; errors matching both busy and blocked remain 2. Retry flags alone do not
select 3. The four stale details (`lock boot id`, `lock pid`, `lock process`,
`lock start time`), other busy details and other errors remain 1. D-034g remains
true: only `ErrMembersBlocked` maps to 2, and CLI usage errors remain 64.

| Command | Live-holder busy path | Before | After |
|---|---|---|---|
| `run` | `Engine.Run` → `withLock` | 1 | 3 |
| `recover` | `Engine.Recover` → `withLock` | 1 | 3 |
| `apply` | `Engine.ApplyPending` → `withLock`, after argument/path validation | 1 | 3 |
| `seed` | `Engine.Seed` → `withLock`, after required arguments | 1 | 3 |
| `flush` | `Engine.Flush` → `withLock`, with a configured sink | 1 | 3 |
| `status` | None: read-only `updatejournal.Snapshot`, no update lock | 0 on read success; 1 on failure | Unchanged |

The first five rows describe contention after valid configuration and arguments;
configuration/usage errors may still return before lock acquisition. `Retain` is
a library entry point, not a CLI command. Its attached-journal busy error maps
from 1 to 3; its former nil-journal early success now also checks the lock.

No CLI printing code changed. Live contention still emits exactly:

```text
fvpn-updater: updater: UPDATE_BUSY: lock
```

The real CLI test checks the trailing newline, empty stdout, exit 3 for a live
same-boot parent process, and exit 1 plus the unchanged stderr line for each of
the four stale-holder diagnostics. Existing command-specific handling of
`ErrMembersBlocked` stderr is preserved.

## Lock and journal order

The complete list of entry points using `withLock` is `Run`, `Recover`, `Flush`,
`Retain`, `ApplyPending` and `Seed` (`engine.go:106,195`, `result.go:201`,
`retain.go:16`, `pending.go:56`, `seed.go:20`). The five operations that open journals
call `open` only within their locked callback; `Retain` only uses an
already attached journal. No production journal open remains before `withLock`.

Before acquisition:

- The CLI parses flags, loads and validates configuration, and constructs the
  engine (including trust, TLS and probe configuration checks). These operations
  do not open `journal.db`.
- `ApplyPending` checks its required path and resolves it to an absolute path;
  `Seed` checks its required paths; `Flush` rejects a missing sink. These keep
  existing input/configuration failures and do not access the journal.
- `nodelock.tryAcquire` creates the lock parent directory, opens/creates the lock
  file and attempts `flock`. Those steps are required to acquire the lock. Holder
  metadata is written only after successful acquisition.

After acquisition, `open` runs `Layout.Ensure`, then opens/migrates the journal
unless one is already attached. A contended caller neither creates a missing
journal directory nor opens a corrupt database. Supplied journal handles are
kept without inspection under contention. The lock remains held across candidate
start, checks, seal and any rollback. Nested `Run` → `Recover` and seal → `Flush`
or `Retain` reuse the same holder; they do not release or reacquire it.

`status` retains its read-only behavior: seeded snapshot output is identical
with a held lock using either `-config` or `-journal`; a missing journal returns
1 and creates no directory. First-boot `Run` still creates its journal under the
lock, and `Retain` with no journal still leaves it absent when the lock is free.

## Proofs before and after

The command before fixes was:

```sh
go test -count=1 -v ./node/updater -run '^TestR29_'
```

| Imported proof (unchanged) | Before fixes | After fixes |
|---|---|---|
| `TestR29_H1_UpdaterExitCodeDistinguishesLiveUpdateBusy` | FAIL: got exit 1, wanted 3 | PASS |
| `TestR29_L1_ContendedRecoverChecksUpdateLockBeforeOpeningJournal` | FAIL: `journal.db` existed (`stat err = <nil>`) | PASS |
| `TestR29_Held_ContendedRecoverDuringStepStartAndCheckDoesNotMutateTxnOrGate` | PASS | PASS |
| `TestR29_Held_CrashedSwitchedAndCheckingTxnsRecoveredUnderFreeLock` | PASS | PASS |

The baseline exited 1, as expected. Its exact output is in the import commit
message and the local hand-off log. After both fixes, the focused run of
`TestR29_` and `TestD155_` passes. The Held crash proof's name mentions both
switched and checking; its concrete injected crash is at `StageSwitched`, then
recovery observes failed candidate health and completes rollback. Existing AT25
crash tests cover the broader stage matrix.

Additional tests:

- `TestD155_ExitCode`: nil, blocked (including wrapped), live busy (including
  wrapped), both meanings in both join orders, all stale details, other/empty/
  non-exact details, a different code and plain text that resembles busy.
- `TestD155_RecoverCLIExitAndStderrUnderHeldLock`: real compiled CLI versus a
  different live process holding flock; live and all four stale diagnostics.
- `TestD155_AllEntryPointsLockBeforeJournal`: all six entry points with missing,
  corrupt and already attached journals; no creation, migration or handle change
  on contention. Corrupt input remains byte-identical with no WAL/SHM sidecars.
- `TestD155_FirstBootAndNestedLock`: first boot, `Run`'s inner recovery, nested
  flush/retain and continued kernel lock ownership until the outer call returns.
- `TestD155_StatusRemainsReadOnlyUnderHeldLock`: unchanged read-only snapshots
  under contention and missing-journal refusal without creation.

These cover UPD02, UPD03 and UPD05 (§4), exclusive recovery (§13.5), rollback and
forwarding gates (§13.9), and busy diagnostics (§13.11); they supplement AT25
and AT26 (§19). They do not establish live F30-01 acceptance or real arm64 health.

## Consumer audit

Repository searches covered updater executable references, `UpdaterRecover`,
`UpdaterCommand`, exit-code branches, service success-status declarations and
AT wrappers. No consumer was changed. The changed integer may be recorded in
diagnostics, but no existing consumer interprets live-lock exit 3 as success or
as a distinct recovery action before the agent-core change.

| Consumer and code evidence | Treatment of old 1 and new 3 for live contention |
|---|---|
| Agent `recoverOnce`, `node/agent/engine.go:1030` | Only nil and exit 2 have special branches. Both 1 and 3 keep `recovered=true`, set `recoverBlocked=true`, and log the same error. This is why F30-01 remains open. |
| Installer baseline, `packaging/installer/baseline.go:36,80,87,101` | Capture stores the numeric status. Seed requires 0; both 1 and 3 produce `baseline_failed` / installer `ExitFailed`. Probe exceptions are limited to recognized exit-64 help or unsupported-command text; neither 1 nor 3 qualifies. |
| Installer status reconciliation, `packaging/installer/check.go:214` | Any command error returns failure/drift. `status` itself cannot produce lock-busy 3. |
| Production systemd units, `deploy/node/systemd/fvpn-updater.service:11` and `fvpn-updater-recover.service:12`; agent copy, `node/agent/systemd/fvpn-updater-recover.service:12` | All use `SuccessExitStatus=0 2`. Both 1 and 3 fail the unit; none has a special restart rule for 3. Installer `upgrade.go:127` reports a failed unit start as `updater_failed` for either. |
| Devstack launcher, `deploy/devstack/entrypoint-node.sh:171` | Captures any status to `updater-result.json`, then consumes the config the same way. It has no status-specific retry/success branch. |
| Nightly update scenario, `tests/e2e/scenarios_test.go:657,679` | Records the integer; decides using journal/health/candidate/probe evidence rather than 1 versus 3. |
| AT updates runner, `tests/e2e/at/updates/live_test.go:504,615` | Run status is recorded only; explicit flush succeeds only on nil command error. Both contention statuses are failed flushes. |
| AT installer hand-off, `tests/e2e/at/updates/installer_test.go:54,232,269,316,341,350,366` | Seed/run/apply success requires 0 and evidence. Expected terminal failures require exact status **and** checkpoint/outcome/hash evidence; a busy stderr line fails these checks with either 1 or 3. Exit-2 recovery checks reject both. |
| Supplemental AT Python, `tests/e2e/at/updates/supplemental.py:24,31,114` | Readback rejects any nonzero; seed success requires 0 and unchanged/verified evidence. Both 1 and 3 fail. |
| Installer container tests, `tests/installer/docker_test.go:207,212`, `handoff_test.go`, `reporting_test.go`, `lifecycle_test.go` | `mustExec` requires 0. Expected failures also require specific stderr/journal evidence (TLS, result submission, rollback, hash), absent on busy. Exit-2 replay checks reject both. Seed help accepts only 0/64 and help text. |
| Installer unit tests, `packaging/installer/handoff_test.go`, `install_test.go`, `reconcile_test.go` | Seed-support fakes use 64; real status/seed integration requires success or an error, with no special exit-3 branch. |
| Historical live follow-up harnesses, `tests/e2e/followup17_live_test.go:92,353`, `followup18_live_test.go:92,246`, `followup19_live_test.go:85,219` | Nil/non-nil success observations or raw numeric diagnostic capture; subsequent decisions require independent status/checkpoint/health evidence. Neither busy code supplies that evidence. |
| Updater's own CLI/tests, `node/updater/cmd/fvpn-updater/main.go:150`, `main_test.go`, `seed_test.go`, `followup15_test.go`; crash helpers in `node/updater/crash_test.go` | CLI now returns the new mapping without changing stderr. Existing tests expect success, blocked, usage or named non-lock failures; the newly imported H-1 proof intentionally requires 3. The crash child maps engine errors itself (nil→0, blocked→2, other→1); its parent records that status. It never consumes the CLI mapping, so it is unchanged. |

Generic exit-status hits outside this list were inspected: the AT stop observer
wraps service commands invoked **by** the updater, not the updater exit;
`tests/e2e/fleet_snapshot.go`'s exact-1 branch checks offline agent enrollment;
`prefix_test.go` checks the shell prefix-ownership helper; usage/security command wrappers do not classify an
updater lock result. Archived handover fixtures are evidence, not executable
status consumers. Handover documentation checks run in the required gate.

## Agent-core hand-off: lock predicates

`node/agent/probeexc.go:67` returns false if `planeLocked`; otherwise it calls
`nodelock.TryAcquire`, releases a successfully acquired lock, and returns
`errors.Is(err, nodelock.ErrBusy)`. The updater uses
`nodelock.AcquireWithBootID` (`node/updater/engine.go:149`), the same underlying
`tryAcquire` (`node/nodelock/lock.go:51`), with five attempts and a configured
boot-ID fallback when the kernel ID cannot be read. `lockBusyDetail`
(`node/updater/engine.go:1119`) preserves the four stale reasons.

They agree for an ordinary live same-boot holder and for a free lock, but are
**not equivalent predicates**. The agent regards stale-record busy errors as
held too; updater exit 3 excludes their detailed errors and leaves them at 1.
The agent also has the `planeLocked` bypass and no boot-ID fallback in this
probe. Agent-core must account for this when combining pre-spawn deferral with
terminal stale-holder behavior. The lock scope must still span start/check/seal.

### Spec questions

No change to the requested exact `OpError` mapping is needed. One existing
boundary deserves explicit hand-off: `nodelock.tryAcquire` returns plain
`ErrBusy` when flock is held but holder metadata cannot be decoded
(`node/nodelock/lock.go:61-72`). That maps to detail `lock` and therefore exit 3,
as D-155 specifies. It is conservative contention handling, not proof of valid
holder metadata. No lock metadata policy was changed here. Agent-core/orchestrator
should keep that distinction and the broader `updateLockHeld` predicate visible
when deciding stale-holder deferral behavior.

## Validation

All commands use `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and worktree-local
`.agent/tmp` for `TMPDIR`/`GOTMPDIR`. Each admitted test/build/lint command follows
`bash "$FVPN_HOLD_SH"` printing `free`. One attempted consumer gate was refused
with 125 during an orchestrator reservation before any tests ran; it is not a
test result. The later admitted run passed. No reservation or Docker objects
were created by this task.

| Gate | Result |
|---|---|
| `go test -count=1 ./node/updater/... ./node/nodelock/... ./node/internal/updatejournal/... ./node/internal/nodelock/... ./node/internal/healthattest/...` | PASS |
| `go test -count=1 ./packaging/installer/... ./tests/e2e/at/updates/...` | PASS |
| `go test -count=1 ./tests/handover/...` | PASS |
| `make go-lint go-with-disk-tmp=` | PASS; whole-module gofmt, vet and pinned staticcheck v0.8.2 |
| `go vet ./...` | PASS |
| `make updater-build` | PASS; updater and release binaries for linux/amd64 and linux/arm64 |
| Whole-repository `gofmt -l` / `test -z` | PASS; no files listed |
| `git diff --check` and public-repository hygiene pattern scan | PASS |

The `go-with-disk-tmp=` override leaves the root lint checks intact and uses the
already exported disk scratch directory, avoiding the root make helper's shell
cleanup trap. No dependency changes were needed. Runtime acceptance and real
arm64 execution were not run; passing the default AT harness package tests does
not claim a live AT run. Local raw output tails are retained in `.agent/REPORT.md`.


The standalone formatting assertion initially contained an extra literal `)` in
its shell argument and exited 1 despite empty `gofmt -l` output. Its one corrected
rerun passed. This was a command setup failure; no source formatting changed.
There were no code/test failures in the final gates and no outside-owned-path
failure requiring a main comparison.

### Requests to orchestrator

1. Agent-core: consume exported `updater.ExitBusy = 3` (D-155). Stderr is unchanged; live contention is exactly `fvpn-updater: updater: UPDATE_BUSY: lock\n`. Keep forwarding blocked and recovery unfinished while deferring; retry after release, clear only on exit 0, keep exits 1/2 terminal. r29 M-1's background recovery/short synchronous fast path remains agent-core's to avoid boot/health self-deadlock. No consumer was modified here.

   The predicates agree for a normal live same-boot holder but are not equivalent. `node/agent/probeexc.go:67-76` calls `nodelock.TryAcquire` and accepts every `errors.Is(err, nodelock.ErrBusy)`, including stale diagnostics, with a `planeLocked` bypass. `node/updater/engine.go:144-160,1119-1129` calls `AcquireWithBootID` (five attempts, boot fallback), retaining stale details; `node/updater/host.go:243-258` maps only exact `UPDATE_BUSY: lock` to 3. `node/nodelock/lock.go:51-76,145-163,201-227` supplies the shared flock and stale checks. Agent-core must reconcile this broader precheck with terminal stale-holder outcomes. Unreadable metadata also returns plain ErrBusy (lines 61-72); D-155's exact mapping intentionally leaves that existing classification intact.

2. Handover: update `docs/handover/operator-guide.md` section 4.3 without changing earlier historical evidence. Replace the paragraph beginning `**Known behaviour at #407 — F30-01` through `does not accept the behavior.` with exactly:

   > **F30-01 — updater half complete under D-155; agent half pending.** During an update, the candidate agent starts while its parent updater holds `update.lock`. The updater now returns exit `3` (`ExitBusy`) for `UPDATE_BUSY` with detail exactly `lock`, and lock-taking entry points check the lock before opening or migrating the journal. Stale-holder diagnostics remain exit `1`; stderr is unchanged. Until agent-core implements deferral and retry, the candidate's boot `recoverOnce` still treats this busy result as a terminal failure and latches `recovery_required`, so member serving does not resume in that process. F30-01 remains open until the agent half lands and an acceptance run passes without the nightly `lease-expiry` quarantine entry. The devstack restart observation is not a verified production operator procedure. See [updater follow-up 24](../updates/node-followup24-2026-11-03.md).

   Immediately after that paragraph, add exactly:

   > **Updater exit statuses (D-034g, D-155):** `0` means the command completed without an operation error; it does not authorize member forwarding. `1` means another failure, including stale-holder details `lock boot id`, `lock pid`, `lock process` and `lock start time`. `2` means only `ErrMembersBlocked`, and takes precedence if an error also matches busy. `3` (`ExitBusy`) means `UPDATE_BUSY` with detail exactly `lock`: recovery has not completed and can be retried after the live holder releases the lock. `64` means usage or flag error. `run`, `recover`, `apply`, `seed` and configured `flush` can return `3`; read-only `status` takes no update lock and returns `0` on a successful read or `1` on failure. Live busy stderr remains `fvpn-updater: updater: UPDATE_BUSY: lock`.

3. Devstack/orchestrator: keep `tests/e2e/nightly_expected_blocked.json`'s `lease-expiry` entry until agent-core's half lands. Then run acceptance without that quarantine entry and close F30-01 only on a pass. No Docker/live acceptance run was performed in this follow-up. Real arm64 runtime evidence remains later work; cross-compilation is the local build check.
