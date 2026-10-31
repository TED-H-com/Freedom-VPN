# Production Deployment Runbook (`docs/handover/deploy.md`)

**Current cut — [follow-up 18](follow-up-18.md) (2026-10-31 EDT):** Deploy migration 00192 before the new control binary. Down refuses once installer corrections exist. #320/#323 provide checkpoint correlation; the devstack 19 serving-node rerun is not yet proved.

Historical follow-up 17 cutoff: [follow-up 17 operator evidence](follow-up-17.md). #313
harness-proves the SECOPS-21 first-offer capture (2/0/0, delivery 4 > retained 3,
matching local digest and central 202); full restore was open for secops 13 at that cutoff.
Earlier restore measurements below retain their original cutoff.

This runbook guides operators through transitioning Freedom VPN from the local development and E2E lab stack (`deploy/devstack`) to production hosts (`§3`, `§5`–`§8`, `§10`–`§13`, `§15`–`§18`, `§21`).

Every command in this runbook is labeled either `[Verified on dev/test stack]` (executed and verified during the Wave 4 hand-over verification run) or `[production only]` (requires privileged host access, external DNS/TLS certificates, or live third-party provider accounts).

---

## 1. Architecture & Network Topology Overview (`§3`, `§16`, `§18.1`)

Freedom VPN v1 operates as a modular monolith in the central plane (`fvpn-control` + `fvpn-worker` sharing PostgreSQL 17 and a signing vault) paired with stateless/state-split Ubuntu 24.04 LTS exit nodes (`fvpn-agent`, `fvpn-updater`, `fvpn-watchdog`) and self-hosted NetBird CE (`0.82.0`) and AdGuard Home (`v0.107.59`).

| Component | Recommended Host / Placement | Exposed Endpoints / Ports | Documentation Example Address |
|---|---|---|---|
| **PostgreSQL 17** | Dedicated DB host or private subnet | `5432/tcp` (internal TLS only, never public) | `198.51.100.10:5432` (`db.internal.example.com`) |
| **`fvpn-control`** | Central API host (behind reverse proxy or direct TLS; see `trusted_proxies` note below) | `:8443/tcp` (Public HTTPS: Portal, OIDC, Webhooks, `/agent/enroll`)<br>`:9443/tcp` (Agent mTLS: `/api/v1/agent/*` except enroll)<br>`127.0.0.1:9091/tcp` (Prometheus `/metrics`) | `https://vpn.example.com` (`198.51.100.20`)<br>`https://agent.vpn.example.com:9443` |
| **`fvpn-worker`** | Co-located with `fvpn-control` or dedicated worker host sharing the signing vault | `127.0.0.1:9092/tcp` (Prometheus `/metrics` & `/healthz`, `/readyz`) | `198.51.100.20` / `198.51.100.21` |
| **OIDC Identity Provider** | External managed OIDC or self-hosted IdP | `443/tcp` (HTTPS) | `https://id.example.com` |
| **NetBird CE (`0.82.0`)** | Dedicated overlay coordination host | `443/tcp` (Management API `/api`, gRPC, Signal, Relay)<br>`3478/udp` (STUN/TURN) | `https://netbird.example.com` (`198.51.100.30`) |
| **AdGuard Home (`v0.107.59`)** | At least 2 resolver instances reachable over the overlay (`§12.2`) | `53/udp`, `53/tcp` (Overlay DNS for `100.64.0.0/10`)<br>`8080/tcp` (Admin HTTP API, restricted to `fvpn-worker` and `fvpn-agent`) | `100.64.0.53`, `100.64.0.54` (IPv6: `2001:db8::53`, `2001:db8::54`) |
| **Release Origin** | Immutable HTTPS object storage / static origin (`§13.3`, `§16.2`) | `443/tcp` (HTTPS read-only to nodes) | `https://releases.example.com` (`203.0.113.10`) |
| **Exit Nodes** | Ubuntu 24.04 LTS (`amd64` or `arm64`) | WireGuard UDP port (`51820/udp`), outbound HTTPS (`443/tcp`, `9443/tcp`) | Public egress `192.0.2.20`, Overlay `100.64.0.100` |

> **Reverse-proxy (`trusted_proxies`) and multi-replica limiter rules (`#141`, `#169`, `#174`, `#182`, `#190`, `#215`, `#228`, `D-073`, `internal/config/proxies.go`, `internal/httpapi/middleware.go`, `internal/enrollment/revoke.go`, `docs/control/README.md`, `docs/control/AUTH.md`, [`docs/e2e/scenarios/gateway-principals.md`](../e2e/scenarios/gateway-principals.md)):** Production either terminates TLS on `fvpn-control` and leaves `trusted_proxies` (`FVPN_TRUSTED_PROXIES`) empty (so `X-Forwarded-For` is ignored and pre-authentication buckets key on the TCP peer `r.RemoteAddr`), or sets `trusted_proxies` to the reverse proxy's source CIDRs; a proxy that connects from several addresses is several CIDRs (each source address as a `/32` or a covering prefix), each at least `/8` (IPv4) or `/32` (IPv6); `0.0.0.0/0`, `::/0`, any shorter prefix, and IPv4-mapped IPv6 CIDRs are rejected by `config.ParseTrustedProxies` because `"trusting the world lets any peer choose its bucket"`; list proxy addresses only (an address inside a trusted CIDR is treated as a hop and skipped during the rightmost `X-Forwarded-For` walk, while `X-Real-IP` and RFC 7239 `Forwarded` stay ignored); under `#190`'s parse rule for `r.RemoteAddr` and `X-Forwarded-For`, bracketed input is never split, an unbracketed multi-colon string is parsed whole before a trailing port is removed, and eight hextets written with a zero-width `::` (`2001:db8:1::2:3:4:5:6`) are `2001:db8:1:2:3:4:5:6`, `/64` `2001:db8:1:2::/64`; on the agent listener (`:9443`), `X-Forwarded-For` is never read and the agent-listener ingress key is the verified leaf certificate's `urn:fvpn:node:<uuid>` URI SAN (`node:<uuid>`, so every still-usable serial of a node shares one burst during renewal overlap, with `sha256(leaf.Raw)` as the certificate-fingerprint fallback only for a verified leaf without that SAN, `#215`; taken with the class concurrency slot before `AuthenticateAgent`, while no verified chain is `401` before the limiter and inserts nothing), so an address-rewriting proxy does not choose the bucket; and replicas do not share in-memory limiter state, so replicas multiply the per-process budgets. By contrast (`D-073`, `#228`, `db/migrations/00187_identity_roaming_and_readiness_grace.sql`), cloned-identity roaming and interleaving detection (`D-072`, `#211`, `ObserveEndpointTx` in `internal/enrollment/revoke.go`) is durable in PostgreSQL `node_identity_endpoints` (`candidate_hash`, `candidate_first_seen_at`, `candidate_last_seen_at`) under a row lock (`SELECT ... FOR UPDATE`), so two `fvpn-control` replicas or a process restart never lose the roaming candidate. In the devstack lab (`#182`, `docs/e2e/scenarios/gateway-principals.md`), `control.json` sets `trusted_proxies` to the lab gateway's exact `/32` (`172.28.30.21/32` for `e2e`) and nginx appends `$proxy_add_x_forwarded_for`, verified by `gateway-principals` (`fvpn-e2e-nodeconfig http-burst`, `SECOPS-13`).

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
| `tests/e2e/cmd/fvpn-e2e-prepare/main.go` | `-grace-seconds N` (`FVPN_E2E_GRACE_SECONDS`, `#182`, `SECOPS-04`) in `members.json` | `0..31622400` (default `0`, writes `grace_reason = "TEST ONLY finite grace override"` when `> 0`) | Lab-only prepare option (requires `E2E_RESET=1` or restarting both `fvpn-control` and `fvpn-worker`); set explicit business-approved `grace_seconds` and `grace_reason` in `/etc/fvpn/members.json` (`calibration.md`) |
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

**Historical follow-up 10 measurement — AT30 passed again (#265), run `0e8ff9a4-4fa1-4154-ae35-847a5be4d985` (`0e8ff9a4`), product `d9e28f8`, on the shared development host:** pre-replay database RPO **95.507504 s**, post-replay service usage RPO **0 s**, protocol RTO **93.532554563 s**, total outage RTO **106.868537734 s**. These are shared-host measurements, not production objectives. Historical #242 and #221 measurements remain separately dated below. SECOPS-16/17 closed against #240 (`past_due` admitted only while `paid` stays true); SECOPS-20 closed against #241 (D-079); SECOPS-18/19 revalidated, SECOPS-05 retained, D-077 proved through HTTP. SECOPS-21 stays open. AT29/AT31 were deferred to devstack follow-up 15 exports; those exports do not by themselves promote the full AT cases. Source: [AT30 Follow-up 10 archive](../e2e/at/secops/history/follow-up-10/AT30.md) and [exact JSON](../e2e/at/secops/history/follow-up-10/restore-reconcile.json). Security-ops follow-up 11 (#274) archived all 104 files byte-identical under D-098; its live AT29 / AT31 / AT30 runs were not executed. D-103 assigns the existing `fvpn-w4-secops-*` namespace to follow-up 12.

**Historical live AT30 (`#242`, run `42396951-2d9a-49f4-95c3-45f33d9519d5`, 2026-10-30 EDT): passed.** Isolated restore `7.17 s` with `6/6` checklist sections; `120.12 s` fail-safe negative without epoch rotation; retained replay `3/3` hashes with zero loss/duplicates/gaps; pre-replay DB RPO `90.301 s`, service RPO `0 s` / `0 B`, protocol RTO `112.35 s`, total outage RTO `121.88 s`. D-076 fenced the T0..T1 revoked device, an old idempotency replay kept the fence, fresh re-grant restored counted forwarding, and both the centrally revoked and newly enrolled negatives behaved correctly (SECOPS-18). During the hold, `node_observations.report_payload.usage` and `fvpn-agent usage snapshot` showed three unacked events / `2352` payload bytes, then zero after replay (SECOPS-19). The run predates the admin HTTP route and `NodeDetail.usage_outbox`; it exercised service-level re-authorization. See [restore drill §2.6](restore-drill.md#26-latest-measurement--at30-passed-265) and [`docs/e2e/at/secops/AT30.md`](../e2e/at/secops/AT30.md) for the evidence and the separate historical #221 values.

After `complete-recovery`, **review devices fenced as authorization-unconfirmed and re-authorize explicitly** where the member is entitled (`D-076`, `D-077`, `D-081`, `#227`, `#238`, `#244`, `#247`). List them with `GET /api/v1/admin/devices?authorization_unconfirmed=true`; `authorization_unconfirmed_since` is the fence time. Use `POST /api/v1/admin/members/{member_id}/devices/{device_id}/authorize` (admin session, CSRF, fresh `Idempotency-Key`, no body). It returns `202` `member.device_authorize`, including the existing operation for an underlying authorized device; confirm `status: authorized` and `authorization_unconfirmed: false` after worker sync. Errors include `409 STATE_CONFLICT` for revoked devices and `409 DEVICE_LIMIT_REACHED`. Entitlement restoration does not clear the fence; re-authorization or revocation does. Migration `db/migrations/00188_device_authorization_fences.sql` stores the fence; correlate `device.authorization.external_withdrawal`, the worker log `device authorization unconfirmed during ledger restore`, and `fvpn_device_restore_fence_decisions_total`. See [Phase 6.3](restore-drill.md).

Portal follow-up 16 (#255) supplies `/admin/devices` with authorization-unconfirmed, status and member filters, filter-bound cursors, a confirmed Re-authorize action, a neutral receipt for an existing operation, and a separate paginated read-back until the grant is visible. The node detail also has the usage-outbox block. Operators may use the portal page or the API above; a 202 receipt alone is not a completed grant.

**SECOPS-21 remains open. Symptom:** security-ops follow-up 10 (#265) saw first post-restore offer generation `11` while the node retained `14`, despite #245's floor over `nodes.desired_config_generation`, `nodes.applied_config_generation` and stored desired-state history. Older offers raise `ErrStaleGeneration`. **Rule (D-095, #270):** stop control/worker, restore T0, and freeze-ingest before starting either service on the restored database. The compiler holds the first post-restore compile until a fresh observation from the current incarnation arrives after `usage_ledger_state.mode` left `normal`. The observable hold is `pending fresh observation after restore` (retryable `503 DEPENDENCY_UNAVAILABLE`, node drift `desired_state_restore_observation`, operation step `fresh_observation`). **Verification:** after a fresh applied `14` report, require first new offer `15`, node acceptance without `ErrStaleGeneration`, then a separate applied ACK and converged `config_generation.desired` / `config_generation.applied`. Completing recovery alone must not release the hold. D-099 is fixed at the HTTP layer in wire-nodes follow-up 29 (#277). `CompileTx` always proposes `max(desired, applied, stored)+1`; cache reuse is allowed for the cached generation or its ordinary one-step proposal only when no higher applied/stored generation exists and no restore observation is outstanding. Policies follow-up 8 (#286) promotes `TestAT30_D095_HTTPRestoreCacheProbe`: the exact 503 hold precedes offer `15` above retained `14`; the pre-#277 negative control fails with `304 / generation 11`. The fixture ACKs are simulated. Do not close SECOPS-21 from the compiler tests alone. Live closure requires the first delivered offer, a matching node-local accepted envelope/ETag, successful application and a separate real applied ACK; absence of a stale-generation log is insufficient. See the [HTTP fix note](../nodes/2026-10-31-secops-21-handover.md) and [live closure checklist](../policies/2026-10-31-secops-21-handover.md).

### 3.2 Least-Privilege Database Roles (`docs/db/README.md`)

Never run `fvpn-control` or `fvpn-worker` as `postgres` superuser. Create the six role-separated database accounts (`§15.3`, `§17.3`):

| Role | Used By | Purpose |
|---|---|---|
| `fvpn_migrator` | `fvpn-migrate up` | Schema owner; creates tables, functions, and base triggers (`db/migrations/*.sql`, one goose version per file). |
| `fvpn_control` | `fvpn-control` | CRUD on operational tables; `INSERT` + `SELECT` on `audit_events` and `usage_events` (no `UPDATE`/`DELETE` on append-only tables, `F-02`–`F-04`). |
| `fvpn_worker` | `fvpn-worker` | Claims `outbox_events`, reconciles NetBird/Stripe/DNS (including threat-feed refreshes in `dns_feed_contents` and `dns_feed_refresh_state`), updates `usage_rollups`. |
| `fvpn_retention` | Retention job | Executes `SELECT purge_retained(...)` (`F-03`) to delete expired raw payloads and logs while recording an audit row. |
| `fvpn_recovery` | `fvpn-usage-admin` | Executes usage ledger freeze, epoch rotation (`usage_recovery_records`, `usage_ledger_epochs`), and adjustments. |
| `fvpn_backup` | `pg_dump` / `pgBackRest` | Read-only physical/logical backup role. |

Create the database and roles `[production only]`:

```sql
-- [mechanical-check: cluster-role-ddl]
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

For an existing deployment, first run the [identity duplicate pre-check and remediation procedure](operator-guide.md#73-identity-continuity-client-rotation-and-duplicate-pre-check-d-052). Historical duplicate issuer/subject groups block the uniqueness migration; there is no automatic merge.

Build and run `fvpn-migrate` using `FVPN_DATABASE_URL` (`fvpn-migrate` never logs or prints connection credentials):

`[Verified on dev/test stack]`
```bash
CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -o /usr/local/bin/fvpn-migrate ./internal/db/cmd/fvpn-migrate

FVPN_DATABASE_URL="postgres://fvpn_migrator:PASSWORD@198.51.100.10:5432/fvpn?sslmode=verify-full&sslrootcert=/etc/fvpn/certs/db-ca.pem" \
  /usr/local/bin/fvpn-migrate up

FVPN_DATABASE_URL="postgres://fvpn_migrator:PASSWORD@198.51.100.10:5432/fvpn?sslmode=verify-full&sslrootcert=/etc/fvpn/certs/db-ca.pem" \
  /usr/local/bin/fvpn-migrate status
```

Expected output (`fvpn-migrate status` must list every file in `db/migrations/*.sql` — one goose version per file, including `00006` which aligns `purge_retained()` audit rows with canonical `<resource_type>:<uuid>` format per `D-035` — as `applied`):

```text
current_database="fvpn"
00001 applied
...
<last file> applied
```

After migrations complete, grant table/sequence permissions to `fvpn_control`, `fvpn_worker`, `fvpn_retention`, `fvpn_recovery`, and `fvpn_backup` (`pg_read_all_data`) `[production only]`:

```sql
-- [mechanical-check: cluster-role-ddl]
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

### 4.1 OIDC Provider (`§9.1`, `§16.1`, `D-052`, `docs/control/AUTH.md`, `docs/members/README.md`, `docs/e2e/README.md`)

1. Configure a single OIDC issuer (`https://id.example.com`) shared by the Portal (`fvpn-control` / `fvpn-worker`) and NetBird (`§2 [S36]`, `§7.3`, `§9.1`, `§15.2`, `D-052`):
   - **Portal Client (`oidc_client_id`)**: Confidential client using Authorization Code + PKCE (`S256`).
     - Allowed Redirect URI: `https://vpn.example.com/auth/callback`
     - Subject identifier type: `public` (non-pairwise). The worker's `devices.discover` handler lists NetBird users (`GET /api/users`), excludes service and blocked accounts, and matches the observed NetBird user ID to `identity_bindings.subject` under `oidc_issuer`, writing `netbird_user_bindings` with `mapping_proof = "shared_issuer_public_subject:" + oidc_issuer` (`docs/members/README.md`). Pairwise or unknown subjects fail closed as `identity_unmapped` (`fvpn_netbird_identity_discovery_total{result="identity_unmapped"}`) and are excluded from `GET /api/v1/member/devices`; matching email addresses or display names is never used as identity evidence.
     - **Lab vs. production issuer note (`docs/e2e/README.md`):** In `deploy/devstack`, `netbird-server:0.82.0` uses its embedded Dex issuer (`https://netbird.example.invalid/oauth2`) and a lab token adapter (`tests/e2e/cmd/fvpn-e2e-idp`) that re-signs portal callback tokens with `TEST ONLY` `roles` and `amr: ["mfa"]`. Production deployments must configure an external OIDC provider shared by both portal and NetBird with real WebAuthn/TOTP MFA.
   - **Admin Role Claim & MFA (`§9.1`, `§16.1`)**:
     - Emit claim `roles` containing `"admin"` (or configure `admin_claim` and `admin_value` in `control.json`).
     - Enforce MFA at the IdP for administrators and emit `amr: ["mfa"]` (or a configured `mfa_acr` value). `fvpn-control` rejects `/auth/login?role=admin` with `403 ADMIN_MFA_REQUIRED` if neither `mfa_amr` nor `mfa_acr` matches, and requires re-authentication within `recent_auth_seconds` (`900s`) for high-risk mutations (`POST /api/v1/admin/nodes/{node_id}/revoke`, `DELETE /api/v1/admin/nodes/{node_id}`, `POST /api/v1/admin/update-batches`, `POST /api/v1/admin/policies/{definition_id}/activate`).

Keep `oidc_issuer` canonical and identical for control, worker, and NetBird identity discovery (`internal/members/issuer/issuer.go`, `docs/members/README.md`): lowercase `http`/`https` scheme and host, no default `:80`/`:443` port, no userinfo/query/fragment, and no **literal** trailing slash (`issuer.Canonical` trims only literal trailing `/` from the escaped path and preserves percent-encoded trailing slashes such as `%2F` or `%2f`, so `issuer.Validate` rejects non-canonical strings at startup and an issuer ending in `%2F` remains distinct from the unencoded path, `#139`). A portal client-ID rotation updates `identity_bindings.client_scope` in place and preserves the existing member identity (`#134`, `#137`); see [identity continuity and login role migration](operator-guide.md#73-identity-continuity-client-rotation-and-duplicate-pre-check-d-052-134-137-139-docscontrolauthmd-docsmembersreadmemd) and [control rate limiters](operator-guide.md#74-control-rate-limiters-and-dedicated-operation-classes-d-060-141-145-152-docscontrolreadmemd-docscontrolauthmd-internalconfigconfiggo).

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
// Ceremony steps executed via internal/signing and internal/pki (see tests/e2e/cmd/fvpn-e2e-prepare for reference):
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
  "trusted_proxies": [],
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

### 6.4 `/etc/fvpn/dns.json` (`dns_config_file`, `D-028`, `D-054`, `D-058d`)

```json
{
  "upstreams": [
    "198.51.100.53:53",
    "[2001:db8::153]:53"
  ],
  "bootstrap_dns": [
    "198.51.100.53:53"
  ],
  "feed_update_cadence_seconds": 21600,
  "feed_max_removal_percent": 50
}
```
> **Note (`docs/dns/FEEDS.md`, `internal/config/dns.go`):** `feed_update_cadence_seconds` (default `21600` seconds / 6 hours, bounds `300..604800`, `D-054`) sets the worker's fallback threat-feed refresh cadence when a `threat_feed` policy revision omits `update_interval_seconds` (`D-058d`). `feed_max_removal_percent` (default `50`, bounds `0..50`, `D-054`) sets the shrink-guard threshold above which a refreshed feed is rejected with `shrink_limit_exceeded` while retaining the last-good approved content in `dns_feed_contents`.

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

### 9.1 Node Host Prerequisites, Directory Layout, and Configuration (`AGENTS.md §4`, `node/enforce/apply.go`, `node/enforce/executor.go`)

On the Ubuntu 24.04 LTS exit node, install the user-space network executables invoked by `fvpn-agent`'s data-plane enforcement (`node/enforce/executor.go` `HostExecutor`, `node/enforce/apply.go`):
- **`nft`** (`nftables` package): atomic `nft -f -` transactions and whole-ruleset `nft -j list ruleset` verification (`NftBatch`, `NftListJSON`, `node/enforce/verify_foreign.go`, [`docs/poc/G2-netlab.md`](../poc/G2-netlab.md), `#220`, `#241`, `D-079`). Under `#220` (`r14` M-3) and **`D-079` (`#241`)**, `VerifyServing` reads the entire host nftables ruleset: any foreign base chain outside `inet fvpn` on `prerouting`, `forward`, or `postrouting` in families `ip`, `ip6`, `inet`, or `bridge`, or on `ingress` or `egress` of `MemberIf` or `UplinkIf` (including a `netdev`/`inet` `ingress`/`egress` chain with no device), that can mangle/rewrite payloads, set `meta mark` or `ct mark` (`MARK_SEMANTICS_VIOLATED`), `notrack`, NAT, `queue`, `drop`, `reject` (or has chain policy `drop` or `reject`), offload via `flow`/`xt` or attach a foreign `flowtable` to `MemberIf`/`UplinkIf` (`FLOWTABLE_BYPASS`), or `jump`/`goto` into such a chain (or has no priority, an unknown hook, or an unresolvable jump target — `ISOLATION_POLICY_INVALID`), fails serving closed with `ISOLATION_POLICY_INVALID`, `MARK_SEMANTICS_VIOLATED`, or `FLOWTABLE_BYPASS` (detail includes `handle <n>`, while tolerating accept-only/match/counter/log foreign chains, `input`/`output` chains, family `arp`, and `ingress`/`egress` chains on other devices). Under **`D-079` (`#241`, `loopback_scoped` rule in `node/enforce/verify_foreign.go`)**, a foreign rule in an `ip`, `ip6`, or `inet` base chain on `prerouting`, `forward`, or `postrouting` whose first expressions are native `ip`/`ip6` `saddr` or `daddr` matches with `"=="` (`""`) on a literal IP or prefix contained in `127.0.0.0/8` or `::1` and whose remaining expressions are only benign or NAT statements is classified as `loopback_scoped` (`VerifyResult.Loopback`, `table <t> family <f> hook <h> priority <p> chain <c> handle <n> statement <s>`) and tolerated without failing serving closed (`xt` matches and chain/table names prove nothing), whereas non-loopback NAT still fails closed as `FLOWTABLE_BYPASS` with `handle <n>`. Production exit nodes must not run Docker or any other NAT for non-loopback traffic; `loopback_scoped` in the read-back evidence is informational (`deploy/devstack` `#240` is an e2e test lab, not a production topology).
- **`tc` and `ip`** (`iproute2` package): `tc -batch -` HTB/fq_codel shaping (`TcBatch`, `TcShowQdisc`, `TcShowClasses`, `TcShowFilters`) and `ip -j -4 addr show` host address discovery (`IPAddrShow`).
- **`conntrack`** (`conntrack` package): `node/enforce/apply.go` `deleteConntrack` runs `conntrack -D -s <ip>` and `conntrack -D -d <ip>` (`ConntrackDeleteSourceV4`, `ConntrackDeleteDestV4`) for every revoked member address (`prev.AllowedV4` minus the new `Bindings`, keyed by `(addr, device_id, member_id)` in `RevokedAddress` / `revocationKey`), and both `tests/netlab/Dockerfile` and `deploy/devstack/Dockerfile.node` (`#153`, which also carries `tcpdump`) install `conntrack`. `SystemPlane.MissingExecutables()` (`node/agent/plane.go`, `#162`, `#166`) uses `enforce.Executor` directly to preflight `nft`, `tc`, and `ip` and calls `enforce.RequireConntrack(p.exec)` (`#156`, `#166`, `LookPath("conntrack")` -> `ISOLATION_POLICY_INVALID`, detail `"conntrack is not installed"`) at startup before installing the boot baseline: if any binary is missing, the agent keeps forwarding off with `safety_check_failed`, logs only the executable name, and reports `DEPENDENCY_UNAVAILABLE` at stage `preflight` plus failed `lease`, `qos`, and `source_binding` checks (`DEPENDENCY_UNAVAILABLE`). After `VerifyServing` and `VerifyQoS` succeed in `enforce.Apply` (`#156`, `#159`, `#172`, `#196`, `#197`, `#202`), `Apply` populates `verified.Counters` and `verified.Samples` (`Samples` skips unbound devices and zero metadata) before running the `m.Revoke` loop and attempts `deleteConntrack` for every address in `m.Revoke`. If any `conntrack -D` call fails, `enforce.Apply` leaves the verified serving ruleset and QoS installed (`ApplyResult.Verified = true`, because `ip saddr != @allowed_v4` and `ip daddr != @allowed_v4` precede `ct state established,related accept` and drop packets for removed bindings immediately) and returns a typed `*ConntrackDeleteError{Pending, Failed}` (`IsConntrackDeleteFailure`). In `fvpn-agent` (`#166`, `#193`, `#202`, `node/agent/revocation.go`, `node/agent/sync.go`, `node/internal/agentstore/store.go`, `docs/node/README.md`), when `plane.Apply` returns `res.Verified == true` with `enforce.IsConntrackDeleteFailure(err)` and consecutive failed attempts remain below 12 (`1..11`), the agent keeps the verified serving ruleset installed (`serving = true`), persists only the failed `(addr, device_id, member_id)` addresses (`Failed` plus any unattempted entries) under `pending_revocations` in local SQLite `meta` (merged with any newly revoked addresses on subsequent generations and retried on every apply tick until `conntrack -D` succeeds), keeps `internet_only`, `qos`, `source_binding`, and `meter` checks `passed` and the local watchdog heartbeat `serving`, and reports one `NodeError` (`code: ISOLATION_POLICY_INVALID`, `stage: "apply"`, `message: "conntrack delete pending; serving ruleset verified"` — one of the twelve revision-8 enforcement verification reasons in `contracts/openapi.yaml` `#164`). After **twelve consecutive failed `conntrack -D` attempts** (`maxConntrackDeleteAttempts = 12`, persisted across restarts in `#193` and `#202`) or if `conntrack` is missing after preflight, the agent fails isolation closed and installs the baseline (`member_forwarding_enabled=false`, `safety_status=failed`, `forwarding_block_reasons=["safety_check_failed"]`, `internet_only=failed` with `ISOLATION_POLICY_INVALID`). Any unverified apply error or non-conntrack failure also withdraws to the fail-closed baseline.

`[production only]`
```bash
apt-get update && apt-get install -y --no-install-recommends nftables iproute2 conntrack ca-certificates
```

Create the split directory hierarchy:

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
  "updater_journal": "/var/lib/vpn-node/update-journal/journal.db",
  "lock_path": "/var/lib/vpn-node/update.lock",
  "control_url": "https://agent.vpn.example.com:9443",
  "server_ca_file": "/etc/vpn-node/ca.pem",
  "trust_bundle": "/etc/vpn-node/trust.json",
  "datapath": {
    "member_ingress": "fvmb0",
    "uplink": "fvup0",
    "snat": "192.0.2.20"
  },
  "egress_probe_target": "192.0.2.10:443",
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
  "updater_config": "/etc/vpn-node/updater.json",
  "dns": {
    "resolvers": [
      { "id": "agh-primary", "admin_url": "http://127.0.0.1:8080", "dns_addr": "127.0.0.1:53" },
      { "id": "agh-secondary", "admin_url": "http://127.0.0.1:8081", "dns_addr": "127.0.0.2:53" }
    ],
    "resolver_credentials_file": "/etc/vpn-node/resolver-credentials.json"
  }
}
```
> **Important (`AGENTS.md §4`, `D-034`, `D-058a`, `D-058b`, `D-061`, `docs/node/README.md`, `node/agent/probe.go`)**:
> - `datapath.member_ingress` and `datapath.uplink` **must** start with `fv` (`^fv[a-z0-9_-]{1,13}$`) and must differ. `enforce.Render` unconditionally merges `datapath.snat/32` into `@host_v4` (`#172`, `node/enforce/render_nft.go`), drops non-53 member traffic to `@resolver_v4` via `ctbadns`, and unions `enforce.SpecDeniedV4()` with `ExtraDeniedV4` (`extra_denied_v4`, `node/enforce/model.go`, `docs/poc/G2-netlab.md`). When an exit node's uplink sits on a public on-link L2 subnet (outside RFC 1918 / `SpecDeniedV4()`), include that on-link prefix and upstream gateway IP in `extra_denied_v4` so `@denied_v4` blocks member packets addressed directly to adjacent uplink hosts or the gateway IP while allowing transit traffic routed through the gateway.
> - `host_reserve_bps` **must** be greater than zero (otherwise `fvpn-agent` exits with code `64` before touching the kernel).
> - `dns.resolvers[]` (`1..8` host-local AdGuard Home targets, each with IP-literal `admin_url` and `dns_addr`) and `dns.resolver_credentials_file` (`D-061`, `#150`, `docs/node/README.md`, `node/agent/dnsconfig.go`) configure node-side resolver convergence (`node/dns.Applier`). Never place credentials inline in `agent.json`: `resolver_credentials_file` must be an absolute path to a regular file owned by `0:0` (`uid 0`, `gid 0`) with mode `0600` and a single hard link (`Nlink == 1`) containing `{"username":"...","password":"..."}`. `fvpn-agent run` loads the credentials at startup and reloads them on `SIGHUP` without stopping; wrong file ownership/group, mode, or hard-link count fails `dns_policy` closed (`FORBIDDEN`, detail `"Resolver credentials are not usable."`) without logging the secret or URL.
> - `egress_probe_target` (`host:port`) is the optional local fallback egress target for the node's `tunnel_probe` check. When the signed `DesiredState.egress_probe` field is non-null on an in-force desired state (`D-058a`), its `1..4` TCP IP-literal targets (default `timeout_ms: 2000`, bounds `100..10000`) override `egress_probe_target`; if a signed `egress_probe` is structurally invalid, `tunnel_probe` fails closed without falling back to the local target. `NodeCheck.detail` (`<= 500` single-line ASCII characters, sanitized by `D-058b` so URLs, tokens, paths, and SQL/stack fragments are never emitted) reports which source ran (`"Egress probe used the signed targets."`, `"Egress probe used the local target."`, `"Egress probe used no extra target."`, or `"Signed egress probe was rejected."`).
> - `updater_journal` is the absolute updater database **file** path, default `/var/lib/vpn-node/update-journal/journal.db`; URLs and relative paths are rejected. Set it to the database actually used by the updater. It is independent of `journal_dir`, which still locates the forwarding latch and health sidecars. Unavailable snapshots yield `update.generation: null`, not completion; a recovery entered at the `rollback` stage without stored health checks reports `stage: "rollback"` with a blank failed health profile (`blankHealth`, never a passed one, `#192`) so central accepts the failure instead of rejecting it with `422`; see [reading update evidence](operator-guide.md#45-reading-a-rolled-back-update-target-evidence-and-result-rejections-d-062-136-140-147-149-192).
> - In `/etc/vpn-node/updater.json` (`#142`, `#154`, `#156`, `#168`, `#171`, `#192`, `docs/updates/node-updater.md`, `docs/updates/node-health-attestation.md`, `docs/node/README.md`, `node/updater/probe.go`, `node/updater/probe_addresses.go`, `node/updater/host.go`, `node/updater/retain.go`, `node/enforce/prefixes.go`), the optional `probe_peer` block (`peer_id`, one `/32` `source_v4`, `1..4` public `/32` `destinations` with `proto` `"tcp"` or `"udp"` and `port` `1..65535`, optional `resolver_dns`) and optional `agent_config` path (default `/etc/vpn-node/agent.json`, regular file `<= 1 MiB` verified via `os.Lstat`, `O_NOFOLLOW|O_NONBLOCK`, `fstat` regular check, and `os.SameFile` so symlinks, FIFOs, and replaced files are refused as `agent_config_unavailable`; resolver credentials are never opened) configure `D-063` synthetic-probe health challenges (`45s` default challenge window). `probe_peer` is validated at `LoadConfig`, `Config.Engine()`, and every challenge issuance (`Engine.challengeProbe`) against: (1) `enforce.SpecDeniedV4()` (`#156`, every IANA IPv4 Special-Purpose Address Registry block with `Globally Reachable != True` plus multicast `224.0.0.0/4`, denying `192.88.99.0/24` while allowing globally reachable `192.31.196.0/24`, `192.52.193.0/24`, and `192.175.48.0/24`); (2) the host's live IPv4 global-unicast interface addresses (`net.InterfaceAddrs`, refusing an empty filtered global-unicast IPv4 list with `node_addresses_unavailable` in `#168`, and refusing any matching host IPv4 address as `source_v4` or destination while allowing other private/CGNAT `/32` overlay addresses as `source_v4`); and (3) the resolved/literal addresses of `control_url` (from `agent_config`) and updater `result_url` (when set), plus `dns.resolvers[].dns_addr` and `dns.resolvers[].admin_url` hosts (refused as destinations regardless of `resolver_dns` or port), sharing a `2s` DNS lookup deadline. Config load and `Engine()` fail closed with `updater: invalid probe_peer` (and malformed JSON fails with `updater: invalid config JSON`). If an input becomes unavailable or forbidden at challenge issuance (`#168`, `#171`, `#192`), `Engine.issueChallenge` logs `"health challenge probe omitted"` once per fixed `probeRefusal` class per `Engine`, writes `probe_omitted {peer_id, reason}` into `health-challenge.json` (mutually exclusive with `probe`, included in `Same`), commits `probe_omission/<txn_id>/<nonce>` in SQLite `meta` and `txn_events.detail` (`Journal.ProbeOmissions`) before challenge publication (`probe_omission/<txn_id>/<nonce>` rows in `meta` are pruned with their transaction via `cleanTxnNotes` on seal and `Retain` for terminal `txns` while the `txn_events` `probe_omitted` event stays, `#192`), and includes `probe_omitted` in local `health_attest` diagnostics; on the agent (`#171`), `ObservationReport.checks[name=tunnel_probe].detail` reports `"probe_omitted: <reason>"` (taking precedence over the `ProbeSource` sentence, without peer ID or address, retained across sidecar removal/consumption until a later probe/omission challenge) while `tunnel_probe` stays `not_run` and `data_health` stays `unknown` with forwarding closed, and `health-attest.json` is answered with the accepted reason token `"probe refused"`. **One-time compatibility boundary (`#168`):** because `healthattest.ReadChallenge` uses `DisallowUnknownFields`, agent binaries built before `#168` reject an omission-bearing challenge and fail closed; release notes and rollout plans must carry this boundary when updating across `#168`. Omitting `probe_peer` keeps probe-less challenges without reading any of these inputs.
> - Use the final AdGuard admin origin in `dns.resolvers[].admin_url`: redirects, including HTTP→HTTPS, fail apply. See the [resolver troubleshooting guidance](operator-guide.md#51-how-threat-feeds-and-resolver-documents-are-published-and-delivered).
> - Never set `test_mode: true` on production nodes.

### 9.2 Issuing a One-Time Enrollment Token (`POST /api/v1/admin/enrollments`)

In the Admin Portal (`/admin/nodes`) or via API, issue an `installer` token (default TTL 15 minutes / `900s`):

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/enrollments" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "delivery": "installer",
    "name": "tw-tpe-01",
    "region": "tw-tpe",
    "notes": "First production exit node tw-tpe-01"
  }'
```

The response carries `Cache-Control: no-store` and returns the one-time `token` (`EnrollmentCreated.token`, `§6.1`) on initial `201` creation alongside `token_sha256` (`Enrollment.token_sha256`, `#191`, `#195`, `#200`, the 64-hex SHA-256 digest of the stored token hash `node_enrollments.token_hash`, also returned on idempotent replay, `getEnrollment`, and `revokeEnrollment` and recorded unredacted in `enrollment.created` `changes.after.token_sha256`).

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

What happens during `fvpn-agent enroll` (`§6.1`–`§6.3`, `ENR01`–`ENR04`, `D-055`, `docs/nodes/WIRING.md`):
1. Generates an ECDSA P-256 private key inside `/var/lib/vpn-node/identity` (`0600`, never leaves the node) and submits a PKCS#10 CSR + `fvpn-enrollment-v1` proof to `POST /api/v1/agent/enroll`.
2. Receives `node_id`, `incarnation_id`, and the signed mTLS client certificate.
3. Zero-overwrites, syncs, and unlinks `/run/vpn-node/enrollment.token` (`agent.NeutralizeToken`).
4. Calls `POST /api/v1/agent/bootstrap` over mTLS (`:9443`) to fetch the one-time NetBird `setup_key` (subsequent calls return `410 BOOTSTRAP_CONSUMED`). Central (`node.provision.requested`) creates a single-use setup key whose `auto_groups` contain the shared staging group (`fvpn-unpublished-nodes`) and a dedicated empty per-enrollment correlation group (`fvpn-enroll-peer-<hash>`, `D-055`, `docs/nodes/WIRING.md`). During `node.peer.discover`, `fvpn-worker` (`DiscoverNodePeer`) correlates the unique non-ephemeral, userless peer in that correlation group without hostname, address, or timing inference, records `node:<node>:<incarnation>` in `external_bindings`, detaches and deletes the temporary correlation group with read-back, and commits `peer_readback`.

### 9.4 Node Host Installer (`fvpn-node-install`, `#239`), Installing systemd Units, and Publishing the Exit (`§5.1`, `§6.1`–`§6.5`, `§7.3`, `§13.5`, `§16.3`, `ENR01`–`ENR04`, `AT06`, `AT07`, `D-057`, `docs/node/install.md`)

For host provisioning and lifecycle commands, follow [`docs/node/install.md`](../node/install.md) (`#239`) and [operator guide §3.1](operator-guide.md#31-enrolling-a-new-node-or-reinstalling-an-existing-node-post-apiv1adminenrollments). That runbook covers verified install, `already_current` re-runs, `check`, staged `upgrade`, `remove`, `reenroll --token-file`, and `image-prepare` / `firstboot`. In particular, `/var/lib/vpn-node/update-staging/pending-release.json` is staging evidence, not confirmation that `current` moved; D-080 pending-file consumption is merged (#253), seed is merged (#269), and central installer-result acceptance is merged in fleet-update follow-up 10 (#273, D-089 / D-097), with keyed HTTP replay in #277. Installer mutations use journal-first rollback under `/var/lib/vpn-node/install/` and preserve `usage.db` on re-enrollment. AT07 still needs the assembled and hardware proofs listed in the operator guide.

If installing the core runtime units manually from `node/agent/systemd/` (`fvpn-updater-recover.service`, `fvpn-agent.service`, `fvpn-watchdog.service`) or `deploy/node/systemd/`, enable them `[production only]`:

```bash
cp /opt/vpn-node/current/systemd/fvpn-updater-recover.service /etc/systemd/system/
cp /opt/vpn-node/current/systemd/fvpn-agent.service           /etc/systemd/system/
cp /opt/vpn-node/current/systemd/fvpn-watchdog.service        /etc/systemd/system/
systemctl daemon-reload
systemctl enable --now fvpn-updater-recover.service fvpn-agent.service fvpn-watchdog.service
```

Create or verify the target member group via `POST /api/v1/admin/groups` (`CreateAdminGroup`, `201 MemberGroup`, audited as `group.create` on `member_group:<id>`, `D-057`, `D-059`) or `GET /api/v1/admin/groups` (`ListAdminGroups`). Once the node reports healthy observations (`provisioning_state: "ready"`, `control_status: "online"`, `safety_status: "passed"`, `data_health: "healthy"`, and `observation.checks` entries `internet_only`, `dns_policy`, and `tunnel_probe` with `status: "passed"`) in `GET /api/v1/admin/nodes/{node_id}`, publish the exit node via `PUT /api/v1/admin/nodes/{node_id}/publication` (`SetNodePublication` -> `exits.SetPublicationTx`, `docs/nodes/WIRING.md`) supplying `publication_intent: "published"`, `group_ids: ["<group_uuid>"]`, `If-Match: "<etag>"`, and `Idempotency-Key` (or pre-attach `member_group_ids` via `PATCH /api/v1/admin/nodes/{node_id}` and omit `group_ids` to preserve existing membership). Wait for the returned `202 Accepted` operation (`node.publish`) to reach `status: "succeeded"` with `publication_observed: "active"` and NetBird route read-back evidence (`netbird_route_readback`) before announcing the exit to members.

Installer ENR-06 (#266) historically failed first start because `/var/run/netbird.sock` was outside the packaged write allowlist and consumers could remove shared runtime state. Installer follow-up 3 (#279) fixes this: the daemon owns `/run/fvpn-netbird/netbird.sock` and shared `/run/vpn-node` survives consumer restarts. Offline seed and firstboot now pass with real daemons, but the required full journaled upgrade stays **red** at the absent meter checkpoint (`final_sample`, `CHECKPOINT_FAILED`, health null). Do not infer success from updater exit 0. See [installer follow-up 3](../node/install-followup3-2026-10-31.md). Devstack follow-up 15 remains historically `21 pass / 1 fail`; enrollment follow-up 6 (#288) now verifies ENR-06 startup on the fixed installer (AT06 376/0/0, AT07 26/5/0). Five AT07 rows remain blocked on installer-node exports and ENR-05; startup success does not close the full upgrade test. Updates follow-up 9 passed regular AT23–AT27 on `5575470`, while its separate installer extension remains blocked on measured F9-01…F9-05; [current dispositions](../ops/README.md#follow-up-15-acceptance-snapshot-2026-10-31-edt).

### 9.5 Configuring the Fleet Offline Lease Ceiling (`GET`/`PUT /api/v1/admin/fleet-config`, `D-065`, `#176`, `#179`, `§14.5`, `§16.3`, `AT29`, `docs/nodes/README.md`, `docs/nodes/WIRING.md`, `internal/nodes/httpwire/fleet_config.go`)

- **What `max_offline_lease_seconds` governs (`§16.3`, [`docs/nodes/README.md`](../nodes/README.md), [`docs/nodes/WIRING.md`](../nodes/WIRING.md)):**
  The singleton `fleet_config.max_offline_lease_seconds` governs the longest a node keeps serving member traffic on its last signed `ServingLeaseEnvelope` when `fvpn-control` is unreachable ($\text{node\_lease\_end} = \text{signed\_server\_time} + \text{max\_offline\_lease\_seconds}$ and $\text{member\_lease\_end} = \min(\text{signed\_server\_time} + \text{max\_offline\_lease\_seconds}, \text{effective\_service\_end}, \text{node\_lease\_end})$). Operators and acceptance harnesses (`AT29`) manage this ceiling through `GET`/`PUT /api/v1/admin/fleet-config` rather than editing `fleet_config` in SQL (`D-065`).
- **Range `60`–`86400` and why the floor is `60` (`D-065`, `#176`, `#179`, `node/agent/sync.go`):**
  `max_offline_lease_seconds` accepts integer values in **`60`–`86400`** seconds. The read-only field `default_max_offline_lease_seconds` is `86400` (`24` hours — the `§16.3` ceiling and migration seed, not a deployment recommendation; see [`calibration.md` §1.4](calibration.md)). The **`60` s floor** exists for test labs (`AT29` deadline faults) because `fvpn-agent` renews its serving lease at `min(120 s, window/2)` remaining (`renewLead` in `node/agent/sync.go`) with a `10 s` heartbeat tick; the Admin Portal warns below `900 s` (`15` minutes).
- **`GET` → strong `ETag` → `PUT` flow, HTTP outcomes (`428`/`412`/`409`/`422`), `fleet_config.update` audit, and lease timing (`internal/nodes/httpwire/fleet_config.go`, `TestD065_FleetConfigHTTPReadWrite`, `TestD065_AT29_FleetConfigOnlyChangesNewLeases`, `TestD065_FleetConfigReleaseWriterInterleaving`, `TestD065_FleetConfigMissingRowUsesIssuerDefault`):**
  1. `GET /api/v1/admin/fleet-config` returns `200 OK` `FleetConfig` and strong `ETag: "<resource_version>"` (if the singleton row is absent, `GET` returns `max_offline_lease_seconds: 86400`, `default_max_offline_lease_seconds: 86400`, `resource_version: "0"`, `updated_at: "1970-01-01T00:00:00Z"`, `ETag: "0"` without writing).
  2. `PUT /api/v1/admin/fleet-config` requires `If-Match: "<resource_version>"`, a UUID `Idempotency-Key`, `X-CSRF-Token`, `Origin`, and `{"max_offline_lease_seconds": <60..86400>, "reason": "<1..500 chars>"}`. It locks `fleet_config` (`SELECT ... FOR UPDATE`, serializing with `fleetupdate.CreateBatch` without modifying `generation` or release-support columns), increments `resource_version`, appends one `fleet_config.update` audit event on `fleet_config:00000000-0000-0000-0000-000000000001` (`operation_id: null`) containing both `before` (`{"max_offline_lease_seconds"}`) and `after` (`{"max_offline_lease_seconds", "resource_version", "updated_at", "reason"}`) snapshots (even on an accepted same-value write), and returns synchronous **`200 OK`** `FleetConfig` with the new strong `ETag` (creating zero `operations` or `outbox_events` rows).
  3. Missing `If-Match` returns `428 PRECONDITION_REQUIRED`; stale `If-Match` returns `412 RESOURCE_VERSION_CONFLICT` with `details.current_resource_version`; reusing an `Idempotency-Key` with a different payload returns `409 IDEMPOTENCY_CONFLICT` (whereas replaying the same key and payload returns the original `200 OK` body and `ETag` reconstructed from the immutable audit row keyed by `idempotency_requests.id`, so keep `audit_days` retention $\ge$ `idempotency_seconds`); and out-of-range values (`< 60` or `> 86400`), weak `If-Match`, bad `reason`, or unknown fields return `422 VALIDATION_FAILED`.
  4. Existing signed leases (`node_leases`) and their replays keep their signed `max_offline_seconds`, `expires_at`, and Ed25519 signature; the next lease renewal (`POST /api/v1/agent/leases/renew`) copies the updated `max_offline_lease_seconds` (`TestD065_AT29_FleetConfigOnlyChangesNewLeases`; see [`operator-guide.md` §3.6](operator-guide.md#36-configuring-the-fleet-offline-lease-ceiling-getput-apiv1adminfleet-config-d-065-176-179-145-163-at29-docsnodesreadmemd-docsnodeswiringmd-internalnodeshttpwirefleet_configgo) for full curl examples).

### Follow-up 16 restore evidence (2026-10-31 EDT)

Security-ops follow-up 12 (#301), run `dd845a7f-2660-431c-a8e6-c513108cc07d`,
passes AT30 on the shared development host: pre-replay database RPO **106.45 s**,
post-replay service usage RPO **0 s**, protocol RTO **66.90 s**, total outage RTO
**74.53 s**, replay **5 events / 5 hashes / 0 lost / 0 duplicate / 0 gaps**.
D-095's exact fresh-observation hold is proved. SECOPS-21 remains open: the
first probe/stored generation 16 exceeds retained 15, but first delivery,
matching local acceptance and the applied ACK remain unproven; capture is
pending devstack follow-up 18 / wire-nodes follow-up 32. Earlier measurements
remain historical, and no candidate-hardware guarantee follows. See the
[current acceptance snapshot](../ops/README.md#follow-up-16-acceptance-snapshot-2026-10-31-edt).
