# Wave 4 security and operations acceptance

The live stack uses `FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=30000`.
The earlier invalid port assignment was an orchestrator error and its blocked
records are superseded. Current verdicts come from the corrected running stack.

```sh
make -f mk/at-secops.mk at-secops
```

The command refuses an occupied namespace, starts the existing dev stack with
fresh TEST ONLY state, runs live probes and an isolated logical restore, removes
its own Docker objects, and runs supporting suites with `-p 1 -parallel 2`.
Exit 2 means at least one acceptance case remains blocked. It never restores the
live stack or the shared PostgreSQL admin database. Go scratch stays on disk.

| Case | Evidence | Proven scope and remaining boundary |
|---|---|---|
| AT28 | [JSON](AT28.json), [page](AT28.md), [live checks](live.json) | Role refusal matrix, real OIDC sessions, two node identities, expired/revoked TLS credentials, replay, object ownership, input/CSRF/rate checks and log/audit scan |
| AT29 | [JSON](AT29.json), [page](AT29.md) | Live serving precondition observed; full flow/deadline drill blocked while generation remains zero. Supporting watchdog/time-anchor/grace/grant tests pass |
| AT30 | [JSON](AT30.json), [page](AT30.md), [logical restore](restore.json) | Real isolated logical backup/restore and replay from ACK 5 to 10. WAL/PITR, actual node archive and coordinated external reconciliation remain blocked |
| AT31 | [JSON](AT31.json), [page](AT31.md) | 20 authenticated members/60 synthetic authorized device rows, bounded concurrent read sample and ledger replay. Full [mixed/relay profile](load-profile.json) remains unexecuted |
| AT32 | [JSON](AT32.json), [page](AT32.md) | [13 public-statement mismatches](scope-review.json); source-independent contract/lifecycle checks. Physical source/platform comparison pending |

`live.json` contains measured HTTP status counts, latency percentiles, resource
snapshots, signing generation, node incarnation, image IDs and refusal results.
Every AT JSON also records build digests, commands, UTC times and expected/actual.
The API load sample is explicitly shorter and narrower than the fixed 300-second
mixed profile; it is not candidate-hardware capacity or a relay throughput claim.
SQL device rows are not connected NetBird peers. The retained replay file contains
original synthetic upload events; it is not a node SQLite archive.

[components.json](components.json) records all selected owner-maintained tests
and subtests. Missing, skipped or failed required tests cannot pass. The offline
`at-secops-test`, pinned-staticcheck `at-secops-lint`, and two-architecture
`at-secops-build` targets also run in CI. `at-secops-components` is supporting
verification only and writes blocked acceptance records; rerun full acceptance
to replace those records with a live snapshot.

[secret-scan.json](secret-scan.json) inventories captured component output,
all twelve container stdout/stderr streams, produced node/member log files and
all audit rows, using runtime secret canaries. No raw HTTP body, certificate,
credential, private path or matched secret is published. Private backups and
logs stay in `.agent/secops`. [Operator procedures](operator-runs.md) and
[requests](requests.md) specify the remaining work. Full-service RPO/RTO and
unmeasured policy/usage/relay timings remain null.
