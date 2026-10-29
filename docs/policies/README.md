# Freedom VPN Policies & Fair-Use Engine (`internal/policies`, `internal/fup`)

This document describes the versioned Policies API store (`internal/policies`) and the Fair-Use Policy (FUP) engine (`internal/fup`), implementing Project Spec v1.0 §4 (`FUP01`–`FUP04`, `DNS01`–`DNS03`), §11 (`§11.1`–`§11.3`, `§11.5`, `§11.6`), §12.4–§12.5, §13.2, §14.2, §14.5, §15.3, §15.5 item 3, and acceptance cases `AT17`, `AT18`, `AT19`, and `AT27` (§19).

> **Important (`待決策` / `TEST ONLY`):** Formal production prices, fair-use byte thresholds, tier rates, and minimum rate floors are pending business decisions (`待決策`). Every numeric price, byte threshold, and rate in this document and in test fixtures is strictly `TEST ONLY`.

---

## 1. Versioned Policies API (`internal/policies`, §14.2, §15.3)

All configurable behavior goes through a single versioned policy model backed by `policy_definitions` and immutable `policy_revisions`:

1. **`GET /api/v1/admin/policies`** (`Store.ListDefinitions`):
   - Lists policy definitions filtered optionally by `type` (`plan`, `fair_use`, `dns_profile`, `domain_rule_set`, `threat_feed`, `release_support`), ordered deterministically by `(type, code)`.
   - Returns each definition's `definition_id`, `type`, `code`, `active_revision_id`, `active_revision`, `generation` (decimal string), `resource_version` (decimal string), `updated_at`, and recent `revisions`.

2. **`POST /api/v1/admin/policies`** (`Store.CreateRevision`):
   - Requires `Idempotency-Key` (UUID, §14.5).
   - Validates the payload against its versioned JSON Schema 2020-12 (`internal/policies/schemas/*.schema.json`) and type-specific semantic validator (`ValidatePayload`).
   - Verifies foreign references and fixed isolation baseline constraints inside the database transaction.
   - Computes `payload_hash` (`SHA-256`) over the RFC 8785 Canonical JSON (`pkg/canonjson`) representation of `payload`.
   - Stores an immutable row in `policy_revisions` with `UNIQUE(definition_id, revision)`, recording `created_by` and `reason`. Database triggers (`guard_append_only()`) forbid any `UPDATE`, `DELETE`, or `TRUNCATE` on `policy_revisions`.

3. **`POST /api/v1/admin/policies/{policy_id}/activate`** (`Store.ActivateRevision`):
   - Requires `Idempotency-Key` (UUID) and `If-Match` carrying the current `resource_version` of the `policy_definitions` row (`428 PRECONDITION_REQUIRED` when missing, `412 PRECONDITION_FAILED` on mismatch, §14.5, §15.5 item 3).
   - Switches `active_revision_id` under `db.UpdateWithVersion`, increments `policy_definitions.generation` (enforced monotonic by DB trigger), and writes the `operations` row (`status = 'pending'`, returning HTTP `202 Accepted`), `audit_events` row, and `outbox_events` row (`policy.activated`) in a single PostgreSQL transaction.
   - **Idempotent re-activation:** Activating the revision that is already active on the definition is a no-op and does not increment `generation`.
   - **Rollback as explicit new activation (§11.5, AT19):** An older revision can be activated again only as an explicit new activation command carrying the current `If-Match` `resource_version`, which increments `generation` to a new higher value (`N + 1`) rather than replaying an old generation number.

---

## 2. Policy Types, Validation Rules, and `TEST ONLY` Examples

### 2.1 `fair_use` (§11.1, §14.2)

**Validation rules:**
- `mode` must be `shadow` or `enforce`, with a finite `effective_at` timestamp and `effective_period` in `next_full_period` (recommended default) or `immediate`.
- `accounting_basis`: `period_type = "utc_calendar_month"`, `metering_version = "forward_admission_v1"`, `measure = "total_bytes"` (`upload_bytes + download_bytes`).
- `tiers` must be non-empty (`1..16` tiers) and `tiers[0].threshold_bytes` must be `"0"` so tiers cover `[0, ∞)` with no gap. A missing tier is never treated as unlimited.
- `threshold_bytes` (decimal uint64 strings) must strictly increase (`tiers[i].threshold_bytes > tiers[i-1].threshold_bytes`).
- Every tier must specify non-negative integer `down_bps` and `up_bps`, `priority` (`0..7`), and `display_name` (`zh-TW` required).
- `down_bps` and `up_bps` must be non-increasing (`tiers[i] <= tiers[i-1]`), and `priority` must be non-decreasing (`tiers[i].priority >= tiers[i-1].priority`) so a higher tier is never faster or more favored.
- Every tier's `down_bps` and `up_bps` must be `>= minimum_rate.down_bps` and `>= minimum_rate.up_bps` (`minimum_rate` > 0).
- `overrides`: at most one entry per `member_id`, with non-negative `down_bps`/`up_bps`, `priority` (`0..7`), finite `valid_until` in the future, non-empty `reason`, and `actor_id`.

**Example (`TEST ONLY` — thresholds and rates are `待決策`):**
```json
{
  "type": "fair_use",
  "code": "default-fair-use",
  "schema_version": 1,
  "reason": "TEST ONLY shadow fair-use policy revision",
  "payload": {
    "mode": "shadow",
    "effective_at": "2026-10-01T00:00:00Z",
    "effective_period": "next_full_period",
    "accounting_basis": {
      "period_type": "utc_calendar_month",
      "metering_version": "forward_admission_v1",
      "measure": "total_bytes"
    },
    "minimum_rate": {
      "down_bps": 5000000,
      "up_bps": 2000000
    },
    "tiers": [
      {
        "threshold_bytes": "0",
        "down_bps": 100000000,
        "up_bps": 50000000,
        "priority": 1,
        "display_name": { "zh-TW": "標準速率 (TEST ONLY)", "en": "Standard (TEST ONLY)" }
      },
      {
        "threshold_bytes": "100000000000",
        "down_bps": 25000000,
        "up_bps": 10000000,
        "priority": 3,
        "display_name": { "zh-TW": "公平使用第一階 (TEST ONLY)", "en": "Fair-Use Tier 1 (TEST ONLY)" }
      }
    ],
    "overrides": []
  }
}
```

### 2.2 `plan` (§9, §14.2)

**Validation & baseline rules:**
- `device_limit` must be in `1..10` (recommended default `3`).
- `exit_group_ids` must be non-empty, unique UUIDs, and every referenced group must exist in `member_groups` (a plan cannot grant a non-existent exit group).
- `fair_use_revision_id` and `dns_profile_revision_id` must reference existing `policy_revisions` of type `fair_use` and `dns_profile` respectively.
- `billing_period_months` in `1..24`, non-negative `price_minor` (`TEST ONLY` / `待決策`), `currency` (`TWD` or `USD`), and non-empty `provider_price_ref`.

**Example (`TEST ONLY` — price is `待決策`):**
```json
{
  "type": "plan",
  "code": "standard-monthly",
  "schema_version": 1,
  "reason": "TEST ONLY standard monthly plan",
  "payload": {
    "display_name": { "zh-TW": "標準方案 (TEST ONLY)", "en": "Standard Plan (TEST ONLY)" },
    "billing_period_months": 1,
    "price_minor": 19900,
    "currency": "TWD",
    "provider_price_ref": "price_test_only_standard_twd",
    "device_limit": 3,
    "exit_group_ids": ["01950000-0000-7000-8000-000000000001"],
    "fair_use_revision_id": "01950000-0000-7000-8000-000000000002",
    "dns_profile_revision_id": "01950000-0000-7000-8000-000000000003"
  }
}
```

### 2.3 `domain_rule_set`, `threat_feed`, and `dns_profile` (§12.4, §12.5, §14.2)

**Validation & fixed isolation baseline rules (`internal/dnspolicy`):**
- **`domain_rule_set`:** Normalizes every domain via IDNA (`dnspolicy.NormalizeDomain`), rejects wildcards, schemes, ports, paths, IP literals, and private/special-use zones (`localhost`, `.local`, `.internal`, `.home.arpa`, `.lan`, `.corp`, `.invalid`, reverse-DNS `in-addr.arpa` / `ip6.arpa`), rejects `allow` rules without an explicit `profile_id` (`DNS_ALLOW_REQUIRES_PROFILE`), rejects duplicate `rule_id`s or conflicting rules for the same normalized domain, and requires `expires_at > created_at`.
- **`threat_feed`:** Validates `feed_code`, `source_url` (`https://`), `content_sha256` (64-hex SHA-256), `license_name`, `license_url`, `commercial_use_basis` (must not forbid commercial use), `categories`, and finite validity window `valid_until > valid_from`. Mutable fetch observations live in observation tables, never inside `policy_revisions`.
- **`dns_profile`:** Validates referenced `domain_rule_set_revision_ids` and `threat_feed_revision_ids`, checks cross-revision domain conflicts and `profile_id` scoping, and on activation compiles an immutable `dnspolicy.CompiledBundle` stored in `dns_compiled_bundles`.

### 2.4 `release_support` (§13.2, §14.2, AT27)

**Validation rules:**
- Membership is determined strictly by **explicit UUID set membership**, never by lexical or semver string comparison (§13.2).
- `minimum_serving_version.serving_release_ids` must be non-empty and free of duplicates.
- `approved_release_ids` must be a subset of `minimum_serving_version.serving_release_ids` (`RELEASE_SET_INVALID` on violation).
- `revoked_release_ids` must be disjoint from both `approved_release_ids` and `minimum_serving_version.serving_release_ids` (`RELEASE_SET_INVALID` on overlap).
- Every referenced release UUID must exist in `releases`. On activation, `fleet_config` is updated atomically in the same transaction.

**Example:**
```json
{
  "type": "release_support",
  "code": "fleet-release-support",
  "schema_version": 1,
  "reason": "Allowlist v1.0.1 and v1.0.2 for serving; approve v1.0.2 for rollout",
  "payload": {
    "approved_release_ids": [
      "01950000-0000-7000-8000-000000000012"
    ],
    "minimum_serving_version": {
      "serving_release_ids": [
        "01950000-0000-7000-8000-000000000011",
        "01950000-0000-7000-8000-000000000012"
      ]
    },
    "revoked_release_ids": [
      "01950000-0000-7000-8000-000000000010"
    ]
  }
}
```

---

## 3. Fair-Use Engine Algorithm & Effective Rate Formula (`internal/fup`, §11.1–§11.3, §11.5)

### 3.1 Tier Selection and Stricter-Only Monotonicity (§11.1)

For a member and UTC accounting period (`YYYY-MM`):
1. Read `total_bytes = upload_bytes + download_bytes` (`decstr.U64`, full 64-bit unsigned integer precision including `2^53 + 1` up to `2^64 - 1`) and the ledger watermark `aggregate_as_of` across all nodes from `UsageReader`.
2. Select tier `i` such that:
   $$\text{tiers}[i].\text{threshold\_bytes} \le U < \text{tiers}[i+1].\text{threshold\_bytes}$$
   Equality ($U == \text{threshold\_bytes}$) enters the new tier immediately (`SelectTier`).
3. **Within-period stricter-only rule:** During normal accumulation within an accounting period, `tier_index` only stays equal or moves to a higher (stricter) index. A lower tier index is permitted within the same period **only** when an audited negative ledger adjustment (`usage_adjustments` with negative bytes / `AllowRelaxation`) is present, which records `relaxed = true` and emits an `audit_events` row (`fup.tier.relaxed`).
4. **Period rollover and late previous-period events (§11.5, AT17, AT19):**
   - At the start of a new UTC period (`YYYY-MM`), evaluation starts fresh at Tier 0 (`normal`).
   - Late usage events belonging to a closed previous period (`period_id < current_period_id`) update only that previous period's evaluation record (`LatePeriodEvent = true`) and **never** increment the member's policy generation or alter the current period's tier.

### 3.2 Effective Rate and Rate Scope Formula (§11.3, AT18)

Per-node desired-state compilation computes directional rates (`down` and `up` independently) using:

$$\text{effective\_rate} = \min(\text{plan\_rate},\ \text{fair\_use\_tier\_rate},\ \text{node\_member\_rate\_cap})$$

$$\text{rate\_scope} = (\text{member\_id},\ \text{node\_id},\ \text{direction})$$

- **All devices on one exit share one class:** All authorized devices of `member_id` routing through `node_id` share a single HTB + `fq_codel` class per direction on that node (`rate_scope = (member_id, node_id, direction)`).
- **Two exits each provide their own $R$:** Because `node_id` is part of `rate_scope`, two distinct exit nodes (`node_A`, `node_B`) each apply their own directional rate $R_A$ and $R_B$ independently on their respective data planes (`BuildNodeMemberRatePolicy`).

---

## 4. Shadow vs. Enforce Rollout Procedure (§11.2, FUP02)

Per §11.2, every new fair-use policy or threshold change **should roll out in `shadow` mode first**:

1. **Create and activate a `shadow` revision (`mode: "shadow"`):**
   - `Engine.Compute` runs the exact same tier-selection, stricter-only clamping, and override resolution code as `enforce` mode.
   - It writes `would_apply_tier_index`, `would_apply_down_bps`, `would_apply_up_bps`, and `impact_summary` to `fup_member_states` and `fup_evaluations`.
   - It keeps `generation = 0`, emits no desired-state outbox event, and shapes no traffic on nodes.
   - Member-facing `FairUseStatus` always reports `state: "normal"`, `applying: false`, `rate: null`, never claiming throttling during shadow observation.
2. **Inspect shadow impact:**
   - Operators review `AdminMemberFairUseStatus` (`would_apply_tier_index`, `would_apply_rate`, `impact_summary`) across the member base to verify tier distributions and ledger watermarks.
3. **Promote to `enforce` mode (`mode: "enforce"`):**
   - Create a new immutable `fair_use` revision with `mode: "enforce"` and activate it via `POST /api/v1/admin/policies/{id}/activate` with `If-Match`.
   - When `Engine.Compute` runs in `enforce` mode, any change to a member's enforced state/tier/rate increments the member's `generation` by 1 and enqueues `fup.member_generation_changed` in `outbox_events` to trigger desired-state compilation for all serving nodes.
   - Re-running `Compute` when the member's enforced tier/rate/override has not changed is idempotent and produces **no** new generation (`AT19`).
   - Member and admin status report `applying: true` (`"pending on N nodes"`) until all serving nodes acknowledge `applied_policy_refs.fup_generation >= generation` in `node_observations` (`FUP03`).

---

## 5. Time-Limited Admin Overrides (§11.1, §11.6, FUP04)

- Administrators can grant a temporary rate override for a member (`Engine.SetOverride`) or include an override in a `fair_use` policy revision.
- Every override requires `member_id`, non-negative `down_bps` and `up_bps`, `priority` (`0..7`), finite `valid_from` and `valid_until` (`valid_until > valid_from`), non-empty `reason`, and `actor_id`.
- Setting or revoking an override writes an `audit_events` row (`fup.override.set` / `fup.override.revoked`) and triggers `Engine.Compute`.
- When `now >= valid_until`, the next scheduled or ledger-triggered `Engine.Compute` automatically ignores the expired override, reverts the member to their usage-based tier, increments `generation`, and triggers desired-state recompilation.

---

## 6. Requirement & Acceptance Test Coverage

| ID | Description | Package & Test Coverage |
|---|---|---|
| `FUP01`, `§11.1` | Configurable FUP tiers (`threshold_bytes` decimal strings, `down_bps`, `up_bps`, `priority`, `minimum_rate`), boundary equality (`U == threshold`), `2^53 + 1` precision, stricter-only within period, audited relaxation on negative adjustment | `internal/policies`: `TestFUP01_FairUseValidationRules`<br>`internal/fup`: `TestFUP01_TierSelectionBoundariesAndLargeTotals`, `Test11_1_StricterOnlyAndAuditedRelaxation`, `Test11_1_CoalescedWorkerAndWatermark` |
| `FUP02`, `§11.2` | Unified shadow and enforce evaluation engine; shadow mode records `would_apply_tier` without shaping or generations; enforce mode emits member policy generations | `internal/fup`: `TestFUP02_AT19_ShadowVsEnforceAndIdempotentGeneration`, `TestFUP02_AT19_PostgresShadowToEnforceAndAppendOnlyGuards` |
| `FUP03`, `§11.6` | Member-visible (`FairUseStatus`) and admin-visible status with per-node ACK tracking (`"pending on N nodes"` until all serving nodes ACK, never claiming applied on 202) | `internal/fup`: `TestFUP03_MemberAndAdminStatusNodeAck`, `TestFUP03_FUP04_PostgresOverridesRelaxationAndNodeAck` |
| `FUP04`, `§11.1` | Time-limited, visible, audited admin overrides with automatic expiry recovery and early revocation | `internal/fup`: `TestFUP04_AdminOverrideAndExpiryRecovery`, `TestFUP03_FUP04_PostgresOverridesRelaxationAndNodeAck` |
| `DNS01`–`DNS03`, `ISO01`, `§12.4`–`§12.5` | `domain_rule_set`, `threat_feed`, and `dns_profile` validation, IDNA normalization, conflict detection, private prefix / special zone rejection, and bundle compilation on activation | `internal/policies`: `TestISO01_DNSIsolationBaselineAndConflicts`, `Test14_2_DNSProfileBundleCompilationAndPlanBaseline` |
| `AT17`, `§11.5` | Period rollover resets to Tier 0; late usage events for a previous period update only the previous period and never alter current period tier or generation | `internal/fup`: `TestAT17_AT19_PeriodRolloverAndLatePreviousPeriodEvent` |
| `AT18`, `§11.3` | Central side of FUP rate calculation: `effective_rate = min(plan_rate, fair_use_tier_rate, node_member_rate_cap)` and `rate_scope = (member_id, node_id, direction)` where two exits each provide their own rate $R$ | `internal/fup`: `TestAT18_EffectiveRateAndTwoExitScope` |
| `AT19`, `§11.5`, `§14.2`, `§15.5` | Immutable policy revisions, `If-Match` optimistic concurrency (`428`/`412`), monotonic generations, activating an older revision produces a new higher generation (`N + 1`), recomputing unchanged FUP state produces no duplicate generation | `internal/policies`: `Test14_2_RevisionImmutabilityAndHash`, `TestAT19_PolicyActivationGenerationsAndConcurrency`<br>`internal/fup`: `TestFUP02_AT19_ShadowVsEnforceAndIdempotentGeneration`, `TestFUP02_AT19_PostgresShadowToEnforceAndAppendOnlyGuards` |
| `AT27`, `§13.2` | `release_support` policy validation (`approved ⊆ serving`, `revoked` disjoint, explicit UUID set membership only) and atomic `fleet_config` update on activation | `internal/policies`: `TestAT27_ReleaseSupportExplicitSetMembership`, `TestAT27_ReleaseSupportActivationUpdatesFleetConfig` |
