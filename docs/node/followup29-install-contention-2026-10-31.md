# Install overruns and admitted-sibling withdrawal — 2026-10-31

## Historical evidence and the question being measured

The [devstack 21 input extract](../e2e/followup21-members.json) retains twelve
original generation-13 Applies. The eight failures (411, 417, 423, 429, 435,
441, 447, 453) have render→installed intervals of 238.823403, 184.488440,
292.382545, 244.369512, 268.957078, 304.391743, 272.238447 and 285.659003 ms.
Each has `install_ok=false`, no guard release and no verification records.
That establishes a failed install stage; it does not retrospectively identify
its command error, scheduling delay or injection's timing.

The [isolation 10 first attempt](../e2e/at/isolation/first-attempt.md) separately
retains 49 failed Applies, all reaching install, and 279 renewal losses
(29 inside completed guards, 250 outside). Follow-up 28's 152.125303 ms failure
and later quiet passes are a third population. None of these historical runs
had contemporaneous per-Apply host context. A new controlled reproduction must
remain separate from those original records. F11-01's unobserved interval is
after inner member egress / before observable inner node ingress; transport
investigation belongs to netlab and is not extended here.

## What Apply failure actually does

[`SystemPlane.Apply`](../../node/agent/plane.go) calls
[`enforce.Apply`](../../node/enforce/apply.go). It shapes first, samples the
final render time, and calls `NftBatch` with a 100 ms context. The candidate
and [`fv_deadline_guard`](../../node/enforce/deadline_install.go) are one atomic
transaction. The guard drops the entire member interface in both directions.
If the command errors, the clock goes backward, or render→installed exceeds
100 ms, Apply calls `installBaseline` and returns without calling
`releaseDeadlineGuard` or the installed-policy verifier. A late candidate
commit cannot expose its relative timeouts by releasing that guard. An
unreleased *operation* does not mean the guard chain must remain forever:
the rollback baseline can replace/remove it while keeping forwarding closed.

`installBaselineModel` freshly reads kernel state, renders the **node-wide
baseline**, preserves counters, and removes owned HTB roots when ownership
checks permit it. It does **not** restore the old generation. Restoring the
old policy could restore the authorization that this generation is revoking.
The admitted sibling loses forwarding as part of this node-wide closure.
This is the existing fail-closed design in §8.2 / §10.3, not a verified
per-member availability guarantee. §11.5 forbids acknowledging failed QoS as
applied. The 100 ms install budget is unchanged; its appropriateness under
contention and the outer-guard ceiling remain netlab's enforcement decisions.
There is no CPU-contention exemption: the current design deliberately rejects
a late install regardless of why it exceeded the budget.

The [`Engine` Apply result path](../../node/agent/engine.go) emits the original
TimingEvidence, then the recorded `apply_rollback` baseline commands.
[`rejectServing`](../../node/agent/serving.go) records a sanitized Apply error,
withdraws with an independent bounded cleanup context and publishes a
non-serving state. Ordinary executor failures become
`HEALTH_CHECK_FAILED/stage=apply`; this is **not a failed tunnel challenge or
an installed-policy verification result**. Later `verification_HEALTH_CHECK_FAILED`
baseline rows name this withdrawal cause; their names do not manufacture
missing verifier records. A nil-error command exceeding the elapsed budget
instead produces the enforcement `CheckError` for the budget failure.

Recovery requires a later successful Apply and live checks using current
signed authorization. A completed guard bounds that install/release operation,
not the whole outage. Repeated failed Applies can prolong baseline indefinitely;
there is no fixed outage upper bound derived from the 100 ms command budget.
No Apply/rollback ordering, deadline, acceptance predicate or production
forwarding behavior is changed by this follow-up.

## Additive TEST ONLY producer context

With both `test_mode=true` and `test_timing_path` set, each Engine Apply row
now adds `host.before_apply` and `host.after_apply`. Each carries actual
CLOCK_MONOTONIC `start_ns` / `end_ns`, `ok`, and `/proc/stat` aggregate
`cpu_total_ticks`, `cpu_idle_ticks`, `cpu_iowait_ticks`, `procs_running` and
`procs_blocked`. Nanoseconds and all kernel counters are decimal strings.
Total excludes double-counted guest/guest_nice; idle excludes iowait.
For valid ordered nondecreasing samples, interval idle percent is
`100 * delta(cpu_idle_ticks) / delta(cpu_total_ticks)`; a zero total delta or
invalid sample has no percentage. `ok=false` means unavailable, never quiet.
The proc view is kernel-visible aggregate, **not cgroup entitlement**.

Samples bracket the entire Apply, outside enforcement's measured operation;
they are neither a render-time instantaneous sample nor a scheduler causal
proof. No proc reads, waits or JSON serialization are added inside the guard.
When the returned error carries a command/exit status, `host.failed_command`
(`nft`, `tc`, `other`), `exit_code` and numeric `signal` retain only that
classification. Error strings, command output, selectors, signed bodies and
packet contents are never exported. Missing exit fields are unknown, not exit
zero. Existing rows, sequence numbers, timing objects, wire evidence and D-106
canonical digest rules are preserved.

For AT02, retain every failure and every lost admitted packet. The diagnostic
label may be **“fail-closed install overrun with recorded host contention”**
when joined install bounds and measured host samples establish those facts.
It must not say “caused by CPU load” solely because runnable/idle observations
coincide. Preserve a separate cause status; the controlled experiment below
can establish reproducibility under its intervention, not the historical
cause. A source drop plus a failed admitted-sibling positive control still
fails AT02. Do not drop the failure from totals or turn baseline into a
completed guard.

## Controlled measurements: the requested overrun did not reproduce

The [retained result artifact](followup29-install-contention-2026-10-31.json)
contains every completed-guard vector, all failed-Apply timing/host/bracket
records, per-leg source identity and host samples. Raw traces, foreground
harnesses, logs and checksum manifest are retained in the worktree under
`.agent/f29-evidence/` and `.agent/f29-gates/`. These are new TEST ONLY runs,
not substitutes for the original AT evidence.

The complete comparison ran **quiet Q2 → loaded L2 → quiet Q3 (agent) /
Q3r1 (live repeat) → loaded L3**, on the same source `f7237e35`. Both suites
ran in every leg. Each loaded leg used 96 foreground Python busy-loop workers
on the host's 24 logical CPUs. No affinity, priority, cgroup settings or other
processes were changed. No other gate/AT/go-test chain was observed during the
loaded legs. `GOMAXPROCS=4` and `GOFLAGS=-p=4` were identical across these lab
runs; the required repository-wide `go test ./...` separately used defaults.

Quiet qualification used `vmstat 5 7`: the six interval samples, excluding the
since-boot first row, each had `r <= 3` and `id >= 80`, plus no other suite.
The original Q3 live postflight reached r=8 and is retained but excluded from
the strict quiet comparison; Q3r1 reran it after a qualifying preflight and
also qualified afterward. Rejected preflights are retained, not test runs.
A redundant Q3r2 preflight was stopped before it launched any test or load.

| Population | Runnable range | Idle % range | D-state threads at samples | Load average observation |
|---|---:|---:|---|---|
| Q2 before/after suites | 0–3 | 86–100 | 11 / 8 / 4 | 26.51 initially; 12.24 after agent |
| L2 under load | 96–101 | 0 | 0 | 89.84 after live; 100.18 after agent |
| Q3 agent pre/post | 0–3 | 86–87 | 2 / 8 | 14.03 / 10.94 |
| Q3r1 qualified pre/post | 0–3 | 82–87 | 12 / 12 | 9.34 / 11.01 |
| L3 under load | 96–102 | 0 | 0 | 90.16 after live; 102.69 after agent |

Load averages are lagging observations, not a quiet criterion. The raw host
records include the three load averages, interval ranges, D-state counts,
monotonic timestamps and suite checks. Q3's first preflight qualified despite
a 101.33 one-minute average left by the preceding load.

All durations below are milliseconds, in **p50 / p90 / p99 / max** order,
using nearest-rank percentiles over all completed guards. Outer means
`release_end - install_start`; install means `installed_by - install_start`;
render→installed means `installed_by - render_start`. Completion alone does
not imply subsequent verification success. Uncompleted guards are never
included in these distributions; their original failed rows are retained.

| Leg / suite | Guards | Outer | Install | Render→installed |
|---|---:|---|---|---|
| Q2-live | 117 | 30.759 / 39.582 / 45.418 / 47.626 | 16.565 / 22.162 / 26.203 / 28.327 | 16.628 / 22.224 / 26.267 / 28.384 |
| Q2-agent/suite | 135 | 28.888 / 37.818 / 44.206 / 47.524 | 14.861 / 19.580 / 26.833 / 27.881 | 14.937 / 19.656 / 27.240 / 27.986 |
| L2-live | 117 | 40.567 / 48.947 / 52.951 / 52.959 | 20.538 / 25.101 / 29.952 / 32.951 | 20.593 / 25.996 / 30.003 / 33.006 |
| L2-agent/suite | 144 | 42.272 / 48.923 / 53.063 / 53.940 | 21.927 / 25.956 / 29.932 / 30.939 | 22.005 / 26.069 / 30.017 / 31.001 |
| Q3-agent/suite | 136 | 28.838 / 37.088 / 46.368 / 49.306 | 14.674 / 21.269 / 24.440 / 28.238 | 14.748 / 21.340 / 24.589 / 28.657 |
| Q3r1-live | 117 | 28.548 / 38.099 / 44.973 / 46.151 | 13.699 / 21.067 / 27.408 / 28.873 | 13.780 / 21.189 / 27.459 / 28.944 |
| L3-live | 117 | 39.823 / 45.959 / 50.950 / 51.946 | 19.950 / 24.953 / 27.120 / 27.421 | 19.998 / 25.006 / 27.156 / 27.483 |
| L3-agent/suite | 141 | 40.910 / 46.912 / 49.915 / 49.931 | 20.886 / 24.938 / 28.915 / 29.913 | 20.998 / 25.013 / 28.996 / 29.999 |

There were **zero failed installs or 100 ms install overruns** in these legs.
Q2 and the selected Q3/Q3r1 suites passed. L2 live passed; its agent suite
failed `TestLabExternalMeterReset`. L3 live passed; its agent suite failed
`TestLab04_RestartKeepsVerifiedLease`. These stress failures remain failures;
they are not relabeled as install overruns. Full-suite streams contain 3, 3,
3 and 4 failed Applies respectively for Q2, L2, Q3 and L3. Most fail before
install in the suite's injected fault phases. L3 Apply 20 reached install
successfully (22.008406 ms render→installed, 41.905056 ms completed guard)
then rejected a foreign tc filter during verification and quarantined. Its
retained bracket is not evidence of the historical install failure mechanism.
Each live-suite stream also retains four pre-install failed Applies.

Preliminary Q1/L1 used zero/24 busy workers. Live passed both times; the agent
suite failed restart in Q1 and expiry correlation plus restart in L1. Their
agent traces were captured only through the assembled D-104 test, so they are
explicitly partial, not full-suite distributions. This exposed a harness gap:
`TestMain` now retains the stream from initial configuration through all tests,
even when a test fails. Existing assembled snapshots and rows are preserved.

## Two-source loaded fixture and F11-03 limits

The opt-in fixture first proves forwarding for two independent TEST ONLY
sources, then changes generation 12→13 to retain one member and revoke the
other. It makes up to 40 real Apply attempts under controlled CPU load,
without an injected command failure, delay or altered deadline. Each S run
has 42 completed guards (two setup plus forty loaded attempts).

| Run | Busy workers / scope | Measured runnable / idle | Outer maximum ms | Render→installed maximum ms |
|---|---|---|---:|---:|
| S1 | 96 host processes | 96–98 / 0% | 48.955041 | 27.352941 |
| S2 | 96 inside lab container | 96–101 / 0% | 66.863855 | 33.694912 |
| S3 | 96 inside lab container | 96–99 / 0% | 184.078220 | 43.349036 |
| S4 | 384 inside lab container | 384–387 / 0% | 109.786531 | 53.133468 |
| S5 | 96 inside lab container | 96–98 / 0% | 59.960120 | 32.514153 |
| S6 | 96 inside lab container | 96–100 / 0% | 75.387173 | 46.382857 |

All six failed the diagnostic's explicit requirement to reproduce an install
overrun. These are negative reproductions, not passing tests. S3's 184.078220 ms
**outer** guard is not a 100 ms **install** overrun. S2's sampled container had
`cpu.max=max 100000`, weight 100, no recorded throttling and CPU pressure
`some avg10=80.48`; external-load L2's container had much lower sampled
pressure. These observations distinguish load placement without proving a
scheduler cause. S4 retained about 163 GiB available memory under load.

S1/S3/S4's final one-shot probes show the admitted source forwarding and the
revoked source dropping. S2's final probe shows **both dropped at baseline**,
but its Apply 335 and following live nft/tc verifier both succeeded. The later
member-path rejection withdrew forwarding: `member_path_unconfigured`, with
baseline command brackets [64186458578812, 64186551257218] and
[64186561126808, 64186634462698] ns. The admitted packet's ingress was
64186799139348 ns, with no egress through 64187304013373 ns. Both baseline
counter increments are retained. The underlying member-path read error was
not retained, so it cannot be classified as a timeout from these records.
This is a separate fail-closed withdrawal, **not F11-03 install rollback**.

S1–S4 retain both one-shot probes, nft before/after, all producer rows and
host/load brackets, but their negative diagnostic exited before exporting the
continuous capture. S5 tried failure cleanup but exposed cancellation of `t.Context()` before
cleanup, producing EOF; its missing capture remains missing. S6 uses a separate
bounded recorder context and retained 2,388 continuous probes with zero reported
socket capture drops. All 1,194 admitted-source probes reached ingress, 1,162
reached egress and 32 did not; 21 losses had ingress timestamps within completed
guard brackets and 11 outside. These are correlations, not kernel verdict
timestamps, and none came with an install failure. All losses remain counted.
The fixture now exports the capture on failure and checks the current verified
D-106 contract on recovery. No retry repairs missing earlier captures.

CPU contention increased measured durations here, but the controlled
intervention did **not** establish the proposed install-overrun mechanism.
In particular, neither a 100 ms failure's rollback-to-sibling-loss interval nor
its recovery upper bound was measured by these runs. The static node-wide
closure explanation above is supported by code and the historical failure
stage; a successful controlled reproduction is still required. Increasing
worker count is not a substitute for the missing original model and command
lifecycle evidence. F11-03 remains open, and AT02 remains failed.

## Hand-off: fields needed beyond this lane — 2026-10-31

Netlab owns `node/enforce/executor.go` and the enforcement bound. To distinguish
command runtime from runnable delay/cancellation, add an opt-in TEST ONLY
command record joined to Apply, with: command kind and phase (`install`,
`release`, `rollback`, `verify`); CLOCK_MONOTONIC context-deadline,
start-attempt, child-start, cancellation-observed, wait-return timestamps;
context error enum; exit code/signal; and `ProcessState` user/system CPU time.
Retain bracketed cgroup `cpu.max`, `cpu.weight`, `cpu.stat`, `cpu.pressure`,
ancestor-limit availability and scheduler context-switch observations when
available. Missing observations need explicit availability flags, not zeros
interpreted as success. A child start error must remain distinct from a killed
child. Do not expose command output or credentials in ordinary logs.

For a fresh controlled fixture also retain the exact TEST ONLY `enforce.Model`,
rendered transaction and their hashes, injection/load start/stop monotonic
bounds, and independent admitted/revoked packet observations. These are needed
to rerun the original work size rather than infer equivalence from generation
number. The historical raw error and full model cannot be recovered here.
Exact render-time host sampling belongs at the enforcement boundary; the new
agent field truthfully brackets the entire Apply instead.

Members 12 / isolation consumers should join by run ID and sequence; keep
`failure_stage`, observed `host_context` and `cause_status` separate. A failed
install with recorded pressure can be labeled as such while cause remains
`unproven`. Preserve the missing-verification state and every loss. The
four follow-up 28 source links now address the byte-identical follow-up 9
archive; all four files were checked on main and against the original source
commit. No evidence/test pin was changed.
