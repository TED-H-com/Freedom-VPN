# Member and billing HTTP/worker integration

The member area implements ten strict operations and ten durable worker event
handlers. It uses the real control authentication/CSRF/idempotency boundary,
PostgreSQL transactions, payment inbox and NetBird adapter (§9, §14.3–§14.5,
§15.2, §15.4, §16.1). Integration tests run signed OIDC login through the actual
browser/session stack and validate successful responses against OpenAPI.

## Composition and remaining startup integration

`apps/control/handlers.go` registers `memberHandlers(pool)`; the implementation is
`internal/members/httpadapter`. `apps/worker/jobs.go` registers the area through
`registerMemberJobs`. Dependencies are immutable and passed explicitly, without
process-global service registries or implicit business defaults.

**The current binaries still need these startup hooks from the orchestrator.**
The task's allowed composition points do not carry configuration, request headers
or worker dependencies. The supplied constructors/wrappers are complete and are
exercised together by integration tests, but the existing command entrypoints do
not call them:

1. Control: after `control.Servers(...)` and before `control.Run(...)`, call
   `control.ConfigureMembers(c, pool, public)`. It loads `members_config_file`,
   constructs the services and wraps the public handler to retain the original
   `Stripe-Signature` header. `httpapi.RawBody(ctx)` still supplies the bounded,
   original body. The wrapper does not replace authentication or validation.
2. Worker: load `config.LoadMembers(c.MembersConfigFile)`, construct
   `membersruntime.New(pool, settings, compiler)` with the node team's configured
   desired-state compiler, and run the existing dispatcher with
   `worker.WithMembers(ctx, services)`. Run
   `worker.RunMemberSchedule(ctx, services)` alongside it; cancel and join both on
   shutdown. Treat a scheduler error as a process error. Use the existing bounded
   shutdown context.
3. The node integration must supply its production entitlement/FUP reader and
   correlate required `nodes` steps with persisted node observations. The
   baseline desired-state DB reader uses `assignments`, whereas member rights
   here use the entitlement's pinned plan and exit groups. Coordinate this
   boundary; do not substitute the compiler's TEST ONLY fallback rates or mark
   node enforcement complete from compilation alone.

Without the control wrapper, the ten registered methods fail with safe 503
responses; they do not retain the transitional 501. Without worker dependencies,
registered jobs fail and the dispatcher retries/dead-letters them. Configure the
hooks before starting workers against production outbox work. A missing compiler
also prevents acknowledgement of member/device policy work after NetBird sync.

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
| `CreateCheckoutSession` | `POST /member/checkout-sessions` | Committed intent and 202 operation; worker creates the hosted session. |
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
cache a plaintext response. For checkout, repeat the same POST, body and
Idempotency-Key after polling its operation: once fulfilled, replay returns the
contract's 201 `CheckoutSession`. Before fulfillment it remains 202. A portal
provider failure returns 503 after its recoverable intent commits; the same key
recovers the same provider session.

The actor always comes from the authenticated principal. `account_status` is
`active` for an open account and `closed` for a tombstone; payment/suspension
remain in `entitlement.state`. Every outward timestamp is UTC `Z`. Operation
resource types follow the contract (`member_device`, `checkout_session`, etc.),
while internal outbox aggregate names remain stable.

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
| `billing.session_requested` | Fulfill the persisted session using its UUID as provider idempotency key; encrypted result is the receipt. |
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
GOARCH=arm64 go build ./...
```

Use disk-backed TMPDIR/GOTMPDIR under `.agent/tmp` (the Make targets do this and
clean up). Integration tests use `dbtest` with `FVPN_TEST_DATABASE_URL`, create
only isolated databases and drop them on cleanup. No Docker, root, host-network
changes or real provider credentials are needed. The HTTP suite uses a signed
OIDC provider, the fake payment provider and an HTTP double with the NetBird
adapter's management response shapes; it never replaces the middleware.
