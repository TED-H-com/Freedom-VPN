# Usage Ledger & Audit Reader Control and Worker Wiring (`docs/usage/WIRING.md`)

This document describes how the central Usage Ledger (`internal/usage`) and Audit Reader (`internal/audit`) are wired into the central HTTP API (`apps/control/wire_usage.go`), background worker (`apps/worker/jobs_usage.go`), and operator CLI (`internal/usage/cmd/fvpn-usage-admin`), implementing Project Spec v1.0 §10.9, §11.6, §14.2, §14.3, §14.8, §15.4, §16.3, §17.3, and acceptance cases `AT14`–`AT17`, `AT19`, and `AT30`.

---

## 1. HTTP Operations Mounted (`apps/control/wire_usage.go`)

`wireUsage(pool)` is composed into `apps/control/handlers.go` via `httpapi.Compose(...)` and mounts the following OpenAPI operations:

| OpenAPI `operationId` | Method & Path | Listener / Auth Role | Headers | Status | Description |
|---|---|---|---|---|---|
| `submitUsageBatch` | `POST /api/v1/agent/usage-batches` | Agent mTLS (`auth.Agent`) | Optional `Idempotency-Key` (UUID) | `200 OK` (`403`, `409`, `422`, `503`) | Extracts `(NodeID, IncarnationID)` from the mTLS principal (`httpapi.Principal(ctx)`). When `Idempotency-Key` is supplied, deduplicates/replays (`200 OK`) or rejects conflicting payload hashes (`409 IDEMPOTENCY_CONFLICT`) in `idempotency_keys` (`scope = agent:submitUsageBatch:<node_id>:<incarnation_id>`). Returns `503 DEPENDENCY_UNAVAILABLE` (`retryable = true`) when the ledger is frozen, `409 LEDGER_EPOCH_MISMATCH` on stale `ledger_epoch`, and never ACKs uncommitted events. Enqueues `fup.evaluate` outbox jobs for affected `(member_id, period_id)` pairs. |
| `getUsageCheckpoint` | `GET /api/v1/agent/usage-checkpoint` | Agent mTLS (`auth.Agent`) | — | `200 OK` | Returns the node incarnation's current `api.UsageCheckpointEnvelope` signed with `envelope.PurposeRecovery` (`"recovery"`, §16.3) using the PKI signing provider (`usage.NewProviderSigner`). |
| `getMemberUsage` | `GET /api/v1/member/usage` | Public (`auth.Member`) | Optional `?period_id=YYYY-MM`, `?granularity=month\|day` | `200 OK` | Returns the authenticated member's `api.MemberUsage` combined with their live `FairUseStatus` (`fup.Engine.GetMemberFairUseStatus`). |

> Note: `GET /api/v1/admin/audit-events` (`listAuditEvents`) is mounted via `audit.Store` in `apps/control/handlers.go`, with `internal/audit/audit.go` updated to decode `purge_retained` JSON `ResourceRef` payloads (`{"type":"fleet_config","id":"00000000-0000-0000-0000-000000000001"}`) and map non-UUID system database roles (`session_user` when `actor_type == "system"`) to `actor.id = null` while counting and skipping malformed rows per page (§14.8).

### Transitional `501 Not Implemented` Responses Removed

Wave 3 removes the transitional `501 Not Implemented` fallback for all three usage endpoints:
- `POST /api/v1/agent/usage-batches` (`submitUsageBatch`)
- `GET /api/v1/agent/usage-checkpoint` (`getUsageCheckpoint`)
- `GET /api/v1/member/usage` (`getMemberUsage`)

---

## 2. Signing Purposes (`internal/usage/signer.go`, `internal/signing`)

`usage.NewProviderSigner(p signing.Provider)` bridges the central PKI signing provider to `usage.Signer` with strict purpose separation (§16.3, §17.3):

| Document / Table | `usage.Signer` Method | `signing.Purpose` | `envelope.Purpose` | Consumers |
|---|---|---|---|---|
| `api.UsageCheckpointEnvelope` (`GET /api/v1/agent/usage-checkpoint`) | `SignCheckpoint(ctx, checkpoint)` | `signing.Recovery` | `envelope.PurposeRecovery` (`"recovery"`) | Node agents (`fvpn-agent`, via node trust bundle) |
| `usage_recovery_records.signed_envelope` (`BeginRecovery`) | `SignRecoveryRecord(ctx, record)` | `signing.LedgerEpoch` | `envelope.PurposeLedgerEpoch` (`"ledger-epoch"`) | Central ledger & audit verification (central-only purpose) |

---

## 3. Worker Outbox Jobs Registered (`apps/worker/jobs_usage.go`)

`registerUsageJobs` (and `RegisterUsageJobsWithPool(r, pool)`) registers the following `outbox_events` handler on the background worker (`apps/worker/jobs.go`):

| Outbox `event_type` | Producer | Worker Handler Behavior & Evidence Reference |
|---|---|---|
| `usage.projection_rebuild` | Operator / maintenance trigger | Runs `usage.Service.RebuildProjection(ctx)` to rebuild `usage_rollups` and `usage_rollup_details` into a new `projection_generation` from committed `usage_samples` and `usage_adjustments` (§10.9, §15.4), returning evidence `usage_ledger_state:00000000-0000-0000-0000-000000000130`. |

---

## 4. Operator CLI & Admin Operations (`internal/usage/cmd/fvpn-usage-admin`)

Because backup restore/recovery (`FreezeIngest`, `BeginRecovery`, `CompleteRecovery`), admin drill-down (`GetMemberDrillDown`), and ledger adjustments (`CreateAdjustment`) are operator/maintenance workflows outside the 41-endpoint HTTP contract, they are exposed both on `usage.Service` and through the operator CLI `go run ./internal/usage/cmd/fvpn-usage-admin`:

| Subcommand | Service Method | Audit Action & Resource | Description |
|---|---|---|---|
| `freeze --actor <uuid> --reason <text>` | `svc.FreezeIngest` | `usage_ledger.freeze` on `fleet_config:00000000-0000-0000-0000-000000000130` | Freezes central usage ingestion (`503 DEPENDENCY_UNAVAILABLE` on `POST /agent/usage-batches`). |
| `begin-recovery --actor <uuid> --reason <text> [--restored-to RFC3339] [--affected-since RFC3339]` | `svc.BeginRecovery` | `usage_ledger.begin_recovery` on `fleet_config:00000000-0000-0000-0000-000000000130` | Rotates `ledger_epoch`, signs the recovery record with `envelope.PurposeLedgerEpoch`, recomputes watermarks from committed rows, and marks affected rollups `reconciliation_in_progress = true`. |
| `complete-recovery --actor <uuid>` | `svc.CompleteRecovery` | `usage_ledger.complete_recovery` on `fleet_config:00000000-0000-0000-0000-000000000130` | Clears `reconciliation_in_progress` and returns the ledger to `normal` mode. |
| `adjust --member-id <uuid> --period YYYY-MM --upload-bytes <int64> --download-bytes <int64> --actor <uuid> --reason <text>` | `svc.CreateAdjustment` | `usage.adjustment.create` on `member:<member_id>` | Appends an immutable row in `usage_adjustments`, updates rollups, and (when net bytes `< 0`) enqueues `fup.evaluate` with `allow_relaxation = true`. |
| `drill-down --member-id <uuid> --period YYYY-MM` | `svc.GetMemberDrillDown` | Read-only | Outputs JSON breakdown by node, device, `(node, device)`, and adjustments. |
| `rebuild-projection` | `svc.RebuildProjection` | Read/projection switch | Rebuilds `member_usage` rollups into `active_projection_generation + 1`. |

---

## 5. Configuration Keys (`internal/config/usage.go`)

Loaded via `config.LoadUsageFromEnv()`:

| Environment Variable | Type | Default | Description |
|---|---|---|---|
| `FVPN_SIGNING_CONFIG_FILE` | `string` (path) | `""` | Path to the PKI signing vault directory/config (`internal/signing.OpenFileProvider`) used for `PurposeRecovery` and `PurposeLedgerEpoch`. |
| `FVPN_USAGE_IDEMPOTENCY_RETENTION` | `time.Duration` (`> 0`) | `24h` | Retention TTL for optional `Idempotency-Key` records on `POST /api/v1/agent/usage-batches`. |
