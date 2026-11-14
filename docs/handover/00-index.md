# Handover index

**目前交付：**[交付導覽](../DELIVERY.md)整理現有功能、操作入口與驗證範圍。下列 follow-up 快照保留各自當時的範圍與結果。

**Current delivery:** The [delivery guide](../DELIVERY.en.md) covers current capabilities, operating entrypoints and verification scope. The follow-up snapshots below retain their historical scope and outcomes.

**Archived cut:** [follow-up 54](follow-up-54.md), #533, #534, #535 and #536 at `2f5159f3`; source-backed changes, verification boundaries and carried gaps.
**Earlier archived cut:** [follow-up 53](follow-up-53.md), #529, #528, #530, #531 and #532 at `e055143e`; source-backed changes, verification boundaries and carried gaps.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
