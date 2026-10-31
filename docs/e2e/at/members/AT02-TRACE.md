# AT02 finite grant and device-to-usage trace — Follow-up 10

Run `2be2d32f-bfd9-46f5-bf29-afaa6617ae94`, harness `731be5a7af7657660d96c240695747a23a3f5c96`, product baseline `d3e9a0f1` (agent-core 27 #325, wire-nodes 32 #321, netlab 19 #318; devstack 18 and dns-publish 8 retained), assigned `w4-members / 24000`.

All safe HTTP bodies, audit rows, operation IDs and transport samples remain in [AT02.json](AT02.json).

| Transition | Authorized / pending before → after | Revision before → after | Deadline | Worker operation | Result |
|---|---|---|---|---|---|
| ReplaceFiniteGrant | 3/0 → 2/1 | 2 → 4 | 2026-10-31T14:34:37.640762951Z | 01a1922b-5c65-7034-ba3c-05dc2c13cc25 | pass |
| ReplaceFiniteGrant | 2/1 → 3/0 | 4 → 6 | 2026-10-31T14:35:00.828316625Z | 01a1922b-b6f6-7ee5-b899-c7989fc9cb19 | pass |
| ReplaceFiniteGrant | 3/0 → 3/0 | 6 → 8 | 2026-10-31T13:36:39Z | 01a1922c-11bd-78f7-b08e-19e8356d5eed | pass |

D-119 (2026-10-31) retires the inherited scheduled-deadline downgrade interpretation. Plan v1.0 has no scheduled plan-change contract. Billing replaces a grant at the period boundary; natural expiry with no other grant yields 0/3. The harness measures both independently and fabricates no fallback.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a1922b-5c5c-721a-8915-62194f01213e` at `2026-10-31T13:34:37.651379+00:00`.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a1922b-b6ee-7731-8af8-357c6903a84e` at `2026-10-31T13:35:00.839457+00:00`.

Revoke audit replay: **pass**; `member.grant_revoke` operation `01a1922c-11b4-7901-9e29-07d03f52406c` at `2026-10-31T13:35:24.07469+00:00`.

Natural expiry: deadline `2026-10-31T13:36:39Z`, observed `2026-10-31T13:36:53.082712748Z`, authorized/pending **0/3**, result **pass**.

Expiry restoration: operation `01a1922d-7837-7583-8130-6a142831d0c6`; the separate completion check requires a succeeded parent operation, 3/0 and fresh counted forwarding.

The interrupted [attempt 1](history/follow-up-9/attempt-1/RUN.md) retained the stale `grant.revoke` query failures. The current selector is `member.grant_revoke`, matching the production service and its integration contract; exactly one audit per revoked grant remains mandatory.

Persistent senders retain the same TCP/UDP sockets through cutoff; [forwarding-observation.md](forwarding-observation.md) records timestamps and bounds. No identity, entitlement, binding, device or observation projection is repaired.

**F10-02:** this run's complete AT02 result is fail (121/0/1): one of the
234 finite-expiry persistent samples has `read_ok=false`. Reply cutoff, 0/3
expiry and 3/0 restoration pass; the missing read is retained as a failure.
[Full failed-check capture](F10-02-withdrawal.json).
