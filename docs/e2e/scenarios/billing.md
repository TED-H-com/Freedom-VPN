# Provider truth and billing return — AT03/AT04 groundwork

The TEST ONLY provider runs at `https://api.stripe.com` inside the lab. Host
harnesses map that hostname to their assigned payment port and trust `ca.pem`.
Stripe-compatible reads require `Authorization: Bearer <stripe.key>`.
Mutation hooks instead require `X-E2E-Secret: <payment.secret>`. Both secrets
are generated under the prepared state directory; never record either.
All responses disable caching. CI does not contact Stripe or another public
provider.

1. Create and activate a valid plan through the policy API, with an existing
   API-created group, Fair-Use policy and DNS profile. Log in as a fresh member.
2. Call `POST /api/v1/member/checkout-sessions`. Retrying the same idempotency
   key preserves the provider session; changing its form returns a conflict.
   Read `billing_sessions.external_session_id` for that returned checkout ID
   using a read-only query, or inspect the provider's checkout readback.
3. Supply truth with `POST /e2e/snapshot`:

   ```json
   {
     "checkout_id": "<actual provider checkout id>",
     "period_start": 1793325600,
     "period_end": 1793329200,
     "amount_minor": 1000,
     "currency": "USD",
     "status": "active",
     "paid": true,
     "cancel_at_period_end": false,
     "refund_minor": 0
   }
   ```

   Use current TEST ONLY period bounds. An unknown checkout returns 404. The
   first snapshot fixes the invoice's period, amount and currency. Its
   subscription inherits the actual checkout metadata, price and customer.
   Subsequent snapshots may set cancellation, `status=canceled`, and increase
   cumulative `refund_minor` up to the invoice amount. Each increment creates
   a distinct immutable refund; earlier refunds never change or disappear.
4. `POST /e2e/event` with `checkout_id` and `type` returns `body` (a JSON string)
   and `signature`. Send the **exact string bytes** to
   `POST /api/v1/webhooks/stripe`, with `Stripe-Signature` and JSON content type.
   Supported events: `checkout.session.completed`, `invoice.paid`,
   `customer.subscription.updated`, `customer.subscription.deleted`,
   `charge.refunded`. Hook response bodies/signatures are secrets and are not
   evidence. To replay an event, retain and resend the same returned bytes.
5. Poll the product projection, not just webhook HTTP 202. The scenario checks
   a reconciled subscription with the exact paid interval, one immutable paid
   invoice and an active entitlement. A partial refund retains that interval;
   cancellation is projected; a full refund removes it under the lab's explicit
   `revoke_original_period` setting. Duplicate deliveries must not duplicate
   invoices/refunds. Each phase has a 90-second bound.

The provider serves subscription detail/list, paid invoice/list and line
items, invoice-payment linkage, and refunds over the same authenticated API
used by the production worker. Unit tests use the production Stripe adapter
for every read and webhook signature verification.

The gateway serves the built Portal. `/billing/return` redirects to
`/member/billing?checkout=return`; a Playwright probe checks the actual billing
page through the member session without saving cookies, screenshots or traces.
A returned page does not grant service. See [shared hooks](../RUN.md) and
[latest billing record](../runs/2026-10-30/billing.md) for observed reconciliation
or its exact product prerequisite. No subscription or entitlement is seeded.
