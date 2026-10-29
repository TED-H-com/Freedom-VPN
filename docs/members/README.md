# Member and billing HTTP/worker integration

The member area implements ten strict operations and ten durable worker event
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
baseline desired-state reader still needs its owner's D-030 follow-up to consume
`entitlement.ListServingMembersForGroups`. `assignments` is reserved in v1:
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
| `GetMemberSelf` | `GET /member/me` | Principal-owned account, entitlement, subscription, limits and next actions. |
| `ListMemberDevices` | `GET /member/devices` | Principal-owned devices; unobserved authorization appears pending. |
| `RevokeMemberDevice` | `DELETE /member/devices/{device_id}` | Ownership in SQL, local slot release, tombstone and 202 read-back operation. |
| `CreateCheckoutSession` | `POST /member/checkout-sessions` | Reserve intent, call provider after commit, persist encrypted result and return 201 with checkout_url. |
| `CreateBillingPortalSession` | `POST /member/billing-portal-sessions` | Commit intent first, then provider I/O, then 201 hosted URL. |
| `ReceiveProviderWebhook` | `POST /webhooks/{provider}` | Signature over original bytes, transactional encrypted inbox plus dispatch intent, then 202. |

There are **ten** operation IDs in this table: four Admin operations, five Member
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
WireGuard identities. Discovery reads peers through `internal/netbird`, requires
an already verified `netbird_user_bindings`/`identity_bindings` proof, and admits
under the member lock. It never links by email, display name or assumed equality
of pairwise subjects. Setup-key/server peers and ephemeral peers are excluded.
The competing final slot is serialized in PostgreSQL.

`devices.NetBirdAdapter` implements the existing narrow `devices.Adapter`.
`EnsureMemberDevice` verifies the actual user, including service/blocked/unsafe
auto-group checks. `EnsurePaidGroup` recovers stable group identities;
`ApplyDeviceAuthorization` removes obsolete owned memberships before additions.
Unmanaged groups and their memberships are preserved. Device revocation removes
all owned access memberships and retains the native peer and durable tombstone;
suspension also withdraws access without deleting identity. A changed user/IP
mapping is denied and observed withdrawal is recorded, never treated as a grant.

### NetBird binding and lock-namespace convention

`devices.NetBirdBindings` persists the adapter's opaque keys in
`external_bindings`. All adapter writers for a deployment must use its shared
PostgreSQL session advisory lock (namespace 119). Device synchronization uses a
separate per-device lock. No database transaction spans NetBird HTTP calls.
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

## Events and maintenance

The central dispatcher owns global claims, leases, completion, jittered backoff
and dead-lettering. Area handlers process exactly their referenced work and do
not claim/discard unrelated rows.

| Event | Action |
|---|---|
| `billing.session_requested` | Legacy events only; no new checkout or portal request emits this event. The existing unowned worker registration remains a recovery compatibility path pending removal by its owner. |
| `billing.inbox_received` | Process exactly `inbox_id`; fetch authoritative provider state, append evidence and recompute rights. |
| `member.entitlement_changed` | Sync every member device from current state, then compile affected node policies. |
| `member.suspended` | Idempotently sync current member state and compile affected node policies. |
| `device.authorized`, `device.revoked` | Sync current device state, then compile affected node policies. |
| `netbird.device_sync` | Independently reconcile the referenced device and persist version-fenced observation. |
| `billing.reconcile` | Page known subscriptions into synthetic inbox events; run/page identity deduplicates retries. |
| `members.reconcile` | Sweep finite deadlines, create missing first-login projections and repair member device drift. |
| `devices.discover` | Observe native peers and admit only verified, entitled identities within quota. |

`RunMemberSchedule` enqueues billing reconciliation and member/device maintenance
at the configured intervals, including at startup. Time-slot and page identities
deduplicate concurrent schedulers. Work is paged at 100 rows. Failed inbox work
uses dispatcher backoff when invoked by its outbox event; standalone `ProcessOne`
retains its bounded retry delay. Unknown event types and unsupported schema
versions are never acknowledged. Investigate dead letters before explicit retry;
there is no silent reset of failed work.

## Configuration

Set `members_config_file` in the central configuration, or
`FVPN_MEMBERS_CONFIG_FILE`. The path is redacted from effective configuration.
The referenced strict JSON file has these keys; none of its business settings
has a production default. See [config.example.json](config.example.json) for
**TEST ONLY** values and secret-file placeholders.

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

Shared-issuer setup and an independent cross-client identity proof remain
required before inserting a verified NetBird user binding. Existing identity
rows are consumed, not inferred or manufactured. Plan changes and refunds use
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
