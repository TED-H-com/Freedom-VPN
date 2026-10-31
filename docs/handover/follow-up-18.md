# Follow-up 18 — operator evidence through #324

Verified against main `6d0022d83345ae78d26aaaf31c64f026d38dea3b`, 2026-10-31 EDT.
This cut reconciles #316–#324, including updater 17 (#323), which was already
merged when this task started. `TestHandover_Followup18Claims` pins the claims
under D-098 / D-109: frozen owner snapshots, run-ID evidence, roadmap rows and
Go test names. Handover ran no new live acceptance.

## Restore closure (§17.3, AT30, DATA01, MTR03)

Security-ops follow-up 13 (#316) closes **SECOPS-21 on the assembled stack** in
restore run `5218b7d5-1415-4e1b-8618-48ea2d3cf03e`. Devstack 18's exports
captured the first delivered offer: generation **19 > retained 15**, complete
HTTP 200 write, matching local store digest, accepted/applied generation 19,
and an applied **202** receipt joined to central by report ID. The capture was
armed once; no generation repair or substitute compiler/probe row was used.
This is **harness-proved on the assembled stack, not a live-fleet guarantee**.

AT30 passed: pre-replay database **RPO 92.163 s (−14.287 s)** and total outage
**RTO 81.016 s (+6.486 s)** versus follow-up 12's published rounded comparison
baselines (106.45 / 74.53 s). Exact current measurements are 92.163101 and
81.016113709 s. Post-replay service usage RPO is zero. These are shared-host
measurements, not production objectives. Follow-up 12 is archived byte-identically.
Full AT29 remains blocked on the independently clocked disposable guest and
manual grant-withdrawal export; full AT31 remains blocked on candidate hardware
and the 20-member/60-device mixed-relay campaign. The devstack export's own
4 > 3 proof is separate from this 19 > 15 restore closure.

Sources: [owner handover](../../tests/handover/testdata/history/follow-up-18/secops13.md),
[closure](../../tests/handover/testdata/5218b7d5-1415-4e1b-8618-48ea2d3cf03e-SECOPS-21.json),
[AT30](../../tests/handover/testdata/5218b7d5-1415-4e1b-8618-48ea2d3cf03e-AT30.json).

## Restart reports and central receipts (§8.2, §14, AT09)

Members follow-up 9 (#317), run `11e305fe-6360-4dbd-95b1-67274aa4988c`, closes
**F8-01 in the original generation-30 sequence**. Desired/applied generations
stay 30 → 30; fresh serving returns in 15.728602 s and counted traffic in
21.373814 s, without state, topology or generation repair. Follow-up 8 is
archived and adopted; all 13 handover snapshots matched.

**AT09 is failing on F9-01**, with 3 complete cases passed / 6 blocked / 1 failed
in the lane. The restarted agent's report carries both `safety_check_failed`
and `awaiting_member_grants`: lease/qos/source_binding per-check codes are
`TC_FOREIGN_FILTER`, while the stage = lease error is `AWAITING_MEMBER_GRANTS`.
D-126 declares those reasons valid and cumulative. Per-check codes carry the
most severe blocking condition: **safety failure > provisional / awaiting > ok**;
a stage error carries that stage's own cause. Retain both reasons unchanged.
Agent-core 27 must confirm the composition or fix a reporter race; members 10
must relax the D-067 revision-11 predicate to D-126 and rerun acceptance.

Wire-nodes follow-up 32 (#321) proves the central half from that exact captured
AT09 report. `SubmitObservation` accepts it, stores and projects every value
unchanged, keeps publication blocked, and replay returns the stored ACK. The
HTTP response remains **202 Accepted** per the contract; it is a receipt, not
completed publication or a serving guarantee.

**D-128:** stored unchanged means complete JSON-value equality, byte-identical
JCS output and the same SHA-256. `report_payload` is JSONB: literal whitespace
and key order are not retained; no schema change is planned. D-113 is unchanged.
No revision-15 first-delivery marker is needed for SECOPS-21.

Sources: [members summary](../../tests/handover/testdata/history/follow-up-18/members9.md),
[AT09](../../tests/handover/testdata/11e305fe-6360-4dbd-95b1-67274aa4988c-AT09.json),
[captured report](../../tests/handover/testdata/history/follow-up-18/mixed-observation.json),
[central HTTP/storage proof](../../internal/nodes/httpwire/followup32_integration_test.go).

## Native transport and renewal timing (§8.3, AT09, AT13)

Netlab follow-up 19 (#318) leaves the native **11,011 / 64,064-byte F8-02
reconnect failure unattributed**. Handshake MSS is unchanged across the observed
exit hops: member/origin advertisements 1240/1460 in the failed and successful
captured connections. The kernel-only path passed with zero tc drops and
fragmentation-needed traversal in both directions; it neither reproduces nor
clears the native failure. Its exit-link bottleneck differs from the native
member-underlay path. **No single transport cause is established.** Missing
ordinary TCP data/ACK/retransmission and outer-fragmentation headers prevent
causal attribution. No observed ICMP need-fragmentation is not proof of
suppression. A later 64,064-byte pass does not erase a partial transfer.

`ServingEvidence.TimingEvidence()` is the producer-timing surface: a detached
local sibling of D-106 with Apply/live-verify start/end, final render, guarded
install/release command brackets, each verifier and its nft/precision/tc reads,
and the matching D-106 digest. Failed/partial operations and both recovery
verifiers remain visible; nanoseconds are decimal strings. It adds no HTTP
field and performs no I/O inside the guard. Netlab's zero-loss runs **do not
clear isolation's 17 historical renewal losses**. Isolation 9 must correlate a
new assembled capture with actual producer brackets; missing or ambiguous
matches stay unproved. D-120's AT10 allowance does not apply to that diagnostic.

D-123 keeps the v1.0 overlay **IPv6Block by design**, including both IPv6 drop
rules. Native IPv6 underlay selection is a separate measurement; those labels
do not prove IPv6 overlay payload or a policy cause for the native failure.

Sources: [native attribution](../../tests/handover/testdata/history/follow-up-18/native-attribution.md)
(from [netlab's dated note](../poc/G2-f802-native-attribution-2026-10-31.md)),
[producer timing](../../tests/handover/testdata/history/follow-up-18/producer-timing.md)
(from [netlab's hand-off](../poc/G2-followup19-netlab-agent-2026-10-31.md)).

## Checkpoint identity and ledger ownership (§10.5, §13.6, MTR03, UPD05)

Agent-core follow-up 26 (#320) implements **D-125**, a product mechanism:
every checkpoint names the latch identity it finalized against.
`MeterCheckpoint.BarrierID` (`barrier_id`, omitempty) echoes the exact UUID the
updater wrote to its latch file. The agent validates the opaque spelling and
repeats verified member withdrawal and durable finalization for every new
identity, pins the checkpoint bytes across same-identity restart, and removes
the evidence before resume. A new latch cannot be satisfied by relabeling old
bytes; absent or malformed latch content publishes no checkpoint.

Updater follow-up 17 **is merged (#323) at this cutoff**. It requires a valid
UUID byte-equal to the persisted latch identity; fresh and late-within-bound
component tests succeed, while foreign, malformed, withdrawn and other-boot
evidence fails closed. Component kernel/service/health adapters do not prove a
live serving-node upgrade. Devstack 19's serving-node rerun is **not yet proved**.

**D-118 locking:** devstack 18 request 2 identified a real gap in #312.
Adoption of `RetirePreviousAt` now occurs under the node lock, and a separate
`usage.db.lock` is held for the lifetime of the meter / uploader / store.
A replacement cannot open or retire until the predecessor finishes final
sampling, cancels and joins its loops, and closes the meter and SQLite store.
The follow-up 26 chain resume fixed only the D-118 ENR-05 test: cancellation
after central acceptance but before ACK delivery legally retries one key and
one body digest with one accepted event. **Runtime unchanged by that resume**.

Sources: [agent producer](../../tests/handover/testdata/history/follow-up-18/agent26.md),
[updater consumer](../../tests/handover/testdata/history/follow-up-18/updater17.md),
[locking and retry proof](../../node/agent/followup25_usage_test.go).

## Bounded installer corrections (§13.10–§13.11, UPD05)

Fleet-update follow-up 12 (#322) implements **D-127**, closing r17 M-1 in
PostgreSQL tests. On the same incarnation, a later eligible `succeeded` after
an applied D-111 pre-health seed, or `rolled_back` after `succeeded`, is one
bounded correction that moves `current_release_id` / `last_good_release_id`
under the unchanged generation fence. Below the central high-water, evidence
is accepted as history without moving those pointers. The first receipt stays
immutable history with its original ACK flags; the correction has its own
evidence and ACK. Replay of either payload returns its stored ACK without
reapplying a historical version. Every other distinct outcome, including a
third payload (even seed → success → rollback), remains `ACK_CONFLICT`.

**Deploy migration 00192 before the new control binary.** Its payload-derived
UUID `result_id` key and unique `outcome_sequence` bounded to 1..2 preserve the
first receipt and cap each node/job at two rows. `Down` refuses atomically once
corrections exist. Retain the migration and use a forward fix in that case.
This is **harness-proved on PostgreSQL tests**. Live install → seed → success
and success → rollback captures are devstack / updates work **not yet run**.

Source: [installer outcome contract](../../tests/handover/testdata/history/follow-up-18/fleet12.md).

## Live usage and review provenance (§10.5–§10.8, MTR02–MTR04)

Usage AT follow-up 5 (#324) reports **AT14–AT20 pass in live run 170ea3e3**
(harness 3a59beb4, product base 7e044c71, including #312). It exercises the
retired-stream hand-off on a controlled installer exit, preserves the immutable
tail across reinstall, and proves AT20 same-key replay with independent origin
and current ACKs. AT20 is the usage lane's ENR-05 extension, not the DNS case
numbered AT20 in §19. Follow-up 4 is archived. Fixture provenance is recorded only in the decisions
register.

The member total is **33,750 + 7,782 = 41,532 bytes**, origin ACK 2 and current
ACK 0. Canonical retained payloads, hashes and ACKs stay invariant; usage DB/WAL
file hashes can change when bootstrap activates D-118. The nonblocking
RESPONSES.md step-4 correction remains with devstack. Live over-window rejection
and audit, retired-certificate refusal and retained partial revoke-operation
status are not replaced by inferred completion.

Handover follow-up 17 (#319) reconciled through **#313**, including #310/#311,
then reconciled #316's scope manifest on resume. Its eleven subtests include
`post_cutoff_secops_notice`; the merged manifest held **242 digests / 141 paths**.
Its SECOPS-21 closure and AT30 numbers were reserved for this follow-up.

The [r17 archive](../reviews/r17-rev14-d104-d117-review.md) is byte-identical to
`.agent/REVIEW.md` at `origin/r17-rev14-d104-d117-review`, commit
`dcabc41e7d0c12725b1a7090425e2831faa3a12e`, by grok/grok-4.7 from main 62f24e6d.
One finding, M-1 (Medium), routes to fleet-update 12 under D-127; r16 H-1 / M-1 /
M-2 / L-1 were re-run and held. Its “Not examined” list records register drift.
The current register and claims now explicitly pin D-118 activation in #312,
locking in #320 and the usage acceptance in #324; earlier pending assertions
are confined to labeled historical snapshots.

Wire-nodes 32 recommends D093-12 option (ii): an explicit
`nodes.release_fetcher = fixture` selector compiled only into a dev-tag worker.
This is **design advice, not implemented**; the documented SQL shortcut stays
labelled. It does not change the production fetcher or retire D093-12.

Sources: [usage summary](../../tests/handover/testdata/history/follow-up-18/usage5.md),
[AT20](../../tests/handover/testdata/170ea3e3-AT20.json),
[review dispositions](../reviews/README.md#r17-disposition-at-the-follow-up-18-cutoff).

## Still open at this cutoff

| Work | Boundary / consuming owner |
|---|---|
| AT09 F9-01 | Agent-core 27 confirms composition or fixes a reporter race; members 10 relaxes the D-067 predicate to D-126. #321 proves only the central half. |
| F8-02 | Native reconnect failure remains unattributed; devstack / netlab need the missing captures. |
| Devstack 18 serving-node updater handshake | Measured on pre-#307 binary source `84b6a342`, before #307's F9-03 exit codes and #310's fail-closed handshake. Historical sealed failure remains; devstack 19 rerun is not yet proved. |
| Devstack 18 post-upgrade re-enrollment seed `HASH_MISMATCH` | Updater 17 reproduces it with the pre-#307 extractor and verifies #307's fix; live updater 17 / installer 6 consumption remains open. |
| Devstack 18 native MTU 1280 in both families | Historical failures remain; D-123 does not dismiss IPv4 payload failures over either selected underlay family. |
| Five AT07 rows | Enrollment 8 consuming rerun; usage #324 does not promote enrollment's full case. |
| Isolation's 17 historical renewal losses | Isolation 9 correlation using `TimingEvidence()`; new zero-loss lab runs do not clear them. |
| AT29 / AT31 | Independently clocked guest / 60-peer candidate hardware and remaining full campaigns need resources. |
| Qualification | Real Windows, macOS, iOS and Android clients; physical arm64 and power-loss qualification remain unproved. |
