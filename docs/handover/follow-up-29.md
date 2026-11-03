# Follow-up 29 — adoption gap, unqualified boundary sample and measurement end

Verified against `main` **d640ce96d9145e57a60e04e5875e049070f30b96**,
through **#396** on **2026-11-01 EST**. This cut is **#391–#396**:
usage 8, isolation 15, agent-core 35, devstack 27, handover 28 and netlab 25.
Nothing was merged after the cutoff at task start. Agent-core 36, isolation 16,
members 16, devstack 28 and netlab 26 were running; their results are outside
this cut. Handover ran no new live acceptance.

**D-098 / D-109 / D-132:** `TestHandover_Followup29Claims` pins
[six final roadmap rows](../../tests/handover/testdata/followup29-main.json),
[eight historical handover snapshots](../../tests/handover/testdata/followup29-snapshots.json)
and [dated evidence / archive manifests](../../tests/handover/testdata/followup29-evidence.json),
including **731 archive and run file checks**. Go test names are checked on
main. Usage's b8a6e9e1 and isolation's 0b9e4510 root records are living:
cite them by roadmap row / PR only, never as claims-test inputs. Isolation's
`history/follow-up-14/` and `history/follow-up-15-stream/` are immutable.
No roadmap row was edited.

## Previous cut — handover 28 (#395)

[Follow-up 28](follow-up-28.md) covers **#386–#390 at aae6763f**.
`TestHandover_Followup28Claims` has **19 subtests / 5 exact roadmap rows /
16 immutable snapshots / 1,644 archive and run file checks**. Its findings
register has **47 rows**; both counts are verified against main, not inferred
from the delivery message. Its seven requests are owner decisions, retained
below. Earlier claims stay true using the frozen register and entrypoints.

## Usage 8 (#391) — coverage, no live run

Netlab 24's request 1 is closed as coverage. AT14's synthetic input carries
nonzero `ctdenyd` (80 packets / 80,000 bytes). `enforce.Samples` is identical
with and without it and returns no samples from it alone. Complete rendered
statements establish the download order: resolver TCP / UDP source-port-53
`nd_` replies, then `denied_v4` / `host_v4` / `resolver_v4` guards, then every
device's `d_` admission. `TestAT14_ForwardAdmissionMetering_Mechanism` pins the
proof (MTR01 / MTR05, §10.1 / §10.3 / §19 AT14). There was no live run;
run b8a6e9e1 is cited only by the final #391 roadmap row (D-132).

## Isolation 15 (#392) — evaluator pass, qualification failure

The D-098 [follow-up 14 archive](../e2e/at/isolation/history/follow-up-14/archive.json)
preserves run **56d20530**, 117 files and subtrees.
`TestD098_Followup14ArchivePins` names its owner check. The immutable
[follow-up 15 stream archive](../e2e/at/isolation/history/follow-up-15-stream/archive.json)
retains eight lossless compressed files with both-form hashes and lengths.

Run **0b9e4510 on #388's tree**: **AT10 PASS 9 / 0 / 0 (ISO-F6-01 6 / 6)**;
**AT12 PASS 10 / 0 / 0**. AT11 and AT13 stay blocked on carried operator rows.
These current-run facts are pinned only through the final #392 roadmap row.

Under D-140, the unchanged `EvaluateDeadline` passes: last positive
**D−98.630 ms**, first drop **D−97.598 ms**. But the no-interleaving
qualification fails once: **packet 171's egress followed packet 172's ingress
by 3.821 µs**. The whole-node row stays **unqualified**, and this attempt
stays **failed qualification**. `TestAT13_D140InterleavingQualification` and
`TestAT13_D104StrictBoundaryAndMissingEvidence` name the separate checks.
Isolation F14-01 (run 56d20530) stays failed. The next sample belongs to
isolation 16; it is not in this cut. No acceptance predicate or failure is waived.

Parent receipts show **three agent exits 0**. The terminal source node's agent
ended by **signal 15, with no exit code**. Speed does not establish cooperative
termination; the cause is unproven and routed to agent-core 36. No stale-lease
hold formed, so isolation did **not exercise the populated hold**. Run
0b9e4510's root files remain living until isolation 16 archives them (D-132).
This supports ISO01 / ISO02, §8.1 / §8.2 / §16.3 / §19 AT10–AT13;
it does not close the full operator matrix.

## Agent-core 35 (#393) — diagnostic exclusion and a reachable adoption gap

The [dated hand-off](../node/followup35-diagnostic-and-restart-2026-11-01.md)
proves `ctdenyd` inert for usage across meter selection, the reset matcher,
persisted readings, the usage snapshot and observation consumers. Positive
member controls remain. No production change was needed. Proof names:
`TestAT14_F35_CtdenydUsageSelection`,
`TestAT16_F35_CtdenydDoesNotChangePersistedUsageOrEpoch`,
`TestAT16_F35_CtdenydResetAndReportConsumers`,
`TestAT14_F35_CtdenydCounterSamples` and
`TestAT16_F35_CtdenydApplyPersistenceDoesNotBecomeUsage`
(MTR01 / MTR03 / MTR05 / SEC02, §10.1 / §10.5 / §10.7 / §19 AT14 / AT16).

**New open, production-reachable finding:** after restart with a larger
configured `extra_denied_v4` or `host_v4`, same-boot adoption can serve on the
old, smaller `denied_v4` / `host_v4` until the first successful Apply. A busy
node lock or control timeout can delay that Apply. The verifier never reads
those two sets' elements. Rebuilding `appliedModel` during adoption does not
install that model.

`TestAT13_F35_RestartFirstApplyRendersConfiguredDenyGrowth/same_boot_adoption`
pins the known gap and the first Apply's render proof. The first successful
full Apply renders the new configured sets; no maximum wait until that Apply
is claimed. **Routed in two steps:** agent-core 36 refuses adoption on changed
sets, then netlab 26 adds the verifier's content check. The gap is **open at
#396**. **New unconfirmed finding:** the same blind spot may let a set changed
live under a serving agent go undetected; netlab 26 is proving that reading.
It is not a confirmed live-drift defect or measured traffic result.

The dated hand-off gives **no safe procedure**. The operator guide's node
configuration section records the known behaviour and routed fix without
promising a workaround. Scope: ISO01 / ISO02, §8.1 / §8.2 / §16.3 / §19 AT13.

## Devstack 27 (#394) — TEST ONLY hold and serving-node campaign

The immutable [dated analysis](../e2e/runs/2026-11-01/followup27/analysis.md)
and [hold proof](../e2e/runs/2026-11-01/followup27/hold-fixture.md) describe the
**TEST ONLY** seam. `labstack.Stack` exports `ArmMatchingLeaseHold`,
`MatchingLeaseHold`, `ReleaseMatchingLeaseHold`, `GenerationHoldSnapshot` and
`LocalLeaseReadback`; these are Go identifiers on main. For a bounded time,
the proxy answers **503 only to successful matching lease-renewal responses**.
It never edits, replays or synthesizes a signed document. Desired-state and
observation traffic continue; release forwards the next genuine central lease.
`TestAT13_MatchingLeaseTransportHold` names the transport proof.

Final fixture: **13 / 0 / 0**, **two members**, incident **50.000298 s**.
The **Step-3 report carried the alarm**. Central escalated **once**, at incident
age **9.887 s**, triggered by the alarm and **not the 30 s fallback**.
Recovery ended the incident **`generation_matched`**. The two earlier fixture
runner failures stay failed. This proves fixture support for §16.3 / AT29;
it does not pass the consumers' rows: **unrelated-failure precedence,
active-stop retention, recovery inside a full row, a distinct later incident,
and sole-member `members_revoked` closure** remain with isolation / members.

Serving-node campaign: **8 / 8 fresh first selections**, **58,619 sends**,
**zero missing node ingress**. All **183 ingress-without-origin losses** are
counted. IPv6 **`::1`-source type-4 data and type-2 handshake responses** are
retained as anomalies. Neither the five-SYN failure nor the five-second gap
was reproduced. No cause is assigned to members' **F15-F11-01, F15-AT02 or
F14-01**; their files are byte-identical and their rows are unchanged. These
observations do not establish a common cause. Devstack's **request 2** adds
the `::1` anomalies to the owner's **F11-01 evidence**, repair direction and
upstream-report decision. No complete AT01 / AT02 acceptance is claimed.

## Netlab 25 (#396) — bounded D-120 measurement end

The per-follow-up [evidence tree](../poc/G2-followup25-evidence/overview.md)
is pinnable under D-098 like follow-up 24's. The harness ends the measured
stream **only after the final guard bracket closes**. It records **at most
one successor cycle after a lost tail**; a failed successor stays failed.
A **15 s parent watchdog fails an incomplete measurement**. This fixes the
measurement end, not the D-120 acceptance definition.

`classify_d120` and its tests are **byte-identical**, as recorded by the
[analyzer identity receipt](../poc/G2-followup25-evidence/analyzer-identity.json).
**A2, the earlier branch attempt and `gate2-replay` stay failed**. The
[replay verdict](../poc/G2-followup25-evidence/a2-after-verdict.txt) again reports
**generation 1087 / sequence 57**, **`(iii): next datagram missing`**.

One live run on a quiet host: **100 reloads, 101 verified Applies, zero UDP
losses, zero TCP timeouts and zero extra cycles**. The lost-tail extension is
**proven offline only**. The run does not retroactively supply any successor
to a failed historical stream (ISO01 / ISO02, §8.1 / §8.2 / §19 AT10 / AT13).

Large evidence is lossless gzip with **SHA-256 and length for both forms**.
The Apply JSONL is **13,646,265 → 720,796 bytes**; the
[retention manifest](../poc/G2-followup25-evidence/retention-pins.json) and
publication replay retain the exact original bytes and verdict.

## Decisions and findings

No decision was added after D-141 on main. **D-135, D-136, D-137, D-138,
D-140 and D-141 stay provisional.** The [register](README.md#findings-register-at-the-follow-up-29-cutoff)
has 51 rows: the previous 47, plus the same-boot adoption gap, unconfirmed live
drift, isolation 15's unqualified sample and its unexplained signal-15 receipt.
D-120's measurement end is fixed; its failed runs are not repaired evidence.
Every other open finding of follow-up 28 is unchanged. Every historical
failure stays failed. D-141 keeps option (c) in production; no owner decision,
production value or acceptance predicate changed.

### Requests to orchestrator

2026-11-01 — **Owner decision 1: D-135 + D-136 confirmation**.

2026-11-01 — **Owner decision 2: D-137 hardening option**; keep the failed
receive-only MTU-1280 limitation without upgrade or waiver.

2026-11-01 — **Owner decision 3: D-138 overturn option**.

2026-11-01 — **Owner decision 4: D-140 overturn option**; unchanged evaluator,
failed qualification and historical F14-01 retained.

2026-11-01 — **Owner decision 5: D-141 overturn option**, retaining its witness,
qualifying-share, promotion-cost and design-finding prerequisites.

2026-11-01 — **Owner decision 6: F11-01 repair direction and upstream report**,
now including devstack 27's `::1` data and handshake-response evidence (request 2).
The members 15 / F14-01 causes remain unproven.

2026-11-01 — **Owner question 7: RELATED ICMP from spec-denied outer sources**
under §8.1 / §8.2 / §19 AT13. Keep the fail-closed source guard pending an owner
decision; no exception, defect disposition or fix is promised.

Already routed, not new requests (2026-11-01): adoption gap → agent-core 36
then netlab 26; unconfirmed live drift → netlab 26; signal-15 cause → agent-core
36; next D-140 sample → isolation 16; populated-hold consumer rows → isolation /
members; remaining `ctdenyd` consumer coverage → devstack 28. These items stop
with their owners. No outside-owned pin needed changing. **Seven owner
requests**, with no additional implementation request from this lane.
