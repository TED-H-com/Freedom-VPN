# Central PostgreSQL 17 Backups, Point-in-Time Recovery (PITR) & Post-Restore Reconciliation Runbook (`docs/ops/backups.md`)

This runbook specifies the production and lab backup architecture, continuous WAL archiving configuration, encryption and key separation rules, monitoring signals, point-in-time restore (PITR) execution, and the ordered post-restore reconciliation protocol for Freedom VPN (`§3.1`, `§15.4`, `§16.2`, `§16.4`, `§17.1`–`§17.5`, `§19 AT30`, `SECOPS-05`).

---

## 1. Recovery Objectives & Pinned Tooling Baseline (`§16.5`, `§17.2`)

| Parameter / Component | Specification | Notes & Spec Reference |
|---|---|---|
| **Database Engine** | PostgreSQL 17 (`postgres@sha256:31ba9c94a03ada9988f2bb493a22a75eef3a185884533e5cdbcd498ea835b08f`) | Drop-in image built from [`deploy/backup/postgres/Dockerfile`](../../deploy/backup/postgres/Dockerfile) (`§3.1`, `§15.4`). |
| **Backup Engine** | **pgBackRest `2.60.0`** (`2.60.0-1.pgdg13+1`) | Physical full + differential base backups and continuous WAL archiving (`§17.2`). Logical `pg_dump` is an auxiliary export only, never the primary recovery path. |
| **Central Database RPO Target** | **`<= 15 minutes`** (`900s`) | Enforced via `archive_mode = on` and `archive_timeout = 300s` (`§16.5`, `§17.2`). |
| **Usage Ledger Effective RPO** | **`0` bytes lost / `0` double-counted** (within node retention window) | Exit nodes retain acknowledged usage events in a dedicated SQLite archive (`>= 72 hours` spec minimum, `30 days` default) and replay after `ledger_epoch` rotation (`§10.5`, `§16.4`, `§17.3`, `AT30`). |
| **Central Service RTO Target** | **`<= 2 hours`** (`docs/handover/restore-drill.md`; `<= 4 hours` ceiling in `§16.5`/`§17.2`) | Covers physical DB/WAL restore, migration check, checklist inspection, ledger freeze/epoch rotation, replay, and provider/NetBird reconciliation. |
| **Signing Vault (`/var/lib/fvpn/vault`) RPO** | **`0` (Zero Rollback)** | Stored outside PostgreSQL and never included in database snapshots (`§16.2`, `§17.2`). |

---

## 2. Production & Lab Configuration (`§3.1`, `§15.4`, `§17.2`)

### 2.1 PostgreSQL 17 WAL Archiving Fragment (`deploy/backup/postgres/fvpn-archiving.conf`)

Installed at `/etc/postgresql/conf.d/50-fvpn-archiving.conf` (included via `include_dir = '/etc/postgresql/conf.d'` in `postgresql.conf`):

```ini
wal_level = replica
archive_mode = on
archive_timeout = 300s
archive_command = 'pgbackrest --stanza=fvpn archive-push %p'
max_wal_senders = 5
```

- **Production**: `archive_timeout = 300s` (5 minutes) guarantees that even during low-traffic periods, PostgreSQL forces a WAL segment switch at least every 5 minutes so continuous WAL archiving satisfies the `<= 15 minutes` RPO target (`§17.2`).
- **Lab / CI Override (`TEST ONLY`)**: Containerized lab and drill environments may set `FVPN_PG_ARCHIVE_TIMEOUT=10s` on `deploy/backup/postgres/Dockerfile` to bound idle archive lag during automated tests.
- **Isolated Verification Containers**: When `restore-pitr.sh` launches an isolated verification container (`--network none`), it sets `FVPN_PG_ISOLATED_RESTORE=1` and mounts the backup repository read-only (`:ro`). The entrypoint writes `archive_mode = off` inside the verification container so promotion to a new timeline never pushes WAL segments back to the primary repository.

### 2.2 pgBackRest Repository Configuration (`deploy/backup/pgbackrest.conf.example`)

See [`deploy/backup/pgbackrest.conf.example`](../../deploy/backup/pgbackrest.conf.example) for the complete reference file:
- **Production Profile (`repo1-type=s3`)**: Archives to an off-host S3-compatible object store (`repo1-s3-verify-tls=y`, `repo1-cipher-type=aes-256-cbc`, `compress-type=zst`).
- **Lab Profile (`repo1-type=posix`)**: Archives to a mounted Docker volume at `/var/lib/pgbackrest` for local devstack and CI PITR drills.

### 2.3 Least-Privilege Database Roles (`docs/handover/deploy.md §3.2`, `docs/db/README.md`)

- `fvpn_backup`: Read/replication role for backup verification and logical exports (`GRANT pg_read_all_data TO fvpn_backup`); holds no mutation privileges on application tables. Physical pgBackRest backups run as the local `postgres` OS/cluster account over the Unix domain socket (`/var/run/postgresql`).
- `fvpn_migrator`: Schema owner used exclusively by `fvpn-migrate up` and `fvpn-migrate status`.
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
```bash
install -d -m 0755 /usr/local/lib/fvpn/backup
install -m 0755 deploy/backup/backup.sh deploy/backup/restore-pitr.sh deploy/backup/verify.sh /usr/local/lib/fvpn/backup/
install -m 0644 deploy/backup/post-restore-checklist.sql /usr/local/lib/fvpn/backup/
cp deploy/backup/systemd/fvpn-backup-*.service deploy/backup/systemd/fvpn-backup-*.timer /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now fvpn-backup-full.timer fvpn-backup-diff.timer fvpn-backup-verify.timer
```

### 3.2 Retention & Post-Restore Data Minimization (`§16.4`, `§17.2`)

- **Backup Retention (`pgbackrest.conf`)**:
  - `repo1-retention-full-type=time`
  - `repo1-retention-full=35` (retains full base backups and continuous WAL archives covering 35 days, aligned with `retention_config.raw_usage_days = 35` in `§16.4`).
  - `repo1-retention-diff=14` (retains 14 days of differential backups).
- **Re-applying Retention & Anonymization After Restore (`§16.4`)**:
  Restoring an earlier backup can re-introduce raw usage payloads, rollups, or audit rows whose retention cutoff has already passed, or personal metadata (`email`, `display_name`) for accounts closed during the recovery window. Before returning the restored cluster to normal service, run the authorized retention purge (`SELECT purge_retained(...)` as `fvpn_retention` after projection reconciliation completes) and re-apply account closure/anonymization actions recorded after the restore point.

---

## 4. Encryption & Key Separation Contract (`§16.2`, `§17.2`)

1. **Repository Encryption at Rest**:
   All pgBackRest base backups and WAL archives are encrypted client-side using `repo1-cipher-type=aes-256-cbc` before leaving the database host.
2. **Repository Key Separation**:
   The pgBackRest encryption passphrase (`PGBACKREST_REPO1_CIPHER_PASS`) and object-store credentials (`PGBACKREST_REPO1_S3_KEY`, `PGBACKREST_REPO1_S3_KEY_SECRET`) are stored in `/etc/fvpn/secrets/pgbackrest-repo.env` (owned by `root:postgres`, mode `0600`), **outside PostgreSQL** and **outside the central signing vault (`/var/lib/fvpn/vault`)**. Escrowed recovery copies of `pgbackrest-repo.env` are kept in the operator hardware/offline secret store separate from the S3 bucket.
3. **Central Signing Vault Never Inside Database Backups (`§16.2`, `§17.2`)**:
   The central signing vault (`/var/lib/fvpn/vault`) holds the active and historical keys and revocation state (`state.json`) for `node-ca`, `bootstrap-secret`, `desired-state`, `lease`, `recovery`, and `ledger-epoch`. It is **never** stored in PostgreSQL or backed up inside pgBackRest snapshots. Consequently, restoring PostgreSQL to an earlier timestamp never rolls back a key rotation or un-revokes a compromised signing key (`§16.2`).

---

## 5. Monitoring & Alerting Signals (`§16.5`, `§17.2`)

Operators and monitoring collectors watch three primary signals to guarantee the `<= 15 minutes` RPO target:

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
    EXTRACT(EPOCH FROM (clock_timestamp() - COALESCE(last_archived_time, clock_timestamp())))::bigint AS archive_lag_seconds
FROM pg_stat_archiver;
```
- **Warning threshold**: `archive_lag_seconds > 360` (6 minutes, i.e., `archive_timeout = 300s` + 60s grace) or `failed_count` increasing.
- **Critical threshold (RPO breach risk)**: `archive_lag_seconds > 600` (10 minutes; approaches the 15-minute RPO bound).

### 5.2 Last Successful Base Backup Age (`pgbackrest info`)

Inspect the repository metadata JSON via `deploy/backup/backup.sh info`:
```bash
deploy/backup/backup.sh --stanza fvpn info
```
Monitor inside the JSON output:
- `.[0].status.code`: Must equal `0` (`"ok"`). Any non-zero code indicates a missing stanza or broken archive stream.
- `.[0].backup[-1].timestamp.stop`: Unix epoch timestamp of the most recent successful backup. Alert if `now - stop > 93600` (26 hours, indicating a missed daily backup).
- `.[0].archive[-1].max`: Latest archived WAL segment in the repository.

### 5.3 Log Lines Watched by Operators

- **PostgreSQL server log (`/var/log/postgresql/` or container stderr)**:
  - Failure pattern: `LOG:  archive command failed with exit code` or `DETAIL:  The failed archive command was: pgbackrest --stanza=fvpn archive-push`
- **pgBackRest log (`/var/log/pgbackrest/fvpn-*.log`) & systemd journal (`journalctl -u fvpn-backup-*`)**:
  - Success pattern: `INFO: archive-push command end: completed successfully` / `INFO: backup command end: completed successfully`
  - Failure pattern: `ERROR: [` (followed by pgBackRest numeric error code `[025]`–`[125]`).
- **Daily Verification Service (`journalctl -u fvpn-backup-verify.service`)**:
  - Success pattern: `status=PASS` with `migrations_pending=0` and `rpo_ms` / `rto_ms`.

---

## 6. Point-in-Time Restore (PITR) Procedure (`deploy/backup/restore-pitr.sh`)

`deploy/backup/restore-pitr.sh` restores a backup + WAL stream to an exact UTC timestamp (`--target-time`), WAL LSN (`--target-lsn`), or the end of the archived WAL stream (`--latest`).

> **Safety Contract**: `restore-pitr.sh` **never** restores over a live primary. It refuses execution if `--isolated-container` matches `--primary-container`, if the target container or volume is already in use, or if `--target-dir` contains `postmaster.pid` or any existing files.

### 6.1 Restoring into an Isolated Container (`--network none`, Recommended for Verification & Staging)

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
2. Verify that `/var/lib/fvpn/vault/state.json` and all key files in `/var/lib/fvpn/vault` are intact and untouched by the database restore (`§16.2`).

### Step 2: Execute Physical PITR Restore (`restore-pitr.sh`)

Restore PostgreSQL to `RESTORED_TO_UTC` (or target LSN) using `deploy/backup/restore-pitr.sh` as shown in Section 6, start PostgreSQL on the restored data directory, and confirm `SELECT pg_is_in_recovery();` returns `f` (promoted).

### Step 3: Verify Schema Migration Status (`fvpn-migrate status`)

Verify that the restored database schema matches the deployed central binaries (`docs/db/README.md`):
```bash
FVPN_DATABASE_URL="$(cat /etc/fvpn/secrets/database.url)" /usr/local/bin/fvpn-migrate status
FVPN_DATABASE_URL="$(cat /etc/fvpn/secrets/database.url)" /usr/local/bin/fvpn-migrate up
```
Confirm that every migration (`00001` through `00181`) reports `applied` and zero migrations report `pending`.

### Step 4: Run the Read-Only Post-Restore Checklist (`deploy/backup/post-restore-checklist.sql`)

Execute [`deploy/backup/post-restore-checklist.sql`](../../deploy/backup/post-restore-checklist.sql) against the restored database:
```bash
psql "$(cat /etc/fvpn/secrets/database.url)" \
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
   /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
     freeze \
     --actor "${ADMIN_ACTOR_UUID}" \
     --reason "PITR restore to ${RESTORED_TO_UTC}"
   ```
2. **Rotate `ledger_epoch` and enter `reconciling` mode**:
   ```bash
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
   ```bash
   systemctl start fvpn-control.service
   ```
2. When each online exit node uploads its next usage batch under the old `ledger_epoch`, `fvpn-control` responds with `409 LEDGER_EPOCH_MISMATCH` (`current_ledger_epoch` in error details).
3. The node agent (`node/internal/usageupload`) automatically fetches `GET /api/v1/agent/usage-checkpoint`, verifies the Ed25519 `recovery` signature against `/etc/vpn-node/trust.json`, lowers its upload cursor to the restored `highest_contiguous_seq`, and replays retained events from its local SQLite archive (`node/internal/usagestore`).
   - Events already present in the restored snapshot return `status: "duplicate"` and are never double-counted (`MTR03`).
   - Events committed after the restore point return `status: "accepted"`.
   - Any sequence range purged from the node's local archive because it exceeded the retention window is reported in `replay_unavailable`, inserting zero-byte tombstones and setting `gap = TRUE` on affected member rollups without inventing bytes (`§17.3 Step 9`).

### Step 7: Reconcile Entitlements, Revocations, Publication Intents & Fleet Updates (`§17.4`, `SECOPS-05`)

1. **Re-apply Post-Restore Revocations and Suspensions (`§6.4`, `§9.2`, `§17.4 Step 1`, `SECOPS-05`)**:
   - Before enabling background workers or opening public traffic, cross-check Section `[5/6]` of `post-restore-checklist.sql` against out-of-band operator audit logs.
   - Re-revoke any node incarnation (`POST /api/v1/admin/nodes/{node_id}/revoke`) or member device (`DELETE /api/v1/admin/members/{member_id}/devices/{device_id}`) that was revoked after `RESTORED_TO_UTC`. Never restore a revoked node's service identity to retrieve historical usage (`§17.4`).
   - Re-apply any administrative member suspension (`POST /api/v1/admin/members/{member_id}/suspend`) or manual grant revocation executed after `RESTORED_TO_UTC`.
2. **Start `fvpn-worker` and Reconcile Billing & Entitlements (`§9.3`, `§17.4 Step 2`)**:
   ```bash
   systemctl start fvpn-worker.service
   ```
   - `billing.reconcile` queries the payment provider for current subscriptions, invoices, and refunds, repairing any webhooks missed during the RPO window without duplicating charges (`BILL01`) and without clearing `members.admin_suspended` (`ENT02`).
3. **Reconcile NetBird Overlay, Exit Publication Intents & DNS (`§5.2`, `§12.5`, `§17.4 Steps 3–4`)**:
   - `node.reconcile` and `members.devicereconcile` read back observed NetBird peers, routes, and groups and converge them to central desired state, disabling any peer absent from restored `member_devices` or `node_incarnations` (fail-closed).
   - For any exit node listed in Section `[3/6]` of `post-restore-checklist.sql` whose post-restore publication intent cannot be confirmed, keep `publication_intent` in `unpublished` or `disabled` (fail-closed).
   - Verify `dns_publication_heads` and resolver convergence (`dns.publish`).
4. **Reconcile Fleet Update Batches & Node Releases (`§13.8`, `§17.4 Step 3`)**:
   - Inspect Section `[4/6]` of `post-restore-checklist.sql`. As nodes send heartbeats (`POST /api/v1/agent/observations`), central records their `actual_release_id` and `applied_generation`. Central **never** forces a node to downgrade from a newer approved/safe release solely because the database snapshot was older (`§17.4 Step 3`).

### Step 8: Complete Ledger Recovery & Resume Normal Operations (`§17.3`, `§17.4 Steps 5–6`)

1. Once all active nodes have adopted `new_ledger_epoch` and replayed their retained archives, exit reconciling mode:
   ```bash
   /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
     complete-recovery \
     --actor "${ADMIN_ACTOR_UUID}"
   ```
   This marks the `usage_recovery_records` row `completed`, clears `reconciliation_in_progress` on `usage_rollups` (preserving `gap = TRUE` where `replay_unavailable` occurred), and sets `usage_ledger_state.mode = 'normal'`.
2. Re-run retention purges (`SELECT purge_retained(...)`) if any expired rows were reintroduced by the snapshot (`§16.4`).
3. Run post-restore verification probes (`curl -fsS https://vpn.example.com:8443/readyz`, isolated test member login, DNS resolution, and usage query) and record the measured RPO and RTO.

---

## 8. Automated CI Drill & Measured Results (`mk/backup.mk`, `docs/ops/backups/`)

The full physical backup, WAL switch, and isolated PITR restore drill is automated in `tests/backup/drill_integration_test.go` (`TestAT30_BackupPITRDrill`) and runs on every push touching backup paths via `.github/workflows/backup.yml`:

```bash
make -f mk/backup.mk backup-lint backup-test backup-drill
```

What `backup-drill` proves on every run:
1. Builds `deploy/backup/postgres/Dockerfile` (`PostgreSQL 17` + `pgBackRest 2.60.0`) and starts primary container `fvpn-w4-ops-backups-primary` with `archive_mode=on` and repo volume `fvpn-w4-ops-backups-repo`.
2. Applies all forward migrations via `fvpn-migrate up`.
3. Creates drill-only table `_backup_drill` (never touching product tables, `D-050`) and commits Marker `A`.
4. Runs `deploy/backup/backup.sh --container fvpn-w4-ops-backups-primary full` to take a physical full base backup.
5. Records `targetTimestamp` (UTC) and `targetLSN`, commits Marker `B`, and forces a WAL switch (`backup.sh wal-switch`).
6. Verifies that `restore-pitr.sh` **refuses** to overwrite the live primary container (`live_primary_overwrite_refused = true`).
7. Runs `deploy/backup/restore-pitr.sh --target-time <targetTimestamp>` into an isolated container `fvpn-w4-ops-backups-restore` (`--network none`, `0` host ports).
8. Asserts Marker `A` is present and Marker `B` is absent in the restored database.
9. Asserts `fvpn-migrate status` reports all migrations `applied` (`0` pending).
10. Executes `deploy/backup/post-restore-checklist.sql` and `deploy/backup/verify.sh`.
11. Writes the measured RPO/RTO and backup set SHA-256 digests to [`docs/ops/backups/drill-2026-10-29.json`](backups/drill-2026-10-29.json) and [`docs/ops/backups/drill-2026-10-29.md`](backups/drill-2026-10-29.md), then removes all `fvpn-w4-ops-backups-*` Docker resources.

### Latest Drill Summary (`drill-2026-10-29`)

| Metric | Spec / Handover Target (`§17.2`) | Measured Value (`drill-2026-10-29`) | Status |
|---|---|---|---|
| **Continuous WAL Archive Lag (RPO)** | `<= 15 minutes` (`900 s`) | **`277 ms`** (`0.277 s`) | **PASS** |
| **Physical PITR Restore + Promotion (Database RTO)** | `<= 2 hours` (`7200 s`) | **`3551 ms`** (`3.551 s`) | **PASS** |
| **Full Throwaway Verification (`verify.sh`)** | `<= 2 hours` (`7200 s`) | **`3931 ms`** (`3.931 s`) | **PASS** |
| **Total End-to-End Automated Drill Duration** | `<= 10 minutes` CI budget | **`15070 ms`** (`15.070 s`) | **PASS** |
| **Restored Markers (`_backup_drill`)** | `A` present, `B` absent | `A` (`marker_a_present=true`, `marker_b_absent=true`) | **PASS** |
| **Restored Schema Migrations (`fvpn-migrate status`)** | `20 applied`, `0 pending` | `20 applied`, `0 pending` | **PASS** |
| **Isolated Restore Network & Published Ports** | `--network none`, `0` ports | `network=none`, `published_ports=0` | **PASS** |

