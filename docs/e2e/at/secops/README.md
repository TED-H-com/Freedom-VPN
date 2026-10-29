# Wave 4 security and operations acceptance

AT28–AT32 are **blocked**, not passed. The assigned stack instance is
`FVPN_E2E_PREFIX=w4-secops FVPN_E2E_PORT_BASE=23000`; the checked-in
`parseLayout` accepts only `18000 + 6000*n`. No alternative port or copied stack
is used. Read [requests](requests.md) for the precise unblock work.

Run from the repository root:

```sh
make -f mk/at-secops.mk at-secops
```

Exit 2 means acceptance is blocked. The command attempts only its assigned
namespace, refuses an occupied namespace, cleans up its own attempted bring-up,
then runs the supporting suites with `-p 1 -parallel 2`. All Go scratch is on
worktree disk and removed on exit. PostgreSQL tests use `internal/db/dbtest`;
the shared admin database is never migrated, restored or reconfigured.

`at-secops-test` runs the offline boundary/watchdog/schema and evidence-parser
checks. `at-secops-lint` uses the root's pinned staticcheck v0.8.2, including
`e2e,integration` tags. `at-secops-build` compiles both Linux architectures.
`at-secops-components` runs database-backed component suites and still writes
blocked AT records; a zero component exit is never an acceptance pass.

| Case | Record | Scope of supporting evidence |
|---|---|---|
| AT28 | [JSON](AT28.json), [page](AT28.md) | All 39 protected contract operations; sessions, roles, mTLS, enrollment, device/operation ownership, CSRF, size, limiter, idempotency, audit and canary checks |
| AT29 | [JSON](AT29.json), [page](AT29.md) | Lease math/grant/grace, transaction locks, independent watchdog decisions, elapsed-time/boot/replay tests, rendered kernel timeouts |
| AT30 | [JSON](AT30.json), [page](AT30.md) | Isolated database recovery model, signed epochs/checkpoints, durable SQLite archives, replay, fake-provider and peer reconciliation |
| AT31 | [JSON](AT31.json), [page](AT31.md) | [Unexecuted 20/60 load profile](load-profile.json); no capacity measurements |
| AT32 | [JSON](AT32.json), [page](AT32.md) | Contract/lifecycle checks and [13 documentation findings](scope-review.json); three-source physical comparison pending |

[components.json](components.json) records exact commands, every selected test
and subtest, UTC time, exit codes and scan summaries. Required tests that vanish,
skip, fail or cannot compile cannot be marked passed. Digests identify the
acceptance **test binaries** and source revision; absent deployed image/policy/
incarnation metadata is explicitly null. These are not invented deployment IDs.

[secret-scan.json](secret-scan.json) inventories every captured component command
stream. Application-level canary assertions also run inside named tests; this is
not a complete scan of a running stack. Raw diagnostic logs remain private in
`.agent/secops`, with mode 0600. Published output contains no raw HTTP bodies,
certificates, tokens, DSNs, private paths or matched secret values.

The evidence is a current-run snapshot and is replaced on rerun. Git history
retains previous snapshots. [Operator runs](operator-runs.md) describe the exact
remaining procedures and expected evidence. A component restore model is not a
DB/WAL restore, a fake enforcer is not a stopped flow, and cross-compilation is
not physical arm64 execution (§19.1, §19.2).

[validation.json](validation.json) records final root/area checks, the reverified
probe revision and its two architecture digests. The earlier acceptance snapshot
retains its own exact source revision and build digests.
