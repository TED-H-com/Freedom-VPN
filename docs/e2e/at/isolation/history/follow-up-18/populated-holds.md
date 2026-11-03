# Follow-up 18 populated stale-lease rows

Run `9f1bc0ff-d01f-4a3f-ad41-6c6a086e6e04`. Three distinct planned incidents; no retry. All original seven coverage categories pass across the first two incidents. Central 30-second fallback passes in the third. Pure `stale_generation` remains **not exercised**.

[Original attempts](populated-holds.json.gz), [accepted reports and own signed leases](generation-holds-central.json.gz), [checked joins](populated-holds-review.json), [general review](generation-holds-review.json).

## Incident 1

Arm `01d992dd-4a66-4ecd-acbe-8e81924f970f`; incident `01a1a1be-ea82-776b-9e85-cbee85a25501`. Node `01a1a1bb-e75c-7971-823d-ded91df161ad`, incarnation `01a1a1bb-eaa8-7bb7-9f6d-e993c97b0e4d`. Old lease `a5979e28-d750-4121-8fd1-f2534a7488e8`, generation 10, **2 members**.

First seen **2026-11-03T14:10:06.081443+00:00**, last seen **2026-11-03T14:10:12.969288+00:00**; ended **2026-11-03T14:10:13.946649+00:00**, `generation_matched`. These are central receipt times, not packet instants.

| Row | Result |
|---|---|
| populated_hold | pass |
| independent_denial | pass |
| alarm_single_escalation | pass |
| unrelated_failure_precedence | pass |
| active_stop_retention | pass |
| matching_recovery | pass |
| distinct_later_incident | not exercised |
| pure_stale_generation | not exercised |

Accepted hold report IDs: `01a1a1be-ea7c-722e-88bb-8c57c70c8fb8`, `01a1a1be-f1e7-700e-bebc-2348abcf44ac`, `01a1a1be-f9c0-764e-9e87-d820573026d4`, `01a1a1bf-018d-707c-ad1c-607607166ab4`, `01a1a1bf-0562-7d84-aaf7-e909f1ed0eab`.

Audit IDs: `01a1a1be-f1ef-716e-a034-f5d9dc479bad`.

Audit trigger **reported_alarm**, incident age **1.898665000 s**; reported Steps 3, elapsed 1903 ms, alarm `true`, alarm report `01a1a1be-f1e7-700e-bebc-2348abcf44ac`.

Active-stop reset: report `01a1a1bf-018d-707c-ad1c-607607166ab4` **7 Steps / 5910 ms** → `01a1a1bf-0562-7d84-aaf7-e909f1ed0eab` **1 Step / 0 ms**. Same open incident, first_seen, single audit, and still-armed proxy.

Unrelated failure report: `01a1a1bf-018d-707c-ad1c-607607166ab4`; classification remains `check_failed`.

Accepted matching report IDs: `01a1a1bf-0934-7ee4-b9c2-d82a2c9b6666`.

Genuine forwarded/local lease `496eb557-d751-429c-add9-46aefd34a7cc` matches by body digest. Separate counted recovery **8015→10646 bytes**. Matching incident closure is not alone a serving assertion.

## Incident 2

Arm `f0c95f5c-3ca0-404d-9b61-85c820b9f811`; incident `01a1a1bf-92a9-7de5-9d8e-c9606d945c9c`. Node `01a1a1bb-e75c-7971-823d-ded91df161ad`, incarnation `01a1a1bb-eaa8-7bb7-9f6d-e993c97b0e4d`. Old lease `964d3d3c-753a-4f14-b6b7-c9c61d8d1761`, generation 11, **2 members**.

First seen **2026-11-03T14:10:49.129063+00:00**, last seen **2026-11-03T14:10:54.030778+00:00**; ended **2026-11-03T14:10:55.125297+00:00**, `generation_matched`. These are central receipt times, not packet instants.

| Row | Result |
|---|---|
| populated_hold | pass |
| independent_denial | pass |
| alarm_single_escalation | pass |
| unrelated_failure_precedence | not exercised |
| active_stop_retention | not exercised |
| matching_recovery | pass |
| distinct_later_incident | pass |
| pure_stale_generation | not exercised |

Accepted hold report IDs: `01a1a1bf-92a3-7e51-b710-af3ac15faab5`, `01a1a1bf-9a11-74a2-826d-fb6e8fac383b`, `01a1a1bf-a1dc-78e4-9a57-61ae91c10f5b`, `01a1a1bf-a5c8-7871-a4fa-1942a534be1a`.

Audit IDs: `01a1a1bf-9a1a-792e-ba51-437239eff17c`.

Audit trigger **reported_alarm**, incident age **1.901769000 s**; reported Steps 3, elapsed 1909 ms, alarm `true`, alarm report `01a1a1bf-9a11-74a2-826d-fb6e8fac383b`.

Accepted matching report IDs: `01a1a1bf-aa0e-76ce-9f7e-570f7b4f1718`.

Genuine forwarded/local lease `def58a81-d20f-4271-8095-ce659d729319` matches by body digest. Separate counted recovery **13278→15909 bytes**. Matching incident closure is not alone a serving assertion.

## Incident 3

Arm `8e449c45-8fc1-4164-a393-4c5eecc6e60f`; incident `01a1a1c0-2f03-714c-8b8c-2ef3c10ab6a3`. Node `01a1a1bb-e75c-7971-823d-ded91df161ad`, incarnation `01a1a1bb-eaa8-7bb7-9f6d-e993c97b0e4d`. Old lease `c9864c87-92b0-4106-b7f7-428a6b37f13e`, generation 12, **2 members**.

First seen **2026-11-03T14:11:29.154474+00:00**, last seen **2026-11-03T14:12:03.343346+00:00**; ended **2026-11-03T14:12:05.355593+00:00**, `generation_matched`. These are central receipt times, not packet instants.

| Row | Result |
|---|---|
| populated_hold | pass |
| independent_denial | pass |
| alarm_single_escalation | not exercised |
| unrelated_failure_precedence | not exercised |
| active_stop_retention | not exercised |
| matching_recovery | pass |
| distinct_later_incident | pass |
| central_30s_fallback | pass |
| pure_stale_generation | not exercised |

Accepted hold report IDs: `01a1a1c0-2efd-7256-a47b-a3948eacd4fb`, `01a1a1c0-ace1-74f1-84fe-86787cd3b7ce`, `01a1a1c0-b487-7e44-b080-c791e3aa4764`.

Audit IDs: `01a1a1c0-a916-78a5-8fd4-702780152337`.

Audit trigger **central_age_30s**, incident age **31.251664000 s**; reported Steps 1, elapsed 6 ms, alarm `false`, alarm report `None`.

Accepted matching report IDs: `01a1a1c0-bc65-7b40-83e8-bf09182cb8cd`.

Genuine forwarded/local lease `20cfb66d-fbf0-46d0-a26b-7bb8836aebc8` matches by body digest. Separate counted recovery **18541→21173 bytes**. Matching incident closure is not alone a serving assertion.

## Limits

All **13** accepted hold reports across **4** incidents join their own signed lease, node/incarnation/report sample and incident. All 13 applied generations match delivered desired state, 13 match the later central read. All **792** accepted central report IDs match lease-history receipts. Reverse-array classification agrees. 5 publication completions are retained; none credits a hold report or succeeds during an open incident.

Pure `stale_generation` cannot be claimed: genuine reports carry a failed QoS `read_at=...` diagnostic while central exempts only the exact dependent detail `Rate policy could not be applied.`. Classification is **10 check_failed / 3 isolation_unconfirmed**. [F17-01](follow-up-18-findings.json) retains the actual check/error fields and owner request. No independent safety failure is bypassed.
