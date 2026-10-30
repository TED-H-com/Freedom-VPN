# Physical backup and PITR — AT30 groundwork

Lab PostgreSQL is built from `deploy/backup/postgres/Dockerfile`, pinned as the
run's image ID. It mounts `fvpn-<prefix>-pgbackrest-repo` at `/var/lib/pgbackrest`.
`FVPN_PG_ARCHIVE_TIMEOUT=10s` is TEST ONLY (production default: 300s).

The `backup` scenario creates an enrollment through the admin API, verifies
archiving and takes a physical full backup. It chooses PostgreSQL's UTC time
strictly after backup completion, creates another enrollment through the API,
forces a WAL switch and verifies archiving. It uses the shipped restore tool:

```sh
deploy/backup/backup.sh --container "fvpn-$FVPN_E2E_PREFIX-postgres" check
deploy/backup/backup.sh --container "fvpn-$FVPN_E2E_PREFIX-postgres" full
deploy/backup/backup.sh --container "fvpn-$FVPN_E2E_PREFIX-postgres" wal-switch
# Use the recorded UTC target and the primary container's immutable image ID.
deploy/backup/restore-pitr.sh --image "$PRIMARY_IMAGE_ID" \
  --repo-volume "fvpn-$FVPN_E2E_PREFIX-pgbackrest-repo" \
  --target-volume "fvpn-$FVPN_E2E_PREFIX-restore-pg" \
  --isolated-container "fvpn-$FVPN_E2E_PREFIX-restore-verify" \
  --primary-container "fvpn-$FVPN_E2E_PREFIX-postgres" \
  --target-time "$RESTORE_TARGET_UTC" --db-user fvpn --db-name fvpn
```

The restore uses `--network none`, zero published ports, fresh PGDATA and a
read-only repository. The primary has both enrollment rows; the restored copy
must retain only the pre-target row, finish recovery and have archiving off.
Each archive operation is capped at 90 seconds; backup and restore at 150
seconds each; the entire drill at six minutes. Evidence records actual timings,
backup label, target time and readback counts without enrollment tokens.
Restore containers and their target volume are removed even on failure.

See [latest backup record](../runs/2026-10-30/backup.md). This proves physical
backup/WAL/PITR mechanics. Ledger freeze, post-restore integrity checks, epoch
recovery and replay are the `at-secops` owner's remaining full AT30 work. The
shared `fvpn-pg` test database is never accessed by this drill.
