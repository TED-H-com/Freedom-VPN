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
| `billing.session_requested` | billing session / `"1"` | `session_id`, `member_id`, nullable `operation_id` |

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
   Binding revision is independent of device resource version. IP ownership is
   open until revoke; PostgreSQL infinity on a binding is historical ownership,
   **never** an entitlement or lease deadline. Empty ownership intervals support
   authorization followed by revocation at the same timestamp.
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
   provider calls run outside a transaction. It rechecks observed identity/IP,
   writes owned groups idempotently, reads them back, and fences evidence by
   both the device version and entitlement revision. Lost identity mapping
   withdraws authorization. Failed/mismatched read-back is retryable and does
   not mark the operation complete. Run periodic `SyncMember` reconciliation to
   repair drift or an ambiguous external timeout.
7. Device/member operations have required `netbird` and `nodes` steps. These
   services fill only `netbird` after evidence; the node consumer must fill
   `nodes`, attach observed references, and transition to `succeeded` only when
   every required step is done. Superseded generations should be marked
   `superseded`. Do not display an accepted/running operation as completed.
8. `billing.session_requested` calls `billing.Service.FulfillSession`; the hosted
   URL stays encrypted and is returned only through the member-scoped result API.
   Checkout operations use the contract kind `checkout.create`. Portal intents
   have no public operation: `CreateBillingPortalSession` completes before 201.

`internal/netbird` is absent from this worktree's baseline. Implement the narrow
`devices.Adapter` using that task's observed peer/user IDs and idempotent managed
group operations. `ObservePeer.Groups` must contain only platform-owned access
groups; preserve unrelated groups. Never map identities by email. Real NetBird
and node read-back evidence remain integration work; fake-adapter success is not
network enforcement evidence.
