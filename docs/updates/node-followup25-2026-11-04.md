# Node updater follow-up 25 — 2026-11-04

## Diagnosis before changes (D-141)

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
the following complete list outside restart/adoption and stop:

| Acquisition | Work while holding the lock | Length bound | Every new generation? |
|---|---|---|---|
| `enforce`, `node/agent/engine.go:1146`, release deferred at 1161 | Reread updater latch; retain/verify an unchanged serving model, or prepare usage, read topology, reserve revocations, install nft/tc, verify again, persist applied state, snapshot counters, resume/sample meter, and publish heartbeat. See 1164, 1172, 1202, 1230, 1257, 1280, 1303, 1350, 1379, 1411, 1429, 1437 and 1441. | The remaining Step local deadline, up to 10 s from Step entry, for cooperative work. Individual serving verification is capped at 1 s; the entire section is not capped at 2 s. Enforcement failure cleanup may outlive the Step deadline, as detailed below. | Yes when a new generation reaches the serving/closed-recovery Apply branch. It also acquires on unchanged serving passes. `canRetainServing` rejects a changed generation (`node/agent/retain.go:33`); a non-serving decision instead takes the baseline branch (`engine.go:1130`). |
| `quarantineForeign`, `node/agent/quarantine.go:139` | If not already `planeLocked`, acquire for `plane.Quarantine`, retain sanitized containment evidence and baseline status (149–164). Under an existing Apply lock it reuses that ownership. | One 5 s quarantine allowance, shortened by the shared local deadline (`quarantine.go:110–127`, `local_budget.go:32`). | No; foreign tc / failed containment path. |
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
allowance is **10 s**. `newLocalBudget` preserves it through local helpers
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
| `recover`, `engine.go:196` | Pre-agent boot service (`deploy/node/systemd/fvpn-updater-recover.service:6,13`); agent Boot/Step deferred child (`node/agent/updaterecover.go:29–97`); nested `Run` recovery (`engine.go:112`). | The same agent child, configured by the devstack agent configuration; manual/AT recovery. | Boot service precedes the agent. Deferred child is a separate process, but recovery blocks the initiating agent's serving decision; it may contend with the outer updater. Manual recovery can overlap. Keep its prompt window under D-155. Nested recovery reuses the outer lock. |
| `apply`, `pending.go:65` | Explicit manual installer hand-off command; the normal installer upgrade starts `run`, not `apply`. | Installer AT invokes the explicit command (`tests/e2e/at/updates/installer_test.go:306`); no ordinary devstack background apply loop. | Yes for the explicit command; it does not stop the serving agent before trying the lock. |
| `seed`, `seed.go:24` | Installer baseline (`packaging/installer/baseline.go:99`), after starting units including the agent (`install.go:255–269`, `constants.go:30–36`); already-current repair releases the installer lock before seeding (install.go:124–125); reenroll similarly (reenroll.go:117–122). | Installer acceptance uses the real installer and direct seed/replay; no ordinary node-launcher seed loop. | Yes, including an automated production path. It cannot be assumed to be a before-agent-only command. |
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
