# Requests to orchestrator — follow-up 7

Run `71078ab8-edf8-4d23-af72-ea247f704011`, source `c8d4a00cf1bf594b20016263c2cfc3c164d92bc9`, evidence `2026-10-30T18:09:36Z`. Prefix `w4-updates`, port base 42000.

AT24/AT26/AT27 pass. Make exit **2** solely for retained AT23/AT25, which
remain blocked on F5-02 and were not rerun.

## Open: F5-02 — probe-peer export

**Owner: w3-devstack-e2e follow-up 12 (probe-peer export).** Export the existing
shared dedicated peer setup/config and read-only observer. AT23 and AT25 keep
the follow-up 6 records and original run IDs/timestamps. No host access to the
0700 journal, peer construction, alias agent config or synthetic challenge is
used to bypass this prerequisite. Rerun AT23/AT25 in follow-up 8 after the
export lands. [Retained probe evidence](probe-issuance.json).

## Resolved: F6-02 — corrected recorded c2/c3 expectations (#186)

The orchestrator confirmed the earlier expectations were a misreading.
Accepted results own current/last-good and recovery; later heartbeats only
confirm. Once any result is accepted, seeding closes for later jobs in that
incarnation. Live c2/c3 retain accepted 1.0.1, withhold the observation timestamp
and leave unconfirmed=true, with **zero** seeding-path unexpected-release
audits. Fresh-incarnation unexpected claims still produce exactly one audit
per job/release despite repetition. A fresh agreeing null last-good permits
current seeding without establishing last-good. [Policy quotes and exact
observations](pre-result.md), [raw evidence](pre-result.json).

The implementation remains unchanged: `internal/fleetupdate/serving.go:125`
limits seeding/audits to before first acceptance, `:142` requires agreeing
last-good, and `:159` withholds mismatching job observation timestamps.
AT24 now also proves offline result completion and post-batch inheritance
through a separate successful first_sync job, while preserving the original
denominator. [Raw inheritance evidence](inheritance.json).

## Earlier resolved findings

F5-01 remains resolved: fresh case c's unrecorded boot release gets one audit;
no boot provenance is fabricated. The real first result establishes last-good.
The #170 agent_config path correction remains recorded in follow-up 6; F6-01
was never observed. #129 terminal rollback and #136 target evidence are freshly
verified by AT26, including candidate failure, restored health, current ACK
and later authenticated confirmation. No product changes were made.
