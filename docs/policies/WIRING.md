# Policies & Fair-Use Control and Worker Wiring (`docs/policies/WIRING.md`)

This document describes how the versioned Policies store (`internal/policies`) and Fair-Use engine (`internal/fup`) are wired into the central HTTP API (`apps/control/wire_policies.go`) and background worker (`apps/worker/jobs_policies.go`), implementing Project Spec v1.0 §11, §12.4, §14.2, §14.5, §15.4, and acceptance cases `AT18`, `AT19`, and `AT27`.

---

## 1. HTTP Operations Mounted (`apps/control/wire_policies.go`)

`wirePolicies(pool)` is composed into `apps/control/handlers.go` via `httpapi.Compose(...)` and mounts the following OpenAPI operations:

| OpenAPI `operationId` | Method & Path | Auth Role | Headers & Concurrency | Status | Description |
|---|---|---|---|---|---|
| `listPolicies` | `GET /api/v1/admin/policies` | `admin` | Optional `?type=`, `?cursor=`, `?limit=` | `200 OK` | Lists policy definitions with active revision metadata, generation, resource version, and recent revisions. |
| `createPolicyRevision` | `POST /api/v1/admin/policies` | `admin` (MFA) | Required `Idempotency-Key` (UUID) | `201 Created` | Validates and persists an immutable policy revision (`plan`, `fair_use`, `dns_profile`, `domain_rule_set`, `threat_feed`, `release_support`) inside `httpapi.Transaction(ctx)` with `httpapi.SetResult` and `Replay` (`policies.commandResult`). |
| `activatePolicyRevision` | `POST /api/v1/admin/policies/{policy_id}/activate` | `admin` (MFA) | Required `Idempotency-Key` (UUID), required `If-Match` (`ETag` / `resource_version`, `428` when missing, `412` on mismatch) | `202 Accepted` | Atomically updates `policy_definitions.active_revision_id` and increments `generation`, runs type-specific activation hooks (`plans` update, `dnspolicy` bundle compilation, or `releases.ApplySupportTx` for `release_support`), writes an `audit_events` row (`policy.activate`), enqueues `policy.activated` in `outbox_events`, and returns a `pending` `api.Operation` (`kind = "policy.activate"`) with `ETag` header. |

### Transitional `501 Not Implemented` Responses Removed

Wave 3 removes the transitional `501 Not Implemented` fallback for all three policy endpoints:
- `GET /api/v1/admin/policies` (`listPolicies`)
- `POST /api/v1/admin/policies` (`createPolicyRevision`)
- `POST /api/v1/admin/policies/{policy_id}/activate` (`activatePolicyRevision`)

---

## 2. Worker Outbox Jobs Registered (`apps/worker/jobs_policies.go`)

`registerPoliciesJobs` (and `RegisterPoliciesJobsWithPool(r, pool)`) registers the following `outbox_events` handlers on the background worker (`apps/worker/jobs.go`):

| Outbox `event_type` | Producer | Worker Handler Behavior & Evidence Reference |
|---|---|---|
| `policy.activated` | `policies.Store.ActivateRevision` | Reads back committed `policy_definitions` state (`active_revision_id`, `generation`). When `type` is `fair_use` or `plan`, schedules a coalesced Fair-Use sweep across active members (`fup.Worker.ScheduleMemberSweep` + `Flush`). Transitions the `policy.activate` operation's `desired_state_compile` step to `succeeded` with read-back evidence `policy_definition:<definition_id>`. |
| `fup.evaluate` | `usage.Service.IngestBatch` / `usage.Service.CreateAdjustment` (`jobs_usage.go`) | Runs `fup.Worker` (`NotifyLedgerChange` or `NotifyLedgerCorrection` when `allow_relaxation = true`) and `Flush` against `usage.Service.GetMemberPeriodTotal`, recording the evaluation's `LedgerWatermark` (`ledger_epoch`, `projection_generation`, `aggregate_as_of`, per-stream watermarks). Returns evidence `fup_evaluation:<evaluation_id>` (or `member:<member_id>`). |
| `fup.member_generation_changed` | `fup.Engine.Compute` (in `enforce` mode when enforced tier/rate/override changes) | Reads back `fup_member_states` for the aggregate ID and returns evidence `fup_member_state:<state_id>` so downstream desired-state compilation can observe the member's new rate policy generation. |
| `fleet.release_support_changed` | `releases.Service.ApplySupportTx` (invoked inside `ActivateRevision` for `release_support`) | Reads back `fleet_config` (`00000000-0000-0000-0000-000000000001`) and returns evidence `fleet_config:<id>`. |

### Exported Desired-State Rate Policy Helper

`fup.BuildNodeMemberRatePolicy` remains exported with its contract signature unchanged for desired-state compilation:

```go
func BuildNodeMemberRatePolicy(eval *EvaluationResult, planRate, nodeCap *api.RatePair) api.MemberRatePolicy
```

---

## 3. Configuration Keys (`internal/config/policies.go`)

Loaded via `config.LoadPoliciesFromEnv()`:

| Environment Variable | Type | Default | Description |
|---|---|---|---|
| `FVPN_FUP_MODE` | `shadow \| enforce` (or empty) | `""` (uses active `fair_use` policy revision's `mode`) | Optional operator override for Fair-Use evaluation mode (`shadow` vs. `enforce`, §11.2). |
| `FVPN_POLICIES_SIGNING_KEY_REF` | `string` | `"central-policy-key"` | Default key reference recorded on `policy_revisions.signing_key_ref` when not explicitly supplied. |
| `FVPN_FUP_COALESCE_INTERVAL` | `time.Duration` (`> 0`) | `5s` | Coalescing window for `fup.Worker` member evaluation sweeps. |
