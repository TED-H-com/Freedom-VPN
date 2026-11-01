# Follow-up 32: shutdown, stale-lease holds and rejected candidates

This change addresses r18 M-1/M-2/M-3/L-2 and isolation follow-up 12 request 3.
It preserves D-135 and corrected D-136: a restrictive generation still installs
the baseline, and only a retained Apply error enables the verified deny-all
recovery candidate. Option (b) remains a design proposal. No packet loss is waived.
Relevant contracts are spec §5, §8.2, §13.6, §14.7 and §16.3; AT10 and AT13 remain
subject to their independent packet and platform acceptance evidence.

## M-1: cancellation ends control work

`Step` attaches the control budget before deferring `finishServingStep`. Every
Step observation POST uses that budget, which ends on caller cancellation or
the control timeout. Shutdown may still enqueue truthful local evidence for a
successor to retry. The independent local reads and baseline retain their own
shared deadline; SIGTERM alone is not evidence of a broken kernel policy.

## M-2: shutdown accounting

The local budget begins at Step entry, before its initial serving read. Its
maximum is `min(5s, watchdog_stale/3) + 5s`, **10 seconds** with the default 30s
watchdog setting. Initial/final serving reads, quarantine, baseline, persistence
and final observation all consume that one allowance. A read additionally has
its existing at-most-one-second bound. Quarantine gets at most five seconds
within the shared allowance and runs at most once per Step. Its pending latch
stays set for a subsequent Step; a second cleanup caller in the current Step
cannot create a fresh quarantine allowance. Other cleanup helpers preserve the
same enclosing deadline too.

`Close` waits at most **three seconds**, including checkpoint/handoff, both
usage-loop joins, database close and ledger-lock release. Its worker uses one
deadline; repeated Close calls reuse it. If it cannot finish, Close returns
`context.DeadlineExceeded` and the worker retains ownership until it finishes or
the process exits. It never releases the ledger while a sampler could still use
it. A timed-out Engine is terminal and must not be used for another Step.

The configured shutdown work bound from SIGTERM during a Step through Close is
therefore **at most 13 seconds** (remaining Step allowance ≤10s, then Close ≤3s),
plus scheduling and context-uninterruptible synchronous system-call overhead.
This is a cooperative bound, not a hard guarantee against a wedged filesystem
or executor. The installed service should set **`TimeoutStopSec=15s`**, allowing
two seconds beyond those budgets and providing systemd's forced-stop backstop.
`deploy/node/systemd/fvpn-agent.service` belongs to installer and is unchanged
here; until that change lands, no 15-second service cap is claimed. Kernel lease
deadlines and the independent watchdog still govern forwarding after process
death; no expired or torn checkpoint is made adoptable by this timeout.

## M-3: a distinct stale-lease hold and an alarm

Every decision with a generation mismatch carries `policy_missing` and a
`STALE_GENERATION` error with `stage=lease`, including when a retained safety
failure takes precedence in the lease/qos/source-binding checks (D-126). This
uses the existing OpenAPI vocabulary. Missing documents do not manufacture this
error. Safety/provisioning continue to describe baseline readiness and its ACK;
neither is a claim that member forwarding is enabled.

Once the acknowledged generation is ahead of the lease and forwarding is off,
the process tracks consecutive Steps and monotonic elapsed time. Every newly
constructed hold report includes the existing lease-stage error summary:
`Serving lease generation hold: applied=<decimal> lease=<decimal> steps=<decimal> elapsed_ms=<decimal> alarm=<bool>.`
It contains no identities, addresses or secrets. At **three Steps or 30 elapsed
seconds**, whichever comes first, the agent emits one warning and an immediate
observation transition with `alarm=true`. This is an alarm, not a cap that opens
forwarding, changes a signed deadline, or discards an ACK. The baseline persists.
These are diagnostic thresholds, not an availability SLA or a business grace.

Additional reports within a Step do not increment the streak. New desired
generations and repeated old leases do not restart it; wall-clock steps cannot
extend it. A matching lease, ended mismatch, revocation or identity freeze ends
this particular streak. A process restart starts a new local streak; central
must retain the incident across process boots. Pending report bodies keep their
original counters and timestamps under retry; alarm transitions get a separate
report ID. Outbox capacity and Retry-After still apply.

Wire-nodes should key generation-hold recognition on `policy_missing` plus
`STALE_GENERATION` in a lease check **or lease-stage error**, even when
`applied_config_generation == desired_config_generation`. Do not infer
convergence from that ACK or apply the one-shot `stale_generation_grace_at`
comparison as a hold timer. `PublicationDataPathFailure` should preserve this
specific classification before its generic `check_failed`/`forwarding_disabled`
fallback, without overriding unrelated safety failures. Central can join the
reported lease ID to its signed generation; the human-readable alarm summary
does not introduce a typed wire field. Central retention/escalation is a
wire-nodes task; this change does not edit its code or schema.

## L-2: the ceiling comparison

The [D-133 acceptance paragraph](followup31-generation-change-availability-2026-11-01.md#recommendation-and-d-133-acceptance-plan)
now requires upload and download ceilings for every surviving member and the
node to be no higher than paired A and the stricter authenticated old/new
limits. A higher ceiling fails the leg despite improved loss percentiles. The
adopted r18 predicate covers a raised sibling ceiling, raised node ceilings in
both directions, and a stricter new policy. No option-(b) code is introduced.

## ISO-F6-01: candidate transitions survive retry

Isolation's read-only [gap index](../e2e/at/isolation/iso-f6-01-observation-gap.json)
and [request 3](../e2e/at/isolation/requests.md#follow-up-12-requests--2026-11-01)
record run `1c25ee44`: desired generation 30 arrived at 09:56:19.013Z; seven
Applies failed before installation. The normal agent log records
`TC_FOREIGN_FILTER` at those attempts. The available verifier/tc surface does
not prove a successful installed-generation read. The generation-29 rejection
observed at 09:56:18.153Z was accepted at 09:56:41.080Z, after removal at
09:56:25.091Z. Repeated 502s preceded the drain; the receiver contains no
generation-30 rejection. Kernel samples stayed closed and preserved foreign
objects. That remains a missing observation assertion, not forbidden forwarding.

The agent should retain a separate rejection of the newer attempted candidate,
even if installation never completes and the applied ACK stays 29. Previously,
`observationState` keyed on the applied generation, forwarding, statuses, reasons
and check codes, omitting the rejected candidate. `observe` coalesced an equal
state while *any* older report was pending, even after `lastObs` was reset.
`flush` correctly retried immutable bodies in order with backoff. Thus equal
failure codes could hide generation 30 behind generation 29 indefinitely until
recovery. This defect reproduces independently with seven failed pre-install
Applies, a retained old rejection and a 60-second Retry-After.

The key now includes failed candidate generation/code and retained serving-read
generation/verification outcome. A failed Apply adds an existing `NodeError`
summary with `stage=apply_candidate`, its actual error code, the failure time,
and `candidate_generation=<decimal> verified=false`. This identifies the attempt
without inventing a verifier read or changing the applied ACK. Existing qos read
details, `tc_evidence` and first-containment provenance remain separate. Repeated
attempts of the same candidate still coalesce; the original body is not rewritten.
The regression drains both distinct bodies after control restoration and checks
their IDs, timestamps, schema and unchanged bytes.

The exact historical in-memory queue was not exported. Deduplication plus the
observed backoff is a supported mechanism, not a proof of every runtime step or
of #369 introducing it. The pending-state key/coalescing predates that patch.

For isolation's next independent sample, first require an accepted current
report after control restoration and confirm observation retries have drained.
Then keep the foreign filter present until the receiver records the dirty
candidate's actual `TC_FOREIGN_FILTER` rejection with forwarding disabled and
an observation time inside the injection interval. The report's applied ACK may
remain on the prior generation. Join `apply_candidate` to the original command
trace for a pre-install failure; do not substitute it for installed-verifier
evidence where an assertion specifically requires that read. If no actual
foreign-filter error was observed, report the real error and keep that predicate
blocked. Neither a local failure nor later clean recovery is a substitute for
central acceptance. No longer timeout or relaxed assertion is prescribed.

## Proof boundaries

All agent r18 scenarios are adopted; the four findings assert their corrected
behavior. Passing r18 safety/restart/config proofs stay intact. Additional
follow-up-32 tests exercise the shared final-read deadline, Close ownership at
timeout, monotonic alarm/recovery, safety-code precedence and retry retention.
Netlab owns r18 L-1 and `node/enforce/review_r18_lifecycle_test.go`; neither is
adopted here. Cross-compilation does not establish physical-arm64 acceptance.
