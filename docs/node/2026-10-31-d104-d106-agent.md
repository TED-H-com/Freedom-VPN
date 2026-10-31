# Agent D-104 / D-106 / D-118 — 2026-10-31

Follow-up 24 delivered the fail-closed recovery candidate, final expiry guard,
retained applied model, historical tc rejection rows and revision 13 QoS detail.
It deferred the entire D-104 consumer because the contract was absent when that
work began. Follow-up 25 consumes the now-published deadline and evidence APIs.

## What the agent sends and installs

For §16.3 / ISO01, `buildModel` keeps the absolute monotonic minimum of the
member lease, entitlement, whole-node lease, offline cap, desired-policy expiry
and applicable binding expiry. `MemberLease.DeadlineMono` and `Model.NowMono`
share the lease request's CLOCK_MONOTONIC anchor and its signed wall snapshot.
Multiple bindings for one member use their earliest limit. Each pointer owns
its value. The agent no longer floors remaining time to seconds or sets the
legacy `TimeoutCeilSeconds` cap. Pure rendering is tested with 1.9 seconds,
900 milliseconds, 1 millisecond and less than a quantum remaining.

Production still uses `enforce.Apply`, with the D-104 1 ms quantum and 100 ms
installation reserve. Every Apply installs and releases a member-only guard;
the 100 ms value is a checked budget, not a fixed 100 ms sleep. A late install
stays guarded. Repeated renewals therefore have real interruption intervals.
The retained applied model keeps its original sample for live verification.
A read spanning expiry cannot emit a new serving heartbeat. See the
[enforcement contract](../poc/G2-deadline-contract-2026-10-31.md).

For D-106, Apply, live serving reads and blocked audits retain
`ServingEvidence.TCReadbackEvidence()` before diagnostic serialization. Reports
set `tc_evidence` from that authoritative snapshot, or explicitly null before
any retained read. A call that performs no read does not erase a previous real
read. Candidate generation is a uint64 decimal string; `read_at` is the actual
UTC verifier completion time. Outcome describes the complete verifier, while
outer Apply, member-path, lease, deadline and heartbeat gates still control
serving and ACKs. Clean tc with a failed nft check is not a verified read.

The existing `errors[stage=tc_evidence]` transport is unchanged. A new clean
snapshot does not remove its historical rejection or rewrite its time. Bounds
are 16 attachments, four objects per attachment, four actions per object and
one deciding object. Tokens follow the existing redaction rules; no selectors,
action parameters, command output, counters or destinations enter the field.
The digest is the unchanged netlab RFC 8785 SHA-256 projection of
`{reads, objects, truncated, tc}`; generation, time and outcome are outside it.
The four contract ObservationReport examples round-trip through generated Go
types, including max uint64 and null. See the authoritative
[G2 mapping and six recovery assertions](../poc/G2-followup14.md).

## Recovery and usage hand-off

ISO-F6-01's original circular dependency was retained Apply error → suppressed
ACK/renewal → stale/grant-less lease → no clean Apply. Follow-up 24 already
breaks it with a deny-all candidate, for unchanged and newer generations.
Follow-up 25 also fixes the remaining generation-driven renewal suppression:
while an Apply error is retained, `needRenew` compares the lease with valid
**desired** generation. The request still omits the applied-generation ACK;
acquiring a signed lease cannot acknowledge unverified forwarding. The real
lab exposed this when a newer topology candidate required a matching lease
before it could pass D-100. Dirty tc still blocks both serving and the new ACK.
A clean full verifier and restored retained topology are required before a
candidate clears the error. Only a matching populated lease plus another
verified Apply permits member forwarding. No signed generation is fabricated.

D-118 activates `RetirePreviousAt: e.clock.Wall()` in `ensureUsage`. The test
obtains a new certificate for the same node, preserves the real final meter
sample, restarts after identity installation, uploads the original canonical
payloads first with `origin_incarnation_id`, and verifies per-stream ACKs and
current sequence 1. An unchanged-incarnation restart creates no retirement.
The appended [usage upload note](usage-upload.md) records the activation for
ENR-05. The TLS fake-control fixture does not replace central D-112 window
validation or assembled reinstall acceptance.

## Assertions for isolation follow-up 8

1. Capture initial serving generation, deciding tc attachment/object and actual
   rejection time, quarantine and a non-serving heartbeat.
2. Remove every injected object, including clsact; retain a fresh read proving
   absence. A filter deletion alone is the negative control.
3. Show actual clean Apply and lease renewal for desired generation, including
   the deny-all candidate and pending transitions. Keep the dirty-generation
   control non-serving and unacknowledged; enforcement must not delete it.
4. Preserve successful Apply and fresh live D-106 snapshots with matching
   candidate generation, actual read times, verified=true, null error/rejection
   and recomputable digest. Keep the old rejection explicitly historical.
5. Prove retained member-path restoration, marker clearance, serving heartbeat
   and observation, applied generation and counted member packets. Name the
   route/address owner. Repeat after a same-generation node restart with a
   transient boot tc object and provisional lease; never force a generation.
6. Measure renewal availability including every guarded Apply. Capture signed
   times, lease send anchor, absolute D and Apply/read timings on one monotonic
   clock. Require last positive ≥ D − 101 ms, first expiry drop ≤ D + 100 ms,
   no positives at/after D and no later positives after expiry's first drop.
   Classify earlier renewal guard losses separately and include them in the
   availability denominator. A phased 100 ms expiry trace and dense renewal
   probes serve different measurement purposes.

## Measurements

The final process lab passed on 2026-10-31 at 10:35 UTC. Injection to contained
read took 953.727 ms; removal to clean link-up took 508.334 ms; removal to counted
serving observation took 2.016 s. Applied and lease generations were both 3.
The old rejection remained, and both initial and recovered clean reports passed
the D-106 field/digest assertions. Meter equality was 12,112 bytes, with the
contiguous ACK through sequence 3. These elapsed read/command observations are
upper bounds, not kernel transition timestamps or production SLOs.

The final live run passed at 10:40 UTC on amd64, kernel `7.0.0-38-generic`,
`nftables v1.0.9`, `iproute2-6.1.0` / libbpf 1.3.0:

| Measurement | Result |
| --- | --- |
| Same-generation restart | generation 2; three clean Apply attempts; 1.419 s from foreign-object removal to serving, including explicit veth restoration |
| Renewal configuration | TEST ONLY 60 s lease, 30 s renewal lead with existing jitter; one-second Engine.Step cadence |
| Renewal availability | two ordinary timed renewals; 75 guarded Applies over 76.080 s; 74,796 / 76,000 probes delivered; **98.4158%** |
| Guard correlation | all 1,204 losses had ingress timestamps inside guarded install-start → release-completion command brackets; none outside |
| Guard command bracket | median 31.498 ms; maximum 55.215 ms; these brackets include subprocess/read completion overhead |
| Deadline continuity | last positive **D − 100.651423 ms**; first dropped send **D − 0.719753 ms**; 27 positives / 7 drops; no post-D or post-first-drop positives |
| Final heartbeat | non-serving after expiry; separate deterministic read-spanning-expiry regression passes |

The renewal trace uses one-millisecond probes and includes every Apply guard;
the final expiry trace uses the D − 101 ms phased 100 ms cadence required by
D-104. No additional cadence allowance is added to the early-expiry bound.
Command-bracket correlation does not identify the exact kernel drop instant.
The measured availability is for the explicitly stated fast lab cadence; it
must not be presented as production availability or uninterrupted forwarding.

The first live attempt passed same-generation restart but exposed the remaining
desired-generation renewal suppression; that production path is now fixed.
The next trace stopped before its second timed renewal (one renewal, 65 Applies,
64,968 / 66,000 delivered); its count assertion failed and the trace was retained.
Extending the measurement to 76 seconds satisfied the unchanged two-renewal
assertion. No deadline, packet or foreign-state assertion was relaxed.

Gate logs, command timestamps and complete packet/guard traces are retained in
the lane's ignored `.agent/f25-*.log`, `.agent/f25-*.json` and
`.agent/f25-live-evidence/`; `d104-renewals.live2.json` is the shorter failed
measurement, `d104-renewals.json` is the final measurement, and
`d104-expiry.json` is the final one-clock expiry trace. The launcher recreates
the final trace and removes its own Docker container/image on either outcome.

## Measurement scope

The reproducible gates are `node-test-netlab` and `node-test-live-netlab` in
`mk/node.mk`. The latter includes real nft/tc, TLS leases, Engine.Step,
same-generation replacement, renewal guards and deadline packets. Its veth
address/route restoration belongs to the harness, not NetBird or the agent.
Other kernels/tool builds, physical arm64 and suspend/resume remain 待實測.
Both Linux architectures cross-compile; this does not qualify arm64 hardware.
The assembled native NetBird restart, continuous crash/reboot/updater matrix
and isolation/member AT results remain their acceptance owners' reruns.
