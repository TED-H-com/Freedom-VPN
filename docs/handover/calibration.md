# Parameter Calibration Guide (`docs/handover/calibration.md`)

This guide catalogs every business, policy, capacity, timing, DNS, and retention parameter that requires explicit decision or empirical calibration before and during production operations (`§4`, `§6`–`§13`, `§15`–`§17`, `§20`–`§22`).

In accordance with `AGENTS.md §1` and Decision `D-045`, parameters marked `待決策` (pending business decision) or `待實測` (needs measurement) never carry silent compiled-in production defaults. Every parameter below specifies:
- **Location**: Config file JSON key, active policy payload (`POST /api/v1/admin/policies/{kind}/revisions`), or node/API field.
- **Default Status**:
  - `none`: No compiled default; startup or validation fails closed if omitted.
  - `test only`: Repository fixtures/examples supply a `TEST ONLY` placeholder that must be replaced in production.
  - `proposed`: Spec defines a `建議預設` (recommended default) implemented as a configurable parameter.
- **Spec Section**: Constraining section(s) in `docs/spec/PROJECT_SPEC.zh-TW.md`.
- **Decision Owner**: `Ted` (commercial/product/legal owner) or `core team` (engineering/operations team).

---

## 1. Master Parameter Calibration Matrix

### 1.1 Prices, Plans, and Billing Entitlements (`§5.1`, `§9`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Plan Definitions & Prices (`plans[]`, `provider_price_ids`, `amount_minor`, `currency`)** | Policy `plan` (`contracts/schemas/policies/plan-payload.schema.json`) + `members.json:price_ids` | `test only` | Repo test fixture: `price_test_monthly` / `e2e-price`. Must be non-empty strings (`1..255` chars) matching live Stripe Price IDs. | `§9.1`, `§9.2`, `§14.5`, `§21` (`待決策`) | `Ted` |
| **Live Payment Mode (`live_mode`)** | `members.json:live_mode` | `test only` | Repo example: `false`. Set `true` in production so `billing` rejects test-mode Stripe webhooks. | `§9.2`, `§9.4` (`必須`) | `Ted` |
| **Full Refund Policy (`full_refund_policy`)** | `members.json:full_refund_policy` | `test only` | Enum: `"revoke_original_period"` or `"keep_original_period"`. Repo example uses `"revoke_original_period"`. | `§9.3`, `§21` (`待決策`) | `Ted` |
| **Payment Past-Due Grace Period (`grace_seconds`, `grace_reason`)** | `members.json:grace_seconds`, `members.json:grace_reason` | `none` (pointer required; `0` in test fixtures) | Integer `≥ 0` seconds (`0` disables grace; e.g., `259200` = 3 days). | `§9.3`, `§21` (`待決策`) | `Ted` |
| **Trial Maximum Duration (`trial_max_seconds`)** | `members.json:trial_max_seconds` | `none` (pointer required; `0` in test fixtures) | Integer `≥ 0` seconds (`0` disables trials). | `§9.3`, `§21` (`待決策`) | `Ted` |
| **Manual Grant Maximum Duration (`grant_max_seconds`)** | `members.json:grant_max_seconds` | `test only` | Integer `> 0` seconds (test fixture: `86400`; e.g., `2592000` = 30 days max per admin grant, `F-05`). | `§9.3`, `§21` (`待決策`) | `Ted` |
| **Billing Reconcile Cadence (`reconcile_seconds`)** | `members.json:reconcile_seconds` | `test only` | Bounds: `1..86400` seconds (test fixture: `300`; recommended production: `900`). | `§9.3`, `§17.4` (`建議預設`) | `core team` |
| **Checkout Reservation Lease (`billing_lease_seconds`)** | `members.json:billing_lease_seconds` | `test only` | Bounds: `1..600` seconds (test fixture: `120`). Prevents duplicate concurrent checkout sessions (`00121`). | `§9.2`, `D-031` (`必須`) | `core team` |

### 1.2 Device Limits & Member Overlay Sync (`§7.3`, `§9.4`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Default Device Limit per Member (`device_limit`)** | `members.json:device_limit` + Policy `plan` (`device_limit`) | `proposed` | Spec recommended default: **`3`** (`1..10`). Enforced atomically during central device authorization (`member.device_authorize`, `409 DEVICE_LIMIT_REACHED`). | `§7.3`, `§9.4`, `§19 AT02`, `§21` (`建議預設`) | `Ted` |
| **Per-Member Device Limit Override (`device_limit_override`)** | `PATCH /api/v1/admin/members/{member_id}` (`entitlements.device_limit`) | `none` | Nullable integer `1..20`. Null inherits the active plan/default `device_limit`. | `§9.2`, `§14.5` (`建議預設`) | `Ted` |
| **Device Overlay Sync Cadence (`device_sync_seconds`)** | `members.json:device_sync_seconds` | `test only` | Bounds: `1..3600` seconds (test fixture: `15`; recommended production: `60`). | `§7.3`, `§9.4` (`建議預設`) | `core team` |

### 1.3 Fair-Use Policy (FUP) Thresholds, Tiers, and Enforcement Mode (`§11`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **FUP Enforcement Mode Override (`fup_mode`)** | `policies.json:fup_mode` + Policy `fair_use` (`mode`) | `proposed` | Enum: `""`, `"shadow"`, `"enforce"`. **Must** launch v1 in `"shadow"` (`FUP01`, `§11.2`). | `§11.1`, `§11.2`, `§21` (`建議預設`: `shadow` at launch) | `Ted` & `core team` |
| **FUP Tier Thresholds (`tiers[].threshold_bytes`)** | Policy `fair_use` (`contracts/schemas/policies/fair-use-payload.schema.json`) | `test only` | Decimal strings (`decstr.U64`), strictly increasing across tiers (`§11.1`). Evaluated against UTC calendar-month total (`[YYYY-MM-01T00:00:00Z, next_month)`). | `§11.1`, `§11.2`, `§21` (`待決策` / `待實測`) | `Ted` |
| **FUP Tier Rates & Priorities (`down_bps`, `up_bps`, `priority`)** | Policy `fair_use` (`tiers[]`) | `test only` | Decimal strings (`down_bps`, `up_bps` non-increasing across tiers; `priority` `1..7` non-decreasing; every tier `≥ minimum_rate`). Shared across all devices of the same member on the same exit node (`§11.3`). | `§11.1`–`§11.3`, `§21` (`待決策` / `待實測`) | `Ted` & `core team` |
| **Configured Shaping Floor Rate (`minimum_rate.down_bps`, `minimum_rate.up_bps`)** | Policy `fair_use` (`minimum_rate`) | `test only` | Must be `> 0` (`§11.1`, `§11.5`–`§11.7`: configured HTB shaping floor so FUP tiering itself does not set `0 bps`; not a guaranteed delivered throughput floor, and mandatory fail-closed rules still stop forwarding on lease/safety/meter failure). | `§11.1`, `§11.5`–`§11.7`, `§19 AT18, AT19`, `§21` (`待決策`) | `Ted` |
| **FUP Worker Sweep Cadence (`fup_sweep_seconds`, `fup_batch_size`)** | `policies.json:fup_sweep_seconds`, `policies.json:fup_batch_size` | `none` (`60` / `64` in `policiestest`) | `fup_sweep_seconds`: `1..86400` (recommended `60`). `fup_batch_size`: `1..10000` (recommended `256`). | `§11.2`, `D-045` (`建議預設`) | `core team` |

### 1.4 Lease Durations, `max_offline_seconds`, and Node Lifetimes (`§6.1`, `§6.4`, `§13`, `§16.3`, `§16.5`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Serving Lease Duration (`lease_seconds` / `max_offline_seconds`)** | Central lease issuer (`internal/leases`) & signed `ServingLease` envelope (`max_offline_seconds`) | `proposed` | Spec recommended default: **`86400` seconds (24 hours)**. Node fail-closes member forwarding at the earliest of `effective_service_end`, `node_lease_end`, and `signed_server_time + max_offline_seconds` (`§16.3`, `AT29`). | `§16.3`, `§16.5`, `§19 AT29`, `§21` (`建議預設`) | `core team` |
| **Installer Enrollment Token TTL (`installer_seconds`)** | `nodes.json:installer_seconds` | `proposed` | Spec recommended default: **`900` seconds (15 minutes)** (`1..31536000`). | `§6.1`, `§6.4`, `§21` (`建議預設`) | `core team` |
| **Pre-installed Hardware Enrollment Token TTL (`preinstalled_seconds`)** | `nodes.json:preinstalled_seconds` | `proposed` | Spec recommended default: **`604800` seconds (7 days)** (`1..31536000`). | `§6.1`, `§6.4`, `§21` (`建議預設`) | `Ted` & `core team` |
| **NetBird Bootstrap Setup-Key Window (`bootstrap_seconds`)** | `nodes.json:bootstrap_seconds` | `none` (`60` in `nodestest`, `300` in `devstack`) | Bounds: `1..31536000` seconds (recommended production: `600`). Expired unconsumed bootstrap returns `410 BOOTSTRAP_EXPIRED`. | `§6.2`, `D-015` (`建議預設`) | `core team` |
| **Node mTLS Certificate Lifetime & Renewal Overlap (`certificate_seconds`, `renewal_overlap_seconds`)** | `nodes.json:certificate_seconds`, `nodes.json:renewal_overlap_seconds`, `agent.json:renew_lead_seconds` | `none` (`86400` / `300` in `nodestest`; `120` in `agent.json`) | `certificate_seconds`: `1..31536000` (recommended `604800` = 7 days). `renewal_overlap_seconds`: `0 .. certificate_seconds-1` (recommended `86400` = 24h). | `§6.4`, `§21` (`待決策`) | `core team` |
| **Desired-State Envelope Validity (`desired_state_seconds`)** | `nodes.json:desired_state_seconds` | `none` (`600` in `nodestest`) | Bounds: `1..31536000` seconds (recommended `600`–`3600`). | `§14.7` (`建議預設`) | `core team` |
| **Update Job Authorization Window (`authorization_seconds`)** | `nodes.json:authorization_seconds` | `none` (`300` in `nodestest`) | Bounds: `1..31536000` seconds (recommended `3600`). Updater refuses to start apply if `now >= authorization_expires_at` (`§13.4`). | `§13.4`, `§13.8` (`建議預設`) | `core team` |
| **Node Online & Stuck Update Thresholds (`online_seconds`, `stuck_seconds`)** | `nodes.json:online_seconds`, `nodes.json:stuck_seconds` | `none` (`180` / `600` in `nodestest`) | `online_seconds`: `180` (node marked stale/offline if no heartbeat). `stuck_seconds`: `600`–`900` (flags `UpdateTarget.needs_attention = true`, `§13.10`). | `§5`, `§13.10`, `§16.5` (`建議預設`) | `core team` |
| **Agent Heartbeat & Watchdog Stale Cutoff (`heartbeat_seconds`, `watchdog_stale_seconds`)** | `agent.json:heartbeat_seconds`, `agent.json:watchdog_stale_seconds` | `proposed` | Defaults in `agent.LoadConfig`: `heartbeat_seconds = 10`, `watchdog_stale_seconds = 30`. `fvpn-watchdog` fail-closes local member forwarding if the local agent heartbeat file is older than `watchdog_stale_seconds` (`§16.3`). | `§16.3`, `§19 AT29` (`建議預設`) | `core team` |

### 1.5 DNS List Sources, Upstream Resolvers, and Update Cadence (`§12`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **External Threat-Feed Sources & Commercial Licenses (`sources[]`)** | Policy `threat_feed` (`contracts/schemas/policies/threat-feed-payload.schema.json`) | `none` (`待決策`) | Commercial threat-feed URLs and license terms are `待決策` (`§12.3`, `§21`). Never enable an unlicensed third-party feed in production; use first-party `domain_rule_set` until feed licenses are signed. | `§12.3`, `§21` (`待決策`) | `Ted` |
| **Threat-Feed Refresh Interval (`refresh_interval_seconds`)** | Policy `threat_feed` (`sources[].refresh_interval_seconds`) | `proposed` | Bounds: `300..604800` seconds (recommended `3600`–`14400` = 1–4 hours). Compiler retains last known good snapshot on upstream fetch failure (`§12.3`). | `§12.3` (`建議預設`) | `core team` |
| **DNS Blocked Response Mode & TTL (`block_mode`, `blocked_ttl_seconds`)** | Policy `dns_profile` (`contracts/schemas/policies/dns-profile-payload.schema.json`) | `proposed` | `block_mode`: `"nxdomain"` (required by `§12.1`). `blocked_ttl_seconds`: `1..86400` (recommended `60` seconds as a starting point for the DNS response TTL header; not an end-to-end propagation SLA, `§12.5`). | `§12.1`, `§12.5`, `§19 AT20, AT22` (`建議預設`) | `core team` |
| **Recursive Upstream Resolvers (`upstreams`, `bootstrap_dns`)** | `dns.json:upstreams`, `dns.json:bootstrap_dns` | `none` (`test only` in `docs/dns/config.example.json`) | Non-empty list of `IP:port` literals (`D-028`). Injected into AdGuard Home by `fvpn-worker`. | `§12.2`, `D-028` (`必須`) | `core team` |
| **Node Overlay DNS Resolver Addresses (`dns_resolver_ipv4`, `dns_resolver_ipv6`)** | `nodes.json:dns_resolver_ipv4`, `nodes.json:dns_resolver_ipv6` | `none` (`test only` in `nodestest`) | At least one IPv4 or IPv6 literal required (`§12.2` recommends `≥ 2` resolvers for failover). `node/enforce` permits `TCP/53` and `UDP/53` to `@resolver_v4` (`dns_service_v1`) and drops non-53 traffic (`ISO04`). | `§8.1`, `§12.2`, `ISO04` (`必須`) | `core team` |

### 1.6 Data Retention Periods (`§10.5`, `§15.4`, `§17.3`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Central Raw Usage Payload Retention (`raw_usage_retention_days`)** | `usage.json:raw_usage_retention_days` + DB `retention_config` (`00000000-0000-0000-0000-000000000002`) | `proposed` | Spec recommended default: **`90` days** (validated bounds in `config.UsageConfig`: `30..3650` days; DB trigger `F-03` enforces `≥ 90` days and refuses to purge months with `gap = TRUE`). | `§15.4`, `F-03` (`建議預設`: `90` days) | `Ted` & `core team` |
| **Central Monthly/Daily Rollup Retention (`rollup_retention_months`)** | `usage.json:rollup_retention_months` + DB `retention_config` | `proposed` | Spec recommended default: **`24` months** (validated bounds: `12..120` months). | `§15.4` (`建議預設`: `24` months) | `Ted` |
| **Central Audit Log Retention (`audit_retention_days`)** | `usage.json:audit_retention_days` + DB `retention_config` | `proposed` | Spec recommended default: **`365` days** (validated bounds: `180..3650` days; `F-03` enforces `≥ 365` days). | `§15.4`, `F-03` (`建議預設`: `365` days) | `Ted` |
| **Service Log Retention (`service_log_days`)** | DB `retention_config` (`00000000-0000-0000-0000-000000000002`) + journald/logrotate | `proposed` | Spec recommended default: **`30` days** (`F-03` enforces `≥ 7` days). | `§15.4` (`建議預設`: `30` days) | `core team` |
| **Node Local ACKed Usage Archive Retention** | `node/internal/usagestore` (`DefaultACKedRetention`) | `proposed` | Spec recommended default: **`30` days** (`§10.5`). Unacknowledged events are retained until ACKed or local disk budget (`256 MiB`) triggers fail-closed forwarding block. | `§10.5`, `§17.2` (`建議預設`: `30` days) | `core team` |
| **Billing & Tax Record Retention** | `billing_events`, `billing_subscriptions` | `none` (`待決策`) | Retained per statutory accounting/tax law in the operating jurisdiction (`§15.4`, `§21`). Never purged by `purge_retained`. | `§15.4`, `§21` (`待決策`) | `Ted` |

### 1.7 Node Capacity Ceilings & Datapath Reserves (`§7.2`, `§8.3`, `§10`, `§13.4`, `§20.2`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Node Total Capacity Ceiling (`capacity.down_bps`, `capacity.up_bps`)** | `PATCH /api/v1/admin/nodes/{node_id}` (`capacity`) | `none` (`待實測`) | Decimal strings (`decstr.U64`, `> 0`). Used as the `tc` HTB root ceiling on `fvmb0` (`egress`) and `fvup0` (`egress`). If unset (`null`), node uses observed link capacity or policy defaults. | `§7.2`, `§8.3`, `§20.2` (`待實測`) | `core team` |
| **Node Per-Member Ceiling (`capacity.member_down_bps_cap`, `capacity.member_up_bps_cap`)** | `PATCH /api/v1/admin/nodes/{node_id}` (`capacity`) | `none` (`待實測`) | Decimal strings (`decstr.U64`, `> 0`, `≤ down_bps` / `up_bps`). Caps any single member's HTB class rate on that exit node even in Tier 0 (unthrottled). | `§7.2`, `§11.3`, `§20.2` (`待實測`) | `Ted` & `core team` |
| **Host Control-Plane Bandwidth Reserve (`host_reserve_bps`)** | Node `/etc/vpn-node/agent.json:host_reserve_bps` | `none` (`test only` `1000000` in `devstack`) | Mandatory `uint64 > 0` (`D-034`, exits `64` if `0` or omitted). Reserves bandwidth on `fvup0` for mTLS control, heartbeats, usage uploads, and SSH/management (`§8.3`). | `§8.3`, `§20.2`, `D-034` (`待實測`) | `core team` |
| **Updater Disk Free-Space Floor (`required_free_bytes` / `MinFreeBytes`)** | Release manifest `rollback.required_free_bytes` + `node/updater` (`512 MiB` floor) | `proposed` | Updater enforces `free_bytes >= artifact.size_bytes + artifact.installed_size_bytes + max(rollback.required_free_bytes, 536870912)` (`507 DISK_SPACE_LOW` on violation, `§13.4`). | `§13.4`, `§16.3` (`建議預設`: `512 MiB`) | `core team` |

---

## 2. Step-by-Step Calibration Workflow (`§11.4`, `§20.2`)

### 2.1 Calibrating Per-Node Capacity Ceilings and `host_reserve_bps` (`§20.2`)

Before setting `publication_intent = "published"` on a new exit node hardware profile (`amd64` or `arm64`):
1. Measure sustained NIC throughput and CPU softirq head-room inside an isolated test or staging environment (`docs/poc/G2-netlab.md`).
2. Set `host_reserve_bps` in `/etc/vpn-node/agent.json` to `max(10 Mbps, 2% of uplink capacity)` (e.g., `20000000` = `20 Mbps` on a `1 Gbps` node) so mTLS heartbeats, lease renewals, and usage batch uploads never starve under member congestion.
3. Configure `capacity` on the node via `PATCH /api/v1/admin/nodes/{node_id}` (`If-Match` required) to `85%–90%` of measured sustainable WireGuard + `tc` HTB throughput so `fq_codel` controls queue delay on the node rather than dropping packets in upstream provider buffers.

### 2.2 Calibrating Fair-Use Thresholds from Shadow Mode (`§11.4`, `FUP04`)

1. Launch with `"fup_mode": "shadow"` in `/etc/fvpn/policies.json` and activate an initial `fair_use` policy revision.
2. In `shadow` mode, `fvpn-worker` computes tier transitions (`fup_member_states`) and emits audit/telemetry records, while compiling full unthrottled rates into node desired state and returning `throttled: false` to members.
3. At the end of the first billing cycle, query the distribution of member monthly usage in PostgreSQL:
   ```sql
   SELECT
     percentile_cont(0.50) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)) AS p50_bytes,
     percentile_cont(0.90) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)) AS p90_bytes,
     percentile_cont(0.95) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)) AS p95_bytes,
     percentile_cont(0.99) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)) AS p99_bytes
   FROM usage_rollups
   WHERE granularity = 'month'
     AND period = to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM');
   ```
4. Once `Ted` approves the production `threshold_bytes` and tier rates (`down_bps`, `up_bps`, `minimum_rate`), create and activate a new `fair_use` policy revision (`POST /api/v1/admin/policies/fair_use/revisions` -> `POST .../activate`) and switch `"fup_mode": "enforce"`.
