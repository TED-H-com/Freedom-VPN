# Updates follow-up 8

Run `71f2aa37-b0b9-4568-abe0-2ba7665377e4`, source `bc93ce430ebe8a50e5dfdaf4121352047fe89249` (on `main` `52dc77026cb168df5684ae2ccd5cbcadb4dbaeff`), evidence `2026-10-30T20:59:15Z`. Prefix `w4-updates`, port base 42000.

Fresh-stack make exit **0** (Go test **0**). All five scenarios (`AT23`–`AT27`) executed in a single run on the fresh stack and **pass**:

| AT | Extended status | Decisive observation |
|---|---|---|
| [AT23](AT23.md) | pass | Real 1.0.1 candidate answered D-053 with exported D-063 probe peer (`#204`) issued in `health-challenge.json` (`configured_probe_issued=true`, 0 omissions) and bounded D-064 exception verified (`authenticated_udp_replies=14`, `grant_samples=4`, `elements_after=0`); `locally_succeeded`, then central `succeeded` and confirmed |
| [AT24](AT24.md) | pass | Under D-068 (`#189`), accepted 1.0.1 remains authoritative across c1/c2/c3; offline target caught up through a real 1.0.2 success result; post-batch `node2` inherited 1.0.2 through its own completed `first_sync` job with original snapshot `target_count=1` unchanged |
| [AT25](AT25.md) | pass | Byte-tampered 1.0.2 archive rejected at preflight (`HASH_MISMATCH`, `stage=""`) before any transaction, candidate executable, or health challenge (`fresh_challenges=[]`, `grant_samples=0`, `elements_after=0`); probe scope does not apply to an archive that never passes preflight; null-health `ARTIFACT_INVALID` result accepted and `failed_requires_cause` verified |
| [AT26](AT26.md) | pass | Candidate `tunnel_probe=failed`, healthy 1.0.1 restored (`allPassed()`), one current `rolled_back` ACK (`accepted_as_history=false`), confirmed target with `needs_attention=true` and revision 7 fields |
| [AT27](AT27.md) | pass | Under D-068 (`#189`), fresh unexpected releases on the current job are audited once (`1.0.2`, `1.0.3`, `0.1.0`), recorded c2/c3 emit 0 seeding audits and withhold `last_observed_at` (`unconfirmed=true`), and genuine stale ACK replay returns HTTP 202 with its original ACK without moving the newer job |

## Exported probe peer verification (F5-02 closed by #204)

`stack.SetupUpdaterProbe(ctx, "node", EnrolledNode{NodeID, IncarnationID})` provisions the dedicated non-member setup-key peer (`probe_not_member=true`, `probe_route_selected=true`) and supplies `probe.Config` and `probe.AgentConfig` (`/etc/vpn-node/agent-run.json`) to `/var/lib/vpn-node/e2e/updater.json`. Both the exported `probe.Observe(ctx)` observer and the in-node read-only journal stream (`health-challenge.json` / `health-attest.json` via `docker exec`, with no host read of the `0700` journal and raw `peer_id` redacted in published evidence) were captured:

- **AT23**: `configured_probe_issued=true`, `omission_logs=[]`, `probe_observer.verified=true` (`authenticated_udp_replies=14`, origin receipt `requests=14` from node SNAT `192.0.7.20`, `grant_samples=4`, `max_timeout_seconds=44`, `probe_up_bytes=1078`, `probe_down_bytes=1078`, `member_bytes_before=0`, `member_bytes_after=0`, `elements_after=0`).
- **AT25**: Tampered archive fails preflight (`HASH_MISMATCH`) before `stageCheck`, so `fresh_challenges=[]` and `probe_observer` records `grant_samples=0`, `authenticated_udp_replies=0`, `elements_after=0`. Probe scope does not apply to an archive that never passes preflight.

## D-068 alignment (#189)

We re-read `docs/updates/central-lifecycle.md` ("Serving and observed versions", "Result and observation authority") after PR #189 (`c5f3c65`, **D-068**) and confirmed all functional assertions in `restated_test.go` and `preresult_test.go` hold unchanged. Two verbatim policy quotes in `restated_test.go` were updated to match the D-068 wording in `docs/updates/central-lifecycle.md`:

> A generation-bearing heartbeat is attributed to the **node's current job only**
> (§13.2, §13.8; D-068, r13 M-1).

Source: `docs/updates/central-lifecycle.md:434`.

> Before that incarnation's first accepted result, pre-result heartbeat seeding
> fills **only a null `nodes.current_release_id`**

Source: `docs/updates/central-lifecycle.md:454`.

> On the current job, each distinct unexpected release/job pair
> gets one `update_observation.unexpected_release` audit

Source: `docs/updates/central-lifecycle.md:479` (replacing the pre-D-068 wording `"Each distinct unexpected\nrelease/reported-job pair gets one `update_observation.unexpected_release` audit"`).

> A
> later agreeing null last-good permits the issued target to fill the null
> `current_release_id`, without establishing `last_good_release_id`.

Source: `docs/updates/central-lifecycle.md:501` (replacing the pre-D-068 wording `"A later agreeing null\nlast-good permits the issued target to seed current, without establishing\nlast-good."`).

See [probe issuance evidence](probe-issuance.json) ([notes](probe-issuance.md)), [pre-result and D-068 authority evidence](pre-result.json) ([notes](pre-result.md)), [offline and inheritance evidence](inheritance.json) ([notes](inheritance.md)), [run and per-record provenance](run.json), [requests](requests.md), [how to run](RUN.md), and [follow-up 7 history](history/follow-up-7/README.md) (`#186`).
Full original §19 coverage, including the additional pre-existing enrolling snapshot member and physical arm64 execution, remains outside this focused follow-up.
