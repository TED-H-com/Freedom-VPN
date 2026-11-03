# Handover index

**Current cut — [follow-up 30](follow-up-30.md) (2026-11-03 EST):** Through #403: agent-core 36 refuses same-boot adoption on changed deny / host sets; r20's 2 High / 3 Medium / 1 Low are routed, not yet judged, and the live-drift verifier check stays open with netlab 27. Isolation 16 unqualified on a 3.609 µs overlap; F16-01's fixture disposition merged, its run stays failed. Members 16: AT02 / AT09 fail. The nightly billing failure was a missing runner browser; no scheduled-run result is claimed. D-142–D-144 registered. Nine owner decisions.
**Previous cut — [follow-up 29](follow-up-29.md):** #391–#396 at d640ce96; 22 subtests, six rows, eight snapshots, 731 checks and 51 findings. Historical failures stay failed.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

Start with the [runbook catalogue](README.md). The [database and retention index](11-database-and-retention.md) includes `00189_node_incarnation_usage_outbox.sql` (`#243`, SECOPS-19 / D-075 / D-077 §2). The migration inventory comes from `db/migrations/*.sql`; use the deployment checklist to compare the complete inventory with the target database.

D-084 adds `00190_serving_leases_clamp.sql` (#259): run the forward migration before deploying the freeze clamp. Fleet-update follow-up 10 (#273) adds `00191_installer_update_results.sql` with Down for D-089 / D-097. Apply it before the new control binary, whose node summary reads the table. Db follow-up 2 (#280) enforces the test-time Down rule above 00188; production migrations remain forward-only.
