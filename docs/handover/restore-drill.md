# Backup and Restore Drill Record (`docs/handover/restore-drill.md`)

This document provides the repeatable backup, point-in-time restore (PITR), and post-restore reconciliation procedure (`§9.3`, `§17.2`, `§17.3`, `§17.4`, `§19 AT30`), together with the measured RPO and RTO evidence from the Wave 4 verification drill.

---

## 1. Recovery Objectives (`§17.2`)

| Subsystem | RPO Target | RTO Target | Architectural Guarantee |
|---|---|---|---|
| **Central PostgreSQL 17** (Members, Entitlements, Nodes, Policies, Audit) | `≤ 15 minutes` | `≤ 2 hours` | Continuous WAL archiving (`archive_timeout = 300s`) + daily base backups. |
| **Usage Ledger (`usage_events`, `usage_checkpoints`, `usage_rollups`)** | **Effective RPO `0`** (within node 30-day archive)<br>`≤ 15 minutes` (if node archive also lost) | `≤ 2 hours` (ingest resumes; reconciliation runs asynchronously) | Nodes retain ACKed events for `30 days` (`§10.5`). Rotating `ledger_epoch` forces nodes to verify a signed checkpoint and replay post-snapshot events without double-counting (`AT30`). |
| **NetBird Management Store** | `≤ 15 minutes` | `≤ 2 hours` | Reconciled from central desired state (`internal/netbird/reconcile.go`) after restore. |
| **Signing Vault (`/var/lib/fvpn/vault`) & PKI Revocations** | **`0` (Zero Loss)** | Immediate | Stored outside PostgreSQL snapshots (`§16.2`, `§17.2`). Never overwritten by a database rollback. |

---

## 2. Wave 4 AT30 Restore Drill Record (`2026-10-29`)

### 2.1 Drill Scenario & Topology

On `2026-10-29`, the Wave 4 AT30 point-in-time restore and reconciliation drill was executed against PostgreSQL 17 (`fvpn_w4_handover_drill`) and verified alongside `TestAT30_RestoreSnapshotRecoveryReplayAndReplayUnavailable` and `TestAT30_HTTPRecoveryFreezeEpochRotationReplayAndAuditPagination`:

1. **Pre-incident baseline**:
   - Member `0196ed00-0000-7000-8000-000000000101` (`active`), Device `0196ed00-0000-7000-8000-000000000201` (`100.64.0.10`, `binding_revision=1`).
   - Exit Node `0196ed00-0000-7000-8000-000000000301`, Incarnation `0196ed00-0000-7000-8000-000000000401`.
   - Node emitted usage events `event_seq = 1..5` (`forward_admission_v1`, `seq * 1000` upload bytes, `seq * 2000` download bytes per 1-minute window starting `2026-10-29T18:00:00Z`).
   - Database snapshot (`restored_to = 2026-10-29T18:02:00Z`) contained only `event_seq = 1..2` (`3,000` upload bytes, `6,000` download bytes, `highest_contiguous_seq = 2`, epoch `1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba`).
   - Events `3..5` were ACKed prior to the simulated central DB loss. On the node, `event_seq = 3` remained in the local SQLite archive, while `event_seq = 4..5` simulated local retention expiry (`retention_expired`), followed by new post-restore traffic `event_seq = 6` (`6,000` upload bytes, `12,000` download bytes).

### 2.2 Measured RPO and RTO Summary

| Metric | Spec Target (`§17.2`) | Wave 4 Measured Value | Result |
|---|---|---|---|
| **Snapshot Window Lag (Raw DB RPO before node replay)** | `≤ 15 minutes` | `3 minutes` (`18:02:00Z` snapshot vs `18:05:00Z` pre-crash watermark) | **PASS** |
| **Effective Usage Ledger RPO (Retained node archive `seq 1..3`)** | `0 bytes lost / 0 double-counted` | **`0 bytes lost / 0 bytes double-counted`** (`seq 1..2` deduplicated, `seq 3` replayed and accepted) | **PASS** |
| **Unrecoverable Node Range (`seq 4..5`) Handling** | Honest `gap=true` (`0` invented bytes) | **`gap=true`**, `2` zero-byte `replay_unavailable` tombstones recorded, `0` fabricated bytes | **PASS** |
| **Database Schema + Snapshot Restore Time** | Part of `≤ 2 hours` RTO | `342 ms` (isolated PostgreSQL 17 schema + state load) | **PASS** |
| **Ledger Freeze -> Epoch Rotation -> Node Replay -> Complete Recovery (Service RTO)** | `≤ 2 hours` | **`64 ms`** (`406 ms` total drill wall-clock including migrations) | **PASS** |

### 2.3 Recorded Drill Timeline & Verified Command Outputs

- **`2026-10-29T22:23:20.006Z` — Verify migrations on restored target (`fvpn-migrate status`)** `[Verified on dev/test stack]`:
  ```text
  current_database="fvpn_w4_handover_drill"
  00001 applied
  00002 applied
  00003 applied
  00004 applied
  00005 applied
  00006 applied
  00100 applied
  00101 applied
  00110 applied
  00111 applied
  00120 applied
  00121 applied
  00130 applied
  00131 applied
  00140 applied
  00150 applied
  00160 applied
  00161 applied
  00180 applied
  ```

- **`2026-10-29T22:23:20.348Z` — Step 1: Freeze usage ingestion (`fvpn-usage-admin freeze`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    freeze \
    --actor 0196ed00-0000-7000-8000-000000000999 \
    --reason "AT30 point-in-time restore drill"
  ```
  Output:
  ```json
  {
    "actor": "0196ed00-0000-7000-8000-000000000999",
    "reason": "AT30 point-in-time restore drill",
    "status": "frozen"
  }
  ```

- **`2026-10-29T22:23:20.371Z` — Step 2: Rotate `ledger_epoch` and enter `reconciling` mode (`fvpn-usage-admin begin-recovery`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    begin-recovery \
    --actor 0196ed00-0000-7000-8000-000000000999 \
    --reason "AT30 restore from 2026-10-29T18:02:00Z snapshot" \
    --restored-to 2026-10-29T18:02:00Z \
    --affected-since 2026-10-29T18:00:00Z
  ```
  Output:
  ```json
  {
    "recovery_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f",
    "previous_ledger_epoch": "1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba",
    "new_ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
    "mode": "reconciling",
    "reason": "AT30 restore from 2026-10-29T18:02:00Z snapshot",
    "actor": "0196ed00-0000-7000-8000-000000000999",
    "signed_record": {
      "signed": {
        "actor": "0196ed00-0000-7000-8000-000000000999",
        "affected_since": "2026-10-29T18:00:00Z",
        "created_at": "2026-10-29T22:23:20.371746519Z",
        "new_ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
        "previous_ledger_epoch": "1f7b1c2a-48b1-4ab2-a75a-b125bec8e0ba",
        "reason": "AT30 restore from 2026-10-29T18:02:00Z snapshot",
        "recovery_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f",
        "restored_to": "2026-10-29T18:02:00Z",
        "stream_count": 1
      },
      "signatures": [
        {
          "key_id": "01a189c2-afc3-7077-bd89-df83873cc3d8",
          "algorithm": "ed25519",
          "signature_b64": "xdNE2jfT0g0Ll3oMmlEgctQ+ajJSVMzZnmRsrjo0D2pT2UlwHfFQBtTKzywoOBemLx5y2KptS6RqEYOiq3l1Bg=="
        }
      ]
    },
    "recomputed_streams": 1,
    "created_at": "2026-10-29T22:23:20.371746519Z"
  }
  ```

- **`2026-10-29T22:23:20.383Z` — Step 3: Node pulls signed checkpoint (`GET /api/v1/agent/usage-checkpoint`) and replays `seq 1..3` + `replay_unavailable [4, 5]` + `seq 6`** `[Verified on dev/test stack]`:
  Signed checkpoint verified against the node `recovery` trust key:
  ```json
  {
    "signatures": [
      {
        "algorithm": "ed25519",
        "key_id": "01a189c2-afbf-7250-b72a-a29091e7e92e",
        "signature_b64": "5/jgJgmCJ8/u14FKMCnC3Qfd2fLQZ622olWZIqv9NkGDDo3lVDym9cRXgAJZIqNje63hLh5FjR7+2wR8jtsEAg=="
      }
    ],
    "signed": {
      "highest_contiguous_seq": "2",
      "issued_at": "2026-10-29T22:23:20.383726806Z",
      "ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
      "max_seen_seq": "2",
      "missing_ranges": [],
      "node_id": "0196ed00-0000-7000-8000-000000000301",
      "node_incarnation_id": "0196ed00-0000-7000-8000-000000000401",
      "recovery_record_id": "01a189c2-b133-7b6a-9006-2d8d7a99895f"
    }
  }
  ```
  Batch ingest response (`POST /api/v1/agent/usage-batches`):
  ```json
  {
    "blocked_seq": null,
    "highest_contiguous_seq": "6",
    "ledger_epoch": "01a189c2-b133-7b6b-9a37-d34f1e963d61",
    "node_id": "0196ed00-0000-7000-8000-000000000301",
    "node_incarnation_id": "0196ed00-0000-7000-8000-000000000401",
    "received_at": "2026-10-29T22:23:20.385579324Z",
    "results": [
      { "error": null, "event_seq": "1", "status": "duplicate" },
      { "error": null, "event_seq": "2", "status": "duplicate" },
      { "error": null, "event_seq": "3", "status": "accepted" },
      { "error": null, "event_seq": "6", "status": "accepted" }
    ],
    "max_seen_seq": "6"
  }
  ```

- **`2026-10-29T22:23:20.412Z` — Step 4: Exit reconciling mode (`fvpn-usage-admin complete-recovery`) and verify member rollup (`fvpn-usage-admin drill-down`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    complete-recovery \
    --actor 0196ed00-0000-7000-8000-000000000999

  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    drill-down \
    --member-id 0196ed00-0000-7000-8000-000000000101 \
    --period 2026-10
  ```
  Verified totals: `upload_bytes = "12000"` (`1000 + 2000 + 3000 + 6000`), `download_bytes = "24000"` (`2000 + 4000 + 6000 + 12000`), `total_bytes = "36000"`, `gap = true`, `reconciliation_in_progress = false`.

---

## 3. Repeatable AT30 Restore & Reconciliation Checklist (`§17.2`–`§17.4`)

Execute every step in order during a real disaster recovery event or scheduled quarterly restore drill.

### Phase 1: Isolate Central Services and Preserve Out-of-Band State

- [ ] **1.1 Stop `fvpn-control` and `fvpn-worker`** so no new mutations, outbox jobs, or usage ACKs occur against the damaged database `[production only]`:
  ```bash
  systemctl stop fvpn-control.service fvpn-worker.service
  ```
  *(Note: Exit nodes continue forwarding traffic for existing members while their serving leases and member service deadlines remain valid (`86,400s` default max offline lease, `§16.3`), and buffer new usage events in local SQLite (`§10.5`, `§10.8`).)*

- [ ] **1.2 Verify `/var/lib/fvpn/vault` Integrity (`§16.2`, `§17.2`)**:
  Confirm that `/var/lib/fvpn/vault/state.json` and all key files are intact and were **not** rolled back with the database snapshot. Any key revocations or rotations recorded in `state.json` must remain in effect after DB restore.

### Phase 2: Restore PostgreSQL 17 and Freeze Usage Ingestion

- [ ] **2.1 Restore PostgreSQL 17 to the Target Point in Time (`restored_to`)**:
  - **Physical PITR (`pgBackRest`, recommended)** `[production only]`:
    ```bash
    systemctl stop postgresql
    pgbackrest --stanza=fvpn --type=time "--target=2026-10-29 18:02:00+00" --target-action=promote restore
    systemctl start postgresql
    ```
  - **Logical Restore (`pg_restore`)** (`docs/db/README.md`, `F-02`–`F-04`):
    Append-only guards block `DELETE`/`TRUNCATE` on `audit_events`, `usage_events`, `usage_samples`, `usage_checkpoints`, and `usage_rollups` unless the session runs as schema owner (`fvpn_migrator`) with `SET LOCAL fvpn.restore = 'on'` inside the restore transaction `[production only]`:
    ```bash
    pg_restore --single-transaction -U fvpn_migrator -d fvpn /backups/fvpn-logical.dump
    ```

- [ ] **2.2 Verify Schema Migration Status (`fvpn-migrate status`)** `[Verified on dev/test stack]`:
  ```bash
  FVPN_DATABASE_URL="$(cat /etc/fvpn/secrets/database.url)" /usr/local/bin/fvpn-migrate status
  FVPN_DATABASE_URL="$(cat /etc/fvpn/secrets/database.url)" /usr/local/bin/fvpn-migrate up
  ```

- [ ] **2.3 Freeze Usage Ingestion Before Starting `fvpn-control` (`§17.3 Step 1`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    freeze \
    --actor "${ADMIN_ACTOR_UUID}" \
    --reason "PITR restore from snapshot ${RESTORED_TO_UTC}"
  ```
  While `mode = 'frozen'`, `POST /api/v1/agent/usage-batches` returns `503 DEPENDENCY_UNAVAILABLE` (`retryable: true`) and never ACKs events against stale watermarks.

### Phase 3: Rotate `ledger_epoch` and Start Central Services (`§17.3 Steps 2–6`)

- [ ] **3.1 Begin Ledger Recovery (`fvpn-usage-admin begin-recovery`)** `[Verified on dev/test stack]`:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    begin-recovery \
    --actor "${ADMIN_ACTOR_UUID}" \
    --reason "Restore from ${RESTORED_TO_UTC} snapshot" \
    --restored-to "${RESTORED_TO_UTC}" \
    --affected-since "${AFFECTED_SINCE_UTC}"
  ```
  Verify in the JSON output that:
  1. `new_ledger_epoch` differs from `previous_ledger_epoch`.
  2. `mode` is `"reconciling"`.
  3. `signed_record` carries a valid `ed25519` signature from the central `ledger-epoch` key.

- [ ] **3.2 Start `fvpn-control` and `fvpn-worker`** `[production only]`:
  ```bash
  systemctl start fvpn-control.service fvpn-worker.service
  curl -fsS https://vpn.example.com:8443/readyz
  ```

### Phase 4: Reconcile Payments and Member Entitlements (`§9.3`, `§17.4`)

A point-in-time database restore may roll back webhook receipts (`billing_events`) or subscription state transitions that occurred between `restored_to` and the incident.

- [ ] **4.1 Verify Automatic Billing Reconciliation (`billing.reconcile`)**:
  `fvpn-worker` runs the `billing.reconcile` loop every `reconcile_seconds` (`900s` in production config). To trigger or verify reconciliation immediately:
  1. Replay unacknowledged webhook events from the Stripe Dashboard (`Developers -> Webhooks -> Send/Resend`) covering `[restored_to - 1h, now]`. Webhook handling is idempotent by `provider_event_id` and never creates duplicate charges (`BILL01`).
  2. Inspect `billing_subscriptions`, `entitlements`, and `members` for any subscription updated during the RPO window:
     ```sql
     SELECT id, member_id, provider_subscription_id, status, current_period_end, updated_at
     FROM billing_subscriptions
     ORDER BY updated_at DESC LIMIT 50;
     ```
- [ ] **4.2 Re-apply Post-Snapshot Admin Suspensions or Manual Grants (`§9.2`, `§9.3`, `§17.4`)**:
  By design (`internal/billing/service.go`), payment reconciliation updates subscription periods but **never** clears an active administrative suspension (`members.status = 'suspended'`) or overwrites an active manual grant (`entitlement_grants`). If an operator manually suspended a member or issued a grant after `restored_to`, re-apply it via the Admin Portal (`/admin/members/{member_id}`) using a fresh `Idempotency-Key`.

### Phase 5: Reconcile NetBird Overlay, Node PKI, and DNS State (`§5.2`, `§6.4`, `§12.2`, `§17.4`)

Central PostgreSQL is the source of truth for desired state; NetBird and AdGuard Home are downstream enforcement targets (`§3`, `§15`).

- [ ] **5.1 Re-revoke Any Post-Snapshot Node Incarnations (`§6.4`, `§17.2`)**:
  Cross-check node revocation records against out-of-band operator logs. If any node was revoked or retired after `restored_to`, immediately invoke `POST /api/v1/admin/nodes/{node_id}/revoke` (or `DELETE /api/v1/admin/nodes/{node_id}`) so the node's mTLS certificate is revoked and its NetBird peer/route are removed.
- [ ] **5.2 Verify NetBird Node & Member Peer Reconciliation (`node.reconcile`, `members.reconcile`)**:
  `fvpn-worker` continuously runs `node.reconcile` (every `nodes.json:reconcile_seconds`) and `members.devicereconcile` (every `members.json:device_sync_seconds`). It calls `netbird.Client.ReadObservedState` and converges:
  - Any NetBird peer that was authorized in NetBird during the RPO window but is absent from restored `member_devices` or `node_incarnations` is disabled/deleted in NetBird (fail-closed).
  - Any published exit node whose NetBird route or policy drifted is re-converged.
  Verify zero stuck operations in the Admin Portal (`/admin/operations?state=failed`) or via SQL:
  ```sql
  SELECT id, kind, state, target_resource, error_code, updated_at
  FROM operations
  WHERE state IN ('pending', 'running', 'failed', 'partially_succeeded')
  ORDER BY updated_at DESC;
  ```
- [ ] **5.3 Verify AdGuard Home DNS Policy Convergence (`dns.publish`)**:
  Confirm that all AdGuard Home resolvers report the active `domain_rule_set` and `dns_profile` generations in `dns_resolver_states`:
  ```sql
  SELECT resolver_id, applied_generation, last_success_at, last_error_code
  FROM dns_resolver_states;
  ```

### Phase 6: Node Usage Replay and Complete Recovery (`§10.8`, `§17.3 Steps 7–10`)

- [ ] **6.1 Monitor Node Checkpoint Pull & Replay (`§17.3 Steps 7–9`)**:
  As each online exit node attempts `POST /api/v1/agent/usage-batches` with the pre-restore `ledger_epoch`, `fvpn-control` returns `409 LEDGER_EPOCH_MISMATCH` (`current_ledger_epoch` in details). The node automatically:
  1. Calls `GET /api/v1/agent/usage-checkpoint` and verifies the `ed25519` signature using its `recovery` trust bundle.
  2. Replays all locally retained events with `event_seq > highest_contiguous_seq` (plus any `missing_ranges`) under the new `ledger_epoch`. Events already present in the restored snapshot return `status: "duplicate"` and are never double-counted.
  3. Reports any purged sequence ranges in `replay_unavailable` (accepted only while `mode = 'reconciling'`, `D-027`), which inserts zero-byte tombstones and sets `gap = TRUE` on affected member rollups.
  Check stream convergence in PostgreSQL:
  ```sql
  SELECT node_id, incarnation_id, ledger_epoch, highest_contiguous_seq, max_seen_seq, missing_ranges, updated_at
  FROM usage_checkpoints
  ORDER BY updated_at DESC;
  ```

- [ ] **6.2 Complete Ledger Recovery (`fvpn-usage-admin complete-recovery`)** `[Verified on dev/test stack]`:
  Once all active nodes have adopted `new_ledger_epoch` and drained their replay queues, exit reconciling mode:
  ```bash
  /usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
    complete-recovery \
    --actor "${ADMIN_ACTOR_UUID}"
  ```
  This marks the `usage_recovery_records` entry `completed`, clears `reconciliation_in_progress` on `usage_rollups` (preserving `gap = TRUE` where `replay_unavailable` occurred), and sets `usage_ledger_state.mode = 'normal'`.
