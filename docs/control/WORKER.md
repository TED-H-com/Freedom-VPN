# Durable worker jobs

Add one domain constructor line to `Domains()` in `apps/worker/jobs.go`.
The domain build returns its job registry, immutable context values and any
background runners through the [composition seam](COMPOSITION.md). The registry is copied at dispatcher construction and is
immutable during execution. Unknown event types retry and eventually dead-letter;
they are never silently acknowledged.

```go
func handler(ctx context.Context, event db.ClaimedEvent) (string, error) {
    // Decode and validate the versioned domain payload.
    // Query by the persisted external ID before attempting another create.
    // Use event.ID as a stable external idempotency key where supported.
    // Apply the desired effect, read it back, and persist observed evidence plus
    // the operation step transition in a short database transaction.
    // Return the persisted reference, for example "observation:<uuid>".
    return service.ReconcileAndObserve(ctx, event)
}
```

Workers use `ClaimOutbox` with SKIP LOCKED leases, a unique process name and fencing
token. Only a current unexpired claim can finish, retry or dead-letter. Successful
completion atomically records its evidence receipt and delivery time. Handlers
must honor context cancellation and return before the configured job timeout,
which is shorter than the lease. A crashed worker's expired lease makes the same
event available for recovery. Two current dispatchers never own the same event.
An expired/zombie external call still requires domain idempotency and read-back:
fencing the database cannot undo an uncertain external side effect. Delivery is
at least once, not exactly once.

Failures use exponential jittered backoff capped by configuration. After the
configured attempt budget, a fenced transaction sets `dead_lettered_at`, a safe
error code and `available_at=infinity`, releases the lease and appends an audit
row. `delivered_at` stays null. Death after the final attempt is recovered by the
next claimant without invoking the handler again. No provider error bodies or
secrets are persisted/logged. There is deliberately no automatic reset/replay of
dead letters: investigate and repair the domain condition before an operator
creates a new explicitly audited retry intent.

`fvpn-worker -config control.json` uses the same secret-file/configuration loader.
Its separate metrics/health listener defaults to loopback; keep it private.
Prometheus exports `fvpn_worker_events_total{result}`, `fvpn_worker_active`,
`fvpn_worker_job_seconds` and the persisted `fvpn_worker_dead_letters` count.
Dead letters are also visible through the Admin audit endpoint and this read-only
operator query (run with an appropriate database identity):

```sql
SELECT id, event_type, attempt, dead_lettered_at, last_error_code
FROM outbox_events
WHERE dead_lettered_at IS NOT NULL
ORDER BY dead_lettered_at, id;
```

SIGTERM/SIGINT cancel handlers and drain work under the shutdown timeout. A job
that did not establish completion remains leased for recovery; shutdown never
manufactures delivery. `worker.Open` merges all registered domains before any claims; duplicate events
fail startup. The application supervises both dispatcher and domain runners.

Dead-letter audit references use contract resource types (D-029). Existing contract
names are kept; aliases map explicitly: `node-incarnation → node`,
`device → member_device`, `billing_session → checkout_session`,
`fleet → fleet_config`, `fleet_update → update_batch`. The aggregate UUID is
preserved. The actor is `system` with the dispatcher's process UUID.

Dead-lettering is always terminal (D-040). An unmapped aggregate type, including
`billing_inbox`, uses the seeded singleton audit resource
`fleet_config:00000000-0000-0000-0000-000000000001`. Its allowlisted diff contains
`dead_lettered`, `aggregate_type` and `aggregate_id`, subject to the audit helper's
normal redaction. No event payload or provider error is copied into the diff.
The fenced terminal update and audit append commit together, just as for known
resource types; an unknown mapping never causes an endless retry loop.

After a fallback commits, it emits a structured `AUDIT_RESOURCE_TYPE_UNKNOWN`
error with only the event UUID as identifying context and increments
`fvpn_worker_events_total{result="dead_letter_mapping_error"}` as well as the
normal dead-letter metrics. Aggregate values are not logged or used as metric
labels. A stale claim or repeated completion attempt cannot add another audit,
mapping signal or terminal count. The existing lease checks and transactional
rollback still apply if persistence fails.
