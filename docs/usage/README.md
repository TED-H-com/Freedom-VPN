# Central Usage Ledger (`internal/usage`)

This document describes the architecture, ingest algorithm, sequence watermark/ACK rules, query model, and requirement/acceptance test coverage for the Freedom VPN central usage ledger (`§10`, `§15.4`, `§16.4`, `§17.1`, `§17.3`).

Companion documents:
- [`docs/usage/projection.md`](projection.md): Projection receipts, boundary splitting (`§10.8`), rollups, adjustments, and generation rebuilds (`§10.9`).
- [`docs/usage/restore-runbook.md`](restore-runbook.md): Database snapshot restore, `ledger_epoch` rotation, signed checkpoints, replay deduplication, `replay_unavailable` handling (`§17.3`), and partition/retention operations (`§15.4`, `§16.4`).

---

## 1. Architecture & Core Invariants

The central usage ledger (`internal/usage.Service`) is the single source of truth for member bandwidth accounting across all nodes (`§10.1`). It guarantees:

1. **Identity-Bound Stream Attribution (`§10.2`, `§10.5`)**:
   - Every node stream is uniquely keyed by `(node_id, incarnation_id, event_seq)`.
   - The caller passes `NodeIdentity{NodeID, NodeIncarnationID}` extracted from the node's mTLS client certificate (`§7.2`).
   - Any batch whose payload `node_id` or `node_incarnation_id` disagrees with the mTLS identity—or whose incarnation is missing or revoked in `node_incarnations`—is rejected as a whole (`403 FORBIDDEN`) before any ledger row is touched.
2. **Historical Device Binding Attribution (`§10.2`, `§15.2`)**:
   - Usage samples carry `(member_id, device_id, binding_revision, interval_start, interval_end)`.
   - Both the service validator (`validateEventBindingsInTx`) and the database trigger (`check_sample_binding` on `usage_samples`) verify against `device_bindings` history that `(device_id, binding_revision)` belonged to `member_id` throughout `[interval_start, interval_end)`.
   - Usage is **never** attributed by current IP ownership, current exit node, or login location (`§10.2`, `AT19`).
3. **Canonical Payload Integrity (`§10.4`, `§14.1`)**:
   - All byte counters, sequence numbers, revisions, and generations use decimal strings (`pkg/decstr.U64`) and `NUMERIC(20,0)` / `BIGINT` in PostgreSQL, preserving exact precision up to `2^64 - 1` (`18446744073709551615`) without ever passing through `float64` or JavaScript `Number`.
   - Each event's `payload_sha256` is recomputed over the RFC 8785 (JCS) canonical JSON representation of the event with `payload_sha256` set to `""` (`pkg/canonjson`).
4. **Fail-Closed Availability (`§10.6`)**:
   - When the database is unreachable or the ledger is in `frozen` mode (`§17.3`), `IngestBatch` returns `503 DEPENDENCY_UNAVAILABLE` (`retryable: true`) and never advances or returns an ACK.
5. **Database Guard Invariants (`F-02`..`F-15` in `docs/db/README.md`)**:
   - Every accepted event with `record_count > 0` commits its `usage_event_payloads` row and all `record_count` rows in `usage_samples` in the same transaction (`check_usage_event_contents`, `F-04`). Zero-byte `replay_unavailable` tombstones commit with `record_count = 0` and zero `usage_samples` rows, satisfying `check_usage_event_contents` without inventing bytes.
   - Every `usage_checkpoints` write (`ensureAndLockCheckpointInTx`, `recomputeStreamWatermarkWithConflictsInTx`, and single-statement epoch rotation `resetStreamCheckpointForNewEpochInTx`) matches the exact contiguous receipt count and `coalesce(max(event_seq), 0)` in `usage_events` (`check_usage_checkpoint`, `F-04`).
   - Signing purposes (`§17.3`): In Wave 2, both `BeginRecovery` (`usage_recovery_records`) and `GetCheckpoint` (`UsageCheckpointEnvelope`) sign with `envelope.PurposeRecovery` (`"recovery"`). In Wave 3, when the PKI signing provider is wired with the contract revision, the `ledger_epoch` recovery record will use the new `ledger-epoch` purpose while checkpoint envelopes keep `recovery`.

---

## 2. Ingest Algorithm (`§10.6`)

`Service.IngestBatch(ctx, identity, batch)` processes an `api.UsageBatchRequest` in three phases:

### Phase 1: Whole-Batch Gatechecks
1. **Ledger State & Epoch Check**:
   - Reads `usage_ledger_state`. If `mode != 'normal'`, returns `503 SERVICE_UNAVAILABLE` so nodes retain their local SQLite spool and retry after recovery starts or completes.
   - If `batch.LedgerEpoch` is non-zero and does not match the active `current_epoch`, returns `409 LEDGER_EPOCH_MISMATCH`. Nodes must call `GET /api/v1/agent/usage-checkpoint` to synchronize to the new epoch before resuming uploads (`§17.3`).
2. **mTLS Identity & Active Incarnation Check**:
   - Verifies `batch.NodeId == identity.NodeID` and `batch.NodeIncarnationId == identity.NodeIncarnationID`.
   - Verifies that every event in `batch.Events` has `event.NodeId == identity.NodeID` and `event.NodeIncarnationId == identity.NodeIncarnationID`.
   - Queries `nodes` and `node_incarnations` to confirm the node and incarnation exist and are not revoked.
   - If any check fails, the entire batch fails with `403 FORBIDDEN` and zero database writes.

### Phase 2: Per-Event Atomic Transactions (`§10.6`)
Events are processed individually, each inside its own `db.InTx` transaction serialized per stream by locking the stream's `usage_checkpoints` row (`SELECT ... FOR UPDATE`):

1. **Structural, Hash & Historical Binding Validation**:
   - Validates `schema_version == 1`, `event_id != nil`, `event_seq >= 1`, `interval_end > interval_start`, `len(records) > 0` (unless `quality == "gap"`), and `gap != nil` iff `quality == "gap"`.
   - Parses all decimal strings (`event_seq`, `binding_revision`, `upload_bytes`, `download_bytes`, `monotonic_elapsed_ns`, `missing_seq_start`, `missing_seq_end`) via `pkg/decstr.Parse`.
   - Recomputes `payload_sha256` via `ComputeEventHash(ev)` and compares it with `ev.PayloadSha256`.
   - Validates each record's `(member_id, device_id, binding_revision, interval_start, interval_end)` against `device_bindings` and verifies `meter_epoch_id` is non-nil and `metering_version` is non-empty.
   - **On validation failure**: Records the diagnostic details in `usage_event_rejections` (append-only diagnostic log), updates `usage_checkpoints.max_seen_seq` if `event_seq > max_seen_seq`, and returns `status: "rejected"` (`422` field details) for that event only. It does **not** insert into `usage_events`, so `highest_contiguous_seq` cannot advance past `event_seq`.
2. **Idempotency & Conflict Detection**:
   - Queries `usage_events` by `(node_id, incarnation_id, event_seq)` or `id = event.EventId`:
     - **Same key + same `payload_hash`**: Idempotent duplicate resend (`status: "duplicate"`). Ensure projection is complete and recompute stream watermarks cleanly.
     - **Same key + different `payload_hash`** (or same `event_id` attached to a different stream key): Stores a conflict record in `usage_event_conflicts` preserving both the existing payload hash and the conflicting payload + hash (`§10.6`, `AT15`), flags the affected member rollup periods with `gap = TRUE`, and returns `status: "conflict"` with `error_code: "USAGE_EVENT_CONFLICT"`. A conflict on event $k$ **never** deletes or rolls back already-committed or subsequent valid events, and blocks `highest_contiguous_seq` at $k - 1$.
3. **Accept, Sample Insertion, Synchronous Projection & Watermark Update**:
   - Inserts the immutable row into `usage_events` and the canonical JCS payload bytes into `usage_event_payloads`.
   - Inserts each record into `usage_samples`.
   - Runs `projectSamplesInTx` in the **same transaction** (`§10.9`), writing `projection_receipts`, `usage_rollups`, and `usage_rollup_details` (and marking `gap = TRUE` if `ev.Quality == "gap"`).
   - Recomputes `highest_contiguous_seq`, `max_seen_seq`, and `missing_ranges` on `usage_checkpoints`.

### Phase 3: Batch Response Assembly
After all events in the batch have been processed, `IngestBatch` reads the committed checkpoint state and `blocked_seq` and returns `BatchResult` (`api.UsageBatchResponse`).

---

## 3. ACK & Sequence Watermark Rules (`§10.7`) with Worked Examples

For each `(node_id, incarnation_id)` stream, the central ledger tracks:
- `highest_contiguous_seq`: The highest sequence number $C \ge 0$ such that every integer sequence $1, 2, \dots, C$ has a committed row in `usage_events` (or a resolved `replay_unavailable` tombstone) **and** zero unresolved conflicts (`usage_event_conflicts`) or rejections (`usage_event_rejections`).
- `max_seen_seq`: The maximum `event_seq` observed across committed events, conflicts, rejections, and tombstones.
- `missing_ranges`: Exact inclusive intervals `[start_seq, end_seq]` in $(C, \text{max\_seen\_seq}]$ that have no committed `usage_events` row.
- `blocked_seq`: The lowest sequence number $> C$ that is blocking contiguous ACK advancement (either a missing sequence hole, a conflict, or a rejected event).

### Worked Example 1: Out-of-Order Arrival (`100`, `102`, then `101`) — `AT15`
Suppose a node stream has committed events `1..99` (`highest_contiguous_seq = 99`, `max_seen_seq = 99`).

1. **Node sends `[100, 102]`**:
   - Event `100` is committed (`accepted`). Contiguous prefix advances from `99` to `100`.
   - Event `102` is committed (`accepted`). Sequence `101` is missing, so `highest_contiguous_seq` stays `100`, while `max_seen_seq` becomes `102`.
   - Response:
     - `highest_contiguous_seq`: `"100"`
     - `max_seen_seq`: `"102"`
     - `blocked_seq`: `"101"`
     - Checkpoint `missing_ranges`: `[{"start_seq": "101", "end_seq": "101"}]`
   - Node behavior: Deletes local spool up to `100`; retains `101` and `102`.

2. **Node resends `[100, 101, 102]`**:
   - Event `100` matches existing `(node_id, incarnation_id, 100)` and `payload_sha256` $\rightarrow$ `duplicate`.
   - Event `101` is new and valid $\rightarrow$ `accepted`.
   - Event `102` matches existing `(node_id, incarnation_id, 102)` and `payload_sha256` $\rightarrow$ `duplicate`.
   - Now `1..102` are all committed and conflict-free.
   - Response:
     - `highest_contiguous_seq`: `"102"`
     - `max_seen_seq`: `"102"`
     - `blocked_seq`: `nil`
     - Checkpoint `missing_ranges`: `[]`

### Worked Example 2: Conflicting Resend on Sequence `103` — `AT15`
Continuing from `highest_contiguous_seq = 102`:
1. Node sends `[103 (hash A), 104 (hash B)]` $\rightarrow$ both `accepted`; `highest_contiguous_seq = 104`.
2. Suppose instead `103` had not yet arrived, `104` arrived first (`highest_contiguous_seq = 102`, `max_seen_seq = 104`, `blocked_seq = 103`), or suppose `103` is resent with a **different** `payload_sha256` (`hash A' != hash A`):
   - Central records a row in `usage_event_conflicts` with both `existing_payload_hash` (`hash A`) and `conflicting_payload_hash` (`hash A'`) plus the raw conflicting JSON.
   - Central returns `status: "conflict"`, `error_code: "USAGE_EVENT_CONFLICT"` for `103`.
   - `highest_contiguous_seq` is capped at `102` (`blocked_seq = "103"`) so the conflict is never silently ACKed past (`§10.6`).
   - Crucially, event `104` remains committed and projected; the conflict on `103` never deletes later valid events.

### Worked Example 3: Invalid Event in a Mixed Batch (`[201 valid, 202 invalid, 203 valid]`)
Continuing from `highest_contiguous_seq = 200`:
1. Event `201` is valid $\rightarrow$ `accepted`.
2. Event `202` has an invalid `binding_revision` $\rightarrow$ `rejected` with `422` field detail; logged in `usage_event_rejections`.
3. Event `203` is valid $\rightarrow$ `accepted`.
4. Resulting state:
   - `highest_contiguous_seq`: `"201"`
   - `max_seen_seq`: `"203"`
   - `blocked_seq`: `"202"`
   - Checkpoint `missing_ranges`: `[{"start_seq": "202", "end_seq": "202"}]`

---

## 4. Queries & Fair-Use Integration (`§10.9`, `§11.2`)

- **`GetMemberUsage(ctx, memberID, period, granularity, fup)`**:
  - Returns `api.MemberUsage` for a UTC month (`YYYY-MM`) and `month` or `day` granularity from the active `projection_generation`.
  - Applies net `usage_adjustments` (`upload_delta_bytes`, `download_delta_bytes`) onto the `month` total while keeping adjustments in their own append-only table (`§15.4`).
  - Sets `quality` to `"gap"` if any rollup or ledger gap affects the period, `"estimated"` if any boundary sample was proportionally split (`boundary_estimated`), or `"authoritative"` otherwise.
- **`GetMemberUsageSummary(ctx, memberID, period, fup)`**:
  - Lightweight member summary (`api.MemberUsageSummary`) for portal dashboard cards.
- **`GetMemberPeriodTotal(ctx, memberID, period)`**:
  - Consumed by the Fair-Use Policy evaluator (`§11.2` step 1). Returns cross-node `UploadBytes`, `DownloadBytes`, `TotalBytes` (including adjustments), quality flags, and the exact `LedgerWatermark` (`LedgerEpoch`, `ProjectionGeneration`, `AggregateAsOf`, and per-stream `(NodeID, NodeIncarnationID, HighestContiguousSeq, MaxSeenSeq)`) used to compute the total.
- **`GetMemberDrillDown(ctx, memberID, period, granularity)`**:
  - Admin breakdown (`usage_rollup_details`) grouped by `(node_id, device_id, period)` using historical `device_bindings` attribution (`§10.2`, `AT19`).

---

## 5. Requirement & Acceptance Test Coverage

| ID | Spec Section | Requirement / Acceptance Criterion | Implementation & Verification |
|---|---|---|---|
| `MTR01` | `§4`, `§10.1`, `§10.2` | Attribute traffic by historical `(member_id, device_id, binding_revision)`, never current IP or login location | `validateEventBindingsInTx` in [`ingest.go`](../../internal/usage/ingest.go) + DB trigger `check_sample_binding`; verified in `TestAT17_MonthBoundaryLateEventsAndHistoricalBinding` and `TestAT19_MultiNodeAggregationDrillDownAdjustmentsAndRebuild` |
| `MTR02` | `§4`, `§10.5`, `§10.7` | Monotonic `(node_id, incarnation_id, event_seq)` stream with contiguous watermark ACK | `AdvanceContiguous` & `MissingRanges` in [`seq.go`](../../internal/usage/seq.go), `recomputeStreamWatermarkWithConflictsInTx` in [`checkpoint.go`](../../internal/usage/checkpoint.go); verified in `TestAT15_AdvanceContiguousAndMissingRanges` and `TestAT15_DedupeOutOfOrderAckAndConflict` |
| `MTR03` | `§4`, `§10.6` | Idempotent dedupe on same key+hash; conflict preservation (`USAGE_EVENT_CONFLICT`) on different hash; per-event atomicity | `ingestSingleEvent` and `recordConflictInTx` in [`ingest.go`](../../internal/usage/ingest.go); verified in `TestAT15_DedupeOutOfOrderAckAndConflict` and `TestMTR03_ConcurrentSameStreamIngest` |
| `MTR04` | `§4`, `§10.4`, `§14.1` | 64-bit unsigned integer decimal strings (`pkg/decstr.U64`) and 128-bit split math with zero precision loss above `2^53` | [`period.go`](../../internal/usage/period.go), [`ingest.go`](../../internal/usage/ingest.go), [`queries.go`](../../internal/usage/queries.go); verified in `TestMTR04_PrecisionAbove2Pow53EndToEnd` |
| `MTR05` | `§4`, `§10.4` | RFC 8785 JCS canonical JSON payload hash (`payload_sha256`) verification | `ComputeEventHash` in [`signer.go`](../../internal/usage/signer.go); verified against `contracts/vectors/usage_event_hash.json` in `TestMTR05_PayloadSha256GoldenVector` |
| `SEC02` | `§4`, `§16.3` | Privacy: no DNS query names, URLs, destination IPs, or packet contents in usage tables or logs | Schema constraints in `00003_ledger_and_work.sql` & `00130_usage_ledger.sql` store only byte counters, intervals, and UUIDs |
| `AT14` | `§19` | Normal single-stream usage ingest, sample projection, and monthly/daily rollup query | Verified in `TestAT15_DedupeOutOfOrderAckAndConflict` and `TestAT19_MultiNodeAggregationDrillDownAdjustmentsAndRebuild` |
| `AT15` | `§19` | Duplicate resend, out-of-order arrival (`100, 102, 101`), conflict handling (`USAGE_EVENT_CONFLICT`), and partial batch rejection | Verified in `TestAT15_DedupeOutOfOrderAckAndConflict` and `TestAT15_AdvanceContiguousAndMissingRanges` |
| `AT16` | `§19` | Event sequence gap detection and node-reported `quality="gap"` propagation to rollup quality | Verified in `TestAT16_EventSeqContinuityAndNodeReportedGap` |
| `AT17` | `§19` | UTC month/day boundary sample proportional splitting (`boundary_estimated`), conservation of bytes, and late event routing | Verified in `TestAT17_BoundarySplitConservationProperty` and `TestAT17_MonthBoundaryLateEventsAndHistoricalBinding` |
| `AT19` | `§19` | Multi-node usage aggregation, admin node/device drill-down, append-only adjustments, FUP period totals, and projection generation rebuild | Verified in `TestAT19_MultiNodeAggregationDrillDownAdjustmentsAndRebuild` |
| `AT30` | `§19`, `§17.3` | Database snapshot restore, `FreezeIngest`, `BeginRecovery` (`ledger_epoch` bump + signed recovery record), signed `GetCheckpoint`, replay dedupe, `replay_unavailable`, and `CompleteRecovery` | Verified in `TestAT30_RestoreSnapshotRecoveryReplayAndReplayUnavailable` and `Test17_3_CheckpointSigningGoldenVector` |
