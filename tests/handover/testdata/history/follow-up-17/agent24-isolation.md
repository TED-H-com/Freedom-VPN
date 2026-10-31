# Follow-up 24 to isolation follow-up 8 — 2026-10-31

The ISO-F6-01 latch was a circular dependency. A retained `TC_FOREIGN_FILTER`
blocked a new generation ACK and its renewal. The candidate decision ignored
that error but still rejected the old-generation lease, so no verified Apply
could clear the error.

Recovery retries on the next ordinary Step, for both the same and a newer
desired generation. With a matching populated lease, the full serving model
is retried. With a stale-generation or grant-less lease, a local deny-all model
is installed and verified first: no bindings, no member grants, no member
forwarding. This checks nft/tc and the retained topology before ACK and renewal.
It does not modify either signed envelope. Forwarding requires a matching
signed lease and a second full Apply with a fresh serving read. A retained
foreign attachment rejects both paths and prevents the new ACK. The old
`applied_config_generation` remains historical applied-state evidence while
forwarding is false; renewal requests omit the generation while Apply is failed.

The unit reproduction now requires Apply delta at least one, renewal delta at
least one, lease generation 3 and serving after read-back. The companion tests
hold the foreign filter through repeated attempts, then remove it at the same
or a newer generation. No route/address writer was added. Under D-108 NetBird
owns its native IPv4 topology; the agent waits for retained hashes and peer-route
checks. D-100 still requires a clean verified newer published generation before
refreshing expectations. A deny-all recovery candidate cannot refresh them.

## Revision 13 evidence

`checks[name=qos].detail` contains the retained verifier's read time, candidate
generation, origin (`live` or `apply`), actual verification outcome and
`ServingEvidence.ReportDetail()`, within 500 characters. Its check status and
error code still come from the real decision. The timestamp is a read-completion
time, not the report time. Apply's executor captures tc read completion before
rollback; a later live serving read replaces that snapshot. Blocked/rollback
audits do not masquerade as a clean serving verification.

The first rejection's bounded attachment/action rows remain in
`errors[stage=tc_evidence]` after recovery, with their original candidate generation
and read time. They are historical report-only errors, not a success-shaped
NodeError. Quarantine snapshots remain separately identified. This uses the
revision 13 surface; no `ObservationReport.tc_evidence` field was added.

## Kernel proof

`TestLabISO_F6_RecoveryAndEvidence` in `tests/netlab/agent` runs the real agent,
TLS fake control and real nft/tc. It checks the first counted observation's clean
qos evidence, injects a foreign redirect, confirms member-down containment, holds
the filter across a newer generation, removes it, observes the topology hold,
lets the topology owner restore its saved IPv4/IPv6 routes and IPv6 address, and requires
counted forwarding plus matching applied/lease generations and retained failure
rows. This veth fixture's owner is the harness, not NetBird. Its timestamps are
host monotonic durations including Docker commands and read completion, so they
are upper bounds, not a production containment guarantee. It does not claim zero
escaped packets between injection and containment.

The final full suite passed on 2026-10-31 at 09:51–09:52 UTC. Injection to
contained read was 897.120435 ms; removal to clean link-up was 494.491529 ms;
removal to counted serving observation was 3.161412923 s. Applied and lease
generations were both 3. The initial counted observation carried a live verified
generation-2 read at 09:51:10.583689753Z; recovery retained the historical rejection.
The same suite recorded exactly 12,112 admission bytes, contiguous ACK through
sequence 4, and an external-reset gap followed by recovery in 1.850 s.

The separate live-read lab passed at 09:52 UTC: 90 reads over 30 Steps,
15.812 ms mean read, 16.823 ms p95, 17.471 ms maximum; combined live and topology
verification cost 49.879 ms per Step. Its injected next-Step withdrawal took
155.650 ms. The explicit bypass test counted one uplink packet before quarantine
and zero afterward, with node uplink control and TLS observation still passing.
The native NetBird recovery evidence remains D-108's G2 follow-up 14 campaign;
this mechanism test does not promote assembled AT10–AT13 or physical arm64.

The harness restores both address families after its injected quarantine. Its
health journal now has the required private 0700 directory mode. Existing
meter/watchdog cases start the watchdog only after a fresh serving heartbeat;
the crash/loss/timeout assertions remain. Ordinary cases run a lab topology
owner that restores saved addresses and peer routes only while the link is
already UP. It never raises a quarantined link. This handles fail-closed
quarantine during a canceled shutdown read on the static veth fixture. The
ISO-F6 injection test leaves that owner stopped and requires its explicit
post-Apply repair, preserving the deciding topology-hold assertion. This is
harness behavior and does not claim native NetBird or production serialization.
The unpublished case requires an observation newer than its publication change;
revocation runs last because that incarnation cannot be reused afterward.

## D-087 safe provisional recipe

Use a clean, unfrozen node with a passed baseline, a working meter and a passing
real tunnel probe. Keep the member interface and its owner's routes intact.
Do not combine this transition with foreign tc injection, NetBird restart,
clone detection, a missing resolver or an invalid probe target.

1. Establish a verified published generation with desired device bindings.
2. In the isolated fake-control fixture, call `SetMemberGrantsReady(false)` and
   allow a normal renewal (or restart the agent, which requires a fresh lease).
   Keep that generation and publication unchanged. Do not delete bindings or
   advance the policy generation to create the transition.
3. Capture every observation POST. The qualifying report is ready/passed,
   forwarding false, `awaiting_member_grants`, and exactly the lease-stage
   `AWAITING_MEMBER_GRANTS` error. Failed tunnel/safety checks do not qualify.
4. Restore `SetMemberGrantsReady(true)` and let the scheduled renewal admit
   members. Keep grant expiry and control time valid throughout.

This is a deterministic mechanism recipe, not a private central database edit
or a claim that production provides a grant-suppression API. An assembled
consumer needs an equivalent supported control fixture from the central owner.
Use a fresh node for the single-error predicate: historical rejection rows are
intentionally retained on a node that previously experienced a tc incident.

The agent now forces an observation on semantic state changes, including a
non-serving-to-provisional transition. Pending copies of the same state coalesce
behind the original deciding report. Failures stop that flush after one failed
POST; later flushes honor Retry-After and bounded exponential jittered backoff.
The lease schedule has its own budget. Central rate limits are unchanged.

## D-104 and the earlier expiry trace

`docs/poc/G2-deadline-contract-2026-10-31.md` was absent from origin/main
`37fcaf7` at task start. D-104 waits for netlab follow-up 15. The allowance and
follow-up 23 diagnostic assertions remain unchanged; ISO-F6-03 stays open.

The isolation follow-up 7 last SYN-ACK at -229 ms and first failed SYN at
-177 ms bracket an early cutoff relative to signed wall expiry. Integer kernel
allowance and the conservative monotonic anchor can produce an early cutoff;
those two timestamps alone do not separate their contributions. The baseline
read at +1.303–1.325 s and first heartbeat read at +7.541–7.571 s bracket later
observations, not the instant of the last allowed packet. Control HTTP work can
delay Step completion and its heartbeat under control loss. Follow-up 23 added
the deadline wake/final-read guard, but cannot establish sub-second continuity
without the renderer/install-latency contract. Do not reinterpret the failed
one-second-tick acceptance predicate as passing.

The unchanged diagnostic in this follow-up's live lab again reports the gap:
last positive packet at -927.430422 ms and first dropped send at -827.627804 ms,
with 100 ms cadence (19 positives, 15 drops). The diagnostic passes as a
reproduction; strict near-deadline continuity remains unproved and open.
