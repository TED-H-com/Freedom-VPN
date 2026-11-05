# Handover index

**Current cut — [follow-up 47](follow-up-47.md) (2026-11-04 EST):** #474–#477 at 1c39a0e7; 66 decisions, 136 findings. D-163 fixes the scheduled mapping; F24-01 keeps isolation's local scheduled run red. D-162 is TEST ONLY and implemented; F22-D146 stays open. D-158 records its first live wait, 446 ms acquired. r31 archived, M-1 fixed; observation records documented.
**Previous cut — [follow-up 46](follow-up-46.md) (2026-11-04 EST):** #469–#473 at 52b93845; F22-01 answered, all five r30 findings fixed, clarified D-158 implemented. 65 decisions, 133 findings; D-162 is TEST ONLY and running with devstack 36. Members AT02 adds 179 ms / 7 losses; D-158’s wait remains unmeasured live.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
