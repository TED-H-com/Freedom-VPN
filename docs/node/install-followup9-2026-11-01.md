# Installer follow-up 9 — agent stop budget, 2026-11-01

This addresses r18 M-2 and agent-core 32 request 1. The [agent-core shutdown
accounting](followup32-shutdown-hold-observations-2026-11-01.md#m-2-shutdown-accounting)
sets one Step-local deadline of `min(5 s, watchdog_stale/3) + 5 s`, at most
10 s, shared by reads, quarantine, baseline, persistence and final observation.
Close then waits at most 3 s. Thus the cooperative SIGTERM-to-exit budget is
at most **10 + 3 = 13 s**, plus scheduling and context-uninterruptible system
call overhead. `fvpn-agent.service` now sets **`TimeoutStopSec=15s`**, providing
**2 s margin** and systemd's forced-stop backstop. It is not proof that a wedged
kernel task can exit within 15 s, or that shutdown produced an adoptable meter
checkpoint.

Only the agent timeout and watchdog ordering directives change. The watchdog
now has `Before=fvpn-agent.service`; systemd reverses that ordering when both
services stop, keeping the watchdog alive throughout the agent stop. Argument
order in `systemctl stop` alone does not provide this ordering. There is still
no `Requires`, `PartOf` or `BindsTo` between them: stopping or crashing the agent
alone leaves the watchdog running (§17.1). The watchdog has no matching 15 s
timeout: its `Once` work has a caller-cancelable 5 s context and its signal
context terminates the loop. Its existing systemd default stop timeout remains;
the paired command is therefore **not** claimed to have a total 15 s bound.
Kernel lease expiry remains independent of both processes (§16.3).

## Stop-path audit

| Path | Bound established by code | Failure behavior / disposition |
|---|---|---|
| Installer binary replacement: `Install` in `packaging/installer/install.go` → `systemdManager.Stop` in `services.go` → `Env.run` | Previously the CLI supplied `context.Background()` and no stop deadline; library callers could supply a shorter context. Now every production `Stop` gets a fresh **20 s** deadline: the agent's 15 s backstop plus **5 s** for command completion. An already canceled caller issues no command. Once submitted, the stop uses `context.WithoutCancel` so an earlier caller deadline cannot abandon it; caller cancellation is returned after the stop. Stops in `stopOrder` are sequential, with separate allowances. | Previously the replacement loop ignored stop errors. It now persists the stop intent, records `stop-failed` on error/timeout, returns failure and rolls back before extracting the candidate, changing `current`, or replacing units. A journal-write failure is returned along with the stop error. `context.DeadlineExceeded` is retained even if the subprocess returns `signal: killed`. Killing systemctl does not cancel its systemd job; rollback can request the old units again. No new release is reported installed. |
| Updater generated `stop_command`: `packaging/installer/config.go` → `CommandServices.Stop` / `runArgv` in `node/updater/host.go` | The rendered argv remains `systemctl stop fvpn-agent.service fvpn-watchdog.service`. `runArgv` uses `exec.CommandContext` with its caller's context. The production CLI in `node/updater/cmd/fvpn-updater/main.go` creates **`context.Background()`**, and the apply/recovery stop callers pass it through: **no caller deadline** caps the stop. The bounded meter wait creates a child context locally; that deadline does not become the service-stop context. The 30 s HTTP client timeout also does not govern systemctl. | No production context shorter than the unit budget was found, so no updater-owner change is requested. A library caller can supply its own earlier cancellation/deadline. Normal `stepStop` failure is treated as destructive and enters rollback with `START_FAILED` at `stop_services`. This audit does not alter other updater recovery/cleanup error handling. |

The installer timeout applies to production systemd stops generally; the new
durable failure handling here is specifically the binary-replacement loop.
Other units can have longer systemd timeouts, so a 20 s installer timeout is a
reported replacement failure, not evidence their systemd stop job ended. The
unit being stopped is the currently loaded version; installing a new archive
does not retroactively change an older unit's active timeout.

## Regression and measurement scope

`TestPKG01_AT07ShippedAgentStopBudget` checks the source unit, the unit extracted
from the signed fixture archive, and the actual installer destination, including
the `[Service]` section and exactly one `TimeoutStopSec=15s` directive.
`TestPKG01_StopWaitsThroughUnitBudget` uses Go's virtual test clock for a 16 s
successful command, a 1 s caller deadline, the actual 20 s stop deadline, and
preexisting cancellation. These are unit tests, not measured agent shutdowns.
`TestPKG01_AT25ReplaceStopFailureIsJournaled` injects command/timeout failures
and checks the durable record, failure result, unchanged installed state/unit
and current pointer, and absence of candidate extraction.

The privileged `installer-test` suite runs the installed agent under systemd
PID 1 in an isolated Ubuntu 24.04 container. Its lifecycle test uses strace
attached to the actual process to observe SIGTERM and its successful exit, and
container clock readings around `systemctl stop` to measure command return.
It exercises both an agent-only stop (watchdog remains active) and the exact
paired stop from updater config. Systemd monotonic timestamps verify that the
watchdog begins stopping only after the agent exits. No ExecStart, signal
handler or shutdown work is replaced for these measurements.

The test image adds Ubuntu-packaged strace (LGPL-2.1-or-later) to observe actual
signals and process exit without adding tracing code to the production agent.
No Go module or production package dependency changes.

Measured on 2026-11-01, linux/amd64, Ubuntu 24.04 / systemd 255:

| Installed-unit stop | SIGTERM → main process exit (status 0) | `systemctl stop` return |
|---|---|---|
| Agent only | 8.652 ms | 14.395356 ms |
| Agent and watchdog | 13.384 ms | 19.951860 ms |

The joint-stop watchdog transition began **1.046 ms after** agent exit, using
systemd's monotonic timestamps. Agent-only stop left the watchdog active.
Both services reported `Result=success`; the agent's loaded `TimeoutStopUSec`
was `15s`. Command timing excludes Docker exec startup, but includes the local
shell/date measurement overhead. The earlier suite pass measured the
`exit_group(0)` syscall; the table uses the rerun's confirmed process exit.

These are normal stops in the enrollment-only fixture, with tracer overhead and wall-clock
measurement resolution. They do not exercise the worst-case Step plus Close,
forced SIGKILL, an uninterruptible syscall, a serving member flow, or physical
arm64. Those remain 待實測. PKG01 / §6.3 / AT07 and the failure aspect of
UPD02 / UPD05 / §13.6 / AT25 are covered only within these stated scopes.

## Requests to orchestrator

2026-11-01: No requests. No updater, agent, contract or migration file changed.
