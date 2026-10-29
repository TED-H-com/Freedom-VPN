# Provider setup

## Local fake provider

`billing.NewFakeProvider()` creates an in-memory sandbox provider with a fresh
random webhook secret. It has no network/card-processing capability. Feed it
`Subscription` and `Invoice` fixtures with `SetState`; sign serialized `Event`
bytes using `Sign`, then pass the exact bytes to `Service.Ingest`. `ProcessOne`
reads the current state set on the fake, independently of the event's label.
SetState copies slices so the caller cannot mutate concurrent provider reads.
Fake checkout/portal links use `https://payments.example.invalid/` and are
**TEST ONLY**; they are not functional payment pages.

Use `entitlement.TestConfig()` and `billing.TestConfig()` only in development.
Create a fresh 32-byte inbox/session encryption key at runtime, keep it outside
source control, and construct `billing.NewVault`. Seed a plan with explicit test
price/group/FUP/DNS references; activate it through the plan API. These Go APIs
are ready for the control/worker composition task; no extra executable or HTTP
endpoint is introduced by this worktree.

## Stripe candidate adapter

Stripe remains a provider candidate, not a product selection. `billing.Provider`
is the integration boundary; a second adapter does not change entitlement logic.
This implementation uses a typed standard-library `net/http` client, without a
Stripe SDK or any additional module dependency.

Configuration:

| Setting | Requirement |
| --- | --- |
| `StripeConfig.APIKeyFile` | Service-readable file containing the sandbox API key; never a committed example key |
| `WebhookSecretFiles` | One or more service-readable endpoint signing-secret files; overlap during rotation |
| `LiveMode` | `false` for sandbox; signed events from the other mode are rejected |
| `PortalSessionTTL` | Explicit local URL presentation lifetime, positive and at most five minutes; Stripe does not expose an expiry field |
| `Client` | Optional HTTP client/transport; timeout defaults to 15 seconds and redirects are disabled |
| Vault active ID and key files | Separate AES-256 keys for encrypted inbox and hosted session data; no signing/API secret is stored in PostgreSQL |
| Refund/grace/trial settings | Approved production values required; sample constructors are TEST ONLY |
| Return URLs | Exact HTTPS URLs, for example `https://portal.example.invalid/billing/return`; no prefix/suffix matching |

Pin the account webhook destination to **snapshot** events and
`2026-09-30.endive`, matching the client's `Stripe-Version`. Subscribe to
`customer.subscription.*`, `invoice.paid`, `invoice.payment_failed`,
`checkout.session.completed`, and `charge.refunded`. Do not configure payment
method expansions. Irrelevant authenticated events are durably acknowledged
without granting access. Unsupported relevant API versions fail closed.

Use a single recurring price, one subscription item, quantity one. Disable portal
plan changes/prorations and multi-product invoices until explicit allocation and
revision-transition rules are approved. The adapter rejects mixed/prorated lines
and shared invoice payments instead of guessing the service period or refund
allocation. It follows pagination for invoices, lines, invoice payments and
refunds; a 100-page safety bound fails the sync instead of silently truncating it.
It reads service periods from the recurring invoice line, not the subscription's
current period or invoice creation time. Refund state comes from current invoice
payments and refund listings. Periodic reconciliation covers later refunds even
when the corresponding webhook is lost.

Checkout creates hosted sessions with server-owned `fvpn_checkout_id` subscription
metadata. A GET read-back confirms the session object before completion. Portal
session creation has no retrieval endpoint; its authenticated create response is
the provider evidence. Both reuse stable request UUIDs as Stripe idempotency keys.
A return URL is UI navigation only. Create test payments exclusively through
Stripe's hosted sandbox UI; do not send card fields to Freedom VPN.

The raw webhook HMAC (`timestamp + '.' + raw body`, SHA-256) is Stripe's protocol,
not the platform Ed25519 envelope protocol. Verification accepts configured
rotation secrets and multiple v1 signatures, uses constant-time comparison, and
rejects duplicate timestamps or timestamps more than five minutes in the past or
future. Synchronize the control host's clock. Never disable timestamp checking.

Provider response bodies, request URLs and authentication headers are never
included in returned errors. The worker stores sanitized subscription references,
periods and amounts; raw authenticated webhook bodies are encrypted at rest.
Unexpected card/secret-bearing fields are rejected before persistence. Configure
provider data minimization accordingly and treat rejected events as an operator
alert; reconciliation remains available for current payment state.

Real Stripe sandbox and native-client/NetBird runs are **待實測** because credentials
and the central adapter are not available in this task. `httptest` exercises the
pinned protocol against typed local fixtures; it is not evidence of a live API run.

## Primary references

Implementation consulted these current Stripe sources (2026-10-29):

- [API versioning](https://docs.stripe.com/api/versioning): stable version and explicit request/webhook pinning.
- [Webhook delivery and signature verification](https://docs.stripe.com/webhooks): raw body, signature/timestamp verification, retries and ordering (spec S19).
- [Subscription webhooks](https://docs.stripe.com/billing/subscriptions/webhooks): current subscription/invoice handling (spec S18).
- [Subscription object](https://docs.stripe.com/api/subscriptions/object): current subscription item shape.
- [Invoice line item](https://docs.stripe.com/api/invoice-line-item/object): service interval, price and subscription-item parent.
- [Invoice payments](https://docs.stripe.com/api/invoice-payment/list): paginated invoice payment references.
- [Refund listing](https://docs.stripe.com/api/refunds/list): current refunds scoped by payment intent/charge.
- [Checkout creation](https://docs.stripe.com/api/checkout/sessions/create): hosted checkout parameters and idempotent session creation.
- [Supported currencies](https://docs.stripe.com/currencies): currency-dependent minor units.
