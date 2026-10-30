# Usage Ledger & Audit Reader Control and Worker Wiring (`docs/usage/WIRING.md`)

This document describes how the central Usage Ledger (`internal/usage`) and Audit Reader (`internal/audit`) are wired into the central HTTP API (`apps/control/wire_usage.go`), background worker (`apps/worker/jobs_usage.go`), and operator CLI (`internal/usage/cmd/fvpn-usage-admin`) via the composition seam (`docs/control/COMPOSITION.md`, decision D-026), implementing Project Spec v1.0 §10.9, §11.6, §14.2, §14.3, §14.8, §15.4, §16.3, §17.3, and acceptance cases `AT14`–`AT17`, `AT19`, and `AT30`.

---

## 1. Control Domain Composition (`apps/control/wire_usage.go`)

`usage()` (`NewUsageDomain(nil, nil, nil)`) is registered in `apps/control/handlers.go` (`Domains()`) and constructs `UsageHandler` from `control.Dependencies{Config, Pool}`:

| OpenAPI `operationId` | Method & Path | Listener / Auth Role | Headers | Status | Description |
|---|---|---|---|---|---|
| `submitUsageBatch` | `POST /api/v1/agent/usage-batches` | Agent mTLS (`auth.Agent`) | Optional `Idempotency-Key` (UUID) | `200 OK` (`403`, `409`, `422`, `503`) | Extracts `(NodeID, IncarnationID)` from the mTLS principal (`auth.FromContext(ctx)`). When `Idempotency-Key` is supplied, persists the canonical JSON (`pkg/canonjson`, ≤ 1 MiB) of `api.UsageBatchResult` in `usage_batch_replays` (`db/migrations/00131_usage_batch_replay.sql`) inside `httpapi.Transaction(ctx)` keyed by `db.IdempotencyRecord.ID` with the record's `expiry`, and replays the exact `UsageBatchResult` across restarts via `UsageHandler.Replay` (or rejects conflicting payload hashes with `409 IDEMPOTENCY_CONFLICT`). Expired rows are deleted both by the `usage_batch_replays_cleanup_on_idempotency` trigger on `idempotency_requests` and by `UsageHandler.RunCleanup`. Returns `503 DEPENDENCY_UNAVAILABLE` (`retryable = true`) when the ledger is frozen, `409 LEDGER_EPOCH_MISMATCH` on stale `ledger_epoch`, and never ACKs uncommitted events. Enqueues `fup.evaluate` outbox jobs for affected `(member_id, period_id)` pairs. |
| `getUsageCheckpoint` | `GET /api/v1/agent/usage-checkpoint` | Agent mTLS (`auth.Agent`) | — | `200 OK` | Returns the node incarnation's current `api.UsageCheckpointEnvelope` signed with `envelope.PurposeRecovery` (`"recovery"`, §16.3) using the PKI signing provider (`usage.NewProviderSigner`). |
| `getMemberUsage` | `GET /api/v1/member/usage` | Public (`auth.Member`) | Optional `?period_id=YYYY-MM`, `?granularity=month\|day` | `200 OK` | Returns the authenticated member's `api.MemberUsage` combined with their live `FairUseStatus` (`fup.Engine.GetMemberFairUseStatus`). |

> Note: `GET /api/v1/admin/audit-events` (`listAuditEvents`) is mounted via `audit.Store` in `apps/control/server.go` (`core()`). Per decision D-035 and base migration `00006_purge_retained_audit_format.sql`, `internal/audit/audit.go` uses a single canonical format (`resource = <api.ResourceType>:<uuid>`, UUID actor, `result` ∈ `succeeded|failed|denied`) while counting and skipping malformed rows per page (§14.8).

### Transitional `501 Not Implemented` Responses Removed

Wave 3 removes the transitional `501 Not Implemented` fallback for all three usage endpoints:
- `POST /api/v1/agent/usage-batches` (`submitUsageBatch`)
- `GET /api/v1/agent/usage-checkpoint` (`getUsageCheckpoint`)
- `GET /api/v1/member/usage` (`getMemberUsage`)

---

## 2. Signing Purposes (`internal/usage/signer.go`, `internal/signing`)

`usage.NewProviderSigner(p signing.Provider)` bridges the central PKI signing provider to `usage.Signer` with strict purpose separation (§16.3, §17.3):

| Document / Table | `usage.Config` Field | `usage.Signer` Method | `signing.Purpose` | `envelope.Purpose` | Consumers |
|---|---|---|---|---|---|
| `api.UsageCheckpointEnvelope` (`GET /api/v1/agent/usage-checkpoint`) | `Config.Signer` | `SignCheckpoint(ctx, checkpoint)` | `signing.Recovery` | `envelope.PurposeRecovery` (`"recovery"`) | Node agents (`fvpn-agent`, via node trust bundle) |
| `usage_recovery_records.signed_envelope` (`BeginRecovery`) | `Config.LedgerEpochSigner` | `SignRecoveryRecord(ctx, record)` | `signing.LedgerEpoch` | `envelope.PurposeLedgerEpoch` (`"ledger-epoch"`) | Central ledger & audit verification (central-only purpose) |

---

## 3. Worker Domain Composition (`apps/worker/jobs_usage.go`)

`usage()` (`NewUsageDomain(nil)`) is registered in `apps/worker/jobs.go` (`Domains()`) and returns a `worker.Built` whose `Jobs` registry and `Context` decorator (`WithUsage`) wire the following `outbox_events` handler without reading environment variables or opening separate database pools:

| Outbox `event_type` | Producer | Worker Handler Behavior & Evidence Reference |
|---|---|---|
| `usage.projection_rebuild` | Operator / maintenance trigger | Runs `usage.Service.RebuildProjection(ctx)` to rebuild `usage_rollups` and `usage_rollup_details` into a new `projection_generation` from committed `usage_samples` and `usage_adjustments` (§10.9, §15.4), returning evidence `usage_ledger_state:00000000-0000-0000-0000-000000000130`. |

---

## 4. Operator CLI & Admin Operations (`internal/usage/cmd/fvpn-usage-admin`)

Because backup restore/recovery (`FreezeIngest`, `BeginRecovery`, `CompleteRecovery`), admin drill-down (`GetMemberDrillDown`), and ledger adjustments (`CreateAdjustment`) are operator/maintenance workflows outside the 41-endpoint HTTP contract, they are exposed both on `usage.Service` and through the operator CLI `go run ./internal/usage/cmd/fvpn-usage-admin`:

| Subcommand | Service Method | Audit Action & Resource | Description |
|---|---|---|---|
| `freeze --actor <uuid> --reason <text>` | `svc.FreezeIngest` | `usage_ledger.freeze` on `fleet_config:00000000-0000-0000-0000-000000000001` (metadata `usage_ledger_id = 00000000-0000-0000-0000-000000000130`, D-051) | Freezes central usage ingestion (`503 DEPENDENCY_UNAVAILABLE` on `POST /agent/usage-batches`). |
| `begin-recovery --actor <uuid> --reason <text> [--restored-to RFC3339] [--affected-since RFC3339]` | `svc.BeginRecovery` | `usage_ledger.begin_recovery` on `fleet_config:00000000-0000-0000-0000-000000000001` (metadata `usage_ledger_id = 00000000-0000-0000-0000-000000000130`, `recovery_id`, D-051) | Rotates `ledger_epoch`, signs the recovery record with `envelope.PurposeLedgerEpoch` via `Config.LedgerEpochSigner`, recomputes watermarks from committed rows, and marks affected rollups `reconciliation_in_progress = true`. |
| `complete-recovery --actor <uuid>` | `svc.CompleteRecovery` | `usage_ledger.complete_recovery` on `fleet_config:00000000-0000-0000-0000-000000000001` (metadata `usage_ledger_id = 00000000-0000-0000-0000-000000000130`, `recovery_id`, D-051) | Clears `reconciliation_in_progress` and returns the ledger to `normal` mode. |
| `adjust --member-id <uuid> --period YYYY-MM --upload-bytes <int64> --download-bytes <int64> --actor <uuid> --reason <text>` | `svc.CreateAdjustment` | `usage.adjustment.create` on `member:<member_id>` | Appends an immutable row in `usage_adjustments`, updates rollups, and (when net bytes `< 0`) enqueues `fup.evaluate` with `allow_relaxation = true`. |
| `drill-down --member-id <uuid> --period YYYY-MM` | `svc.GetMemberDrillDown` | Read-only | Outputs JSON breakdown by node, device, `(node, device)`, and adjustments. |
| `rebuild-projection` | `svc.RebuildProjection` | Read/projection switch | Rebuilds `member_usage` rollups into `active_projection_generation + 1`. |

---

## 5. Configuration (`internal/config/usage.go`, Decision D-045)

Production composition in `fvpn-control`, `fvpn-worker`, and `fvpn-usage-admin` loads the usage configuration file referenced by `Dependencies.Config.UsageConfigFile` (`"usage_config_file"` in central `config.json`) via `config.LoadUsage(path)` and validates central bounds via `config.ValidateUsageCentralBounds(d.Config)`. `signing_config_file` inside the usage configuration file is required and opened with `signing.OpenFileProvider`; there are no compiled-in signing keys or fallback signers (`UsageDefaults`, `UsageFromConfig`, `defaultTestRecoverySigner`, and `defaultTestLedgerEpochSigner` have been removed). A missing or unreadable `usage_config_file` or `signing_config_file` fails startup with `&config.SettingError{Name: "usage_config_file"}` or `&config.SettingError{Name: "signing_config_file"}`. Composition tests use `internal/usage/usagetest.New(t)` under `t.TempDir()`.

| JSON Field / Setting Name | Type | Requirement | Description |
|---|---|---|---|
| `signing_config_file` | `string` (non-empty path) | Required | Path to the PKI signing vault directory opened with `internal/signing.OpenFileProvider` for `signing.Recovery` (`"recovery"`) and `signing.LedgerEpoch` (`"ledger-epoch"`). |
| `raw_usage_retention_days` | `int` (`1..3650`) | Required | Retention window in days for raw `usage_event_payloads` (§15.4). |
| `rollup_retention_months` | `int` (`1..120`) | Required | Retention window in months for `usage_rollups` and `usage_rollup_details` (§15.4). |
| `audit_retention_days` | `int` (`1..3650`) | Required | Retention window in days for `audit_events` (§15.4). |
| `projection_batch_size` | `int` (`1..10000`) | Required | Batch size for usage projection rebuild/processing. |
