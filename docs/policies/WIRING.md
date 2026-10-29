# Policies & Fair-Use Control and Worker Wiring (`docs/policies/WIRING.md`)

This document describes how the versioned Policies store (`internal/policies`) and Fair-Use engine (`internal/fup`) are wired into the central HTTP API (`apps/control/wire_policies.go`) and background worker (`apps/worker/jobs_policies.go`) via the composition seam (`docs/control/COMPOSITION.md`, decision D-026), implementing Project Spec v1.0 §11, §12.4, §14.2, §14.5, §15.4, and acceptance cases `AT18`, `AT19`, and `AT27`.

---

## 1. Control Domain Composition (`apps/control/wire_policies.go`)

`policies()` (`NewPoliciesDomain(nil)`) is registered in `apps/control/handlers.go` (`Domains()`) and constructs `PoliciesHandler` from `control.Dependencies{Config, Pool}`:

| OpenAPI `operationId` | Method & Path | Auth Role | Headers & Concurrency | Status | Description |
|---|---|---|---|---|---|
| `listPolicies` | `GET /api/v1/admin/policies` | `admin` | Optional `?type=`, `?cursor=`, `?limit=` | `200 OK` | Lists policy definitions with active revision metadata, generation, resource version, and recent revisions. |
| `createPolicyRevision` | `POST /api/v1/admin/policies` | `admin` (MFA) | Required `Idempotency-Key` (UUID) | `201 Created` | Validates and persists an immutable policy revision (`plan`, `fair_use`, `dns_profile`, `domain_rule_set`, `threat_feed`, `release_support`) inside `httpapi.Transaction(ctx)` with `httpapi.SetResult` and `Replay` (`policies.commandResult`). |
| `activatePolicyRevision` | `POST /api/v1/admin/policies/{policy_id}/activate` | `admin` (MFA) | Required `Idempotency-Key` (UUID), required `If-Match` (`ETag` / `resource_version`, `428` when missing, `412` on mismatch) | `202 Accepted` | Atomically updates `policy_definitions.active_revision_id` and increments `generation`, runs type-specific activation hooks (`plans` update, `dnspolicy` bundle compilation, or `releases.ApplySupportTx` for `release_support`), writes an `audit_events` row (`policy.activate` on `policy_definition:<uuid>`), enqueues `policy.activated` in `outbox_events`, and returns a `pending` `api.Operation` (`kind = "policy.activate"`) with `ETag` header. |

### Transitional `501 Not Implemented` Responses Removed

Wave 3 removes the transitional `501 Not Implemented` fallback for all three policy endpoints:
- `GET /api/v1/admin/policies` (`listPolicies`)
- `POST /api/v1/admin/policies` (`createPolicyRevision`)
- `POST /api/v1/admin/policies/{policy_id}/activate` (`activatePolicyRevision`)

---

## 2. Worker Domain Composition (`apps/worker/jobs_policies.go`)

`policies()` (`NewPoliciesDomain(nil, nil)`) is registered in `apps/worker/jobs.go` (`Domains()`) and returns a `worker.Built` whose `Jobs` registry and `Context` decorator (`WithPolicies`, `WithFUPCompiler`, and `membersruntime.WithCompiler`) wire the following `outbox_events` handlers without reading environment variables or opening separate database pools:

| Outbox `event_type` | Producer | Worker Handler Behavior & Evidence Reference |
|---|---|---|
| `policy.activated` | `policies.Store.ActivateRevision` | Reads back committed `policy_definitions` state (`active_revision_id`, `generation`). When `type` is `fair_use` or `plan`, schedules a coalesced Fair-Use sweep across active members (`fup.Worker.ScheduleMemberSweep` + `Flush`). Transitions the `policy.activate` operation's `desired_state_compile` step to `succeeded` with read-back evidence `policy_definition:<definition_id>`. |
| `fup.evaluate` | `usage.Service.IngestBatch` / `usage.Service.CreateAdjustment` (`wire_usage.go`, `adjustments.go`) | Runs `fup.Worker` (`NotifyLedgerChange` or `NotifyLedgerCorrection` when `allow_relaxation = true`) and `Flush` against `usage.Service.GetMemberPeriodTotal`, recording the evaluation's `LedgerWatermark` (`ledger_epoch`, `projection_generation`, `aggregate_as_of`, per-stream watermarks). Returns evidence `fup_evaluation:<evaluation_id>` (or `member:<member_id>`). |
| `fup.member_generation_changed` | `fup.Engine.Compute` (in `enforce` mode when enforced tier/rate/override changes) | Reads back `fup_member_states` for the aggregate ID and returns evidence `fup_member_state:<state_id>` so downstream desired-state compilation can observe the member's new rate policy generation. |
| `fleet.release_support_changed` | `releases.Service.ApplySupportTx` (invoked inside `ActivateRevision` for `release_support`) | Enqueued by `release_support` policy activation; dispatched by the `nodes` worker domain (`apps/worker/jobs_nodes.go`) to refresh node desired state against `fleet_config`. |

### Exported Desired-State Rate Policy Helpers

`fup.BuildNodeMemberRatePolicy` remains exported with its contract signature unchanged for desired-state compilation, and `worker.FUPCompilerFromContext(ctx)` exposes `EvaluateAndBuildMemberRatePolicy` on the worker dispatch context:

```go
func BuildNodeMemberRatePolicy(eval *EvaluationResult, planRate, nodeCap *api.RatePair) api.MemberRatePolicy
```

---

## 3. Configuration (`internal/config/policies.go`, Decision D-045)

Production composition in both `fvpn-control` and `fvpn-worker` loads the policies configuration file referenced by `Dependencies.Config.PoliciesConfigFile` (`"policies_config_file"` in central `config.json`) via `config.LoadPolicies(path)` and validates central bounds via `config.ValidatePoliciesCentralBounds(d.Config)`. There are no compiled-in business defaults (`PoliciesDefaults` and `PoliciesFromConfig` have been removed); a missing or invalid file fails startup with `&config.SettingError{Name: "policies_config_file"}` or the invalid field's `SettingError`. Composition tests use `internal/policies/policiestest.New(t)` under `t.TempDir()`.

| JSON Field / Setting Name | Type | Requirement | Description |
|---|---|---|---|
| `fup_mode` | `"" \| "shadow" \| "enforce"` | Optional (`""` uses active `fair_use` policy revision's `mode`) | Optional operator override for Fair-Use evaluation mode (`shadow` vs. `enforce`, §11.2). |
| `fup_sweep_seconds` | `int` (`1..86400`) | Required | Coalescing / sweep interval in seconds for `fup.Worker` member evaluation sweeps. |
| `fup_batch_size` | `int` (`1..10000`) | Required | Maximum member batch size per coalesced Fair-Use sweep. |
| `dns_upstream_profile` | `string` (non-empty) | Required | Upstream DNS resolver profile name used by policy compilation. |
