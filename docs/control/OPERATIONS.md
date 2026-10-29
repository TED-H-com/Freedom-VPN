# Operations and audit

`operations.Create` belongs in the transaction that writes desired state and its
outbox (§14.5, §15.5). Supply a central request ID, principal, resource reference,
optional decimal-string resource version/desired generation and 1–64 named steps.
At least one step must be required; optional-only work is rejected. Operations
are inserted as pending, and required steps start pending. Every omitted external
effect must instead be an explicit `Required:false` step with a reason; it becomes `not_required`.

`operations.Transition` row-locks the operation and checks step transitions.
Success requires a stored evidence reference such as `observation:<uuid>` plus its
observation time. The domain must first persist verified read-back evidence; a
successful outbound request or a newly created outbox record is not read-back.
The database independently enforces required-step evidence. Observed generation
cannot decrease or exceed this operation's desired generation. Finished successful
or superseded operations cannot be changed, including their evidence and metadata.
All operation rows are protected against DELETE/TRUNCATE (DB F-10).

Status is derived from required steps: all pending → pending; started/incomplete
→ running; all successful → succeeded; terminal failures with successful steps →
partially_succeeded; terminal failures without successes → failed; terminal
cancellations → superseded. Failed steps may retry through running; a successful
step is not retried. Explicit not-required steps never block completion. Desired
and observed generations stay distinct. `Accepted` supplies the generated 202
response's Location header. UI must show accepted work as waiting for application,
never completed (§18.3).

Admin can read any operation; Member queries include the principal scope in SQL.
Another member's operation returns 404, just like a nonexistent operation.
The response exposes step state and observation time but not raw evidence payloads.
There are no legacy operations requiring request-ID backfill; every new operation
requires the central request UUID. `resource_version` on an operation is a snapshot
of the target resource's version, not an operation revision. Step transitions leave
that snapshot unchanged. Mutable domain resources must increment their own version
on every UPDATE, including metadata changes (DB F-14, §15.1).

`audit.Append` wraps `db.AppendAudit` within the caller's transaction. Supply the
actor UUID/type, dotted action, `resource_type:uuid`, request/operation IDs and
result. Its diff includes only explicitly allowlisted changed fields, with recursive
secret-name redaction. Never allowlist raw provider payloads, browser history or
unbounded free text. Audit is append-only under the existing DB guards.

The Admin audit endpoint supports resource type/ID, actor ID, request ID and
half-open time filters. Pages default to 50 and cap at 100. Ordering is descending
`(occurred_at,id)`; a versioned opaque binary cursor encodes the last tuple and a
hash of its filters, preventing accidental reuse with a changed query. Cursors
are pagination hints, not authorization credentials; authorization runs on every
request. Tied timestamps cannot omit or duplicate rows. New rows appear on a fresh
first page. Rows with malformed contract representations are omitted without
inventing replacement identities or rewriting history; the producer owns the fix.
The reader validates actor/resource types, UUID references, action, result, timestamp
range and diff shape. Each omission emits a safe `audit row omitted` structured log with the row
ID, fixed reason and process-local `audit_anomalies_total` encounter count. The
same count is available through `(*audit.Store).AnomalyCount()`; repeated reads
count repeated encounters, and a process restart resets it. Malformed values and
diffs never enter anomaly logs. The control logger is passed into the audit store.

The limit bounds scanned rows, so a page may be short or empty when rows are
malformed. The cursor advances past all scanned rows, including omitted rows.
Clients must continue while `next_cursor` is present, even on an empty page. A
malformed row cannot make the following valid rows unreachable. Version 2 cursors
preserve PostgreSQL timestamps, including infinities, to advance past invalid dates. Database query
or transport failures still fail the request through the generic error envelope.
