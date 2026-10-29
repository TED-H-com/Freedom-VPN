# Members, entitlement, devices and billing

These packages implement the central Go services for §2, §9, §14.2–5, §15.1–2,
§15.5.2–3, §16.1–2 and §17.4.2. HTTP/session and worker adapters are described in
[the member integration guide](../members/README.md); that guide lists the
remaining binary startup hooks and node-enforcement boundary. No provider credentials, card fields or hosted checkout URLs
are emitted to logs; the packages do not install a logger.

## Entitlement state machine

`entitlement.Compute` is a pure function of explicit time, member state,
suspension, a plan revision, subscription facts and finite grants. `Store.Recompute`
locks the member, loads the sources, compares canonical JSON, and writes the
projection, immutable history, audit and exactly one policy event only when the
result changed. The caller owns the transaction. All source writers use the
same member lock. Member status is the coarse projection (`active` includes
period-end cancellation/grace); the detailed state remains in `entitlements`.
Every changed entitlement projection increments the member ETag version. Time-only changes are processed
by `Store.Sweep` using persisted `next_transition_at` deadlines.

| Input / transition | Result and service behavior |
| --- | --- |
| Login or checkout without paid invoice/grant | `pending_payment`, no service interval |
| Current paid service period | `active`, finite invoice-derived deadline |
| Period-end cancellation before deadline | `cancel_at_period_end`, paid rights remain |
| Cancellation reaches deadline | `expired`, withdraw all peers/node allow sets |
| `past_due` without a usable paid period/grace | `past_due`, deny |
| Approved grace after paid period | `grace` until the separately recorded end |
| Grace reaches its end | `expired`, deny |
| Finite manual/trial/compensation/migration grant | `active` during its interval, preserves billing facts |
| Full refund | Apply the configured policy to that invoice's original interval |
| Partial refund | Record it; do not derive or subtract service time |
| Manual suspension, including later successful payment | `suspended`; payment cannot clear the manual decision |
| Resume | Recompute from current facts; resume itself never grants time |
| Close | `closed`; tombstone account/devices and close their IP bindings |

Overlapping or adjacent periods of the same plan may form a continuous service
interval. Disconnected future periods never bridge a gap. Different simultaneously
active plan revisions are rejected as ambiguous rather than combined into wider
rights. A grant with no current plan needs an explicit `plan_revision_id`.
There is no automatic trial from a provider's `trialing` status; zero-value
trial invoices are recorded as `trial_invoice` and do not create paid entitlement. Trial entitlement
requires a finite reasoned grant. Existing authorized devices continue to count
against the limit while their member is suspended or expired; device revocation
releases a slot immediately in the local transaction.

`entitlement.Config` is mandatory: grace duration/reason, maximum grant/trial
length and configurable device default. `entitlement.TestConfig()` is **TEST ONLY**
(72-hour grace, seven-day trial maximum, 366-day grant maximum, three devices).
These are **待決策**, except the recommended configurable device default of 3.
Production must supply approved values. Zero grace/trial durations disable them.
No arbitrary far-future grant, permanent grace or node lease can create service
rights. Changing limits requires reviewing already issued grants/grace periods.

## Payment pipeline and recovery

1. Verify the provider signature over the exact raw body before parsing or
   persisting anything. Remove card/payment-method metadata before storage; reject unexpected full
   card inputs or credentials. Configure only the required event types and do not
   expand payment methods into webhooks.
2. Encrypt the authenticated, minimized payload with AES-256-GCM. Raw bytes
   are retained unchanged when no card metadata needs removal. The encrypted envelope has
   a key ID, random nonce and ciphertext; provider/event identity is authenticated
   associated data. `UNIQUE(source,external_event_id)` is the durable dedupe key.
   Only return 2xx after the insert commits. A conflicting body under the same
   event ID fails with idempotency conflict. The digest always covers the original authenticated raw bytes through a canonical
   JSON base64 envelope, accommodating provider decimals without float parsing.
3. `ProcessOne` claims pending/failed work with a recoverable lease, then fetches
   the provider's **current** subscription and invoice/refund state. Webhook event
   type and `created` never select a newer state or add a fixed number of days.
4. A second lease serializes reads for each subscription. A stale worker cannot
   commit after lease expiry or replacement. No provider call runs in a database
   transaction. In one transaction, lock the member, verify the lease, append
   billing evidence, rebuild the paid intervals, update the subscription,
   recompute entitlement, append audit and enqueue changed policy.
5. Paid invoice and refund references are unique permanently. Retries never add
   the same period twice. Existing recorded paid evidence is retained when a
   late event or a provider listing omits a newer invoice. Conflicting immutable
   invoice facts fail closed. Reconciliation anomalies are sanitized codes.
6. Failed inbox work retries after a bounded delay; expired claims are reclaimable.
   The host worker should apply backoff to provider errors, drain available work,
   and schedule periodic reconciliation/sweeps. `Reconcile(runID,after,limit)`
   pages through known subscriptions and durably queues synthetic events. Reusing
   a run ID deduplicates scheduling. Canceled subscriptions are included so later
   refunds can be found. Successful processing sets `last_reconciled_at`.

`Vault` supports overlapping key IDs for rotation. `NewVaultFromFile` reads a
32-byte raw key from a service-readable file. Keep old keys until ciphertext
retention expires; back up keys independently from the database. Key and file
errors omit file paths and payloads. Never put a key in a URL or environment log.
No billing retention period is invented here: it remains a production decision.

New subscriptions bind through `fvpn_checkout_id` metadata set by the server,
then validated against the stored checkout intent, pinned plan/price and central
customer mapping. Unknown subscriptions without this proof stay unprovisioned.
`billing_customers` is unique by member/provider and provider/customer. Portal
session creation derives the customer exclusively from this table. Return URLs
match an explicit HTTPS URL allowlist exactly; return pages only poll the central
state and cannot create billing evidence or entitlement.

Checkout intents and their operation references commit before calling the provider;
new sessions do not emit an outbox event. The response is synchronous 201 only
after the encrypted hosted result is persisted.
A stable UUID is also the provider idempotency key. Hosted responses are encrypted.
Ambiguous requests older than 23 hours are refused for operator reconciliation,
so a provider-pruned key cannot create an accidental second session. API adapters
must send `Cache-Control: no-store` for member/billing/secret-bearing responses.

## Refund policy — 待決策

`billing.Config.FullRefundPolicy` is required. `revoke_original_period` removes a
fully refunded invoice's original service interval; independent payments/grants
remain. `keep_original_period` retains it. `billing.TestConfig()` chooses the first
policy **TEST ONLY**. Cumulative successful partial refunds can reach a full refund;
pending/failed refunds do not revoke. Individual refund amounts and original
invoice references are append-only. Partial refunds never imply hours/days.
`paid_through` stays the historical paid billing fact even after a full refund.

Amounts use integer `amount_minor` plus currency. `CurrencyDigits` is an explicit
supported-currency table and rejects unknown currencies; it handles zero-, two-
and three-decimal units without floating point. Extending the table requires a
review of provider currency rules and applicable amount limits. Prices and FUP
thresholds/rates have no production defaults in these packages.

## Service integration

- Construct `Actor` from verified central identity; never from request JSON.
  The control framework enforces OIDC audience/issuer, MFA, CSRF and request IDs.
- `members.Service`: `List`, `Get`, `Patch`, `Close`, `Grant`, `RevokeGrant`.
  PATCH takes the generated `api.MemberPatch`, an explicit resource version and
  command key. Validate unknown JSON fields at the HTTP boundary. Suspension
  requires a reason. Grant source and end are mandatory. Internal close/revoke
  APIs do not introduce extra contract HTTP endpoints.
- `devices.Service`: `ObserveAndAuthorize`, `AuthorizeDevice` (trusted observed
  peer only), member-scoped `List`/`Get`, `RevokeDevice`, `SyncDevice`/`SyncMember`.
  Row locking makes the last-slot race atomic; unmapped/setup-key peers cannot
  get member rights. Changed peer ownership/IP is denied pending explicit rebinding.
- `billing.CreatePlanRevision` takes generated `api.PlanPayload` and caller-owned
  definition/revision IDs in the policy store's transaction. It validates group,
  FUP and DNS references, and never edits old revisions/subscribers. Activate with
  the definition version. `ReadPlan`/`ListPlans` expose immutable/offered data.
- `billing.Service`: durable `Checkout`, `CreateBillingPortalSession`, scoped
  `SessionResult`, `Ingest`, `ProcessOne`, `Reconcile`, `FulfillSession`.
- Map domain errors via `pkg/apierr.Write`: invalid signature 401 `WEBHOOK_SIGNATURE_INVALID` (the binding OpenAPI
  contract overrides the task prose's 400); invalid payload 422 as required by the route; missing version 428; version conflict 412; foreign IDs
  404; device limit 409 `DEVICE_LIMIT_REACHED`; unavailable entitlement 403
  `FORBIDDEN`; idempotency conflict 409; provider errors 503. Never
  expose raw database/constraint/provider errors. Strict handlers now live in `internal/members/httpadapter`; see the member
  integration guide for composition and replay.
- Dispatch policy work as described in [EVENTS.md](EVENTS.md). Expiry and revocation
  of existing TCP/UDP flows require the node consumer; these central tests do not
  claim real gateway enforcement or convergence timing.

## Verification and acceptance scope

Run `make -f mk/billing.mk billing-verify`. Integration tests require
`FVPN_TEST_DATABASE_URL` and create/drop isolated databases using `dbtest`.
All test scratch is private disk storage under `.agent/tmp`; no tmpfs or host
network changes are used. No container is required when the shared test DSN exists.

| Acceptance | Automated evidence here | Evidence owned elsewhere / still required |
| --- | --- | --- |
| AT01 / ID01 / MEM01 | Verified cross-client binding, unmapped/setup-key rejection, member-scoped reads | Four real native clients, common-issuer setup, actual exit selection and controlled traffic |
| AT02 / MEM02 / MEM03 | Two simultaneous requests/one slot, exactly one winner; foreign read/revoke 404; revoke frees slot; adapter read-back failure does not complete | Gateway rejects loser; real NetBird adapter wiring |
| AT03 / BILL01 / BILL03 / ENT01 | Ten duplicate deliveries; all six event orders; forged return cannot grant; invalid signature no insert; late failure cannot regress renewal; encrypted inbox; missed-webhook repair | Stripe sandbox credentials and actual gateway authorization |
| AT04 / BILL02 / ENT02 / MEM03 | Cancel until deadline; full/partial refunds; renewal; suspend followed by paid; finite grace expiry; one changed projection/one policy event; rollback atomicity | Existing TCP/UDP flow termination on all exits, convergence timing and portal rendering |

No real payment or NetBird credentials are needed for the fake-provider tests.
See [SANDBOX.md](SANDBOX.md) for provider configuration and current Stripe sources.


D-029–D-031 review integration: see [member composition](../members/README.md)
for the shared serving readers, reserved assignments table, typed operation/audit
resources, and synchronous checkout. `Checkout` completes provider work before
returning its populated URL; HTTP uses `ReserveCheckout` inside its transaction
and `CompleteSession` only after commit. Migration 00121 adds the effect state
for durable failed reservations. Provider errors are persisted only as fixed
safe codes/messages; same-key retries preserve the provider idempotency key.
