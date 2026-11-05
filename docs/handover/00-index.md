# Handover index

**Current cut — [follow-up 49](follow-up-49.md) (2026-11-05 EST):** #482–#485 at aaffdc7f (#485 outside the expected three-lane list, included in full); 67 decisions (14 confirmed by Ted on 2026-11-04 17:1x EST, 1 new provisional D-164, 52 unchanged), 138 findings (6 updated, 0 added, 132 unchanged). Six Node CI failures fixed in #484 and reviewed by r33 (f0bed96a, 0 findings, 10/10 parent / 33/33 subtest replay); clone runtime control_url repaired offline in #482 (live isolation validation and copied-release SNAT open); D-164 ServingReadEvidence contract compatibility in #485 (API 1.11.0); NetBird #8473 comment 19411545 posted (2026-11-05T04:30:00Z, F22-AT01/F11-01 still open under D-148).
**Previous cut — [follow-up 48](follow-up-48.md) (2026-11-04 EST):** #478–#481 at b921c75f; 66 decisions, 138 findings. F24-01 closed and isolation's local scheduled-mode run bd73d7a4 passes (GitHub's 11-04 scheduled job 41836274140 stays failed, next run pending); assist.py retry and D-162 reach audited; F22-D146 answered under D-162 with 18 stored/enqueued/accepted records across 9 windows; r32 archived (0 findings).

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
