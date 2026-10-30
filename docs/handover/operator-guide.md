# Operator Guide (`docs/handover/operator-guide.md`)

This guide defines the routine operational runbooks for Freedom VPN administrators (`§6`–`§15`, `§17`–`§18`).

All administrative browser sessions require OIDC authentication with MFA (`amr: ["mfa"]` or configured `acr`) via `/auth/login?role=admin` (`§9.1`, `§16.1`). High-risk mutations (`POST /api/v1/admin/nodes/{node_id}/revoke`, `DELETE /api/v1/admin/nodes/{node_id}`, `POST /api/v1/admin/update-batches`, `POST /api/v1/admin/policies/{definition_id}/activate`) additionally require recent authentication within `recent_auth_seconds` (`900s`, `403 RECENT_AUTH_REQUIRED` if stale) and explicit confirmation in the Admin Portal (`§9.1`, `§16.1`, `§18.1`).

---

## 1. Daily and Weekly Operational Checklists (`§17.1`, `§18.1`)

### 1.1 Daily Operator Tasks

1. **Review Admin Overview (`/admin`)**:
   - The Admin Portal overview aggregates `GET /api/v1/admin/nodes`, `GET /api/v1/admin/members`, `GET /api/v1/admin/policies`, `GET /api/v1/admin/releases`, and `GET /api/v1/admin/audit-events`. Check online vs. stale/offline exit nodes (`control_status`), active member count, policy and threat-feed status (`threat_feed_status`), and usage ledger mode (`normal`, `frozen`, or `reconciling` in `usage_ledger_state`).
2. **Inspect Failed or Stuck Asynchronous Operations (`GET /api/v1/admin/operations/{operation_id}`)**:
   - Check operations with `status = 'failed'` or `status = 'partially_succeeded'`:
     ```sql
     SELECT id, kind, status, resource_type, resource_id, error_code, updated_at
     FROM operations
     WHERE status IN ('pending', 'running', 'failed', 'partially_succeeded')
     ORDER BY updated_at DESC;
     ```
   - Check the Prometheus counter `fvpn_worker_dead_letters` on `127.0.0.1:9092/metrics` and inspect dead-letter outbox events (`00003_ledger_and_work.sql`, `00100_control.sql`, `docs/control/WORKER.md`):
     ```sql
     SELECT id, event_type, attempt, dead_lettered_at, last_error_code
     FROM outbox_events
     WHERE dead_lettered_at IS NOT NULL
     ORDER BY dead_lettered_at DESC, id;
     ```
3. **Check Exit Node Health & Update Batch Attention Flags (`/admin/nodes`, `/admin/updates`)**:
   - Investigate any node with `control_status != 'online'`, `safety_status != 'passed'`, `data_health = 'failed'`, any `observation.checks` entry (`internet_only`, `dns_policy`, `tunnel_probe`) with `status != 'passed'`, or any update target flagged `needs_attention = true` or `unconfirmed = true`.
4. **Check Audit Anomalies (`audit_anomalies_total`)**:
   - Verify that `fvpn-control` logs zero `WARN audit row omitted` events (`audit_anomalies_total = 0`).

### 1.2 Weekly Operator Tasks

1. **Verify PostgreSQL Backup & WAL Archive Freshness (`§17.2`)**:
   - Confirm that daily base backups and continuous WAL archiving (`archive_timeout = 300s`) succeeded and that the latest restorable timestamp is within `< 15 minutes`.
2. **Review Fair-Use Policy (FUP) Tier Distribution, Threat-Feed Refresh State & DNS Resolver Publication (`/admin/policies`)**:
   - Check `GET /api/v1/admin/policies?type=threat_feed` (`PolicyDefinition.threat_feed_status`, `D-054`, `D-058c`) and inspect `dns_publication_heads`, `dns_resolver_documents` (`00180_dns_publish.sql`), `dns_feed_refresh_state`, and `dns_feed_contents` (`00181_dns_feed_contents.sql`) to confirm all DNS profiles and threat feeds are healthy (`§12.2`, `§12.5`, `docs/dns/PUBLISH.md`, `docs/dns/FEEDS.md`):
     ```sql
     SELECT h.definition_id,
            d.profile_code,
            h.profile_revision_id,
            h.policy_generation,
            d.id AS document_id,
            d.document_sha256,
            h.status,
            h.updated_at
     FROM dns_publication_heads h
     JOIN dns_resolver_documents d
       ON d.definition_id = h.definition_id
      AND d.profile_revision_id = h.profile_revision_id
      AND d.policy_generation = h.policy_generation
     ORDER BY h.updated_at DESC;
     ```
     ```sql
     SELECT r.feed_revision_id,
            r.last_good_content,
            r.last_error_code,
            r.consecutive_failures,
            r.last_attempt_at,
            r.last_success_at,
            r.next_attempt_at,
            c.rule_count,
            c.size_bytes
     FROM dns_feed_refresh_state r
     LEFT JOIN dns_feed_contents c
       ON c.feed_revision_id = r.feed_revision_id
      AND c.content_sha256 = r.last_good_content
     ORDER BY r.last_attempt_at DESC NULLS LAST;
     ```
3. **Run Scheduled Retention Purge (`purge_retained`, `§15.4`, `§16.4`, `F-03`)**:
   - Connect as `fvpn_retention` and purge expired raw usage payloads (`> 90 days`, where `retention_config.raw_usage_days` enforces `>= 35` days and refuses to purge months with `gap = TRUE`), service logs (`> 30 days`), and audit events (`> 365 days`, where `retention_config.audit_days` enforces `>= 180` days) `[production only]`:
     ```sql
     SELECT purge_retained('usage_event_payloads', now() - interval '90 days', 'weekly scheduled retention purge');
     SELECT purge_retained('audit_events',         now() - interval '365 days', 'weekly scheduled retention purge');
     ```
4. **Check Certificate & Signing Key Expiry Windows (`§6.4`, `§16.2`)**:
   - Inspect `/var/lib/fvpn/vault/state.json` and `/etc/vpn-node/trust.json` to ensure no active signing key (`desired-state`, `lease`, `recovery`, `ledger-epoch`, `node-ca`, `release`) is within 14 days of `not_after`.

---

## 2. How to Read Operations, Audit Events, and OpenAPI 1.1.1 Contract Status (`§14.4`, `§14.5`, `§15.3`, `D-029`, `D-056`, `D-057`, `D-059`)

### 2.1 Asynchronous Operations (`GET /api/v1/admin/operations/{operation_id}`, `GET /api/v1/member/operations/{operation_id}`)

Side-effecting commands that touch external systems (NetBird, AdGuard Home, node desired-state rollout) never block HTTP transactions on external I/O (`AGENTS.md §2, §4`). Instead, they commit desired state in PostgreSQL, enqueue an `outbox_events` row, and return `202 Accepted` with an `Operation` resource.

- **Important Contract (`AGENTS.md §4`)**: Desired state and observed state are always separate. A `202 Accepted` response means the intent is recorded—**never** that the external effect has completed.
- **Operation Kinds (`OperationKind`, `contracts/openapi.yaml`)**:
  - Node & release operations: `node.enroll`, `node.update`, `node.publish`, `node.disable`, `node.unpublish`, `node.revoke`, `node.retire`, `release.register`, `update_batch.create`, `update_batch.retry`
  - Member & billing operations: `checkout.create`, `member.update`, `member.suspend`, `member.unsuspend`, `member.grant`, `member.grant_revoke`, `member.device_authorize`, `member.device_revoke`
  - Policy operations: `policy.activate`
- **Operation Statuses (`operations.status`)**:
  - `pending` -> `running` -> `succeeded` | `partially_succeeded` | `failed` | `superseded`
- **Database Guard (`F-01`, `00005_evidence_and_deadlines.sql`)**:
  An operation row in `operations` cannot transition to `status = 'succeeded'` unless its JSONB `steps` array and `observed_refs` array are non-empty and every step with `required = true` has `status = 'succeeded'` and a non-empty `observed` evidence object (read-back proof from NetBird, AdGuard Home, or node observation).

Inspecting an operation via API `[Verified on dev/test stack]`:
```bash
curl -fsS "https://vpn.example.com:8443/api/v1/admin/operations/${OPERATION_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}"
```

### 2.2 Audit Events (`GET /api/v1/admin/audit-events`)

Every administrative, member, agent, or system mutation appends an immutable row to `audit_events` in the same transaction (`§15.3`, `F-02`).

- **Canonical Formatting (`D-029`, `D-035`, `D-059`, `docs/control/OPERATIONS.md`)**:
  - `actor_role`: `admin` | `member` | `node` | `system`
  - `actor_id` (DB column `actor`): Canonical UUID (`audit.SystemActor.ID` is `00000000-0000-0000-0000-000000000003`; SQL `purge_retained()` uses `00000000-0000-0000-0000-000000000000` per `D-035`).
  - `resource`: Canonical `<resource_type>:<uuid>` where `<resource_type>` is one of the 15 contract `ResourceType` values (`node`, `exit`, `enrollment`, `member`, `member_device`, `member_group`, `entitlement_grant`, `checkout_session`, `policy_definition`, `policy_revision`, `release`, `update_batch`, `update_target`, `admin_identity`, `fleet_config`; system singleton `fleet_config:00000000-0000-0000-0000-000000000001`).
  - `result`: `succeeded` | `failed` | `denied`
  - `before` / `after`: Redacted JSON snapshots (secrets, tokens, private keys, and raw webhook payloads are never stored).

Querying audit events filtered by resource and action `[Verified on dev/test stack]`:
```bash
curl -fsS "https://vpn.example.com:8443/api/v1/admin/audit-events?resource=node:0196ed00-0000-7000-8000-000000000301&limit=20" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}"
```

### 2.3 OpenAPI 1.1.1 Contract Inventory (`44` Operations) & Pending Handler Wiring (`docs/contracts/README.md`, `docs/ROADMAP.md`)

`contracts/openapi.yaml` (`version: 1.1.1`, Revisions 5 and 5.1) defines **44 HTTP operations** across `/api/v1`:
- **41 operations** are wired in `apps/control` on `main`.
- **3 operations** added in Revisions 5 and 5.1 currently return `501 NOT_IMPLEMENTED` from the generated composite server on `main`, pending downstream handler wiring:
  1. `GET /api/v1/admin/groups` (`ListAdminGroups`, `D-057`): Lists member groups (`MemberGroupList`).
  2. `POST /api/v1/admin/groups` (`CreateAdminGroup`, `D-057`, `D-059`): Creates an immutable member group (`MemberGroupCreate` -> `201 MemberGroup`, audited under `member_group:<group_id>`).
  3. `GET /api/v1/agent/resolver-documents/{document_id}` (`GetResolverDocument`, `D-056`): Agent mTLS fetch of a canonical `resolver_document` (`200 ResolverDocument`, `Cache-Control: no-store`).

---

## 3. Enrolling, Classifying, Publishing, Revoking, and Retiring Nodes (`§6`, `§7`, `§14.5`, `D-055`–`D-058`)

### 3.1 Enrolling a New Node or Reinstalling an Existing Node (`POST /api/v1/admin/enrollments`)

- **New Node**: Omit `node_id`. Set `delivery: "installer"` (default 15-minute TTL / `900s`) or `"preinstalled"` (default 7-day TTL / `604800s`), plus optional `name`, `region`, `notes`, and `member_group_ids`.
- **Bootstrap & One-Off NetBird Correlation Group (`D-055`, `docs/nodes/WIRING.md`)**:
  When the enrolled node calls `POST /api/v1/agent/bootstrap` over mTLS, central creates a single-use NetBird setup key (`fvpn-node-<node_id>-<incarnation_id>`) attached to a dedicated one-off NetBird correlation group (`fvpn-enroll-peer-<16-hex-sha256>`). The worker's `node.enroll.correlate` handler finds the unique peer in that correlation group, verifies its hostname and `/32` overlay IPv4 (`100.64.0.0/10`), records `external_bindings` (`node_peer`), deletes the temporary correlation group, and sets `netbird_peer_id` on `nodes`.
- **Reinstalling an Existing Node (`§6.2`, `00111_pki_reinstall.sql`, `AT07`)**:
  1. First **revoke** the node's current incarnation (`POST /api/v1/admin/nodes/{node_id}/revoke`) so no active incarnation remains (`409 NODE_INCARNATION_ACTIVE` is returned if an incarnation is still active; `409 NODE_RETIRED` if the node was permanently retired).
  2. Call `POST /api/v1/admin/enrollments` with `"node_id": "<existing_node_uuid>"` and `"delivery": "installer"`.
  3. Run `fvpn-agent enroll -config /etc/vpn-node/agent.json -token-file /run/vpn-node/enrollment.token` on the node. A new `incarnation_id` and mTLS certificate are issued under the same `node_id`.

### 3.2 Updating Node Region, Member Groups, or Capacity (`PATCH /api/v1/admin/nodes/{node_id}`)

First fetch the node (`GET /api/v1/admin/nodes/{node_id}`) and capture the response `ETag` header (`"<resource_version>"`). Supply `If-Match` and `Idempotency-Key` (`428 Precondition Required` if `If-Match` is missing; `412 Precondition Failed` if stale):

`[Verified on dev/test stack]`
```bash
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "tw-tpe-01",
    "region": "tw-tpe",
    "member_group_ids": [
      "01950000-0000-7000-8000-000000000001"
    ],
    "capacity": {
      "down_bps": "900000000",
      "up_bps": "900000000",
      "member_down_bps_cap": "100000000",
      "member_up_bps_cap": "100000000"
    }
  }'
```
*(To clear an explicit capacity ceiling and revert to policy defaults, send `"clear_capacity": true` and omit `"capacity"`. Updating `member_group_ids` or `capacity` increments `desired_config_generation` and returns `202 Accepted` with a `node.update` operation; metadata-only changes to `name` or `notes` return `200 OK`.)*

### 3.3 Publishing, Disabling, and Unpublishing Exits (`PUT /api/v1/admin/nodes/{node_id}/publication`)

Exit publication controls whether member devices route traffic through the node (`§5.2`, `§7.3`, `D-057`, `docs/nodes/WIRING.md`):
- `"published"`: Creates/enables the NetBird exit route (`0.0.0.0/0`) and `member-to-exit` policy for the exit's attached `exit_groups` (`member_group_ids`), and sets `publication_intent: "published"` in the node's signed desired state. Requires the node to have a valid incarnation, `provisioning_state = "ready"`, an approved serving release (`409 MINIMUM_VERSION_VIOLATION` or `409 STATE_CONFLICT` otherwise), and non-empty `exit_groups` so `netbird_route_readback` can complete.
  - **Group attachment (`D-057`):** OpenAPI `1.1.1` adds optional `group_ids` on `PublicationRequest` (and `group_ids` on `NodeDetail` alongside `member_group_ids`) to atomically replace `exit_groups` under the same `If-Match`. On current `main`, attach `member_group_ids` via `PATCH /api/v1/admin/nodes/{node_id}` before calling `PUT /api/v1/admin/nodes/{node_id}/publication`.
- `"disabled"`: Temporarily removes the node from member exit routing (`node.disable`) while keeping the node enrolled and online for diagnostics or updates.
- `"unpublished"`: Withdraws the exit publication (`node.unpublish`).

`[Verified on dev/test stack]`
```bash
curl -fsS -X PUT "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}/publication" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "publication_intent": "published",
    "reason": "Exit readiness checks verified"
  }'
```
Always poll the returned `Operation` (`GET /api/v1/admin/operations/{operation_id}`) until `status == "succeeded"` and `publication_observed == "active"`.

### 3.4 Node Egress Probe & Sanitized Check Details (`D-058a`, `D-058b`, `node/agent/probe.go`)

Each node's observation report (`POST /api/v1/agent/observations`) includes three required checks (`internet_only`, `dns_policy`, `tunnel_probe`):
- **Signed `egress_probe` precedence (`D-058a`)**: When `DesiredState.egress_probe` is non-null on an in-force desired state (`1..4` TCP IP-literal targets, `timeout_ms` `100..10000` default `2000` ms), `fvpn-agent` dials every listed target and ignores the local `/etc/vpn-node/agent.json:egress_probe_target`. If a signed `egress_probe` is structurally invalid, `tunnel_probe` fails closed (`status: "failed"`, `data_health: "degraded"`) without falling back to the local target. When `DesiredState.egress_probe` is null or absent, the agent dials `egress_probe_target` if configured.
- **Sanitized `NodeCheck.detail` (`D-058b`)**: `NodeCheck.detail` is bounded to `1..500` single-line printable ASCII characters and sanitized on both agent and central ingest so URLs, tokens, secret keys, file paths, SQL, and stack traces never enter `node_observations` or the Admin Portal. For `tunnel_probe`, `detail` reports `"Egress probe used the signed targets."`, `"Egress probe used the local target."`, `"Egress probe used no extra target."`, or `"Signed egress probe was rejected."`.

### 3.5 Revoking (`POST .../revoke`) vs. Retiring (`DELETE /api/v1/admin/nodes/{node_id}`)

- **Revoke (`POST /api/v1/admin/nodes/{node_id}/revoke`)**:
  Immediately revokes the node's current mTLS client certificate (`node_certificates`), revokes its serving lease, unpublishes its NetBird exit route, and deletes its NetBird peer (`AT07`, `AT13`, `internal/nodes/httpwire/revocation_adapter.go`). The `node_id` remains in `revoked` state and can be reinstalled with a new incarnation (`§6.2`).
- **Retire (`DELETE /api/v1/admin/nodes/{node_id}`)**:
  Permanently decommissions the node (`lifecycle_state = 'retired'`). A retired node can never be re-enrolled (`409 NODE_RETIRED`), while historical usage events and audit logs remain intact (`§7.1`).

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}/revoke" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Host maintenance / key compromise precaution"}'
```

---

## 4. Running an Update Batch, Health Attestation, and Handling `needs_attention` & `unconfirmed` (`§13`, `D-053`, `docs/updates/central-lifecycle.md`, `docs/updates/node-updater.md`, `docs/updates/node-health-attestation.md`)

### 4.1 Registering a Release and Updating `release_support` (`§13.3`, `§13.4`)

1. Build and sign the release (`amd64` + `arm64`) using `fvpn-release` and upload artifacts to `https://releases.example.com/<release_id>/` (see [`deploy.md` §8](deploy.md)).
2. Register the signed manifest envelope via `POST /api/v1/admin/releases` (`ReleaseRegisterRequest`):
   `[Verified on dev/test stack]`
   ```bash
   curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/releases" \
     -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
     -H "Idempotency-Key: $(uuidgen)" \
     -H "Content-Type: application/json" \
     -d "{
       \"manifest_url\": \"https://releases.example.com/${RELEASE_ID}/manifest.envelope.json\",
       \"manifest\": $(cat dist/manifest.envelope.json)
     }"
   ```
3. Create and activate a `release_support` policy revision (`POST /api/v1/admin/policies` -> `POST /api/v1/admin/policies/{definition_id}/activate`) that includes `${RELEASE_ID}` in both `approved_release_ids` and `minimum_serving_version.serving_release_ids` (`approved_release_ids` must be a subset of `serving_release_ids`, and `revoked_release_ids` must be disjoint from both, `AT27`).

### 4.2 Creating an `all_nodes` Update Batch (`POST /api/v1/admin/update-batches`)

In v1, fleet updates use `scope: "all_nodes"` and `set_fleet_default: true` (`§13.4`, `§13.10.1`), guarded by `expected_generation` (the current `fleet_config.generation` decimal string; mismatch returns `409 STATE_CONFLICT`).

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/update-batches" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d "{
    \"release_id\": \"${RELEASE_ID}\",
    \"scope\": \"all_nodes\",
    \"set_fleet_default\": true,
    \"expected_generation\": \"${FLEET_GENERATION}\",
    \"reason\": \"Roll out release 2026.10.1 to all active nodes\"
  }"
```

### 4.3 Node Update Execution, `D-053` Health Attestation, and Rollback (`node/updater`, `docs/updates/node-updater.md`, `docs/updates/node-health-attestation.md`)

When `fvpn-updater` executes a signed `UpdateIntent`:
1. **Preflight & Verification (`AT23`–`AT25`, `AT27`)**:
   - Acquires `/var/lib/vpn-node/update.lock` (`UPDATE_BUSY` if held), checks `now < authorization_expires_at` (`AUTHORIZATION_EXPIRED` if expired), verifies `release_id` is in `serving_release_ids` and not in `revoked_release_ids` (`MINIMUM_VERSION_VIOLATION` / `RELEASE_REVOKED`), checks disk headroom (`free >= size_bytes + installed_size_bytes + max(required_free_bytes, 512 MiB)`), downloads the manifest and archive from an HTTPS origin matching `release_origins`, verifies Ed25519 signatures and SHA-256 digests (`HASH_MISMATCH` on tamper), and checks schema compatibility.
2. **Switch & `D-053` Challenge-Response Health Attestation**:
   - Atomically switches `/opt/vpn-node/current` to `releases/<target_release_id>`, writes `/var/lib/vpn-node/update-journal/health-challenge.json` with a random 64-hex `nonce`, `release_id`, `issued_at`, and `expires_at` (`HealthChallengeTTL = 45s`), and restarts `fvpn-agent`.
   - Polls `/var/lib/vpn-node/update-journal/health-attest.json` every `250ms` for up to `30s` (`HealthAttestWait = 30s`) across up to `8` check attempts (`CheckInterval = 2s`).
   - All five health checks (`agent_process`, `control_channel`, `nftables_baseline`, `netbird_client`, `dns_resolver_path`) must report `"passed"`, `release_id` and `nonce` must match the challenge, `netbird_version` must equal `compatibility.pinned_netbird_version` (`0.82.0`), and `policy_generation` must be a valid decimal string (`> 0` when published).
3. **Automatic Rollback & Fail-Closed Latch (`AT26`, `UPD03`–`UPD05`)**:
   - If health attestation fails, `fvpn-updater` rolls `/opt/vpn-node/current` back to `last_good_release_id` (up to `3` attempts, `RollbackInterval = 1s`) **only** if `last_good_release_id` is still in `serving_release_ids`, not in `revoked_release_ids`, and passes its own fresh `D-053` challenge-response health attestation (`status: "rolled_back"`, exit code `1`).
   - If no valid rollback target exists or rollback health attestation fails, `fvpn-updater` writes `/var/lib/vpn-node/update-journal/forwarding.blocked`, forces the fail-closed `inet fvpn` baseline (blocking all member forwarding while keeping control/SSH up), and records `status: "failed"`, `recovery_required: true`, `error_code: "ROLLBACK_FAILED"` (exit code `2`).

### 4.4 Monitoring Targets and Handling `needs_attention` & `unconfirmed` (`GET /api/v1/admin/update-batches/{batch_id}`)

`GET /api/v1/admin/update-batches/{batch_id}` embeds a cursor-paginated `targets` list (`?cursor=...&limit=...`; `targets_url` points to the same endpoint, `Contract Facts`). Each `UpdateTarget` includes two required boolean flags (`§13.10`):

| Flag / State | Meaning | Operator Action |
|---|---|---|
| **`unconfirmed: true`** | The node was dispatched an update intent (`queued` / `dispatched` / `running`), or its heartbeat shows the new release, **but** central has not yet received a signed/authenticated `POST /api/v1/agent/update-results` receipt confirming terminal completion and `policy_generation` (`D-043`). | 1. Check if the node is online (`GET /api/v1/admin/nodes/{node_id}`).<br>2. If the node completed the update while central was unreachable, `fvpn-agent` / `fvpn-updater flush` will automatically re-submit the journaled result from `/var/lib/vpn-node/update-journal` on its next poll.<br>3. On the node, you can trigger immediate result delivery with `/opt/vpn-node/current/fvpn-updater flush -config /etc/vpn-node/updater.json`. |
| **`needs_attention: true`** | Set when any of the following holds:<br>• Target state is `failed`, `rolled_back`, or `recovery_required`<br>• Target reported `recovery_required: true` in its `UpdateResult`<br>• Target has been in a non-terminal state longer than `nodes.json:stuck_seconds` (`900s`) without progress. | 1. Inspect `last_error_code`, `stage`, and `recovery_required` on the target.<br>2. **Transient failure** (`ARTIFACT_FETCH_FAILED`, `DISK_SPACE_LOW` after freeing disk, `AUTHORIZATION_EXPIRED`, `UPDATE_BUSY`): Retry via `POST /api/v1/admin/update-batches/{batch_id}/retry` (increments `attempt` and issues a fresh `authorization_expires_at`).<br>3. **`rolled_back`**: Node automatically rolled back to `last_good_release_id` and restored member forwarding if `last_good` is still in `serving_release_ids` (`AT26`). Inspect binary/health-check logs before retrying.<br>4. **`recovery_required: true`** (rollback also failed or current release was revoked with no valid rollback target): Node has fail-closed member forwarding (`forwarding.blocked` latch set, exit code `2`). Log into the host or reinstall the node via `POST /api/v1/admin/nodes/{node_id}/revoke` + `POST /api/v1/admin/enrollments` (`node_id` set). |

Retrying failed/rolled-back targets in a batch (`UpdateBatchRetryRequest`) `[Verified on dev/test stack]`:
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/update-batches/${BATCH_ID}/retry" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d "{
    \"target_ids\": [\"${TARGET_ID}\"],
    \"reason\": \"Disk space cleared on node; retry attempt\"
  }"
```

---

## 5. Threat Feeds, Resolver Document Publication, and `dns.feed.refresh_failed` Runbook (`§12`, `D-054`, `D-056`, `D-058c`, `D-058d`, `docs/dns/PUBLISH.md`, `docs/dns/FEEDS.md`)

### 5.1 How Threat Feeds and Resolver Documents Are Published and Delivered

1. **Threat-Feed Refresh (`dns.feed.refresh`, `00181_dns_feed_contents.sql`, `D-054`)**:
   - Activating a `threat_feed` policy revision enqueues `dns.feed.refresh` in `outbox_events`.
   - `fvpn-worker` fetches `source_url` over HTTPS (rejecting private, loopback, overlay, link-local, and documentation literals), verifies the exact SHA-256 against the revision's pinned `content_sha256`, normalizes domains via IDNA, enforces the shrink guard (`feed_max_removal_percent`, default `50%`), and writes approved entries to `dns_feed_contents` and status to `dns_feed_refresh_state`.
   - `GET /api/v1/admin/policies?type=threat_feed` exposes `PolicyDefinition.threat_feed_status` (`D-058c`) with `status` (`healthy`, `stale`, `failed`, or `bootstrap_pending`), `last_good_sha256`, `entry_count`, `next_refresh_at`, and `last_error_code`.
2. **Resolver Document Compilation (`dns.publish`, `00180_dns_publish.sql`, `D-056`)**:
   - When a `dns_profile`, `domain_rule_set`, or `threat_feed` updates, `fvpn-worker` compiles a deterministic canonical JSON `resolver_document` (`kind = "resolver_document"`, `schema_version = 1`), stores it in `dns_resolver_documents`, and advances `dns_publication_heads`.
   - `desiredstate.Compiler` embeds a signed `ResolverDocumentRef` (`document_id`, `profile_revision_id`, `generation`, `content_sha256`, `size_bytes`) in `DesiredState.dns.resolver_document`.
3. **Node Delivery & Verification (`node/agent/resolver.go`, `node/dns`, `D-056`)**:
   - `fvpn-agent` fetches `GET /api/v1/agent/resolver-documents/{document_id}` over mTLS (or reads its SQLite `resolver_cache`), verifies the exact byte length (`size_bytes`) and SHA-256 (`content_sha256`) before parsing, applies the policy to all configured AdGuard Home instances via `node/dns.Applier` (with automatic rollback on partial failure), and reports `ObservationReport.dns_policy_evidence` (`outcome`: `applied`, `failed`, `partial`, or `not_run`, plus `feeds[]` evidence).
   - *(Note: On current `main`, the central HTTP handler for `GET /api/v1/agent/resolver-documents/{document_id}` returns `501 NOT_IMPLEMENTED` pending control-plane handler wiring; see §2.3.)*

### 5.2 Troubleshooting `dns.feed.refresh_failed` (`docs/dns/FEEDS.md`)

When a threat-feed fetch fails (`fetch_failed`, `content_mismatch`, `parse_failed`, `conflict_detected`, `shrink_limit_exceeded`, or `expired`), `fvpn-worker` retains the previous approved snapshot in `dns_feed_contents`, records `last_refresh_status = 'failed'` in `dns_feed_refresh_state`, increments `fvpn_dns_feed_refresh_total{outcome="retained_last_good"}` (or `"failed_no_last_good"` on initial bootstrap), and writes an `audit_events` row with `action = 'dns.feed.refresh_failed'`.

Inspect failed threat feeds and recent audit events:
```sql
SELECT r.feed_revision_id,
       r.last_good_content,
       r.last_error_code,
       r.consecutive_failures,
       r.last_attempt_at,
       r.last_success_at,
       r.next_attempt_at,
       c.content_sha256 AS retained_sha256,
       c.rule_count AS retained_rules
FROM dns_feed_refresh_state r
LEFT JOIN dns_feed_contents c
  ON c.feed_revision_id = r.feed_revision_id
 AND c.content_sha256 = r.last_good_content
WHERE r.consecutive_failures > 0
ORDER BY r.last_attempt_at DESC NULLS LAST;
```
- **`content_mismatch`**: Upstream feed content changed without a new pinned revision. Inspect the new upstream feed, compute its SHA-256, and create + activate a new `threat_feed` policy revision with the updated `content_sha256`.
- **`shrink_limit_exceeded`**: Upstream feed dropped more than `feed_max_removal_percent` (default `50%`) of its entries. Verify whether the upstream source was truncated before creating a new revision.

---

## 6. Freezing, Recovering, and Adjusting the Usage Ledger (`§10.8`, `§10.9`, `§17.3`)

The `fvpn-usage-admin` CLI operates directly against PostgreSQL and the central signing vault (`/var/lib/fvpn/vault`) and records an append-only `audit_events` row for every mutation.

### 6.1 Freezing and Recovering the Usage Ledger (`freeze`, `begin-recovery`, `complete-recovery`)

Use this sequence whenever central PostgreSQL is restored from backup or when investigating a ledger anomaly (see [`restore-drill.md`](restore-drill.md) for full `AT30` details):

`[Verified on dev/test stack]`
```bash
# 1. Freeze ingestion (nodes receive retryable 503 DEPENDENCY_UNAVAILABLE and retain local SQLite events)
/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
  freeze \
  --actor "${ADMIN_ACTOR_UUID}" \
  --reason "Maintenance / ledger recovery"

# 2. Rotate ledger_epoch, recompute stream watermarks from committed rows, and enter reconciling mode
/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
  begin-recovery \
  --actor "${ADMIN_ACTOR_UUID}" \
  --reason "Reconcile streams after restore" \
  --restored-to "2026-10-29T18:02:00Z" \
  --affected-since "2026-10-01T00:00:00Z"

# 3. After nodes pull signed checkpoints and finish replay, return to normal mode
/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
  complete-recovery \
  --actor "${ADMIN_ACTOR_UUID}"
```

### 6.2 Inspecting Member Usage Breakdown & Recording Adjustments (`drill-down`, `adjust`)

Raw `usage_events` and `usage_samples` are immutable (`F-02`). If a member requires a usage credit or correction, record an explicit signed adjustment in `usage_adjustments` (`§10.9`):

`[Verified on dev/test stack]`
```bash
# Inspect member total, watermarks, per-node and per-device breakdown for a UTC month
/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
  drill-down \
  --member-id "${MEMBER_ID}" \
  --period "2026-10"

# Record a signed byte delta (negative values credit/reduce counted usage; floored at 0 on read)
/usr/local/bin/fvpn-usage-admin --config /etc/fvpn/control.json \
  adjust \
  --member-id "${MEMBER_ID}" \
  --period "2026-10" \
  --upload-bytes -1073741824 \
  --download-bytes -5368709120 \
  --actor "${ADMIN_ACTOR_UUID}" \
  --reason "Support ticket #1042: credit during node maintenance window"
```

---

## 7. Suspending Members, Managing Grants, and FUP Overrides (`§5.1`, `§9.3`, `§11`, `§14.5`)

### 7.1 Suspending and Reactivating a Member (`PATCH /api/v1/admin/members/{member_id}`)

Suspending a member (`"admin_suspended": true`) immediately revokes member forwarding across all exit nodes and withdraws access from all of the member's NetBird peers via an asynchronous `member.suspend` operation (`AT04`, `docs/members/README.md`). Payment webhooks never auto-unsuspend an admin-suspended member (`§9.3`).

`[Verified on dev/test stack]`
```bash
# Fetch current member ETag first via GET /api/v1/admin/members/${MEMBER_ID}
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "admin_suspended": true,
    "suspension_reason": "Terms of service violation / abuse report"
  }'
```
To reactivate, send `"admin_suspended": false` and `"suspension_reason": ""` with the new `If-Match` ETag (`member.unsuspend` operation).

### 7.2 Issuing Finite Entitlement Grants (`POST /api/v1/admin/members/{member_id}/grants`)

Manual entitlement grants (`GrantCreateRequest`) require a `source` (`admin_grant`, `compensation`, `trial`, or `migration`), a `reason`, and a finite `[valid_from, valid_until)` interval bounded by `members.json:grant_max_seconds` (`F-05` rejects unbounded grants):

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}/grants" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "admin_grant",
    "valid_from": "2026-10-29T00:00:00Z",
    "valid_until": "2026-11-05T00:00:00Z",
    "reason": "7-day courtesy access while billing issue is resolved"
  }'
```
*(Note: Grant creation returns `202 Accepted` with a `member.grant` operation on `entitlement_grant:<grant_id>`. There is no HTTP `DELETE` endpoint for grants in the 44-operation OpenAPI contract; grants expire automatically at `valid_until` via `members.reconcile`, or member access can be withdrawn immediately via `PATCH /api/v1/admin/members/{member_id}` with `"admin_suspended": true`.)*

---

## 8. Key and Secret Rotation Runbooks (`§6.4`, `§16.2`, `docs/pki/rotation.md`)

### 8.1 Rotating Central Ed25519 Signing Keys (`desired-state`, `lease`, `recovery`, `ledger-epoch`)

Central signing keys in `/var/lib/fvpn/vault` use a **two-step overlap rotation** (`signing.FileProvider.Rotate`) so nodes never fail signature verification during rollout:
1. **Generate & Overlap (`Rotate`)**:
   Call `vault.Rotate(ctx, purpose, validityDuration, overlapDuration)` (where `overlapDuration >= 2 * desired_state_seconds`). The new key becomes `active` for signing, while the previous key moves to `retiring` and remains in `vault.TrustBundle(ctx)` until its overlap window expires.
2. **Distribute Updated Trust Bundle to Nodes**:
   Export `vault.TrustBundle(ctx)` to `/etc/vpn-node/trust.json` on nodes (or include in the next node release).
3. **Emergency Compromise (`Revoke`)**:
   If a central signing key is compromised, call `vault.Revoke(ctx, keyID)`, immediately distribute the updated `/etc/vpn-node/trust.json` (which marks `"revoked": true` for that `key_id`; `pkg/envelope` rejects any envelope carrying a signature from a known revoked key even if another valid signature is present), and re-issue desired state and leases.

### 8.2 Rotating the Offline Release Signing Key (`fvpn-release` & `fvpn-release-trust-v1`)

1. On the offline build machine, generate the new release signing key (`fvpn-release keygen -key-id release-prod-2027-01 ...`).
2. Construct a `fvpn-release-trust-v1` rotation envelope containing both the retiring key and the new key, and sign it with **both** the retiring release key and the new release key (`docs/pki/rotation.md`).
3. Apply the signed rotation envelope to the central vault via `vault.UpdateReleaseTrust(ctx, rotationEnvelopeBytes, now)` and distribute the updated `/etc/vpn-node/release-trust.json` to nodes before signing new releases solely with `release-prod-2027-01`.

### 8.3 Rotating Stripe Webhook Secrets and Billing Vault Keys (`members.json`)

- **Stripe Webhook Secret**:
  1. Create a new webhook signing secret in Stripe alongside the existing one.
  2. Add the new secret file path to `members.json:webhook_secret_files` (keep both current and old files in the array) and reload `fvpn-control`.
  3. Retire the old secret in Stripe and remove its path from `webhook_secret_files`.
- **Billing PII Vault Key (`billing.Vault`)**:
  1. Generate a new 32-byte random key file (`0600`), add it to `members.json:vault_key_files` under a new key ID (e.g., `"vault-prod-2027-01"`), keeping `"vault-prod-2026-01"` in the map so existing ciphertexts decrypt cleanly.
  2. Set `"vault_active_key": "vault-prod-2027-01"` and restart `fvpn-control` and `fvpn-worker`.
