# Central PostgreSQL 17 Backups, Point-in-Time Recovery (PITR) & Post-Restore Reconciliation Runbook (`docs/ops/backups.md`)

This runbook specifies the production and lab backup architecture, continuous WAL archiving configuration, encryption and key separation rules, monitoring signals, point-in-time restore (PITR) execution, and the ordered post-restore reconciliation protocol for Freedom VPN (`§3.1`, `§15.4`, `§16.2`, `§16.4`, `§17.1`–`§17.5`, `§19 AT30`, `SECOPS-05`).

---

## 1. Recovery Objectives & Pinned Tooling Baseline (`§16.5`, `§17.2`)

| Parameter / Component | Specification | Notes & Spec Reference |
|---|---|---|
| **Database Engine** | PostgreSQL 17 (`postgres@sha256:31ba9c94a03ada9988f2bb493a22a75eef3a185884533e5cdbcd498ea835b08f`) | Drop-in image built from [`deploy/backup/postgres/Dockerfile`](../../deploy/backup/postgres/Dockerfile) (`§3.1`, `§15.4`). |
| **Backup Engine** | **pgBackRest `2.60.0`** (`2.60.0-1.pgdg13+1`) | Physical full + differential base backups and continuous WAL archiving (`§17.2`). Logical `pg_dump` is an auxiliary export only, never the primary recovery path. |
| **Central Database RPO Target** | **`<= 15 minutes`** (`900s`) | Requires healthy archiving and measured repository WAL age; configured with `archive_mode = on` and `archive_timeout = 300s` (`§16.5`, `§17.2`). |
| **Usage Ledger Effective RPO** | **`0` bytes lost / `0` double-counted only when the restore point is inside the available node archive window** | Exit nodes keep ACKed payloads at least `72 hours` after local ACK (`DefaultArchiveRetain = 72 * time.Hour`; no node configuration override). Unacknowledged payloads are retained and resent, with new member forwarding blocked at oldest-event age `72 hours` or decimal `256 MB`, whichever comes first. The replay archive has its own `1000000000`-byte engineering budget, not a spec number. Outside the retained window, replay can be unavailable; shortening it requires updated parameters, recovery documents and risk notes ([usage storage](../node/usage-store.md#ack), `§10.8`, `§17.3`, `AT30`). |
| **Central Service RTO Target** | **`<= 4 hours`** (`§16.5`/`§17.2`; unmeasured by this local drill) | Covers physical DB/WAL restore, migration check, checklist inspection, ledger freeze/epoch rotation, replay, and provider/NetBird reconciliation. |
| **Signing Vault (`/var/lib/fvpn/staging-authority/vault`) RPO** | **Target: `0` (Zero Rollback)** | Stored outside PostgreSQL and never included in database snapshots (`§16.2`, `§17.2`). Carried gap: production custody and independent anti-rollback; FileProvider cannot detect whole-directory rollback. See [rotation/restore limits](../pki/rotation.md). |

---

## 2. Production & Lab Configuration (`§3.1`, `§15.4`, `§17.2`)

### 2.1 PostgreSQL 17 WAL Archiving Fragment (`deploy/backup/postgres/fvpn-archiving.conf`)

Installed at `/etc/postgresql/conf.d/50-fvpn-archiving.conf` (included via `include_dir = '/etc/postgresql/conf.d'` in `postgresql.conf`):

Carried gap: the PostgreSQL server/archiver must receive the repository passphrase and object-store credentials for encrypted `archive-push`. Only the three backup services load `/etc/fvpn/secrets/pgbackrest-repo.env`; their `EnvironmentFile` does not configure PostgreSQL. The repository has no production procedure to deliver this environment to PostgreSQL.
- **Lab / CI Override (`TEST ONLY`)**: Containerized lab and drill environments may set `FVPN_PG_ARCHIVE_TIMEOUT=10s` on `deploy/backup/postgres/Dockerfile` to bound idle archive lag during automated tests.
- **Isolated Verification Containers**: When `restore-pitr.sh` launches an isolated verification container (`--network none`), it sets `FVPN_PG_ISOLATED_RESTORE=1` and mounts the backup repository read-only (`:ro`). The entrypoint writes `archive_mode = off` inside the verification container so promotion to a new timeline never pushes WAL segments back to the primary repository.

Carried gap: provision the repository and stanza, including an initial recoverable base backup, before relying on this archiver configuration. The first-host provisioning procedure remains absent; §3.1 carries the same prerequisite for scheduled backups.

```ini
wal_level = replica
archive_mode = on
archive_timeout = 300s
archive_command = 'pgbackrest --stanza=fvpn archive-push %p'
max_wal_senders = 5
```

- **Production**: `archive_timeout = 300s` (5 minutes) forces segment switches when WAL activity occurs; repository availability and archive-age monitoring are still required for the `<= 15 minutes` RPO target (`§17.2`).

### 2.2 pgBackRest Repository Configuration (`deploy/backup/pgbackrest.conf.example`)

See [`deploy/backup/pgbackrest.conf.example`](../../deploy/backup/pgbackrest.conf.example) for the complete reference file:
- **Production Profile (`repo1-type=s3`)**: Archives to an off-host S3-compatible object store (`repo1-s3-verify-tls=y`, `repo1-cipher-type=aes-256-cbc`, `compress-type=zst`).
- **Lab Profile (`repo1-type=posix`)**: Archives to a mounted Docker volume at `/var/lib/pgbackrest` for local devstack and CI PITR drills.

Production installs Profile A from `deploy/backup/pgbackrest.conf.example` as `/etc/pgbackrest/pgbackrest.conf` (0640, root:postgres). The image-generated POSIX config uses full-count=7 and diff-count=7 and is only the lab default; it is not the production retention policy.

### 2.3 Least-Privilege Database Roles (`docs/handover/deploy.md §3.2`, `docs/db/README.md`)

- `fvpn_backup`: Logical read-only role for backup verification and `pg_dump` exports (`GRANT pg_read_all_data TO fvpn_backup`); holds no mutation privileges on application tables and is not the physical backup account. Physical pgBackRest backups run as the local `postgres` OS/cluster account over the Unix domain socket (`/var/run/postgresql`).
- `fvpn_migrator`: Schema-owner operator credential for migration status/apply and the checked restored candidate's checklist in Steps 3–4; it is not a control/worker runtime credential.
- `fvpn_recovery`: Controlled recovery role used by `fvpn-usage-admin` (`internal/usage/cmd/fvpn-usage-admin`) to freeze usage ingestion, rotate `ledger_epoch` (`usage_recovery_records`, `usage_ledger_epochs`, `usage_checkpoints`), and complete recovery (`§17.3`).

---

## 3. Schedules & Retention Policy (`§16.4`, `§17.2`)

### 3.1 Production Schedule (`deploy/backup/systemd/`)

Install the systemd service and timer units from [`deploy/backup/systemd/`](../../deploy/backup/systemd/) onto the central database host:

| Unit | Schedule (`UTC`) | Action |
|---|---|---|
| `fvpn-backup-full.timer` / `.service` | Weekly: `Sun *-*-* 01:00:00 UTC` | Runs `deploy/backup/backup.sh --stanza fvpn full` to create a physical full base backup. |
| `fvpn-backup-diff.timer` / `.service` | Daily: `Mon..Sat *-*-* 01:00:00 UTC` | Runs `deploy/backup/backup.sh --stanza fvpn diff` to create a differential backup against the latest full backup. |
| Continuous WAL Archiving | Continuous (`archive_timeout = 300s`) | PostgreSQL archiver invokes `pgbackrest --stanza=fvpn archive-push %p` on every completed or timed-out WAL segment. |
| `fvpn-backup-verify.timer` / `.service` | Daily: `*-*-* 03:00:00 UTC` | Runs `deploy/backup/verify.sh` to restore the latest backup into an isolated `--network none` container, verify `fvpn-migrate status`, run `post-restore-checklist.sql`, and emit RPO/RTO metrics. |

Enable the timers in production:

Carried gap: first-host repository/stanza creation and the initial base-backup procedure are not supplied here. This includes provisioning the units' required writable directories `/var/log/pgbackrest`, `/var/spool/pgbackrest` and `/var/lib/pgbackrest` before enabling the timers. The configuration and timers alone do not establish a recoverable repository. See the [carried list](../handover/follow-up-51.md#existing-procedures-linked-and-carried-gaps).
Before enabling these timers, provide the restricted `/etc/fvpn/secrets/pgbackrest-repo.env` file described in [§4](#4-encryption--key-separation-contract-162-172). All three services load it; the verify service requires a nonempty repository passphrase.

```bash
install -d -m 0755 /usr/local/lib/fvpn/backup
install -m 0755 deploy/backup/backup.sh deploy/backup/restore-pitr.sh deploy/backup/verify.sh deploy/backup/repo-copy.py /usr/local/lib/fvpn/backup/
install -m 0644 deploy/backup/archive-state.sql deploy/backup/post-restore-checklist.sql /usr/local/lib/fvpn/backup/
cp deploy/backup/systemd/fvpn-backup-*.service deploy/backup/systemd/fvpn-backup-*.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now fvpn-backup-full.timer fvpn-backup-diff.timer fvpn-backup-verify.timer
```

The host needs Python 3, pgBackRest 2.60.0, PostgreSQL client tools, Docker, the built backup image and a linux/amd64 `/usr/local/bin/fvpn-migrate` for the verify unit's `--migrate-bin`. Without `--migrate-bin`, `verify.sh` builds a linux/amd64 migrator. The verify unit requires the secrets file and a nonempty `PGBACKREST_REPO1_CIPHER_PASS`. `--host-repo --primary-host --db-user postgres` reads primary archive state via host psql and copies repo 1 using the same host pgBackRest configuration/environment as `backup.sh`. Raw encrypted objects are staged under `/var/lib/fvpn/backup-verify` on disk, preserving repository modification times, then checked and restored in a network-none container. Allow space for the retained repository plus the restored database. Missing/changed objects or expiry during copying fail verification; rerun with a fresh copy. Cleanup removes only that run's scratch. This full-copy path can be expensive; measure it on the production repository before adopting an RTO. Repository `PGBACKREST_REPO1_*` environment variables are forwarded by name; the explicit staged mount selects POSIX path/type for recovery.

### 3.2 Retention & Post-Restore Data Minimization (`§16.4`, `§17.2`)

Carried gap: authorized audit-event and usage-rollup purge scheduling and coordination of JSON/database retention are not implemented as an operator procedure; service-log retention is also absent. See [calibration retention](../handover/calibration.md#16-data-retention-periods-105-154-164-173-21) before treating the purge references below as completed setup.

- **Backup Retention (`pgbackrest.conf`)**:
  - `repo1-retention-full-type=time`
  - `repo1-retention-full=35` (35-day minimum recovery window; an older full is retained until another full covers the window).
  - `repo1-retention-diff=14` (retains a count of 14 differential backups).
  - Leave `repo1-retention-archive` unset: WAL follows the retained fulls. Time retention keeps the only full and can exceed 35 days to preserve a recoverable base. See [pgBackRest retention semantics](https://pgbackrest.org/configuration.html#section-repository/option-repo-retention-full-type).
- **Re-applying Retention & Anonymization After Restore (`§16.4`)**:
  Restoring an earlier backup can re-introduce raw usage payloads, rollups, or audit rows whose retention cutoff has already passed, or personal metadata (`email`, `display_name`) for accounts closed during the recovery window. Before returning the restored cluster to normal service, run the authorized retention purge (`SELECT purge_retained(...)` as `fvpn_retention` after projection reconciliation completes) and re-apply account closure/anonymization actions recorded after the restore point.

---

## 4. Encryption & Key Separation Contract (`§16.2`, `§17.2`)

Carried gap: the repository environment must also reach PostgreSQL archiving and native restore processes, as noted in §§2.1 and 6.2; creating the secret file alone only supplies the backup services that explicitly load it.

1. **Repository Encryption at Rest**:
   All pgBackRest base backups and WAL archives are encrypted client-side using `repo1-cipher-type=aes-256-cbc` before leaving the database host.
2. **Repository Key Separation**:
   The pgBackRest encryption passphrase (`PGBACKREST_REPO1_CIPHER_PASS`) and object-store credentials (`PGBACKREST_REPO1_S3_KEY`, `PGBACKREST_REPO1_S3_KEY_SECRET`) are stored in `/etc/fvpn/secrets/pgbackrest-repo.env` (owned by `root:postgres`, mode `0600`), **outside PostgreSQL** and **outside the central signing vault (`/var/lib/fvpn/staging-authority/vault`)**. Escrowed recovery copies of `pgbackrest-repo.env` are kept in the operator hardware/offline secret store separate from the S3 bucket.
3. **Central Signing Vault Never Inside Database Backups (`§16.2`, `§17.2`)**:
   The central signing vault (`/var/lib/fvpn/staging-authority/vault`) holds the active and historical keys and revocation state (`state.json`) for `node-ca`, `bootstrap-secret`, `desired-state`, `lease`, `recovery`, and `ledger-epoch`. It is **never** stored in PostgreSQL or backed up inside pgBackRest snapshots. Consequently, restoring PostgreSQL to an earlier timestamp never rolls back a key rotation or un-revokes a compromised signing key (`§16.2`).

---

## 5. Monitoring & Alerting Signals (`§16.5`, `§17.2`)

Operators and monitoring collectors watch these signals to assess the `<= 15 minutes` RPO target:

### 5.1 WAL Archive Lag & Failed Archive Pushes (`pg_stat_archiver`)

Run against the primary database every 60 seconds:
```sql
SELECT
    archived_count,
    last_archived_wal,
    last_archived_time,
    failed_count,
    last_failed_wal,
    last_failed_time,
    EXTRACT(EPOCH FROM (clock_timestamp() - last_archived_time))::bigint AS archive_lag_seconds
FROM pg_stat_archiver;
```
- **Unknown state**: NULL `last_archived_time`, disabled archiving, missing repository WAL, or an inaccessible repository is a failure, never zero lag.
- **Warning threshold**: `archive_lag_seconds > 360` (6 minutes, i.e., `archive_timeout = 300s` + 60s grace) or `failed_count` increasing.
- **Critical threshold (RPO breach risk)**: `archive_lag_seconds > 600` (10 minutes; approaches the 15-minute RPO bound).

### 5.2 Last Successful Base Backup Age (`pgbackrest info`)

Carried gap: manual `backup.sh info` and `verify.sh` invocations also need the repository environment inherited by the calling process; only the installed backup services load the environment file automatically. The production process-environment procedure remains absent.

Inspect the repository metadata JSON via `deploy/backup/backup.sh info`:
```bash
deploy/backup/backup.sh --stanza fvpn info
```
Monitor inside the JSON output:
- `.[0].status.code`: Must equal `0` (`"ok"`). Any non-zero code indicates a missing stanza or broken archive stream.
- `.[0].backup[-1].timestamp.stop`: Unix epoch timestamp of the most recent successful backup. Alert if `now - stop > 93600` (26 hours, indicating a missed daily backup).
- Separately select `backup[] | select(.type == "full")` and alert if the newest full is absent or older than 8 days (`691200s`, weekly schedule plus grace). A fresh differential must not silence this full-backup alert.
- `.[0].archive[-1].max`: Latest archived WAL segment in the repository.

### 5.3 Log Lines Watched by Operators

- **PostgreSQL server log (`/var/log/postgresql/` or container stderr)**:
  - Failure pattern: `LOG:  archive command failed with exit code` or `DETAIL:  The failed archive command was: pgbackrest --stanza=fvpn archive-push`
- **pgBackRest log (`/var/log/pgbackrest/fvpn-*.log`) & systemd journal (`journalctl -u fvpn-backup-*`)**:
  - Success pattern: `INFO: archive-push command end: completed successfully` / `INFO: backup command end: completed successfully`
  - Failure pattern: `ERROR: [` (followed by pgBackRest numeric error code `[025]`–`[125]`).
- **Daily Verification Service (`journalctl -u fvpn-backup-verify.service`)**:
  - Success pattern: `status=PASS` with `migrations_pending=0` and `archiver_lag_ms`, `repository_archive_lag_ms` and `local_restore_verify_ms`.

---

## 6. Point-in-Time Restore (PITR) Procedure (`deploy/backup/restore-pitr.sh`)

`deploy/backup/restore-pitr.sh` restores a backup + WAL stream to an exact UTC timestamp (`--target-time`), WAL LSN (`--target-lsn`), or the end of the archived WAL stream (`--latest`).

> **Safety Contract**: `restore-pitr.sh` **never** restores over a live primary. It refuses execution if `--isolated-container` matches `--primary-container`, if the target container or volume is already in use, or if `--target-dir` contains `postmaster.pid` or any existing files.

### 6.1 Restoring into an Isolated Container (`--network none`, Recommended for Verification & Staging)

Carried gap: when this isolated restore uses an encrypted repository, the calling shell must already export the repository passphrase and any required repository credentials. `restore-pitr.sh` loads no secret file and forwards only already-exported `PGBACKREST_REPO1_*` names to the helper container. The repository has no production environment-delivery procedure for this invocation. The lab profile leaves encryption off unless explicitly enabled; that unencrypted case does not need a decryption passphrase.

```bash
deploy/backup/restore-pitr.sh \
  --stanza fvpn \
  --target-time "2026-10-29T18:02:00.000000Z" \
  --isolated-container fvpn-w4-ops-backups-restore \
  --target-volume fvpn-w4-ops-backups-restore-pgdata \
  --repo-volume fvpn-w4-ops-backups-repo \
  --primary-container fvpn-w4-ops-backups-primary \
  --image fvpn-w4-ops-backups-postgres:17
```

### 6.2 Restoring into a Clean Host Data Directory (Production Host Replacement)

Carried gap: the native restore process running as the `postgres` OS account must inherit the repository passphrase and object-store credentials. `restore-pitr.sh` does not load `/etc/fvpn/secrets/pgbackrest-repo.env`; the backup services do not supply the environment to this invocation. The repository has no procedure for safely delivering it to this process; the command below alone cannot restore the encrypted production repository.

On a replacement database host (or after stopping `postgresql.service` and moving the damaged data directory aside to `/var/lib/postgresql/17/main.damaged`):
```bash
systemctl stop fvpn-control.service fvpn-worker.service postgresql.service
mv /var/lib/postgresql/17/main /var/lib/postgresql/17/main.damaged

sudo -u postgres /usr/local/lib/fvpn/backup/restore-pitr.sh \
  --stanza fvpn \
  --target-time "2026-10-29T18:02:00.000000Z" \
  --target-dir /var/lib/postgresql/17/main \
  --primary-dir /var/lib/postgresql/17/main.damaged
```

### 6.3 Automated Verification Script (`deploy/backup/verify.sh`)

Carried gap: this manual verification invocation needs the inherited repository credentials described in §5.2. The production environment-delivery procedure is absent; a backup service environment is not inherited by a separate shell invocation.

To run an end-to-end throwaway verification of the latest backup at any time:
```bash
deploy/backup/verify.sh \
  --stanza fvpn \
  --repo-volume fvpn-w4-ops-backups-repo \
  --primary-container fvpn-w4-ops-backups-primary \
  --image fvpn-w4-ops-backups-postgres:17
```

---

## 7. Ordered Post-Restore Reconciliation Protocol (`§17.3`, `§17.4`, `AT30`)

Execute the following steps **strictly in order** after restoring PostgreSQL to a target point in time (`RESTORED_TO_UTC`). Never start `fvpn-control` or `fvpn-worker` before completing Steps 1–5.

### Step 1: Stop Writers & Preserve Out-of-Band Signing Vault (`§16.2`, `§17.2`, `§17.3 Step 1`)

1. Stop `fvpn-control` and `fvpn-worker` so no HTTP mutations, usage batch ACKs, or outbox workers write against stale state:
   ```bash
   systemctl stop fvpn-control.service fvpn-worker.service
   ```
   *(Exit nodes continue forwarding traffic for authorized members within their valid serving leases and buffer usage events in local SQLite, `§16.3`, `§17.1`.)*
2. Verify that `/var/lib/fvpn/staging-authority/vault/state.json` and all key files in `/var/lib/fvpn/staging-authority/vault` are intact and untouched by the database restore (`§16.2`).

Carried gap: production custody and independent anti-rollback remain unimplemented. Use the [staging inspection boundary](../pki/inspect.md#key-use-inspection); local key inspection cannot prove that the whole vault directory was not rolled back.

### Step 2: Execute Physical PITR Restore (`restore-pitr.sh`)

Carried gap: an encrypted repository requires passphrase/credentials in the restore process environment; the production delivery procedure is absent. The notices in §6 apply to the selected invocation before running it.

Restore PostgreSQL to `RESTORED_TO_UTC` (or target LSN) using `deploy/backup/restore-pitr.sh` as shown in Section 6, start PostgreSQL on the restored data directory, and confirm `SELECT pg_is_in_recovery();` returns `f` (promoted).

### Step 3: Verify Schema Migration Status (`fvpn-migrate status`)

Verify that the restored database schema matches the deployed central binaries (`docs/db/README.md`). Supply `RESTORED_MIGRATOR_DSN` as the separate `fvpn_migrator` operator credential for the checked restored candidate, as in [restore migration selection](../handover/restore-drill.md#phase-2-restore-postgresql-17-and-freeze-usage-ingestion). Provision `RESTORED_RECOVERY_DSN_FILE` as a service-readable mode-`0600` file containing that candidate's `fvpn_recovery` DSN; use it for all recovery CLI commands below. Neither credential is the control/worker runtime DSN. Check the target before applying migrations or recovery mutations:
```bash
FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate status
FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate up
FVPN_DATABASE_URL="${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" /usr/local/bin/fvpn-migrate status
```
Use the migration binary built from the deployed release and confirm `current_database` identifies the restored target. Every file in that release's `db/migrations/*.sql` must be applied on the restored target, and the final `fvpn-migrate status` must report no pending version. [`deploy/backup/test/review_proof.sh`](../../deploy/backup/test/review_proof.sh), run by `make -f mk/backup.mk backup-review-proof`, derives the expected versions and count from that glob and checks status on its isolated PITR target.

### Step 4: Run the Read-Only Post-Restore Checklist (`deploy/backup/post-restore-checklist.sql`)

Execute [`deploy/backup/post-restore-checklist.sql`](../../deploy/backup/post-restore-checklist.sql) against the restored database:
```bash
psql "${RESTORED_MIGRATOR_DSN:?supply the restored candidate migrator DSN}" \
  -v ON_ERROR_STOP=1 \
  -v restore_point="${RESTORED_TO_UTC}" \
  -f /usr/local/lib/fvpn/backup/post-restore-checklist.sql
```
Review all six read-only output sections:
1. **`[1/6]` Usage ledger epoch, mode, and highest acknowledged sequence per node** (`usage_ledger_state`, `usage_checkpoints`, `usage_events`, `usage_rollups`).
2. **`[2/6]` Entitlements whose `effective_service_end` is after `restore_point`** and active/grace `subscriptions` + pending `inbox_events` requiring payment provider reconciliation (`§9.3`, `§17.4`).
3. **`[3/6]` Publication intents and external bindings in flight** (`exits`, `external_bindings`, `dns_publication_heads`, `operations`, `outbox_events`).
4. **`[4/6]` Update batches and node update jobs in flight** (`fleet_config`, `update_batches`, `update_jobs`).
5. **`[5/6]` Revocations, retirements, and administrative suspensions** (`nodes`, `node_incarnations`, `node_certificates`, `node_enrollments`, `members`, `member_devices`, `entitlement_grants`, `fleet_config.revoked_release_ids`) to cross-check against out-of-band operator and signing-vault logs (`SECOPS-05`).
6. **`[6/6]` Pre-formatted `fvpn-usage-admin` recovery commands** populated with the restored cluster's `restore_point`.

### Step 5: Freeze Usage Ingestion & Rotate `ledger_epoch` (`fvpn-usage-admin`, `§17.3 Steps 1–6`)

Using the existing recovery CLI (`./internal/usage/cmd/fvpn-usage-admin`, backed by `Service.FreezeIngest` and `Service.BeginRecovery` in [`internal/usage/recovery.go`](../../internal/usage/recovery.go)):

1. **Freeze usage ingestion** so `POST /api/v1/agent/usage-batches` returns `503 DEPENDENCY_UNAVAILABLE` (`retryable: true`) and cannot ACK batches against pre-recovery watermarks:
   ```bash
   FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
     /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
     freeze \
     --actor "${ADMIN_ACTOR_UUID}" \
     --reason "PITR restore to ${RESTORED_TO_UTC}"
   ```
2. **Rotate `ledger_epoch` and enter `reconciling` mode**:
   ```bash
   FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
     /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
     begin-recovery \
     --actor "${ADMIN_ACTOR_UUID}" \
     --restored-to "${RESTORED_TO_UTC}" \
     --affected-since "${AFFECTED_SINCE_UTC}" \
     --reason "PITR restore to ${RESTORED_TO_UTC}"
   ```
   `begin-recovery` atomically:
   - Generates a new `ledger_epoch` UUID (`new_ledger_epoch != previous_ledger_epoch`).
   - Recomputes `highest_contiguous_seq`, `max_seen_seq`, and `missing_ranges` for every `(node_id, incarnation_id)` stream strictly from committed `usage_events` rows in the restored database (`§17.3 Step 3`).
   - Marks affected `usage_rollups` rows with `reconciliation_in_progress = TRUE`.
   - Signs the `RecoverySignedPayload` using the central `ledger-epoch` Ed25519 key (`envelope.PurposeLedgerEpoch`) and records it in `usage_recovery_records`.
   - Sets `usage_ledger_state.mode = 'reconciling'`.

### Step 6: Start `fvpn-control` & Trigger Node Archive Replay (`§17.3 Steps 4–9`)

1. Start `fvpn-control.service` so exit nodes can reach the agent mTLS listener (`:9443`):
   Carried gap: restricted recovery startup is absent. Control can issue leases from restored authorization before Step 7.1 re-applies newer intent. Keep the worker stopped through that step; D-076 is conditional, not a global recovery fence. There is no implemented mode that completes usage replay, entitlement computation and policy sync before the automatic worker starts (spec §17.4 step 5). See [restore step 3.2](../handover/restore-drill.md#phase-3-rotate-ledger_epoch-and-start-central-services-173-steps-26) for the exposure and ordering.

   ```bash
   systemctl start fvpn-control.service
   ```
2. When each online exit node uploads its next usage batch under the old `ledger_epoch`, `fvpn-control` responds with `409 LEDGER_EPOCH_MISMATCH` (`current_ledger_epoch` in error details).
3. The node agent (`node/internal/usageupload`) automatically fetches `GET /api/v1/agent/usage-checkpoint`, verifies the Ed25519 `recovery` signature against the configured agent `trust_bundle` (the installer writes `/etc/vpn-node/trust.json`), lowers its upload cursor to the restored `highest_contiguous_seq`, and replays retained events from its local SQLite archive (`node/internal/usagestore`).
   - Events already present in the restored snapshot return `status: "duplicate"` and are never double-counted (`MTR03`).
   - Events committed after the restore point return `status: "accepted"`.
   - Any sequence range purged from the node's local archive because it exceeded the retention window is reported in `replay_unavailable`, inserting zero-byte tombstones and setting `gap = TRUE` on affected member rollups without inventing bytes (`§17.3 Step 9`).

### Step 7: Reconcile Entitlements, Revocations, Publication Intents & Fleet Updates (`§17.4`, `SECOPS-05`)

1. **Re-apply Post-Restore Revocations and Suspensions (`§6.4`, `§9.2`, `§17.4 Step 1`, `SECOPS-05`)**:
   - Before enabling background workers or opening public traffic, cross-check Section `[5/6]` of `post-restore-checklist.sql` against out-of-band operator audit logs.
   - Re-revoke any node incarnation with `POST /api/v1/admin/nodes/{node_id}/revoke`. Never restore a revoked node's service identity to retrieve historical usage (`§17.4`).
   - For a member device revoked after `RESTORED_TO_UTC`, the owning member uses `DELETE /api/v1/member/devices/{device_id}` (`operationId: revokeMemberDevice` in [`contracts/openapi.yaml` v1.2.0](../../contracts/openapi.yaml)). It requires the owner's member session, CSRF token, path parameter `device_id`, and `Idempotency-Key`; there is no request body. It is not an admin endpoint. Keep the member administratively suspended until the recorded revocation is enforced if the owner cannot complete it.
   - Re-apply administrative suspension with `PATCH /api/v1/admin/members/{member_id}` (`operationId: updateMember` in the same contract), an admin session, CSRF token, current `If-Match` ETag, and a fresh `Idempotency-Key`. Send `{"admin_suspended": true, "suspension_reason": "Reapply recorded post-restore suspension"}`. These are the exact `MemberPatch` fields for suspension: `admin_suspended` is boolean and `suspension_reason` is required when true (1–500 characters). Re-apply each recorded grant withdrawal with `POST /api/v1/admin/members/{member_id}/grants/{grant_id}/revoke`, following [explicit grant withdrawal](../members/README.md#explicit-grant-withdrawal-revision-14-at29): admin session, CSRF token, Origin, fresh UUID `Idempotency-Key` and `{"reason":"Reapply recorded post-restore grant withdrawal"}`; no grant ETag is required. The transaction commits `revoked_at`, `revocation_reason` and recomputed entitlement, then returns `202` with `member.grant_revoke`. Keep the member suspended while any recorded withdrawal remains unresolved; acceptance is not observed enforcement.
   - Node/device revocation and grant withdrawal commit the intent before returning an accepted operation. Suspension commits the member change and recomputes entitlement; it returns `202` when a changed projection creates an operation, or `200` when no operation is needed. The worker reads these durable changes on its first reconciliation. Start it only in Step 7.2 after recorded withdrawal intents have been committed, then verify terminal operation results and observed enforcement before opening public traffic; use `GET /api/v1/admin/operations/{operation_id}` for every returned operation. A `200` or `202` receipt alone does not establish remote enforcement.
2. **Start `fvpn-worker` and Reconcile Billing & Entitlements (`§9.3`, `§17.4 Step 2`)**:
   Carried gap: restricted recovery startup cannot finish usage replay, entitlement computation and policy sync before this automatic worker starts. Inventory and reconstruction of provider subscriptions absent from the restored rows also remain missing; the stored-subscription sweep below does not establish complete recovery.

   ```bash
   systemctl start fvpn-worker.service
   ```
   - `billing.reconcile` schedules synthetic inbox events for subscriptions already present in the restored database, including canceled ones. Processing those events reads current provider subscription, invoice and refund state and recomputes entitlement without clearing administrative suspension. It does not discover every subscription absent from the restored rows.
   - Carried gap: inventory and reconstruction of provider subscriptions absent from the restore point; the known-subscription sweep is not proof that every missed provider event has been recovered.
3. **Reconcile NetBird Overlay, Exit Publication Intents & DNS (`§5.2`, `§12.5`, `§17.4 Steps 3–4`)**:
   Carried gap: production AdGuard bootstrap and listener wiring remain prerequisites for the resolver convergence checked in this step; see [deployment §4.3](../handover/deploy.md#43-adguard-home-resolvers-v010759-12-docsdnsbackendmd-docsdnsprivacymd).
   - `node.reconcile` handles generation holds, fleet database job state and bootstrap expiry. `members.reconcile` recomputes entitlements and syncs existing device rows with authorization read-back; D-076 fences qualifying unexplained withdrawals. The separate `devices.discover` sweep skips userless/ephemeral peers and may authorize an eligible unknown member peer. These jobs do not promise global orphan-peer deletion or published-route/policy drift cleanup; see the [restore reconciliation limits](../handover/restore-drill.md#phase-5-reconcile-netbird-overlay-node-pki-and-dns-state-52-64-122-174).
   - Carried gap: inventory and revocation of peers authorized after the restore point, including peers absent from restored records. The sweeps are not proof that all such peers, routes or policies were withdrawn.
   - For any exit node listed in Section `[3/6]` of `post-restore-checklist.sql` whose post-restore publication intent cannot be confirmed, keep `publication_intent` in `unpublished` or `disabled` (fail-closed).
   - Verify `dns_publication_heads` and resolver convergence (`dns.publish`).
4. **Reconcile Fleet Update Batches & Node Releases (`§13.8`, `§17.4 Step 3`)**:
   - Inspect Section `[4/6]` of `post-restore-checklist.sql`. As nodes send heartbeats (`POST /api/v1/agent/observations`), central records their `actual_release_id` and `applied_generation`. Central **never** forces a node to downgrade from a newer approved/safe release solely because the database snapshot was older (`§17.4 Step 3`).

### Step 8: Complete Ledger Recovery & Resume Normal Operations (`§17.3`, `§17.4 Steps 5–6`)

1. Once all active nodes have adopted `new_ledger_epoch` and replayed their retained archives, exit reconciling mode:
   ```bash
   FVPN_DATABASE_URL_FILE="${RESTORED_RECOVERY_DSN_FILE:?supply the restored candidate recovery DSN file}" \
     /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
     complete-recovery \
     --actor "${ADMIN_ACTOR_UUID}"
   ```
   This marks the `usage_recovery_records` row `completed`, clears `reconciliation_in_progress` on `usage_rollups` (preserving `gap = TRUE` where `replay_unavailable` occurred), and sets `usage_ledger_state.mode = 'normal'`.
2. Carried gap: authorized audit/rollup purge execution and scheduling, plus JSON/database retention coordination, still need a procedure. The function supports only its documented targets and enforces database cutoff policy; the reference `SELECT purge_retained(...)` is not an executable purge instruction. Review [calibration retention](../handover/calibration.md#16-data-retention-periods-105-154-164-173-21) before minimizing rows reintroduced by the snapshot (`§16.4`).
3. Run post-restore verification probes (`curl -fsS https://vpn.example.com:8443/readyz`, isolated test member login, DNS resolution, and usage query) and record the measured RPO and RTO.

---

## 8. Automated CI Drill & Measured Results (`mk/backup.mk`, `docs/ops/backups/`)

The full physical backup, WAL switch, and isolated PITR restore drill is automated in `tests/backup/drill_integration_test.go` (`TestAT30_BackupPITRDrill`) and runs on every push touching backup paths via `.github/workflows/backup.yml`:

```bash
make -f mk/backup.mk backup-lint backup-test backup-drill
```

What `backup-drill` proves on every run. Container, volume, network, and image-tag names come from `FVPN_BACKUP_DRILL_PREFIX` when that variable matches `fvpn-w4-ops-backups-<lowercase-run-id>`. A non-empty invalid value fails the drill before it creates resources. Only an unset/empty value selects `fvpn-w4-ops-backups-` plus 8 lowercase hex digits (`docs/handover/restore-drill.md`). The test removes only that prefix.
1. Builds `deploy/backup/postgres/Dockerfile` (`PostgreSQL 17` + `pgBackRest 2.60.0`) and starts primary container `<prefix>-primary` with `archive_mode=on` and repo volume `<prefix>-repo`.
2. Applies all forward migrations via `fvpn-migrate up`.
3. Creates drill-only table `_backup_drill` (never touching product tables, `D-050`) and commits Marker `A`.
4. Runs `deploy/backup/backup.sh --container <prefix>-primary full` to take a physical full base backup.
5. Commits Marker `M` after the backup stop, then records `targetTimestamp` (UTC) and `targetLSN`, commits Marker `B`, and forces a WAL switch (`backup.sh wal-switch`).
6. Verifies that `restore-pitr.sh` **refuses** to overwrite the live primary container (`live_primary_overwrite_refused = true`).
7. Runs `deploy/backup/restore-pitr.sh --target-time <targetTimestamp>` into an isolated container `<prefix>-restore` (`--network none`, `0` host ports).
8. Asserts the restored set is exactly `{A, M}` and separately that Marker `B` is absent in the restored database.
9. Asserts `fvpn-migrate status` reports all migrations `applied` (`0` pending).
10. Executes `deploy/backup/post-restore-checklist.sql` and `deploy/backup/verify.sh` (verify prefix `<prefix>-verify`).
11. Only when `FVPN_BACKUP_DRILL_RECORD=1`, writes the archive ages, local command durations and backup set SHA-256 digests to [`docs/ops/backups/drill-2026-10-29.json`](backups/drill-2026-10-29.json) and [`docs/ops/backups/drill-2026-10-29.md`](backups/drill-2026-10-29.md), then removes Docker objects owned by `<prefix>`. Stale `fvpn-w4-ops-backups-*` objects are removed only by `make -f mk/backup.mk backup-drill-clean`, which the test does not call.

### Latest Drill Evidence (`drill-2026-10-29`)

See [the generated evidence](backups/drill-2026-10-29.md) for archive ages and local restore-command durations. `wal_switch_to_archive_ms` is a stopwatch, not archive lag. The repository age uses the newest WAL object's modification time in the repository used for restore; it is an archive-freshness observation, not a decoded commit timestamp. Both it and `pg_stat_archiver.last_archived_time` must be known and no older than 15 minutes; unknown or stale state fails verification. These local POSIX measurements do not establish the 4-hour service RTO or production off-host RPO. A separate service drill must include secrets, settings, reconciliation and user probes.

Evidence writes are opt-in: ordinary tests write nothing under `docs/`. Set `FVPN_BACKUP_DRILL_RECORD=1` (optionally `FVPN_BACKUP_DRILL_DATE` and `FVPN_BACKUP_DRILL_OUTPUT_DIR`) for a reviewed record. CI enables this only via the manual `workflow_dispatch.record_evidence` input, never on pull requests. Run `make -f mk/backup.mk backup-review-proof` for the adversarial controls.
