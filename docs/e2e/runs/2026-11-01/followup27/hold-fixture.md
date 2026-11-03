# Follow-up 27 populated-hold fixture

2026-11-01. TEST ONLY. Final fresh runner **13 pass / 0 blocked / 0 fail**. The first two original runners stay **11 / 0 / 1** because of fixture readback bugs; their separate read-only supplements prove the released bytes without rewriting those failures. [All exact joins](hold-joins.json).

| Attempt | Runner | Populated incident duration | Escalation trigger / age | Audits | Recovery |
|---|---|---:|---|---:|---|
| 1: [132c9279](132c9279-6cbf-4d6b-b626-aaa60bf1f9e3/populated-hold.md) | fail | 49.984783 s | reported alarm / 9.885814 s | 1 | generation_matched |
| 2: [ff177856](ff177856-14b8-4c58-9880-2e5d8f9a0d16/populated-hold.md) | fail | 49.996047 s | reported alarm / 9.897527 s | 1 | generation_matched |
| 3: [cf799c62](cf799c62-df90-4e7e-ab53-610ab9058a7b/populated-hold.md) | pass | 50.000298 s | reported alarm / 9.887144 s | 1 | generation_matched |

The real capacity command moves desired/applied 7 → 8 while the generation-7 signed lease remains valid for two distinct members. Desired state and accepted reports continue; forwarding is false during the hold. Ten matching-generation responses are withheld in each run. A 45-second transport deadline releases on the next normal request, so the observed incident lasts about 50 seconds. No response is cached, replayed or re-signed.

In the final run, the node reports Step 3 / elapsed 9,891 ms / alarm=true. The audit joins that exact accepted report before central incident age 30 s: **the reported alarm triggered escalation**, not the age fallback. A later alarm report remains in the same incident with no second audit. Recovery uses the next genuine central-issued lease and the local body digest is equal byte-for-byte.

| Final join | ID / generation |
|---|---|
| Node | `01a197a3-6e35-7ad4-b222-6a74240ce7b5` |
| Incarnation | `01a197a3-7148-769d-8dc7-65242aac1194` |
| Arm | `141d6ebb-f2f0-48bf-a540-5ad27e7c2595` |
| Before serving report | `01a197a5-47d5-7a54-9958-0f001911517e` |
| Old populated lease | `6bcc30f4-b55b-4111-9e62-08c94c0906a9` / 7 |
| Incident | `01a197a5-5b5d-76c7-a866-c359da0703a8` |
| Hold report, Step 1, alarm=false | `01a197a5-5b57-79bb-a1bb-0db65382757c` |
| Hold report, Step 3, alarm=true | `01a197a5-81f4-7284-8fc3-5e2c8712547f` |
| Hold report, Step 10, alarm=true | `01a197a6-0aab-706a-8a3a-949162e94f16` |
| Escalation audit | `01a197a5-81fd-7151-b4ab-d77111a39d0b` |
| Released matching lease | `d6075d26-1b48-4bfc-b0fa-4b93f4b9fd2c` / 8 |
| Matching recovery report | `01a197a6-1ea5-78a5-845e-01ac2ab5e520` |

Incident first_seen `2026-11-01T15:05:58.876729+00:00`, last_seen `2026-11-01T15:06:43.762138+00:00`, ended `2026-11-01T15:06:48.877027+00:00`. The recovery report is in the exported `reports` array; it does not add another hold sample.

Use the [exact arming contract](../../../scenarios/harness-exports.md#follow-up-27--bounded-populated-generation-hold-test-only). The shared fixture proves alarm, single escalation, and matching recovery only. Isolation and members own unrelated-failure precedence within the same incident, agent-stop retention with local counter reset, a distinct later incident on the same node, and sole-member `members_revoked` closure. None of those rows is passed by these fresh independent stacks. No product defect was demonstrated.

Attempt 01 retained a host permission failure; attempt 02 retained the helper build working-directory failure. Their dated amendments and local supplements are preserved. The final runner fixes both without altering store permissions or product state. Historical bootstrap incidents with empty leases remain separate from the populated incidents above.
