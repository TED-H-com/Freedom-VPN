# Node updater follow-up 25 — 2026-11-04

## Diagnosis before changes (D-141; original stop, superseded)

This section records the original delivery at `72585ab0`. The orchestrator
accepted the stop and revised D-158 on 2026-11-04 at 02:23 EST. The resume
section below describes the implementation of that revision; historical
statements of unchanged code and deferred work below describe only the stop.

**Verdict: ordinary lock contention is the supported explanation, but the code
does not support a two-second upper bound on ordinary agent holds. Item 2 is
stopped under the brief's explicit stop condition.** No production code or test
is changed. The proposed replacement acquisition budget is **10 seconds**, based
on the agent's cooperative Step deadline, with the limitations below; it needs
an orchestrator decision before implementation. Two seconds would cover the
sampled holds, but that is a narrower, empirical claim.

Base is `3cd48798`. Devstack 34's merged PR body is retained in commit
`4aa1fb0d` (#458); its measured source is `ab0899e6`. There is no difference
between that source and this base in `node/agent` or `node/enforce`.
Evidence below uses repository-relative paths and line numbers on this base.
`run34/` abbreviates `docs/e2e/runs/2026-11-04/followup34/`.

### Agent critical sections

The configured lock defaults to `/var/lib/vpn-node/update.lock`
(`node/agent/config.go:124`). Searching all non-test agent acquisitions gives
the following complete list outside the restart.go adoption and stop paths:

| Acquisition | Work while holding the lock | Length bound | Every new generation? |
|---|---|---|---|
| `enforce`, `node/agent/engine.go:1146`, release deferred at 1161 | Reread updater latch; retain/verify an unchanged serving model, or prepare usage, read topology, reserve revocations, install nft/tc, verify again, persist applied state, snapshot counters, resume/sample meter, and publish heartbeat. See 1164, 1172, 1202, 1230, 1257, 1280, 1303, 1350, 1379, 1411, 1429, 1437 and 1441. | The remaining Step local deadline, up to 10 s from Step entry, for cooperative work. Individual serving verification is capped at 1 s; the entire section is not capped at 2 s. Enforcement failure cleanup may outlive the Step deadline, as detailed below. | Yes when a new generation reaches the serving/closed-recovery Apply branch. It also acquires on unchanged serving passes. `canRetainServing` rejects a changed generation (`node/agent/retain.go:33`); a non-serving decision instead takes the baseline branch (`engine.go:1130`). |
| `quarantine`, `node/agent/quarantine.go:139` | If not already `planeLocked`, acquire for `plane.Quarantine`, retain sanitized containment evidence and baseline status (149–164). Under an existing Apply lock it reuses that ownership. | One 5 s quarantine allowance, shortened by the shared local deadline (`quarantine.go:110–127`, `local_budget.go:32`). | No; foreign tc / failed containment path. |
| `retainClosedRecovery`, `node/agent/recovery.go:47` | Reread latch and verify the unchanged deny-all model under `planeLocked` (51–58). | Serving read timeout at most 1 s, plus any rejection/cleanup within the local deadline. | No; only an already held empty recovery model with provisional grants (30–45). |
| `openUsage`, `node/agent/usage.go:280` | Retire/adopt a different incarnation's usage ledger by reopening it with `RetirePreviousAt` (278–286). Same-incarnation opens do not acquire this lock. | Caller local work context: Step up to 10 s, Boot 10 s. SQLite open/migration uses that context; synchronous filesystem work is not a hard real-time bound (`node/internal/usagestore/store.go:77–90,125–134`). | No; initialization/adoption only. `ensureUsage` stops after `usageOnce` (`usage.go:130`). The lifetime lock at 136 is a different file, `usage.db.lock`. |
| `updateLockHeld`, `node/agent/probeexc.go:70` | One nonblocking attempt, write/sync holder metadata if successful, immediately release at 72. No probe or recovery runs while that descriptor is held. Used by the challenge check (84) and recovery precheck (`updaterecover.go:59`). | No retry timer; filesystem open/metadata fsync only (`node/nodelock/lock.go:51–79,115–130`). | No generation dependency. It may be called on probe or deferred recovery passes. |

For completeness, `restart.go:71` and `restart.go:178` acquire for same-boot
adoption and preservation on stop. **201 attempts at 10 ms bounds their wait
for another holder**, not their own critical section. Adoption evaluates health,
verifies the model/topology and prepares usage after acquisition (84–118).
Boot's local allowance is 10 s and Close's is 3 s (`engine.go:441–456`). These
are not the periodic generation holds in this run.

The watchdog is a separate process, not another agent acquisition. It also
takes this lock for presence checking and possible baseline installation
(`node/watchdog/watch.go:119–163`), using a 5 s context. It is relevant when
attributing an unidentified PID.

### What bounds ordinary Apply

`Step` creates its local deadline at entry as
`controlPassTimeout() + 5*time.Second` (`node/agent/engine.go:680–684`).
`controlPassTimeout()` is `min(5s, watchdogStale/3)` with a 30 s default
(`node/agent/serving.go:246–251`, `config.go:139`). Thus the normal configured
allowance is **10 s**. Devstack configures the same 30 s watchdog stale value
(`tests/e2e/cmd/fvpn-e2e-prepare/main.go:395`). `newLocalBudget` preserves it through local helpers
(`local_budget.go:19–40`). The call at `engine.go:1303` passes that context to
`SystemPlane.Apply` (`plane.go:93–98`).

There is no shorter whole-Apply deadline. For example, ordinary preflight reads
and shaping use the parent context (`node/enforce/apply.go:135,158`), as do
retired-class cleanup, guard release and verification (211,217,231). The
100 ms `DeadlineInstallBound` only bounds the final nft installation
(`node/enforce/model.go:94–97`, `apply.go:190–205`). A successful shaping phase
taking over two seconds is not prohibited by that installation bound. Member
path reads also use the parent context, including route reads per binding
(`node/agent/memberpath.go:37–98`). The later agent serving verification has
its own at-most-one-second timeout (`serving.go:34–49`), not a deadline for the
whole holder.

Accordingly, **propose a 10 s `run` acquisition budget with 20 ms polling** to
cover the remaining cooperative ordinary Step allowance, rather than inferring
a hold deadline from restart's acquisition retries. This is still a bounded
contention policy, not a guarantee that every holder releases: failed enforcement
can start separate 5 s cleanup with `context.WithoutCancel`
(`node/enforce/apply.go:319–334`); kernel completion, process reaping, synchronous
filesystem calls and scheduling can overrun context deadlines. No finite wait
guarantees acquisition against repeated competing holders. Those cases must
still return `ExitBusy` when the chosen acquisition budget ends.

The brief explicitly says to stop item 2 if the agent code does not support
two seconds covering ordinary holds. That condition applies. Neither a 10 s
replacement nor a two-second empirical compromise is silently shipped here.

### Attribution of the samples

The observer reads `/proc/locks` and the heartbeat separately; it does not
capture a call stack, executable or exact acquire/release event
(`tests/e2e/update_observer_test.go:24–35`). The following targeted extractions
include the neighboring samples without the holder:

| `run34/update-recovery-heartbeats.txt` lines | First / last sample with PID 876 (UTC) | Neighboring samples without it | Generation after release |
|---|---|---|---|
| 239–308 | 05:50:22.851057984 / 23.281326209 | 22.629076360 / 23.497303637 | 91 |
| 547–616 | 05:50:27.694129866 / 28.125726771 | 27.474221728 / 28.338727770 | 92 |
| 883–952 | 05:50:32.851205106 / 33.284011965 | 32.638301513 / 33.497407447 | 93 |
| 1205–1274 | 05:50:37.826551424 / 38.260400930 | 37.608095736 / 38.478173310 | 94 |

The repeated agent PID, generation advance and meter epoch openings at
22.714, 27.726, 32.708 and 37.722Z
(`run34/update-recovery-agent.txt:255,263,272,280`) identify these as the
ordinary generation Apply section at `engine.go:1146`, rather than boot
recovery. That attribution is a source/evidence inference, not a sampled stack.
The earlier `policy_missing` baseline writes finish before these holds
(`update-recovery-agent.txt:248–280`). Meter preparation at `engine.go:1257`
and the final heartbeat at 1441 explain an old-generation heartbeat while the
agent owns the lock, followed by the new generation after it releases.

Assuming one uninterrupted hold per cluster, the observed spans are
0.430–0.434 s and the neighboring-free upper intervals are 0.859–0.870 s.
The brief's 0.86 s is approximate; two intervals exceed 0.86 s. Polling cannot
exclude release/reacquire between samples. It also did not capture the failed
`run` acquisition instant (`run34/analysis.md:117–123`). Roughly one collision
in seven is a plausible phase model, not a measured probability; the 80 ms
retry window further distinguishes an initial collision from an eventual busy
exit. `update-good.json:60–65,132` records exit 3 and no local success at
05:50:29Z; `analysis.md:105–108` confirms no candidate/probe.

**PID 185678 cannot be positively identified as update-bad's updater from
the retained records.** It is present from 05:50:34.139868862 through
36.954024990Z (`update-recovery-heartbeats.txt:981–1176`), between update-good's
record at 29Z and update-bad's at 38Z. The latter records exit 1 and
`HASH_MISMATCH` (`update-bad.json:32–45,76–84`), so that attribution is consistent
and likely. However, the observer stores only the lock PID; neither scenario
result stores its updater PID, and the launcher stores only exit status
(`deploy/devstack/entrypoint-node.sh:171–175`). A search of every other run34
file finds no occurrence of 185678. The watchdog's empty log does not identify
its PID or exclude a successful presence read. This is an evidence gap, not
proof of a stale lock or a different cause for update-good.

### History and why earlier runs could pass

`git log -S '5, 20*time.Millisecond' -- node/updater/engine.go` traces the
retry parameters to `ef622895` (#18), where acquisition was at line 240.
`b596c462` (#33) introduced the outer `withLock`, with the same parameters at
`engine.go:115`. `0cf6b8ad` (#253) changed it to `AcquireWithBootID`, retaining
the parameters. It therefore predates #446 by many follow-ups.

The diff of `439dbc0b` (#446) moves `Run`'s `open(ctx)` inside the locked
callback (`engine.go:106–112`; `open` is at 235–246). Thus `run` reaches the
lock before layout setup and SQLite open/migration, instead of after them.
That can change its phase relative to the agent, and puts journal work inside
the successful holder's section; it does not shorten the 80 ms retry window.
It also changes the live busy exit from 1 to 3, not failure into success.

Earlier passes are compatible with this race: invocation is not synchronized
to Apply, and most sampled time has no agent holder. The launcher polls for
the scenario's config every 2 s and consumes it after one run
(`deploy/devstack/entrypoint-node.sh:167–178`); scenario generation and config
preparation have their own elapsed time (`tests/e2e/scenarios_test.go:637–653`).
No retrospective attribution to a particular scheduling change is warranted.
The earlier pass is recorded in `run34/analysis.md:86,92–101`. Production's
unit invokes the same command and accepts only 0/2
(`deploy/node/systemd/fvpn-updater.service:11–12`); its timer is start-relative
`OnUnitActiveSec=5min`, with `AccuracySec=1min`, not an exact five-minute retry
after failure (`fvpn-updater.timer:3–10`).

### Updater entry points and concurrent callers

The five CLI operations dispatch at `node/updater/cmd/fvpn-updater/main.go:139–149`.
All six library lock entry points are listed below. Every one still uses five
attempts with 20 ms intervals (`engine.go:144–160`).

| Entry point / lock site | Production invocation | Devstack / acceptance invocation | Can overlap a serving agent Apply? |
|---|---|---|---|
| `run`, `engine.go:107` | Timer/service above; manual CLI; installer upgrade releases its lock and starts that service (`packaging/installer/upgrade.go:103–127`). | Config-consumer loop (`deploy/devstack/entrypoint-node.sh:167–178`), driven by update scenarios. | Yes, routinely. |
| `recover`, `engine.go:196` | Pre-agent boot service (`deploy/node/systemd/fvpn-updater-recover.service:6,13`); agent Boot/Step deferred child (`node/agent/updaterecover.go:29–97`); nested `Run` recovery (`engine.go:112`). | The same agent child, configured at `tests/e2e/cmd/fvpn-e2e-prepare/main.go:397`; manual/AT recovery. | Boot service precedes the agent. Deferred child is a separate process, but recovery blocks the initiating agent's serving decision; it may contend with the outer updater. Manual recovery can overlap. Keep its prompt window under D-155. Nested recovery reuses the outer lock. |
| `apply`, `pending.go:65` | Explicit manual installer hand-off command; the normal installer upgrade starts `run`, not `apply`. | Installer AT invokes the explicit command (`tests/e2e/at/updates/installer_test.go:303`); no ordinary devstack background apply loop. | Yes for the explicit command; it does not stop the serving agent before trying the lock. |
| `seed`, `seed.go:24` | Installer baseline (`packaging/installer/baseline.go:99`), after starting units including the agent (`install.go:255–269`, `constants.go:30–36`); already-current repair releases the installer lock before seeding (install.go:124–125); reenroll similarly (reenroll.go:117–122). | Installer acceptance uses the real installer and direct seed/replay (`tests/e2e/at/updates/installer_test.go:225–233`); no ordinary node-launcher seed loop. | Yes, including an automated production path. It cannot be assumed to be a before-agent-only command. |
| `flush`, `result.go:205` | Manual result delivery; seal invokes it inside the already-held lock (`apply.go:513`). No separate flush timer found. | AT explicit result delivery (`tests/e2e/at/updates/live_test.go:617`). | Yes standalone, including after a successful candidate. Nested seal has no acquisition wait. |
| `Retain`, `retain.go:17` | Library only, called during seal (`apply.go:516`); no CLI command or independent production caller found. | Engine tests; no launcher invocation. | In production it reuses the outer holder. A standalone library caller can contend; keep the existing window. |

`status` uses a read-only snapshot (`main.go:68`); `help` and `version` never
take this lock (`main.go:25–42`). The installer and watchdog are additional
users of the common lock, not additional updater entry points. The agent's
usage-ledger lifetime lock is a different resource.

The audit therefore also disproves an assumption that only `run` can encounter
a serving agent. `seed` has an automated overlap; `apply` and standalone `flush`
have manual/AT overlap. Because item 2 is stopped, their wait policy is left
unchanged and these findings go to the orchestrator's revised scope decision.

### Current error, journal and logging contracts

Live same-boot metadata maps to bare `nodelock.ErrBusy`; stale metadata wraps it
with one of four reasons (`node/nodelock/lock.go:59–69,145–161`). Unreadable
metadata also conservatively returns bare busy; exit 3 alone is not forensic
proof of the holder's metadata. `lockBusyDetail` preserves the four stale
details (`node/updater/engine.go:1119–1127`); `ExitCode` maps only exact `lock`
to 3, with `ErrMembersBlocked` taking precedence at 2 (`host.go:247–258`).

One further baseline correction: **stale diagnostics currently also retry for
the nominal 80 ms**. `AcquireWithBootID` retries every `errors.Is(ErrBusy)`
result (`node/nodelock/lock.go:201–226`). D-158's immediate stale return would
be new behavior for `run`, not behavior already implemented. It should not
silently change `recover` or the common library's other consumers.

L-1 remains satisfied: `open` is inside each lock callback; nested callers
bypass acquisition with `lockHeld` (`engine.go:145–148`). No error formatting
changes are made. Live contention still emits exactly:

```text
fvpn-updater: updater: UPDATE_BUSY: lock
```

The CLI has JSON slog records separate from that error line, but uses the same
stderr stream (`main.go:136,151–154`). **There is currently no acquisition-wait
record** in `withLock`; none is added by this diagnosis-only delivery.

## Implementation stop and proof disposition (superseded by resume)

Item 2 was not started because of the code-supported budget mismatch described
above. There is no new budget constant, retry loop, log event or test assertion.
All production behavior, D-034g, L-1, nested lock reuse and existing tests remain
unchanged. No existing assertion was changed to accommodate a longer wait;
the r29 and follow-up 24 test files contain no assertion pinning `run` to 80 ms.

The requested D-158 proofs have the following explicit disposition. The source
predictions are not represented as executed before/after evidence:

| Requested proof | Before: source prediction / existing evidence | After this delivery |
|---|---|---|
| Real live holder releases after about 600 ms; `run` proceeds | Current nominal 80 ms retry delays would return busy before release. The requested new deterministic test was not run. | Not implemented or run; no passing claim. |
| Real live holder exceeds the new budget; exit 3 no earlier than budget | Current holder exclusion and exit 3 are covered by existing tests; there is no new-budget timing assertion. | Not implemented or run. |
| Each stale detail; exit 1 well before new budget | Existing real CLI `recover` test checks all four details and exact stderr; common acquisition currently spends its nominal 80 ms retries. No new `run` timing proof. | Not implemented or run. |
| `recover` stays within its current live-holder window | Existing r29/follow-up 24 contention tests retain their original exit/journal assertions; no new timing assertion. | Original tests unchanged; no D-158 timing proof claimed. |
| Cancellation promptly ends the longer wait | The current wait selects on `ctx.Done()` (`node/nodelock/lock.go:218–224`). | No longer wait implemented or tested. |

These are intentional omissions under the requested stop branch, not harness
failures or successful proofs. Existing regression results and the required
gates are recorded below after completion. **F30-01 remains open** until a
future acceptance run reaches the real candidate path and passes. No live run,
container, reservation or physical arm64 measurement is part of this delivery.

## Original decisions and remaining work (superseded by resume)

- §13.5 exclusion and §13.11 bounded contention remain intact. The diagnosis
  relates to UPD02/UPD03/UPD05 (§4) and the AT25/AT26 (§19) regression paths;
  it supplies no new live acceptance coverage.
- A revised D-158 decision must select between a 10 s acquisition policy tied
  to the cooperative ordinary Step allowance and an explicitly empirical 2 s
  policy tied only to the sampled holds. Neither should claim a maximum lock
  occupancy or guaranteed acquisition. Polling at 20 ms is the proposal, not
  an implemented constant.
- The separate updater JSON log stream can carry elapsed wait and acquisition
  outcome without changing the fixed error line. The current CLI routes both
  kinds of records to stderr; no separate log file is opened by the updater.
- No agent critical section was shown to hold the lock longer than its required
  work. Latch rechecks, usage attribution, topology/serving verification, durable
  applied state and the final heartbeat all have ordering obligations under
  this lock. In particular `engine.go:1441` intentionally publishes the renewed
  expiry before release. No agent-core shortening request is justified here.
- Spec question: §13.11 permits waiting or bounded retries but supplies no
  numeric acquisition budget. D-158's revised duration and the treatment of
  overlapping standalone `seed`/`apply`/`flush` therefore remain policy decisions.
- The sample trace is not a lock-duration benchmark; D-133 confounding is
  recorded in `run34/analysis.md:35–39`. A larger retry budget alone cannot
  supply the missing candidate-start, deferred-recovery-clearance and first
  candidate-service evidence.

The original requests for a revised duration and caller scope were answered by
D-158's revision on 2026-11-04 at 02:23 EST. Current devstack and operator-guide
hand-offs are under Requests to orchestrator in the resume section below.

## Original validation on the unchanged runtime (stop delivery)

Every gate was preceded by a separate `bash "$FVPN_HOLD_SH"` command returning
`free`. No admission deferral, reservation or harness rerun occurred. Commands
used `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, and worktree-local `.agent/tmp` for
both `TMPDIR` and `GOTMPDIR`. GOCACHE was read with `go env GOCACHE` and was not
overridden. Root lint used `make go-lint go-with-disk-tmp=` to retain the selected
disk scratch directory without invoking the makefile's cleanup trap.

| Command | Result |
|---|---|
| `go test -count=1 ./node/updater/... ./node/nodelock/... ./node/internal/updatejournal/... ./node/internal/nodelock/... ./node/internal/healthattest/...` | Exit 0; output below. Includes all four unchanged r29 proofs and all five follow-up 24 `TestD155_` tests. |
| `go test -count=1 ./node/agent/... ./packaging/installer/... ./tests/e2e/at/updates/...` | Exit 0; output below. |
| `go test -count=1 ./tests/handover/...` | Exit 0; output below. |
| `make go-lint go-with-disk-tmp=` | Exit 0; no output. Uses the existing pinned staticcheck v0.8.2. |
| `go vet ./...` | Exit 0; no output. |
| `test -z "$(gofmt -l $(git ls-files '*.go'))"` | Exit 0; no output. |
| `git diff --check` | Exit 0; no output. |
| `make updater-build` | Exit 0; updater and release binaries compile for linux/amd64 and linux/arm64. |

Real test output:

```text
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	19.859s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	1.599s
?   	github.com/TED-H-com/Freedom-VPN/node/updater/fsutil	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/nodelock	0.032s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal	0.713s
?   	github.com/TED-H-com/Freedom-VPN/node/internal/nodelock	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/healthattest	0.051s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	198.232s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.309s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	9.067s
?   	github.com/TED-H-com/Freedom-VPN/packaging/installer/cmd/fvpn-node-install	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates	0.026s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates/stopobserve	0.083s
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	3.939s
```

Real cross-build output:

```text
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-updater ./node/updater/cmd/fvpn-updater
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-updater ./node/updater/cmd/fvpn-updater
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-release ./packaging/release/cmd/fvpn-release
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-release ./packaging/release/cmd/fvpn-release
```

No outside-path gate failed, so no main-only failure comparison was needed.
Only this new dated note and `docs/updates/node-updater.md` are changed;
production code, tests, dependencies, earlier follow-up documents and other
lanes' files are unchanged. The public-hygiene pattern check has no matches.
Cross-compilation is not evidence of an arm64 device run or live F30-01 closure.

## Resume — 2026-11-04: revised D-158 implemented

The orchestrator accepted the original stop and revised D-158 at 02:23 EST:
**10 s / 20 ms for `run`, `seed`, `apply` and standalone `flush`**. This section
supersedes the original deferral, proof omissions and proposed hand-offs above.
The worktree was clean at `72585ab0` when resumed. No earlier follow-up document
was edited.

`node/updater/lock_wait.go:13` defines the single `updateLockWaitBudget` constant
as 10 seconds. `withLockFor` selects it for the four commands; `withLock` keeps
the existing five attempts at 20 ms for `Recover` and standalone `Retain`
(`node/updater/engine.go:144–172`). The `lockHeld` bypass precedes either
acquisition path, so seal's Flush/Retain and Run's inner Recover wait for nothing.
Journal open remains inside the locked callback. Lock scope across start,
check, seal and rollback is unchanged.

The new helper performs real single-attempt `AcquireWithBootID` calls, using
monotonic elapsed time to bound live contention. Cancellation is checked before
attempts and while waiting on the timer. The common `node/nodelock` and
`node/internal/nodelock` APIs and their agent, watchdog and installer consumers
are untouched. Unreadable metadata remains conservatively live busy.

The first stale answer starts the original **five-attempt** policy, including
that answer; four further attempts are separated by 20 ms. Earlier live
attempts do not consume this window, and a later successful acquisition still
proceeds. As in the existing lock library, exhaustion returns the last error.
Thus a retained stale detail stays exit 1 after its prompt window, rather than
waiting out the remaining live budget. The stale window is counted from its
first answer even if that answer arrives near the live deadline; it is not
restarted on each stale retry. Synchronous filesystem operations and scheduling
can exceed nominal timer durations; this is not a maximum lock-occupancy SLA.

Only the four budgeted commands emit a new record, and only if their first
attempt fails. One `update lock wait` slog event records `command`, `waited_ms`,
`acquired` and `budget_ms=10000` when acquisition ends, before journal/command
work. The CLI's existing JSON handler writes it to stderr. An acquired lock
does not imply that subsequent command work succeeds. The terminal error line,
D-034g and the exit table are unchanged. Recover's exact stderr is unchanged.

### Before/after proofs

Host admission initially deferred for another lane and an orchestrator
reservation. To continue useful preparation, the original four changed source
files were saved byte-for-byte from `72585ab0`. The before run used Go's file
overlay for those files and excluded the new `lock_wait.go`. `GOFLAGS` passed
the same overlay to the compiled CLI child build. The after run used no overlay.
Both runs used identical new proof bodies, disk temporary directories and a
separate admission returning `free`. No harness failure or harness rerun occurred.

The command for both was:

```sh
go test -count=1 -v ./node/updater ./node/updater/cmd/fvpn-updater -run '^TestD158_'
```

The parent starts a real subprocess holder and waits for its readiness line
after durable metadata is written. It releases that holder after 600 ms.
The CLI tests use a real parent flock and a separately compiled updater child.
All holders and timers are cleaned up; no Docker or host networking is used.
The real-default test uses the actual 10 s budget; no shorter-budget seam was
needed. Times below are observations from these offline proof runs, not live
acceptance or performance bounds.

| Proof | Before (`72585ab0` runtime) | After (`ae690cfe` implementation) |
|---|---|---|
| `TestD158_CommandsProceedAfter600msHolder/run` | FAIL: exit 3 at 81.071 ms | PASS: exit 0 at 617.335 ms; acquired record, no-work Run reaches its ordinary completion. |
| Same proof, `seed` | FAIL: exit 3 at 81.403 ms | PASS: exit 0 at 617.479 ms; verified baseline read back. |
| Same proof, `apply` | FAIL: exit 3 at 81.243 ms | PASS: exit 0 at 664.315 ms; staged release becomes current and pending record is consumed as succeeded. |
| Same proof, standalone `flush` | FAIL: exit 3 at 81.343 ms | PASS: exit 0 at 615.934 ms; standalone journal/outbox read completes. |
| CLI `real default budget` | FAIL: exit 3 at 86.265 ms, before the required budget | PASS: exit 3 at **10.006258811 s**, logged `waited_ms=10000`, `acquired=false`; exact live-busy error line. |
| `TestD158_StaleDetailsKeepPromptWindow` | PASS: all four details on all four commands, about 81–82 ms | PASS: all 16 combinations, about 81 ms; exit 1, exact detail, no journal open. |
| `TestD158_StaleWindowStartsAtFirstStaleAnswer` | FAIL: live-busy exit 3 at 81.006 ms, before metadata changes | PASS: metadata becomes stale at about 300 ms; exit 1 at 385.748 ms, preserving four additional retry intervals. |
| `TestD158_RecoverAndRetainKeepPromptWindow` | PASS: recover 81.252 ms; Retain 81.611 ms | PASS: recover 81.221 ms; Retain 81.300 ms; exit 3 and no wait record. |
| `TestD158_NestedCallsDoNotWaitOrLog` | PASS | PASS: Run/Recover/Flush/Retain finish under the existing holder, no wait log, real kernel ownership retained until the outer return. |
| `TestD158_CancellationAndUnreadableMetadata` | FAIL for both live and unreadable metadata: busy at about 81 ms before cancellation | PASS for both: context cancellation at about 300 ms ends the live wait, `acquired=false`, no journal open. Unreadable metadata did not select the stale window. |
| CLI `uncontended`, plus all four engine replays | Existing CLI no-work line PASS | PASS: no wait record; the pre-existing no-work JSON line is unchanged except its timestamp. All four commands omit a wait record on a free lock. |
| CLI `released` | FAIL: exit 3 at 86.288 ms | PASS: exit 0 at 622.663 ms; one JSON record (`acquired=true`) followed by the original no-work diagnostic. |
| CLI four stale cases / record fields | Exit 1 and original error lines; FAIL because the new record is absent | PASS: one record each at 80–81 logged ms, `acquired=false`; each original error line byte-identical. |
| CLI `acquired operation error` | FAIL: contention prevents reaching the operation error | PASS: after acquisition, the missing-pending operation's error line is byte-identical to its uncontended version; the sole added record says `command=apply`, `acquired=true`. |

Real output tails:

```text
FAIL    github.com/TED-H-com/Freedom-VPN/node/updater                  2.288s
FAIL    github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater  1.043s
```

```text
ok      github.com/TED-H-com/Freedom-VPN/node/updater                  5.224s
ok      github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater  12.033s
```

These baseline failures reached their subjects and are expected red proofs.
The earlier r29 and follow-up 24 test files remain byte-identical to `72585ab0`;
none pins an 80 ms window for the newly budgeted commands. In particular the
unchanged all-entry-points L-1 test now waits the real 10 s for each live-held
run/seed/apply/flush case instead of weakening its journal assertions.

### Consumer audit

| Consumer checked, unchanged | Why the new record is compatible |
|---|---|
| Installer `captureCommand` / `seedBaseline`, `packaging/installer/baseline.go:36–62,79–102` | Captures the complete stderr string and numeric exit. Only the separate `seed --help` probe parses help text; it never takes the lock. Real seed success is decided by exit 0, so a preceding JSON wait record is diagnostic data. Existing `TestD089_InstallAndReenrollSeedThroughInstalledBinary`, `TestD089_RealUpdaterConsumesWriterRecord`, and `TestD089_CheckAfterJournaledSwitch` are in the consumer gate. |
| Agent recovery child, `node/agent/updaterecover.go:74–85,102–115` | Uses `exec.ExitError` and exact statuses 3/2, without parsing updater stderr. Recover keeps its original window and gains no wait record. Existing `TestD155_RecoverExitAndRetry`, `TestD155_StaleHolderIsTerminal`, `TestD155_RecoveryLocalBudgetSingleChildAndJoin` and `TestD155_CrashedTransactionsKeepAgentRunning` pass unchanged in the consumer gate. |
| Update AT launcher/results, `tests/e2e/at/updates/live_test.go:504–509,617`; installer AT capture, `installer_test.go:53–69,268–270,303–317` | Retains command output and status; success/failure predicates require status plus independent journal/health evidence, not an exact all-stderr string. These live-tagged bodies were source-audited and compiled with `-tags=e2e -run '^$'`; the required untagged package gate passes unchanged. Neither command executes Docker acceptance. No live AT pass is claimed. |

### Resume validation

All required gates ran sequentially in the foreground after their own admission
command printed `free`. Initial admission deferrals were waited out before the
proof runs; none was treated as a test result. No reservation, Docker object,
harness failure or harness rerun was needed. All started processes were joined.
The environment remained `CGO_ENABLED=0`, `GOTOOLCHAIN=local`, with `.agent/tmp`
for `TMPDIR` and `GOTMPDIR`; GOCACHE was read, not overridden.

| Command | Result on the implementation |
|---|---|
| `go test -count=1 ./node/updater/... ./node/nodelock/... ./node/internal/updatejournal/... ./node/internal/nodelock/... ./node/internal/healthattest/...` | Exit 0, including all four r29 proofs and all five follow-up 24 tests unchanged. |
| `go test -count=1 ./node/agent/... ./packaging/installer/... ./tests/e2e/at/updates/...` | Exit 0; the audited consumers pass unchanged. |
| `go test -count=1 ./tests/handover/...` | Exit 0. |
| `make go-lint go-with-disk-tmp=` | Exit 0, no output; root gofmt, vet and pinned staticcheck v0.8.2. The empty make helper retains the disk scratch environment without a shell cleanup trap. |
| `go vet ./...` | Exit 0, no output. |
| `test -z "$(gofmt -l $(git ls-files '*.go'))"` | Exit 0, no output. |
| `git diff --check` | Exit 0, no output. |
| `make updater-build` | Exit 0; linux/amd64 and linux/arm64 updater/release binaries. |
| `go test -count=1 -tags=e2e -run '^$' ./tests/e2e/at/updates/...` | Exit 0; compile-only check of the live-tagged AT consumers. No tests or Docker operations run. |

Real output from the three required test gates:

```text
ok  	github.com/TED-H-com/Freedom-VPN/node/updater	166.362s
ok  	github.com/TED-H-com/Freedom-VPN/node/updater/cmd/fvpn-updater	13.575s
?   	github.com/TED-H-com/Freedom-VPN/node/updater/fsutil	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/nodelock	0.033s
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/updatejournal	0.720s
?   	github.com/TED-H-com/Freedom-VPN/node/internal/nodelock	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/node/internal/healthattest	0.052s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent	146.802s
ok  	github.com/TED-H-com/Freedom-VPN/node/agent/cmd/fvpn-agent	0.301s
ok  	github.com/TED-H-com/Freedom-VPN/packaging/installer	7.963s
?   	github.com/TED-H-com/Freedom-VPN/packaging/installer/cmd/fvpn-node-install	[no test files]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates	0.072s
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates/stopobserve	0.083s
ok  	github.com/TED-H-com/Freedom-VPN/tests/handover	5.759s
```

Real cross-build and AT compile output:

```text
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-updater ./node/updater/cmd/fvpn-updater
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-updater ./node/updater/cmd/fvpn-updater
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o bin/linux_amd64/fvpn-release ./packaging/release/cmd/fvpn-release
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=arm64 go build -trimpath -o bin/linux_arm64/fvpn-release ./packaging/release/cmd/fvpn-release
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates	0.008s [no tests to run]
ok  	github.com/TED-H-com/Freedom-VPN/tests/e2e/at/updates/stopobserve	0.002s [no tests to run]
```

No outside-path gate failed, so no comparison run on main was needed. The
original exit table and D-034g text were checked against `72585ab0` and are
unchanged. Only owned updater code/tests and the two follow-up 25 documentation
files changed; no dependency, common lock API, earlier follow-up document or
other lane's file changed. The longer all-entry-points L-1 test is expected:
its live-held budgeted commands now exercise the actual ten-second policy.
Cross-compilation and offline tests do not establish arm64 device performance
or close F30-01.

### Requests to orchestrator

2026-11-04 (resume):

1. Devstack's next authorized acceptance run should retain the new JSON record
   and its associated error line for **each** invocation. The exact record
   shape, from the compiled CLI proof, is:

   ```json
   {"time":"2026-11-04T02:41:23.589840709-05:00","level":"INFO","msg":"update lock wait","command":"run","waited_ms":616,"acquired":true,"budget_ms":10000}
   ```

   A real default-budget failure emitted:

   ```text
   {"time":"2026-11-04T02:41:33.598565271-05:00","level":"INFO","msg":"update lock wait","command":"run","waited_ms":10000,"acquired":false,"budget_ms":10000}
   fvpn-updater: updater: UPDATE_BUSY: lock
   ```

   `command` is `run`, `seed`, `apply` or `flush`; times/elapsed values vary.
   No wait record means the first attempt succeeded (or execution did not reach
   a failed acquisition), not missing evidence of a retry. `acquired=true`
   proves acquisition only, not candidate success. `recover`, `Retain` and
   nested calls do not add this event. **`deploy/devstack/entrypoint-node.sh:171`
   overwrites `updater.out` at every invocation**, so preserve it before the
   next update scenario. No launcher or AT predicate is changed here.
2. Handover's exact replacement operator-guide sentence:
   **“Under D-158, `fvpn-updater run`, `seed`, `apply` and standalone `flush`
   wait for live `update.lock` contention with a 10-second budget and 20 ms
   polling before exit 3; `recover` and standalone `Retain` keep their prompt
   five-attempt window, and stale-holder details retain exit 1 after five
   attempts counted from the first stale answer.”**
3. The orchestrator schedules the next devstack candidate-path run.
   **F30-01 remains open** until acceptance reaches the real candidate and
   passes. The unit proofs and cross-compilation cannot close it. No agent-core
   request is added: this change establishes no excessive agent hold.
