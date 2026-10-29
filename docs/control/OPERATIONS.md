# Operations and audit

`operations.Create` belongs in the transaction that writes desired state and its
outbox (§14.5, §15.5). Supply a central request ID, principal, resource reference,
optional decimal-string resource version/desired generation and 1–64 named steps.
Required steps start pending. Every omitted external effect must instead be an
explicit `Required:false` step with a reason; it becomes `not_required`.

`operations.Transition` row-locks the operation and checks step transitions.
Success requires a stored evidence reference such as `observation:<uuid>` plus its
observation time. The domain must first persist verified read-back evidence; a
successful outbound request or a newly created outbox record is not read-back.
The database independently enforces required-step evidence. Observed generation
cannot decrease or exceed this operation's desired generation. Finished successful
or superseded operations cannot be changed.

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
first page. Legacy rows with non-contract resource/actor conventions require the
owning producer to migrate its representation; malformed references fail closed.
