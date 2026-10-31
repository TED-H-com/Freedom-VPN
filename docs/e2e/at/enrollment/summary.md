# Enrollment acceptance — follow-up 9

AT06 **662 pass / 0 blocked / 0 failed**; AT07 **48 pass / 0 blocked / 0 failed**. All five formerly blocked AT07 rows pass.

Run `891ee9b8-bf01-4583-a47a-9542c8b04c8a`, `2026-10-31T14:59:07Z`–`2026-10-31T15:16:38Z`.

| Row | Status | Evidence |
|---|---|---|
| installer_rerun | pass | Same identity, stopped file equality, restored serving and counted traffic. |
| controlled_reinstall | pass | Same Node, new incarnation, retired certificate denial, fresh serving, gap ACK 1 and counted event ACK 2. |
| counted_unacked_usage | pass | Held CLI records plus persistent TCP/UDP receipts and admission counters. |
| identity_ledger_preserved | pass | Original canonical payloads/hashes/ACK survive production adoption and original-stream upload. |
| historical_usage_retained | pass | Four named AT20 assertions, once-only samples and distinct retired/current totals. |

Retired usage: **908 + 1816 = 2724 bytes**, original ACK **3**. The fresh stream independently advances **1 → 2**, attributing **550 upload + 358 download = 908 bytes** to its own incarnation, member, device and binding revision. Historical payloads, hashes, samples and totals remain unchanged.

[AT06](AT06.json), [AT07](AT07.json), [gap resolution](current-stream-gap.md), [runbook](RUN.md), [D-098 archive](history/follow-up-8/README.md).

Physical arm64 and OS reboot remain unmeasured; arm64 compilation and agent process restart are separate evidence. Post-upgrade seed behaviour belongs to **devstack 19 / installer 7 / updater 18** and is not re-measured here. The [historical devstack 18 seed failure](../../runs/2026-10-31/followup18/README.md#candidate-health-and-post-upgrade-reenrollment) and [updater diagnosis](../../../updates/node-updater.md#post-upgrade-seed-diagnosis--2026-10-31) remain references, not new enrollment acceptance. This run performs no upgrade.
