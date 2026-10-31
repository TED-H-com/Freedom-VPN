# Enrollment acceptance requests — follow-up 9

2026-10-31 — **0 requests to orchestrator.** ENR-07 is closed by the measured D-130 result; all five rows pass.

| Row | Status | Evidence |
|---|---|---|
| installer_rerun | pass | Same identity, stopped file equality, restored serving and counted traffic. |
| controlled_reinstall | pass | Same Node, new incarnation, retired certificate denial, fresh serving, gap ACK 1 and counted event ACK 2. |
| counted_unacked_usage | pass | Held CLI records plus persistent TCP/UDP receipts and admission counters. |
| identity_ledger_preserved | pass | Original canonical payloads/hashes/ACK survive production adoption and original-stream upload. |
| historical_usage_retained | pass | Four named AT20 assertions, once-only samples and distinct retired/current totals. |

Retired usage: **908 + 1816 = 2724 bytes**, original ACK **3**. The fresh stream independently advances **1 → 2**, attributing **550 upload + 358 download = 908 bytes** to its own incarnation, member, device and binding revision. Historical payloads, hashes, samples and totals remain unchanged.

Run `891ee9b8-bf01-4583-a47a-9542c8b04c8a` re-asserts all four named [AT20 assertions](../usage/AT20.md). See [full JSON](AT07.json) and [resolution](current-stream-gap.md).

## D-102 answers consumed

The [enrollment and usage responses](../../RESPONSES.md) supply the installer-node hold, snapshot, transport and service wrappers, API retirement, same-node replacement, coherent stopped ledger export and original/current ACK selection. D-116/#302 and D-118/#312 remain measured prerequisites. No manual installer invocation, SQL transition or synthetic observation supplies this result.

## Known gaps

Physical arm64 and OS reboot remain unmeasured; arm64 compilation and agent process restart are separate evidence. Post-upgrade seed behaviour belongs to **devstack 19 / installer 7 / updater 18** and is not re-measured here. The [historical devstack 18 seed failure](../../runs/2026-10-31/followup18/README.md#candidate-health-and-post-upgrade-reenrollment) and [updater diagnosis](../../../updates/node-updater.md#post-upgrade-seed-diagnosis--2026-10-31) remain references, not new enrollment acceptance. This run performs no upgrade.
