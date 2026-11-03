# Follow-up 17 populated stale-lease rows

Run `0ec14b39-467e-4a37-a239-1a33ef12eb71`. Three distinct planned incidents; no retry. All original seven coverage categories pass across the first two incidents. Central 30-second fallback passes in the third. Pure `stale_generation` remains **not exercised**.

[Original attempts](populated-holds.json.gz), [accepted reports and own signed leases](generation-holds-central.json.gz), [checked joins](populated-holds-review.json), [general review](generation-holds-review.json).

## Incident 1

Arm `e2d95227-e5ed-4a02-8007-c1a7d03cb744`; incident `01a1a011-b499-74b1-9d04-dc9f95871230`. Node `01a1a00e-b542-7930-bdc9-226a3f5d56ab`, incarnation `01a1a00e-b8c2-761e-a8dd-5f6dffb3183e`. Old lease `9c7f35ee-910a-4402-b1ea-27a45d256c22`, generation 12, **2 members**.

First seen **2026-11-03T06:21:17.336384+00:00**, last seen **2026-11-03T06:21:24.032983+00:00**; ended **2026-11-03T06:21:25.009765+00:00**, `generation_matched`. These are central receipt times, not packet instants.

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

Accepted hold report IDs: `01a1a011-b492-797c-964b-a626658a75aa`, `01a1a011-bc0b-74b4-ae75-1c0e4f96f35a`, `01a1a011-bff9-748f-8e8d-dec6775da53a`, `01a1a011-c7ce-79b9-9d4d-1998c4ea05c7`, `01a1a011-cbbe-7de5-b53e-cefe1c7aff25`, `01a1a011-cebb-783e-bdce-33456102c8b0`.

Audit IDs: `01a1a011-bc11-7670-ac23-4a047797200b`.

Audit trigger **reported_alarm**, incident age **1.911284000 s**; reported Steps 3, elapsed 1916 ms, alarm `true`, alarm report `01a1a011-bc0b-74b4-ae75-1c0e4f96f35a`.

Active-stop reset: report `01a1a011-cbbe-7de5-b53e-cefe1c7aff25` **7 Steps / 5936 ms** → `01a1a011-cebb-783e-bdce-33456102c8b0` **1 Step / 0 ms**. Same open incident, first_seen, single audit, and still-armed proxy.

Unrelated failure report: `01a1a011-cbbe-7de5-b53e-cefe1c7aff25`; classification remains `check_failed`.

Accepted matching report IDs: `01a1a011-d28b-7974-a035-fc2356888ccf`.

Genuine forwarded/local lease `67569b9f-59d0-4ae6-82f0-a7428b059727` matches by body digest. Separate counted recovery **7937→10608 bytes**. Matching incident closure is not alone a serving assertion.

## Incident 2

Arm `f03cfbae-13bf-4daf-b02f-14a80c410476`; incident `01a1a012-5bd1-7f24-b817-8d485cb2b98f`. Node `01a1a00e-b542-7930-bdc9-226a3f5d56ab`, incarnation `01a1a00e-b8c2-761e-a8dd-5f6dffb3183e`. Old lease `b082ef7a-ca7c-4205-a2ee-12683198548a`, generation 13, **2 members**.

First seen **2026-11-03T06:22:00.14538+00:00**, last seen **2026-11-03T06:22:05.032393+00:00**; ended **2026-11-03T06:22:06.116319+00:00**, `generation_matched`. These are central receipt times, not packet instants.

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

Accepted hold report IDs: `01a1a012-5bcc-78bc-b7e3-610c450e7f59`, `01a1a012-6324-7de7-9554-3ff07fb3d0f0`, `01a1a012-6718-7401-8ccf-9ef5b2fc29bc`, `01a1a012-6ee1-7a98-be01-74d4c166ec3f`.

Audit IDs: `01a1a012-632c-7af8-a174-04653f176992`.

Audit trigger **reported_alarm**, incident age **1.880655000 s**; reported Steps 3, elapsed 1884 ms, alarm `true`, alarm report `01a1a012-6324-7de7-9554-3ff07fb3d0f0`.

Accepted matching report IDs: `01a1a012-731d-7aaa-8833-0ff58c86c5d9`.

Genuine forwarded/local lease `2a1d01a5-5d33-432e-8ded-3911016b84f3` matches by body digest. Separate counted recovery **13281→15953 bytes**. Matching incident closure is not alone a serving assertion.

## Incident 3

Arm `8f2a4045-f023-41ae-ad2d-1e9930015c58`; incident `01a1a012-f825-7e4a-ab4c-268766baa273`. Node `01a1a00e-b542-7930-bdc9-226a3f5d56ab`, incarnation `01a1a00e-b8c2-761e-a8dd-5f6dffb3183e`. Old lease `5ae9a52b-7830-4aa7-b709-134d2330a817`, generation 14, **2 members**.

First seen **2026-11-03T06:22:40.165098+00:00**, last seen **2026-11-03T06:23:14.470702+00:00**; ended **2026-11-03T06:23:15.458937+00:00**, `generation_matched`. These are central receipt times, not packet instants.

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

Accepted hold report IDs: `01a1a012-f81f-790f-889b-adea472405a5`, `01a1a013-727a-7f2e-8417-9b38d3f24ddb`, `01a1a013-7a1f-77f9-ae30-5448eab4b39d`, `01a1a013-7e1f-7c24-b045-1ffd29a7b8ba`.

Audit IDs: `01a1a013-7057-7843-b023-47ac112e60c8`.

Audit trigger **central_age_30s**, incident age **30.769780000 s**; reported Steps 1, elapsed 4 ms, alarm `false`, alarm report `None`.

Accepted matching report IDs: `01a1a013-81fc-787e-826f-b57fbb3df83b`.

Genuine forwarded/local lease `0951f9bd-1fde-483d-bbc3-1731e3d8b0b4` matches by body digest. Separate counted recovery **18584→21215 bytes**. Matching incident closure is not alone a serving assertion.

The agent stopped after the original Step-1 report; the proxy stayed armed. Central escalated before the agent restarted, with no alarm report. The third incident does not claim a local counter decrease: 1→1 would not prove one. Its stop receipt remains in the six-stop review.

## Limits

All **16** accepted hold reports across **five** incidents join their own signed lease, node/incarnation/report sample and incident; two additional short unarmed incidents remain separate from these three planned arms. All 16 applied generations match delivered desired state, 15 match the later central read. All **806** accepted central report IDs match lease-history receipts. Reverse-array classification agrees. Six publication completions are retained; none credits a hold report or succeeds during an open incident.

Pure `stale_generation` cannot be claimed: genuine reports carry a failed QoS `read_at=...` diagnostic while central exempts only the exact dependent detail `Rate policy could not be applied.`. Classification is **12 check_failed / 4 isolation_unconfirmed**. The four post-restart samples retain `safety_status=expired` with the expected dependent QoS text; safety precedence is not bypassed. The seam cannot rewrite those reports. [F17-01](follow-up-17-findings.json) retains the actual check/error fields and owner request. No independent safety failure is bypassed.

The first general offline review incorrectly demanded counter-reset evidence from the separate Step-1 fallback experiment. [Original output](initial-general-hold-review.json) and [correction](consumer-review-corrections.json) are retained; the corrected review excludes only that inapplicable reset assertion, retaining every third-incident report/lease/sample/audit join. No physical run was repeated.
