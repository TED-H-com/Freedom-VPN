# Follow-up 42: deadline, epoch and renewal consumers — 2026-11-03

This record describes follow-up 42, preserving earlier records as history.
The [judgments](followup42-r25-judgments-2026-11-03.md) precede production
changes. The rules implement §14.7 and §16.3 lease enforcement, §10.3–§10.7
metering, ISO01/ISO02, MTR01–MTR04 and AT10/AT13/AT16 mechanisms.

## Deadline meanings and consumers

`Decision.PolicyExpires` and `Engine.policyExpires` remain the earliest
installed binding boundary, with the node cap for an intentionally empty
policy. `buildModel` intersects each binding with its member lease/service end,
node lease, desired-state expiry and node offline window. It returns the
minimum as the agent's next expiry wake. MemberLease.DeadlineMono contains
member and node limits only; the renderer separately caps each device by
ValidUntil. No enforcement runtime changes were made.

`servingDeadline` derives the whole-node deadline from the installed model,
never a fetched candidate. It is the maximum of those independently capped
binding deadlines. Since every binding includes all node caps, the maximum
cannot cross a node limit. The helper caps wall durations before monotonic
addition to avoid distant-date overflow. A missing model/member deadline
returns zero; an intentionally empty installed policy retains its node cap.

| Consumer | Meaning after 42 | Safety and availability |
| --- | --- | --- |
| `nextStepDelay` | Wake at earliest installed binding expiry | The ordinary Step re-decides then Applies survivors; expiry does not wait for the poll interval |
| `expireServing`, called at Step entry and before/after current live verification | At the wake, refresh time/desired validity and run Decide | A serving decision continues without baseline. A non-serving decision withdraws. Last binding and all node limits still close forwarding |
| `canRetainServing` gates | Earliest boundary must match the fresh decision and remain future; every prior renderer, pending, boot, health and usage gate still applies | Expiry/model changes cannot skip Apply |
| Retained live-read branch | Re-decide after both live verification and member-path read | A read that crosses a boundary falls through to ordinary Apply of survivors on the same Step; a now-blocked decision withdraws |
| `Beat.PolicyExpires` in `beat` and `bootHeartbeat` | Whole-node deadline from installed admission | Earlier expired elements are removed by their kernel timeouts. A survivor's heartbeat cannot be rejected merely because a sibling expired |
| Watchdog `policy_expired` | Close at the whole-node deadline | The watchdog still rechecks the heartbeat under the shared node lock before baseline. Earlier device/member expiry needs no node baseline |
| Watchdog `serving_without_deadline` | A serving heartbeat with no valid whole-node deadline is invalid | Zero/missing deadlines remain fail closed; heartbeat-stale/boot/clock/baseline rules are unchanged |
| `adoptStoredServing` | Its existing heartbeat freshness/expiry check now sees the whole-node limit | Same boot, signed documents, rebuilt authorization, live rules and meter ownership remain mandatory. It derives the next wake from the fresh decision; adoption cannot extend kernel timeouts |
| Restart diagnostics and optional heartbeat diagnostic record | `policy_expires_mono_ns` matches the heartbeat's whole-node value; restart logs separately name `next_binding_expiry_mono_ns` | No diagnostic claims a new lease or a new installation. Optional diagnostic output was not enabled for timing measurement here |
| `baselineReason` | Earliest boundary still identifies an expiry-triggered baseline call after preceding failure causes | A diagnostic only; it neither authorizes forwarding nor triggers withdrawal |
| Apply/adoption assignments and baseline/withdraw/revoke/quarantine resets | Store earliest boundary on successful admission; clear on withdrawal | Retained audit models are never enough to manufacture a serving deadline: `servingDeadline` also requires actual serving and a positive wake |
| Observation report / `observationState` | Actual `MemberForwardingEnabled`, current decision checks/reasons, signed lease's original ExpiresAt | The wire lease timestamp still describes the signed lease, not the earliest device. No new API field or invented deadline is reported |

A fresh heartbeat can cross the earlier device boundary while control I/O is
in progress and the node lock is free. The new watchdog proofs exercise that
exact unlocked path, as well as the interval before the next heartbeat. The
watchdog leaves the survivor serving; a stopped heartbeat still closes after
its stale bound, and a fresh heartbeat still closes at the last deadline.
Device/member expiries at Step entry, initial verification and retained
verification all produce one ordinary Apply with no baseline in between.
Ordinary Apply can still use its existing guard; no hitless claim is made.

## Accounting at expired-only removal

The normal identity-change path still uses baseline and final sample. There
is one narrow exception: same generation, strict subset of exactly unchanged
bindings, and every removed binding is already past its installed deadline.
`onlyExpiredBindingsRemoved` also requires that the meter snapshot describes
the installed binding set. Additions, rebindings, unexpired removals, new
generations and uncertain provenance cannot use the exception.

Under the node lock, `settleExpiredBindings` verifies the installed rules,
then uses the meter's paused nonfinal checkpoint. Kernel expiry has stopped
the removed devices from incrementing counters. The checkpoint commits their
outgoing deltas before the surviving usage intent is persisted. Surviving
counters can keep increasing because their identities are unchanged. Apply
uses its ordinary conntrack, meter, verification, persistence and heartbeat
paths, then resumes sampling. The existing nonfinal checkpoint API is reused;
this does not publish an updater barrier or close the counting interval.

Crash ordering: before checkpoint, the old intent still covers both sets;
after checkpoint and before intent replacement, the old intent is still safe;
after replacement, removed identities have no possible new admission and
survivors have unchanged identities. Failure to checkpoint withdraws with
pending settlement and the original intent. Failure to persist or Apply uses
the normal withdrawal path. Tests check exact bytes, no gap/recount after
restart, and failed settlement refusing the surviving Apply.

## Truthful heartbeat and observation

The incorrect sibling withdrawal/early return is removed. `reportingDecision`
also refreshes a cached decision with no reasons when a current blocking
reason exists; heartbeat, report fields and semantic coalescing all use that
view. Existing reasons and diagnostic errors remain reportable. The actual
serving flag is never inferred from `Decision.Serve`.

The zero Decision before the first evaluation has no checks and legitimately
retains `Reason: "-"` if the generic `beat` path is called (for example an
early shutdown). Boot's dedicated heartbeat instead explicitly writes
`boot_baseline` or `boot_unverified`. The zero-decision exception is tested;
after an evaluated serving decision, a persistence failure is reported as
`meter_storage_failed`, not dash/empty reasons.

## Epoch continuity consumers

`canRetainServing` reads Head, verifies current boot, open epoch, generation
and bindings. Equal epochs need no historical query. Different epochs use
`SameCountingInterval`: only successive same-boot counter-discontinuity
close/open pairs qualify. Final samples, a closed head, boot changes, missing
history or other boundaries reject retention. All earlier pending/failure
gates remain in place.

Both the post-Apply `switchEpoch` producer and the periodic sampler may
rotate the head. They leave memory and durable intent together on the older
origin until the next retained tick. That origin remains valid evidence
through the existing restart continuity check. Before completing retention,
`recordUsageIntent` writes the current head to durable `usage_apply_intent`
and only then updates `restartIntent`. A failed write leaves both at the old
origin, marks usage persistence failed and withdraws. A restart before the
write uses 40's continuity proof; after it, restart loads the new epoch.
No extra Apply is needed for either producer. Unchanged equal-epoch ticks
still avoid a metadata write. No SQL schema or ledger event semantics change.

## Renewal jitter consumers (F32-01)

`leadJitter` samples an integer uniformly from the inclusive millisecond
range `[-floor(base/10/ms), +floor(base/10/ms)]`. `crypto/rand.Int` uses
rejection sampling, avoiding modulo bias across the full span. Production
uses crypto/rand.Reader; tests inject bytes. A failed or short read uses zero
offset and caches that result, retaining the safe base lead.

`needRenew` uses `renewLead(window) + renewalJitter(window)`. The draw is keyed
by accepted lease ID and issued-at time, reused across every check and replay,
and redrawn for the next lease. `acceptLease` passes the draw to SaveLease;
agentstore stores `lease_renew_jitter_ns` in the same transaction as the lease.
`loadRuntime` restores it. Legacy databases without that metadata use zero
until a new lease arrives. An explicit configuration change can clamp the
saved offset to the new base's ±10% bound without drawing again. Corrupt
metadata reads fail through the existing store-load error path.

At the default 120-second base the lead is 108–132 seconds before expiry.
Jitter applies after renewLead's window/2 cap: the upper bound is 1.1 times
the capped base, hence 0.55 of the window when the half-window cap applies.
For a 60-second lease the range is 27–33 seconds. The existing one-second
base floor is retained (relevant only to sub-two-second synthetic windows;
accepted production leases have longer windows). Draw resolution is 1 ms.
The distinct grant-less D-067 schedule retains its unjittered base/cap and
bounded exponential waits; no random check can perturb it.

Tests enumerate all 24,001 default-span values, endpoints/midpoint at the
default and capped base, rejection of an out-of-range draw, stable repeated
checks, next lease, short/failed read, replay/restart reuse, configuration
clamp and rollback when the lease/jitter transaction fails.

## Validation limits and requests — 2026-11-03

Functional gates and the full node-test-netlab use the final production
commit after the single main merge. Receipts are in the dated outcome record.
Earlier r16–r25 proofs remain unchanged. The only node/enforce edit is the
approved DeadlineMono comment. The usage lab retry remains unchanged.

No reservation, quiet-slot admission experiment or D-133 timing comparison
was run. The mandatory lab keeps its existing functional deadline diagnostics;
its optional whole-suite timing export is disabled. Physical arm64, field
network behavior and hitless transitions are not claimed. The orchestrator
must schedule the owed D-133 A,B,A,B for 37–42 after review.
