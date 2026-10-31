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
