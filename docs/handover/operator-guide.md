# Operator Guide (`docs/handover/operator-guide.md`)

This guide defines the routine operational runbooks for Freedom VPN administrators (`§6`–`§15`, `§17`–`§18`).

For API examples, use the admin role cookie and obtain `ADMIN_CSRF_TOKEN` from `GET /auth/session?role=admin`; authenticated mutations also require the exact configured Origin ([control authentication](../control/AUTH.md)).

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

## 2. How to Read Operations, Audit Events, and OpenAPI Contract Status (`§14.4`, `§14.5`, `§15.3`, `D-029`, `D-056`, `D-057`, `D-059`)

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

The admin operation view exposes only bounded step failure causes, including `member_forwarding_unconfirmed`, the `data_path_probe` reason vocabulary (`forwarding_disabled`, `stale_generation`, `tunnel_not_passed`, `check_failed`, `serving_checks_unconfirmed`, `isolation_unconfirmed`, `readiness_unconfirmed`, `lease_unconfirmed`), `desired_state_unavailable`, the publication/read-back vocabulary, and `publish_rejected:<400..599>`. Other stored messages, and every member operation view, use `Step failed; quote the operation ID.`; a generic message is not evidence that no internal cause was recorded. See [control operations](../control/OPERATIONS.md), [node wiring](../nodes/WIRING.md), and `internal/operations/operations.go` for the allowlist.

Inspecting an operation via API `[Verified on dev/test stack]`:
```bash
curl -fsS "https://vpn.example.com:8443/api/v1/admin/operations/${OPERATION_ID}" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}"
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
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}"
```

### 2.3 OpenAPI Contract Inventory (`47` Operations) (`D-065`, `#164`, `#176`, `#179`, `docs/contracts/README.md`, `docs/nodes/WIRING.md`, `pkg/apierr/apierr.go`)

`contracts/openapi.yaml` (OpenAPI `1.7.0`, revision 11) defines **47 HTTP operations** across `/api/v1`. All 47 are wired in `apps/control` on `main` (wire-nodes follow-up 17 `#179` wired `getFleetConfig` and `updateFleetConfig` at `GET`/`PUT /api/v1/admin/fleet-config` in `internal/nodes/httpwire/fleet_config.go`, `D-065`; wire-nodes follow-up 19 `#200` wired `listNodeIncarnations`; see §3.6 below):
- **Fleet Offline Lease Ceiling (`GET /api/v1/admin/fleet-config`, `PUT /api/v1/admin/fleet-config`, `D-065`, `#176`, `#179`, `internal/nodes/httpwire/fleet_config.go`)**:
  `GetFleetConfig` and `UpdateFleetConfig` expose the singleton `fleet_config.max_offline_lease_seconds` (`60..86400`, `default_max_offline_lease_seconds: 86400`) under strong `ETag` / `If-Match`, `Idempotency-Key`, and a required `reason`, recording an immutable `fleet_config.update` audit event with both `before` and `after` snapshots and returning synchronous `200 OK` `FleetConfig` (see §3.6 below for the operator workflow, HTTP status codes, and lease renewal timing).
- **Member Groups (`GET /api/v1/admin/groups`, `POST /api/v1/admin/groups`, `D-057`, `D-059`)**:
  `CreateAdminGroup` writes an immutable `member_groups` row (unique `code` and `name`), records a typed `group.create` audit event on `member_group:<id>`, and stores the idempotent `201 MemberGroup` receipt in one transaction (`replay` returns the original `201` body). `ListAdminGroups` orders by `code`, supports exact `code` filtering and cursor pagination, and counts every attached exit in `exit_count` (including unpublished or offline exits).
- **Authenticated Resolver Document Delivery (`GET /api/v1/agent/resolver-documents/{document_id}`, `D-056`)**:
  `GetResolverDocument` authorizes the mTLS-authenticated node's current incarnation and current or immediately previous desired-state generation before consulting `nodes.ResolverDocumentSource` (`store.NewDocumentReader(pool)`). Unknown and unissued document IDs both return `404 NOT_FOUND`. Validated responses stream the exact canonical RFC 8785 JSON bytes (`<= 1 MiB`) with `ETag: "sha256:<hex>"`. See §5.1 below and [node wiring](../nodes/WIRING.md) for signed issuance and periodic reconciliation.
- **Node Enforcement Verification Reasons in `ErrorCode` (contract revision 8 (#164), `docs/contracts/README.md`, `pkg/apierr/apierr.go`, `apps/portal/src/api/errors.ts`)**:
  Contract revision 8 (#164) adds the nine `node/enforce` verification reasons to the `ErrorCode` enum alongside the existing `METER_PATH_INVALID`, `ISOLATION_BASELINE_MISSING`, and `METER_EPOCH_CHANGE` codes (12 enforcement reasons in total): `ISOLATION_POLICY_INVALID`, `GENERATION_MISMATCH`, `HOOK_PRIORITY_MISMATCH`, `COUNTER_ORDER_INVALID`, `FLOWTABLE_BYPASS`, `DEVICE_CLASS_MISSING`, `QOS_CLASS_MISSING`, `MARK_SEMANTICS_VIOLATED`, and `PROBE_UNBOUNDED`. These codes are mirrored as report-only constants in `pkg/apierr` (without HTTP status mappings) and mapped to operator messages in `apps/portal/src/api/errors.ts`; they appear in `ObservationReport.errors[].code` (`NodeError.code`) and in the `error_code` field of the `source_binding`, `qos`, `lease`, `internet_only`, and `meter` entries of `ObservationReport.checks`, and are never returned as a browser-facing HTTP `Error.code`.

---

## 3. Enrolling, Classifying, Publishing, Revoking, Retiring Nodes, and Fleet Lease Config (`§6`, `§7`, `§14.5`, `§16.3`, `D-055`–`D-058`, `D-061`, `D-065`)

Portal pointers: `/admin/groups` lists and creates member groups; `/admin/nodes/{node_id}` shows attached groups and submits replacement attachments with publication under `If-Match`. Its checks table shows sanitized `NodeCheck.detail`. `/admin/policies/dns` shows threat-feed refresh status and the revision cadence. [Portal README](../portal/README.md) and [portal operations](../portal/OPERATIONS.md) remain authoritative for portal behavior; group IDs/codes are admin-only.

### 3.1 Enrolling a New Node or Reinstalling an Existing Node (`POST /api/v1/admin/enrollments`)

- **New Node**: Omit `node_id`. Set `delivery: "installer"` (default 15-minute TTL / `900s`) or `"preinstalled"` (default 7-day TTL / `604800s`), plus optional `name`, `region`, `notes`, and `member_group_ids`.
- **Bootstrap & One-Off NetBird Correlation Group (`D-055`, `docs/nodes/WIRING.md`)**:
  When the enrolled node calls `POST /api/v1/agent/bootstrap` over mTLS, central (`node.provision.requested`) creates a single-use NetBird setup key (`usage_limit=1`, non-ephemeral) whose `auto_groups` contain `fvpn-unpublished-nodes` and a dedicated empty per-enrollment correlation group (`fvpn-enroll-peer-<hash>`). The worker's `node.peer.discover` handler (`DiscoverNodePeer`) correlates the unique non-ephemeral, userless peer in that correlation group without hostname, address, or timing inference, records `node:<node>:<incarnation>` in `external_bindings`, detaches and deletes the temporary correlation group with read-back, and commits `peer_readback`.
- **Discovery conflicts and withdrawal**: `correlation_unavailable_revoke_and_re_enroll` means the attempt has lost safe correlation; revoke that attempt and re-enroll. Never infer a peer from its hostname or IP. Revocation reads back deletion of the uniquely correlated peer even before discovery records `peer_id`; ambiguous membership fails closed and leaves evidence for investigation. See [node wiring](../nodes/WIRING.md#d-055-adversarial-read-back-and-withdrawal).
- **Reinstalling an Existing Node (`§6.2`, `00111_pki_reinstall.sql`, `AT07`)**:
  1. First **revoke** the node's current incarnation (`POST /api/v1/admin/nodes/{node_id}/revoke`) so no active incarnation remains (`409 NODE_INCARNATION_ACTIVE` is returned if an incarnation is still active; `409 NODE_RETIRED` if the node was permanently retired).
  2. Call `POST /api/v1/admin/enrollments` with `"node_id": "<existing_node_uuid>"` and `"delivery": "installer"`.
  3. Run `fvpn-agent enroll -config /etc/vpn-node/agent.json -token-file /run/vpn-node/enrollment.token` on the node. A new `incarnation_id` and mTLS certificate are issued under the same `node_id`.
- **Enrollment Acceptance Harness (`#173`, `#182`, [`docs/e2e/at/enrollment/RUN.md`](../e2e/at/enrollment/RUN.md), [`docs/e2e/at/enrollment/summary.md`](../e2e/at/enrollment/summary.md), [`docs/e2e/at/enrollment/requests.md`](../e2e/at/enrollment/requests.md), [`docs/e2e/at/enrollment/AT06.md`](../e2e/at/enrollment/AT06.md), [`docs/e2e/at/enrollment/AT07.md`](../e2e/at/enrollment/AT07.md), [`docs/e2e/RUN.md`](../e2e/RUN.md))**:
  The records-first enrollment acceptance driver (`tests/e2e/at/enrollment/run.py` and `tests/e2e/at/enrollment/live_test.go`, `#173`) drives `AT06` (token delivery modes, expiry, single-use, correlation-group peer discovery, and raw-token log/response redaction) and `AT07` (CSR key-usage/SAN discipline, mTLS enforcement, revocation fence, reinstall under the same `node_id` with a new `incarnation_id`, and retirement `409 NODE_RETIRED`). Initially recorded as `blocked` on `ENR-01` because the orchestrator's assigned `66000` was never a legal port base, it runs on stack `w4-enroll` / port base `12000` after devstack follow-up 11 (`#182`) added slots `12000` and `6000` to the 10-slot port base table (`6000 + 6000*n`, `n = 0..9`; harness-side items `ENR-02`..`ENR-07` are tracked in `docs/e2e/at/enrollment/requests.md`).

### 3.2 Updating Node Region, Member Groups, or Capacity (`PATCH /api/v1/admin/nodes/{node_id}`)

First fetch the node (`GET /api/v1/admin/nodes/{node_id}`) and capture the response `ETag` header (`"<resource_version>"`). Supply `If-Match` and `Idempotency-Key` (`428 Precondition Required` if `If-Match` is missing; `412 Precondition Failed` if stale):

`[Verified on dev/test stack]`
```bash
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
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
- `"published"`: Creates/enables the NetBird exit route (`0.0.0.0/0`) and `member-to-exit` policy for the exit's attached `exit_groups` (`group_ids` / `member_group_ids`), and sets `publication_intent: "published"` in the node's signed desired state. Requires the node to have a valid incarnation, `provisioning_state = "ready"`, an approved serving release (`409 MINIMUM_VERSION_VIOLATION` or `409 STATE_CONFLICT` otherwise), and non-empty `exit_groups` so `netbird_route_readback` can complete.
  - **Atomic group attachment (`D-057`, `exits.SetPublicationTx`):** `PublicationRequest.group_ids` is persisted atomically under the node/exit locks and `If-Match` alongside the publication intent, single generation/version advance, operation, outbox event, and `exit:<uuid>` audit row. Omitting `group_ids` preserves existing membership, `[]` clears it, and a non-empty array replaces it (`422` with `/group_ids` detail if any group UUID is unknown). `NodeDetail` exposes the sorted membership in both `group_ids` and `member_group_ids`. A group disappearing after validation also returns `422 VALIDATION_FAILED`, detail path `/group_ids`, message `unknown group`; the entire membership/publication transaction rolls back. Refresh the groups and node ETag before correcting and resubmitting.
- `"disabled"`: Temporarily removes the node from member exit routing (`node.disable`) while keeping the node enrolled and online for diagnostics or updates.
- `"unpublished"`: Withdraws the exit publication (`node.unpublish`).

`[Verified on dev/test stack]`
```bash
curl -fsS -X PUT "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}/publication" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "publication_intent": "published",
    "group_ids": [
      "01950000-0000-7000-8000-000000000001"
    ],
    "reason": "Exit readiness checks verified"
  }'
```
Poll the returned operation and node detail; completion requires `status == "succeeded"` and `publication_observed == "active"`. A failed operation needs investigation, not indefinite polling.

If publication returns `422 NODE_NOT_SERVICEABLE` before creating an operation, inspect readiness, capacity, and baseline checks first.

**Publication `data_path_probe`, zero-binding rule, and attachment basis (`SECOPS-11`, `#148`, `#160`, `internal/nodes/publication_evidence.go`, `internal/nodes/publication_gate.go`, `internal/exits/publication_gate.go`, `docs/nodes/WIRING.md`):** `data_path_probe` fails with `NODE_NOT_SERVICEABLE` unless the current incarnation's selected observation satisfies `PublicationForwardingEvidence` for the issued signed `DesiredStateEnvelope` at the acknowledged generation:

1. The agent reports `member_forwarding_enabled: true`, `provisioning_state: ready`, `data_health: healthy`, `safety_status: passed`, `tunnel_probe: passed`, and a nonzero `applied_config_generation` at least as new as the operation's requested generation.
2. **With one or more `device_bindings` in the signed envelope:** `lease.lease_id` must be a nonzero UUID with `lease.expires_at` in the future, and `lease`, `qos`, and `source_binding` must all report `passed`.
3. **Zero-binding published generation (`device_bindings: []`, `#148`):** When the current incarnation's issued signed envelope (`desired_state_envelopes.payload->'signed'`) at the acknowledged generation has `publication_intent = "published"` and an explicit empty `device_bindings` list (`[]`), `lease`, `qos`, and `source_binding` may be `passed` or `not_run` and no armed member lease is required. Any check reporting `failed` still blocks publication (`check_failed`).
4. **Publication attachment basis and first-populated-generation gate (`#160`, `internal/nodes/publication_gate.go`, `internal/exits/publication_gate.go`, `docs/nodes/WIRING.md`):**
   - Before a `node.publish` operation completes, `RecordPublicationAttachmentTx` records `publication_attachment_state` and `publication_attachment_basis` (`empty_exception` when the acknowledged signed envelope had `device_bindings: []`, or `populated_verified` when it had one or more `device_bindings`) in `operations.observed_refs`.
   - **What `publication_observed = 'withdrawn'` with cause `member_forwarding_unconfirmed` means and how it clears:** When the most recent succeeded `node.publish` operation used `empty_exception` and the first populated desired-state generation (`device_bindings` non-empty) is compiled, `QueuePublicationGateTx` queues a `node.publish` operation under `system:publication-gate` via `node.publication_changed`. While the node's observation is still on an older generation (`stale_generation`) or lacks passing `lease`, `qos`, and `source_binding` checks (`serving_checks_unconfirmed`), `failGatedPublish` under the per-node advisory lock (`fvpn:publication:<node_id>`) withdraws the NetBird route (`WithdrawExit`), confirms its absence or disabled state via `ReadExitState`, sets `exits.publication_observed = 'withdrawn'` while keeping `publication_intent = 'published'` (recording the fixed failure cause `member_forwarding_unconfirmed` with the specific gate reason in the step `reason`, bounded retry, and a D-029 `node.publish` transition audit on `exit:<id>` under `system:publication-gate`), and `CheckPublicationDelivery` withholds the populated desired-state envelope with retryable `503 DEPENDENCY_UNAVAILABLE` (`"Publication route verification pending."`) until withdrawal is confirmed. Once withdrawal is confirmed, the node fetches and applies the populated desired-state generation and submits an observation with passing `lease`, `qos`, and `source_binding` checks; observation progress then queues a fresh recovery `node.publish` operation under `system:publication-gate` that re-attaches and reads back the NetBird route, transitions `exits.publication_observed` back to `'active'` (with a D-029 `node.publish` audit), and establishes `publication_attachment_basis = 'populated_verified'`.
   - **`populated_verified` attachments never block delivery:** Once `populated_verified` is recorded, this gate never withdraws the route and never blocks `GET /api/v1/agent/desired-state` (`200 OK`), including when adding a second binding or revoking a device while NetBird is unavailable.
5. **Operation step `reason` vocabulary (`internal/nodes/publication_evidence.go`):** When `data_path_probe` fails, the step's `observed.reason` (and admin step `message`) records one bounded cause alongside the fallback `member_forwarding_unconfirmed`: `forwarding_disabled`, `stale_generation`, `tunnel_not_passed`, `check_failed`, `serving_checks_unconfirmed`, `isolation_unconfirmed`, `readiness_unconfirmed`, or `lease_unconfirmed`.
6. **Agent heartbeat `forwarding.reason`, lease `Retry-After`, host preflight, `VerifyServing` model rules, and `ISOLATION_POLICY_INVALID` on pending conntrack delete (`#146`, `#156`, `#159`, `#162`, `#164`, `#166`, `#172`, `docs/node/README.md`, `docs/poc/G2-netlab.md`, `node/agent/revocation.go`, `node/agent/decide.go`, `node/agent/plane.go`, `node/agent/sync.go`, `node/enforce/apply.go`, `node/enforce/verify.go`):**
   - In `ObservationReport.forwarding.reason`, the node agent reports `policy_missing` when desired state is missing/unverified/expired/unpublished or when the stored lease is missing or belongs to another generation (in which case the `lease` check reports `STALE_GENERATION`), and reports `lease_expired` only when the anchored monotonic deadline of the lease for the applied generation has passed (other reasons: `safety_check_failed`, `check_failed`, `meter_unavailable`, `update_in_progress`, `recovery_required`). `awaiting_member_grants` (D-067) is the distinct provisional case, and revision 11 names it `AWAITING_MEMBER_GRANTS`: the node lease for this generation is inside its anchored deadline and `config_generation` equals the desired generation, but `lease.members` is empty while desired `device_bindings` is non-empty. `safety_status` stays `passed`. The `lease` check is `failed` with detail `Serving lease has no member grants yet.`; `qos` and `source_binding` carry `AWAITING_MEMBER_GRANTS` (not `VALIDATION_FAILED`) with details `Rate policy could not be applied.` and `No member binding was admitted.`; `errors` carries exactly one entry `code=AWAITING_MEMBER_GRANTS`, `stage=lease`, `message=Serving lease has no member grants yet.`. The agent renews on the D-067 schedule (one heartbeat, then doubling, capped at 60 seconds). This is not a deadline failure and not a malformed policy. A passed deadline stays `lease_expired` / `LEASE_EXPIRED` (`Serving lease is not in force.`). A generation mismatch stays `policy_missing` / `STALE_GENERATION`. An empty desired `device_bindings` list carries neither reason nor this code. The step that applies a new desired generation requests a serving lease immediately when the stored lease's `config_generation` differs, without waiting for the renewal lead window, and admits bare IPv4 `allowed_ipv4` addresses in lease bindings as `/32`.
   - **Single-attempt lease renewal and `Retry-After` (`#162`, `#166`, `node/internal/controlclient/api.go`, `node/agent/sync.go`, `node/agent/decide.go`):** `controlclient.Client.RenewLeaseOnce` returns `StatusError` (`RetryAfter`, `RetryAt`) immediately while `Client.RenewLease` waits out `Retry-After` without a second request. The agent calls `RenewLeaseOnce` at most once per step (`e.leaseAttempted`) and honours `Retry-After` via monotonic `e.leaseRetryAt` (`Input.LeaseRetryPending`) without blocking heartbeats or observations. Under `#166`, when the node already holds a verified, unexpired lease for the applied generation (`leaseInForce == true`), a `Retry-After` backoff **keeps `member_forwarding_enabled: true` (`serving`)**, reports the `lease` check as `status: "passed"` with `detail: "Lease renewal is waiting for Retry-After."`, and keeps the local watchdog heartbeat `serving`. Only when no fresh lease exists for the applied generation (for example, on the initial lease request for a new generation) does `LeaseRetryPending` report the `lease` check as `failed` (`error_code: RATE_LIMITED`, `detail: "Lease renewal is waiting for Retry-After."`) and withhold forwarding (`policy_missing`).
   - **Host prerequisites, `VerifyServing` exact model rules, and `ISOLATION_POLICY_INVALID` on pending `conntrack -D` (`#156`, `#159`, `#162`, `#164`, `#166`, `#172`, `node/agent/plane.go`, `node/agent/revocation.go`, `node/internal/agentstore/store.go`, `node/enforce/apply.go`, `node/enforce/verify.go`, `docs/poc/G2-netlab.md`):**
     - `SystemPlane.MissingExecutables()` (`#162`, `#166`) uses `enforce.Executor` directly to preflight `nft`, `tc`, and `ip` and calls `enforce.RequireConntrack(p.exec)` at startup before installing the boot baseline; a missing binary keeps forwarding off with `safety_check_failed`, logs only the executable name, and reports `DEPENDENCY_UNAVAILABLE` at stage `preflight` plus failed `lease`, `qos`, and `source_binding` checks (`DEPENDENCY_UNAVAILABLE`).
     - `VerifyServing` (`#172`, `node/enforce/verify.go`, `node/enforce/parse_nft.go`) accepts only the rendered model: every `accept` rule in `forward` must be a verified serving accept (member DNS to `@resolver_v4:53` after upload `@allowed_v4` drop, per-device upload/download accept after `@denied_v4` and `@host_v4` drops, or download `ct state established,related` after download `@allowed_v4` drop) or a verified probe accept (`ct state related` counts as established; upload and download `@allowed_v4` drops are tracked separately; `@resolver_v4` exact match, `@denied_v4`/`@host_v4` drops, and non-53 drop `ctbadns` to `@resolver_v4` are required with or without a probe; only chains `{forward, input, output, postrouting}` are allowed in `inet fvpn` keyed by `(family, table, name)` with duplicates rejected; `input` accepts before drops and `interval` flags on `allowed_v4`, `probe_v4`, `probe_dst_v4`, and `resolver_v4` are rejected). `Render` (`#172`) unconditionally merges `Datapath.SNAT/32` into `@host_v4` and unions `SpecDeniedV4()` with `ExtraDeniedV4` (when the node's uplink sits on a public on-link L2 subnet, the operator must include that prefix and gateway IP in `ExtraDeniedV4`).
     - In `enforce.Apply` (`#156`, `#159`, `#172`), after `VerifyServing` and `VerifyQoS` succeed, `Apply` populates `verified.Counters` and `verified.Samples` (`Samples` skips unbound devices and zero metadata) before running the `m.Revoke` loop, and attempts `deleteConntrack` (`conntrack -D -s <ip>` and `conntrack -D -d <ip>`) for every address in `m.Revoke` before returning `*ConntrackDeleteError{Pending}` (`ApplyResult.Verified = true`).
     - **How to interpret `ISOLATION_POLICY_INVALID` on a serving vs. non-serving node (`#164`, `#166`):** When `plane.Apply` returns `res.Verified == true` alongside a conntrack delete failure (`enforce.IsConntrackDeleteFailure(err)`), the verified nftables ruleset has already removed the revoked bindings from `@allowed_v4` and drops their upload and download packets ahead of `ct state established,related accept`. Therefore, `fvpn-agent` (`#166`) **keeps the verified serving ruleset installed (`member_forwarding_enabled: true`, local heartbeat `serving`, and `internet_only`, `qos`, `source_binding`, and `meter` checks `passed`)**, persists the failed addresses under `pending_revocations` in local SQLite `meta` (merged with any newly dropped addresses on later generations and retried on every apply tick until `conntrack -D` succeeds), and reports one `NodeError` in `ObservationReport.errors` with `code: ISOLATION_POLICY_INVALID`, `stage: "apply"`, and `message: "conntrack delete pending; serving ruleset verified"`. By contrast, when `ISOLATION_POLICY_INVALID` or another revision-8 enforcement verification code (`GENERATION_MISMATCH`, `HOOK_PRIORITY_MISMATCH`, `COUNTER_ORDER_INVALID`, `FLOWTABLE_BYPASS`, `DEVICE_CLASS_MISSING`, `QOS_CLASS_MISSING`, `MARK_SEMANTICS_VIOLATED`, `PROBE_UNBOUNDED`, `METER_PATH_INVALID`, `ISOLATION_BASELINE_MISSING`, `METER_EPOCH_CHANGE`) appears on a failed `source_binding`, `qos`, `lease`, `internet_only`, or `meter` check with `member_forwarding_enabled: false` (`safety_check_failed`), ruleset or QoS verification failed and the agent withdrew to the fail-closed baseline without acknowledging the generation or renewing the lease.

The publication gate is agent evidence that forwarding is armed, not a traffic measurement. The [security-ops follow-up 4 publication record](../e2e/at/secops/publication.md) verifies both live halves of `SECOPS-11` (`422 VALIDATION_FAILED` when capacity is unset, then `202 -> succeeded` on a zero-binding published generation under `#148`), the [members follow-up 3 forwarding record](../e2e/at/members/forwarding-observation.md) proves counted member forwarding (`0 -> 2214/2411` bytes, origin `200`) once the client selects the advertised exit, [devstack follow-up 9 (`#153`)](../e2e/followup9-verification.md) and [devstack follow-up 10 (`#170`)](../e2e/RUN.md) install `conntrack` and `tcpdump` in `deploy/devstack/Dockerfile.node`, add real device revocation/replacement (`device-revoke`), a dedicated updater probe peer (`FVPN_E2E_PROBE=1`, `update-good` with `probe_omission_count=0`), 60-peer native admission, and a **second real exit lab hook** (`Stack.StartSecondExit` / `EnrollAt(..., "node2")` via `deploy/devstack/compose.yaml` `second-exit` profile, verified by `TestSecondExit` / `second-exit` as an operator-run prerequisite for multi-exit `AT09` and `AT21`), and the [security-ops follow-up 5 records (`#165`)](../e2e/at/secops/README.md) verify counted serving flow through the selected exit ([`counted-flow.json`](../e2e/at/secops/counted-flow.json), `SECOPS-09` closed with `0 -> 399` upload / `0 -> 678` download bytes, `usage_events = 2`, `usage_rollups = 4`, and zero `probe_up`/`probe_down` leakage), post-revoke control/node coherence ([`post-revoke.json`](../e2e/at/secops/post-revoke.json), `F3-02` passing), `AT29` deadline and watchdog faults ([`deadlines.json`](../e2e/at/secops/deadlines.json)), and the `AT31` 2-device counted-capacity subset ([`capacity.json`](../e2e/at/secops/capacity.json)). In addition, [portal follow-up 10 (`#175`)](../../apps/portal/e2e/portal.spec.ts) replaces single-shot post-operation DOM reads with auto-retrying web-first Playwright assertions (`toBeVisible`, `not.toHaveCount(0)`, `toContainText`) and `OperationBanner` settlement checks; [agent follow-up 14 (`#181`)](../../node/agent/review_d063_test.go) synchronizes meter review sampling via `manualUsage` / `Meter.Barrier` (`node/agent/engine_test.go`), proves `Meter.Run` skips an `ErrBusy` tick without losing counters (`TestMTR03_RunSkipsBusySampleWithoutLosingCounters` in `node/internal/meter/meter_test.go`), and pins the test clock in `TestProbeException_AbsentChallengeDoesNotInstall` (`node/agent/probeexc_test.go`); and [devstack follow-up 11 (`#182`)](../e2e/followup11-verification.md) (`26 pass / 1 blocked / 0 fail`, [`docs/e2e/README.md`](../e2e/README.md), [`docs/e2e/RUN.md`](../e2e/RUN.md), [`docs/e2e/scenarios/gateway-principals.md`](../e2e/scenarios/gateway-principals.md), [`docs/e2e/scenarios/device-revoke.md`](../e2e/scenarios/device-revoke.md)) exercises lab `trusted_proxies` (`/32` gateway trust) and nginx `X-Forwarded-For` (`gateway-principals`, `fvpn-e2e-nodeconfig http-burst`), adds the **TEST ONLY** finite-grace prepare option (`fvpn-e2e-prepare -grace-seconds N` / `FVPN_E2E_GRACE_SECONDS`, `SECOPS-04`), adds controlled stale-client revocation (`Stack.PauseNativeControl` `SIGSTOP` in `device-revoke`, while the strict post-withdrawal node-ingress drop assertion remains `blocked` awaiting an independent node-side ingress fixture with `w1-netlab`), adds read-only node resource telemetry (`Stack.StartNodeTelemetry` at `5s` cadence sampling cgroup v2 CPU/memory, process FDs, and `nf_conntrack_count` — workload measurement only, not hardware certification), and expands the port base table to 10 slots (`6000 + 6000*n`, `n = 0..9`: `12000` assigned to `w4-enroll`, `6000` reserved, `18000..60000` unchanged, `66000` rejected).

### 3.4 Node Egress Probe & Sanitized Check Details (`D-058a`, `D-058b`, `#171`, `node/agent/probe.go`, `node/agent/probeexc.go`)

Each node's observation report (`POST /api/v1/agent/observations`) includes three required checks (`internet_only`, `dns_policy`, `tunnel_probe`):
- **Signed `egress_probe` precedence (`D-058a`)**: When `DesiredState.egress_probe` is non-null on an in-force desired state (`1..4` TCP IP-literal targets, `timeout_ms` `100..10000` default `2000` ms), `fvpn-agent` dials every listed target and ignores the local `/etc/vpn-node/agent.json:egress_probe_target`. If a signed `egress_probe` is structurally invalid, `tunnel_probe` fails closed (`status: "failed"`, `data_health: "degraded"`) without falling back to the local target. When `DesiredState.egress_probe` is null or absent, the agent dials `egress_probe_target` if configured.
- **Sanitized `NodeCheck.detail` and `probe_omitted` precedence (`D-058b`, `#168`, `#171`, `docs/node/README.md`, `node/agent/probeexc.go`, `node/agent/attest.go`)**: `NodeCheck.detail` is bounded to `1..500` single-line printable ASCII characters and sanitized on both agent and central ingest so URLs, tokens, secret keys, file paths, SQL, and stack traces never enter `node_observations` or the Admin Portal. Normally, `tunnel_probe.detail` reports which egress probe source ran (`"Egress probe used the signed targets."`, `"Egress probe used the local target."`, `"Egress probe used no extra target."`, or `"Signed egress probe was rejected."`). When a live matching `health-challenge.json` carries `probe_omitted {peer_id, reason}` (`#168`, `#171`, because the updater's D-063 issuer refused the configured `probe_peer`), `tunnel_probe.detail` instead reports `"probe_omitted: <reason>"` (taking precedence over the `ProbeSource` sentence, never containing the peer ID or any address, and retained across sidecar removal or consumption until a later challenge carrying `probe` or `probe_omitted` replaces it), while `tunnel_probe` remains `status: "not_run"` and `data_health` remains `"unknown"` with member forwarding closed (`ProbeRan = false`).

Node detail also includes typed `dns_policy_evidence` from the selected current-incarnation observation. The admin node page labels it as node-reported and shows document ID, digest, generation, outcome, applied time, error code and per-feed evidence. `null` means not reported; neither a newer desired policy nor successful document delivery is substituted as proof of apply. Members see the explanation at `/member/dns-protection`; support must inspect the admin evidence for them.

### 3.5 Revoking (`POST .../revoke`) vs. Retiring (`DELETE /api/v1/admin/nodes/{node_id}`)

- **Revoke (`POST /api/v1/admin/nodes/{node_id}/revoke`)**:
  Immediately revokes the node's current mTLS client certificate (`node_certificates`), revokes its serving lease, unpublishes its NetBird exit route, and deletes its NetBird peer (`AT07`, `AT13`, `internal/nodes/httpwire/revocation_adapter.go`). The `node_id` remains in `revoked` state and can be reinstalled with a new incarnation (`§6.2`).
- **Retire (`DELETE /api/v1/admin/nodes/{node_id}`)**:
  Permanently decommissions the node (`lifecycle_state = 'retired'`). A retired node can never be re-enrolled (`409 NODE_RETIRED`), while historical usage events and audit logs remain intact (`§7.1`).

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/nodes/${NODE_ID}/revoke" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{"reason": "Host maintenance / key compromise precaution"}'
```

### 3.6 Configuring the Fleet Offline Lease Ceiling (`GET`/`PUT /api/v1/admin/fleet-config`, `D-065`, `#176`, `#179`, `§14.5`, `§16.3`, `AT29`, `docs/nodes/README.md`, `docs/nodes/WIRING.md`, `internal/nodes/httpwire/fleet_config.go`)

- **What `max_offline_lease_seconds` governs (`§16.3`, [`docs/nodes/README.md`](../nodes/README.md), [`docs/nodes/WIRING.md`](../nodes/WIRING.md))**:
  `fleet_config.max_offline_lease_seconds` governs the longest a node keeps serving member traffic on its last signed `ServingLeaseEnvelope` when `fvpn-control` is unreachable. On each lease issuance or renewal (`POST /api/v1/agent/leases/renew` / `RenewServingLease`), `internal/nodes` stamps `max_offline_seconds = fleet_config.max_offline_lease_seconds` and computes:
  $$\text{node\_lease\_end} = \text{signed\_server\_time} + \text{max\_offline\_lease\_seconds}$$
  $$\text{member\_lease\_end} = \min(\text{signed\_server\_time} + \text{max\_offline\_lease\_seconds},\ \text{effective\_service\_end},\ \text{node\_lease\_end})$$
  Once a node's monotonic deadline for its applied generation expires without a fresh signed lease, `fvpn-agent` fails closed (`member_forwarding_enabled: false`, `forwarding.reason: "lease_expired"`) and withdraws member forwarding while keeping the control channel up. Operators and acceptance harnesses (`AT29`) configure this ceiling through `GET`/`PUT /api/v1/admin/fleet-config` rather than writing `fleet_config` in SQL (`D-065`). It does not alter any member-facing API or portal view.
- **Range `60`–`86400` and why the floor is `60` (`D-065`, `#176`, `#179`, `node/agent/sync.go`)**:
  `max_offline_lease_seconds` must be an integer in **`60`–`86400`** seconds (`1` minute to `24` hours). The read-only field `default_max_offline_lease_seconds` is always `86400` (`24` hours — the `§16.3` ceiling and migration seed, not a deployment recommendation; see [`calibration.md` §1.4](calibration.md)). The **`60` s floor** is a test-lab floor (`AT29` deadline faults) chosen so renewal still runs ahead of expiry: `fvpn-agent` renews its serving lease at `min(120 s, window/2)` remaining (`renewLead` in `node/agent/sync.go` — `30 s` remaining on a `60 s` window) with a `10 s` heartbeat tick. The Admin Portal warns when an operator enters a value below `900 s` (`15` minutes).
- **`GET` → strong `ETag` → `PUT` flow with `If-Match`, `Idempotency-Key`, and `reason` (`internal/nodes/httpwire/fleet_config.go`, `TestD065_FleetConfigHTTPReadWrite`, `TestD065_FleetConfigMissingRowUsesIssuerDefault`, `TestD065_FleetConfigReleaseWriterInterleaving`)**:
  1. **Read current config and `ETag` (`GET /api/v1/admin/fleet-config`):** Returns `200 OK` `FleetConfig` (`max_offline_lease_seconds`, `default_max_offline_lease_seconds: 86400`, `resource_version`, `updated_at`) and strong `ETag: "<resource_version>"`. If the singleton row (`fleet_config.id = 00000000-0000-0000-0000-000000000001`) is absent in an unseeded database, `GET` returns the issuer default (`max_offline_lease_seconds: 86400`, `default_max_offline_lease_seconds: 86400`, `resource_version: "0"`, `updated_at: "1970-01-01T00:00:00Z"`, `ETag: "0"`) without writing (`TestD065_FleetConfigMissingRowUsesIssuerDefault`).
  2. **Update (`PUT /api/v1/admin/fleet-config`):** Supply the quoted strong `If-Match: "<resource_version>"` header, a UUID `Idempotency-Key`, `X-CSRF-Token`, `Origin`, and JSON body `{"max_offline_lease_seconds": <60..86400>, "reason": "<1..500 chars>"}` (`reason` is trimmed and required). Inside the shared HTTP request transaction, `UpdateFleetConfig` ensures the singleton exists (`INSERT ... resource_version = 0 ON CONFLICT DO NOTHING`, so `If-Match: "0"` inserts the missing row and advances to version `"1"`), locks the singleton row (`SELECT ... FOR UPDATE`, serializing with `fleetupdate.CreateBatch` and release-support writers without touching `generation` or release-support columns, `TestD065_FleetConfigReleaseWriterInterleaving`), checks `If-Match`, updates `max_offline_lease_seconds`, `resource_version = resource_version + 1`, and `updated_at = clock_timestamp()`, records the `fleet_config.update` audit event, completes the idempotency record (`ResourceType: "fleet_config"`, `ResponseStatus: 200`, `OperationID: nil`), and returns synchronous **`200 OK`** `FleetConfig` with the new strong `ETag` (creating zero `operations` or `outbox_events` rows).

  `[Verified on dev/test stack]`
  ```bash
  # 1. Read the current FleetConfig and capture the strong ETag ("<resource_version>")
  ETAG=$(curl -fsSI "https://vpn.example.com:8443/api/v1/admin/fleet-config" \
    -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
    | awk -F': ' 'tolower($1)=="etag"{gsub(/\r|"/,"",$2); print $2}')

  # 2. Update max_offline_lease_seconds with If-Match, Idempotency-Key, and a reason
  curl -fsS -X PUT "https://vpn.example.com:8443/api/v1/admin/fleet-config" \
    -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
    -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
    -H "Origin: https://vpn.example.com" \
    -H "If-Match: \"${ETAG}\"" \
    -H "Idempotency-Key: $(uuidgen)" \
    -H "Content-Type: application/json" \
    -d '{
      "max_offline_lease_seconds": 3600,
      "reason": "Tighten offline serving lease ceiling to 1 hour for maintenance window"
    }'
  ```
- **HTTP error outcomes (`428` / `412` / `409` / `422`, `internal/nodes/httpwire/fleet_config.go`, `TestD065_FleetConfigHTTPReadWrite`)**:
  - **`428 PRECONDITION_REQUIRED`**: Missing `If-Match` header (`"If-Match header is required."`).
  - **`412 RESOURCE_VERSION_CONFLICT`**: Stale `If-Match` ETag; the error envelope includes `details.current_resource_version` with the current decimal-string `resource_version`.
  - **`409 IDEMPOTENCY_CONFLICT`**: Reusing an `Idempotency-Key` with a different request payload (`max_offline_lease_seconds` or `reason`). Replaying the same `Idempotency-Key` with the identical payload (`replayFleetConfig`) returns `200 OK` with the original `FleetConfig` body and `ETag` reconstructed from the immutable `audit_events.after_diff` row keyed by the command reservation UUID `idempotency_requests.id` (the plaintext `Idempotency-Key` is never stored), even after subsequent writes have advanced `fleet_config`.
  - **`422 VALIDATION_FAILED`**: `max_offline_lease_seconds` outside `60..86400` (e.g., `59` or `86401`), empty or `> 500`-character `reason`, unknown JSON fields, or weak/malformed `If-Match` (`W/"..."`). (`400 MALFORMED_REQUEST` is returned if `Idempotency-Key` is missing or invalid; `403 FORBIDDEN` for a member session or missing/invalid CSRF token.)
- **Audit event `fleet_config.update` with both snapshots (`internal/nodes/httpwire/fleet_config.go`, `TestD065_FleetConfigHTTPReadWrite`)**:
  Every accepted `PUT /api/v1/admin/fleet-config` (including an accepted same-value write) appends one immutable `audit_events` row in the same transaction with `action = "fleet_config.update"`, `actor_role = "admin"`, `resource = "fleet_config:00000000-0000-0000-0000-000000000001"`, `operation_id = null`, `result = "succeeded"`, `before = {"max_offline_lease_seconds": <before>}`, and `after = {"max_offline_lease_seconds": <after>, "resource_version": "<ver>", "updated_at": "<rfc3339>", "reason": "<reason>"}`. Because idempotent replay reconstructs the original `200 OK` snapshot from this audit row, keep `audit_days` retention at least as long as `idempotency_seconds` (`180` days vs. `7` days default).
- **Lease timing (`TestD065_AT29_FleetConfigOnlyChangesNewLeases`, `internal/nodes/httpwire/fleet_config_integration_test.go`, [`docs/nodes/README.md`](../nodes/README.md))**:
  Updating `max_offline_lease_seconds` does **not** rewrite or revoke already-issued signed leases (`node_leases`): existing leases (and natural or `Idempotency-Key` replays of an existing lease while inside its fresh window) retain their signed `max_offline_seconds`, `expires_at`, and Ed25519 signature. The next lease issuance or renewal (`POST /api/v1/agent/leases/renew`) reads `fleet_config.max_offline_lease_seconds` inside the issuance transaction and copies the new window into the newly signed `ServingLeaseEnvelope`.

---

## 4. Running an Update Batch, Health Attestation, and Handling `needs_attention` & `unconfirmed` (`§13`, `D-053`, `docs/updates/central-lifecycle.md`, `docs/updates/node-updater.md`, `docs/updates/node-health-attestation.md`)

### 4.1 Registering a Release and Updating `release_support` (`§13.3`, `§13.4`)

1. Build and sign the release (`amd64` + `arm64`) using `fvpn-release` and upload artifacts to `https://releases.example.com/<release_id>/` (see [`deploy.md` §8](deploy.md)).
2. Register the signed manifest envelope via `POST /api/v1/admin/releases` (`ReleaseRegisterRequest`):
   `[Verified on dev/test stack]`
   ```bash
   curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/releases" \
     -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
     -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
     -H "Origin: https://vpn.example.com" \
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
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
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
   - All five `UpdateHealth` checks (`processes`, `identity_and_state`, `internet_only`, `dns_policy`, `tunnel_probe`) must report `"passed"`. The attestation must match the live challenge, release, transaction, node, incarnation and boot IDs, its nonce and validity window; observed agent and NetBird versions must match the manifest components. See [health attestation](../updates/node-health-attestation.md) for the exact file contract.
3. **Automatic Rollback & Fail-Closed Latch (`AT26`, `UPD03`–`UPD05`)**:
   - If health attestation fails, `fvpn-updater` rolls `/opt/vpn-node/current` back to `last_good_release_id` (up to `3` attempts, `RollbackInterval = 1s`) **only** if `last_good_release_id` is still in `serving_release_ids`, not in `revoked_release_ids`, and passes its own fresh `D-053` challenge-response health attestation (`status: "rolled_back"`, exit code `1`).
   - If no valid rollback target exists or rollback health attestation fails, `fvpn-updater` writes `/var/lib/vpn-node/update-journal/forwarding.blocked`, forces the fail-closed `inet fvpn` baseline (blocking all member forwarding while keeping control/SSH up), and records `status: "failed"`, `recovery_required: true`, `error_code: "ROLLBACK_FAILED"` (exit code `2`).

**Bounded synthetic probes (`§13.9`, `D-063`, `D-064`, `#142`, `#144`, `#150`, `#154`, `#156`, `#162`, `#168`, `#170`, `#171`, `docs/updates/node-updater.md`, `docs/updates/node-health-attestation.md`, `node/updater/probe.go`, `node/updater/probe_addresses.go`, `docs/node/README.md`, `docs/poc/G2-netlab.md`):**
- **Updater `probe_peer` configuration, per-issuance address validation, and `probe_omitted` evidence (`#142`, `#154`, `#156`, `#168`, `#171`, `node/updater/probe.go`, `node/updater/probe_addresses.go`, `node/updater/host.go`, `node/enforce/prefixes.go`, `docs/updates/node-health-attestation.md`):** `/etc/vpn-node/updater.json` accepts an optional `probe_peer` object (`peer_id` nonblank up to 128 bytes with no ASCII control characters, `source_v4` one IPv4 `/32` prefix, `destinations` `1..4` public IPv4 `/32` destination objects with `addr_v4`, `proto` `"tcp"` or `"udp"`, and `port` `1..65535`, and optional `resolver_dns`) plus optional `agent_config` (default `/etc/vpn-node/agent.json`, regular file `<= 1 MiB` verified via `os.Lstat`, `O_NOFOLLOW|O_NONBLOCK`, `fstat` regular check, and `os.SameFile` so symlinks, FIFOs, and replaced files fail closed as `agent_config_unavailable`; resolver credentials never opened). `probe_peer` is validated at `LoadConfig`, `Config.Engine()`, and every `issueChallenge` (`Engine.challengeProbe`) against:
  1. `enforce.SpecDeniedV4()` (`#156`): every IANA IPv4 Special-Purpose Address Registry block with `Globally Reachable != True` plus multicast `224.0.0.0/4` (`0.0.0.0/8`, `10.0.0.0/8`, `100.64.0.0/10`, `127.0.0.0/8`, `169.254.0.0/16` and `169.254.169.254/32`, `172.16.0.0/12`, `192.0.0.0/24`, `192.0.2.0/24`, `192.88.99.0/24` RFC 7526 6to4 relay anycast, `192.168.0.0/16`, `198.18.0.0/15`, `198.51.100.0/24`, `203.0.113.0/24`, `224.0.0.0/4`, `240.0.0.0/4`, `255.255.255.255/32`; globally reachable `192.31.196.0/24`, `192.52.193.0/24`, and `192.175.48.0/24` are allowed);
  2. the host's live IPv4 global-unicast interface addresses (`net.InterfaceAddrs`, refusing an empty filtered global-unicast IPv4 list with `node_addresses_unavailable` in `#168`, and refusing any matching host IPv4 address as `source_v4` or destination; other private/CGNAT `/32` overlay addresses remain valid as `source_v4`); and
  3. the resolved/literal addresses of `control_url` (from `agent_config`) and updater `result_url` (when set), plus `dns.resolvers[].dns_addr` and `dns.resolvers[].admin_url` hosts (refused as destinations regardless of `resolver_dns` or port), sharing a `2s` DNS lookup deadline.
  Config load and `Config.Engine()` fail closed with `updater: invalid probe_peer` (and malformed JSON fails with `updater: invalid config JSON` without echoing values, `SEC02`). If an input becomes unavailable or forbidden at challenge issuance (`#168`, `#171`), `Engine.issueChallenge` logs `"health challenge probe omitted"` once per fixed `probeRefusal` class per `Engine` (`malformed`, `denied_destination`, `node_source`, `node_destination`, `control_destination`, `resolver_destination`, `agent_config_unavailable`, `node_addresses_unavailable`, `control_addresses_unavailable`, or `resolver_addresses_unavailable`), writes `probe_omitted {peer_id, reason}` into `health-challenge.json` (mutually exclusive with `probe`, included in `Same`), commits `probe_omission/<txn_id>/<nonce>` in SQLite `meta` and `txn_events.detail` (`Journal.ProbeOmissions`) before publishing the challenge, and includes `probe_omitted` in local `health_attest` diagnostics (`fvpn-updater status`). On the agent (`#171`), a live matching challenge carrying `probe_omitted` sets `ObservationReport.checks[name=tunnel_probe].detail` to `"probe_omitted: <reason>"` (without peer ID or address, retained across sidecar removal/consumption until a later probe or omission challenge), leaves `tunnel_probe` `not_run` and `data_health` `unknown` with forwarding closed, and answers `health-attest.json` with the existing accepted reason token `"probe refused"`. **One-time compatibility boundary (`#168`):** because `healthattest.ReadChallenge` uses `DisallowUnknownFields`, agent binaries built before `#168` reject an omission-bearing challenge and fail closed; release notes and update rollout plans must account for this boundary when updating across `#168`. With `probe_peer` absent, none of these inputs is read (in `deploy/devstack` `#170`, `FVPN_E2E_PROBE=1` provisions a dedicated `probe-client` peer and read-only `probe-watch` observer on UDP `:4443`, passing `update-good` with `probe_omission_count = 0` and `probe_traversal_verified = true`).
- **Agent amended `D-064` probe exception and durable issuance (`#150`, `#162`, `docs/node/README.md`, `node/agent/probeexc.go`, `node/agent/plane.go`, `node/agent/attest.go`):** Under amended `D-064` (`#162`), the `§13.9` probe exception is installed **only** while another process holds `/var/lib/vpn-node/update.lock` (`nodelock.TryAcquire` returns `nodelock.ErrBusy`; `updateLockHeld` never waits for or keeps the lock) and a live `health-challenge.json` names this process and matches `Same`. Both installation and removal use `ApplyBaselineModel` (`BaselineModel`): a serving node withdraws member forwarding (`e.withdraw`) before installing the probe exception on the baseline model (zero member bindings), and cleanup re-applies the bare baseline (`bare.Probe = nil`). Both `installProbeException` and `answerHealth` validate the challenge window (`issued_at < now < expires_at`, `>= 1s` left, window `<= 300s`) using `trustedNow`. Any node-local destination (`datapath.snat` or any discovered `host_v4` address) on any port and every `enforce.SpecDeniedV4()` destination (including `192.88.99.0/24`) are refused by `challengeProbe` before dry-render, along with member peer IDs, non-`/32` sources, and current member binding collisions. Before touching the kernel, `installProbeException` records durable `probe_issuance` (`{nonce, hash, issued_at, deadline_ns, cleared}`, where `hash` is the canonical JSON challenge SHA-256 digest without storing the destination list, `SEC02`) in `agentstore` (`Meta` / `SetMeta`); a cleared issuance is never reinstalled across restarts, and an incomplete issuance on the same boot is capped to `min(old.Deadline - mono, expires_at - trustedNow)`. Cleanup failure clears `baselineOK` and `serving` and records the probe as `probe_clear_failed` (never attested as `passed`). `probe_up` and `probe_down` counters are excluded from member usage, and `fvpn-watchdog` never installs a probe exception.
- **Troubleshooting `PROBE_UNBOUNDED` and probe DNS carve-out (`#144`, `#156`, `docs/poc/G2-netlab.md`, `node/enforce/verify.go`):** In `node/enforce`, the `@probe_v4` and `@probe_dst_v4` sets fail closed (`PROBE_UNBOUNDED` / `ReasonProbeUnbounded`) on prefixes wider than `/32`, `nft -j` prefix or port-range objects, over-long lifetimes (`max(timeout, expires) > 300s` or `expires` exceeding the model's remaining window plus `1s` slack), `> 4` destinations (`ProbeMaxDestinations = 4`), or any overlap with a member binding's `AllowedV4`. When `ResolverDNS` is enabled on the probe (`#156`, `verifyProbeChains`, `verifyResolverSet`), `VerifyBaseline` and `VerifyServing` require the probe DNS carve-out to be at most one TCP and one UDP accept rule from `saddr @probe_v4` to `daddr @resolver_v4` port 53 (`iif` member ingress, `oif` uplink) with `@resolver_v4` matching the exact configured resolver IPv4 addresses; any wider probe rule fails closed with `ISOLATION_POLICY_INVALID`. Inspect `/etc/vpn-node/updater.json:probe_peer`, `/etc/vpn-node/updater.json:agent_config`, the active `health-challenge.json`, and the `inet fvpn` `probe_v4` / `probe_dst_v4` sets; see [`docs/poc/G2-netlab.md`](../poc/G2-netlab.md) for the enforcement rules and proof cases.

### 4.4 Monitoring Targets, Observation Authority, and Handling `needs_attention` & `unconfirmed` (`GET /api/v1/admin/update-batches/{batch_id}`, `#140`, `#150`, `docs/updates/central-lifecycle.md`, `internal/fleetupdate/serving.go`, `internal/fleetupdate/progress.go`)

`GET /api/v1/admin/update-batches/{batch_id}` embeds a cursor-paginated `targets` list (`?cursor=...&limit=...`; `targets_url` points to the same endpoint, `Contract Facts`). Operators see `unconfirmed` and `needs_attention` on each `UpdateTarget` in `GET /api/v1/admin/update-batches/{batch_id}` (`internal/fleetupdate/progress.go`) and as status badges on `/admin/updates` ([`apps/portal/src/pages/admin/AdminFleetUpdatesPage.tsx`](../../apps/portal/src/pages/admin/AdminFleetUpdatesPage.tsx), `§13.10`):

| Flag / State | Meaning | What Clears It / Operator Action |
|---|---|---|
| **`unconfirmed: true`** | Central lacks an agreeing authenticated observation for this job's exact `job_id`, `generation`, and `incarnation_id` (or the job has not been issued, or the latest matching heartbeat reported `current_release_id` / `last_good_release_id` values that disagree with the accepted result on `nodes`). | Cleared (`unconfirmed: false`) when an authenticated `POST /api/v1/agent/observations` arrives for the same `incarnation_id` carrying both the exact `update.job_id` and `update.generation` of the job and `current_release_id` / `last_good_release_id` values that match `nodes`. Compare `last_result_at`, `last_observed_at`, and the node's `update` object; flushing a result alone does not clear `unconfirmed`. |
| **`needs_attention: true`** | Failed, rolled-back or blocked job; stored attention flag (including applying/checking beyond configured `nodes.json:stuck_seconds`); or `already_current` inconsistent with the observed release. | Inspect target `error_code`, `stage`, `recovery_required`, `previous_release_id`, `health`, `candidate_health`, and local `fvpn-updater status`. Fix the underlying cause before calling `POST /api/v1/admin/update-batches/{batch_id}/retry`. |

**Observation authority rules and `update_observation.unexpected_release` (`#140`, `#150`, `#155`, `#167`, `docs/updates/central-lifecycle.md`, `internal/fleetupdate/serving.go`, `node/agent/observe.go`):**
1. **Bounded pre-result heartbeat seeding (`#140`, `#155`, `#167`, `Service.observeTx`, `auditUnexpectedRelease`):** Pre-result heartbeat seeding and `update_observation.unexpected_release` audits bind to the **reported job** (`effectiveJob := jobID`, falling back to `n.job` only for the legacy sequence adapter when `jobID == nil`, `#167`). Before the current incarnation has any accepted result (`update_results` row with `ack->>'accepted' = 'true'`), an observation may seed `nodes.current_release_id` **only** when the reported `last_good_release_id` agrees with the recorded `nodes.last_good_release_id` (`null` agreeing with `null`, `#167`) and `current_release_id` is either a release already recorded in `nodes.current_release_id` / `nodes.last_good_release_id` or the reported job's target (`targetRelease`), which is eligible **only** when the reported job is the node's current job (`equalID(effectiveJob, n.job)`), its `execution_incarnation_id` matches the authenticated incarnation, and its `NodeUpdateIntent` has been issued (`j.issued != nil`, `#167` — preventing a pre-intent heartbeat from seeding `nodes.current_release_id` and poisoning `attempt_release_id` in `update_jobs.observed_evidence`). If `last_good_release_id` disagrees or is unexpected, both `nodes.current_release_id` and `nodes.last_good_release_id` remain intact (`#167`). Heartbeats never establish `nodes.last_good_release_id` (only accepted results do), and `fleet_node_state.recovery_required` is derived from the recorded `last_good_release_id`, never an unverified heartbeat claim. Once at least one accepted result exists for the incarnation, only accepted results change `nodes.current_release_id`, `nodes.last_good_release_id`, and `fleet_node_state.recovery_required` (a heartbeat can raise `recovery_required` if the recorded `last_good_release_id` is in `revoked_release_ids`, never clear it).
2. **How to read `update_observation.unexpected_release` (`#155`, `#167`):** Before the incarnation's first accepted result, any other release reported in `current_release_id` or `last_good_release_id` is ignored for the version columns, withholds confirmation (`unconfirmed = true`), and appends at most one D-029 audit event `update_observation.unexpected_release` (`actor_role: node`, `actor_id: <node_uuid>`, `resource: node:<node_uuid>`, `result: failed`, `after: {"job_id", "release_id"}` bound to the reported `effectiveJob`, `#167`) per distinct `(job_id, release_id)` under the fleet lock. Because a fresh node incarnation starts with `nodes.current_release_id = NULL` and `nodes.last_good_release_id = NULL` (its boot release is centrally recorded only by its first accepted update result), a pre-apply heartbeat from a fresh node under its first job reports its boot release (`!= targetRelease`), which produces **one `update_observation.unexpected_release` audit for that boot release and job**—this single audit for a fresh node under its first job is expected, not a fault. By contrast, an `update_observation.unexpected_release` audit naming an unapproved/unexpected release or appearing on a node whose version columns were already recorded warrants inspecting `/opt/vpn-node/current`, `/etc/vpn-node/agent.json:release_id`, and `journal.db`.
3. **Disagreeing heartbeat withholds confirmation:** If a heartbeat for the job's `job_id` and `generation` reports a `current_release_id` or `last_good_release_id` that differs from the accepted values on `nodes` (or an untargeted pre-result release), central keeps the recorded values on `nodes`, leaves `last_update_generation` behind the job's generation, and keeps `UpdateTarget.unconfirmed = true` until an agreeing heartbeat arrives.
4. **Jobless reports record arrival only:** An observation report without both `update.job_id` and `update.generation` updates `last_observed_at` only; the numeric generation high-water mark (`last_update_generation`) and serving release are unchanged. `ObserveReportTx` validates `update.state` and `update.stage` against the generated contract enums in the domain before writing or confirming anything.
5. **Node-side release provenance (`#150`, `docs/node/README.md`, `node/agent/observe.go`):** On the node, `ObservationReport.current_release_id` defaults to the running binary's configured `release_id`; `journal.db` pointers override `current_release_id` only when the journal snapshot has a valid nonzero `job_id` at `locally_succeeded` or `rolled_back`.

### 4.5 Reading a Rolled-Back Update, Target Evidence, and Result Rejections (`D-062`, `#136`, `#140`, `#147`, `#149`)

1. **Target and Portal evidence (`#136`, `#147`, `internal/fleetupdate/progress.go`, `docs/updates/central-lifecycle.md`):** On `GET /api/v1/admin/update-batches/{batch_id}`, the central target mapper populates `previous_release_id`, `health`, and `candidate_health` on each `UpdateTarget` from the latest attempt's latest current accepted receipt (`ack.accepted = true` and `accepted_as_history = false`) and sets them to explicit JSON `null` when no current accepted receipt exists (`candidate_health` is populated only when the accepted result state is `rolled_back` and is `null` otherwise; history-only and rejected receipts never overwrite them). For `rolled_back`, `current_release_id` names the restored release, `previous_release_id` names the failed candidate, `health` describes the restored release, `candidate_health` describes the failed candidate (`null` if rollback began before candidate checks), and `error_code` retains the candidate failure. On `/admin/updates` ([`apps/portal/src/pages/admin/AdminFleetUpdatesPage.tsx`](../../apps/portal/src/pages/admin/AdminFleetUpdatesPage.tsx), [`docs/portal/OPERATIONS.md`](../portal/OPERATIONS.md)), admins see the restored health, failed candidate health, and candidate release on a rolled-back target, and an explicit "not yet reported" state for a target without a receipt. An empty manifest `rollback.allowed_release_ids` adds no restriction, while a nonempty list must include the restored release; serving, trust, revocation, and schema checks still apply.
2. **Local journal inspection (`node/updater`):** On the node, read the journal without starting an update:
   ```bash
   /opt/vpn-node/current/fvpn-updater status -config /etc/vpn-node/updater.json
   ```
   Alternatively use `status -journal /var/lib/vpn-node/update-journal/journal.db` (file, not directory; do not combine with `-config`). Inspect `candidate_health`, `candidate_health_json`, `health`, `journal_stage`, `recovery_required`, `last_result_state` and `last_rejection`. The snapshot is local durable evidence, not central completion. Restoration of a healthy binary alone does not prove member forwarding resumed; recheck publication, lease and serving checks.
3. **Result validation, failed-result cause, and conflict bounds (`#140`, `internal/fleetupdate/results.go`):** Every `failed` result requires a non-null `error_code` regardless of stage (a missing `error_code` at `check` or any other stage returns `422 VALIDATION_FAILED` with an immutable rejection receipt while the job remains `queued` with `needs_attention: false`). A valid pre-health failure (`claim` through `start`) can carry `health: null` with a non-null `error_code`; when accepted it marks the target `failed` with `needs_attention: true`. For conflicting replays (`409 ACK_CONFLICT` when a known `result_id` arrives with different payload bytes), central writes the `update_result_conflicts` row and the `update_result.conflict` audit event once per `(result_id, conflicting_hash)` in a single transaction; replaying an already-recorded conflicting hash returns `409 ACK_CONFLICT` with no new row or audit ([central lifecycle](../updates/central-lifecycle.md#terminal-evidence-and-receipts-d-062)). Do not rewrite journal payloads or result IDs manually.
4. **Updater delivery states:** The updater retains 409 as `conflict` and other permanent 4xx (except 429) as `rejected`, stops resending those rows, and allows later results through. `last_rejection` contains bounded HTTP/API/local codes. Transport failures, 5xx and 429 remain pending; 429 honors `Retry-After` across restarts. After correcting connectivity, `fvpn-updater flush -config /etc/vpn-node/updater.json` can send pending results; successful flush alone does not mean every result was accepted.
5. **Recovery latch:** If `recovery_required` is true or rollback fails, retain the fail-closed latch and follow [node recovery](../updates/node-updater.md); investigate locally or revoke and re-enroll if recovery cannot restore a supported release. Do not remove the latch to claim success.

**Direct terminal result and acceptance evidence (`#129`, `#135`, `#149`, `#158`, `#177`, [`docs/updates/central-lifecycle.md`](../updates/central-lifecycle.md)):** Central accepts a validated current-job terminal result (`succeeded`, `failed`, or `rolled_back`) directly from `queued` without requiring an earlier `applying` or `checking` receipt. The [updates follow-up 4 acceptance record](../e2e/at/updates/follow-up-4.md) passes `AT23`–`AT27`, the [updates follow-up 5 record (`#158`)](../e2e/at/updates/history/follow-up-5/summary.md) ([pre-result seeding](../e2e/at/updates/pre-result.md), [probe issuance](../e2e/at/updates/probe-issuance.md)) passes `AT23`, `AT25`, and `AT26`, and the [updates follow-up 6 record (`#177`)](../e2e/at/updates/summary.md) ([requests](../e2e/at/updates/requests.md)) on a fresh `#170` stack (`w4-updates` / `42000`) passes **`AT26`** (`rolled_back` with revision 7 `previous_release_id`, `health`, and `candidate_health` evidence) and resolves **`F5-01`** (fresh pre-result case `c` emits one bounded `update_observation.unexpected_release` audit and keeps null columns under `#167`'s `last_good_release_id` agreement rule), while `AT23` and `AT25` wait for the shared-lab `probe_peer` export (`F5-02`) and `AT24` and `AT27` are governed by the **incarnation-wide accepted-result authority rule** in [`docs/updates/central-lifecycle.md`](../updates/central-lifecycle.md) (`F6-02`: once any `UpdateResult` is accepted for an incarnation, heartbeats cannot seed `nodes.current_release_id` or emit `update_observation.unexpected_release` audits on subsequent jobs for that incarnation, so pre-result seeding on a later batch requires a fresh node incarnation and product behavior is not weakened). Physical `arm64` execution remains recorded as a real-hardware item (`§9`).

Retrying failed/rolled-back targets in a batch (`UpdateBatchRetryRequest`) `[Verified on dev/test stack]`:
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/update-batches/${BATCH_ID}/retry" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d "{
    \"target_ids\": [\"${TARGET_ID}\"],
    \"reason\": \"Disk space cleared on node; retry attempt\"
  }"
```

---

## 5. Threat Feeds, Resolver Document Publication, and `dns.feed.refresh_failed` Runbook (`§12`, `D-054`, `D-056`, `D-058c`, `D-058d`, `D-061`, `docs/dns/PUBLISH.md`, `docs/dns/FEEDS.md`, `docs/node/README.md`)

### 5.1 How Threat Feeds and Resolver Documents Are Published and Delivered

Threat-feed `attribution: null` (or omission) is accepted. A non-null blank or whitespace-only note is rejected in source validation, policy validation (`/payload/attribution`, `schema_invalid`) and before worker fetch; null does not bypass source URL, host, port or size checks. See [contract conventions](../contracts/README.md) and `internal/dnspolicy/sources.go`.

1. **Threat-Feed Refresh (`dns.feed.refresh`, `00181_dns_feed_contents.sql`, `00182_dns_feed_review.sql`, `D-054`)**:
   - Activating a `threat_feed` policy revision enqueues `dns.feed.refresh` in `outbox_events` and requires both `desired_state_compile` and `dns_feed_refresh` steps (`observed_refs.dns_feed_content`) on the `policy.activate` operation (`00182_dns_feed_review.sql`).
   - `fvpn-worker` fetches `source_url` over HTTPS (`NewSourceHTTPClient` with SSRF protection rejecting loopback, RFC1918, CGNAT `100.64.0.0/10`, link-local, multicast, ULA, and documentation literals, and refusing redirects), verifies the exact SHA-256 against the revision's pinned `content_sha256` on bootstrap, normalizes domains via IDNA, enforces the removal guard (`feed_max_removal_percent`, default `50%`) and addition cap (`MaxAdditionPercent = 100%`), and writes immutable entries to `dns_feed_contents` and status to `dns_feed_refresh_state` (fenced by `activation_generation` and `last_receipt`).
   - `GET /api/v1/admin/policies?type=threat_feed` exposes `PolicyDefinition.threat_feed_status` (`D-058c`) with `approved_content_sha256`, `last_good_content_sha256`, `rule_count`, `last_attempt_status` (`never`, `success`, `failed`), `consecutive_failures`, `next_due_at`, and bounded `last_error_code`.
2. **Resolver Document Compilation (`dns.publish`, `00180_dns_publish.sql`, `00182_dns_feed_review.sql`, `D-056`)**:
   - Activating a `dns_profile` requires both `desired_state_compile` and `dns_document` steps (`observed_refs.dns_document`). `fvpn-worker` compiles a deterministic canonical JSON `resolver_document` (`kind = "resolver_document"`, `schema_version = 1`), stores it in `dns_resolver_documents`, and advances `dns_publication_heads` (`dns_profile_feeds_eligible` ensures all referenced threat-feed revisions remain active and within `[valid_from, valid_until)`).
   - The desired-state compiler includes the eligible resolver reference before hashing/signing and records issuance in the same transaction. `node.resolver_documents.reconcile` delivers changed references for the one active fleet DNS profile to current registered nodes, including unpublished nodes, without another publication command. Its schedule is bounded by `min(nodes.json:reconcile_seconds, 60)` seconds, subject to dispatcher/dependency success, not an apply SLA. Equal references cause no generation/ETag churn; unavailable documents retain last-good, while ambiguous profiles or invalid documents fail closed ([node wiring](../nodes/WIRING.md)).
3. **Node Delivery, Verification & Local AdGuard Home Credentials (`node/agent/resolver.go`, `node/dns`, `D-056`, `D-061`, `docs/node/README.md`, `docs/nodes/WIRING.md`)**:
   - `GetResolverDocument` (`GET /api/v1/agent/resolver-documents/{document_id}`) authorizes the mTLS-authenticated node's current incarnation and current/previous desired-state generation via `RecordIssuedResolverDocumentTx` and streams the canonical RFC 8785 bytes (`ETag: "sha256:<hex>"`). See §5.1 below and [node wiring](../nodes/WIRING.md) for signed issuance and periodic reconciliation.
   - On the exit node (`D-061`, `#150`, `docs/node/README.md`, `node/agent/dnsconfig.go`), `agent.json` configures `dns.resolvers[]` (`1..8` host-local AdGuard Home targets with IP-literal `admin_url` and `dns_addr`) and `dns.resolver_credentials_file` (an absolute path to a regular JSON file `{"username","password"}` with mode `0600`, owned by `0:0`, and a single hard link `Nlink == 1`). `fvpn-agent run` loads `resolver_credentials_file` at startup and reloads it on `SIGHUP` without stopping; wrong file mode, non-root owner/group, or multiple hard links refuses the check (`FORBIDDEN`, detail `"Resolver credentials are not usable."`) without logging the secret or URL.
   - `fvpn-agent` verifies the exact byte length (`size_bytes`) and SHA-256 (`content_sha256`) before parsing, applies the policy to all configured AdGuard Home instances via `node/dns.Applier` (with automatic rollback on partial failure), and reports `ObservationReport.dns_policy_evidence` (`outcome`: `applied`, `failed`, `partial`, or `not_run`, plus `feeds[]` evidence). `dns_policy` is recorded as `applied` (`passed`) only when the applier's read-back `DocumentHash` (case-insensitive, trimmed) and decimal `PolicyGeneration` match the signed `ResolverDocumentRef`; any digest or generation mismatch reports `failed` / `VALIDATION_FAILED` (`#150`, `node/agent/resolver.go`).

The agent's `dns_policy` check passes only after the exact document ID and digest in the in-force desired state have been applied. A revoked/expired policy cannot be attested as passed from desired state alone; missing resolver application support reports `not_run` / `DEPENDENCY_UNAVAILABLE` (see [node runtime](../node/README.md)).

**Redirecting AdGuard admin listener:** The node's resolver admin client refuses every HTTP redirect, including same-host port changes. An AdGuard admin origin behind an HTTP→HTTPS redirect makes policy apply fail with an error such as `HTTP 307 from GET /control/dns_info`. Configure `dns.resolvers[].admin_url` to the final permitted origin directly, with usable TLS trust if HTTPS, then restart the agent for the configuration change. Credential-only changes can use `SIGHUP`. The client omits the redirect location and credentials from errors; the agent exposes `failed` / `VALIDATION_FAILED` / `Resolver policy was not applied.` rather than the raw origin (`node/dns/client.go`, `node/agent/resolver.go`).

### 5.2 Troubleshooting `dns.feed.refresh_failed` (`docs/dns/FEEDS.md`)

When a threat-feed fetch fails (`fetch_failed`, `http_status`, `read_failed` -> `DEPENDENCY_UNAVAILABLE`; `content_mismatch` -> `HASH_MISMATCH`; `oversize`, `content_too_large` -> `PAYLOAD_TOO_LARGE`; `invalid_config`, `invalid_size`, `outside_validity`, `ssrf_refused`, `parse_error`, `zero_rules`, `delta_breach` -> `VALIDATION_FAILED`), `fvpn-worker` retains `last_good_content` in `dns_feed_refresh_state`, increments `consecutive_failures` and `fvpn_dns_feed_refresh_total`, and appends an `audit_events` row with `action = 'dns.feed.refresh_failed'`.

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
- **`content_mismatch` (`HASH_MISMATCH`)**: Bootstrap bytes differ from the reviewed snapshot pin. Confirm the intended upstream version and approve + activate a new `threat_feed` revision with the reviewed `content_sha256`.
- **`delta_breach` (`VALIDATION_FAILED`)**: Upstream feed removed more than `feed_max_removal_percent` (default `50%`) of entries relative to either the previous last-good or the bootstrap snapshot, or added more than `MaxAdditionPercent` (`100%`) relative to the bootstrap snapshot. Verify whether the upstream list was truncated or materially replaced before approving a new revision.

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

Suspending a member (`"admin_suspended": true`) requests withdrawal of member forwarding across all exit nodes and withdraws access from all of the member's NetBird peers via an asynchronous `member.suspend` operation (`AT04`, `docs/members/README.md`). Payment webhooks never auto-unsuspend an admin-suspended member (`§9.3`).

`[Verified on dev/test stack]`
```bash
# Fetch current member ETag first via GET /api/v1/admin/members/${MEMBER_ID}
curl -fsS -X PATCH "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "If-Match: \"${ETAG}\"" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "admin_suspended": true,
    "suspension_reason": "Terms of service violation / abuse report"
  }'
```
To reactivate, send `"admin_suspended": false` and `"suspension_reason": ""` with the new `If-Match` ETag (`member.unsuspend` operation).

Plan downgrades keep the oldest authorizations (`authorized_at`, then device ID); excess devices become `pending` and are withdrawn. Loss of entitlement withdraws access; restoration readmits only previously authorized, non-revoked devices with verified mappings within the current quota. Removed exit groups trigger policy withdrawal on previously serving nodes too. Inspect `device.authorize` denial audits for the bounded reasons `device_limit_reached`, `entitlement_inactive` or `identity_unmapped`; denials create no operation. See [members operations](../members/README.md#quota-withdrawals-and-finite-binding-history).

### 7.2 Issuing Finite Entitlement Grants (`POST /api/v1/admin/members/{member_id}/grants`)

Manual entitlement grants (`GrantCreateRequest`) require a `source` (`admin_grant`, `compensation`, `trial`, or `migration`), a `reason`, and a finite `[valid_from, valid_until)` interval bounded by `members.json:grant_max_seconds` (`F-05` rejects unbounded grants):

`[Verified on dev/test stack]`
```bash
curl -fsS -X POST "https://vpn.example.com:8443/api/v1/admin/members/${MEMBER_ID}/grants" \
  -H "Cookie: __Host-fvpn-admin=${ADMIN_SESSION}" \
  -H "X-CSRF-Token: ${ADMIN_CSRF_TOKEN}" \
  -H "Origin: https://vpn.example.com" \
  -H "Idempotency-Key: $(uuidgen)" \
  -H "Content-Type: application/json" \
  -d '{
    "source": "admin_grant",
    "valid_from": "2026-10-29T00:00:00Z",
    "valid_until": "2026-11-05T00:00:00Z",
    "reason": "7-day courtesy access while billing issue is resolved"
  }'
```
*(Note: Grant creation returns `202 Accepted` with a `member.grant` operation on `entitlement_grant:<grant_id>`. There is no HTTP `DELETE` endpoint for grants in the 47-operation OpenAPI contract; grants expire automatically at `valid_until` via `members.reconcile`, or asynchronous withdrawal can be requested via `PATCH /api/v1/admin/members/{member_id}` with `"admin_suspended": true`.)*

### 7.3 Identity Continuity, Client Rotation and Duplicate Pre-Check (`D-052`, `#134`, `#137`, `#139`, `docs/control/AUTH.md`, `docs/members/README.md`)

One member is identified by canonical `(issuer, subject)`, independent of the portal client ID (`client_scope`). Both browser login (`internal/auth/store.go`) and NetBird identity discovery (`internal/devices/identity.go`) normalize issuers through `internal/members/issuer.Canonical` (`internal/members/issuer/issuer.go`):
- `issuer.Canonical` lowercases the `http`/`https` scheme and host, removes default ports (`:80` for `http`, `:443` for `https`), and trims only **literal** trailing slashes (`/`) from the escaped path while preserving percent-encoded trailing slashes (`%2F`, `%2f`) and other percent-encoded bytes as written (`#139`, `docs/members/README.md`).
- For an operator configuring `oidc_issuer` in `control.json`, `issuer.Validate` requires the configured string to equal its `issuer.Canonical` form at startup (no uppercase scheme/host, no default port, no literal trailing slash, no userinfo, query, or fragment). Because `%2F` is preserved, an issuer path ending in `%2F` is a distinct canonical issuer from the unencoded path; never rely on percent-encoded slashes collapsing onto an unencoded path.
- Control and worker must use the same shared issuer and public subjects.
- When an existing `(issuer, subject)` binding logs in after a portal `client_id` rotation (`#134`, `#137`), `PostgresStore.Login` (`internal/auth/store.go` `updateMemberBindingScope`) executes a single `UPDATE identity_bindings SET client_scope=$2, verified_at=clock_timestamp() WHERE id=$1` (enabled by the mutable-`client_scope` trigger in `db/migrations/00186_identity_bindings_client_scope_mutable.sql`) and appends one `identity_binding.proof_updated` audit event carrying the binding ID and fixed reason `client_scope_updated`. The old savepoint and `verified_at`-only fallback were removed in `#137`; subsequent logins from the rotated client see the stored `client_scope` and do not repeat `identity_binding.proof_updated`. It never creates a duplicate member or links accounts by email. See [auth](../control/AUTH.md) and [members](../members/README.md).

Before applying forward migrations to an existing deployment, run this read-only, count-only pre-check:

```sql
SELECT count(*) AS duplicate_identity_groups
FROM (
  SELECT issuer, subject
  FROM identity_bindings
  GROUP BY issuer, subject
  HAVING count(*) > 1
) AS conflicts;
```

The issuer/subject uniqueness migration locks and repeats this check; any duplicate group aborts it with SQLSTATE `23505` and only the group count. The pre-check does not replace the migration's locked check. If nonzero, stop the upgrade and have the identity owner investigate historical bindings and their dependent members, devices, grants and billing/audit history in an authorized private maintenance session. A duplicate login fails closed with `403 FORBIDDEN`. There is no merge tool or automatic merge by design: obtain a reviewed data-remediation plan from the member/control owners, preserve identity/history, then rerun the count and migrations after resolving the ambiguity. Do not delete arbitrary bindings, guess from email, bypass the unique constraint, or put raw subjects/issuers into tickets or logs.

The requested browser role now lives in `oidc_login_flows.role` (`admin` or `member`). The role migration backfills legacy verifier prefixes; the reader accepts that legacy format for one release so in-flight logins can complete. A requested admin role still requires the configured admin claim and MFA. Do not manually edit login-flow verifiers ([auth](../control/AUTH.md)).

### 7.4 Control Rate Limiters and Dedicated Operation Classes (`D-060`, `#141`, `#145`, `#152`, `#165`, `#169`, `#170`, `#174`, `docs/control/README.md`, `docs/control/AUTH.md`, `internal/config/config.go`, `internal/config/proxies.go`, `internal/httpapi/middleware.go`, `internal/httpapi/jsondoc.go`)

- **Connection-address principal and `trusted_proxies` (`#141`, `#174`, `internal/config/proxies.go`, `internal/httpapi/middleware.go`, `docs/control/README.md`, `docs/control/AUTH.md`):** Pre-authentication `ingress` (per class), `anonymous`, and browser `login` / `session` limiters key IPv4 (including IPv4-mapped IPv6) per `/32` address and IPv6 per canonical `/64` prefix so a single IPv6 `/64` allocation cannot mint a fresh bucket per `/128` and exhaust `max_rate_principals` (default `10000`). With `trusted_proxies` (`FVPN_TRUSTED_PROXIES`) empty (the default) or when the TCP peer (`r.RemoteAddr`) is outside the configured CIDRs, forwarding headers are ignored and the principal is the TCP peer. When the TCP peer belongs to a listed CIDR in `trusted_proxies` (validated by `config.ParseTrustedProxies`: each CIDR must be at least `/8` for IPv4 or `/32` for IPv6, rejecting `0.0.0.0/0`, `::/0`, shorter prefixes, and IPv4-mapped IPv6 CIDRs because `"trusting the world lets any peer choose its bucket"`), the public listener walks `X-Forwarded-For` from the right, skips trusted hops (entries that themselves match `trusted_proxies`), and uses the rightmost untrusted address (at most `64` entries examined; a missing header or a chain of only trusted hops keeps the TCP peer). Any TCP peer or chosen `X-Forwarded-For` entry that is not a parseable IPv4 or IPv6 address fails closed with HTTP `429 RATE_LIMITED` (`Retry-After`, counted on `limiter="address"`, never falling back to an entry further left and never inserted into the principal map). `X-Real-IP` and RFC 7239 `Forwarded` stay ignored even from a trusted peer; the agent mTLS listener never reads `X-Forwarded-For` (and mTLS `subject` keys, the login concurrency gate, access logs, and metric labels are unchanged).
- **Per-class limiter tables, separate `anonymous` class table, mTLS certificate-before-insert, O(1) LRU eviction, and concurrency metric (`#169`, `internal/httpapi/middleware.go`, `docs/control/README.md`, `docs/control/AUTH.md`):** Pre-authentication connection limiters are partitioned into per-class `ingress[class]` tables (`browser`, `agent`, `heartbeat`, `lease`, `usage`, `anonymous`), unauthenticated `anonymous` routes (`EnrollNode`, `ReceiveProviderWebhook`) use a dedicated `anonClass` table separate from the authenticated `subjects` table, and each table is independently capped at `max_rate_principals = 10000` (so a single IPv6 `/48` of 65,536 `/64` prefixes can fill only the anonymous tables and never evicts or blocks browser or mTLS principals). On the agent mTLS listener (`AgentListener: true`), machine client-certificate authentication (`AuthenticateAgent`) runs **before** any limiter table lookup or insert (`401 UNAUTHENTICATED` occupies no limiter slot). When a limiter table reaches `max_rate_principals`, an O(1) intrusive-LRU tail check evicts the least-recently-updated bucket if it is idle at full burst tokens (`tokens >= burst`) or idle `> 1h`; otherwise the request returns `429 RATE_LIMITED` with `Retry-After: 1`. When the per-class concurrency gate (`gates[class]`, `max_concurrent = 64`) is full, `httpapi.New` increments `fvpn_control_rate_limited_total{class, limiter="concurrency"}` before writing `429 RATE_LIMITED`.
- **Browser login vs. session buckets (`D-060`, `docs/control/AUTH.md`):** `control.json:login_rate` defaults to `per_second: 1`, `burst: 40` per address/prefix key for `/api/v1/auth/login` and `/api/v1/auth/callback`; `/api/v1/auth/session` and `/api/v1/auth/logout` use the separate `browser_rate` (`session`) bucket (`per_second: 10`, `burst: 30`). After OIDC verification, `login_subject_per_minute` defaults to `10` per canonical `(issuer, client_id, subject)`.
- **Five routes with dedicated rate classes (`#145`, `docs/control/README.md`, `internal/config/config.go`, `internal/httpapi/middleware.go`):** The router matches the loaded PascalCase OpenAPI `operationId` values so high-frequency agent and webhook routes use their own buckets instead of falling into the `agent` (`per_second: 5`, `burst: 10`) or `browser` (`per_second: 10`, `burst: 30`) tag buckets:

| Operation ID | Route | Rate Class & Config Key | Default Rate (`per_second` / `burst`) | Configured Body Limit |
|---|---|---|---|---|
| `EnrollNode` | `POST /api/v1/agent/enrollments` | `anonymous` (`anonymous_rate`) | `2` / `10` (per address / `/64`) | `agent_body_bytes` (`1 MiB`) |
| `ReceiveProviderWebhook` | `POST /api/v1/webhooks/payment/{provider}` | `anonymous` (`anonymous_rate`) | `2` / `10` (per address / `/64`) | `webhook_body_bytes` (`1 MiB`) |
| `SubmitObservation` | `POST /api/v1/agent/observations` | `heartbeat` (`heartbeat_rate`) | `1` / `5` (per node principal) | `agent_body_bytes` (`1 MiB`) |
| `RenewServingLease` | `POST /api/v1/agent/leases/renew` | `lease` (`lease_rate`) | `1` / `5` (per node principal) | `agent_body_bytes` (`1 MiB`) |
| `SubmitUsageBatch` | `POST /api/v1/agent/usage-batches` | `usage` (`usage_rate`) | `2` / `10` (per node principal) | `usage_body_bytes` (`8 MiB`) |

  > **Note on pre-`2026-10-30` measurements (`#145`):** Until `#145` (`2026-10-30`), `classify` matched camelCase operation IDs while the router loaded PascalCase IDs from `contracts/openapi.yaml`, so those five routes shared the `agent` or `browser` tag buckets. Capacity and rate-limit figures recorded before `2026-10-30` for those five streams are superseded.
- **Request-body limits and 4-step check order (`#161`, `#169`, `SECOPS-15` closed, `docs/control/AUTH.md`, `internal/httpapi/middleware.go`, `internal/httpapi/jsondoc.go`):** Every request body is bounded before JSON or OpenAPI validation (`browser_body_bytes = 1 MiB` for `browser`, `agent_body_bytes = 1 MiB` for `agent`/`heartbeat`/`lease`/`EnrollNode`, `webhook_body_bytes = 1 MiB` for `ReceiveProviderWebhook`, and `usage_body_bytes = 8 MiB` for `SubmitUsageBatch` in `docs/control/config.example.json`; bare `Config.ApplyDefaults` uses `64 KiB` browser / `256 KiB` agent & webhook when unset):
  1. **Size (`http.MaxBytesReader`):** A body above the class cap returns `413 PAYLOAD_TOO_LARGE` (`"Request body exceeds the configured limit."`).
  2. **JSON well-formedness (`checkJSONDocument`, skipped for webhooks, `#161`, `#169`):** Bodies up to `canonjson.MaxBytes` (`1 MiB`, the signed-document bound) pass `canonjson.Canonicalize`, and larger bodies in `(1 MiB, class cap]` (`SubmitUsageBatch` above `1 MiB` up to `8 MiB`) pass `checkLargeJSONDocument` (`internal/httpapi/jsondoc.go`, `#169`), a streaming token/syntax walk enforcing the `canonjson` ruleset (valid UTF-8, no UTF-8 BOM, no unescaped ASCII control characters or NUL bytes, no lone UTF-16 surrogates, no duplicate object keys at any nesting level after unescaping, max depth `256`, integers within `±(2^53-1)` with no fraction, exponent, leading zero, or `-0`, and no trailing data); invalid JSON returns `400 MALFORMED_REQUEST` (`"Invalid JSON document."`).
  3. **Schema (`openapi3filter.ValidateRequest`, skipped for webhooks):** Returns `422 VALIDATION_FAILED`.
  4. **Identity (`matchingIdentity` on the parsed body for agent mTLS, including nested usage events, `#169`):** Returns `400 MALFORMED_REQUEST` if `json.Unmarshal` fails and `403 INCARNATION_MISMATCH` on node/incarnation mismatch. (`fvpn_control_rate_limited_total` counts `429` only, not `413` or `400`.)
- **Live measurements, 60-peer admission figures, and gateway notes (`#152`, `#161`, `#165`, `#170`, `#178`, `#182`, [`docs/e2e/at/secops/summary.md`](../e2e/at/secops/summary.md), [`docs/e2e/at/secops/limiter.md`](../e2e/at/secops/limiter.md), [`docs/e2e/at/secops/limiter.json`](../e2e/at/secops/limiter.json), [`docs/e2e/at/secops/capacity.json`](../e2e/at/secops/capacity.json), [`docs/e2e/at/secops/gateway-inventory.json`](../e2e/at/secops/gateway-inventory.json), [`docs/e2e/at/secops/secret-scan.json`](../e2e/at/secops/secret-scan.json), [`docs/e2e/at/secops/requests.md`](../e2e/at/secops/requests.md), [`docs/e2e/scenarios/gateway-principals.md`](../e2e/scenarios/gateway-principals.md), [`docs/e2e/RUN.md`](../e2e/RUN.md)):**
  - **Rate limiters and body bounds (`#152`, `#161`, `#165`, `#178`, `AT28`, `SECOPS-14`, `SECOPS-15`):** `heartbeat` (`SubmitObservation`, burst `5`) and `lease` (`RenewServingLease`, burst `5`) refused at request `6` with `429 RATE_LIMITED` and `Retry-After: 1`; `usage` (`SubmitUsageBatch`, burst `10`), `enrollment` (`EnrollNode`, burst `10`), `webhook` (`ReceiveProviderWebhook`, burst `10`), and `anonymous_shared` (`EnrollNode` + `ReceiveProviderWebhook` from one address) refused at request `11` with `Retry-After: 1`; 20/20 concurrent browser logins from one address completed with `HTTP 200` on `/` (`SECOPS-14` resolved); in security-ops follow-up 6 (`#178`, `docs/e2e/at/secops/limiter.json`), `120` anonymous `POST /api/v1/webhooks/payment/stripe` requests from one client produced `10 × 400` + `110 × 429` (`Retry-After: 1`) while concurrent authenticated admin `GET /api/v1/admin/nodes` and mTLS `RenewServingLease` requests returned `200` (`cross_class_isolation_verified = true` after `#169`); and `SubmitUsageBatch` accepted valid payloads at `1,048,576` B (`1 MiB`), `2,097,152` B (`2 MiB`), `8,388,607` B (`8 MiB - 1`), and `8,388,608` B (`8 MiB`) with `HTTP 200`, rejected `8,388,609` B (`8 MiB + 1`) with `413 PAYLOAD_TOO_LARGE`, rejected malformed JSON with `400 MALFORMED_REQUEST`, and rejected `SubmitObservation` at `1,048,577` B (`1 MiB + 1`) with `413 PAYLOAD_TOO_LARGE` (`SECOPS-15` closed).
  - **`SECOPS-10` 60-peer native admission figures (`#170`, `#178`, `docs/e2e/at/secops/summary.md`, `docs/e2e/at/secops/capacity.json`, `docs/e2e/at/secops/gateway-inventory.json` — shared development host, not candidate hardware):** After devstack follow-up 10 (`#170`) raised lab nginx to `worker_processes 2` × `worker_connections 4096` with a `.128/25` dynamic member address pool, security-ops follow-up 6 (`#178`) measured `20` users × `3` devices = `60` real native NetBird peers (`60/60` `authorized`, `60/60` distinct `/32` `overlay_ipv4`, `node1` `applied_config_generation = "62"`, `device_bindings = 60`, `0` failed operations) on two fresh stacks (`w4-secops` / `30000`) **on the shared development host, not candidate hardware**:
    1. **Sequential `20×3` admission profile (`admission_60_peer`):** total admission wall `904.360 s`, per-peer join `p50 / p95 / p99 / max` = `14,892.523 / 16,451.805 / 18,000.086 / 18,000.086 ms`; `600/600` concurrent `GET /api/v1/member/devices` reads (`20` users × `30` iterations) returned `200` (`0` errors, `0` foreign device IDs) at `p50 / p95 / p99 / max` = `9.616 / 16.413 / 17.969 / 34.084 ms`; gateway socket inventory (`gateway-inventory.json`) recorded `398` established TCP sockets, `441` open FDs (`ulimit -n = 1,048,576`), `time_wait = 0`, and `0` listener drops across `2` nginx workers (`worker_connections = 4096`).
    2. **Concurrent `ScaleNativeMembers` helper profile (`admission_60_peer_scale_helper`, 4-worker NetBird join pool):** total admission wall `300.386 s`; `600/600` concurrent `GET /api/v1/member/devices` reads returned `200` (`0` errors, `0` foreign device IDs) at `p50 / p95 / p99 / max` = `10.721 / 47.604 / 56.136 / 95.367 ms`; gateway socket inventory recorded `385` established TCP sockets, `429` open FDs, and `0` listener drops.
    *(Because these measurements come from the shared development host, not candidate hardware, production candidate hardware certification remains tracked under `§9`.)*
  - **Lab gateway `trusted_proxies`, `X-Forwarded-For`, and secret-scan hygiene (`#170`, `#178`, `#182`, `docs/e2e/scenarios/gateway-principals.md`, `docs/e2e/at/secops/secret-scan.json`, `SECOPS-13`, `SECOPS-16`, `D-066`):** Devstack follow-up 11 (`#182`) configures `trusted_proxies: ["<gateway-ipv4>/32"]` (`172.28.30.21/32` for `e2e`) in `control.json` and `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for` on the lab nginx `:8443` server; `gateway-principals` (`fvpn-e2e-nodeconfig http-burst`) proves that exhausting the `anonymous_rate` bucket (`10 × 400`, then `5 × 429`) from `node1` behind the gateway leaves `node2`'s independent bucket unthrottled (`200`/`400`, `0 × 429`) and fails closed with `429 RATE_LIMITED` (`limiter="address"`) when a trusted peer sends a malformed rightmost-untrusted `X-Forwarded-For` token (`SECOPS-13`). In `SECOPS-16` (`#178`, `docs/e2e/at/secops/secret-scan.json`), `135` retained streams scanned clean (`0` hits), while the `60` transient `/var/run/fvpn-nb-login-*.log` files inside client containers are root-owned `0600` (`42–43` bytes) created via `os.OpenFile(..., 0o600)` before `defer os.Remove(...)` and classified as non-retained under `D-066`.

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
