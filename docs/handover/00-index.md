# Handover index

**Current cut — [follow-up 43](follow-up-43.md) (2026-11-04 EST):** #446–#456 at 04d31ea5; D-155/D-156 fixes merged, F30-01 acceptance pending; D-157 routed, no merged fix. 60 decisions, 124 findings; r29 archived. AT29 failed at setup. See [update recovery](operator-guide.md#f30-01), [r28 H-1 fixed](operator-guide.md#r28-h-1), [release rule](operator-guide.md#d156-release-rule) and [closed-recovery interval](operator-guide.md#at02-closed-recovery).
**Previous cut — [follow-up 42](follow-up-42.md):** #439–#445 at 5a942aa1; 15 subtests, 57 decisions and 113 findings; historical verdicts unchanged.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
