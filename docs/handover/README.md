# Wave 4 Hand-Over Documentation (`docs/handover/`)

This directory contains the production deployment, backup/restore, parameter calibration, operator, and bilingual member guides for Freedom VPN v1 (`§16`–`§22`, Work Package `WP12`). Lab and acceptance evidence across the repository is recorded as dated JSON and Markdown pairs under `docs/e2e/runs/<utc-date>/<id>.json` and `<id>.md` sharing a `run_id` and `source_commit` (`docs/e2e/README.md`).

## Deliverables Index

| Document | Scope & Audience | Spec References |
|---|---|---|
| [**Production Deployment Runbook (`deploy.md`)**](deploy.md) | Transitioning from `deploy/devstack` to production hosts: PostgreSQL 17 with WAL archiving and role separation, `fvpn-control` (`:8443` / `:9443`) and `fvpn-worker`, OIDC with Admin MFA (`issuer.Canonical` literal-slash rule), NetBird CE (`0.82.0`), AdGuard Home (`v0.107.59`), immutable release origin, all 5 domain settings files (`members_config_file`, `nodes_config_file`, `dns_config_file`, `policies_config_file`, `usage_config_file`), key ceremonies (`node-ca`, `release`, `ledger-epoch`), node host prerequisites (`nft`, `tc`, `ip`, `conntrack`), and first node enrollment. Includes complete `TEST ONLY` replacement inventory. | `§3`, `§5`–`§8`, `§12`–`§13`, `§15`–`§17`, `§21` |
| [**Backup & Restore Drill Record (`restore-drill.md`)**](restore-drill.md) | Repeatable `AT30` point-in-time restore checklist, Wave 4 measured RPO and RTO evidence (`2026-10-29`), usage ledger freeze & `ledger_epoch` rotation (`fvpn-usage-admin`), node checkpoint signature verification & replay (`replay_unavailable` zero-byte tombstones), and post-restore reconciliation for payments (`billing.reconcile`) and NetBird (`members.reconcile` and node state reconciliation). | `§9.3`, `§10.8`, `§17.2`–`§17.4`, `§19 AT30` |
| [**Parameter Calibration Guide (`calibration.md`)**](calibration.md) | Master calibration table for prices/plans, device limits, fair-use thresholds/tiers (`shadow` vs. `enforce` and `minimum_rate`), serving leases (`max_offline_seconds`), PKI/token lifetimes, updater synthetic `probe_peer` bounds, DNS list sources, cadence & node resolver credentials (`D-054`, `D-058d`, `D-061`), retention periods, per-node capacity ceilings (`host_reserve_bps`, `down_bps`, `up_bps`), and per-class HTTP rate limiters (`D-060`) with location, default status (`none`, `test only`, `proposed`), spec constraint, and decision owner (`Ted` or `core team`). | `§4`, `§5`–`§13`, `§16.1`–`§16.5`, `§20`–`§21` |
| [**Operator Guide (`operator-guide.md`)**](operator-guide.md) | Daily and weekly operator checklists, reading asynchronous `Operation` receipts, `audit_events`, and current OpenAPI contract status (`44` operations), enrolling/reinstalling/classifying/publishing/revoking/retiring nodes (`D-055`, `D-057`, `D-058`, `D-061`, zero-binding publication, and `data_path_probe` reasons), running `all_nodes` update batches, `D-053` / `D-064` health attestation, update observation authority (`unconfirmed`), triaging `needs_attention` and rolled-back targets, threat-feed and resolver-document troubleshooting (`D-054`, `D-056`), freezing/recovering/adjusting the usage ledger, suspending members, per-class rate limiters (`D-060`), and rotating signing keys and secrets. | `§5`–`§16`, `§17`–`§18.3` |
| [**Member Guide (`member-guide.md`)**](member-guide.md) | Bilingual (`zh-TW` and `en`) guide covering candidate client versions (`needs operator run`) and automatic central device authorization (`D-052` shared-issuer public-subject discovery) for Windows (`0.82.0`), macOS (`0.82.0`), iOS (`0.4.1 build 2` TestFlight candidate), and Android (`0.6.2`), manual verification scripts (`AT01`, `AT09`, `AT21`, `AT29`), device limits & self-service revocation (`AT02`, `AT04`), usage (`IPv6Block` mode) & fair-use display (`AT17`–`AT19`), DNS behaviour & explicit technical limits (DoT/DoQ vs. DoH), and support contact flow. | `§7.3`, `§8.1`–`§8.3`, `§9.4`, `§10`–`§12`, `§16.3`, `§16.5`, `§18.4`–`§18.5`, `§19` |
| [**Adversarial Review Archive (`../reviews/README.md`)**](../reviews/README.md) | Self-contained archive of the eight Wave 4 adversarial review reports (`r3-d055-review` through `r10-d063-review`), recording the review date, model, scope, invariants that held, findings with closing PRs, and proof tests adopted onto `main`. | `D-052`–`D-064`, `§5`–`§17` |

## Self-Verification (`tests/handover` and `mk/handover.mk`)

To prevent documentation drift, the handover, operations, and review-archive Markdown files (`docs/handover/README.md`, `docs/handover/deploy.md`, `docs/handover/calibration.md`, `docs/handover/operator-guide.md`, `docs/handover/member-guide.md`, `docs/ops/README.md`, `docs/reviews/*.md`, and `README.md`) are mechanically verified by `tests/handover` and `mk/handover.mk`:

1. **Plain Lint & Structural Verification (offline, no database or Docker required)**:
   ```sh
   make -f mk/handover.mk handover-lint
   go test -v ./tests/handover/...
   ```
   Asserts that across all owned handover, ops, and review Markdown files:
   - The member guide states the current `contracts/openapi.yaml` `info.version`, and stated OpenAPI versions and operation counts match the parsed contract.
   - Every relative Markdown link resolves to an existing file in the repository.
   - Every repository path in backticks (`deploy/…`, `mk/…`, `docs/…`, `contracts/…`, `db/migrations/…`, `apps/…`, `node/…`, `internal/…`, `pkg/…`, `packaging/…`, `tests/…`) exists.
   - Every `make -f mk/<file>.mk <target>` command names an existing target in `mk/<file>.mk`.
   - Every `fvpn-*` command names a built binary under `apps/*/cmd/*`, `node/*/cmd/*`, `internal/*/cmd/*`, `packaging/*/cmd/*`, or `tests/e2e/cmd/*`.
   - Every file in `docs/reviews/` contains the review date, reviewing model, and a findings table with the four columns `Severity`, `Finding`, `Status`, and `Closing PR` (`TestHandover_ReviewsArchive`).
   - The migration inventory is referenced as `db/migrations/*.sql` (one goose version per file) with no hardcoded literal migration count or last-migration ID, and `deploy.md` shows the abbreviated `fvpn-migrate status` example (`00001 applied` … `<last file> applied`).
   - Any SQL block that cannot be executed as a normal tenant query (specifically cluster-level role/database creation and `GRANT` statements in `deploy.md`) is explicitly labeled with `-- [mechanical-check: cluster-role-ddl]` and reported by the test rather than skipped silently.

2. **PostgreSQL Schema & SQL Execution Verification (`integration` tag)**:
   ```sh
   go test -v -tags integration ./tests/handover/...
   make -f mk/handover.mk handover-verify
   ```
   Creates a freshly migrated, isolated PostgreSQL 17 database via `internal/db/dbtest` (`D-039`: schema only, using `$FVPN_TEST_DATABASE_URL`), asserts that the applied migration set in `goose_db_version` equals the files in `db/migrations/*.sql`, and executes every domain SQL statement extracted from the owned handover Markdown files so any table, column, function, or type drift fails immediately.
