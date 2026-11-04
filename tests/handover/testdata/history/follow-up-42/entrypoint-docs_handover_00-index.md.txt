# Handover index

**Current cut — [follow-up 41](follow-up-41.md) (2026-11-03 EST):** #435–#438 at c359d4a9: agent-core 45 has no comparison; members 19 restores early returns (offline only); devstack 32 handles exact-code 412s, with forced real-API proof 8/8. Isolation’s assembled run is pending in 20; agent-core 46 and read-only r27 are status only. 55 decisions, 105 findings; no operator/member behavior change.
**Previous cut — [follow-up 40](follow-up-40.md):** #430–#434 at 3f7dbc25; D-146/D-147 on main, D-151/D-152 provisional; 14 subtests, 76 immutable records, 11 run snapshots, 84 Go declarations; 55 decisions and 102 findings.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
