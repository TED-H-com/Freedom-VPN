# Follow-up 30 — Bounded Survivor-Hold Continuation Proof (2026-11-03)

This directory retains the fresh-stack live proof for Follow-up 30's opt-in TEST ONLY survivor-hold continuation in the D-102 lease-hold seam (`Stack.ArmMatchingLeaseHoldSurvivor`).

## Summary

- Run ID: [`8576dfe5-aa76-417c-9c12-f104cef47fdb`](8576dfe5-aa76-417c-9c12-f104cef47fdb/survivor-hold-continuation.md)
- Source commit before commit of Follow-up 30: `38d10b3b450b41a67ae5223d05e82c05c659c429`
- Attempts: 1 attempt (`10 pass / 0 blocked / 0 fail`), no retries.
- Records:
  - [`8576dfe5-aa76-417c-9c12-f104cef47fdb/survivor-hold-continuation.json`](8576dfe5-aa76-417c-9c12-f104cef47fdb/survivor-hold-continuation.json)
  - [`8576dfe5-aa76-417c-9c12-f104cef47fdb/survivor-hold-continuation.md`](8576dfe5-aa76-417c-9c12-f104cef47fdb/survivor-hold-continuation.md)
  - [`8576dfe5-aa76-417c-9c12-f104cef47fdb/central-record.json`](8576dfe5-aa76-417c-9c12-f104cef47fdb/central-record.json)

## Recorded IDs and Timeline

| Field | Value |
|---|---|
| `node_id` | `01a1a02d-29a3-74ce-b6bf-f63f739267c4` |
| `incarnation_id` | `01a1a02d-2d37-786c-a359-090bc44c43a9` |
| `survivor_member_id` | `01a1a02d-7200-7f22-b4ba-9e1c79ce69ee` |
| `sibling_member_id` | `01a1a02d-df51-7e7d-8e43-afd3884056b0` |
| `retained_lease_id` (generation `8`, 2 members) | `0e45897f-2b45-48dd-9956-4cfe2c987274` (`sha256: a2894628abfefd89d5f98101d4aa9da26fdc2dcae6d760e9bccf571a9504cae8`) |
| `arm_id` (`hold_seconds: 90`, `max_continued_generations: 1`) | `c81a7351-ce93-451b-b988-68aedb4fbb90` |
| Initial node capacity command (`PATCH /api/v1/admin/nodes/{id}`) | `operation_id: 9ea0bcfa-02b7-4816-86a3-6dadf085724a`, `desired_generation: 9` |
| `incident_id` | `01a1a02f-3e1d-7521-836c-c994dfc8190a` |
| `entry_hold_report_id` (Step 1, applied `9` / lease `8`) | `01a1a02f-3e18-75c9-a1ca-df9ecde0daab` (`2026-11-03T06:53:33.084681+00:00`) |
| `alarm_hold_report_id` (Step 3, `reported_alarm: true`, `elapsed_ms: 9916`) | `01a1a02f-64cf-7760-b675-b7f64dbb77d3` (`2026-11-03T06:53:42.997065+00:00`) |
| `escalation_audit_id` (`node.generation_hold.escalated`, count `1`) | `01a1a02f-64d5-7ee1-93c3-951a99af5e9a` (`2026-11-03T06:53:42.995957+00:00`) |
| Sibling suspension (`PATCH /api/v1/admin/members/{sibling}`) | `operation_id: 01a1a02f-6675-713d-98b1-22190987579d`, completed `2026-11-03T06:53:43.415287241Z` |
| Proxy `continuation_started` | `continued_generation: 11` at `2026-11-03T06:53:47.941470275Z` (withheld survivor-only lease `53dd9d93-50d3-4d7e-bce7-72a5ba53888e`) |
| `post_command_hold_report_id` (Step 4, applied `11` / lease `8`, `reported_alarm: true`) | `01a1a02f-785d-79f2-a660-7a99fc16c580` (`2026-11-03T06:53:48.003245+00:00`, while incident open) |
| Explicit release (`ReleaseMatchingLeaseHold`) | `2026-11-03T06:53:52.925435523Z` (`reason: explicit_release`) |
| `recovery_survivor_lease_id` (forwarded generation `11`, survivor only) | `b36b4be7-84c2-4728-8e36-7049d5523c48` (`sha256: bec6929bb25334fc7a4082e2cb6a908718207e3ebb8a85a064fe935067c7c2ee`, matches local durable lease) |
| `recovery_serving_report_id` (applied `11`, `member_forwarding_enabled: true`, closes incident `generation_matched`) | `01a1a02f-8c27-7587-b6fe-96f8b865a6db` (`2026-11-03T06:53:53.070299+00:00`) |
