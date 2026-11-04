# Follow-up 45 outcome — 2026-11-03

| Leg and window | Pooled (reason if not) | Scenarios passed | Marked intervals | Retained / excluded whole-Step spans | Outer guard p99 / maximum (ms) | Whole serving Step p99 / maximum (ms) | Failed operations |
| --- | --- | --- | --- | --- | --- | --- | --- |
| A1 / 1 | No: parent-shell preflight match | 0/13 | 1 | 0 / 0 | — / — | — / — | 0 producer; 1 harness refusal/failure |
| B1 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A2 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| B2 / 1 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A3 / 2 | No: harness preparation; no Step spans | 0/13 | 1 | 0 / 0 | — / — | — / — | 0 producer; 1 harness refusal/failure |
| B3 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| A4 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |
| B4 / 2 | No: not started after window stop | 0/13 | 0 | 0 / 0 | — / — | — / — | — |

No comparison is permitted: A and B each have zero pooled legs. All five
retained distributions are empty, with zero excluded spans and no producer
failed operations. These are absent measurements, not successful operations.
No B-only failure or attributable shift is established.

The corrected clock prerequisite passed the dry preflight and all three A3
clock samples: different namespaces, zero monotonic/boottime offsets on both
sides, and ordered host/probe brackets. All source/tree/overlay hashes match
the prior declaration. B's non-test production Go diff against #429 returned
exit 0 with empty output, using the exact command in the declaration pins.

Window 1 reserved immediately, waited 60 seconds, then refused A1 because
its parent declaration-writing shell matched the suite-process pattern.
No scenario ran. It released at 2026-11-03T22:47:53.544637+00:00.
Window 2 reserved 600.071204 seconds after release,
waited 60 seconds and admitted A3. Its non-CPU monitoring and clocks passed,
but make exited 2 after 13.992014 seconds before scenarios. The frozen harness
combines Docker stdout/stderr and expects exactly `true`; inherited admission
stderr adds `free`. The synthetic actual-wrapper probe reproduces
`free\ntrue\n`. This is a harness failure, not a product or clock failure.
Window 2 released at 2026-11-03T22:59:12.410622+00:00. Neither reservation had a
refusal or wait beyond its approximately 0.07-second successful call. No third
reservation, replacement leg, requalification or production change occurred.

There are six qualification samples and two marked foreign-CPU intervals.
Each mark's endpoints and five largest processes, every sample, all empty
per-leg/pooled p50/p90/p99/max/count metrics, source/binary/fixture identities,
clock readings and raw evidence hashes are in the
[results](followup45-timing-results-2026-11-03.json). The pre-Apply accounting
boundary remains separate from the command guard. No packet predicate was
obtained from a timing leg. The inherited sample visibility limit for short-
lived processes remains explicit. Foreign Codex processes were not excluded.

The required ordinary **main lab passed all 13 scenarios**, including D087/D104,
after both reservations released. It ran without overlays on
`3f7dbc254ba1ce001628af8151a4d81c6954412d`, including netlab 34's **D-146 TCP MSS clamp
and D-147 RELATED fragmentation-needed rules**. Admission diagnostics were
captured in the ordinary gate wrapper's separate receipt log, as in 44's
functional run; the committed timing runner remains unchanged. This is
functional evidence only and supplies no pooled timing leg.

Expiry predicate: True; 97 positives and
15 drops, last positive -100.028400 ms
and first drop -63.789971 ms relative to deadline;
0 positive egresses at/after expiry.
Renewal: 2 actual renewals,
1517/1518 delivered,
1 loss (1 inside guards,
0 outside); zero-loss predicate
False. The suite reports renewal losses without asserting zero loss; its
13/13 pass does not establish loss-free renewal. No predicate or tolerance was relaxed.
The [functional receipt](followup45-production-lab-2026-11-03.json) retains
all diagnostic failed operations, packet predicates and identities.

| Gate | Exit | Seconds |
| --- | --- | --- |
| node-lint-build | 0 | 8.592 |
| node-test | 0 | 137.143 |
| node-integration | 0 | 139.254 |
| reviews-r16-r19 | 0 | 20.015 |
| reviews-r20-r26 | 0 | 46.488 |
| handover | 0 | 3.848 |
| go-lint | 0 | 4.754 |
| vet | 0 | 0.395 |
| evidence | 0 | 3.239 |
| fixture-hygiene | 0 | 8.668 |
| final-all-tests | 0 | 152.422 |
| ordinary main lab (13/13) | 0 | 230.853 |
| synthetic tests (41 root; 17 other directory) | 0 | — |

The [validation record](followup45-validation-2026-11-03.json) contains real
gate tails, hashes, environment and zero final owned Docker inventory. Builds
cover linux/amd64 and linux/arm64; physical arm64 remains unmeasured. No
production, node/enforce, dependencies, earlier tools/tests or records changed.
Three of 44's tests and one of 43's still fail from another working directory;
those legacy tests are unchanged. All 45 tests use repository disk scratch.

### Requests to orchestrator — 2026-11-03

- **2026-11-03 — D-133/D-150:** arrange a later runner follow-up and freshly declared comparison. Preserve admission receipts outside Docker streams consumed by the frozen harness's `CombinedOutput`; cover that exact consumer in synthetic tests and dry preflight. Both reservations here are exhausted. This is harness work, with no production fix inferred.
- **2026-11-03 — invocation:** keep future timing driver launches standalone. A declaration-writing parent shell containing a suite-pattern literal disqualified window 1; no ancestor exclusion was added.
