# Operator Guide (`docs/handover/operator-guide.md`)

This guide defines the routine operational runbooks for Freedom VPN administrators (`§6`–`§15`, `§17`–`§18`).

All administrative browser sessions require OIDC authentication with MFA (`amr: ["mfa"]` or configured `acr`) via `/auth/login?role=admin` (`§15.1`). High-risk mutations (`POST /admin/nodes/{id}/revoke`, `DELETE /admin/nodes/{id}`, `POST /admin/update-batches`, `POST /admin/policies/{kind}/revisions/{rev}/activate`) additionally require recent authentication within `recent_auth_seconds` (`900s`, `403 RECENT_AUTH_REQUIRED` if stale) and explicit confirmation in the Admin Portal (`§15.1`, `§18.1`).

---

## 1. Daily and Weekly Operational Checklists (`§17.1`, `§18.1`)

### 1.1 Daily Operator Tasks

1. **Review Admin Overview (`/admin`, `GET /api/v1/admin/overview`)**:
   - Check online vs. stale/offline exit nodes, active member count, pending/failed operations, and usage ledger mode (`normal`, `frozen`, or `reconciling`).
2. **Inspect Failed or Stuck Asynchronous Operations (`/admin/operations`)**:
   - Filter by `state=failed` or `state=partially_succeeded`.
   - Check the Prometheus counter `fvpn_worker_dead_letters` on `127.0.0.1:9092/metrics` and inspect dead-letter outbox jobs (`docs/control/WORKER.md`):
     ```sql
     SELECT id, topic, key, attempts, last_error_code, updated_at
     FROM outbox_jobs
     WHERE state = 'dead_letter'
     ORDER BY updated_at DESC;
     ```
3. **Check Exit Node Health & Update Batch Attention Flags (`/admin/nodes`, `/admin/updates`)**:
   - Investigate any node with `liveness_state != 'online'`, `isolation_verified = false`, `dns_enforcement_verified = false`, or any update target flagged `needs_attention = true` or `unconfirmed = true`.
4. **Check Audit Anomalies (`audit_anomalies_total`)**:
   - Verify that `fvpn-control` logs zero `WARN audit row omitted` events (`audit_anomalies_total = 0`).

### 1.2 Weekly Operator Tasks

1. **Verify PostgreSQL Backup & WAL Archive Freshness (`§17.2`)**:
   - Confirm that daily base backups and continuous WAL archiving (`archive_timeout = 300s`) succeeded and that the latest restorable timestamp is within `< 15 minutes`.
2. **Review Fair-Use Policy (FUP) Tier Distribution & DNS Publication Status (`/admin/policies`, `/admin/dns`)**:
   - Check `GET /api/v1/admin/dns/rules` and `dns_resolver_states` to confirm all AdGuard Home resolvers applied the latest generation (`§12.2`).
3. **Run Scheduled Retention Purge (`purge_retained`, `§15.4`, `F-03`)**:
   - Connect as `fvpn_retention` and purge expired raw usage payloads (`> 90 days`, excluding any month with `gap = TRUE`), service logs (`> 30 days`), and audit events (`> 365 days`) `[production only]`:
     ```sql
     SELECT purge_retained('usage_event_payloads', now() - interval '90 days', 'weekly scheduled retention purge');
     SELECT purge_retained('audit_events',         now() - interval '365 days', 'weekly scheduled retention purge');
     ```
4. **Check Certificate & Signing Key Expiry Windows (`§6.4`, `§16.2`)**:
   - Inspect `/var/lib/fvpn/vault/state.json` and `/etc/vpn-node/trust.json` to ensure no active signing key (`desired-state`, `lease`, `recovery`, `ledger-epoch`, `node-ca`, `release`) is within 14 days of `not_after`.

---

## 2. How to Read Operations and Audit Events (`§14.4`, `§14.5`, `§15.3`, `D-029`)

### 2.1 Asynchronous Operations (`GET /api/v1/admin/operations`, `GET /api/v1/admin/operations/{operation_id}`)

Side-effecting commands that touch external systems (NetBird, AdGuard Home, node desired-state rollout) never block HTTP transactions on external I/O (`AGENTS.md §2, §4`). Instead, they commit desired state in PostgreSQL, enqueue an `outbox_jobs` entry, and return `202 Accepted` with an `Operation` resource.

- **Important Contract (`AGENTS.md §4`)**: Desired state and observed state are always separate. A `202 Accepted` response means the intent is recorded—**never** that the external effect has completed.
- **Operation Kinds (`OperationKind`)**:
  - `node.enroll`, `node.reconcile`, `node.publish`, `node.disable`, `node.unpublish`, `node.revoke`, `node.retire`
  - `member.suspend`, `member.reactivate`, `member.device_authorize`, `member.device_revoke`, `member.grant_revoke`
  - `policy.activate`, `dns.publish`, `update.batch_create`, `update.batch_retry`, `billing.reconcile`
- **Operation States**:
  - `pending` -> `running` -> `succeeded` | `partially_succeeded` | `failed` | `superseded`
- **Database Guard (`F-01`, `00005_evidence_and_deadlines.sql`)**:
  An operation cannot transition to `succeeded` until every `required = true` step in `operation_steps` has `state = 'succeeded'` and a non-empty `observed` JSON evidence object (read-back proof from NetBird/AdGuard/node).

Inspecting an operation via API `[Verified on dev/test stack]`:
```bash
curl -fsS "https://vpn.example.com:8443/api/v1/admin/operations/${OPERATION_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}"
```

### 2.2 Audit Events (`GET /api/v1/admin/audit-events`)

Every administrative, member, agent, or system mutation appends an immutable row to `audit_events` in the same transaction (`§15.3`, `F-02`).

- **Canonical Formatting (`D-029`, `D-035`)**:
  - `actor_type`: `admin` | `member` | `node` | `system`
  - `actor_id`: Canonical UUID (system actions use `00000000-0000-0000-0000-000000000003` or `00000000-0000-0000-0000-000000000000`).
  - `resource`: Canonical `<resource_type>:<uuid>` (e.g., `node:0196ed00-0000-7000-8000-000000000301`, `fleet_config:00000000-0000-0000-0000-000000000001`, `retention_config:00000000-0000-0000-0000-000000000002`).
  - `result`: `succeeded` | `failed` | `denied`
  - `before` / `after`: Redacted JSON snapshots (secrets, tokens, private keys, and raw webhook payloads are never stored).

Querying audit events filtered by resource and action `[Verified on dev/test stack]`:
```bash
curl -fsS "https://vpn.example.com:8443/api/v1/admin/audit-events?resource=node:0196ed00-0000-7000-8000-000000000301&limit=20" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}"
```

---

## 3. Enrolling, Classifying, Publishing, Revoking, and Retiring Nodes (`§6`, `§7`, `§14.5`)

### 3.1 Enrolling a New Node or Reinstalling an Existing Node (`POST /api/v1/admin/enrollments`)

- **New Node**: Omit `node_id`. Choose `enrollment_mode: "installer"` (default 15-minute TTL) or `"preinstalled"` (default 7-day TTL).
- **Reinstalling an Existing Node (`§6.5`, `00111_pki_reinstall.sql`)**:
  1. First **revoke** the node's current incarnation (`POST /api/v1/admin/nodes/{node_id}/revoke`) so no active incarnation remains (`409 NODE_INCARNATION_ACTIVE` is returned if an incarnation is still active; `409 NODE_RETIRED` if the node was permanently retired).
  2. Call `POST /api/v1/admin/enrollments` with `"node_id": "<existing_node_uuid>"`.
  3. Run `fvpn-agent enroll -config /etc/vpn-node/agent.json -token-file /run/vpn-node/enrollment.token` on the node. A new `incarnation_id` and mTLS certificate are issued under the same `node_id`.

### 3.2 Updating Node Class, Region, or Capacity (`PATCH /api/v1/admin/nodes/{node_id}`)

First fetch the node (`GET /api/v1/admin/nodes/{node_id}`) and capture the response `ETag` header (`"<resource_version>"`). Supply `If-Match` and `Idempotency-Key` (`428 Precondition Required` if `If-Match` is missing; `412 Precondition Failed` if stale):

`[Verified on dev/test stack]`
```bash
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "node_class": "standard",
    "region": "tw-tpe",
    "capacity": {
      "down_bps": "900000000",
      "up_bps": "900000000",
      "member_down_bps_cap": "100000000",
      "member_up_bps_cap": "100000000"
    }
  }'
```
*(To clear an explicit capacity ceiling and revert to policy defaults, send `"clear_capacity": true` and omit `"capacity"`.)*

### 3.3 Publishing, Disabling, and Unpublishing Exits (`PUT /api/v1/admin/nodes/{node_id}/publication`)

Exit publication controls whether member devices route traffic through the node (`§7.3`):
- `"published"`: Creates/enables the NetBird exit route and `member-to-exit` policy for the node's class, and sets `publication_intent: "published"` in the node's signed desired state. Requires the node to have a valid incarnation, `provisioning_state = "ready"`, and an approved serving release (`409 MINIMUM_VERSION_VIOLATION` or `409 STATE_CONFLICT` otherwise).
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
Always poll the returned `Operation` (`GET /api/v1/admin/operations/{operation_id}`) until `state == "succeeded"`.

### 3.4 Revoking (`POST .../revoke`) vs. Retiring (`DELETE /api/v1/admin/nodes/{node_id}`)

- **Revoke (`POST /api/v1/admin/nodes/{node_id}/revoke`)**:
  Immediately revokes the node's current mTLS client certificate (`node_certificates`), revokes its serving lease, unpublishes its NetBird exit route, and deletes/disables its NetBird peer (`AT07`). The `node_id` remains in `revoked` state and can be reinstalled with a new incarnation (`§6.5`).
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

## 4. Running an Update Batch and Handling `needs_attention` & `unconfirmed` (`§13`, `docs/updates/central-lifecycle.md`)

### 4.1 Registering a Release and Updating `release_support` (`§13.3`, `§13.4`)

1. Build and sign the release (`amd64` + `arm64`) using `fvpn-release` and upload artifacts to `https://releases.example.com/<release_id>/` (see [`deploy.md` §8](deploy.md)).
2. Register the signed manifest envelope via `POST /api/v1/admin/releases`:
   `[Verified on dev/test stack]`
   ```bash
   curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/releases" \
     -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
     -H "Idempotency-Key: $(uuidgen)" \
     -H "Content-Type: application/json" \
     -d "{
       \"manifest_url\": \"https://releases.example.com/${RELEASE_ID}/manifest.envelope.json\",
       \"manifest_envelope\": $(cat dist/manifest.envelope.json)
     }"
   ```
3. Create and activate a `release_support` policy revision that includes `${RELEASE_ID}` in both `approved_release_ids` and `serving_release_ids` (`approved_release_ids` must be a subset of `serving_release_ids`, and `revoked_release_ids` must be disjoint from both).

### 4.2 Creating an `all_nodes` Update Batch (`POST /api/v1/admin/update-batches`)

In v1, fleet updates target `mode: "all_nodes"` (`§13.4`). Only one active update batch is permitted at a time (`F-06` fleet lock).

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/update-batches" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d "{
    \"release_id\": \"${RELEASE_ID}\",
    \"target_selector\": { \"mode\": \"all_nodes\" },
    \"reason\": \"Roll out release 2026.10.1 to all active nodes\"
  }"
```

### 4.3 Monitoring Targets and Handling `needs_attention` & `unconfirmed` (`GET /api/v1/admin/update-batches/{batch_id}`)

`GET /api/v1/admin/update-batches/{batch_id}` embeds a cursor-paginated `targets` list (`?cursor=...&limit=...`; `targets_url` points to the same endpoint, `Contract Facts`). Each `UpdateTarget` includes two required boolean flags (`§13.10`):

| Flag / State | Meaning | Operator Action |
|---|---|---|
| **`unconfirmed: true`** | The node was dispatched an update intent (`queued` / `dispatched` / `running`), or its heartbeat shows the new release, **but** central has not yet received a signed/authenticated `POST /api/v1/agent/update-results` receipt confirming terminal completion and `policy_generation` (`D-043`). | 1. Check if the node is online (`GET /api/v1/admin/nodes/{node_id}`).<br>2. If the node completed the update while central was unreachable, `fvpn-agent` / `fvpn-updater flush` will automatically re-submit the journaled result from `/var/lib/vpn-node/update-journal` on its next poll.<br>3. On the node, you can trigger immediate result delivery with `/opt/vpn-node/current/fvpn-updater flush -config /etc/vpn-node/updater.json`. |
| **`needs_attention: true`** | Set when any of the following holds:<br>• Target state is `failed`, `rolled_back`, or `recovery_required`<br>• Target reported `recovery_required: true` in its `UpdateResult`<br>• Target has been in a non-terminal state longer than `nodes.json:stuck_seconds` (`900s`) without progress. | 1. Inspect `last_error_code`, `stage`, and `recovery_required` on the target.<br>2. **Transient failure** (`ARTIFACT_FETCH_FAILED`, `DISK_SPACE_LOW` after freeing disk, `AUTHORIZATION_EXPIRED`, `UPDATE_BUSY`): Retry via `POST /api/v1/admin/update-batches/{batch_id}/retry` (increments `attempt` and issues a fresh `authorization_expires_at`).<br>3. **`rolled_back`**: Node automatically rolled back to `last_good_release_id` and restored member forwarding if `last_good` is still in `serving_release_ids` (`AT26`). Inspect binary/health-check logs before retrying.<br>4. **`recovery_required: true`** (rollback also failed or current release was revoked with no valid rollback target): Node has fail-closed member forwarding (`forwarding.blocked` latch set, exit code `2`). Log into the host or reinstall the node via `POST /api/v1/admin/nodes/{node_id}/revoke` + `POST /api/v1/admin/enrollments` (`node_id` set). |

Retrying failed/rolled-back targets in a batch `[Verified on dev/test stack]`:
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/update-batches/${BATCH_ID}/retry" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d "{
    \"node_ids\": [\"${NODE_ID}\"],
    \"reason\": \"Disk space cleared on node; retry attempt\"
  }"
```

---

## 5. Freezing, Recovering, and Adjusting the Usage Ledger (`§10.8`, `§10.9`, `§17.3`)

The `fvpn-usage-admin` CLI operates directly against PostgreSQL and the central signing vault (`/var/lib/fvpn/vault`) and records an append-only `audit_events` row for every mutation.

### 5.1 Freezing and Recovering the Usage Ledger (`freeze`, `begin-recovery`, `complete-recovery`)

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

### 5.2 Inspecting Member Usage Breakdown & Recording Adjustments (`drill-down`, `adjust`)

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

## 6. Suspending Members, Managing Grants, and FUP Overrides (`§5.1`, `§9.3`, `§11`, `§14.5`)

### 6.1 Suspending and Reactivating a Member (`PATCH /api/v1/admin/members/{member_id}`)

Suspending a member (`"status": "suspended"`) immediately revokes member forwarding across all exit nodes and disables all of the member's NetBird peers via an asynchronous `member.suspend` operation (`AT04`). Payment webhooks never auto-unsuspend an admin-suspended member (`§9.3`).

`[Verified on dev/test stack]`
```bash
# Fetch current member ETag first via GET /api/v1/admin/members/${MEMBER_ID}
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "suspended",
    "suspension_reason": "Terms of service violation / abuse report",
    "reason": "Suspend member pending investigation"
  }'
```
To reactivate, send `"status": "active"` and `"suspension_reason": null` with the new `If-Match` ETag.

### 6.2 Issuing and Revoking Finite Entitlement Grants (`POST /api/v1/admin/members/{member_id}/grants`)

Manual entitlement grants must have a finite `[starts_at, ends_at)` interval bounded by `members.json:grant_max_seconds` (`F-05` rejects unbounded grants):

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}/grants" \
  -H "Cookie: __Host-fvpn_session=${ADMIN_SESSION}" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "kind": "courtesy",
    "starts_at": "2026-10-29T00:00:00Z",
    "ends_at": "2026-11-05T00:00:00Z",
    "reason": "7-day courtesy access while billing issue is resolved"
  }'
```
To revoke a grant early, call `DELETE /api/v1/admin/members/${MEMBER_ID}/grants/${GRANT_ID}` with `Idempotency-Key`.

---

## 7. Key and Secret Rotation Runbooks (`§6.4`, `§16.2`, `docs/pki/rotation.md`)

### 7.1 Rotating Central Ed25519 Signing Keys (`desired-state`, `lease`, `recovery`, `ledger-epoch`)

Central signing keys in `/var/lib/fvpn/vault` use a **two-step overlap rotation** (`signing.FileProvider.Rotate`) so nodes never fail signature verification during rollout:
1. **Generate & Overlap (`Rotate`)**:
   Call `vault.Rotate(ctx, purpose, validityDuration, overlapDuration)` (where `overlapDuration >= 2 * desired_state_seconds`). The new key becomes `active` for signing, while the previous key moves to `retiring` and remains in `vault.TrustBundle(ctx)` until its overlap window expires.
2. **Distribute Updated Trust Bundle to Nodes**:
   Export `vault.TrustBundle(ctx)` to `/etc/vpn-node/trust.json` on nodes (or include in the next node release).
3. **Emergency Compromise (`Revoke`)**:
   If a central signing key is compromised, call `vault.Revoke(ctx, keyID)`, immediately distribute the updated `/etc/vpn-node/trust.json` (which marks `"revoked": true` for that `key_id`; `pkg/envelope` rejects any envelope carrying a signature from a known revoked key even if another valid signature is present), and re-issue desired state and leases.

### 7.2 Rotating the Offline Release Signing Key (`fvpn-release` & `fvpn-release-trust-v1`)

1. On the offline build machine, generate the new release signing key (`fvpn-release keygen -key-id release-prod-2027-01 ...`).
2. Construct a `fvpn-release-trust-v1` rotation envelope containing both the retiring key and the new key, and sign it with **both** the retiring release key and the new release key (`docs/pki/rotation.md`).
3. Apply the signed rotation envelope to the central vault via `vault.UpdateReleaseTrust(ctx, rotationEnvelopeBytes, now)` and distribute the updated `/etc/vpn-node/release-trust.json` to nodes before signing new releases solely with `release-prod-2027-01`.

### 7.3 Rotating Stripe Webhook Secrets and Billing Vault Keys (`members.json`)

- **Stripe Webhook Secret**:
  1. Create a new webhook signing secret in Stripe alongside the existing one.
  2. Add the new secret file path to `members.json:webhook_secret_files` (keep both current and old files in the array) and reload `fvpn-control`.
  3. Retire the old secret in Stripe and remove its path from `webhook_secret_files`.
- **Billing PII Vault Key (`billing.Vault`)**:
  1. Generate a new 32-byte random key file (`0600`), add it to `members.json:vault_key_files` under a new key ID (e.g., `"vault-prod-2027-01"`), keeping `"vault-prod-2026-01"` in the map so existing ciphertexts decrypt cleanly.
  2. Set `"vault_active_key": "vault-prod-2027-01"` and restart `fvpn-control` and `fvpn-worker`.
