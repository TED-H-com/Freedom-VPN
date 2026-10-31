# Follow-up 19 — operator evidence through #337

Verified against main `57a8cc9fc2ddb8178d3166642536dad6a1c3b95b`, the updates 10
cutoff on 2026-10-31. Handover ran no new live acceptance. Counts below are
pass / blocked / fail unless stated otherwise. The
[13 roadmap rows](../../tests/handover/testdata/followup19-main.json) and
[37 source pins](../../tests/handover/testdata/followup19-snapshots.json) preserve
this cut under D-098 / D-109; later main changes do not promote these results.
The [decision and findings register](README.md#follow-up-19-decisions-register-2026-10-31-edt)
records current dispositions. Follow-up 18 remains a dated historical cut.

## Observation and restart evidence — #325 / #328

Agent-core 27 (#325) confirms D-126 as a deliberate single-decision composition,
with no reporter race: the captured AT09 report carries both the safety verdict
and the provisional lease state. Reasons accumulate; per-check precedence is
safety failure > provisional / awaiting > ok, and each stage error retains its
own cause. The captured report is replayed by
`TestD126_AT09_CapturedProvisionalSafetyComposition`. D-128 continues to mean
complete JSON-value equality, identical JCS and SHA-256, rather than JSONB
whitespace retention. A mixed report does not authorize forwarding.

The [producer hand-off snapshot](../../tests/handover/testdata/history/follow-up-19/agent27-isolation.md)
records `ServingEvidence.TimingEvidence()` immediately after every real Apply,
live VerifyServing and blocked audit in the opt-in TEST ONLY measurement
surface, including failures, run/boot identity and clock anchors. D-087 already
has a qualifying producer observation: no producer fix was needed. Its unit
and agent/kernel/TLS fake-control proofs do not close assembled AT10; the
missing central grant-eligibility seam is addressed by #335 below. The roadmap
also records 12/1,518 process-lab renewal losses, ten loss windows overlapping
install commands, median/worst outer guards 32.271/51.114 ms, without D-120.
These scoped numbers do not retrospectively timestamp isolation's 17 older losses.

Members 10 (#328), run `2be2d32f`, closes **F9-01** under D-126 / D-128: the
predicate accepts the cumulative report while still requiring active safety
evidence. Fresh AT09 remains **blocked, 38/5/0**, with desired/applied **29 → 29**
in the unchanged mutation sequence. Its exact-generation concern F10-04 led
to **D-129**: qualification requires desired == applied across the restart,
under the D-126 predicate, at whatever generation the unchanged sequence yields.
**30 is not a contract value.** This is a test-acceptance definition, not a
product change or a full AT09 pass; the original generation-30 F8-01 closure stands.
The [merged D-129 paragraph](../../tests/handover/testdata/history/follow-up-19/D129-PR328.txt)
supersedes the lane's historical request for an exact-30 reproduction.

All four native bulk rows pass (**4/4**), including the dual-underlay MTU-1280
row; the original F8-02 remains unattributed. New members findings are F10-01
(the source pin corrected by netlab 20), **F10-02** (AT02 **121/0/1**, one
unreadable persistent-expiry sample among 234, devstack 20), and **F10-03**
(AT20 **80/0/1**, positive answer changes before TTL, dns-publish 9 / devstack 20).
The unreadable sample does not establish a post-expiry leak; the DNS capture
does not identify the responding resolver. Neither failure is dismissed.

## Installer and serving upgrade — #326 / #332

Installer 6 (#326) consumes ENR-05 and passes **13/13 privileged tests**. The
journaled upgrade uses a real correlated FileMeter success with D-125
`barrier_id`, then post-switch revocation and re-enrollment seed pass on the
combined source without tree repair. The devstack 18 seed `HASH_MISMATCH` was
reproduced with the pre-#307 extractor; the exact mismatching entry in that old
live tree remains an inference because its per-entry inventory was not retained.

Devstack 19 (#332), serving run `d6aaa798`, passes **14/0/0**: byte-equal
correlated barrier, real withdrawal and durable finalization, apply exit 0,
candidate switch, all five health checks, sealed receipt and candidate seed
without `HASH_MISMATCH` or tree repair. This closes the devstack 18 serving-node
failure and its pre-#307 binary caveat, and supplies the successful candidate
seed rerun. Successful local apply does not assert republication.

The separate stopped-agent sequence remains **13/0/1**: without the producer it
cannot answer the fresh latch, seals `CHECKPOINT_FAILED`, apply/replay exit 1,
recover exit 2, and never switches. Its unchanged-tree seed success is distinct
from the serving run's candidate seed. Two native campaigns pass **7/0/0** each;
the broader DNS campaign is **15/0/1**. Exact NODATA + SOA TTL 5 passes, but
MTU-1280 bulk stops at **12,012 bytes**. F8-02 remains unattributed; receive-link
MTU rejection is an inference. D-123 keeps IPv6Block in the overlay and retains
native underlay failures. D-118 origin retention is **8/8**; D-093 has **15 agreements**.
D-087 export was blocked on the central seam. The D-124 brief named a file
without a `Decide` call on that base: a process fact, not a product claim.
[Combined-source snapshot](../../tests/handover/testdata/history/follow-up-19/devstack19.md).

## Enrollment and immutable usage — #329 / #330 / #331 / #333

Enrollment 8 (#329) recorded AT06 **501/0/1** (the sole failure is the shared
AT07 process exit) and AT07 **43/1/2**. Four of five D-116 / D-105 rows closed.
`controlled_reinstall` remained blocked by ENR-07: a 1 ns gap collapsed at
PostgreSQL microsecond precision against `usage_events_interval_check`.
The proof first called its source `planSwitch`; enrollment 9 corrects it to
**planBaseline**, the unknown-continuity baseline path. The failed run remains
unchanged in its 51-file follow-up 8 archive.

**D-130 has both halves merged.** Usage-agent 5 (#330) uses one `eventInterval`
helper in `node/internal/meter/delta.go`: UTC whole-microsecond endpoints with
width ≥ 1 µs at all four former 1 ns sites — planBaseline, planSample,
planClockGap and planSwitch. Sealed events and hashes are untouched.
Usage-central 5 (#331) stores microsecond projections of the original nanosecond
canonical values. Strict canonical order remains validated as `interval_order`;
equal or reversed canonical values remain invalid.

Deploy `00193_usage_events_interval_precision.sql` through the normal migration
runner before ingestion resumes. It changes the stored checks on usage_events,
usage_samples and usage_ledger_gaps to **<=**, preserving finite/presence guards;
constraints are changed by migration, never bypassed. **Down refuses equal-endpoint
rows** atomically. Do not delete evidence to downgrade. Unexpected constraint
failures remain retryable **503 DEPENDENCY_UNAVAILABLE**, with no ACK and no
definitive rejection of an immutable event because of a storage failure.
After deployment the already sealed valid 1 ns event is accepted on its next
retry, without operator action on the spool, re-enrollment, epoch rotation or
event rewriting/skipping. Central tests prove ACK 1 → 2 and once-only 908-byte
attribution for that old payload; the fresh live run below proves the new path.
[Central rule and rollout](../../tests/handover/testdata/history/follow-up-19/usage-central5.md).

Enrollment 9 (#333), run `891ee9b8`, closes **ENR-07**: AT06 **662/0/0** and
AT07 **48/0/0**, all five composed rows passing. The fresh gap reaches ACK 1;
sequence 2 reaches ACK 2 with **550 + 358 = 908 bytes** on a strictly ordered
**1 µs stored interval**. All four AT20 assertions were re-executed, preserving
retired **2,724 bytes / ACK 3** separately; these are usage hand-off assertions,
not §19's DNS AT20 case. Freeze withdrawal measured **187 ms**. Physical arm64
and OS reboot remain unmeasured.
[Gap closure snapshot](../../tests/handover/testdata/history/follow-up-19/enrollment9-gap.md).

## Isolation 9 — #334, ISO-F6-03 OPEN

Run `2be2797b` records AT12 **10/0/0**, AT10 **8/1/0** blocked on D-087, AT11
**2/1/0** blocked on native IPv6 / member-platform routes, and **AT13 5/1/1 FAIL**.
AT10 has 562 observation POSTs, zero qualifying accepted reports and no driven
false/true transition. Historical **4,767 / 5,021 / 5,025 ms** stand.

**ISO-F6-03 is a real open product defect, not a harness limitation.** Under a
still-valid generation-30 lease, forwarding disappears **8.1 s before the deadline**:
**96 early drops / 65 positives / 41 final drops**, with **95 early drops outside
every completed guard**. Apply 955 and live reads 956/957 succeed; verifier 958
reports `GENERATION_MISMATCH`. A periodic kernel read sees a generation-0
baseline under the valid lease; Apply 961 restores generation 30. The baseline
writer is not identified. Watchdog participation remains an inference and its
retained log is empty. **Agent-core 28 is in flight, not on main at #337**;
no fix or closure is claimed here.
[Failure snapshot](../../tests/handover/testdata/history/follow-up-19/isolation9-expiry.json).

The separate renewal diagnostic counts all **3 losses / 1,396 probes**, packet
IDs **54/294/1315**. Their ingress overlaps the install commands; the surrounding
outer guard brackets are **37.676074 / 36.090627 / 50.196105 ms** (about 36–50 ms).
Those are guard durations, not the shorter install-command durations. Every
loss remains counted; **no D-120 tolerance** applies. Historical 17 losses
still lack their original producer timing. Recovery passes **6/6**, D-106
**557/557**. Exact TEST ONLY producer timing retains **3,289 snapshots** and
clock-namespace proof. Follow-up 8 is archived **80/80** byte-identically.
[Isolation summary snapshot](../../tests/handover/testdata/history/follow-up-19/isolation9.md).

## Central D-087 seam and evidence pins — #335 / #336

Wire-nodes 33 (#335), **D-131**, puts the TEST ONLY override in lease issuance:
`httpwire.Open` accepts
`WithTestOnlyGrantEligibilityReaderFactory(func(pgx.Tx) leases.EntitlementReader)`.
The factory is sampled once per renewal transaction. Default remains
`leases.DBEntitlementReader`; a factory returning a nil reader fails closed.
There is no config / environment / HTTP surface. Desired-state compilation
keeps its production reader, so bindings and generation never move. The switch
changes future grants only; an already issued lease remains valid. The consuming
chain **devstack 20 → isolation 10 is not on main at #337**. This constructor
proof alone does not close assembled AT10.
[Seam snapshot](../../tests/handover/testdata/history/follow-up-19/wire-nodes33.md).

Netlab 20 (#336) closes members F10-01's evidence-pin issue. The members AT09
source row/link and three isolation pins now name byte-identical D-098 archives
with unchanged digests. Strict offline verification passes **200/200 pins**
and **20 regression cases**. Netlab's rule: pins outside `docs/poc/` must name
a **history/<snapshot>/** path or a **UUID run directory under docs/e2e/runs/**;
a matching digest at a living file is insufficient. No new acceptance run is
implied. [Pin-policy snapshot](../../tests/handover/testdata/history/follow-up-19/netlab20.md).

## Updates 10 — #337, D-132

Run `695f977c` passes the installer → updater → central extension **23/23**:
D-110 / D-111 / D-125 / D-127, real receipts, exact replays, 409 conflicts,
TEST ONLY rollback correction, `HASH_MISMATCH` and `DOWNGRADE` refusal.
The correction is an HTTP stimulus; it does not prove physical rollback.
Serving withdrawal/nonzero accounting is consumed from devstack's run d6aaa798.

**Regular AT23 is blocked.** The candidate exits
`agent: active release identity is inconsistent`: the shared devstack fixture
builds `bin/fvpn-agent`, while `node/agent/release.go` verifies the release-root
binary like the real installer layout. That check landed in agent-core 24
(#305), after updates 9's pass (#283). Updates **F10-01 → devstack 20**.
AT24–AT27 remain blocked behind AT23; **no earlier pass is carried forward**.
Updates **F10-02 → installer 7**: generated `result_url` is empty; the extension
supplied the owned endpoint/trust explicitly, so generated-config reporting is
not proved. [Updates snapshot](../../tests/handover/testdata/history/follow-up-19/updates10.md).

**D-132:** product tests may pin AT evidence only at immutable
`docs/e2e/at/<lane>/history/<follow-up>/` paths, never living root files.
Two tests in `internal/fleetupdate/d062_integration_test.go` read living updates
evidence and broke the integration chain after the D-098 move. Under the dated
one-file exception in #337, both now read the follow-up 9 archive; fleet-update
13 owns the follow-up. This is evidence hygiene, not new product acceptance.
[Exact merged exception](../../tests/handover/testdata/history/follow-up-19/D132-PR337.txt).

## Previous cut and still-open work

Handover 18 (#327) covered through #324: exact r17 archive, D-124–D-128 register,
`TestHandover_Followup18Claims` with **12 subtests / 26 snapshots**. Its
[original register](../../tests/handover/testdata/history/follow-up-19/register18.md)
is frozen before this update. D-124–D-128 stay in the current decision register.

At #337, the active findings are ISO-F6-03 (agent-core 28); D-087 assembled AT10
(devstack 20 → isolation 10); updates F10-01/F10-02 (devstack 20 / installer 7);
members F10-02/F10-03 (devstack 20 / dns-publish 9); and unattributed F8-02,
including native MTU 1280 under D-123. AT11 still needs real platform routes;
AT13 still needs the full continuous fault matrix including physical arm64.
AT29 / AT31 remain resource-blocked. Physical arm64, OS reboot, power-loss and
real Windows/macOS/iOS/Android clients remain unmeasured in the required matrix.
ENR-07, F9-01, the devstack 18 serving-node failure and the candidate seed rerun
are closed in this batch; their historical failed records remain unchanged.
