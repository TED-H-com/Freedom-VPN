# Production Deployment Runbook (`docs/handover/deploy.md`)

This runbook guides operators through transitioning Freedom VPN from the local development and E2E lab stack (`deploy/devstack`) to production hosts (`§3`, `§5`–`§8`, `§10`–`§13`, `§15`–`§18`, `§21`).

Every command in this runbook is labeled either `[Verified on dev/test stack]` (executed and verified during the Wave 4 hand-over verification run) or `[production only]` (requires privileged host access, external DNS/TLS certificates, or live third-party provider accounts).

---

## 1. Architecture & Network Topology Overview (`§3`, `§16`, `§18.1`)

Freedom VPN v1 operates as a modular monolith in the central plane (`fvpn-control` + `fvpn-worker` sharing PostgreSQL 17 and a signing vault) paired with stateless/state-split Ubuntu 24.04 LTS exit nodes (`fvpn-agent`, `fvpn-updater`, `fvpn-watchdog`) and self-hosted NetBird CE (`0.82.0`) and AdGuard Home (`v0.107.59`).

| Component | Recommended Host / Placement | Exposed Endpoints / Ports | Documentation Example Address |
|---|---|---|---|
| **PostgreSQL 17** | Dedicated DB host or private subnet | `5432/tcp` (internal TLS only, never public) | `198.51.100.10:5432` (`db.internal.example.com`) |
| **`fvpn-control`** | Central API host (behind reverse proxy or direct TLS) | `:8443/tcp` (Public HTTPS: Portal, OIDC, Webhooks, `/agent/enroll`)<br>`:9443/tcp` (Agent mTLS: `/api/v1/agent/*` except enroll)<br>`127.0.0.1:9091/tcp` (Prometheus `/metrics`) | `https://vpn.example.com` (`198.51.100.20`)<br>`https://agent.vpn.example.com:9443` |
| **`fvpn-worker`** | Co-located with `fvpn-control` or dedicated worker host sharing the signing vault | `127.0.0.1:9092/tcp` (Prometheus `/metrics` & `/healthz`, `/readyz`) | `198.51.100.20` / `198.51.100.21` |
| **OIDC Identity Provider** | External managed OIDC or self-hosted IdP | `443/tcp` (HTTPS) | `https://id.example.com` |
| **NetBird CE (`0.82.0`)** | Dedicated overlay coordination host | `443/tcp` (Management API `/api`, gRPC, Signal, Relay)<br>`3478/udp` (STUN/TURN) | `https://netbird.example.com` (`198.51.100.30`) |
| **AdGuard Home (`v0.107.59`)** | At least 2 resolver instances reachable over the overlay (`§12.2`) | `53/udp`, `53/tcp` (Overlay DNS for `100.64.0.0/10`)<br>`8080/tcp` (Admin HTTP API, restricted to `fvpn-worker` and `fvpn-agent`) | `100.64.0.53`, `100.64.0.54` (IPv6: `2001:db8::53`, `2001:db8::54`) |
| **Release Origin** | Immutable HTTPS object storage / static origin (`§13.3`, `§16.2`) | `443/tcp` (HTTPS read-only to nodes) | `https://releases.example.com` (`203.0.113.10`) |
| **Exit Nodes** | Ubuntu 24.04 LTS (`amd64` or `arm64`) | WireGuard UDP port (`51820/udp`), outbound HTTPS (`443/tcp`, `9443/tcp`) | Public egress `192.0.2.20`, Overlay `100.64.0.100` |

---

## 2. Repository `TEST ONLY` Inventory and Production Replacements (`§1`, `AGENTS.md §1, §3, §8`)

By engineering contract (`D-026`, `D-039`, `D-045`), no production binary compiles in signing keys or business defaults. Every `TEST ONLY` value present in repository examples, test fixtures (`*test`), or `deploy/devstack` must be replaced before production startup:

| Repository Location | Setting / Artifact | `TEST ONLY` Value in Repo | Production Replacement |
|---|---|---|---|
| `docs/control/config.example.json` | `public_origin`, `oidc_issuer`, `oidc_client_id` | `https://portal.example.com`, `https://idp.example.com`, `fvpn-portal-test` | Real HTTPS portal origin (`https://vpn.example.com`), production OIDC issuer URL, and confidential OAuth client ID |
| `docs/members/config.example.json` | `live_mode` | `false` | `true` (enforces Stripe live-mode event check in `billing.Webhook`) |
| `docs/members/config.example.json` | `price_ids` | `["price_test_monthly"]` | Approved live Stripe Price IDs (must match active `plan` policy payload `provider_price_ids`) |
| `docs/members/config.example.json` | `vault_active_key`, `vault_key_files` | `"v1-test"`, `/etc/fvpn/secrets/billing-vault-v1.key` | Production key ID (e.g., `"vault-prod-2026-01"`) pointing to a `0600` file containing 32 bytes of CSPRNG output |
| `docs/members/config.example.json` | `grace_seconds`, `trial_max_seconds`, `grant_max_seconds` | `0`, `0`, `86400` | Explicit business-approved durations (see [`calibration.md`](calibration.md)) |
| `internal/nodes/nodestest/config.go` | `netbird_deployment`, `netbird_token_file` | `"TEST ONLY"`, `"TEST ONLY synthetic token"` | Production deployment identifier (e.g., `"prod-main"`) and restricted NetBird service-user PAT file (`0600`) |
| `internal/nodes/nodestest/config.go` | `certificate_seconds`, `renewal_overlap_seconds`, `bootstrap_seconds`, `online_seconds`, `stuck_seconds` | `86400`, `300`, `60`, `180`, `600` | Calibrated production PKI and liveness windows (e.g., 7–30 day node certs with 24–72h overlap; see [`calibration.md`](calibration.md)) |
| `internal/nodes/nodestest/config.go` | `release_origins` | `["https://releases.example.invalid"]` | Exact HTTPS origin(s) of the immutable production release bucket (`["https://releases.example.com"]`) |
| `internal/nodes/nodestest/config.go` | `dns_resolver_ipv4`, `dns_resolver_ipv6` | `["100.64.0.53"]`, `["2001:db8::53"]` | Production AdGuard Home overlay listener IPs (at least 2 IPv4 addresses recommended per `§12.2`) |
| `docs/dns/config.example.json` | `upstreams`, `bootstrap_dns` | `["192.0.2.53:53", "[2001:db8::53]:53"]` | Production recursive upstream resolvers (`IP:port` literals only; never left to AdGuard defaults, `D-028`) |
| `internal/policies/policiestest/config.go` | `dns_upstream_profile` | `"TEST ONLY"` | Production upstream profile identifier matching `dns_profile` policy payloads (e.g., `"prod-recursive-v1"`) |
| `internal/usage/usagetest/config.go` | `signing_config_file` | Temp dir `usage-vault` | `/var/lib/fvpn/vault` (persistent `0700` directory outside PostgreSQL backup scope) |
| `tests/e2e/cmd/fvpn-e2e-prepare/main.go` | `host_reserve_bps` in `agent.json` | `1000000` (1 Mbps) | Measured host control-plane reserve per node NIC capacity (e.g., `20000000` for 20 Mbps; mandatory `> 0`, exits `64` if omitted, `D-034`) |
| `tests/e2e/cmd/fvpn-e2e-prepare/main.go` | `test_mode`, `test_exempt_v4` in `agent.json` | `true` | Omit or set `false` (`test_exempt_v4` is rejected by `fvpn-agent` when `test_mode` is `false`) |
| `node/updater/host.go` | `allow_loopback`, `allowed_artifact_origins` in `updater.json` | `true` / lab RFC1918 exemptions | Set `allow_loopback: false` and `allowed_artifact_origins: []` so the private/CGNAT/loopback artifact guard remains enforced (`D-046`) |
| `internal/dnspolicy/cmd/bootstrap/main.go` | AdGuard admin password | `"testonly-<hex>"` | High-entropy random password per resolver, stored in `/etc/fvpn/secrets/adguard.password` (`0600`) and hashed with bcrypt in `AdGuardHome.yaml` |
| `contracts/vectors/*.json` | Ed25519 seed vectors | `"TEST ONLY - NOT A SECRET"` | Freshly generated hardware/host keys via `signing.InitializeFileProvider` and `fvpn-release keygen` |

---

## 3. PostgreSQL 17 Provisioning, Roles, and WAL Archiving (`§3.1`, `§15.4`, `§17.2`)

### 3.1 PostgreSQL 17 Server & WAL Archiving Configuration

Provision PostgreSQL 17 on a dedicated volume with TLS enabled and continuous WAL archiving to an off-host object store so point-in-time recovery (PITR) meets the RPO target (`≤ 15 minutes` for central control-plane tables, `§17.2`).

In `postgresql.conf` `[production only]`:

```ini
ssl = on
ssl_cert_file = '/etc/postgresql/17/main/server.crt'
ssl_key_file = '/etc/postgresql/17/main/server.key'
password_encryption = scram-sha-256

# Continuous WAL archiving (§17.2)
wal_level = replica
archive_mode = on
archive_timeout = 300s
archive_command = 'pgbackrest --stanza=fvpn archive-push %p'
max_wal_senders = 5
```

### 3.2 Least-Privilege Database Roles (`docs/db/README.md`)

Never run `fvpn-control` or `fvpn-worker` as `postgres` superuser. Create the six role-separated database accounts (`§15.3`, `§17.3`):

| Role | Used By | Purpose |
|---|---|---|
| `fvpn_migrator` | `fvpn-migrate up` | Schema owner; creates tables, functions, and base triggers (`00001`–`00180`). |
| `fvpn_control` | `fvpn-control` | CRUD on operational tables; `INSERT` + `SELECT` on `audit_events` and `usage_events` (no `UPDATE`/`DELETE` on append-only tables, `F-02`–`F-04`). |
| `fvpn_worker` | `fvpn-worker` | Claims `outbox_jobs`, reconciles NetBird/Stripe/DNS, updates `usage_rollups`. |
| `fvpn_retention` | Retention job | Executes `SELECT purge_retained(...)` (`F-03`) to delete expired raw payloads and logs while recording an audit row. |
| `fvpn_recovery` | `fvpn-usage-admin` | Executes usage ledger freeze, epoch rotation (`usage_recovery_records`, `usage_ledger_epochs`), and adjustments. |
| `fvpn_backup` | `pg_dump` / `pgBackRest` | Read-only physical/logical backup role. |

Create the database and roles `[production only]`:

```sql
CREATE DATABASE fvpn;
\c fvpn

CREATE ROLE fvpn_migrator WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_1';
CREATE ROLE fvpn_control  WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_2';
CREATE ROLE fvpn_worker   WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_3';
CREATE ROLE fvpn_retention WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_4';
CREATE ROLE fvpn_recovery WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_5';
CREATE ROLE fvpn_backup   WITH LOGIN PASSWORD 'REPLACE_WITH_SCRAM_SECRET_6';

GRANT ALL PRIVILEGES ON DATABASE fvpn TO fvpn_migrator;
ALTER SCHEMA public OWNER TO fvpn_migrator;
```

### 3.3 Running Forward Migrations (`fvpn-migrate`)

Build and run `fvpn-migrate` using `FVPN_DATABASE_URL` (`fvpn-migrate` never logs or prints connection credentials):

`[Verified on dev/test stack]`
```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -o /usr/local/bin/fvpn-migrate ./internal/db/cmd/fvpn-migrate

FVPN_DATABASE_URL="postgres://fvpn_migrator:PASSWORD@198.51.100.10:5432/fvpn?sslmode=verify-full&sslrootcert=/etc/fvpn/certs/db-ca.pem" \
  /usr/local/bin/fvpn-migrate up

FVPN_DATABASE_URL="postgres://fvpn_migrator:PASSWORD@198.51.100.10:5432/fvpn?sslmode=verify-full&sslrootcert=/etc/fvpn/certs/db-ca.pem" \
  /usr/local/bin/fvpn-migrate status
```

Expected output (`19` migrations applied, including `00006` which aligns `purge_retained()` audit rows with canonical `<resource_type>:<uuid>` format per `D-035`):

```text
current_database="fvpn"
00001 applied
00002 applied
00003 applied
00004 applied
00005 applied
00006 applied
00100 applied
00101 applied
00110 applied
00111 applied
00120 applied
00121 applied
00130 applied
00131 applied
00140 applied
00150 applied
00160 applied
00161 applied
00180 applied
```

After migrations complete, grant table/sequence permissions to `fvpn_control`, `fvpn_worker`, `fvpn_retention`, `fvpn_recovery`, and `fvpn_backup` (`pg_read_all_data`) `[production only]`:

```sql
GRANT USAGE ON SCHEMA public TO fvpn_control, fvpn_worker, fvpn_retention, fvpn_recovery, fvpn_backup;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO fvpn_control, fvpn_worker, fvpn_recovery;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO fvpn_control, fvpn_worker, fvpn_recovery;
GRANT EXECUTE ON FUNCTION purge_retained(text, timestamptz, text) TO fvpn_retention, fvpn_worker;
GRANT pg_read_all_data TO fvpn_backup;
-- Note: Base triggers in 00004_guards.sql (F-02, F-03, F-04) block UPDATE/DELETE on
-- audit_events, usage_events, usage_samples, usage_checkpoints, and usage_rollups
-- regardless of table-level GRANTs, unless inside purge_retained() or fvpn.restore='on'.
```

---

## 4. External Dependencies Setup

### 4.1 OIDC Provider (`§9.1`, `§16.1`, `docs/control/AUTH.md`, `docs/portal/AUTH.md`)

1. Configure a single OIDC issuer (`https://id.example.com`) shared by the Portal (`fvpn-control`) and NetBird (`§7.3`, `§9.1`, `§15.2`):
   - **Portal Client (`oidc_client_id`)**: Confidential client using Authorization Code + PKCE (`S256`).
     - Allowed Redirect URI: `https://vpn.example.com/auth/callback`
     - Subject identifier type: `public` (so `iss` + `sub` matches across Portal and NetBird clients, allowing central device synchronization `member.device_authorize` / `members.devicereconcile` to verify ownership and authorize `MemberDevice` records automatically without manual peer entry, `§7.3`, `§9.4`).
   - **Admin Role Claim & MFA (`§9.1`, `§16.1`)**:
     - Emit claim `roles` containing `"admin"` (or configure `admin_claim` and `admin_value` in `control.json`).
     - Enforce MFA at the IdP for administrators and emit `amr: ["mfa"]` (or a configured `mfa_acr` value). `fvpn-control` rejects `/auth/login?role=admin` with `403 ADMIN_MFA_REQUIRED` if neither `mfa_amr` nor `mfa_acr` matches, and requires re-authentication within `recent_auth_seconds` (`900s`) for high-risk mutations (`POST /admin/nodes/{id}/revoke`, `DELETE /admin/nodes/{id}`, `POST /admin/update-batches`, `POST /admin/policies/{kind}/revisions/{rev}/activate`).

### 4.2 NetBird CE (`0.82.0` Management, Signal, Relay, STUN) (`§7.1`, `§7.2`, `docs/netbird/VERSIONS.md`)

1. Pin the server image to `netbirdio/netbird-server:0.82.0` (`sha256:7b5b53b2dc8bc25639f791cf72d47cf86911894ff3f8a97cb79bab9be958109c`) and dashboard `netbirdio/dashboard:v2.96.0`.
2. Configure `/etc/netbird/config.yaml` with persistent storage, TLS via reverse proxy (`https://netbird.example.com`), and the shared OIDC issuer `[production only]`:
   ```yaml
   server:
     listenAddress: ":8080"
     exposedAddress: "https://netbird.example.com"
     stunPorts: [3478]
     metricsPort: 9090
     healthcheckAddress: ":9000"
     logLevel: "warn"
     logFile: "console"
     authSecret: "REPLACE_WITH_64_HEX_CSPRNG"
     dataDir: "/var/lib/netbird"
     disableAnonymousMetrics: true
     disableGeoliteUpdate: true
     auth:
       issuer: "https://id.example.com"
       dashboardRedirectURIs:
         - "https://netbird.example.com/nb-auth"
         - "https://netbird.example.com/nb-silent-auth"
       cliRedirectURIs:
         - "http://localhost:53000/"
     store:
       engine: "sqlite"
       encryptionKey: "REPLACE_WITH_BASE64_32_BYTE_KEY"
   ```
3. **Mandatory Security Baseline (`§7.1`, `ISO01`)**:
   - Create a dedicated NetBird **Service User** with the `admin` role and generate a Personal Access Token (PAT). Write the PAT to `/etc/fvpn/secrets/netbird.token` (`chmod 0600`). Never place this token in `apps/portal` or on exit nodes.
   - Delete or disable NetBird's built-in `Default` (`All-to-All`) access control policy immediately (`internal/netbird.Client.RemoveDefaultPolicy` / `BootstrapAccess`). Member-to-member traffic must be blocked at both the NetBird control plane and the exit-node `inet fvpn` nftables data plane.

### 4.3 AdGuard Home Resolvers (`v0.107.59`) (`§12`, `docs/dns/BACKEND.md`, `docs/dns/PRIVACY.md`)

1. Deploy at least **two** AdGuard Home instances (`v0.107.59`, image digest `sha256:5d5e3aef39a8c8906a324f0c2da0a6123929e459b57c1e3f05e401dca5c7bf02`) reachable on the overlay network (e.g., `100.64.0.53` and `100.64.0.54`).
2. **Mandatory Privacy Hardening (`SEC02`, `§12.7`)**:
   - In `AdGuardHome.yaml` (`schema_version: 29`), ensure:
     - `querylog.enabled: false` and `querylog.file_enabled: false` (never log member DNS queries to disk or memory).
     - `statistics.enabled: false`.
     - `dns.anonymize_client_ip: true`.
     - `dns.fallback_dns: []` (never fall back to unconfigured public resolvers).
     - `filtering.filtering_enabled: true`, `filtering.protection_enabled: true`, `filtering.blocking_mode: nxdomain`.
   - Bind the directory `/opt/adguardhome/conf` (not the single file) when running under Docker so atomic `AdGuardHome.yaml` rewrites succeed.

### 4.4 Immutable Release Origin (`§13.3`, `§16.2`)

1. Provision an HTTPS origin (`https://releases.example.com`) backed by object storage or an immutable file server.
2. Enable write-once / object-lock rules so existing paths under `/<release_id>/*` cannot be overwritten.
3. Nodes only require outbound HTTPS `GET` access to `https://releases.example.com`; the release signing private key never touches the origin server or central API servers.

---

## 5. Key Ceremonies (`§6.4`, `§13.3`, `§16.2`, `§17.3`, `docs/pki/rotation.md`)

### 5.1 Central Signing Vault & Node CA Ceremony (`FileProvider`)

The central signing directory (`/var/lib/fvpn/vault`, mode `0700`) stores six central key purposes outside PostgreSQL backups (`§16.2`, `§17.2`):
- `node-ca` (ECDSA P-256): Signs node mTLS client certificates (`§6.4`).
- `bootstrap-secret` (AES-256-GCM): Encrypts one-time NetBird setup keys at rest in PostgreSQL (`§6.2`).
- `desired-state` (Ed25519): Signs node desired-state envelopes (`§14.7`).
- `lease` (Ed25519): Signs node serving leases (`§15.3`, `§16.3`).
- `recovery` (Ed25519): Signs usage checkpoint envelopes returned on `GET /api/v1/agent/usage-checkpoint` (`§10.8`, `§17.3`).
- `ledger-epoch` (Ed25519, **central-only**): Signs `usage_recovery_records` when rotating `ledger_epoch` (`§17.3`). Never exported into the node trust bundle.

### 5.2 Offline Release Signing Key Ceremony (`fvpn-release keygen`)

On an isolated build/release workstation (never on a central server or exit node), generate the Ed25519 `release` signing key and export its initial public trust store:

`[Verified on dev/test stack]`
```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -o /usr/local/bin/fvpn-release ./packaging/release/cmd/fvpn-release

/usr/local/bin/fvpn-release keygen \
  -key-id release-prod-2026-01 \
  -out /secure/offline/release-prod-2026-01.key.json \
  -trust-out /tmp/release-trust.json \
  -not-before 2026-01-01T00:00:00Z \
  -not-after 2027-01-01T00:00:00Z
```

Output:
```text
key_id=release-prod-2026-01
purpose=release
fvpn-release: private key stays on the build machine; do not copy it to nodes
```

### 5.3 Initializing the Central Vault and Pinning the Release Public Key

Run a one-time initialization ceremony on the central host to populate `/var/lib/fvpn/vault`, generate `/etc/fvpn/secrets/agent-ca.pem`, pin the offline `release` public key via `PinReleaseRoot`, and export `/etc/vpn-node/trust.json` for nodes `[production only]`:

```go
// Ceremony steps executed via internal/signing and internal/pki (see cmd/fvpn-e2e-prepare for reference):
// 1. vault, _ := signing.InitializeFileProvider("/var/lib/fvpn/vault")
// 2. caKeyID, _ := vault.Rotate(ctx, signing.NodeCA, 365*24*time.Hour, 30*24*time.Hour)
// 3. For each purpose in [BootstrapSecret, DesiredState, Lease, Recovery, LedgerEpoch]:
//      vault.Rotate(ctx, purpose, 180*24*time.Hour, 14*24*time.Hour)
// 4. Export Node CA certificate PEM to /etc/fvpn/secrets/agent-ca.pem (record caKeyID for nodes.json)
// 5. Pin offline release public key from /tmp/release-trust.json:
//      vault.PinReleaseRoot(ctx, releaseTrustedKey)
// 6. Export node trust bundle (contains desired-state, lease, recovery, release; excludes ledger-epoch):
//      bundle, _ := vault.TrustBundle(ctx) -> write to /etc/vpn-node/trust.json
```

Verify that `/var/lib/fvpn/vault` is owned by `fvpn:fvpn` with permissions `0700` and all `.key` / `state.json` files are `0600`.

---

## 6. Central Configuration Files (`D-026`, `D-045`)

Both `fvpn-control` and `fvpn-worker` read `/etc/fvpn/control.json` and validate all five domain configuration files at startup. Any unknown JSON field, trailing data, or missing file aborts startup immediately with `config.SettingError`.

### 6.1 `/etc/fvpn/control.json`

```json
{
  "public_address": ":8443",
  "agent_address": ":9443",
  "metrics_address": "127.0.0.1:9091",
  "public_origin": "https://vpn.example.com",
  "tls_certificate_file": "/etc/fvpn/secrets/tls.crt",
  "tls_key_file": "/etc/fvpn/secrets/tls.key",
  "agent_ca_file": "/etc/fvpn/secrets/agent-ca.pem",
  "database_url_file": "/etc/fvpn/secrets/database.url",
  "oidc_issuer": "https://id.example.com",
  "oidc_client_id": "fvpn-portal-prod",
  "oidc_secret_file": "/etc/fvpn/secrets/oidc.secret",
  "admin_claim": "roles",
  "admin_value": "admin",
  "mfa_amr": ["mfa"],
  "mfa_acr": [],
  "recent_auth_seconds": 900,
  "session_idle_seconds": 1800,
  "session_absolute_seconds": 43200,
  "login_ttl_seconds": 300,
  "idempotency_seconds": 604800,
  "read_timeout_seconds": 15,
  "write_timeout_seconds": 30,
  "idle_timeout_seconds": 60,
  "shutdown_seconds": 30,
  "browser_body_bytes": 1048576,
  "agent_body_bytes": 1048576,
  "usage_body_bytes": 8388608,
  "webhook_body_bytes": 1048576,
  "max_concurrent": 128,
  "max_rate_principals": 10000,
  "ingress_rate": { "per_second": 20, "burst": 60 },
  "browser_rate": { "per_second": 10, "burst": 30 },
  "agent_rate": { "per_second": 5, "burst": 10 },
  "heartbeat_rate": { "per_second": 1, "burst": 5 },
  "lease_rate": { "per_second": 1, "burst": 5 },
  "usage_rate": { "per_second": 2, "burst": 10 },
  "anonymous_rate": { "per_second": 2, "burst": 10 },
  "worker_concurrency": 4,
  "worker_lease_seconds": 60,
  "worker_job_seconds": 40,
  "worker_poll_seconds": 1,
  "worker_attempts": 10,
  "worker_backoff_seconds": 1,
  "worker_backoff_max_seconds": 300,
  "migrate_on_start": false,
  "members_config_file": "/etc/fvpn/members.json",
  "nodes_config_file": "/etc/fvpn/nodes.json",
  "dns_config_file": "/etc/fvpn/dns.json",
  "policies_config_file": "/etc/fvpn/policies.json",
  "usage_config_file": "/etc/fvpn/usage.json"
}
```

### 6.2 `/etc/fvpn/members.json` (`members_config_file`)

```json
{
  "provider": "stripe",
  "api_key_file": "/etc/fvpn/secrets/stripe.key",
  "webhook_secret_files": [
    "/etc/fvpn/secrets/stripe-webhook-current.secret"
  ],
  "vault_active_key": "vault-prod-2026-01",
  "vault_key_files": {
    "vault-prod-2026-01": "/etc/fvpn/secrets/billing-vault-2026-01.key"
  },
  "live_mode": true,
  "price_ids": [
    "price_prod_standard_monthly"
  ],
  "return_urls": [
    "https://vpn.example.com/member/billing"
  ],
  "full_refund_policy": "revoke_original_period",
  "grace_seconds": 259200,
  "grace_reason": "payment_past_due_grace",
  "trial_max_seconds": 0,
  "grant_max_seconds": 2592000,
  "device_limit": 3,
  "portal_ttl_seconds": 60,
  "billing_lease_seconds": 120,
  "max_webhook_bytes": 1048576,
  "reconcile_seconds": 900,
  "device_sync_seconds": 60,
  "netbird_url": "https://netbird.example.com/api",
  "netbird_deployment": "prod-main",
  "netbird_token_file": "/etc/fvpn/secrets/netbird.token",
  "overlay_scope": "prod-overlay"
}
```

### 6.3 `/etc/fvpn/nodes.json` (`nodes_config_file`)

```json
{
  "provider_directory": "/var/lib/fvpn/vault",
  "ca_certificate_file": "/etc/fvpn/secrets/agent-ca.pem",
  "ca_key_id": "0196ed00-0000-7000-8000-000000000010",
  "certificate_seconds": 604800,
  "renewal_overlap_seconds": 86400,
  "bootstrap_seconds": 600,
  "installer_seconds": 900,
  "preinstalled_seconds": 604800,
  "authorization_seconds": 3600,
  "online_seconds": 180,
  "stuck_seconds": 900,
  "reconcile_seconds": 60,
  "fetch_seconds": 30,
  "desired_state_seconds": 600,
  "netbird_url": "https://netbird.example.com/api",
  "netbird_deployment": "prod-main",
  "netbird_token_file": "/etc/fvpn/secrets/netbird.token",
  "dns_resolver_ipv4": [
    "100.64.0.53",
    "100.64.0.54"
  ],
  "dns_resolver_ipv6": [
    "2001:db8::53",
    "2001:db8::54"
  ],
  "release_origins": [
    "https://releases.example.com"
  ]
}
```

### 6.4 `/etc/fvpn/dns.json` (`dns_config_file`)

```json
{
  "upstreams": [
    "198.51.100.53:53",
    "[2001:db8::153]:53"
  ],
  "bootstrap_dns": [
    "198.51.100.53:53"
  ]
}
```

### 6.5 `/etc/fvpn/policies.json` (`policies_config_file`)

```json
{
  "fup_mode": "shadow",
  "fup_sweep_seconds": 60,
  "fup_batch_size": 256,
  "dns_upstream_profile": "prod-recursive-v1"
}
```
> **Note (`§11.4`, `FUP04`)**: Start v1 production with `"fup_mode": "shadow"` until at least one full billing cycle of usage and tier transition logs is reviewed and calibrated (`calibration.md`).

### 6.6 `/etc/fvpn/usage.json` (`usage_config_file`)

```json
{
  "signing_config_file": "/var/lib/fvpn/vault",
  "raw_usage_retention_days": 90,
  "rollup_retention_months": 24,
  "audit_retention_days": 365,
  "projection_batch_size": 256
}
```

---

## 7. Deploying `fvpn-control`, `fvpn-worker`, and `apps/portal` (`§3.1`, `§14`, `§18`)

### 7.1 Building Central Binaries and Portal Assets

`[Verified on dev/test stack]`
```bash
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o dist/fvpn-control ./apps/control/cmd/fvpn-control
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o dist/fvpn-worker  ./apps/worker/cmd/fvpn-worker
CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 go build -trimpath -o dist/fvpn-usage-admin ./internal/usage/cmd/fvpn-usage-admin

cd apps/portal
npm ci
npm run typecheck
npm run lint
npm test
npm run build
```

### 7.2 Starting `fvpn-control` and `fvpn-worker` under systemd

Install systemd units for `fvpn-control.service` and `fvpn-worker.service` running as an unprivileged `fvpn` system user `[production only]`:

```ini
# /etc/systemd/system/fvpn-control.service
[Unit]
Description=Freedom VPN Central Control API
After=network-online.target postgresql.service
Wants=network-online.target

[Service]
Type=simple
User=fvpn
Group=fvpn
ExecStart=/usr/local/bin/fvpn-control -config /etc/fvpn/control.json
Restart=on-failure
RestartSec=2s
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/fvpn/vault
ReadOnlyPaths=/etc/fvpn

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/fvpn-worker.service
[Unit]
Description=Freedom VPN Background Worker
After=network-online.target fvpn-control.service
Wants=network-online.target

[Service]
Type=simple
User=fvpn
Group=fvpn
Environment=FVPN_METRICS_ADDRESS=127.0.0.1:9092
ExecStart=/usr/local/bin/fvpn-worker -config /etc/fvpn/control.json
Restart=on-failure
RestartSec=2s
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/fvpn/vault
ReadOnlyPaths=/etc/fvpn

[Install]
WantedBy=multi-user.target
```

Verify readiness `[Verified on dev/test stack]`:
```bash
curl -fsS https://vpn.example.com:8443/healthz
curl -fsS https://vpn.example.com:8443/readyz
```

---

## 8. Building & Registering the First Release and Seeding Initial Policies (`§13.3`, `§14.5`)

Before enrolling any exit node, register the initial node software release (`amd64` + `arm64`) and activate the baseline policy revisions (`release_support`, `fair_use`, `domain_rule_set`, `dns_profile`, `plan`).

### 8.1 Packaging and Signing the Initial Node Release

Cross-compile the node binaries for both `linux/amd64` and `linux/arm64` (`AGENTS.md §3`, `§13.4`):

`[Verified on dev/test stack]`
```bash
for ARCH in amd64 arm64; do
  mkdir -p "dist/release-${ARCH}"
  CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH="${ARCH}" go build -trimpath -o "dist/release-${ARCH}/fvpn-agent"    ./node/agent/cmd/fvpn-agent
  CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH="${ARCH}" go build -trimpath -o "dist/release-${ARCH}/fvpn-updater"  ./node/updater/cmd/fvpn-updater
  CGO_ENABLED=0 GOTOOLCHAIN=local GOOS=linux GOARCH="${ARCH}" go build -trimpath -o "dist/release-${ARCH}/fvpn-watchdog" ./node/watchdog/cmd/fvpn-watchdog
  /usr/local/bin/fvpn-release package \
    -dir "dist/release-${ARCH}" \
    -out "dist/node-ubuntu2404-${ARCH}.tar.zst" \
    -mtime 1793299200
done
```

Build, sign, and verify the release manifest envelope (`manifest.envelope.json`):

`[Verified on dev/test stack]`
```bash
/usr/local/bin/fvpn-release build-manifest \
  -in build-input.json \
  -out dist/manifest.signed.json

/usr/local/bin/fvpn-release sign \
  -signed dist/manifest.signed.json \
  -key /secure/offline/release-prod-2026-01.key.json \
  -out dist/manifest.envelope.json

/usr/local/bin/fvpn-release verify \
  -envelope dist/manifest.envelope.json \
  -trust /etc/vpn-node/release-trust.json \
  -artifact amd64=dist/node-ubuntu2404-amd64.tar.zst \
  -artifact arm64=dist/node-ubuntu2404-arm64.tar.zst
```

Upload `node-ubuntu2404-amd64.tar.zst`, `node-ubuntu2404-arm64.tar.zst`, `manifest.envelope.json`, `sbom.spdx.json`, `NOTICES.txt`, and `source.tar.zst` to `https://releases.example.com/<release_id>/`, then register the release with `fvpn-control` (`POST /api/v1/admin/releases`) and activate a `release_support` policy revision containing `<release_id>` in both `approved_release_ids` and `serving_release_ids` (`§13.4`).

---

## 9. First Exit-Node Provisioning and Enrollment (`§6`, `§8`, `§13.5`, `§16.3`)

### 9.1 Node Directory Layout and Configuration (`AGENTS.md §4`)

On the Ubuntu 24.04 LTS exit node, create the split directory hierarchy:

`[production only]`
```bash
install -d -m 0755 /opt/vpn-node/releases /etc/vpn-node
install -d -m 0700 /var/lib/vpn-node/identity /var/lib/vpn-node/state /var/lib/vpn-node/update-journal /var/lib/vpn-node/update-staging
```

Extract the initial verified release into `/opt/vpn-node/releases/<release_id>` and point the relative symlink `/opt/vpn-node/current` to `releases/<release_id>`:

`[production only]`
```bash
ln -sfn "releases/0196ed00-0000-7000-8000-000000000701" /opt/vpn-node/current
```

Copy the central TLS CA (`/etc/vpn-node/ca.pem`), the node trust bundle (`/etc/vpn-node/trust.json`), and the release trust bundle (`/etc/vpn-node/release-trust.json`, `0644`).

Write `/etc/vpn-node/agent.json` (`0600`) `[production only]`:

```json
{
  "state_dir": "/var/lib/vpn-node/state",
  "identity_dir": "/var/lib/vpn-node/identity",
  "journal_dir": "/var/lib/vpn-node/update-journal",
  "lock_path": "/var/lib/vpn-node/update.lock",
  "control_url": "https://agent.vpn.example.com:9443",
  "server_ca_file": "/etc/vpn-node/ca.pem",
  "trust_bundle": "/etc/vpn-node/trust.json",
  "datapath": {
    "member_ingress": "fvmb0",
    "uplink": "fvup0",
    "snat": "192.0.2.20"
  },
  "agent_release": "2026.10.1",
  "release_id": "0196ed00-0000-7000-8000-000000000701",
  "renew_lead_seconds": 86400,
  "heartbeat_seconds": 10,
  "watchdog_stale_seconds": 30,
  "watchdog_interval_seconds": 5,
  "poll_seconds": 10,
  "observation_seconds": 30,
  "host_reserve_bps": 20000000,
  "netbird_backend": "kernel",
  "updater_recover": "/opt/vpn-node/current/fvpn-updater",
  "updater_config": "/etc/vpn-node/updater.json"
}
```
> **Important (`AGENTS.md §4`, `D-034`)**:
> - `datapath.member_ingress` and `datapath.uplink` **must** start with `fv` (`^fv[a-z0-9_-]{1,13}$`) and must differ.
> - `host_reserve_bps` **must** be greater than zero (otherwise `fvpn-agent` exits with code `64` before touching the kernel).
> - Never set `test_mode: true` on production nodes.

### 9.2 Issuing a One-Time Enrollment Token (`POST /api/v1/admin/enrollments`)

In the Admin Portal (`/admin/nodes`) or via API, issue an `installer` token (default TTL 15 minutes / `900s`):

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/enrollments" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "enrollment_mode": "installer",
    "intended_class": "standard",
    "intended_region": "tw-tpe",
    "note": "First production exit node tw-tpe-01"
  }'
```

The response carries `Cache-Control: no-store` and returns the one-time `enrollment_token` (`§6.1`).

### 9.3 Running `fvpn-agent enroll` on the Node

Write the token to a `0600` file on the node and run `fvpn-agent enroll` (for the initial enrollment request, point `control_url` at the public listener `https://vpn.example.com:8443` or route `/api/v1/agent/enroll` on `:8443`):

`[Verified on dev/test stack]`
```bash
umask 077
printf '%s\n' "${ENROLLMENT_TOKEN}" > /run/vpn-node/enrollment.token

/opt/vpn-node/current/fvpn-agent enroll \
  -config /etc/vpn-node/agent-enroll.json \
  -token-file /run/vpn-node/enrollment.token
```

What happens during `fvpn-agent enroll` (`§6.1`–`§6.3`, `ENR01`–`ENR04`):
1. Generates an ECDSA P-256 private key inside `/var/lib/vpn-node/identity` (`0600`, never leaves the node) and submits a PKCS#10 CSR + `fvpn-enrollment-v1` proof to `POST /api/v1/agent/enroll`.
2. Receives `node_id`, `incarnation_id`, and the signed mTLS client certificate.
3. Zero-overwrites, syncs, and unlinks `/run/vpn-node/enrollment.token` (`agent.NeutralizeToken`).
4. Calls `POST /api/v1/agent/bootstrap` over mTLS (`:9443`) to fetch the one-time NetBird `setup_key` (subsequent calls return `410 BOOTSTRAP_CONSUMED`).

### 9.4 Installing Node systemd Units and Publishing the Exit (`§7.3`, `§13.5`, `§16.3`)

Install the three systemd units from `node/agent/systemd/` (`fvpn-updater-recover.service`, `fvpn-agent.service`, `fvpn-watchdog.service`) and enable them `[production only]`:

```bash
cp /opt/vpn-node/current/systemd/fvpn-updater-recover.service /etc/systemd/system/
cp /opt/vpn-node/current/systemd/fvpn-agent.service           /etc/systemd/system/
cp /opt/vpn-node/current/systemd/fvpn-watchdog.service        /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now fvpn-updater-recover.service fvpn-agent.service fvpn-watchdog.service
```

Once the node reports healthy observations (`provisioning_state: "ready"`, `kernel_Compatibility: true`, `isolation_verified: true`, `dns_enforcement_verified: true`) in `GET /api/v1/admin/nodes/{node_id}`, publish the exit node (`PUT /api/v1/admin/nodes/{node_id}/publication` with `publication_intent: "published"` and `If-Match: "<etag>"`). Wait for the returned `202 Accepted` operation (`node.publish`) to reach `state: "succeeded"` with read-back evidence from NetBird before anunciar the exit to members.
