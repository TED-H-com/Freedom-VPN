# Wave 4 security and operations acceptance

Follow-up 1 uses the existing dev stack with
`FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000`. D-050 replaces the original
projected SQL fixture with admin grants and native NetBird device admission.

```sh
make -f mk/at-secops.mk at-secops
```

The command refuses an occupied namespace, starts fresh TEST ONLY state, runs
live probes and a separate network-none logical restore, removes its own Docker
objects, then runs supporting suites with `-p 1 -parallel 2`. Exit 2 preserves
blocked acceptance. It never restores the live stack or shared admin database.
Backups and public result files from previous runs are removed before bring-up.

| Case | Evidence | Scope |
|---|---|---|
| AT28 | [record](AT28.json), [page](AT28.md), [live](live.json) | Actual sessions, native-device ownership, role refusals, two node certificates, expired/revoked TLS, replay, CSRF/ETag/limits and stream scan |
| AT29 | [record](AT29.json), [page](AT29.md) | Publication now completes; forwarding still requires a positive counted flow. Watchdog, recovery-config and admission-counter observations identify the current blocker |
| AT30 | [record](AT30.json), [page](AT30.md), [logical subset](restore.json) | Real isolated logical restore/replay. Physical WAL/PITR awaits `w4-ops-backups`; full-service RPO/RTO remain null |
| AT31 | [record](AT31.json), [page](AT31.md) | 20 finite API grants, one native device, concurrent login attempt, bounded read sample and synthetic ledger diagnostics. The 60-device mixed/relay profile is blocked |
| AT32 | [record](AT32.json), [review](scope-review.json) | DOC01–DOC13 resolved on main and re-reviewed; physical source/platform comparison still needs operator runs |

The fixture writes only configuration inputs with no HTTP create command:
`member_groups`, `plans`, `plan_revisions`, TEST ONLY empty release-support
`policy_definitions`/`policy_revisions`, and a 90-second `fleet_config` lease
ceiling. The separate PKI component probe retains isolated `nodes`/`node_incarnations`
identity fixtures and certificate revocation state to exercise authentication
without a business handler; these are not live-stack serving evidence.
The product creates all member statuses, entitlements, histories,
native identity mappings, devices and bindings. API grants are finite and
projection is polled with deadlines, including survival over two device sweeps.
The synthetic ledger reads an actual binding revision/interval; unpublished
node B correctly has no serving bindings in its desired state. Neither that
read nor accepted usage events prove that a packet was forwarded.

`live.json` records per-check expected/actual responses, policy generation,
incarnations, image IDs, UTC bounds, CPU/memory/IO snapshots, login refusals,
read percentiles and exact synthetic ledger totals. The read sample is a small
API diagnostic, not the fixed [300-second full profile](load-profile.json) or
candidate-hardware capacity. Its concurrent login burst is measured separately
from the already-authenticated read loop. No limits are tuned to improve results.
[Historical samples](capacity-history.json) retain earlier results and explicitly
invalidate the pre-D-050 SQL fixtures as acceptance evidence.

[Secret scan](secret-scan.json) covers captured component output, all twelve
container stdout/stderr streams, produced node/member log files and audit rows
using generated secret canaries. Raw credentials, response bodies, backups and
logs stay in `.agent/secops`. [Requests](requests.md) name the exact owners and
remaining prerequisites; [operator procedures](operator-runs.md) remain unrun.
Build/image digests and commands are included in every AT JSON. Supporting
component success never changes a blocked full-stack verdict to passed.

Standalone `at-secops-test`, `at-secops-lint` (pinned staticcheck) and
`at-secops-build` targets run in CI. `at-secops-components` writes supporting-only
blocked AT records; rerun full acceptance to replace them with a live snapshot.
