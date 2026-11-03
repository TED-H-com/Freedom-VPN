# Handover index

**Current cut — [follow-up 40](follow-up-40.md) (2026-11-03 EST):** #430–#434 at 3f7dbc25: D-146/D-147 on main (#432); members 18’s three receive-only TCP rows pass under provisional D-151, while AT09 stays failed. Isolation 19’s four source sets pass in the lab; assembled checks blocked by capacity PATCH 412. Agent-core 44 has no comparison; 45 repeats. D-152 answers F19-01; devstack 32 and members 19 requests stay open. 55 decisions.
**Previous cut — [follow-up 39](follow-up-39.md):** #428–#429 at 1b87ab32; 11 subtests, two final rows/PR excerpts, ten frozen inputs, exact r26 and separate audit, 15 dated documents/75 Go tests; 99 findings, 53 decisions, all ten owner requests answered.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
