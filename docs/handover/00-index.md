# Handover index

**Current cut — [follow-up 29](follow-up-29.md) (2026-11-01 EST):** Through #396: same-boot deny-set adoption gap open; isolation 15 unqualified on a 3.821 µs interleave, signal-15 cause unproven. Devstack 27 proves the TEST ONLY populated hold; consumer rows remain open. D-120 measurement end fixed; every failed run stays failed. D-135–D-138, D-140 and D-141 remain provisional. Seven owner decisions retained.
**Previous cut — [follow-up 28](follow-up-28.md):** #386–#390 at aae6763f; 19 subtests, five rows, 16 snapshots, 1,644 checks and 47 findings. Historical failures stay failed.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
