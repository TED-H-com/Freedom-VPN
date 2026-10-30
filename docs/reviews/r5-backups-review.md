# Adversarial Review `r5-backups-review`: PostgreSQL Backups and `AT30` PITR Drill (`D-025`)

- **Branch:** `r5-backups-review` (commit `bc79093`)
- **Date:** 2026-10-29 (`2026-10-29 22:45:59 -0400`)
- **Reviewing model:** Not stated in `.agent/REVIEW.md` or commit `bc79093` (commit had no `Agent:` trailer)
- **Decision / scope reviewed:** Central PostgreSQL backups, pgBackRest WAL archiving, `restore-pitr.sh`, `verify.sh`, systemd backup units, and the `AT30` point-in-time recovery drill (`§15.5`, `§17.2`, `§19.2`, `SECOPS-05`, `AT30`; `deploy/backup/*`, `tests/backup/*`, `docs/ops/backups.md`, `docs/handover/restore-drill.md`)
- **Adopted proof files on `main`:**
  - [`deploy/backup/test/review_proof.sh`](../../deploy/backup/test/review_proof.sh) (adopted in `#100`, shellcheck-cleaned in `#104`, migration-inventory predicate updated in `#111`)
  - Supporting Go safety test [`tests/backup/restore_safety_test.go`](../../tests/backup/restore_safety_test.go) (added in `#100`)

## Findings

| Severity | Finding | Status | Closing PR |
|---|---|---|---|
| High | 1. The PITR drill in `tests/backup/drill_integration_test.go` passes when zero WAL segments generated after the base backup stop are replayed because marker `A` is written before `backup.sh full` and no post-backup marker `M` is required before the recovery target. | closed | #100 |
| High | 2. `deploy/backup/restore-pitr.sh` checks only whether the target directory is a `/var/lib/postgresql/data` mount point and wipes any unmounted non-empty directory or subdirectory under a live `PGDATA` with `find ... -exec rm -rf {} +`. | closed | #100 |
| High | 3. `tests/backup/drill_integration_test.go` destroys fixed-name `fvpn-w4-ops-backups-*` containers and volumes across concurrent runs and unconditionally rewrites tracked `docs/ops/backups/drill-<date>.json` and `docs/ops/backups/drill-<date>.md` evidence files on every run. | closed | #94, #100 |
| Medium | 4. `deploy/backup/verify.sh` never connects to the primary database and reports `latest_archive_age_seconds` from `backup.stop` instead of `pg_stat_archiver` or WAL segment modification time. | closed | #100 |
| Medium | 5. `deploy/backup/backup.sh` passes `-e PGBACKREST_REPO1_CIPHER_PASS=...` on the `docker exec` command line, exposing the repository encryption passphrase in host process listings (`ps` / `/proc/<pid>/cmdline`). | closed | #100 |
| Medium | 6. Full and differential backup runs in `deploy/backup/backup.sh`, `deploy/backup/systemd/fvpn-backup-full.service`, and `deploy/backup/systemd/fvpn-backup-diff.service` do not invoke `verify.sh` or `pgbackrest verify`. | closed | #100 |
| Medium | 7. Post-restore verification SQL in `docs/handover/restore-drill.md` and `docs/ops/backups.md` fails against the current PostgreSQL 17 schema on uncast enum/UUID predicates and is never executed by `tests/backup/drill_integration_test.go`. | closed | #100, #107, #111 |
| Medium | 8. The drill's hardcoded migration count and latest version in `tests/backup/drill_integration_test.go`, `docs/ops/backups.md`, and `docs/handover/restore-drill.md` drift from `db/migrations` instead of deriving the migration inventory dynamically from `db/migrations/*.sql`. | closed | #100, #107, #111 |
| Medium | 9. `deploy/backup/systemd/fvpn-backup-verify.service` sets `PrivateNetwork=true` and omits `EnvironmentFile=-/etc/fvpn/backup.env`, preventing `verify.sh` from reaching PostgreSQL or the S3 repository. | closed | #100 |
| Low | 10. `deploy/backup/postgres/fvpn-postgres-entrypoint.sh` copies `/var/lib/pgbackrest-seed/.` with `/bin/cp -a`, which aborts when the seed directory contains a Unix socket or unreadable special file. | closed | #100 |
| Low | 11. `waitForPostgres` in `tests/backup/drill_integration_test.go` matches the first `"database system is ready to accept connections"` line emitted by the temporary init-phase server before `docker-entrypoint.sh` shuts it down. | closed | #94, #100 |

## What Held

- **`restore-pitr.sh` argument validation:** Empty `TARGET_DATA_DIR`, relative paths, `/`, `/var`, `/var/lib`, `/var/lib/postgresql`, `/var/lib/postgresql/data`, and paths containing `..` are rejected with exit code 2 before `find` runs. `TARGET_TIME` values that are not UTC RFC 3339 timestamps (`YYYY-MM-DDTHH:MM:SS(.ffffff)?Z`) and `TARGET_ACTION` values outside `promote|pause` are rejected.
- **`restore-pitr.sh` mount-point and postmaster.pid checks:** When `TARGET_DATA_DIR` is a mount point whose target is `/var/lib/postgresql/data`, `findmnt -T` catches it and aborts. When `TARGET_DATA_DIR/postmaster.pid` exists directly in the target directory, the script aborts before `find ... -exec rm -rf`.
- **pgBackRest encryption and retention enforcement:** `deploy/backup/pgbackrest.conf.example` configures `repo1-cipher-type=aes-256-cbc` and `repo1-retention-full=14` / `repo1-retention-diff=7`. Restoring from the encrypted repository with the wrong passphrase fails with pgBackRest `CryptoError` (`unable to flush ... [33554536] bad decrypt`) and leaves no usable cluster.
- **Corrupted WAL detection by `pgbackrest verify`:** Overwriting bytes inside an archived `.gz` WAL segment in `repo1/archive/fvpn/` causes `pgbackrest --stanza=fvpn verify` to exit non-zero, and `verify.sh` propagates that non-zero exit.
- **Unwritable archive target surfaces in `pg_stat_archiver`:** Making `repo1/archive` read-only causes `pg_switch_wal()` archiving to fail (`archive-push` error), incrementing `pg_stat_archiver.failed_count` on the primary.
- **Application tables and sequences after PITR:** When post-backup WAL is archived before recovery, `audit_events` (`hash`, `prev_hash`), `usage_events`, `usage_checkpoints`, `desired_state_envelopes`, `enrollment_tokens`, and `goose_db_version` restore cleanly to the target point.

## Detailed Review Notes (Archived from `.agent/REVIEW.md`)

### 1. [High] The PITR drill passes when no WAL generated after the base backup is replayed
- **Where:** `tests/backup/drill_integration_test.go:121-162` (`TestAT30_BackupPITRDrill`).
- **Scenario:** The drill wrote marker `A` before `backup.sh full`, recorded `targetTime = queryUTC(ctx, primaryDSN, "SELECT clock_timestamp()")` after the backup without writing or committing any row between the backup stop and `targetTime`, and then wrote marker `B` after `targetTime`. If every WAL segment archived after the base backup's stop LSN was deleted before restore, the drill still passed (`A=1, B=0`).
- **Proof:** `proof_1_drill_passes_without_post_backup_wal` in `deploy/backup/test/review_proof.sh`.
- **Fix (#100):** Inserted post-backup marker `M` after `backup.sh full`, took `targetTime` between `M` and `B`, switched WAL, and required `A=1, M=1, B=0` after restore.

### 2. [High] `restore-pitr.sh` wipes an existing directory when it is not a mount point and accepts paths under a live `PGDATA`
- **Where:** `deploy/backup/restore-pitr.sh:74-106`.
- **Scenario:** `restore-pitr.sh` only refused `TARGET_DATA_DIR` if `findmnt` reported `/var/lib/postgresql/data` as the mount point or if `postmaster.pid` was at the top of `TARGET_DATA_DIR`. Passing an unmounted directory containing files or a subdirectory of a running `PGDATA` (`${LIVE_PGDATA}/base`) wiped its contents with `find "${TARGET_DATA_DIR}" -mindepth 1 -maxdepth 1 -exec rm -rf {} +`.
- **Proof:** `proof_2_restore_wipes_unmounted_and_subdir_of_live_pgdata` in `deploy/backup/test/review_proof.sh`.
- **Fix (#100):** Refused any non-empty target directory (`restore-pitr.sh` requires an empty or newly created directory) and refused any ancestor or descendant of a directory containing `postmaster.pid` (`tests/backup/restore_safety_test.go`).

### 3. [High] The drill test destroys fixed-name containers and volumes and rewrites tracked evidence files on every run
- **Where:** `tests/backup/drill_integration_test.go:37-44, 62-63, 217-227, 297-315`, `mk/backup.mk:29-39`.
- **Scenario:** `TestAT30_BackupPITRDrill` used fixed Docker names (`fvpn-w4-ops-backups-primary`, etc.) and unconditionally overwrote `docs/ops/backups/drill-<date>.json` and `docs/ops/backups/drill-<date>.md` on every run.
- **Proof:** `tests/backup/drill_integration_test.go`.
- **Fix (#94, #100):** Scoped all Docker objects to `FVPN_BACKUP_DRILL_PREFIX` (or a random per-run suffix) and gated tracked evidence writes behind `FVPN_BACKUP_DRILL_RECORD=1`.

### 4–11. [Medium / Low] `verify.sh` primary check and archive age, cipher pass on argv, `verify.sh` after full/diff backups, runbook SQL and dynamic migration inventory, `fvpn-backup-verify.service` sandbox, `repo-copy.py`, and `waitForPostgresReady`
- **Proofs:** `proof_4_verify_ignores_dead_primary_and_stale_wal`, `proof_5_cipher_pass_exposed_in_process_list`, `proof_6_backup_succeeds_without_verify`, `proof_7_handover_sql_fails_against_schema`, `proof_8_migration_count_drift`, `proof_9_systemd_verify_unit_broken`, `proof_10_entrypoint_cp_fails_on_socket` in `deploy/backup/test/review_proof.sh`.
- **Fix (#94, #100, #104, #107, #111):** `verify.sh` queries `deploy/backup/archive-state.sql` on the primary and fails without a reachable primary; `backup.sh` passes `PGBACKREST_REPO1_CIPHER_PASS` via environment (`-e PGBACKREST_REPO1_CIPHER_PASS`) and runs `verify.sh` after full/diff backups; `fvpn-backup-verify.service` loads `/etc/fvpn/backup.env` without `PrivateNetwork=true`; `deploy/backup/repo-copy.py` copies regular files/dirs/symlinks while skipping sockets/FIFOs/device nodes; runbook SQL blocks and dynamic migration inventory are verified by `tests/backup/drill_integration_test.go` and `tests/handover/handover_test.go`.
