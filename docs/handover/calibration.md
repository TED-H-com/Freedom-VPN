# Parameter Calibration Guide (`docs/handover/calibration.md`)

This guide catalogs every business, policy, capacity, timing, DNS, and retention parameter that requires explicit decision or empirical calibration before and during production operations (`§4`, `§6`–`§13`, `§15`–`§17`, `§20`–`§22`).

In accordance with `AGENTS.md §1` and Decision `D-045`, parameters marked `待決策` (pending business decision) or `待實測` (needs measurement) never carry silent compiled-in production defaults. Every parameter below specifies:
- **Location**: Config file JSON key, active policy payload (`POST /api/v1/admin/policies`), or node/API field.
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
| **Plan Definitions & Prices (`provider_price_ref`, `price_minor`, `currency`, `billing_period_months`, `exit_group_ids`)** | Policy `plan` (`contracts/schemas/policies/plan-payload.schema.json`) + `members.json:price_ids` | `test only` | Repo test fixture: `price_test_only_standard_twd` / `e2e-price`. Must be non-empty strings (`1..255` chars) matching live Stripe Price IDs, with existing `member_groups` UUIDs in `exit_group_ids`. | `§9.1`, `§9.2`, `§14.5`, `§21` (`待決策`) | `Ted` |
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
| **Per-Member Group Assignment (`member_group_ids`)** | `PATCH /api/v1/admin/members/{member_id}` (`member_group_ids`) | `none` | Array of existing `member_groups` UUIDs. Combined with the member's pinned plan `exit_group_ids` to form `entitlements.projection.exit_group_ids` (`docs/members/README.md`). | `§9.2`, `§14.5`, `D-030` (`必須`) | `Ted` & `core team` |
| **Device Overlay Sync Cadence (`device_sync_seconds`)** | `members.json:device_sync_seconds` | `test only` | Bounds: `1..3600` seconds (test fixture: `15`; recommended production: `60`). | `§7.3`, `§9.4` (`建議預設`) | `core team` |

### 1.3 Fair-Use Policy (FUP) Thresholds, Tiers, and Enforcement Mode (`§11`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **FUP Enforcement Mode Override (`fup_mode`)** | `policies.json:fup_mode` + Policy `fair_use` (`mode`) | `proposed` | Enum: `""`, `"shadow"`, `"enforce"`. **Must** launch v1 in `"shadow"` (`FUP01`, `§11.2`). | `§11.1`, `§11.2`, `§21` (`建議預設`: `shadow` at launch) | `Ted` & `core team` |
| **FUP Tier Thresholds (`tiers[].threshold_bytes`)** | Policy `fair_use` (`contracts/schemas/policies/fair-use-payload.schema.json`) | `test only` | Decimal strings (`decstr.U64`), strictly increasing across tiers (`§11.1`). Evaluated against UTC calendar-month total (`[YYYY-MM-01T00:00:00Z, next_month)`). | `§11.1`, `§11.2`, `§21` (`待決策` / `待實測`) | `Ted` |
| **FUP Tier Rates & Priorities (`down_bps`, `up_bps`, `priority`)** | Policy `fair_use` (`tiers[]`) | `test only` | Non-negative integers (`down_bps`, `up_bps` non-increasing across tiers; `priority` `0..7` non-decreasing; every tier `≥ minimum_rate`). Shared across all devices of the same member on the same exit node (`§11.3`). | `§11.1`–`§11.3`, `§21` (`待決策` / `待實測`) | `Ted` & `core team` |
| **Configured Shaping Floor Rate (`minimum_rate.down_bps`, `minimum_rate.up_bps`)** | Policy `fair_use` (`minimum_rate`) | `test only` | Must be `> 0` (`§11.1`, `§11.5`–`§11.7`: configured HTB shaping floor so FUP tiering itself does not set `0 bps`; not a guaranteed delivered throughput floor, and mandatory fail-closed rules still stop forwarding on lease/safety/meter failure). | `§11.1`, `§11.5`–`§11.7`, `§19 AT18, AT19`, `§21` (`待決策`) | `Ted` |
| **FUP Worker Sweep Cadence (`fup_sweep_seconds`, `fup_batch_size`)** | `policies.json:fup_sweep_seconds`, `policies.json:fup_batch_size` | `none` (`60` / `64` in `policiestest`) | `fup_sweep_seconds`: `1..86400` (recommended `60`). `fup_batch_size`: `1..10000` (recommended `256`). | `§11.2`, `D-045` (`建議預設`) | `core team` |

### 1.4 Lease Durations, `max_offline_seconds`, Node Lifetimes, and Health Attestation (`§6.1`, `§6.4`, `§13`, `§16.3`, `§16.5`, `D-053`, `D-058`)

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
| **Node Update Health Attestation & Rollback Timings (`HealthAttestWait`, `HealthChallengeTTL`, `CheckAttempts`, `RollbackAttempts`)** | `node/updater` (`docs/updates/node-health-attestation.md`, `docs/updates/node-updater.md`) | `proposed` | `HealthAttestWait = 30s` (`HealthAttestPoll = 250ms`), `HealthChallengeTTL = 45s` (`health-challenge.json` / `health-attest.json` in `journal_dir`), `CheckAttempts = 8` (`CheckInterval = 2s`), `RollbackAttempts = 3` (`RollbackInterval = 1s`). | `§13.9`, `D-053` (`建議預設`) | `core team` |
| **Updater Synthetic Probe Peer Configuration (`probe_peer`, `agent_config`)** | Node `/etc/vpn-node/updater.json:probe_peer`, `updater.json:agent_config` (`node/updater/probe.go`, `node/updater/probe_addresses.go`, `node/enforce/prefixes.go`, `docs/updates/node-updater.md`) | `proposed` (optional) | Optional `probe_peer` (`peer_id`, `/32` `source_v4`, `1..4` public `/32` `destinations[]` with `proto` `"tcp"`/`"udp"` and `port` `1..65535`, optional `resolver_dns`) and `agent_config` (default `/etc/vpn-node/agent.json`, regular file `<= 1 MiB`, `2s` shared DNS deadline, `#154`). Validated at config load, `Engine()`, and each challenge issuance against `enforce.SpecDeniedV4()` (`#156`: IANA special-purpose blocks with `Globally Reachable != True` plus `224.0.0.0/4`, denying `192.88.99.0/24` and allowing `192.31.196.0/24`, `192.52.193.0/24`, `192.175.48.0/24`), live host IPv4 global-unicast interface addresses, `control_url`/`result_url` addresses, and `dns.resolvers` hosts. If an input is unavailable or forbidden at issuance, `probe` is omitted with one fixed `probeRefusal` log per `Engine` while leaving the challenge object otherwise unchanged. Amended `D-064` (`#162`) installs the probe exception only while another process holds `update.lock` on the baseline model with `trustedNow` and durable `probe_issuance`. | `§13.9`, `D-053`, `D-063`, `D-064` (`建議預設`) | `core team` |
| **Signed Egress Probe Targets & Timeout (`egress_probe`, `egress_probe_target`)** | Signed `DesiredState.egress_probe` (`contracts/schemas/desired-state-envelope.schema.json`) + `agent.json:egress_probe_target` | `proposed` | When `DesiredState.egress_probe` is non-null (`1..4` TCP IP-literal targets, `timeout_ms` `100..10000` default `2000` ms), it overrides `agent.json:egress_probe_target`; an unusable signed probe fails closed without falling back to the local target (`node/agent/probe.go`). | `§5`, `§13.9`, `D-058a`, `D-058b` (`建議預設`) | `core team` |

### 1.5 DNS List Sources, Upstream Resolvers, and Update Cadence (`§12`, `§21`, `D-054`, `D-056`, `D-058c`, `D-058d`, `D-061`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **External Threat-Feed Source, Content Hash & License (`feed_code`, `source_url`, `content_sha256`, `license_name`, `license_url`, `commercial_use_basis`)** | Policy `threat_feed` (`contracts/schemas/policies/threat-feed-payload.schema.json`) | `none` (`待決策`) | Commercial threat-feed URLs and license terms are `待決策` (`§12.3`, `§21`). Every revision pins `content_sha256` (64-hex SHA-256) and `commercial_use_basis`; never enable an unlicensed third-party feed in production. | `§12.3`, `§12.5`, `§21` (`待決策`) | `Ted` |
| **Threat-Feed Refresh Cadence & Shrink Guard (`update_interval_seconds`, `feed_update_cadence_seconds`, `feed_max_removal_percent`)** | Policy `threat_feed` (`update_interval_seconds`) + `dns.json:feed_update_cadence_seconds`, `dns.json:feed_max_removal_percent` | `proposed` | `update_interval_seconds` (per-revision optional override) and `feed_update_cadence_seconds` (worker fallback): `300..604800` seconds, default **`21600` (6 hours)** (`D-054`, `D-058d`). `feed_max_removal_percent`: `0..50`, default **`50`** (`D-054`). On fetch/hash/shrink failure, `fvpn-worker` records `dns.feed.refresh_failed` in `dns_feed_refresh_state` and retains the last-good approved snapshot in `dns_feed_contents` (`00181_dns_feed_contents.sql`, `00182_dns_feed_review.sql`). | `§12.3`, `§12.5`, `D-054`, `D-058c`, `D-058d` (`建議預設`) | `core team` |
| **DNS Blocked Response Mode & TTL (`block_mode`, `blocked_ttl_seconds`)** | Policy `dns_profile` (`contracts/schemas/policies/dns-profile-payload.schema.json`) | `proposed` | `block_mode`: `"nxdomain"` (required by `§12.1`). `blocked_ttl_seconds`: `60..3600` (default `300` seconds in `internal/dnspolicy`, `60` seconds in devstack test fixtures; controls the DNS response TTL header only, not an end-to-end propagation SLA, `§12.5`). | `§12.1`, `§12.5`, `§19 AT20, AT22` (`建議預設`) | `core team` |
| **Recursive Upstream Resolvers (`upstreams`, `bootstrap_dns`)** | `dns.json:upstreams`, `dns.json:bootstrap_dns` | `none` (`test only` in `docs/dns/config.example.json`) | Non-empty list of `IP:port` literals (`D-028`). Injected into canonical `resolver_document` (`dns_resolver_documents`, `00180_dns_publish.sql`) and AdGuard Home by `fvpn-worker`. | `§12.2`, `D-028`, `D-056` (`必須`) | `core team` |
| **Node Overlay DNS Resolver Addresses (`dns_resolver_ipv4`, `dns_resolver_ipv6`)** | `nodes.json:dns_resolver_ipv4`, `nodes.json:dns_resolver_ipv6` | `none` (`test only` in `nodestest`) | At least one IPv4 or IPv6 literal required (`§12.2` recommends `≥ 2` resolvers for failover). `node/enforce` permits `TCP/53` and `UDP/53` to `@resolver_v4` (`dns_service_v1`) and drops non-53 traffic (`ISO04`). | `§8.1`, `§12.2`, `ISO04` (`必須`) | `core team` |
| **Node Local AdGuard Home Targets & Credentials (`dns.resolvers[]`, `dns.resolver_credentials_file`)** | Node `/etc/vpn-node/agent.json:dns` | `none` | `1..8` host-local targets (`admin_url` IP literal with explicit port, `dns_addr` `host:port`) and absolute path to a `root:root`-owned (`uid 0`, `gid 0`), mode `0600`, single-hard-link (`Nlink == 1`) JSON file `{"username","password"}` (`D-061`, `node/agent/dnsconfig.go`, `docs/node/README.md`). Reloaded on `SIGHUP` during `fvpn-agent run`. | `§12.2`, `§12.5`, `§16.2`, `D-061` (`必須`) | `core team` |

### 1.6 Data Retention Periods (`§10.5`, `§15.4`, `§16.4`, `§17.3`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Central Raw Usage Payload Retention (`raw_usage_retention_days` / `raw_usage_days`)** | `usage.json:raw_usage_retention_days` + DB `retention_config.raw_usage_days` (`00000000-0000-0000-0000-000000000002`) | `proposed` | `config.UsageConfig` bounds: `30..3650` days (example `90`). DB `retention_config.raw_usage_days` defaults to **`35` days** (`CHECK >= 30`, `00003_ledger_and_work.sql`); DB trigger `F-03` (`purge_retained`) enforces `cutoff <= clock_timestamp() - make_interval(days => raw_usage_days)` and refuses to purge months with `gap = TRUE`. | `§15.4`, `§16.4`, `F-03` (`建議預設`) | `Ted` & `core team` |
| **Central Monthly/Daily Rollup Retention (`rollup_retention_months` / `rollup_months`)** | `usage.json:rollup_retention_months` + DB `retention_config.rollup_months` | `proposed` | `config.UsageConfig` bounds: `12..120` months (example `24`). DB `retention_config.rollup_months` defaults to **`13` months** (`CHECK >= 12`, `00003_ledger_and_work.sql`). | `§15.4`, `§16.4` (`建議預設`) | `Ted` |
| **Central Audit Log Retention (`audit_retention_days` / `audit_days`)** | `usage.json:audit_retention_days` + DB `retention_config.audit_days` | `proposed` | `config.UsageConfig` bounds: `180..3650` days (example `365`). DB `retention_config.audit_days` defaults to **`180` days** (`CHECK >= 90`, `00003_ledger_and_work.sql`), enforced by `purge_retained('audit_events', ...)`. | `§15.4`, `§16.4`, `F-03` (`建議預設`) | `Ted` |
| **Service Log Retention (`service_log_days`)** | DB `retention_config.service_log_days` (`00000000-0000-0000-0000-000000000002`) + journald/logrotate | `proposed` | DB `retention_config.service_log_days` defaults to **`7` days** (`CHECK >= 7`, `00003_ledger_and_work.sql`), enforced by `purge_retained('service_logs', ...)`. | `§15.4`, `§16.4` (`建議預設`) | `core team` |
| **Node Local ACKed Usage Archive Retention** | `node/internal/usagestore` (`DefaultACKedRetention`) | `proposed` | Spec recommended default: **`30` days** (`§10.5`). Unacknowledged events are retained until ACKed or local disk budget (`256 MiB`) triggers fail-closed forwarding block. | `§10.5`, `§17.2` (`建議預設`: `30` days) | `core team` |
| **Billing & Tax Record Retention** | `billing_records`, `subscriptions`, `inbox_events` (`00001_members.sql`) | `none` (`待決策`) | Retained per statutory accounting/tax law in the operating jurisdiction (`§15.4`, `§16.4`, `§21`). Never purged by `purge_retained`. | `§15.4`, `§16.4`, `§21` (`待決策`) | `Ted` |

### 1.7 Node Capacity Ceilings & Datapath Reserves (`§7.2`, `§8.3`, `§10`, `§13.4`, `§20.2`, `§21`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Node Total Capacity Ceiling (`capacity.down_bps`, `capacity.up_bps`)** | `PATCH /api/v1/admin/nodes/{node_id}` (`capacity`) | `none` (`待實測`) | Decimal strings (`decstr.U64`, `> 0`). Used as the `tc` HTB root ceiling on `fvmb0` (`egress`) and `fvup0` (`egress`). If unset (`null`), node uses observed link capacity or policy defaults. | `§7.2`, `§8.3`, `§20.2` (`待實測`) | `core team` |
| **Node Per-Member Ceiling (`capacity.member_down_bps_cap`, `capacity.member_up_bps_cap`)** | `PATCH /api/v1/admin/nodes/{node_id}` (`capacity`) | `none` (`待實測`) | Decimal strings (`decstr.U64`, `> 0`, `≤ down_bps` / `up_bps`). Caps any single member's HTB class rate on that exit node even in Tier 0 (unthrottled). | `§7.2`, `§11.3`, `§20.2` (`待實測`) | `Ted` & `core team` |
| **Host Control-Plane Bandwidth Reserve (`host_reserve_bps`)** | Node `/etc/vpn-node/agent.json:host_reserve_bps` | `none` (`test only` `1000000` in `devstack`) | Mandatory `uint64 > 0` (`D-034`, exits `64` if `0` or omitted). Reserves bandwidth on `fvup0` for mTLS control, heartbeats, usage uploads, and SSH/management (`§8.3`). | `§8.3`, `§20.2`, `D-034` (`待實測`) | `core team` |
| **Updater Disk Free-Space Floor (`required_free_bytes` / `MinFreeBytes`)** | Release manifest `rollback.required_free_bytes` + `node/updater` (`512 MiB` floor) | `proposed` | Updater enforces `free_bytes >= artifact.size_bytes + artifact.installed_size_bytes + max(rollback.required_free_bytes, 536870912)` (`507 DISK_SPACE_LOW` on violation, `§13.4`). | `§13.4`, `§16.3` (`建議預設`: `512 MiB`) | `core team` |

### 1.8 Central HTTP Rate Limiters & Ingress Body Caps (`§16.1`, `D-060`)

| Parameter | Location | Default Status | Repo / Spec Value & Validation Bounds | Spec Section | Decision Owner |
|---|---|---|---|---|---|
| **Login Rate & Per-Subject Window (`login_rate.requests_per_second`, `login_rate.burst`, `login_subject_per_minute`)** | `control.json:login_rate`, `control.json:login_subject_per_minute` (`internal/config/config.go`, `docs/control/AUTH.md`) | `proposed` | Defaults: `login_rate` `5/min` (`1/12` r/s), burst `10` per client IP (`/32` IPv4 or `/64` IPv6 prefix; unparseable `RemoteAddr` refused `429 RATE_LIMITED`); `login_subject_per_minute = 10` (`1..1000`). | `§16.1`, `D-060` (`建議預設`) | `core team` |
| **Dedicated Route Rate Classes (`anonymous_rate`, `heartbeat_rate`, `lease_rate`, `usage_rate`)** | `control.json` (`internal/config/config.go`, `docs/control/README.md`) | `proposed` | Defaults: `anonymous_rate` (`EnrollNode`, `ReceiveProviderWebhook`) `2/s` burst `10`; `heartbeat_rate` (`SubmitObservation`) `1/s` burst `5`; `lease_rate` (`RenewServingLease`) `1/s` burst `5`; `usage_rate` (`SubmitUsageBatch`) `2/s` burst `10`. Remaining browser/agent routes use `browser_rate` (`20/s`, burst `40`) and `agent_rate` (`10/s`, burst `20`). Capacity figures recorded before `2026-10-30` for those five dedicated streams are superseded. | `§16.1`, `D-060` (`建議預設`) | `core team` |
| **Ingress Request Body Caps (`browser_body_bytes`, `agent_body_bytes`, `webhook_body_bytes`, `usage_body_bytes`)** | `control.json` (`internal/config/config.go`, `internal/httpapi/middleware.go`, `docs/control/AUTH.md`) | `proposed` | Defaults (`1024..67108864`): `browser_body_bytes = 1048576` (`1 MiB`), `agent_body_bytes = 1048576` (`1 MiB`, covering `agent`, `heartbeat`, `lease`, and `anonymous` `EnrollNode`), `webhook_body_bytes = 1048576` (`1 MiB` for `ReceiveProviderWebhook`), and `usage_body_bytes = 8388608` (`8 MiB` for `SubmitUsageBatch`, `#161`, `SECOPS-15` closed). Enforced in 4 steps (`docs/control/AUTH.md`): (1) `http.MaxBytesReader` -> `413 PAYLOAD_TOO_LARGE`; (2) `checkJSONDocument` (skipped for webhooks): bodies `<= canonjson.MaxBytes` (`1 MiB`, the signed-document bound) pass `canonjson.Canonicalize`, and larger bodies up to the class cap (`SubmitUsageBatch` up to `8 MiB`) pass one `json.Decoder` read over `io.LimitReader` rejecting trailing data -> `400 MALFORMED_REQUEST`; (3) OpenAPI schema -> `422 VALIDATION_FAILED`; (4) mTLS body identity (`matchingIdentity`, including nested usage events) -> `403 INCARNATION_MISMATCH`. | `§10.3`, `§14.1`, `§14.6`, `§16.1`, `D-060` (`建議預設`) | `core team` |

---

## 2. Step-by-Step Calibration Workflow (`§11.4`, `§20.2`)

### 2.1 Calibrating Per-Node Capacity Ceilings and `host_reserve_bps` (`§20.2`)

Before setting `publication_intent = "published"` on a new exit node hardware profile (`amd64` or `arm64`):
1. Measure sustained NIC throughput and CPU softirq head-room inside an isolated test or staging environment (`docs/poc/G2-netlab.md`).
2. Set `host_reserve_bps` in `/etc/vpn-node/agent.json` to `max(10 Mbps, 2% of uplink capacity)` (e.g., `20000000` = `20 Mbps` on a `1 Gbps` node) so mTLS heartbeats, lease renewals, and usage batch uploads never starve under member congestion.
3. Configure `capacity` on the node via `PATCH /api/v1/admin/nodes/{node_id}` (`If-Match` required) to `85%–90%` of measured sustainable WireGuard + `tc` HTB throughput so `fq_codel` controls queue delay on the node rather than dropping packets in upstream provider buffers.

### 2.2 Calibrating Fair-Use Thresholds from Shadow Mode (`§11.4`, `FUP04`)

1. Launch with `"fup_mode": "shadow"` in `/etc/fvpn/policies.json` and activate an initial `fair_use` policy revision.
2. In `shadow` mode, `fvpn-worker` computes tier transitions (`fup_member_states`) and emits audit/telemetry records, while compiling full unthrottled rates into node desired state and returning `state: "normal"` (`applying: false`, `rate: null`) to members (`docs/policies/README.md`).
3. At the end of the first billing cycle, query the distribution of member monthly usage in PostgreSQL:
   ```sql
   SELECT
     percentile_cont(0.50) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)::double precision) AS p50_bytes,
     percentile_cont(0.90) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)::double precision) AS p90_bytes,
     percentile_cont(0.95) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)::double precision) AS p95_bytes,
     percentile_cont(0.99) WITHIN GROUP (ORDER BY (upload_bytes + download_bytes)::double precision) AS p99_bytes
   FROM usage_rollups
   WHERE granularity = 'month'
     AND period = to_char(now() AT TIME ZONE 'UTC', 'YYYY-MM');
   ```
4. Once `Ted` approves the production `threshold_bytes` and tier rates (`down_bps`, `up_bps`, `minimum_rate`), create and activate a new `fair_use` policy revision (`POST /api/v1/admin/policies` -> `POST /api/v1/admin/policies/{definition_id}/activate`) and switch `"fup_mode": "enforce"`.
