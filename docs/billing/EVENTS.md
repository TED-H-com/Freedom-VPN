# Member and device policy events

The durable `outbox_events` row is the envelope: `id`, `aggregate_id`,
`aggregate_type`, decimal-string `aggregate_version`, `event_type`,
`schema_version=1`, and the JSON object `payload`. Publication is in the same
transaction as the local change (§9.3, §14.5, §15.5.2–3). Delivery is at least
once. Versions are `pkg/decstr.U64`; consumers must not convert them to floats.
Times are UTC and intervals are half-open.

| Event | Aggregate / version | Payload |
| --- | --- | --- |
| `member.entitlement_changed` | member / entitlement revision | `entitlement.Projection`: `member_id`, `revision`, `computed_at`, `state`, `plan` (revision ID, device limit, exit group IDs), `device_limit`, `exit_group_ids`, `member_group_ids`, nullable `valid_from`, `effective_service_end`, `paid_through`, `grace_until`, `cancel_at_period_end`, `admin_suspended`, `source_refs` |
| `member.suspended` | member / entitlement revision | `member_id`, `revision`, `admin_suspended:true`, nullable `operation_id` |
| `device.authorized` | device / device resource version | `devices.Event`: `member_id`, `device_id`, `device_version`, `entitlement_revision`, `deployment_id`, `peer_id`, `binding_revision`, nullable `operation_id` |
| `device.revoked` | device / device resource version | Same shape as `device.authorized`; the binding interval is already closed |
| `netbird.device_sync` | device / device resource version | Same device event; separate external adapter delivery intent |
| `billing.inbox_received` | billing_inbox / `"1"` | `inbox_id` (UUID); inserted atomically with a new authenticated inbox row |
| `billing.session_requested` (legacy only) | billing session / `"1"` | No longer emitted; old events retain `session_id`, `member_id`, nullable `operation_id`. |

Each *changed* entitlement projection has exactly one
`member.entitlement_changed` row. A repeated computation writes neither a new
revision nor a new policy event. A suspension additionally has a notification
`member.suspended`; it is **not** a second entitlement revision. A unique partial
index protects these event identities. `entitlement_history` preserves the
immutable projection for each revision. Grant source references are grant IDs;
paid and grace sources reference the subscription row, whose snapshot and
append-only billing records retain invoices/refunds and the grace reason.

## Consumer obligations

1. Route by event type and schema version. Do not have an area worker claim and
   discard unrelated global outbox rows. The central dispatcher owns claims via
   `db.ClaimOutbox`, lease renewal/retry orchestration and completion.
2. Compile desired node policy from **current** database state while checking
   the aggregate version. Do not replay an old authorization payload after
   revocation. A member event invalidates every device of that member.
3. Device events reference `device_bindings` for overlay scope, IP and interval.
   Binding revision is independent of device resource version. IP ownership has a finite end, initially the effective service end.
   Recompute extends it on renewal for authorized devices; it never shortens
   historical usage when service rights expire or are refunded. Binding
   ownership alone never grants forwarding. All intervals remain strictly positive. Revocation at the exact authorization
   timestamp closes the binding at the next PostgreSQL microsecond tick; the
   revoked device status immediately denies forwarding independently.
4. `pending_payment`, `past_due`, `expired`, `suspended`, `closed`, absent plans,
   null intervals and intervals outside the current time deny forwarding.
   `active`, `cancel_at_period_end`, and `grace` still require the full deadline
   and device checks. Forwarding expires at the earlier of service end and node
   policy lease expiry (§9.2); node lease time never feeds this engine.
5. Dispatch `netbird.device_sync` to `devices.Service.SyncDevice`. For member
   entitlement events, fan out both `SyncMember` and desired-state compilation
   before acknowledging the event. `member.suspended` can be an informational
   notification because the entitlement event already drives withdrawal.
6. `SyncDevice` serializes by peer with a PostgreSQL **session** advisory lock;
   provider calls run outside a transaction. `SyncMember` takes a non-blocking
   lock on `member:<member_id>` in the same namespace before walking devices.
   A lost lock (`db.ErrLeaseLost`) or optimistic conflict (`entitlement.ErrConflict`,
   including mismatched read-back) is returned for dispatcher retry and does not
   fail the operation step. NetBird HTTP errors, invalid state, and other
   database errors still fail that step with `DEPENDENCY_UNAVAILABLE`. Sync
   rechecks observed identity/IP, writes owned groups idempotently, reads them
   back, and fences evidence by both the device version and entitlement revision.
   Evidence lives in `device_authorization_observations`, separately from desired
   device state, so recording withdrawal after expiry does not reauthorize the
   device or increment its desired resource version. Lost identity mapping
   withdraws authorization. Run periodic `SyncMember` reconciliation to
   repair drift or an ambiguous external timeout. Unmapped-peer denial audits
   are emitted once per decision transition, not once per discovery pass.
7. Device/member operations have required `netbird` and `nodes` steps. These
   services fill only `netbird` after evidence; the node consumer must fill
   `nodes`, attach observed references, and transition to `succeeded` only when
   every required step is done. Superseded generations should be marked
   `superseded`. Do not display an accepted/running operation as completed.
8. Checkout is synchronous (D-031); new commands never enqueue a session job.
   Legacy `billing.session_requested` calls `billing.Service.FulfillSession`; the hosted
   URL stays encrypted and is returned only through the member-scoped result API.
   Checkout operations use the contract kind `checkout.create`. Portal intents
   have no public operation: `CreateBillingPortalSession` completes before 201.

`devices.NetBirdAdapter` now implements the narrow adapter with
`internal/netbird` and the durable `devices.NetBirdBindings` store. It verifies
actual user IDs, preserves unmanaged groups and records independent read-back.
Native SSO creates peers; the worker discovers/adopts them rather than inventing
WireGuard identities through REST. Device tombstones prevent reauthorization.

`billing.inbox_received` calls `ProcessInbox(inbox_id)` for exactly that event,
so the shared dispatcher supplies fenced completion, backoff and dead-lettering.
Standalone `ProcessOne` remains available to library callers, but the registered
jobs do not drain unrelated queues. Periodic `billing.reconcile`,
`members.reconcile` and `devices.discover` intents are enqueued by
`worker.RunMemberSchedule`; pages and scheduler slots have stable dedupe IDs.

Operation and audit resource names use the binding contract
(`member_device`, `checkout_session`, `entitlement_grant`). Internal outbox
aggregate names above intentionally remain unchanged.

See [member integration](../members/README.md) for the registered operation/event
lists, explicit configuration, remaining command-entrypoint hooks and precise
acceptance scope. NetBird read-back does not complete the separate `nodes` step;
real native devices and gateway enforcement remain external acceptance work.
