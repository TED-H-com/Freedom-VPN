# Handover index

**Current cut — [follow-up 45](follow-up-45.md) (2026-11-04 EST):** #464–#468 at c148cf34; F30-01 closed, AT02 deny-all measured at 205 ms / 8 losses, AT29 blocked on the guest clock. 64 decisions, 133 findings; D-161 implemented, r30 archived and routed. D-158's wait remains unmeasured live. See [update recovery](operator-guide.md#f30-01), [closed recovery](operator-guide.md#at02-closed-recovery) and [TCP guard limitation](operator-guide.md#tcp-reload-guard).
**Previous cut — [follow-up 44](follow-up-44.md) (2026-11-04 EST):** #457–#463 at 5c21e670; D-158 lock wait and D-157 renewal merged, F30-01 candidate proof and live AT02 measurement pending. 64 decisions, 126 findings; F22-01 and SECOPS-F17-01 open, AT29 failed and confounded. See [update recovery](operator-guide.md#f30-01), [closed recovery](operator-guide.md#at02-closed-recovery) and [TCP guard limitation](operator-guide.md#tcp-reload-guard).

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
