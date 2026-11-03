# Follow-up 16 populated stale-lease rows

Run `83b457ef-ea16-490b-9923-1fea0df86b36`. Seven requested coverage categories pass across two original attempts; no retry.

Node **N1** = `01a1980a-002a-7249-ad06-6864ab31d3bc`; incarnation **I1** = `01a1980a-0141-7221-9c85-cf0b41558f8c`. Every row below belongs to N1/I1. Times are UTC on 2026-11-01. First/last seen are central incident receipt times, not inferred packet instants.

[Original attempts](populated-holds.json.gz), [accepted central reports and signed leases](generation-holds-central.json.gz), [checked joins](populated-holds-review.json).

## Incident 1

Arm `681d728a-1ef3-470d-8d5b-a24d1bfbbff8`; incident `01a1980c-fd00-7aee-ba24-8c8a94731c87`; audit `01a1980d-046e-7915-b3c7-b309245217e3`.

Node N1 / incarnation I1; first seen **2026-11-01T16:59:10.463824Z**, last seen **2026-11-01T16:59:17.361612Z**; ended **2026-11-01T16:59:18.323291Z**, `generation_matched`.

| Row | Report IDs | Outcome |
|---|---|---|
| Populated hold and independent denial | `01a1980c-fcf9-7272-b35a-4971f51cf072`; all reports listed below | pass; two old-lease members; captured ingress/no egress, timeout |
| Step 3 alarm and single escalation | `01a1980d-0466-7406-9d57-4c16830ebb51` | pass; audit above, reported alarm trigger, incident age 1.899726 s |
| Unrelated failure precedence | `01a1980d-1412-7281-8485-8005ee76aca7` | pass; apply ISOLATION_POLICY_INVALID, central check_failed |
| Active stop and local reset | before `01a1980d-1412-7281-8485-8005ee76aca7`; after `01a1980d-17ea-7128-b9ea-ea9911bb54d1` | pass; 7 Steps/5917 ms → 1 Step/0 ms; same incident/first_seen/audit; proxy armed |
| Matching recovery | `01a1980d-1bac-7575-923a-7314258a97a3` | pass; genuine forwarded/local lease `f3fffcb2-ce60-4cd9-a5de-c3808b5a1f71`, accepted report ends same incident; separate counted recovery |
| Later distinct incident | `01a1980c-fcf9-7272-b35a-4971f51cf072` | not exercised |

Complete accepted hold report IDs: `01a1980c-fcf9-7272-b35a-4971f51cf072`, `01a1980d-0466-7406-9d57-4c16830ebb51`, `01a1980d-0c39-79cf-9690-2654da1fd5db`, `01a1980d-1024-7457-835c-61cd911b3391`, `01a1980d-1412-7281-8485-8005ee76aca7`, `01a1980d-17ea-7128-b9ea-ea9911bb54d1`.

## Incident 2

Arm `bd5cfd83-dc18-43c9-be15-e8fd551ef788`; incident `01a1980d-a8ec-776a-8ce3-f980d4d19edb`; audit `01a1980d-b03a-7c04-9e13-3f7fc02e7311`.

Node N1 / incarnation I1; first seen **2026-11-01T16:59:54.475674Z**, last seen **2026-11-01T16:59:59.351073Z**; ended **2026-11-01T17:00:00.461554Z**, `generation_matched`.

| Row | Report IDs | Outcome |
|---|---|---|
| Populated hold and independent denial | `01a1980d-a8e5-7435-b577-d9123c85b398`; all reports listed below | pass; two old-lease members; captured ingress/no egress, timeout |
| Step 3 alarm and single escalation | `01a1980d-b033-71f6-bde1-26a7af457424` | pass; audit above, reported alarm trigger, incident age 1.868719 s |
| Unrelated failure precedence | none | not exercised in this incident |
| Active stop and local reset | none | not exercised in this incident |
| Matching recovery | `01a1980d-c046-7525-ba04-2cfc5b9039c9` | pass; genuine forwarded/local lease `937d6a82-14ed-4746-a5e0-7c3211ace817`, accepted report ends same incident; separate counted recovery |
| Later distinct incident | `01a1980d-a8e5-7435-b577-d9123c85b398` | pass |

Complete accepted hold report IDs: `01a1980d-a8e5-7435-b577-d9123c85b398`, `01a1980d-b033-71f6-bde1-26a7af457424`, `01a1980d-b429-7002-9479-a3733aeaa193`, `01a1980d-bbf0-779d-a06a-f14caca0ca3c`.

The first generation 16 matching report has forwarding=false. A later matching lease and independent counted probe (7935→10566 bytes) establish serving recovery separately. Generation 17 matching report is forwarding=true; counted recovery is 13199→15832 bytes. Signed responses are genuine central output, joined by lease ID and body digest to the local read; none was stored for replay.

The second arm occurs after first recovery. Both arms are fresh UUIDs and neither is a retry of an earlier incident. Pure stale_generation classification and central 30-second fallback are not exercised; the original opportunistic control-cut is not exercised. Kernel reboot remains the AT13 operator row. The additional brief unarmed incident in the general central export is retained separately in the summary.
