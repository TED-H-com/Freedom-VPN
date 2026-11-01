# Follow-up 23 — month-boundary incident and the re-chained batch

Fixed `main` cutoff **bc011b6f0fb8324d78ad5540c698a15f568fa079**, through
**#362**, checked for the 2026-11-01 EDT handover. The last delivery was
#362, after #361; this cut includes every PR #357–#362. Security-ops 14 was
not merged inside this range. Netlab 22, members 13, devstack 23, agent-core 31
and usage AT 7 were absent from main at start; no later results enter this cut.
Handover ran no new live acceptance.

**D-098 / D-109 / D-132:** [six exact roadmap rows](../../tests/handover/testdata/followup23-main.json)
pin current lane results. The [18 immutable snapshots](../../tests/handover/testdata/followup23-snapshots.json)
retain dated hand-offs, exact PR hand-off excerpts, archive inventories and
frozen owned documents. Raw evidence reads use owner history or UUID run
subtrees. Members 12's F12 roots and isolation 11 / updates 12 root evidence
remain living files: no claims test reads or copies them. Roadmap/PR pins
carry their current results; source paths in copied prose are provenance only.

## Incident and lesson — agent-core 30, #357

From **2026-11-01 00:00 UTC**, unchanged main's
`TestUsageCheckpointRoundTrip` in `node/internal/fakecontrol/usage_test.go`
paired literal `PeriodID: "2026-10"` with a wall-clock interval and failed:
`period_id 2026-10 does not overlap the interval`. The first post-boundary
gate run failed; the recorded handover chain failure was **23:09 EDT**, and
every merge chain stopped until the fix. The [#357 hand-off excerpt](../../tests/handover/testdata/history/follow-up-23/agent30-handoff.md)
records that failure and four other chains stopped before push; the corrected
batch re-chained at 23:23 EDT.

Fake control, checkpoint verification and the usage store now share one fixed
**2026-11-01T00:00:00Z** clock. The event is
**[2026-10-31T23:59:00Z, 2026-10-31T23:59:10Z)** and derives its period from its
own start. `internal/usage/ingest.go`'s overlap predicate is unchanged, verified
against the pre-fix tree. The owned calendar sweep covered **87 test files**,
with **one fix** and **23 candidates kept with reasons**. Six other-lane findings
were routed: usage AT live cases a / b and hand-off, secops PITR and rate classes,
and members usage trace. Their repair is not inferred from #357.

**Lesson:** a test never pairs a calendar literal with the wall clock. A
half-open interval belongs to the month of its start; derive that fixture
period from the interval, including when the checkpoint is in the next month.
This fixture lesson does not change the product's interval-overlap validator.
MTR03 / MTR04, AT15 / AT17 / AT30 (§10.4, §10.6, §10.8–§10.9, §17.3) retain
the same acceptance scope.

## Handover 22 — #358, frozen

The [previous cut](follow-up-22.md) covers **#352–#356**, cutoff **#356** at
**45209609**. `TestHandover_Followup22Claims` retains **19 subtests / 5 rows /
22 snapshots**, the **114-file devstack-20** and **82-file netlab-20** archive
inventories, D-134 registration and provisional D-135. Its live register and
entrypoint reads now use exact frozen excerpts. A whole-function hash checks
that only those input redirects changed; earlier assertions remain intact.

## Updates 12 — #359

D-098 first: run **daddad24**, **43 files**, archived byte-identically under
[history/follow-up-11](../e2e/at/updates/history/follow-up-11/ARCHIVE.md).
Fresh run **2babf9b6** passes **regular AT23–AT27 5 / 5** through the shared
installer wrapper with devstack 21's explicit **TEST ONLY loopback opt-in**,
installed / exit 0, no adapter; installer **27 / 27**. Physical rollback is
included. Generated endpoint, origin, enrollment server CA and client paths
read back unchanged. The 13 result POSTs are 11 explicit stimuli plus one
automatic success and one automatic stopped-producer failure. The uncached
fallback exits 0 with unchanged config and signed release URL.

**Updates F11-01 CLOSED**: #352 supplies the fix, #359 supplies the regular-run
result. **F11-02 CLOSED** by #351. **F18-01** stays the intended **D-125 negative
1 / 1 / 2**. The first attempt's harness path bug (make exit 2 despite 5 / 5)
is retained under [history/follow-up-12-first-attempt](../e2e/at/updates/history/follow-up-12-first-attempt/);
no pass row is carried across runs. **0 requests**.
Proof names: `TestUpdatesFollowup`, `TestD132_Followup11ArchivePins`,
`TestD098_Followup12FirstAttemptArchivePins`.

## Isolation 11 — #360

D-098 first: run **bfcc3880**, **100 files**, archived byte-identically under
[history/follow-up-10](../e2e/at/isolation/history/follow-up-10/archive.json).
Agent-core 29's host context is consumed by run / boot / sequence. The **49
historical failed installs** replay with **host / verification unavailable**,
`cause_status=unproven`; all 279 historical renewal losses stay counted.

Fresh run **edd986c4**: **AT10 PASS 9 / 0 / 0** (real D-087, 379 ms),
**AT12 PASS 10 / 0 / 0**, **AT11 2 / 1 / 0** and **AT13 6 / 1 / 0**
operator-blocked. Fresh whole-node **D-104 passes 161 / 41**, last positive
D−99.872 ms, first drop D−62.925 ms, one pre-D ingress loss counted.
Renewal records 1,396 probes / 18 losses, all inside completed guards. Under
D-134, every pre-D and renewal loss is counted; no predicate is relaxed.

There are **7,885 producer records / 952 Applies / 0 install overruns**.
The **7 failed Applies** are intended **ISO-F6-01 foreign-tc pre-install
rejections**, with timing, host samples and sanitized status. ISO-F6-01 is
6 / 6; D-106 is 639 / 639. The **945 completed guards** have these distributions:

| Population | p50 ms | p90 ms | p99 ms | max ms |
|---|---:|---:|---:|---:|
| Outer guard | 31.55 | 41.57 | 48.39 | 56.77 |
| Install | 16.52 | 22.59 | 28.10 | 30.35 |

**2 carried operator requests: AT11, AT13**. ISO-F6-03's historical writer
remains unattributed. Proof names: `TestD098_Followup10ArchivePins`,
`TestISO_F6_03_ProducerTimingCorrelation`, `TestD087_OriginalWireAndAcceptedTransition`.

## Devstack 22 — #362

Follow-up 21's **63 evidence files** are archived via
[followup21-archive.sha256](../e2e/followup21-archive.sha256); its loose analysis
copies live in [followup21-archive](../e2e/runs/2026-10-31/followup21-archive/).
The dated [follow-up 22 hand-off](../e2e/runs/2026-10-31/followup22/README.md)
and [155 capture pins](../e2e/followup22-captures.sha256) retain both campaigns.
Final run **9b3a89e7** is **13 / 0 / 2**, exit **1**. The original four-row
mutation sequence passes, while the supplemental native IPv6 asymmetric row
reproduces **F8-02: 16,016 / 64,064 bytes in 12,011 ms**. All **15 reverse
1,360-byte outer packets** are absent at member underlay; smaller packets and
all 52 upload fragment pairs arrive. **No PTB** appears. Historical F8-02 rows
stay failed; receive-link rejection is an inference, not a kernel drop trace.

The separate sender-side **ConstrainSelectedNativePeer1280** control passes
64,064 bytes in 1 ms three times by **sender fragmentation**, using only the
selected /128 route. Reactive PTB discovery was not measured. The receive-only
fixture is not real-device proof or a production MTU repair.

**Members F11-01's five-SYN pattern is reproduced**: a matching IPv6
handshake response reaches member underlay with a **loopback source**, with
**no WireGuard type-4 transport data** during the request. The first
unconfirmed step narrows to member inner egress → transport-data emission.
Cause remains open; the capture does not identify historical packet identities
or prove session acceptance. `StartNativeCapture` / `Sample` / `Stop` are
exported. **2 requests:** members F11-01 adoption; the pinned NetBird client's
loopback-source handshake/session trace → devstack 23. Proof names:
`TestE2E_Followup22Native`, `TestAT01_CaptureMissingStatsNeverMeansZeroDrops`.

## Members 12 — #361

D-098 first: run **7edb263f**, **43 files**, archived under
[history/follow-up-11](../e2e/at/members/history/follow-up-11/SHA256SUMS).
Fresh run **b98b78ee**: **0 pass / 4 blocked / 4 fail**, 468 / 0 / 14 subchecks;
AT01 / AT09 were not re-run. **F11-02 CLOSED by fresh populated-grant rows**
at generations 10 / 21; its historical failure stays failed. **F11-03 BLOCKED —
no failed Apply observed**, with all **89 Applies succeeded**. The sibling
capture has 3,976 attempts / 21 losses: 13 inside completed guards, eight
outside, with all eight inside the separate F12-01 bracket.

- **F12-01:** stale-lease `policy_missing` baseline → populated renewal →
  successful Apply, **236.970 ms**, **nine sibling losses**. D-136 classifies
  this transition's designed fail-closed semantics; it does not attribute the
  old failed-Apply mechanism or convert a historical failed row to a pass.
- **F12-02:** the **1 MiB timing-export limit** caused **11 readiness failures**
  in AT03 / AT04 / AT08; AT04's separate monitor ended before the delayed
  refund input. Streamed reader **7b949b68** is **not yet live-qualified**;
  offline/read-only export proof does not replace the original run.
- **F12-03:** negative-cache SOA TTL 2, then same-primary positive refetch after
  **1.229 s**; exact cause unproven, including expiry precision uncertainty.
- **F12-04:** native UDP **SERVFAIL after 5,001 ms**, with positive receipts
  from **both resolvers**; responder/delivery attribution remains open.

**12 requests routed.** Proof names: `TestAT02_F12ReadinessRejectsEmptyLeaseAndLaterWithdrawal`,
`TestAT02_F12RecoveryRequiresNextApplyAndCountsEveryLoss`,
`TestAT02_F12TimingStreamExceedsDiagnosticLimit`,
`TestAT20_F12NegativePrecisionAndOutstandingCompletion`.

## Provisional decisions and findings

The [register](README.md#follow-up-23-decisions-register-2026-11-01-edt) adds
**D-136 (23:15, 2026-10-31 — provisional, to be confirmed together with D-135)**
and **D-137 (23:15 — provisional)** under the orchestrator's follow-up 23
instruction. They are interpretations pending owner confirmation, not new
implementation deliveries. D-135 stays provisional; its original row is retained.

**D-136:** `node/agent/decide.go`'s `leaseGenerationStale` blocks with
`policy_missing` when the serving lease names another generation. The node
serves the node-wide baseline until renewal and the next successful Apply:
designed fail-closed behaviour, not a defect. The existing proof is
`TestD067_StaleGenerationKeepsPolicyMissing`. D-135's bracket rule covers
**every node-wide baseline transition**, failed Apply or stale lease:
**baseline → next successful Apply**, **every loss counted**, **recovery
required**, **never waived**. Stronger availability requires agent-core design
under **D-133** with an **adversarial review**.

**D-137:** §8.3 (spec line **365**) and §19 **AT09** require usability within
the supported scope or a clearly reported limitation. Supported MTU-1280
scope is a **sender-learnable 1280 path**: known PMTU, or a **reactive PTB from
a router hop**. A receive-only 1280 link dropping without PTB is a **PMTU black
hole**, reported as a clearly stated limitation, **never a pass**. Historical
F8-02 rows stay failed. **No global MTU / MSS clamp** is introduced. A lower
tunnel MTU or an MSS clamp is an **owner hardening option**, not selected here.
The known-PMTU control does not establish the reactive-PTB branch of this scope.

Closed: updates F11-01 (#352 fix, #359 regular-run result), members F11-02
(#361 fresh rows). New: F12-01 (D-136), F12-02, F12-03, F12-04. Updated:
F8-02 (native IPv6 #362, D-137 classification, rows stay failed), members
F11-01 (pattern reproduced #362, cause open), members F11-03 (blocked).
Unchanged: install-overrun cause unproven; F9-INFLIGHT; F18-01 intended
negative; ISO-F6-03 writer unattributed; Dependabot #22 open for owner
disposition; production `blocked_response_ttl` owner value.
**Every historical failure stays failed and unattributed.** No product
validation, timeout, forwarding implementation or acceptance predicate changes.

Requirement scope: MTR03 / MTR04 (§10, AT15/AT17/AT30); UPD01 / UPD02 /
UPD03 / UPD05 (§13, AT23–AT27); ISO01 / ISO02 / ISO04 (§8.2, AT10–AT13);
MEM03 / ENT02 / SEC01 (§8.2, §10.3, §11.5, AT02/AT04/AT08); DNS02 /
SEC02 (§12, §16.2, AT20); EXIT02 / APP01 / NFR01 (§8.3, AT01/AT09).
Physical arm64, platform routes, full fault matrices, QUIC, reactive PTB,
power loss and production rollout remain unmeasured where not independently
proved. This cut is evidence reconciliation, not expanded §19 acceptance.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: confirm D-135 + D-136 together**, including
all baseline-transition brackets, counted losses and required recovery. Stronger
availability belongs to agent-core under D-133 plus adversarial review.

2026-11-01 — **Owner decision 2: confirm provisional D-137 scope and choose
whether to pursue its lower-tunnel-MTU / MSS-clamp hardening option.** No
production value or clamp is selected here.

Existing outside-lane work stays routed: devstack 23's pinned-client trace;
members' capture adoption and streamed-reader live qualification; DNS
attribution; isolation's AT11 / AT13 operator resources; the six calendar
findings. These are carried lane requests, not additional handover edits.
No outside-owned pin needed changing; no outside-owned file was edited.
