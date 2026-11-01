# AT02 finite grant and device-to-usage trace — Follow-up 11

Run `7edb263f-1f6e-4f94-9e8e-26099f96a699`, harness `33abb2fd2c2fc95826c6e69eb434045522d6f7ef`, product baseline `96ec0630e4834e13a3a14d65553ba8108bb5da55` (devstack 20 #343, agent-core 28 #344, netlab 20 #336; dns-publish 8 #309). Assigned `w4-members / 24000`, two resolvers. The full mutation order is unchanged.

Safe HTTP bodies, audits, operation IDs and original device-to-usage steps remain in [AT02.json](AT02.json).

| Withdrawal event | Last TCP increment | Last UDP increment | Samples | Failed reads | Result |
|---|---|---|---:|---:|---|
| 2026-10-31T18:32:39.672151678Z | 2026-10-31T18:32:40.646777527Z | 2026-10-31T18:32:40.646777527Z | 28 | 0 | pass |
| 2026-10-31T18:34:37Z | 2026-10-31T18:34:36.591993718Z | 2026-10-31T18:34:36.591993718Z | 186 | 0 | pass |

The finite-expiry row has 186 successful reads, both reader categories `none`, last TCP/UDP increments 408.006282 ms before deadline, active senders and stable/removed counters. Across both withdrawal rows, all 214 reads and 428 producer snapshots join the 798-record complete stream. [Counter diagnostics](F11-AT02-withdrawal.json), [producer records](AT02-enforcement-timing.json). These are command bounds, not exact kernel commit times. No failed read is retried or omitted.

Natural expiry still yields 0/3, with three audit rows and 3/0 restoration. D-119 keeps this separate from immediate 3/0 → 2/1 → 3/0 replacement through `ReplaceFiniteGrant`; Plan v1.0 has no scheduled plan-change contract. Worker completion requires the succeeded parent operation. The current revoke action is `member.grant_revoke`; one audit per grant remains required.

Full AT02 remains **fail, 121 pass / 0 blocked / 2 fail**, due to the D-077 readiness and admitted-sibling ingress controls (F11-02/F11-03). Their accepted-report extract is [retained](F11-AT02-observations.json). Historical F10-02 remains failed and unattributed in the [follow-up 10 capture](history/follow-up-10/F10-02-withdrawal.json). A successful finite-expiry reader measurement does not repair that sample or pass other failed assertions.
