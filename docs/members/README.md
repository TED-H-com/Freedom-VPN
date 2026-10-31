# Member and billing HTTP/worker integration

The member area implements thirteen strict operations and ten durable worker event
handlers. It uses the real control authentication/CSRF/idempotency boundary,
PostgreSQL transactions, payment inbox and NetBird adapter (§9, §14.3–§14.5,
§15.2, §15.4, §16.1). Integration tests run signed OIDC login through the actual
browser/session stack and validate successful responses against OpenAPI.

## Composition and binary activation

Both binaries use the shared [composition seam](../control/COMPOSITION.md).
`members(),` in each `Domains()` list is the only registration line; domain files
build immutable services from the supplied configuration and shared pool.
Control returns strict handlers plus `httpadapter.RequestContext`, applied inside
the common transport on both API listeners. It retains the original bounded
`Stripe-Signature`; `httpapi.RawBody(ctx)` supplies the original request bytes.
There is no post-construction server wrapper or per-domain call in `main`.

The worker domain returns the ten job handlers, a `WithMembers` context decorator
and a supervised `RunMemberSchedule` runner. Both binaries require valid
`members_config_file` settings and readable dependencies at startup; missing
configuration no longer leaves a serving process with 503 member routes.
Startup logs list domain names and the operations/events they serve.

The node/policy domain must still provide its configured production compiler
through `membersruntime.WithCompiler` in its worker context decorator, and
correlate required `nodes` steps with persisted node observations. Without that
compiler, policy-dependent jobs fail/retry/dead-letter after NetBird sync; billing
and reconciliation work still runs. Never substitute the compiler's TEST ONLY
fallback rates or mark node enforcement complete from compilation alone. The
serving-rights predicate is `entitlement.ListServingMembersForGroups`.
`assignments` is reserved in v1:
no new reader or writer may use it. The authoritative group set is
`entitlements.projection.exit_group_ids`, the sorted union of the entitlement's
pinned plan revision groups and `members.member_group_ids`.

`ListServingMembersForGroups(ctx, q, groupIDs, asOf)` and
`ServingExitGroups(ctx, q, memberID, asOf)` share one predicate: active member,
no member or projected suspension, entitlement state outside
pending_payment/suspended/expired/closed, and `valid_from <= asOf <
effective_service_end`. Grace is already folded into that end. Returned plan
rates come from the pinned entitlement plan, including grant-only members;
a nil rate limit means no additional plan cap, never an invented default.
The readers accept a pool or transaction and return only current serving rights.
The node/desired-state/lease owners must adopt these helpers in their follow-ups.

## HTTP operations

| Operation ID | Contract route | Behavior |
|---|---|---|
| `ListMembers` | `GET /admin/members` | Admin page, entitlement-state filter and filter-bound cursor; usage from the usage service. |
| `GetMember` | `GET /admin/members/{member_id}` | Admin detail, finite grants, subscription reconciliation and strong ETag. |
| `UpdateMember` | `PATCH /admin/members/{member_id}` | If-Match under the member lock; metadata 200, changed service rights 202. |
| `CreateMemberGrant` | `POST /admin/members/{member_id}/grants` | Explicit finite, reasoned grant; 202 operation. |
| `RevokeMemberGrant` | `POST /admin/members/{member_id}/grants/{grant_id}/revoke` | Admin session, CSRF, required reason and `Idempotency-Key`; 202 `member.grant_revoke` receipt. |
| `ListAdminDevices` | `GET /admin/devices` | Admin page of every member device. Opaque cursor, stable ascending (`member_id`, `device_id`), limit default 50 maximum 100. Optional `member_id`, `authorization_unconfirmed`, and `status` select the page and do not change the order. |
| `AuthorizeMemberDevice` | `POST /admin/members/{member_id}/devices/{device_id}/authorize` | Admin session, CSRF, `Idempotency-Key`, no body. 202 `member.device_authorize`. Re-grants a fenced device and authorizes an ordinary pending device. |
| `GetMemberSelf` | `GET /member/me` | Principal-owned account, entitlement, subscription, limits and next actions. |
| `ListMemberDevices` | `GET /member/devices` | Principal-owned devices. Unobserved authorization appears pending. Every item sets `authorization_unconfirmed`, `false` included. |
| `RevokeMemberDevice` | `DELETE /member/devices/{device_id}` | Ownership in SQL, local slot release, tombstone and 202 read-back operation. |
| `CreateCheckoutSession` | `POST /member/checkout-sessions` | Reserve intent, call provider after commit, persist encrypted result and return 201 with checkout_url. |
| `CreateBillingPortalSession` | `POST /member/billing-portal-sessions` | Commit intent first, then provider I/O, then 201 hosted URL. |
| `ReceiveProviderWebhook` | `POST /webhooks/{provider}` | Signature over original bytes, transactional encrypted inbox plus dispatch intent, then 202. |

There are **thirteen** operation IDs in this table: seven Admin operations, five Member
operations and one webhook. Operation polling itself remains owned by the control
framework. Member exits and detailed usage routes remain owned by the node and
usage areas; no additional subscription, device-create or hosted-session GET
route is invented.

All commands reuse `httpapi.Transaction`, record `httpapi.SetResult`, and
implement `Replay`. Standalone service callers still own their transactions.
Replay rereads persisted state under the authenticated principal; it does not
cache a plaintext response. Checkout returns 201 with the hosted URL on the
first successful request and every same-key replay. The post-commit visitor
performs provider I/O outside the command transaction. Failure marks the durable
session reservation and provider operation step failed, with safe step and
top-level errors, and returns retryable 503 DEPENDENCY_UNAVAILABLE. Retrying the
same key retains the session UUID/provider idempotency key, including ambiguous
provider timeouts. Concurrent requests wait briefly outside transactions or
return retryable 503 if the winning call is still outstanding. Even an expired
checkout replay returns the original URL/expiry; a new checkout requires a new
key. No return page or session creation changes service rights.

The framework's durable command reference records the intended 201 before I/O
(to satisfy DB F-06); `billing_sessions.fulfillment_state` records the separate
pending/failed/succeeded effect. A failed effect never emits that 201. This does
not require storing plaintext response bodies or deleting deduplication keys.
Portal failures likewise retain their recoverable intent and stable provider key.

The actor always comes from the authenticated principal. `account_status` is
`active` for an open account and `closed` for a tombstone; payment/suspension
remain in `entitlement.state`. Every outward timestamp is UTC `Z`. Operation
resource types follow the contract (`member_device`, `checkout_session`, etc.),
while internal outbox aggregate names remain stable. Audit resources use the
same typed enum and UUID; operation principals use the authenticated scope form.
Grant operations reference `entitlement_grant:<grant_id>`. NetBird completion
follows those grants through their member; node ACK writers must also recognize
that resource type. Device operations belong to the member, so member retries
can read operations initiated by administrators. Audit retains the actual actor.
A device already at its target state returns its original operation with 202.
Legacy rows without an operation enqueue a fresh observation/withdrawal
operation; 202 never has an empty operation or fabricated completion.
Contracts revision 3 gives device authorization its own
`member.device_authorize` kind, including reconciliation of an existing device.
Grant creation uses `member.grant`; grant revocation uses `member.grant_revoke`.
Operation kinds and resource types are generated enums, validated before writes.
Checkout and portal reservation and successful completion have separate audit
events on `checkout_session:<uuid>`. Completion is recorded in the transaction
that stores the encrypted provider result; retries do not duplicate that event.
Hosted URLs and provider payloads never appear in those audit rows.

## Devices and observed evidence

Native NetBird clients create their peers by SSO; NetBird REST cannot create
WireGuard identities. D-052 selects one OIDC issuer with **public subjects** for
both the portal and NetBird clients (§2 [S36], §15.2). `devices.discover` lists
users through `internal/netbird` (`GET /api/users`), excludes service and blocked
users, and matches the observed user ID to `identity_bindings.subject` under the
central `oidc_issuer`. It uses the same issuer that validates portal tokens;
there is no second issuer setting in the members file.

Discovery writes `netbird_user_bindings` with
`mapping_proof = "shared_issuer_public_subject:" + oidc_issuer`, the verified
identity-binding UUID, and revision 1. The member lock and binding write share
one transaction. Provider reads finish before that transaction begins. A proof
change for the same immutable identity increments the revision; unchanged
discoveries preserve the revision and verification timestamp. A different
member or identity for an existing external ID is an `identity_conflict`, and
the existing row remains untouched. Migration
`00183_identity_bindings_issuer_subject.sql` makes
`(issuer, subject)` unique across client scopes. Historical duplicates stop the
migration with a duplicate-group count; it never merges members. Login must
resolve and lock by the canonical `(issuer, subject)` pair, preserving the
original identity row and member when an OIDC client is rotated. `client_scope`
is diagnostic metadata, not part of identity ownership. Migration
`00186_identity_bindings_client_scope_mutable.sql` makes it mutable so login
stores the newly observed scope; repeated logins from that scope do not append
another `identity_binding.proof_updated` audit. The trigger still forbids
changes to `id`, `member_id`, `issuer` and `subject`. Its Down restores the old
scope immutability guard without rewriting stored values. Both unique
constraints from 00183 and the original schema remain. Ambiguous historical
identities fail closed. Native peers are admitted under the member lock after
the proof exists; the competing final slot is serialized in PostgreSQL.

Pairwise/unknown subjects fail closed as `identity_unmapped`. Never fix them by
matching email or display name; configure public subjects at the shared IdP.
Setup-key/server peers and ephemeral peers are excluded. The device contract
has no unmapped-identity reason field, so a peer without a verified mapping is
excluded from `GET /member/devices`. That endpoint reads worker projections and
does not perform provider I/O. Discovery logs a bounded reason for unmapped and
conflicting identities and increments the worker metric
`fvpn_netbird_identity_discovery_total{result=...}`. Results are `created`,
`unchanged`, `proof_updated`, `identity_unmapped`, `identity_conflict`,
`service_user` or `blocked_user`; each discovery observation counts again.
Subjects, emails, names, issuer URLs and provider IDs are not metric labels or
discovery log fields. Subsequent authorization read-back rechecks the user and
proof; stale or blocked identities cannot grant access. Runtime construction
requires an `*devices.IdentityDiscovery`; an adapter missing it returns an
unmapped observation. Admission, synchronization and adapter reads independently
require exact subject equality, `stable_claim_ref = 'sub'`, `binding_method =
'oidc'`, and the exact shared-issuer proof, with verified timestamps.

Mapping creation, proof updates and rejections also append allowlisted audit
events. Denied device admissions commit `device.authorize` with result `denied`
and one fixed reason: `device_limit_reached`, `entitlement_inactive`, or
`identity_unmapped`. These rows survive the rejected command's rollback; audit
write failures propagate for retry. Known resources retain their real UUIDs;
unknown identities use the fleet-configuration singleton instead of inventing a
member or retaining the external subject. Denials create no operation. Accepted
commands carry their actual operation UUID in the formal audit correlation field.
Member metadata audit records contain changed-field booleans, IDs and state;
notes and display names are excluded. Grant withdrawal records its required
reason as specified by revision 14; other member metadata reasons stay excluded.

### Explicit grant withdrawal (revision 14, AT29)

An administrator calls
`POST /api/v1/admin/members/{member_id}/grants/{grant_id}/revoke` with the admin
session cookie, CSRF token, Origin, a UUID `Idempotency-Key` and
`{"reason":"Operator's withdrawal reason"}`. Members cannot call this route.
There is no grant ETag or If-Match requirement. Missing members, missing grants
and grants belonging to another member return `404 RESOURCE_NOT_FOUND`.
A new command for an already revoked grant or one whose deadline has arrived
returns `409 STATE_CONFLICT`. A same-key, same-payload retry returns the original
operation ID and its current persisted state; a changed payload returns
`409 IDEMPOTENCY_CONFLICT`.

The transaction sets `revoked_at` and `revocation_reason`, recomputes entitlement
using only grants with `revoked_at IS NULL`, and commits the operation, audit and
outbox work together (§9.2, §9.4, §14.5, §15.2). The audit action is
`member.grant_revoke`, resource `entitlement_grant:<grant_id>`, with the admin,
request and operation IDs. Its before/after diff records `revoked_at`, member ID,
grant ID and the supplied reason. Do not put credentials in the reason.

The response is `202` with kind `member.grant_revoke`, resource
`entitlement_grant:<grant_id>` and a Location for the admin operation reader.
Acceptance is not completion. The existing member worker confirms NetBird
withdrawal, then compiles affected node policies; the `nodes` step still needs
node enforcement evidence. Revoking a future grant also queues this read-back
when today's entitlement projection does not change. Devices losing service
move to pending through D-050 / D-081; finite expiry remains a separate AT29
case. Live heartbeat/forwarding timing requires the acceptance lab.

Every grant is finite (D-117); withdrawal of a far-future grant recomputes the
projection like any other grant.

Grant withdrawal and entitlement restoration preserve an existing
`authorization_unconfirmed` fence; neither re-authorizes that device. With
**w3-wire-nodes follow-up 31** merged, the device is neither bound in desired
state nor leased. The worker reconciles every current-incarnation signed
binding through the device-revocation compiler fanout before acknowledging the
withdrawal job, including exits no longer in the member's groups. Overlapping
fence and withdrawal work compiles once; the operation still requires NetBird
read-back and node enforcement evidence. The fence behavior remains D-077.

`devices.NetBirdAdapter` implements the existing narrow `devices.Adapter`.
`EnsureMemberDevice` verifies the actual user, including service/blocked/unsafe
auto-group checks. `EnsurePaidGroup` recovers stable group identities;
`ApplyDeviceAuthorization` removes obsolete owned memberships before additions.
Unmanaged groups and their memberships are preserved. Device revocation removes
all owned access memberships and retains the native peer and durable tombstone;
suspension also withdraws access without deleting identity. `SyncMember` attempts
every device, joins failures, and succeeds only with complete observed evidence.
The adapter hashes the immutable identity-binding UUID and user ID, excluding
the mutable proof revision. Existing revision-dependent hashes are converted
only when `devices.NetBirdBindings.VerifyMemberIdentity` independently proves
the exact immutable database tuple and current revision; a generic store with
an unknown hash continues to fail closed. A changed user/IP mapping is denied and observed withdrawal is recorded, never treated as a grant.

### Quota withdrawals and finite binding history

`Entitlements.Recompute` keeps the oldest authorizations ordered by
`authorized_at ASC, id ASC`. Excess devices become `pending` and each withdrawal
is audited. `entitlement.AuthorizedDeviceIDsSQL` is the shared rank predicate for
reconciliation and NetBird synchronization; desired-state readers must apply it
too so legacy excess rows cannot escape the quota. An entitlement restoration
readmits only previously authorized, non-revoked devices with verified mappings,
in the same order and within the available slots.

Loss of service or a slot clamps open bindings to
`greatest(valid_from + interval '1 microsecond', now)`. Restoration inserts a new
binding revision; historical intervals never reopen. A former IP now owned by
another device is not reclaimed. An exclusion-constraint collision is a state
conflict, never a device-limit error. Existing still-open intervals can be
renewed without rewriting their closed predecessors (§9.4, §15.2).

### NetBird binding and lock-namespace convention

`devices.NetBirdBindings` persists the adapter's opaque keys in
`external_bindings`. All adapter writers for a deployment must use its shared
PostgreSQL session advisory lock (namespace 119). Device synchronization uses a
separate per-device lock. No database transaction spans NetBird HTTP calls.
`SyncDevice` takes that per-device lock with non-blocking
`pg_try_advisory_lock(hashtextextended(device_id, 120))`, and `SyncMember` takes
the same form on `member:<member_id>` before it walks the member's devices, so
`member.entitlement_changed` and `member.suspended` cannot interleave one member.
Losing either lock returns `db.ErrLeaseLost`. An optimistic conflict
(`entitlement.ErrConflict`, including a read-back that does not match the write)
is the same kind of overlap. Both are returned to the dispatcher for retry and
do not call `failOperations`, so the operation's `netbird` step stays `pending`
or `running`. NetBird HTTP errors, invalid state, and database errors other than
that conflict still mark the step `failed` with `DEPENDENCY_UNAVAILABLE`.
Discovery emits `device.authorize` and `identity_binding.rejected` once per
decision transition: the first time an unmapped peer or user is seen, and again
only when the reason changes. The last reason is stored per peer and per
external user in `external_bindings` under kind `discovery_decision`; a later
pass with the same set adds no denial row. Admission replaces that stored
reason, so the same denial audits again if it returns. The counter still
increments on every pass. An unmapped peer has no domain row, so those audits
stay on `fleet_config:00000000-0000-0000-0000-000000000001`.

The reversible key is in `resource_type`; the resource UUID derives from the
JCS SHA-256 of kind/key. External IDs are namespaced by kind. An all-zero stored
hash with NULL observation time represents an **unconfirmed** adapter write;
only actual GET read-back installs a confirmed hash. This storage convention must
be shared with the node/NetBird integration rather than independently recreated.

Device read-back is fenced by both device version and entitlement revision.
Operation steps reference immutable audit evidence. The required `netbird` step
completes after matching read-back; `nodes` remains pending until the node area
records enforcement evidence. Neither a successful PUT nor policy compilation
is reported as complete device enforcement. Native clients, actual gateway
traffic, direct/relay behavior and convergence timing remain unmeasured.

### Restored authorization does not re-grant (D-076)

A row restored from an older backup is not a confirmed current entitlement
(§17.4). `SyncDevice` fences a device when all of the following hold: central
status is still `authorized`, `device_authorization_observations.observed_authorized`
is true, the preflight `ObservePeer` shows the peer withdrawn (not authorized,
or in none of the granted exit groups), and central has no newer
`member.device_revoke` operation and no newer `device.revoke` audit for that
device. A second `ObservePeer` in the same sync must still show the peer
withdrawn. The sync then forces the peer to stay withdrawn, completes without
error, and records the decision once.

The durable mark is `device_authorization_fences` (migration
`00188_device_authorization_fences.sql`), not the observation bit and not an
audit row alone. The observation is updated to the withdrawn read-back so
member sync can finish; the fence row is what stops the next sync from
granting. A later sync does not write the audit again. Devices that were never
observed authorized are first-time grants and are authorized as before.
`revoked` and `closed` devices stay withdrawn and are not labeled as an
external withdrawal. A newer `device.revoke` audit whose status is `revoked`,
or a newer `member.device_revoke` operation, while the device row is still
`authorized`, writes the same fence row and leaves the peer withdrawn, without
`device.authorization.external_withdrawal`. That keeps a later sync from
treating the withdrawn read-back as a fresh grant. A quota withdrawal
(`device.revoke` with status `pending`) does not fence a device that central
has since authorized again.
Entitlement restoration does not delete a fence row.

The audit action is `device.authorization.external_withdrawal` on
`member_device:<device_id>`. The diff carries `peer_id`, `observed_groups`,
`member_id` and `reason=external_withdrawal`. It has no provider subject, token
or secret. Operators read it with the existing audit filters
`resource_type=member_device` and `resource_id=<device_id>`. While
`usage_ledger_state.mode` is `frozen` or `reconciling`, that same decision is
logged as `device authorization unconfirmed during ledger restore` (device,
member, peer, ledger mode) and counted in
`fvpn_device_restore_fence_decisions_total` on the worker registry.

Re-authorization deletes the fence in the same transaction as the grant, marks
the stored read-back unconfirmed, and enqueues `device.authorized` so the next
sync grants. The service command is still `AuthorizeDevice`
(`member.device_authorize`) for a `system` or `admin` actor and a new
idempotency key. Discovery uses that command through `ObserveAndAuthorize`.
Replaying the original idempotency key does not clear the fence. Entitlement
restoration does not delete it. Revocation does: `DELETE /member/devices/{device_id}`
removes the fence row in the same transaction as the revoke.

Operators find fenced devices with `GET /admin/devices?authorization_unconfirmed=true`
instead of querying the audit log. `authorization_unconfirmed` is true exactly
while the fence row exists. On that list, `authorization_unconfirmed_since` is
the row's `fenced_at`, and null when the device is not fenced. A fenced device
reads `status: pending`. `GET /member/devices` sets `authorization_unconfirmed`
on every item, `false` included, from the same fence lookup as the device row.
Re-authorize with `POST /admin/members/{member_id}/devices/{device_id}/authorize`
(admin session, CSRF, new `Idempotency-Key`, no body). It returns 202
`member.device_authorize`. A device that does not belong to the member is 404.
A revoked device is 409 `STATE_CONFLICT`. Authorizing a pending device past the
member's allowance returns 409 `DEVICE_LIMIT_REACHED` and leaves the fence row.
The audit trail stays `device.authorization.external_withdrawal` plus the
existing `device.authorize` audit. A member cannot call the admin route. The
member read-back after a successful re-authorization follows the normal
observe path.

## Events and maintenance

The central dispatcher owns global claims, leases, completion, jittered backoff
and dead-lettering. Area handlers process exactly their referenced work and do
not claim/discard unrelated rows.

| Event | Action |
|---|---|
| `billing.inbox_received` | Process exactly `inbox_id`; fetch authoritative provider state, append evidence and recompute rights. |
| `member.entitlement_changed` | Sync every member device from current state, then compile nodes in previous ∪ current exit groups. |
| `member.grant_revoked` | Observe and compile a grant withdrawal whose entitlement projection did not change; keeps its receipt live after an earlier event for that revision was delivered. |
| `member.suspended` | Idempotently sync current member state and compile affected node policies. |
| `device.authorized`, `device.revoked` | Sync current device state, then compile affected node policies. |
| `netbird.device_sync` | Independently reconcile the referenced device and persist version-fenced observation. |
| `billing.reconcile` | Page known subscriptions into synthetic inbox events; run/page identity deduplicates retries. |
| `members.reconcile` | Sweep finite deadlines, create missing first-login projections and repair member device drift. |
| `devices.discover` | Observe native users, persist D-052 public-subject proofs, then admit only verified, entitled native peers within quota. |

The `member.entitlement_changed` payload's `exit_group_ids` is the sorted union
of previous and current groups, so removed-group nodes receive withdrawal too.
The stored entitlement projection and HTTP response contain only current rights;
consumers must not interpret the event's affected-group set as new access rights.

`RunMemberSchedule` enqueues billing reconciliation and member/device maintenance
at the configured intervals, including at startup. Time-slot and page identities
deduplicate concurrent schedulers. Work is paged at 100 rows. Failed inbox work
uses dispatcher backoff when invoked by its outbox event; standalone `ProcessOne`
retains its bounded retry delay. Unknown event types and unsupported schema
versions are never acknowledged. Legacy `billing.session_requested` rows take
the observable unknown-event dead-letter path (D-038); checkout is synchronous.
Investigate dead letters before explicit retry;
there is no silent reset of failed work.

## Configuration

Set `members_config_file` in the central configuration, or
`FVPN_MEMBERS_CONFIG_FILE`. The path is redacted from effective configuration.
The referenced strict JSON file has these keys; none of its business settings
has a production default. See [config.example.json](config.example.json) for
**TEST ONLY** values and secret-file placeholders.

The existing central `oidc_issuer` is also passed into member identity discovery
by both domain builders through `config.LoadMembersForIssuer`. Configuration
must already be canonical: lowercase scheme/host, no default port, no trailing
slash, no query or fragment. Noncanonical input fails startup with the fixed
`oidc_issuer` setting error. The `internal/members/issuer` package is the single
source for issuer comparison: `issuer.Canonical` normalizes comparisons and
`issuer.Validate` requires canonical configuration. Only literal trailing slashes
are trimmed, and encoded bytes are preserved. Token-validation setup must
import those helpers too (control follow-up 13 owns that integration).
`NewIdentityDiscovery` canonicalizes direct
callers with that helper. Both processes must use the same central setting and
the NetBird deployment must authenticate against that issuer with public
subjects. [central.config.example.json](central.config.example.json) is the
central configuration fragment to combine with the full control/worker example;
it is not a standalone configuration or another issuer setting in the members
JSON file. The IdP's public-subject policy is deployment configuration, not an
email-based linking fallback.

| Keys | Purpose |
|---|---|
| `provider`, `live_mode` | Explicit Stripe choice and environment; sandbox mode is never inferred. |
| `api_key_file`, `webhook_secret_files` | Payment credentials and overlapping webhook verification secrets. |
| `vault_active_key`, `vault_key_files` | Active AES-256-GCM key ID and retained key-ID → file map; raw files are exactly 32 bytes. |
| `price_ids`, `return_urls`, `full_refund_policy` | Allowlisted offered prices, exact HTTPS return URLs and explicit full-refund decision. |
| `device_limit` | Required central setting, no default; immutable plan revisions carry their explicit device limit. Tests use 3. |
| `grace_seconds`, `grace_reason` | Required finite grace policy; explicit zero disables grace. |
| `trial_max_seconds`, `grant_max_seconds` | Required finite grant limits; explicit zero disables trials. |
| `portal_ttl_seconds`, `billing_lease_seconds`, `max_webhook_bytes` | Bounded hosted-session, inbox/session lease and body settings. Align with control request/worker timeouts. |
| `reconcile_seconds`, `device_sync_seconds` | Maintenance enqueue intervals; neither extends service rights. |
| `netbird_url`, `netbird_deployment`, `netbird_token_file`, `overlay_scope` | Central HTTPS management connection and stable deployment/overlay namespace. |

Prices must also be pinned in offered immutable plan revisions created through
the policy area. `price_ids` restricts checkout without inventing prices.
Do not mutate a subscribed revision in place. Secrets, hosted URLs and provider
payloads are excluded from operation/audit/access logs. Raw minimized payment
payloads and hosted responses are encrypted. Payment retention remains a
business/legal decision, not the usage retention default (§16.4).

## Manual work and acceptance boundaries

Shared-issuer/public-subject setup is required by D-052. Once a member has logged
in, discovery produces the NetBird binding; operators do not seed binding rows.
Existing verified login identities are consumed, not manufactured. Conflicting
historical mappings require operator investigation and are never reassigned by
discovery. Plan changes and refunds use
the provider's operator workflow and the policy area's immutable plan revisions;
there are no extra HTTP commands for them. Reconciliation imports authoritative
invoice/refund state. A provider price change outside a pinned subscription
requires operator reconciliation and fails closed.

The task's scenario labels differ from Project Spec v1.0. Automated login/device,
quota/isolation, payment replay and suspension/grace evidence maps to actual
**AT01–AT04**, plus API/security portions of AT28. Actual AT05 covers node
sources; AT08 publication; AT09 NAT/relay/MTU; AT20–AT22 DNS/privacy. These tests
exercise payment/privacy components, not those complete acceptance cases. They
make no claim of live payments, four native platforms, gateway flow termination,
real arm64 execution or DNS acceptance.

## Verification

```sh
make -f mk/billing.mk billing-verify
make -f mk/control.mk control-verify
make go-lint
go test -count=1 ./...
go test -count=1 -tags integration ./...
GOOS=linux GOARCH=arm64 go build ./...
make verify
```

Use disk-backed TMPDIR/GOTMPDIR under `.agent/tmp` (the Make targets do this and
clean up). Integration tests use `dbtest` with `FVPN_TEST_DATABASE_URL`, create
only isolated databases and drop them on cleanup. No Docker, root, host-network
changes or real provider credentials are needed. The HTTP suite uses a signed
OIDC provider, the fake payment provider and an HTTP double with the NetBird
adapter's management response shapes; it never replaces the middleware.
