# Usage Projection, Boundary Splitting, and Rollup Rebuilds

This document details the projection architecture in [`internal/usage/projection.go`](../../internal/usage/projection.go) and [`internal/usage/period.go`](../../internal/usage/period.go), implementing `§10.8`, `§10.9`, and `§15.4` of the Project Specification.

---

## 1. Synchronous In-Transaction Projection (`§10.6`, `§10.9`)

The specification (`§10.6`, `§10.9`) permits running projection either synchronously inside the per-event ingest transaction or as an asynchronous worker job with a checkpoint.

### Architectural Choice: Synchronous In-Transaction Projection
`internal/usage` executes `projectSamplesInTx` **inside the same database transaction** that commits each accepted `usage_events` row and its `usage_samples` rows, paired with `RebuildProjection` for offline/administrative generation rebuilds.

### Justification
1. **Zero Lag Between ACK and Fair-Use Evaluation (`§11.2`)**:
   - As soon as `IngestBatch` ACKs an event, its bytes are immediately visible in `usage_rollups`, `usage_rollup_details`, and `GetMemberPeriodTotal`. The Fair-Use Policy evaluator (`§11.2` step 1) never reads stale rollups behind an already-advanced `usage_checkpoints` watermark.
2. **Atomic Receipt + Rollup Increment (`§10.9`, `§15.4`)**:
   - Every sample segment writes a receipt row into `projection_receipts` with `UNIQUE (sample_id, projection_name, generation, period)` (`ON CONFLICT DO NOTHING`) and increments `usage_rollups` and `usage_rollup_details` in the same transaction only when the receipt was newly inserted.
   - If `ingestSingleEvent` sees a duplicate resend or if `RebuildProjection` is invoked, no sample segment can ever be counted twice in the same `(projection_name, generation, period)`.
3. **Retention Guard Compatibility (`§16.4`)**:
   - The database guard `purge_retained('usage_event_payloads', ...)` (`00004_guards.sql`) verifies that every sample of an event has `projection_receipts` and `usage_rollups` whose `[period_start, period_end)` union covers `[interval_start, interval_end)`. Synchronous projection ensures this invariant holds immediately upon commit.

---

## 2. Boundary Sample Splitting (`§10.8`, `AT17`)

Nodes normally flush usage samples at UTC day and month boundaries (`00:00:00Z`). If a node clock skew, temporary stall, or agent restart causes a sample interval `[interval_start, interval_end)` to cross one or more UTC day (`YYYY-MM-DD`) or UTC month (`YYYY-MM`) boundaries, `Split(start, end, upload, download, granularity)` splits the sample proportionally by overlap duration using exact 128-bit unsigned integer arithmetic (`math/big` over `uint64` inputs).

### Algorithm
Given a sample over $[S, E)$ with total duration $D = E - S > 0$ (in nanoseconds) and total bytes $B \in [0, 2^{64}-1]$:
1. Enumerate all contiguous calendar windows $W_1, W_2, \dots, W_k$ (for `month` or `day` granularity in UTC) that overlap $[S, E)$, with overlap durations $d_1, d_2, \dots, d_k$ where $\sum_{i=1}^k d_i = D$.
2. If $k == 1$ (no boundary crossed):
   - Return a single segment with `UploadBytes = upload`, `DownloadBytes = download`, `BoundaryEstimated = false`.
3. If $k > 1$ (boundary crossed):
   - For each earlier segment $i \in \{1, \dots, k-1\}$:
     $$b_i = \left\lfloor \frac{B \times d_i}{D} \right\rfloor$$
     computed with 128-bit intermediate product $B \times d_i$ so `uint64` values above $2^{53}$ never overflow.
   - For the final segment $k$:
     $$b_k = B - \sum_{i=1}^{k-1} b_i$$
     assigning any integer division remainder to the latest segment (`§10.8`).
   - Mark all $k$ segments with `BoundaryEstimated = true`, which sets `boundary_estimated = TRUE` on the affected `usage_rollups` and `usage_rollup_details` rows and surfaces `quality: "estimated"` (unless superseded by `"gap"`).

### Conservation Guarantee
For every sample and every granularity (`month` and `day`):
$$\sum_{i=1}^k \text{upload}_i = \text{upload}, \qquad \sum_{i=1}^k \text{download}_i = \text{download}$$
This invariant is enforced by `TestAT17_BoundarySplitConservationProperty` and verified during `RebuildProjection`.

---

## 3. Late Events (`§10.8`, `AT17`)

Events are always projected into the period(s) of their own sample `[interval_start, interval_end)` timestamps, **never** the wall-clock time at which the central API received the batch (`ingested_at`).
- If a node reconnects on `2026-11-02` and uploads buffered samples from `2026-10-31T22:00:00Z..2026-10-31T23:00:00Z`, those bytes are credited to `2026-10` (and day `2026-10-31`), leaving `2026-11` untouched.
- `usage_rollups.last_observed_at` advances to `MAX(last_observed_at, interval_end)` and `aggregate_as_of` records the projection timestamp.

---

## 4. Append-Only Adjustments (`§15.4`, `AT19`)

Manual or automated billing/traffic corrections are stored in `usage_adjustments`:
- Columns: `id`, `member_id`, `period` (`YYYY-MM`), `upload_delta_bytes`, `download_delta_bytes`, `reason`, `actor_id`, `created_at`.
- Protected by `guard_append_only()` triggers (`00004_guards.sql`)—rows can never be updated or deleted.
- Kept strictly separate from payment refunds (`billing_records`).
- When querying `GetMemberUsage` (for `granularity = "month"`), `GetMemberUsageSummary`, or `GetMemberPeriodTotal`, net adjustments for `(member_id, period)` are added to the projected rollup totals (clamped at 0 if a negative adjustment exceeds raw usage) and surfaced in `AdjustmentsTotalBytes`.

---

## 5. Rebuilding a Projection Generation (`§10.9`, `AT19`)

When projection logic is upgraded or audited, `Service.RebuildProjection(ctx, targetGeneration)` builds a fresh generation without mutating or re-adding into the previous generation:

1. **Create Candidate Generation**:
   - Inserts `targetGeneration` into `usage_projection_generations` with `status = 'building'`.
2. **Re-project All Committed Events**:
   - Scans all committed `usage_events` and their `usage_samples` in deterministic order.
   - Calls `projectSamplesInTx` for `targetGeneration`. Each sample segment writes a receipt `(sample_id, 'member_usage', targetGeneration, period)` and populates `usage_rollups` and `usage_rollup_details` under `projection_generation = targetGeneration`.
   - Propagates any event-level gaps (`quality = 'gap'`), conflict gaps (`usage_event_conflicts`), and ledger gaps (`usage_ledger_gaps`, e.g., `replay_unavailable`) to `targetGeneration` rollups.
   - Preserves `reconciliation_in_progress` flags if the ledger is currently reconciling.
3. **Conservation Verification**:
   - Verifies that the sum of `(upload_bytes + download_bytes)` across all `month` rollups in `targetGeneration` equals the sum across `day` rollups in `targetGeneration` and matches the total sample bytes in `usage_samples`.
4. **Atomic Read-Pointer Switch**:
   - Marks the previous active generation as `'superseded'`, marks `targetGeneration` as `'active'`, and updates `usage_ledger_state.active_projection_generation = targetGeneration` in the same transaction.
   - All subsequent read queries (`GetMemberUsage`, `GetMemberUsageSummary`, `GetMemberPeriodTotal`, `GetMemberDrillDown`) read exclusively from `targetGeneration`.
