# AT02 finite grant and device-to-usage trace — Follow-up 9

Run `11e305fe-6360-4dbd-95b1-67274aa4988c`, harness `49fbb48e013b824c4ce107d3cb551db79b915b42`, product baseline `7e044c71` (agent-core 25 #312, devstack 18 #313, dns-publish 8 #309 and netlab 18), assigned `w4-members / 24000`.

All safe HTTP bodies, audit rows, operation IDs and transport samples remain in [AT02.json](AT02.json).

| Transition | Authorized / pending before → after | Revision before → after | Deadline | Worker operation | Result |
|---|---|---|---|---|---|
| ReplaceFiniteGrant | 3/0 → 2/1 | 2 → 4 | 2026-10-31T12:40:23.854501239Z | 01a191c2-c7cb-7457-866a-4b4c9107ae50 | pass |
| ReplaceFiniteGrant | 2/1 → 3/0 | 4 → 6 | 2026-10-31T12:40:48.493154078Z | 01a191c3-280a-7be9-b165-a32a3297540c | pass |
| ReplaceFiniteGrant | 3/0 → 3/0 | 6 → 8 | 2026-10-31T11:42:25Z | 01a191c3-7f94-70c1-8e3b-e8c038b8c7d0 | pass |

D-119 (2026-10-31) retires the inherited scheduled-deadline downgrade interpretation. Plan v1.0 has no scheduled plan-change contract. Billing replaces a grant at the period boundary; natural expiry with no other grant yields 0/3. The harness measures both independently and fabricates no fallback.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a191c2-c7c3-74dd-b387-68688c7c9d1f` at `2026-10-31T11:40:23.86751+00:00`.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a191c3-2800-7bb9-bc50-1b5854d559f5` at `2026-10-31T11:40:48.505074+00:00`.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a191c3-7f8b-70a9-bd8c-294690200eb8` at `2026-10-31T11:41:10.915173+00:00`.

Natural expiry: deadline `2026-10-31T11:42:25Z`, observed `2026-10-31T11:42:33.858593867Z`, authorized/pending **0/3**, result **pass**.

Expiry restoration: operation `01a191c4-ca97-784e-be76-b45813c28409`; the separate completion check requires a succeeded parent operation, 3/0 and fresh counted forwarding.

The interrupted [attempt 1](history/follow-up-9/attempt-1/RUN.md) retained the stale `grant.revoke` query failures. The current selector is `member.grant_revoke`, matching the production service and its integration contract; exactly one audit per revoked grant remains mandatory.

Persistent senders retain the same TCP/UDP sockets through cutoff; [forwarding-observation.md](forwarding-observation.md) records timestamps and bounds. No identity, entitlement, binding, device or observation projection is repaired.
