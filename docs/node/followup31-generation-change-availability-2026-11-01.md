# Generation-change availability — 2026-11-01

Design proposal for D-025 adversarial review; **no availability behavior change**
is implemented here. D-135 and D-136 remain provisional owner decisions. The
runtime changes are the existing TEST ONLY timing gate opting Apply into
netlab 22's command observations and bounded local completion during cancellation for the
separately reproduced restart races described below. All losses remain losses.

## D-136: confirmed closure, corrected recovery description

References below describe the follow-up 31 source in `node/agent/` unless a full path is given. `policy_missing` is also
used for genuinely absent/invalid desired state; that reason alone does not
prove a generation mismatch.

1. `Step` first verifies the retained serving model (`engine.go:526`), keeping
   the old generation and its original deadlines while fetching desired state.
   `syncDesired` verifies and durably accepts the new document (`sync.go:23`).
   It does not itself replace kernel rules. `leaseGenerationStale`
   (`decide.go:556`) tests two nonzero desired/lease generations for inequality;
   `Decide` adds `policy_missing` (`decide.go:192`).
2. With no retained Apply error, a populated unexpired lease still matching
   `appliedGen` does **not** become renewal-due just because desired advanced:
   `needRenew`/`leaseBehindApplied` (`sync.go:149,231`). A due ordinary renewal
   can instead obtain the new lease before withdrawal; not every generation
   change must exhibit this window. Failed/replayed/older-generation renewal
   does not authorize serving the new generation.
3. `evaluate` → `refreshProbe` → `prepareProbe` (`attest.go:71`,
   `engine.go:762`, `probeexc.go:194`) notices the lost serving gate and
   `withdraw`s. `handoff` pauses sampling, installs baseline and final-samples
   the old epoch (`usage.go:310`); `withdraw` calls baseline again
   (`engine.go:1237`). `enforce`'s ordinary `!d.Serve` branch installs baseline
   once more and then calls `ackApplied` (`engine.go:952`). Generation 0 is the
   installed node-wide baseline; local applied-generation bookkeeping advances
   to the desired generation after that baseline succeeds. It does not claim
   that the new member model is already forwarding.
4. **Correction to the cited D-136 reading:** `engine.go:938–950` is inside
   `if e.applyError != nil`. Only recovery from a retained error uses
   `recoveryDecision` (`recovery.go:8`) to construct a deny-all candidate and
   verify its nft/tc/path before clearing that error and ACKing. The ordinary
   successful stale-lease path above performs baseline calls, not a deny-all
   candidate Apply. Neither path restores old revoked authorization.
5. `renewAppliedLease` (`sync.go:243`) then requests the matching lease in the
   same Step, subject to the one-attempt-per-Step and Retry-After gates. Central
   signs its **desired** generation; client applied generation is observation
   input (`internal/leases/leases.go:463`). Matching accepted renewal is only
   authorization: `evaluate` and `enforce` must still succeed. Apply installs
   behind the deadline guard, releases it, verifies, and returns; Engine runs
   another live verifier and a member-path check before setting `serving=true`,
   ACKing/publishing heartbeat (`engine.go:1037–1186`). End-of-Step verification
   runs again (`serving.go:161`). Kernel egress can resume at guard removal,
   before Apply return or the later serving heartbeat; those are different
   observed endpoints, not an exact packet cutoff.

Each control pass has a bounded I/O budget (`serving.go`, `controlclient`),
reads have their own bound, and the guarded install retains the 100 ms budget
and 1 ms granularity (D-134). **There is no finite recovery/outage ceiling.**
Control failure, retry backoff, provisional grants, further generations, failed
Apply, path rejection or expiry can keep baseline indefinitely. A default
poll/heartbeat cadence is not a recovery guarantee. No old deadline is extended
while waiting. D-135's baseline→successful recovery bracket applies even when
there is no failed Apply; an Apply-end endpoint must be accompanied by later
verification/path/heartbeat evidence and independent egress.

`member_path_unconfigured` can occur during recovery within this broad outage,
before Apply when capturing a new topology, or after successful Apply and live
verification when checking the candidate/retained topology (`engine.go:1005`,
`memberpath.go:123–150`). It triggers a separate withdrawal and can extend or
reopen the outage. A stale-generation test does not itself set this condition.
The ordinary baseline branch has no member-path check. A previously latched
`memberPathPending` can also label a baseline heartbeat during the window. The heartbeat reason
at `engine.go:1256` is a presentation override, not proof of which `ip` read
failed. No nft/tc failure or timeout may be inferred from it.

## Members 12 and netlab 21 joins

The original [F12-01 records](followup31-evidence/inputs/members12-policy-missing.json),
[AT02 trace](followup31-evidence/inputs/members12-AT02-TRACE.md) and
[request](../e2e/at/members/requests.md) retain producer run `3065df99`, same
boot, with these observations:

| Sequences | Observation | Interpretation/limit |
|---|---|---|
| 360–361 | generation-11 live verification/serving heartbeat | Old rules qualified immediately before the transition. |
| 362–364 | Three successful `policy_missing` baseline calls, lease still 11 | Consistent with handoff baseline, withdraw baseline, then non-serving enforce baseline above; no candidate Apply or verifier rows between them. These rows do not carry the desired document itself. |
| 365 | Newly accepted populated generation-13 lease | Renewal bracket 8.240340 ms; authorization obtained, forwarding recovery still pending. |
| 366 | Successful generation-13 Apply, verification true | Apply 78.159370 ms; completed guard 34.971315 ms, distinct from outage. |
| 367–368 | Successful live verification, serving heartbeat | Later readback/control-plane readiness, not an exact kernel-open time. |

Baseline 362 start → Apply 366 end is **236.970251 ms**. Nine admitted-sibling
losses fall within that bracket, eight outside every completed guard. Packet
115 already has egress within the release-command bracket; packet 117 is the
first recorded ingress with egress *after Apply end*. Do not call the latter
the first forwarding packet after baseline. Every one of the run's 21 sibling
losses remains counted. The record is consistent with the expected
stale-generation mechanism and not with failed-Apply rollback. Its historical
missing desired-input/branch observation prevents inventing a definitive
per-call causal trace. Label it **“policy_missing baseline during generation
11→13; stale-lease path consistent with code; no failed Apply”**, keeping
historical `cause_status=unproven` unless the desired input is independently
joined. Fresh controlled measurements below explicitly establish that input.

Netlab 21 request 3 is cross-referenced only. Its three injected verifier-fault
baseline-start→recovery-release-end brackets were **304.815081 / 280.395264 /
285.296508 ms**, including a deliberate **200 ms hold**. It recorded **173
sibling losses, 150 outside every completed guard**. These endpoints differ
from the F12-01 Apply-end bracket. [Exact explanation and source](../poc/G2-followup21-2026-10-31.md#f11-03-admitted-sibling-during-rollback).
Agent-core 29 settled the rollback reading under D-135: node-wide baseline,
no previous-generation restoration, every loss retained and recovery required.
No engine change follows from these fault brackets. Netlab 22's later
312.901 / 323.541 / 319.004 ms injected-fault measurements are another
population, not stale-lease observations or an availability SLA.

## First measurement population and protocol

The first population uses source `6079e5ed`; the final-source quiet repeat is
reported separately below. Both original populations remain retained.

TEST ONLY Engine + local TLS fake control + real nft/tc, inside the lane-owned
agent netlab fixture. Forty desired-first changes, generations 12→52. Member
2 is continuously admitted; member 1 alternates removed/restored. Both sources
send synthetic header-only UDP every 10 ms. There is no forced renewal, delay,
command error, deadline adjustment or allowed-loss threshold. The fixture
uses a one-hour node/member lease and one-second configured cadence; direct
Steps measure the processing window after publication, not a production poll
latency or native NetBird path. Packet timestamps are userspace observations.

The existing producer retains run/boot/sequence, baseline/renewal/Apply/live
verification/heartbeat records and D-106 digest. `commands` is additive. Its
availability flags are authoritative: child-start exact time and hidden host
ancestors remain unavailable; deadline conversion is a bracket. CPU/switch
counts are observations, not contention causality. Disabled mode uses the
unchanged executor path. Raw evidence and every failed attempt are retained.

All **40 transitions completed** and every candidate Apply succeeded. The
[analysis and full vectors](followup31-evidence/generations/analysis.json),
[original producer rows](followup31-evidence/generations/operations.jsonl),
[continuous packet observations](followup31-evidence/generations/continuous.json)
and [transition inputs/sequence joins](followup31-evidence/generations/transitions.json)
are retained. Units below are milliseconds, nearest-rank percentiles over 40
transitions (at this sample size p99 equals max).

| Metric | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Baseline start → Apply end | 177.901491 | 192.802074 | 201.859702 | 201.859702 |
| Baseline start → first serving heartbeat | 205.655616 | 222.701043 | 232.538306 | 232.538306 |
| Baseline start → first observed post-Apply sibling ingress with egress | 181.356245 | 198.600980 | 204.328359 | 204.328359 |
| First baseline writer call | 37.424719 | 43.059216 | 47.140146 | 47.140146 |
| Renewal request → accepted/persisted lease | 2.641230 | 3.138726 | 3.442233 | 3.442233 |
| Apply | 72.738482 | 83.652664 | 93.704372 | 93.704372 |
| Verifier inside Apply | 23.305480 | 26.084291 | 27.008475 | 27.008475 |
| First post-Apply live verifier | 20.101979 | 22.837260 | 23.695661 | 23.695661 |
| Completed outer guard | 32.247691 | 43.514818 | 50.708703 | 50.708703 |
| Render → installed | 15.921472 | 24.505880 | 29.837489 | 29.837489 |

The first baseline writer bracket includes baseline reads/cleanup, not just an
NFT syscall; neither endpoint is an exact kernel commit. Renewal includes
acceptance and durable storage. The later heartbeat additionally requires the
Engine verifier/member-path work. First post-Apply ingress is a sampled recovery
control, not the first moment the kernel reopened.

The continuous sibling stream records **1,448 ingresses / 936 egresses / 512
losses**, all losses within the recorded baseline→Apply brackets: **76 inside
completed guards, 436 outside**. Per-transition loss p50 / p90 / p99 / max is
**13 / 14 / 15 / 15**, with outside-guard counts **11 / 12 / 13 / 13**. All capture
drop counters are zero, every sent probe reaches observed ingress, both sources
have initial positive controls, every transition has later sibling egress, and
the revoked source has no egress in each post-recovery observation interval.
The diagnostic passes by retaining and demonstrating these outages; it does
not pass an uninterrupted-sibling availability requirement.

Host qualification did **not** establish a quiet interval. Eighteen rejected
`vmstat 5 7` preflights preceded the run; no suite started during those attempts.
The admitted run had one competing suite at preflight and every two-second
runtime check, including after startup. Its six interval samples had preflight
r max **3**, idle min **78%**; postflight r max **3**, idle min **45%**, with a
second suite appearing during postflight. The hard two-other-suite startup
limit was respected. [All raw vmstat text, bounds, counts and runtime checks](followup31-evidence/generations/host.json)
are retained; private process text is withheld. No quiet-host or CPU-causal
claim follows. This is a bounded sample on this host, **not a maximum outage
bound**. The no-overlap A/B protocol remains required for a future design.

There are **1,806 command observations**, all started, with CPU/switch samples
available. Exact child-start and hidden host-ancestor availability are **zero**;
these are unknowns, not zeros for runtime/limits. Deadline brackets are validated
as brackets. Run/boot/sequence continuity is checked, no producer row is
renumbered, and all additive command fields remain outside D-106.


## Final-source repeat without competing suites

A second, independent **40/40** population on source `98900942` uses the same
fixture, unchanged generation-serving logic and the completed shutdown repair.
Both six-interval `vmstat 5 7` samples had r max **0**, idle min **87%**; no other
suite appeared before, after, or at any two-second runtime check. These samples
qualify as quiet under the stated r≤3/id≥80 rule. They do not expose every
scheduler event or hidden ancestor. This is **not an A,B,A,B comparison** and
no numerical difference from the first population is assigned to a code change.

[Native producer and input records](followup31-evidence/generations-final/operations.jsonl),
[transition joins](followup31-evidence/generations-final/transitions.json),
[packets](followup31-evidence/generations-final/continuous.json),
[full distributions](followup31-evidence/generations-final/analysis.json), and
[host record](followup31-evidence/generations-final/host.json) are retained.
Milliseconds, nearest-rank percentiles over all 40 transitions:

| Metric | p50 | p90 | p99 | max |
|---|---:|---:|---:|---:|
| Baseline start → Apply end | 173.766526 | 187.813686 | 195.601257 | 195.601257 |
| Baseline start → first serving heartbeat | 200.833240 | 215.327259 | 250.566367 | 250.566367 |
| Baseline start → first observed post-Apply sibling ingress with egress | 178.835766 | 190.278186 | 202.196352 | 202.196352 |
| First baseline writer call | 38.609244 | 44.168038 | 49.509973 | 49.509973 |
| Renewal request → accepted/persisted lease | 2.793474 | 3.293004 | 3.677854 | 3.677854 |
| Apply | 70.012505 | 77.958872 | 88.649438 | 88.649438 |
| Verifier inside Apply | 22.914303 | 26.864766 | 34.056141 | 34.056141 |
| First post-Apply live verifier | 19.179623 | 22.005820 | 31.035104 | 31.035104 |
| Completed outer guard | 30.363947 | 34.736797 | 41.170949 | 41.170949 |
| Render → installed | 15.226535 | 21.019801 | 23.867708 | 23.867708 |

There were **1,444 sibling ingresses / 943 egresses / 501 losses**, all 501 in
baseline→Apply brackets: **66 inside completed guards / 435 outside**.
Per-transition loss p50/p90/p99/max is **12/14/14/14**, inside-guard counts
**2/2/3/3**, outside-guard counts **11/12/12/12**. All capture drops are zero;
every transition has independent later sibling egress and every revoked-source
post-recovery interval remains denied. All candidate Applies succeeded.
The same 1,806 command observations have CPU/switch availability; exact child
start and hidden host ancestors remain unavailable. Across both populations,
**80 transitions and all 1,013 sibling losses remain recorded**; populations
are kept separate, not pooled into a synthetic bound. Option (c)'s current
measured baseline→Apply maximum is 195.601257 ms in this quiet population,
with a later heartbeat maximum of 250.566367 ms and no guaranteed outage ceiling.

## Restart failure: observations and separate cancellation repair

Netlab 22's original qualified A1 still fails. Its
[retained suite log](../poc/G2-followup22-evidence/runtime/A1-agent-suite.log)
and [producer values retained in an excerpt](followup31-evidence/restarts/historical-A1-producer-excerpt.json)
show a loaded generation-5 lease issued at 04:54:36.767145627Z, node expiry one
hour later. At the next boot baseline start its node deadline still has
**3,597.962729776 s** remaining. The outgoing process records
`meter_storage_failed`, then a non-serving heartbeat and shutdown baseline.
The successor loads the lease (it appears in that new run's producer rows),
but cannot adopt because the retained heartbeat is non-serving.
`adoptStoredServing` requires a fresh serving heartbeat, valid matching
same-boot documents, original anchor, live rules/path and meter continuity
(`restart.go:15`). `Boot` deliberately resets `leaseFresh=false` and requires
adoption or successful renewal; later `lease_expired` reports therefore do not
prove that this signed node deadline elapsed. The historical producer does not
export each signed member deadline or the exact cancellation point.

The separately supplied `assembled/expiry.json.restart.json` brackets the
earlier D-104 expiry test (monotonic 92375324379144–92375443069357), **not** this
TestLab04 restart (boot baseline starts at 92416319589497). It cannot supply
missing TestLab04 restart timestamps. Historical storage-failure cause remains
unproven beyond the recorded sequence; do not assign it merely from later logs.

A fresh foreground **30-iteration unchanged-main** run records **25 passes /
5 failures**, with unchanged assertions. Its [all-iteration snapshots and
failed producer windows](followup31-evidence/restarts/main-30.json) read the
actual SQLite lease/anchor and member deadlines before stop, after stop, before
restart and after start. Every failed case is same boot and has node **and**
member remaining validity of **3,599.377190–3,599.448139 s** against the TEST ONLY
**3,600 s** lease. Four failures have `nft` exit -1/signal 9 during Apply (two
install failures, two release failures, none an install-budget overrun); their
subsequent heartbeats are non-serving. The fifth has successful Apply/live
verification followed by `member_path_unconfigured`. These stages are proven;
the unchanged-main caller has no command-lifecycle cancellation sample, so
signal 9 alone is not a scheduler/cancellation cause proof. Read-only snapshots
add observation overhead and this is a dedicated TestLab04 population, not 30
replays of the entire original suite order.

Deterministic regressions separately prove three cancellation mechanisms:
SIGTERM after the independent live read makes later canceled SQLite/meter reads
look like storage failure; cancellation during Apply reaches its rollback;
cancellation during the post-Apply path read rejects otherwise valid topology.
All fail before the corresponding fix. The repair stops at the safe boundary
if already canceled, and thereafter separates local work from control I/O:
control retains the original caller cancellation/budget, while local
transaction/readback work gets a bounded context of the control allowance plus
five seconds (at most ten seconds). Existing nft install/read budgets, lease
anchors, expiry and D-135 rollback stay intact. A genuine injected Apply/tc
failure still withdraws/quarantines even when cancellation coincides. A forced
process kill remains outside graceful completion; no cleanup is infallible.

The corrected source passed **30/30 foreground repetitions** with the same
assertion and observation mode ([all fixed snapshots](followup31-evidence/restarts/fixed-30.json)).
The unchanged-main and corrected runs each overlapped one other suite; their
[host samples and runtime checks](followup31-evidence/restarts/host.json) remain
unqualified for quiet-host performance comparisons. This is functional
reproduction/regression evidence, not a measured attribution of scheduler
contention or proof of the exact historical cancellation point.

The first full agent-suite attempt failed and remains retained
([test output](followup31-evidence/restarts/first-full-suite-tests.log),
[restart snapshots and producer window](followup31-evidence/restarts/first-full-suite-failure.json)).
The expiry collector raised `ProcessLookupError` reading `/proc` after SIGTERM;
its handler covered only `FileNotFoundError`. A deterministic collector test
fails before the narrow correction, while unreadable proc state and shutdown
timeout still fail rather than starting a successor.

That full attempt also failed TestLab04 with **3,598.488237212 s** left on both
node/member grants. The outgoing trace ends after successful Apply/live read,
before a serving checkpoint. Generic `stopProc` escalates to SIGKILL after
200 ms without confirming exit; the exact interrupted operation/kill time was
not captured, so that failure is not assigned a more specific cause. TestLab04
now requires and records confirmed graceful exit (10 s maximum, failure on
exhaustion), with no forced kill, extra grant or serving-readiness wait. Its
precondition and post-restart forwarding assertions are unchanged. Forced crash
behavior remains covered by TestLab03; a packet seen after guard release alone
does not prove a durable serving checkpoint. This harness change is separate
from the cancellation repair. The earlier 30/30 population used the original
200 ms cleanup helper and is not relabeled as this corrected harness.

The full agent suite then passed on `98900942`, including D-104 renewal/expiry
correlation (zero unresolved ingress in that collector), TestLab04, kernel
timeout, watchdog, revocation and meter continuity. The installer-marker test
retains its existing opt-in skip. TestLab04 recorded a **40.839979 ms** confirmed
graceful stop; its [native restart snapshots](followup31-evidence/restarts/full-suite-pass.json)
remain separate from the earlier failed test. Pre/post host samples had r max
0/1 and idle min 86%/86%, with no other detected suites at startup or runtime.
No timing threshold was relaxed; the earlier collector exception had produced
no completed expiry timing verdict. The ordinary live suite also passed (91.08 s): 30 Steps/90 live reads,
read p50 18.315428 ms / max 21.125383 ms against its 500 ms read limit;
75 guarded Applies retained all 1,133 losses from 76,000 renewal probes.
No live timing threshold failed, so the conditional D-133 attribution run was
not triggered. This live population overlapped one other suite and has
unqualified pre/post host samples; no quiet-host performance comparison is
claimed. Final repository qualification is recorded in the hand-off and
qualification artifacts.

## Alternatives and invariants for review

Let **D** be the earliest original same-boot monotonic cutoff over old node
lease, old desired validity, old member service/lease/offline limit and device
binding. Let **E** be the current generation-change withdrawal point: the
first atomic enforcement transaction that removes old authorization after
accepting the new desired document. An unavailable/failed commit never counts
as a completed revocation. Today's safety also depends on the original kernel
cutoff and watchdog; there is no hard wall-clock scheduling SLA for E.

“Never later than today” here requires **no new network wait or grace period
on the revocation path, removal at E before established/related accepts, and
no cutoff later than D**. Merely retaining old rules until their eventual
lease expiry fails that requirement for a known revocation. Measurements can
compare actual E/packet times; they cannot prove a universal OS scheduling
bound. Preserve D-134, control access, metering, QoS, source binding, global
isolation and identity checks for every option. Spec §8.2 requires revocation before established/related acceptance; §16.3
and the §21 lease parameter table leave the lease-length availability/revocation
trade-off with the owner. No option increases that lease length. Owner approval
is required before any new cross-generation authorization semantics.

### (a) Make before break

Fetch and verify the new desired generation's lease while the old rules still
serve, then atomically install the fully authorized replacement. Central
already signs desired generation independent of the request ACK; never falsely
ACK an uninstalled model to obtain a grant. Initial publication may still
return provisional grants until central readiness is satisfied.

**Unqualified version rejected:** waiting for a lease while a new desired
revocation is known delays revocation until an unbounded response or D. To
meet the requirement, use this only for a provably non-restrictive transition
with unchanged bindings, publication, isolation, DNS, QoS and limits. Any
removal, shortened validity, identity freeze, global restriction or uncertainty
must take today's baseline at E without waiting for the response. A previously
prefetched matching lease can avoid the wait, but cannot be assumed available.
This constrained version guarantees the same revocation cutoff and original D;
it does not solve the sibling outage accompanying actual revocations.

Expiry during fetch closes at D; cancellation/replay cannot renew the anchor.
A failed install or verifier uses D-135 baseline, with no restoration;
the existing verified conntrack-delete exception retains its bounded pending
cleanup semantics, never restores revoked admission. A
cloned-identity freeze immediately invalidates both pending and active candidates,
persists the latch and suppresses ACKs. Restart only adopts the existing verified
same-generation model; discard staged work and baseline if durable desired is
newer/restrictive or evidence is ambiguous. No speculative state is called applied.

Proposed files: `node/agent/sync.go`, `node/agent/engine.go`,
`node/agent/decide.go`, `node/agent/restart.go`, `node/agent/freeze.go`,
`node/agent/lease_renew_test.go`, `node/agent/followup31_test.go`,
`node/agent/followup31_netlab_test.go`,
`tests/netlab/agent/scenario_test.go`, and this note. No central issuer/contract
change is justified for this constrained variant; central readiness behavior
must be tested using `internal/nodes/httpwire/lease_transaction_integration_test.go`
by its owner.

D-133 acceptance: paired non-restrictive and restrictive cases under the common
protocol below. Non-restrictive outage distribution must improve; restrictive
cases must preserve today's E and fail-closed outcome. Failure/replay/freeze/
expiry cases must never benefit from an extended deadline. Reject a design
claiming to fix AT02 revocation-sibling availability from this narrower result.

### (b) Intersection serving

At E atomically remove all newly revoked authorization while retaining only a
conservative intersection. It is **not** just old lease member IDs intersected
with new desired member IDs: an old account-level grant could otherwise admit
a newly added device/IP, recycled binding or newer entitlement revision. Require
old *verified desired + lease + applied model* and new verified desired, identical
node/incarnation, member/device/peer/address identity and compatible binding and
entitlement revisions. Deny additions and ambiguous changes. Restrictions are
monotonic: union denials, intersect allowed destinations/resolvers, retain the
stricter QoS ceilings and earliest validity. If a safe global policy/QoS/topology
intersection cannot be established, use today's node-wide baseline at E.

The revocation transaction must not wait for lease fetch, tc reshaping, a new
probe, or other speculative work. Prepare the subset without mutating active
state; if it is not ready at today's withdrawal point, baseline. A separate
revocation-first nft transaction must remove allowset/rule authorization before
established/related accepts, preserve accounting, and arrange exact conntrack
cleanup. Shared-path/global failures and failed ownership/readbacks trigger
baseline/quarantine. The existing shape-first Apply cannot simply be reused as
a delayed revocation transaction. Netlab must design and review that boundary.

Per binding, deadline is the minimum of **all old D limits and new desired
limits**, using the original monotonic lease anchor and unchanged reserve;
re-rendering cannot restart lifetime. Expiry removes the relevant binding;
uncertain node/policy/meter safety closes everyone. D-135 remains untouched:
a failed install/verifier leaves baseline and recovery needs fresh current
authorization. Preserve the existing verified conntrack-delete exception and
its retry bound; it cannot reintroduce revoked admission. Never restore the previous subset after a failed Apply.
Freeze has higher priority than intersection and atomically clears it; retain
the durable incarnation freeze across crashes. Initially **do not adopt an
intersection after restart**: baseline, reacquire matching authorization, then
full verified Apply. Crash at every commit/persist boundary must remain safe.

Serving-generation/readback/ACK semantics are a blocking design question.
An intermediate subset is not a full new-generation Apply, and an old-generation
marker alone cannot describe the new revocations. Retain explicit provenance
for old desired hash, lease ID/generation/original anchor, new desired hash and
subset hash; never mutate/re-sign a lease or use `recoveryDecision`'s temporary
generation rewrite to authorize members. Do not claim the current
`desired=applied=lease` readiness predicate holds. An explicit transitional
observation and verifier model is required; D-106 and contract owners must
approve its representation before code. Until that decision, option (b) is
not implementable by merely removing `leaseGenerationStale`.

Proposed files: `node/agent/decide.go`, `node/agent/engine.go`,
`node/agent/sync.go`, `node/agent/serving.go`, `node/agent/serving_evidence.go`,
`node/agent/memberpath.go`, `node/agent/revocation.go`, `node/agent/freeze.go`,
`node/agent/restart.go`, `node/agent/observe.go`,
`node/internal/agentstore/state.go`, `node/internal/agentstore/store.go`
(local SQLite migration only), `node/agent/followup31_test.go`,
`node/agent/followup31_netlab_test.go`, and this note. Outside this lane:
`node/enforce/apply.go`, `node/enforce/render_nft.go`,
`node/enforce/verify.go`,
`node/enforce/evidence.go`, `contracts/openapi.yaml`,
`internal/nodes/observations.go`, `internal/nodes/transitions.go`, and
`tests/e2e/at/members/forwarding_test.go`. Generated API consumers follow the
contract owner's chosen representation. No such edits occur in this follow-up.

D-133 acceptance: actual restrictive generations with an unchanged sibling,
revoked source and newly added device; prove the subset never forwards the
latter two while stale, including established flows and address reuse. Compare
revocation cutoffs and full sibling loss distributions, not only completed
install guards. All safety/expiry/crash/freeze negatives below are mandatory.

### (c) Status quo

Baseline at E, same-Step renewal if eligible, then successful Apply, live
verification, topology check and final serving heartbeat. Every stale-generation
member is denied. Expiry, provisional grants and transport/Apply failure keep
baseline; neither expiry nor retry can reopen. D-135 rollback and durable
cloned-identity freeze are unchanged. Restart accepts only verified matching
same-boot authorization with the original deadlines; otherwise baseline.

Revocation occurs at today's E and is bounded offline by D under the existing
kernel/guard assumptions. Availability has **only the observed distribution
above, no finite guaranteed bound**. The 100 ms install budget does not bound
renewal, whole-Step work, path verification or total outage. No production
files need change. Reusable measurement files are
`node/agent/followup31_netlab_test.go`,
`tests/netlab/agent/run-live.py` and this note; the opt-in lifecycle caller is
independent diagnostic work. D-133 comparator A uses these same instruments
and inputs; failures remain counted, not exempted because fail-closed is designed.

## Recommendation and D-133 acceptance plan

Recommend **(b) for adversarial design review**, while retaining **(c) in
production** until its provenance/ACK and revocation-first boundary are resolved.
It targets the actual unchanged-sibling outage during a revocation. The safe
restriction of (a) helps only non-restrictive changes; its unrestricted version
violates the no-later revocation requirement. This is a proposal, not approval
to alter a signed-generation contract or to waive AT02.

Freeze A (current main) and B (reviewed candidate) with source/binary hashes;
use the same host, kernel, inputs, fixture image, observation mode, probe cadence,
Go runtime settings and exact signed/model/rendered bytes where reusable. Run
**A1,B1,A2,B2**, with both full agent and live suites in each leg. Record
`vmstat 5 7` before startup, before traffic and after; exclude the since-boot row
from qualification, require no competing suite and six intervals r≤3/id≥80.
Keep rejected preflights, failed legs and unqualified runs. Never replace a
failure with a later pass or compare an instrumented B to an uninstrumented A.

Each leg has at least 40 restrictive and 40 non-restrictive changes. Record
publication/receipt, E command start/end, lease request/acceptance, each Apply,
completed/uncompleted guards, live and path verification, serving heartbeat,
first subsequent sibling egress and every loss inside/outside guards. Publish
all vectors and nearest-rank p50/p90/p99/max, including failure/censoring counts.
Use full run/boot/sequence joins and availability flags. Compare total outage,
not a guard-only percentile. No cause is inferred from CPU/cgroup correlation.

Acceptance requires no later revoked-source cutoff than paired A's permitted
cutoff/old D, no new/additional/stale binding admitted, no larger allowed
expiry bound, and improved sibling outage/loss distributions in both B legs.
A failed live timing gate triggers the D-133 comparison and remains unresolved
until attributed; no threshold inflation. Also inject delayed/replayed/wrong-
generation/empty leases, exhausted I/O budgets/Retry-After, desired supersession,
expiry during each stage, tc/nft/path failure, rollback failure, address/member
reuse, freeze during each stage, same-boot restart and reboot, and crash between
every kernel commit and durable write. Verify meter continuity, no duplicated
accounting, exact revocation cleanup, control access and honest observations.
Cross-compile amd64/arm64 and record real arm64 qualification separately.

## Reproduction and artifact boundaries

Run in the lane worktree, after checking `pgrep -af 'gates.sh|mk/at-'` and
recording `vmstat 5 7`; wait if two other independent suites are active and
recheck after startup. For an attributed D-133 experiment require the stricter
quiet conditions above. Use a fresh evidence directory for each run:

```sh
export TMPDIR="$PWD/.agent/tmp" GOTMPDIR="$PWD/.agent/tmp"
export CGO_ENABLED=0 GOTOOLCHAIN=local
export FVPN_LAB_PREFIX=fvpn-w2-agent-core-f31
mkdir -p "$TMPDIR"
go env GOCACHE
FVPN_F31_GENERATIONS=1 FVPN_AGENT_EVIDENCE_DIR="$PWD/.agent/f31-repro" \
  make -f mk/node.mk node-test-live-netlab
python3 tests/netlab/agent/generation-analysis.py .agent/f31-repro .agent/f31-repro/analysis.json
```

The ordinary live gate omits `FVPN_F31_GENERATIONS`. Restart repetitions use
`make -f mk/node.mk node-test-netlab` with
`NODE_NETLAB_TEST_ARGS='-run=^TestLab04_RestartKeepsVerifiedLease -count=30 -timeout=15m'`
and opt-in `FVPN_F31_RESTART_EVIDENCE_DIR` for the actual SQLite snapshots.
Do not add an unescaped trailing dollar through make: an initial invocation
selected zero tests and is retained as a harness failure, never counted among
the 30. The historical unchanged-main reproduction compiled the pinned main
agent separately while using this lane's diagnostic harness; its binary/source
receipts are retained. Earlier 30-run populations used the original stop helper.

`sha256.json` inventories the published native JSON/JSONL, derived summaries,
source receipts, sanitized qualification logs and host records. Byte copies of
members 12's inputs are pinned separately. Host process command text and absolute
workspace paths are omitted from public logs; complete private originals remain
under `.agent/f31/`. The historical A1 producer excerpt retains selected values,
not the original JSON byte formatting. No private keys or application payloads
are collected. Docker containers, images, networks and volumes are lane-prefixed
and removed after each run; final zero inventory is part of the hand-off.
