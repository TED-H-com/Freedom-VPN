# Usage Ledger Restore, Replay & Partition Runbook

This runbook documents the database snapshot restore and node replay protocol (`§17.3`, `AT30`) as well as the partitioning and retention procedures (`§15.4`, `§16.4`) for the central usage ledger.

---

## 1. Why Special Restore Handling Is Required (`§17.3`)

If PostgreSQL is restored from a backup or point-in-time snapshot taken at $T_{\text{snap}}$, the restored database may have a lower `highest_contiguous_seq` for a node stream than what the central service had ACKed to the node between $T_{\text{snap}}$ and the incident.
- **Within a single `ledger_epoch`**, a node's ACK watermark (`highest_contiguous_seq`) is strictly non-decreasing (`usage_checkpoints_monotonic` trigger in `00004_guards.sql`).
- **Across a snapshot restore**, nodes might hold ACK watermarks from the lost window $[T_{\text{snap}}, T_{\text{incident}}]$ or might already have pruned spool entries below their old ACK watermark.
- Rotating to a new **`ledger_epoch`** with an Ed25519-signed recovery record authorizes nodes to reset their upload cursor to the restored database's committed watermark, replay retained local spool entries through normal deduplication, and explicitly report any pruned ranges via `replay_unavailable`.
- **Signing Purposes (Wave 2 vs. Wave 3 PKI Wiring)**:
  - In Wave 2, both the `ledger_epoch` recovery record (`BeginRecovery` / `usage_recovery_records`) and the signed stream checkpoint (`GetCheckpoint` / `UsageCheckpointEnvelope`) use `envelope.PurposeRecovery` (`"recovery"`).
  - In Wave 3, when the PKI signing provider is wired with the contract revision, the `ledger_epoch` recovery record will use the new `ledger-epoch` purpose that the contract revision adds, while checkpoint envelopes keep `recovery`.

---

## 2. Step-by-Step Restore & Replay Procedure (`§17.3`, `AT30`)

### Step 1: Freeze Ingest (`FreezeIngest`)
Before or immediately upon bringing up the restored database, freeze usage ingestion so nodes receive `503 SERVICE_UNAVAILABLE` (`retryable: true`) and do not advance watermarks or prune local spools:
```go
err := usageSvc.FreezeIngest(ctx)
```
- Sets `usage_ledger_state.mode = 'frozen'`.

### Step 2: Enter Recovery Mode & Rotate `ledger_epoch` (`BeginRecovery`)
Invoke `BeginRecovery` with the operator reason and actor ID:
```go
rec, err := usageSvc.BeginRecovery(ctx, "Restored PostgreSQL from 02:00Z snapshot", "admin:ops-oncall")
```
`BeginRecovery` executes a single atomic transaction (`fvpn.recovery_recompute = 'on'`) that:
1. Allocates `new_epoch = current_epoch + 1` in `usage_ledger_epochs`.
2. Scans every active `(node_id, incarnation_id)` stream and recomputes `highest_contiguous_seq`, `max_seen_seq`, and `missing_ranges` strictly from the committed `usage_events` rows present in the restored database.
3. Updates `usage_checkpoints` to `ledger_epoch = new_epoch` with the recomputed watermarks.
4. Marks all existing `usage_rollups` rows with `reconciliation_in_progress = TRUE` so member and admin queries clearly indicate that post-restore replay is underway (`§17.3`).
5. Signs the `RecoveryRecord` (`ledger_epoch`, `previous_ledger_epoch`, `reason`, `triggered_by`, `stream_count`, `created_at`) with the `recovery` Ed25519 key (`pkg/envelope`) and stores the signed envelope in `usage_recovery_records`.
6. Transitions `usage_ledger_state.mode` to `'reconciling'` (and subsequently allows nodes to fetch checkpoints and replay batches under `new_epoch`).

### Step 3: Node Checkpoint Synchronization (`GetCheckpoint`)
When a node attempts to upload a batch with the old `ledger_epoch`, `IngestBatch` rejects it with `409 LEDGER_EPOCH_MISMATCH`.
The node then calls `GET /api/v1/agent/usage-checkpoint` (`Service.GetCheckpoint`), which returns an `api.UsageCheckpointEnvelope` signed with `purpose = "recovery"`:
```json
{
  "signed": {
    "node_id": "...",
    "node_incarnation_id": "...",
    "ledger_epoch": "2",
    "highest_contiguous_seq": "50",
    "max_seen_seq": "50",
    "missing_ranges": [],
    "reconciliation_in_progress": true,
    "as_of": "2026-10-28T15:00:00Z"
  },
  "signatures": [
    {
      "key_id": "recovery-key-2026",
      "algorithm": "ed25519",
      "signature_b64": "..."
    }
  ]
}
```
The node verifies the `recovery` signature against its local trust store before accepting the lower `highest_contiguous_seq` (`50`) under the new `ledger_epoch` (`2`).

### Step 4: Replay Retained Events & Report `replay_unavailable` (`IngestBatch` / `RecordReplayUnavailable`)
1. **Retained spool events** (e.g., sequences `71..100` still in the node's SQLite retention buffer) are uploaded via `POST /api/v1/agent/usage-batches` with `ledger_epoch: "2"` and processed through normal idempotent deduplication (`accepted` if missing in the snapshot, `duplicate` if already present).
2. **Pruned spool ranges** (e.g., sequences `51..70` that the node had already deleted after receiving an ACK prior to the crash, and whose retention window expired) are reported in `batch.ReplayUnavailable`:
   ```json
   {
     "start_seq": "51",
     "end_seq": "70",
     "affected_months": ["2026-10"],
     "reason": "local spool pruned after prior epoch ACK"
   }
   ```
   For each `replay_unavailable` range, the central ledger:
   - Inserts an audit record into `usage_ledger_gaps` with `source = 'replay_unavailable'`, `start_seq`, `end_seq`, and `affected_months`.
   - Inserts explicit 0-byte tombstone rows (`record_count = 0`, deterministic tombstone hash) into `usage_events` for the unrecoverable sequence numbers `[51..70]` so `highest_contiguous_seq` can advance past the permanently lost range to `100` without inventing any bytes (`§17.3`).
   - Marks `gap = TRUE` on all `usage_rollups` for the node's members in `affected_months` (updating or creating 0-byte gap rollup markers if needed, and **never** overwriting the ledger with monthly summary totals).

### Step 5: Complete Recovery (`CompleteRecovery`)
Once all active nodes have synchronized to the new `ledger_epoch` and replayed their retained events:
```go
err := usageSvc.CompleteRecovery(ctx)
```
- Clears `reconciliation_in_progress = FALSE` on `usage_rollups` while preserving `gap = TRUE` on any periods affected by `replay_unavailable` or event gaps.
- Sets `usage_ledger_state.mode = 'normal'` and records `completed_at` on `usage_ledger_epochs`.

---

## 3. Partitioning & Retention Operations (`§15.4`, `§16.4`)

| Table | Retention Policy (`§16.4`) | Mutability & Purge Mechanism |
|---|---|---|
| `usage_events` | Permanent ledger header (`node_id`, `incarnation_id`, `event_seq`, `payload_hash`, `record_count`, `ingested_at`) | Append-only (`guard_append_only` trigger). Monthly range partitioning by `ingested_at` (`YYYY-MM`) in production deployments. |
| `usage_samples` | Permanent sample attribution (`member_id`, `device_id`, `binding_revision`, `[interval_start, interval_end)`, `upload_bytes`, `download_bytes`) | Append-only (`guard_append_only` trigger). Monthly range partitioning by `interval_start` (`YYYY-MM`) in production deployments. |
| `usage_event_payloads` | Raw canonical JCS JSON retained for `retention_config.raw_usage_days` (default `>= 35` days) | Append-only; purged **only** via `SELECT purge_retained('usage_event_payloads', cutoff, reason)`, which enforces that every sample in the event is fully covered by `projection_receipts` and `usage_rollups` (`00004_guards.sql`, `Test16_4_PayloadPurgeRequiresWholeProjection`). |
| `usage_rollups` / `usage_rollup_details` | Retained for `retention_config.rollup_months` (default `>= 24` months) | Upserted atomically with `projection_receipts`; purged after `rollup_months` via `SELECT purge_retained('usage_rollups', cutoff, reason)`. |
| `usage_adjustments` | Permanent audit trail of manual/automated corrections | Strictly append-only (`guard_append_only` trigger). |
