# Guard frequency and the C1 trade, checked against code

[Entry point](revision.md). These are conditional design bounds, not C1
measurements. H-1 remains agent-core work. Production keeps today's guard.

## After H-1: when the guard still runs

[Engine.Step](../../../node/agent/engine.go) currently requires an I/O error for
its retain branch. With successful unchanged control I/O, it reapplies every
nominal ten-second poll: 6/minute, 360/hour, 8,640/day if continuously serving.
Scheduling, failures and pauses can change those counts; they are not observed
fleet totals. H-1 should replace no-op installs with verified retention.

After that change, real authority/configuration changes still require Apply:
new lease, desired generation (device/member authorization, revocation, FUP,
policy/path changes), and recovery that cannot adopt the installed state.
A successful unchanged verification/heartbeat must not refresh kernel timers.
Lease renewal follows [needRenew](../../../node/agent/sync.go) and
[renewLead](../../../node/agent/engine.go):
`lead = max(1s, min(configuredLead, leaseWindow/2))`, configured default 120s.
`needRenew` also adds `leadJitter(lead)` and is checked on scheduled Steps.
A changed desired generation can force earlier renewal; errors/backoff and
shorter service/policy/device deadlines affect the actual sequence.

| r24 cadence number | Disposition |
|---|---|
| 24h lease → every 86,280s / 23h58m | Keep as the **nominal window-minus-lead illustration**, not an exact schedule. Current jitter/polling correction below. |
| 15m lease → every 780s / 13m | Same qualification. 900s is the portal warning threshold, not a promised production choice. |
| 60s lease → every 30s | Same qualification; this is the test-lab configuration floor, not a production recommendation. |
| Restarts | Not every restart pays a guard immediately. [Same-boot adoption](../../../node/agent/restart.go) verifies without Apply but sets `leaseRefreshPending`, so successful fresh renewal subsequently reapplies. Cold boot, changed sets, lost provenance or failed adoption baseline and then perform fresh guarded Apply. Repeated failed Applies/recovery can cause additional guards/outages. |

Precision correction: `leadJitter` reads two random bytes but takes its modulus
against a span measured in **nanoseconds**. For the default 120s lead its actual
range is `−12s .. −12s+65,535ns`, not a uniform ±12s distribution. Thus ordinary
threshold crossing is at roughly 108s remaining (27s for a 30s lead); nominal
start-to-start intervals on this code are about **86,292s, 792s, 33s**, plus
poll quantization/control delays. A default ten-second Step can add nearly a
poll interval. These are source calculations, not timing promises or a jitter
fix. Agent-core owns that implementation. If it changes, recompute the cadence.
The looser conservative trigger bound `lead + lead/10` is 132s/33s regardless
of that bias; it remains an appropriate design ceiling until schedules are frozen.

The normal-path 60s convergence target and scheduler cadence are distinct from
the 100ms render/install bound. D-134's [D−101ms,D] admission transition remains.
D-135 still permits baseline until the next successful Apply with every outage
counted; neither a 100ms budget nor a five-second cleanup context bounds recovery.

## r24 exposure figures: keep, correct, or qualify

For eligible devices before release, C1 requires
`live_old AND eligible AND full_candidate_policy`. The old kernel elements are
never recreated or refreshed. Thus the hard authorization bound is the remaining
old expiry **E_old <= D_old <= D_new**, even with late installation, dead watchdog
or a stopped lock holder. It is a bound on admission, not draining preexisting
egress queues. The maximum is a residual lifetime, never a new full lease.

| r24 figure/claim | Checked interpretation |
|---|---|
| 28.5ms median, up to 94.7ms guard cost | Keep the measured **outer command brackets**: 28.493114ms median, 94.684882ms max, 101 Applies in F30. Actual commit-to-release closure lies inside those brackets. They are neither exact blackout durations nor upper guarantees for future runs. F30 was host-unqualified and instrumented. F32's member retirement adds a command only when needed; no new timing-cost conclusion is drawn without D-133. |
| “22 UDP losses, 28 TCP retransmissions, 256.8ms stall across AT10” | Do not carry as F30's result: [F30 outcome](../G2-followup30-evidence/guard-outcome.md) records one D-120 UDP boundary loss, seven uniquely matched TCP guard drops, seven retransmissions, and a **416.739885ms** maximum read. These positive observations do not explain historical B1's **1,501.407973ms** failed read. The review's aggregated numbers have no matching source in its cited F30 record. |
| Watchdog closes within 30–40s | **Conditional expectation**, measured from the last heartbeat, with 30s stale threshold and nominal 10s checks. [Watch.Once](../../../node/watchdog/watch.go) try-locks, reads, and installs with a 5s context. Scheduling delay, command failure, lock contention or watchdog death invalidate a hard 40s bound; a kill late in the heartbeat interval can close sooner. |
| SIGKILL releases the lock | Normally the process's lock descriptor closes, but do not infer that an already running child cannot commit. All writer lifecycles/reaping and the lock contract in [protocol.md](protocol.md) remain proof obligations. A stopped/hung lock holder can prevent watchdog acquisition. |
| “Only ~120s during normal renewal” | A useful rough default illustration, not a universal maximum. Current time-triggered renewal starts at <= **108.000065535s** remaining before I/O/render, due to the jitter calculation above; the policy-conservative jitter envelope is <=132s. With defaults this is about two minutes. Non-time-triggered `leaseRefreshPending`/generation/restart renewals can occur immediately after a fresh lease and must use the full residual D_old bound. Configured renewal leads need their own bound. |
| 60s–24h exposure | These are allowed configured lease windows ([contract](../../../contracts/openapi.yaml), §16.3), NOT a minimum exposure. Remaining exposure ranges from zero to at most 24h for supported valid authority, usually shorter due to binding/service/node/policy/timeout limits. At 900s configuration it is at most 15m. `ComputeMemberLeaseEnd` applies the minima; [Decide](../../../node/agent/decide.go) anchors them to the request's monotonic send time. |
| Same maximum as steady state | Keep for **eligible identities under the same old authority**. Today's serving kernel already survives agent loss outside the guarded interval until independent closure or existing expiry. C1 expands that survival to the install/check/release interval. This equality of maximum deadline does NOT preserve today's all-member closure at that cut. Revoked/tightened/new identities cannot use the old gate. |
| “Other 99.9%+ of the time” / “H-1 removes >99% of Applies” | Not unconditional facts. At 28.493ms/10s, the illustrative complement is **99.715%**, not 99.9%; the max-bracket complement is 99.053%. After H-1, a quiet 24h node would avoid about 99.99% of no-op installs, a 15m node about 98.7%, and a 60s node far less. Churn/restarts/failures invalidate these steady periodic illustrations. No fleet exposure fraction has been measured. |

The review's timer conclusion is sound subject to trustworthy per-device
provenance; H-3 means the current retained model alone cannot establish it.
Typical watchdog recovery and worst-case old expiry must be presented separately.

## Question 5: can every member close at the cut and still be hitless?

**Agree with the scoped argument.** At the instant after a candidate transaction
commits but before userspace can observe its completion, the kernel state is
the same whether userspace will continue normally or has just died. Hitless
forwarding requires that state to admit an eligible packet. Immediate all-member
closure on that death requires it to drop the same packet. Without another
observable event those requirements conflict. A second transaction, watchdog or
health check takes time and can itself fail; it cannot supply zero-delay death
detection. C1 instead uses old independently expiring authority during that gap.

This is not an exhaustive proof that C1 is the only conceivable stock-kernel
protocol. Nor does an absolute-monotonic expiry primitive, even if separately
established, imply immediate closure on process death: safe deadline admission
and all-member death closure are different contracts. The reviewed supported
interfaces in [F31 C3](../G2-followup31-design/candidates.md#c3--use-an-absolute-deadline-in-the-kernel)
do not establish such a primitive. No kernel capability or C1 success is newly
claimed here. A design relaxing G5/G6 requires the owner even if it improves AT10.

## The owner's question (for the orchestrator to relay)

Once unnecessary ten-second reinstallation is fixed, the guard would briefly
interrupt members mainly on lease renewals, actual policy changes and recovery
from restarts; the observed outer window was about 28.5ms at the median and
94.7ms at the maximum, with packet recovery sometimes taking longer. C1 would
let unchanged, still-authorized members continue through that window, but if
installation is late or the agent dies there, those members would no longer
close immediately: they could continue until independent shutdown or their
already-running old deadline, about two minutes for an ordinary default timed
renewal and at most the old lease's remaining 24 hours in the general case
(less with a shorter lease). Today's steady state already has that deadline
exposure when the agent fails outside the guard window; C1 would extend it into
the window while still closing revoked or tightened members at the candidate
commit. Do you accept that specific availability-versus-immediate-closure trade,
or should production keep the current guard?
